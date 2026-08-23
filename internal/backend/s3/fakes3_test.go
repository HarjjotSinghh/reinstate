package s3

import (
	"context"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/backend/s3/s3test"
)

// fakeS3 is the shared s3test fake with a helper that builds a Client
// against it.
type fakeS3 struct {
	*s3test.Fake
}

func newFakeS3(t *testing.T, bucket string) *fakeS3 {
	t.Helper()
	return &fakeS3{Fake: s3test.New(t, bucket)}
}

func (f *fakeS3) accept(keys ...string) { f.Accept(keys...) }

func (f *fakeS3) requestLog() []string { return f.RequestLog() }

func (f *fakeS3) client(t *testing.T, cfg Config) *Client {
	t.Helper()
	cfg.Endpoint = f.Srv.URL
	cfg.Bucket = f.Bucket
	cfg.HTTPClient = f.Srv.Client()
	if cfg.Region == "" {
		cfg.Region = "auto"
	}
	c, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
