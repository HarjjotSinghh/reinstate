package s3test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestAnyBucketIsolatesBuckets: the lab locker serves every bucket name,
// but one account's objects must not appear under another account's bucket.
func TestAnyBucketIsolatesBuckets(t *testing.T) {
	f := &Fake{Valid: map[string]bool{}, AcceptPrefix: "FAKEKEY", AnyBucket: true, RejectAs: "ExpiredToken"}
	srv := httptest.NewServer(f)
	defer srv.Close()
	do := func(method, path, body string) int {
		t.Helper()
		req, err := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=FAKEKEY0001/20260824/auto/s3/aws4_request")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	if code := do(http.MethodPut, "/lk-one/keyring.v1.json", "{}"); code != http.StatusOK {
		t.Fatalf("put in lk-one: %d", code)
	}
	tests := []struct {
		bucket string
		want   int
	}{
		{"lk-one", http.StatusOK},
		{"lk-two", http.StatusNotFound},
	}
	for _, tc := range tests {
		if code := do(http.MethodGet, "/"+tc.bucket+"/keyring.v1.json", ""); code != tc.want {
			t.Errorf("get from %s: %d, want %d", tc.bucket, code, tc.want)
		}
	}
}
