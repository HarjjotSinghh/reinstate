//go:build windows

package main

import "golang.org/x/sys/windows"

// stillActive is WinAPI's STILL_ACTIVE (winbase.h, 0x103 == 259): the exit
// code GetExitCodeProcess reports for a process that has not exited yet.
// golang.org/x/sys/windows does not export this constant.
const stillActive = 259

// processAliveNative reports whether pid names a currently running
// process, using OpenProcess + GetExitCodeProcess directly rather than
// shelling out to `tasklist`. tasklist depends on WMI/performance
// counters that can be unavailable ("ERROR: Critical error", observed on
// this development host) in a restricted or sandboxed Windows
// environment, while OpenProcess is a plain kernel32 call with no such
// dependency.
func processAliveNative(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		// ERROR_ACCESS_DENIED (a process this user cannot query, e.g. a
		// different account) is treated as "cannot tell" here, same as
		// "gone" -- hoplab only ever queries pids it started itself, so in
		// practice this branch means the process has exited and its pid
		// was reused by something else, or never existed.
		return false
	}
	defer func() { _ = windows.CloseHandle(h) }()
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == stillActive
}
