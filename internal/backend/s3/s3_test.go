package s3

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/smithy-go"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
)

func TestMapErr(t *testing.T) {
	if mapErr(nil) != nil {
		t.Fatal("nil")
	}
	if !errors.Is(mapErr(errors.New("StatusCode: 404")), backend.ErrNotFound) && mapErr(errors.New("StatusCode: 404")) != backend.ErrNotFound {
		// mapErr returns backend.ErrNotFound directly
		if mapErr(errors.New("StatusCode: 404")) != backend.ErrNotFound {
			t.Fatalf("404: %v", mapErr(errors.New("StatusCode: 404")))
		}
	}
	if mapErr(errors.New("PreconditionFailed")) != backend.ErrPrecondition {
		t.Fatal("precondition")
	}
	api := &fakeAPIError{code: "AccessDenied", msg: "no"}
	if mapErr(api) != backend.ErrUnauthorized {
		t.Fatalf("auth %v", mapErr(api))
	}
}

type fakeAPIError struct {
	code, msg string
}

func (e *fakeAPIError) Error() string                 { return e.msg }
func (e *fakeAPIError) ErrorCode() string             { return e.code }
func (e *fakeAPIError) ErrorMessage() string          { return e.msg }
func (e *fakeAPIError) ErrorFault() smithy.ErrorFault { return smithy.FaultClient }

func TestNewClientConstruction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()
	ctx := context.Background()
	c, err := New(ctx, Config{
		Endpoint:   srv.URL,
		Region:     "auto",
		Bucket:     "reinstate",
		Prefix:     "profiles/test",
		AccessKey:  "AKIAFAKE",
		SecretKey:  "secretfake",
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if c == nil || c.bucket != "reinstate" {
		t.Fatal("client")
	}
	if got := c.key("snap/x"); !strings.HasSuffix(got, "snap/x") {
		t.Fatalf("key %q", got)
	}
}
