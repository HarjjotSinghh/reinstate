// Package s3test is an S3-compatible HTTP front over the in-memory backend
// for tests: it honours If-Match / If-None-Match the way R2 does and
// rejects requests signed with an access key it does not currently accept,
// so a credential can be expired between two requests without any network.
package s3test

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
)

// Fake is one fake bucket behind an httptest server.
type Fake struct {
	// Store holds the objects; tests may inspect it directly.
	Store  *memory.Store
	Srv    *httptest.Server
	Bucket string

	// Mu guards the fields below. Hook runs with it held.
	Mu sync.Mutex
	// Valid is the set of access key ids currently accepted.
	Valid map[string]bool
	// AcceptPrefix, when set, accepts every access key id that starts with
	// it in addition to Valid (the physical lab server uses it to take
	// whatever hopd's fake provider mints).
	AcceptPrefix string
	// AnyBucket serves every bucket name instead of only Bucket: the first
	// path segment is treated as the bucket whatever it is.
	AnyBucket bool
	// RejectAs is the S3 error code answered for a rejected key.
	RejectAs string
	// Requests is "METHOD key as AKID" per request, in order.
	Requests []string
	// Hook runs under Mu before each request is authorised, with the
	// 1-based request number; tests use it to expire a key mid-operation.
	Hook func(n int)
	// ReadOnly refuses every PUT and DELETE with AccessDenied, the way a
	// locker behaves once the account is read-only (lapsed).
	ReadOnly bool
	// PageSize, when positive, truncates listings to that many keys per
	// page and hands out continuation tokens, the way S3 does at 1000.
	PageSize int
}

var credentialRe = regexp.MustCompile(`Credential=([^/]+)/`)

// New starts a TLS fake for bucket; use Srv.Client() to trust it.
func New(t *testing.T, bucket string) *Fake {
	t.Helper()
	f := &Fake{Store: memory.New(), Bucket: bucket, Valid: map[string]bool{}, RejectAs: "ExpiredToken"}
	f.Srv = httptest.NewTLSServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.Srv.Close)
	return f
}

// NewPlain starts the fake over plain HTTP, reachable with any client; CLI
// journeys that cannot inject an HTTP client use it.
func NewPlain(t *testing.T, bucket string) *Fake {
	t.Helper()
	f := &Fake{Store: memory.New(), Bucket: bucket, Valid: map[string]bool{}, RejectAs: "ExpiredToken"}
	f.Srv = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.Srv.Close)
	return f
}

// Accept replaces the set of accepted access key ids.
func (f *Fake) Accept(keys ...string) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.AcceptLocked(keys...)
}

// AcceptLocked is Accept for callers already holding Mu (a Hook).
func (f *Fake) AcceptLocked(keys ...string) {
	f.Valid = map[string]bool{}
	for _, k := range keys {
		f.Valid[k] = true
	}
}

// RequestLog returns a copy of Requests.
func (f *Fake) RequestLog() []string {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	return append([]string(nil), f.Requests...)
}

// URL is the endpoint to configure a client with.
func (f *Fake) URL() string { return f.Srv.URL }

// ServeHTTP lets a Fake be mounted on any server, not only httptest.
func (f *Fake) ServeHTTP(w http.ResponseWriter, r *http.Request) { f.handle(w, r) }

