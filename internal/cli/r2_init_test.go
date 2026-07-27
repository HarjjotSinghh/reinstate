package cli

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/config"
)

func TestInitAdditionalDeviceProbesReadableManifest(t *testing.T) {
	home := t.TempDir()
	projectRoot := t.TempDir()
	profileID := "33333333-3333-4333-8333-333333333333"
	manifestPath := "/reinstate/profiles/" + profileID + "/manifest.age"
	manifest := "synthetic encrypted manifest"
	var requests []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.EscapedPath()
		requests = append(requests, r.Method+" "+path)
		switch {
		case r.Method == http.MethodHead && path == manifestPath:
			http.Error(w, "Bad Request", http.StatusBadRequest)
		case r.Method == http.MethodGet && path == manifestPath:
			checksum := sha256.Sum256([]byte(manifest))
			w.Header().Set("X-Amz-Checksum-Sha256", base64.StdEncoding.EncodeToString(checksum[:]))
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", strconv.Itoa(len(manifest)))
			w.Header().Set("ETag", `"manifest-etag"`)
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, manifest)
		case r.Method == http.MethodPut && strings.Contains(path, "/probes/"):
			_, _ = io.Copy(io.Discard, r.Body)
			w.Header().Set("ETag", `"probe-etag"`)
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodDelete && strings.Contains(path, "/probes/"):
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	t.Setenv("REINSTATE_HOME", home)
	t.Setenv("REINSTATE_BACKEND", "")
	t.Setenv("REINSTATE_S3_ACCESS_KEY_ID", "AKIA_TEST")
	t.Setenv("REINSTATE_S3_SECRET_ACCESS_KEY", "SECRET_TEST")

	out, errb, code := runCLI(
		t,
		"reinstate",
		"init",
		"--profile-id", profileID,
		"--endpoint", srv.URL,
		"--bucket", "reinstate",
		"--project", "local/test="+projectRoot,
		"--yes",
	)
	if code != ExitOK {
		t.Fatalf("init exit=%d stdout=%q stderr=%q requests=%v", code, out, errb, requests)
	}
	if len(requests) < 3 || requests[0] != http.MethodGet+" "+manifestPath {
		t.Fatalf("requests=%v want readable-manifest GET followed by probe PUT/DELETE", requests)
	}
	for _, request := range requests {
		if strings.HasPrefix(request, http.MethodHead+" ") {
			t.Fatalf("unexpected metadata-only manifest probe: %v", requests)
		}
	}
	if _, err := os.Stat(config.ConfigPath(home)); err != nil {
		t.Fatalf("successful additional-device init did not save config: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "state.json")); err != nil {
		t.Fatalf("successful additional-device init did not save state: %v", err)
	}
}
