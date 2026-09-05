package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// The fixtures used below (hopHarness, newHopHarness, fakeControlPlane, ...)
// live in hop_test.go because they are shared across the wider hop CLI test
// surface, not exclusive to login and whoami.

func TestWhoamiWithoutOrWithRevokedToken(t *testing.T) {
	h := newHopHarness(t)
	_, errb, code := h.run("whoami")
	if code != ExitAuthStorage || !strings.Contains(errb, "not signed in") {
		t.Fatalf("exit=%d stderr=%q", code, errb)
	}
	if _, _, code := h.run("login"); code != ExitOK {
		t.Fatal("login failed")
	}
	tok, _ := h.tokens.GetDeviceToken()
	h.plane.revoke(tok.Token)
	_, errb, code = h.run("whoami", "--json")
	if code != ExitAuthStorage || !strings.Contains(errb, "rejected") {
		t.Fatalf("exit=%d stderr=%q", code, errb)
	}
	var e ErrorJSON
	if err := json.Unmarshal([]byte(errb), &e); err != nil || e.Code != "auth_storage" {
		t.Fatalf("json error %+v err=%v", e, err)
	}
}
