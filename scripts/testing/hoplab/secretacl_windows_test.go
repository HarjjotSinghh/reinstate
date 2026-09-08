//go:build windows

package main

import (
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

// currentUserSID returns the SID string for the account running this test
// process, the same lookup restrictSecretFileACL itself makes, so the test
// below compares against the identity the code actually tried to grant
// access to instead of a value that could drift across hosts.
func currentUserSID(t *testing.T) string {
	t.Helper()
	// OpenCurrentProcessToken is deprecated; open the token explicitly
	// with the same TOKEN_QUERY access it requested under the hood, the
	// same fix applied to restrictSecretFileACL in secretacl_windows.go.
	var tok windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &tok); err != nil {
		t.Fatalf("OpenProcessToken: %v", err)
	}
	defer func() {
		_ = tok.Close()
	}()
	user, err := tok.GetTokenUser()
	if err != nil {
		t.Fatalf("GetTokenUser: %v", err)
	}
	return user.User.Sid.String()
}

// TestSaveRecoveryCodeAppliesAnOwnerOnlyACL is the Windows regression for
// the DACL fix (see secretacl_windows.go): os.WriteFile's 0o600 mode bits
// alone do not restrict access on this OS, so the README's and
// docs/testing/windows-acceptance-host.md's "owner only" claim about
// hoplab-recovery-code.secret was false until saveRecoveryCode started
// calling restrictSecretFileACL. This proves the file's real DACL, not
// just its mode bits: exactly one ACE, not inherited from the parent
// directory, naming only the current user.
func TestSaveRecoveryCodeAppliesAnOwnerOnlyACL(t *testing.T) {
	dir := t.TempDir()
	if err := saveRecoveryCode(dir, "OWNER-ONLY-ACL-TEST-CODE"); err != nil {
		t.Fatalf("saveRecoveryCode: %v", err)
	}
	path := recoveryCodePath(dir)

	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatalf("GetNamedSecurityInfo: %v", err)
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		t.Fatalf("SECURITY_DESCRIPTOR.DACL: %v", err)
	}
	if dacl == nil {
		t.Fatal("file has no DACL at all (unrestricted access)")
	}
	if dacl.AceCount != 1 {
		t.Fatalf("DACL has %d ACEs, want exactly 1 (the current user's)", dacl.AceCount)
	}

	var ace *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(dacl, 0, &ace); err != nil {
		t.Fatalf("GetAce(0): %v", err)
	}
	if ace.Header.AceFlags&windows.INHERITED_ACE != 0 {
		t.Fatalf("ACE 0 is marked inherited (AceFlags=%#x); restrictSecretFileACL's protected DACL should prevent every inherited ACE", ace.Header.AceFlags)
	}
	// ACCESS_ALLOWED_ACE's SID is packed immediately after SidStart, not
	// held by a *SID field -- the same layout golang.org/x/sys/windows
	// itself documents for this struct.
	sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
	if got, want := sid.String(), currentUserSID(t); got != want {
		t.Fatalf("ACE 0 grants SID %s, want the current user %s", got, want)
	}
}
