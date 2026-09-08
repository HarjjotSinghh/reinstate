//go:build windows

package main

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// restrictSecretFileACL locks path down to the current user only by
// replacing its DACL outright, rather than relying on the file's mode
// bits.
//
// On Windows, os.WriteFile's perm argument (state.go's saveRecoveryCode
// writes 0o600) does not create a restrictive access-control list: Go
// simply skips ACL work on this OS, so a newly created file inherits
// whatever DACL its parent directory already has -- which, for a lab root
// under a normal user profile, routinely grants access far wider than
// "owner only" (BUILTIN\Users, the account's other processes, etc). The
// hoplab README and docs/testing/windows-acceptance-host.md both describe
// the recovery-code file as owner-only; this is what actually makes that
// true here, the same way secretfd_windows.go's SetHandleInformation calls
// do real Windows-native work where the cross-platform standard library
// alone would not.
//
// The replacement DACL grants the current process's user Full Control and
// nothing else, and is marked protected (SDDL's "P" flag) so it does not
// inherit any ACE from the parent directory -- an unprotected DACL would
// keep whatever the parent already granted alongside the new ACE, which
// would defeat the point.
func restrictSecretFileACL(path string) error {
	tok, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return fmt.Errorf("open the current process token: %w", err)
	}
	defer tok.Close()

	user, err := tok.GetTokenUser()
	if err != nil {
		return fmt.Errorf("read the current user's SID: %w", err)
	}
	sid := user.User.Sid.String()

	// D: introduces the DACL. P protects it from inheriting the parent
	// directory's ACEs. The lone ACE, (A;;FA;;;<sid>), Allows the current
	// user File-All-access, with no inheritance flags of its own -- there
	// is nothing under this file for a flag to propagate to.
	sd, err := windows.SecurityDescriptorFromString(fmt.Sprintf("D:P(A;;FA;;;%s)", sid))
	if err != nil {
		return fmt.Errorf("build the owner-only security descriptor: %w", err)
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return fmt.Errorf("read back the built DACL: %w", err)
	}

	if err := windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, dacl, nil,
	); err != nil {
		return fmt.Errorf("apply the owner-only ACL to %s: %w", path, err)
	}
	return nil
}
