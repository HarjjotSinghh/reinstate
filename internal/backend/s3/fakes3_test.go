package s3

import (
	"bufio"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
)

// fakeS3 is an S3-compatible HTTPS front over the in-memory backend. It honours
// If-Match / If-None-Match the way R2 does and rejects requests signed with an
// access key it does not currently accept, so tests can expire a credential
// between two requests without any network.
type fakeS3 struct {
	t      *testing.T
	store  *memory.Store
	srv    *httptest.Server
	bucket string

	mu       sync.Mutex
	valid    map[string]bool
	rejectAs string // S3 error code for rejected keys
	requests []string
	// hook runs under the lock before each request is authorised, with the
	// 1-based request number; tests use it to expire a key mid-operation.
	hook func(n int)
}

var credentialRe = regexp.MustCompile(`Credential=([^/]+)/`)

func newFakeS3(t *testing.T, bucket string) *fakeS3 {
	t.Helper()
	f := &fakeS3{t: t, store: memory.New(), bucket: bucket, valid: map[string]bool{}, rejectAs: "ExpiredToken"}
	f.srv = httptest.NewTLSServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeS3) accept(keys ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.valid = map[string]bool{}
	for _, k := range keys {
		f.valid[k] = true
	}
}

func (f *fakeS3) requestLog() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.requests...)
}

func (f *fakeS3) client(t *testing.T, cfg Config) *Client {
	t.Helper()
	cfg.Endpoint = f.srv.URL
	cfg.Bucket = f.bucket
	cfg.HTTPClient = f.srv.Client()
	if cfg.Region == "" {
		cfg.Region = "auto"
	}
	c, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func (f *fakeS3) handle(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.EscapedPath(), "/"+f.bucket)
	key := strings.TrimPrefix(path, "/")
	akid := ""
	if m := credentialRe.FindStringSubmatch(r.Header.Get("Authorization")); m != nil {
		akid = m[1]
	}
	f.mu.Lock()
	f.requests = append(f.requests, r.Method+" "+key+" as "+akid)
	if f.hook != nil {
		f.hook(len(f.requests))
	}
	ok := f.valid[akid]
	code := f.rejectAs
	f.mu.Unlock()
	if !ok {
		writeS3Error(w, http.StatusForbidden, code, "credential rejected by fake S3")
		return
	}
	ctx := r.Context()
	switch {
	case r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2":
		items, err := f.store.List(ctx, r.URL.Query().Get("prefix"))
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
			XMLName  xml.Name  `xml:"ListBucketResult"`
			Contents []content `xml:"Contents"`
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
		meta, err := f.store.Put(ctx, key, bytes.NewReader(body), int64(len(body)), opts)
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
		rc, meta, err := f.store.Get(ctx, key)
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
		if err := f.store.Delete(ctx, key); err != nil && !errors.Is(err, backend.ErrNotFound) {
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