func (f *Fake) handle(w http.ResponseWriter, r *http.Request) {
	var path string
	if f.AnyBucket {
		_, rest, _ := strings.Cut(strings.TrimPrefix(r.URL.EscapedPath(), "/"), "/")
		path = "/" + rest
	} else {
		path = strings.TrimPrefix(r.URL.EscapedPath(), "/"+f.Bucket)
	}
	key := strings.TrimPrefix(path, "/")
	akid := ""
	if m := credentialRe.FindStringSubmatch(r.Header.Get("Authorization")); m != nil {
		akid = m[1]
	}
	f.Mu.Lock()
	f.Requests = append(f.Requests, r.Method+" "+key+" as "+akid)
	if f.Hook != nil {
		f.Hook(len(f.Requests))
	}
	ok := f.Valid[akid] || (f.AcceptPrefix != "" && akid != "" && strings.HasPrefix(akid, f.AcceptPrefix))
	code := f.RejectAs
	readOnly := f.ReadOnly
	pageSize := f.PageSize
	f.Mu.Unlock()
	if !ok {
		writeS3Error(w, http.StatusForbidden, code, "credential rejected by fake S3")
		return
	}
	if readOnly && (r.Method == http.MethodPut || r.Method == http.MethodDelete) {
		writeS3Error(w, http.StatusForbidden, "AccessDenied", "this bucket is read-only")
		return
	}
	ctx := r.Context()
	switch {
	case r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2":
		items, err := f.Store.List(ctx, r.URL.Query().Get("prefix"))
		if err != nil {
			writeS3Error(w, 500, "InternalError", err.Error())
			return
		}
		type content struct {
			Key  string `xml:"Key"`
			ETag string `xml:"ETag"`
			Size int64  `xml:"Size"`
		}
		var out struct {
			XMLName               xml.Name  `xml:"ListBucketResult"`
			Contents              []content `xml:"Contents"`
			IsTruncated           bool      `xml:"IsTruncated"`
			NextContinuationToken string    `xml:"NextContinuationToken,omitempty"`
		}
		// S3 lists in key order; a continuation token is the last key of
		// the previous page, so the next page starts after it.
		sort.Slice(items, func(i, j int) bool { return items[i].Key < items[j].Key })
		if after := r.URL.Query().Get("continuation-token"); after != "" {
			n := 0
			for n < len(items) && items[n].Key <= after {
				n++
			}
			items = items[n:]
		}
		if pageSize > 0 && len(items) > pageSize {
			items = items[:pageSize]
			out.IsTruncated = true
			out.NextContinuationToken = items[len(items)-1].Key
		}
		for _, it := range items {
			out.Contents = append(out.Contents, content{Key: it.Key, ETag: `"` + it.ETag + `"`, Size: it.Size})
		}
		w.Header().Set("Content-Type", "application/xml")
		_ = xml.NewEncoder(w).Encode(out)
	case r.Method == http.MethodPut:
		body, err := readBody(r)
		if err != nil {
			writeS3Error(w, 400, "IncompleteBody", err.Error())
			return
		}
		opts := backend.PutOptions{}
		if im := strings.Trim(r.Header.Get("If-Match"), `"`); im != "" {
			opts.IfMatch = im
		}
		if r.Header.Get("If-None-Match") == "*" {
			opts.IfNoneMatch = true
		}
		meta, err := f.Store.Put(ctx, key, bytes.NewReader(body), int64(len(body)), opts)
		switch {
		case errors.Is(err, backend.ErrPrecondition), errors.Is(err, backend.ErrAlreadyExists):
			writeS3Error(w, http.StatusPreconditionFailed, "PreconditionFailed", "At least one of the pre-conditions you specified did not hold")
			return
		case err != nil:
			writeS3Error(w, 500, "InternalError", err.Error())
			return
		}
		w.Header().Set("ETag", `"`+meta.ETag+`"`)
		w.WriteHeader(http.StatusOK)
	case r.Method == http.MethodGet || r.Method == http.MethodHead:
		rc, meta, err := f.Store.Get(ctx, key)
		if errors.Is(err, backend.ErrNotFound) {
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			writeS3Error(w, http.StatusNotFound, "NoSuchKey", "The specified key does not exist.")
			return
		}
		if err != nil {
			writeS3Error(w, 500, "InternalError", err.Error())
			return
		}
		data, _ := io.ReadAll(rc)
		_ = rc.Close()
		w.Header().Set("ETag", `"`+meta.ETag+`"`)
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			_, _ = w.Write(data)
		}
	case r.Method == http.MethodDelete:
		if err := f.Store.Delete(ctx, key); err != nil && !errors.Is(err, backend.ErrNotFound) {
			writeS3Error(w, 500, "InternalError", err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeS3Error(w, http.StatusMethodNotAllowed, "MethodNotAllowed", r.Method)
	}
}

// readBody returns the object bytes, decoding aws-chunked transfer encoding
// when the SDK streams a body with trailing checksums.
func readBody(r *http.Request) ([]byte, error) {
	if !strings.Contains(r.Header.Get("Content-Encoding"), "aws-chunked") {
		return io.ReadAll(r.Body)
	}
	br := bufio.NewReader(r.Body)
	var out bytes.Buffer
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return nil, err
		}
		sizeHex := strings.TrimSpace(strings.SplitN(line, ";", 2)[0])
		size, err := strconv.ParseInt(sizeHex, 16, 64)
		if err != nil {
			return nil, fmt.Errorf("bad chunk size %q: %w", line, err)
		}
		if size == 0 {
			return out.Bytes(), nil
		}
		if _, err := io.CopyN(&out, br, size); err != nil {
			return nil, err
		}
		if _, err := br.ReadString('\n'); err != nil {
			return nil, err
		}
	}
}

func writeS3Error(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>%s</Code><Message>%s</Message></Error>`, code, msg)
}
