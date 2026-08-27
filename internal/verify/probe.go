package verify

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// maxErrorBodyBytes caps how much of a refusal body the probe reads before
// handing it back to the S3 client. An S3 <Error> document is a few hundred
// bytes; the cap only stops a hostile endpoint from streaming forever into
// a check whose whole job is to be cheap.
const maxErrorBodyBytes = 1 << 20

// probeTimeout bounds one reference-locker request end to end. The step is
// two tiny requests, and a host that never answers must not hang the
// command that establishes trust.
const probeTimeout = 30 * time.Second

// Exchange is what the reference probe's transport saw for one request:
// where the request was addressed, what came back, and whether the answer
// was a signed S3 error rather than any 403 at all. The isolation step
// pins its verdict to these observations rather than to the endpoint
// strings the control plane supplied, because only the response shows
// where this account's credential actually went.
type Exchange struct {
	// Scheme is the URL scheme the request was sent under, lowercased. It
	// is part of where the request went: `http` means the credential
	// signing it went out in the clear, which the isolation step refuses
	// whatever endpoint the control plane named.
	Scheme string
	// Host is the host (with port, when the URL carried one) the request
	// was addressed to, lowercased.
	Host string
	// Status is the HTTP status code, or 0 when no response arrived at all
	// (connection refused, a refused redirect, a timeout).
	Status int
	// ErrorCode is the <Code> of an S3 <Error> body. It is empty when the
	// response carried no such body — a bodiless 403 is something any web
	// server answers and says nothing about bucket scope.
	ErrorCode string
	// RedirectedTo is the Location a 3xx answer offered. The probe never
	// follows one; recording it lets the step say why it refused to.
	RedirectedTo string
}

// ProbeClient returns the HTTP client the reference locker must be opened
// with, and a function returning what that client has observed so far.
//
// The client refuses every redirect, so this account's locker credential
// is only ever sent to the host the control plane pinned: a redirect rule
// on the pinned endpoint would otherwise send the credential to a host
// that answers 403 to everything and buy a passing isolation step without
// the credential ever reaching a bucket. base is the round tripper to send
// through; nil means http.DefaultTransport.
func ProbeClient(base http.RoundTripper) (*http.Client, func() []Exchange) {
	if base == nil {
		base = http.DefaultTransport
	}
	p := &probeTransport{base: base}
	client := &http.Client{
		Transport: p,
		Timeout:   probeTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			from := ""
			if len(via) > 0 {
				from = via[len(via)-1].URL.Host
			}
			return fmt.Errorf("refused to follow a redirect from %s to %s: this request carries the locker credential and may only be sent to the endpoint the control plane pinned", from, req.URL.Host)
		},
	}
	return client, p.exchanges
}

// probeTransport records each request's destination and the shape of the
// answer. It reads an error body and puts it back, so the S3 client still
// sees exactly the response it would have seen.
type probeTransport struct {
	base http.RoundTripper

	mu   sync.Mutex
	seen []Exchange
}

func (p *probeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ex := Exchange{Scheme: strings.ToLower(req.URL.Scheme), Host: strings.ToLower(req.URL.Host)}
	resp, err := p.base.RoundTrip(req)
	if err != nil {
		p.record(ex)
		return nil, err
	}
	ex.Status = resp.StatusCode
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		ex.RedirectedTo = resp.Header.Get("Location")
	}
	if resp.StatusCode >= 400 && resp.Body != nil {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		_ = resp.Body.Close()
		if readErr == nil {
			ex.ErrorCode = s3ErrorCode(body)
		}
		resp.Body = io.NopCloser(bytes.NewReader(body))
	}
	p.record(ex)
	return resp, nil
}

func (p *probeTransport) record(ex Exchange) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.seen = append(p.seen, ex)
}

func (p *probeTransport) exchanges() []Exchange {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]Exchange(nil), p.seen...)
}

// s3ErrorCode returns the <Code> of an S3 <Error> document, or "" when the
// bytes are not one. A refusal that carries no such document was not
// signed by an S3 API at all.
func s3ErrorCode(body []byte) string {
	var doc struct {
		XMLName xml.Name `xml:"Error"`
		Code    string   `xml:"Code"`
	}
	if err := xml.Unmarshal(body, &doc); err != nil {
		return ""
	}
	return strings.TrimSpace(doc.Code)
}
