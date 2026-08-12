package fsx

import (
	"fmt"
	"strings"
)

// ownerOnlyDACL validates the D: section of a Windows SDDL string: inheritance
// must be blocked ("P") and every entry must be a plain allow entry naming an
// accepted trustee. It is deliberately free of Windows-only imports so the
// parsing contract behind OwnerOnly is covered on every build platform.
func ownerOnlyDACL(sddl string, allowed map[string]bool) (bool, string) {
	start := strings.Index(sddl, "D:")
	if start < 0 {
		return false, fmt.Sprintf("security descriptor %q has no DACL section", sddl)
	}
	section := sddl[start+len("D:"):]
	if end := strings.Index(section, "S:"); end >= 0 {
		section = section[:end]
	}
	flags := section
	if open := strings.Index(section, "("); open >= 0 {
		flags = section[:open]
		section = section[open:]
	} else {
		section = ""
	}
	if !strings.Contains(flags, "P") {
		return false, fmt.Sprintf("DACL flags %q do not block inherited access", flags)
	}
	entries := 0
	for {
		open := strings.Index(section, "(")
		if open < 0 {
			break
		}
		closing := strings.Index(section[open:], ")")
		if closing < 0 {
			return false, fmt.Sprintf("malformed DACL entry in %q", sddl)
		}
		entry := section[open+1 : open+closing]
		section = section[open+closing:]
		fields := strings.Split(entry, ";")
		if len(fields) < 6 {
			return false, fmt.Sprintf("malformed DACL entry %q", entry)
		}
		if fields[0] != "A" {
			return false, fmt.Sprintf("DACL entry %q is not a plain allow entry", entry)
		}
		if trustee := strings.ToUpper(fields[5]); !allowed[trustee] {
			return false, fmt.Sprintf("DACL grants access to %s", trustee)
		}
		entries++
	}
	if entries == 0 {
		return false, "DACL grants access to nobody and was not installed by ProtectOwnerOnly"
	}
	return true, fmt.Sprintf("protected DACL with %d owner-only entries", entries)
}
