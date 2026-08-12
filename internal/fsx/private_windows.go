//go:build windows

package fsx

import (
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ProtectOwnerOnly installs a protected Windows DACL for the current user,
// LocalSystem, and the built-in Administrators group. This does not rely on
// inherited ACLs or os.Chmod, whose Windows implementation only changes the
// read-only attribute.
func ProtectOwnerOnly(path string, directory bool) error {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	system, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return err
	}
	administrators, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return err
	}
	inheritance := uint32(0)
	if directory {
		inheritance = windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT
	}
	entries := []windows.EXPLICIT_ACCESS{
		ownerAccess(user.User.Sid, windows.TRUSTEE_IS_USER, inheritance),
		ownerAccess(system, windows.TRUSTEE_IS_WELL_KNOWN_GROUP, inheritance),
		ownerAccess(administrators, windows.TRUSTEE_IS_GROUP, inheritance),
	}
	acl, err := windows.ACLFromEntries(entries, nil)
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		acl,
		nil,
	)
}

// OwnerOnly reports whether path currently carries the owner-only protection
// ProtectOwnerOnly installs, plus a human-readable description of what was
// found. Windows cannot express 0600, so the property checked here is the
// protected DACL: inheritance is blocked and every access-allowed entry names
// the current user, LocalSystem, or the built-in Administrators group.
func OwnerOnly(path string, directory bool) (bool, string, error) {
	_ = directory // inheritance flags differ, the trustee set does not
	descriptor, err := windows.GetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		return false, "", err
	}
	control, _, err := descriptor.Control()
	if err != nil {
		return false, "", err
	}
	if control&windows.SE_DACL_PRESENT == 0 {
		return false, "no DACL is present", nil
	}
	if control&windows.SE_DACL_PROTECTED == 0 {
		return false, "DACL still inherits entries from its parent", nil
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		return false, "", err
	}
	if dacl == nil {
		return false, "DACL is empty and therefore fully permissive", nil
	}
	allowed, err := ownerOnlyTrustees()
	if err != nil {
		return false, "", err
	}
	private, detail := ownerOnlyACEs(dacl, allowed)
	runtime.KeepAlive(descriptor)
	return private, detail, nil
}

// ownerOnlyTrustees returns the only principals a private DACL may name: the
// current user plus the two administrative principals Windows always retains.
func ownerOnlyTrustees() ([]*windows.SID, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	system, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return nil, err
	}
	administrators, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return nil, err
	}
	return []*windows.SID{user.User.Sid, system, administrators}, nil
}

// ownerOnlyACEs walks the DACL entries and requires every one of them to be a
// plain allow entry for an accepted trustee. SIDs are compared directly rather
// than through their SDDL spelling, whose well-known aliases vary with the
// account the process runs as.
func ownerOnlyACEs(dacl *windows.ACL, allowed []*windows.SID) (bool, string) {
	if dacl.AceCount == 0 {
		return false, "DACL has no entries and was not installed by ProtectOwnerOnly"
	}
	for index := uint32(0); index < uint32(dacl.AceCount); index++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, index, &ace); err != nil {
			return false, fmt.Sprintf("read DACL entry %d: %v", index, err)
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			return false, fmt.Sprintf("DACL entry %d has type %d and is not a plain allow entry", index, ace.Header.AceType)
		}
		// Valid single-expression uintptr arithmetic: the SID is stored inline
		// at the end of the ACE, and the ACE points into descriptor's memory.
		sid := (*windows.SID)(unsafe.Pointer(uintptr(unsafe.Pointer(ace)) + unsafe.Offsetof(ace.SidStart)))
		if !sidInSet(allowed, sid) {
			return false, fmt.Sprintf("DACL grants access to %s", sid)
		}
	}
	return true, fmt.Sprintf("protected DACL with %d owner-only entries", dacl.AceCount)
}

func sidInSet(set []*windows.SID, sid *windows.SID) bool {
	for _, candidate := range set {
		if candidate.Equals(sid) {
			return true
		}
	}
	return false
}

func ownerAccess(sid *windows.SID, trusteeType windows.TRUSTEE_TYPE, inheritance uint32) windows.EXPLICIT_ACCESS {
	return windows.EXPLICIT_ACCESS{
		AccessPermissions: windows.GENERIC_ALL,
		AccessMode:        windows.GRANT_ACCESS,
		Inheritance:       inheritance,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  trusteeType,
			TrusteeValue: windows.TrusteeValueFromSID(sid),
		},
	}
}
