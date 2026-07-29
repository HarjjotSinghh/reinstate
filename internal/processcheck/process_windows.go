//go:build windows

package processcheck

import (
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"os/exec"
	"strconv"
	"syscall"
	"unsafe"
)

func listProcesses(ctx context.Context) ([]Process, error) {
	const command = `Get-CimInstance Win32_Process | Select-Object ProcessId,Name,CommandLine | ConvertTo-Csv -NoTypeInformation`
	output, err := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", command).Output()
	if err == nil {
		if procs, parseErr := parseProcessCSV(output, true); parseErr == nil {
			return procs, nil
		}
	}

	output, err = exec.CommandContext(ctx, "tasklist", "/FO", "CSV", "/NH").Output()
	if err != nil {
		return nil, err
	}
	return parseTasklistCSV(output)
}

// parseProcessCSV reads ProcessId,Name,CommandLine records.
func parseProcessCSV(output []byte, hasHeader bool) ([]Process, error) {
	reader := csv.NewReader(bytes.NewReader(output))
	reader.FieldsPerRecord = -1
	if hasHeader {
		if _, err := reader.Read(); err != nil {
			return nil, err
		}
	}
	var procs []Process
	for {
		record, err := reader.Read()
		if err == io.EOF {
			return procs, nil
		}
		if err != nil {
			return nil, err
		}
		if len(record) < 2 {
			continue
		}
		pid, convErr := strconv.Atoi(record[0])
		if convErr != nil {
			continue
		}
		cmdline := ""
		if len(record) >= 3 {
			cmdline = record[2]
		}
		procs = append(procs, Process{PID: pid, Image: record[1], CommandLine: cmdline})
	}
}

// parseTasklistCSV reads the "Image Name","PID",... layout tasklist emits.
func parseTasklistCSV(output []byte) ([]Process, error) {
	reader := csv.NewReader(bytes.NewReader(output))
	reader.FieldsPerRecord = -1
	var procs []Process
	for {
		record, err := reader.Read()
		if err == io.EOF {
			return procs, nil
		}
		if err != nil {
			return nil, err
		}
		if len(record) < 2 {
			continue
		}
		pid, convErr := strconv.Atoi(record[1])
		if convErr != nil {
			continue
		}
		procs = append(procs, Process{PID: pid, Image: record[0]})
	}
}

// agentWorkingDirectories has no cheap Windows implementation.
//
// A process working directory lives in the target process's PEB and reading it
// requires cross-process memory access, which is a disproportionate amount of
// privilege for a restore safety check. Project affinity therefore contributes
// no signal on Windows; the file-handle and command-line signals still apply,
// and the command-line signal is what covers `claude --resume <id>`.
func agentWorkingDirectories(_ context.Context, _ string, _ []Process) map[int]string {
	return nil
}

const (
	cchRMMaxAppName = 255
	cchRMMaxSvcName = 63
	rmSessionKeyLen = 32
	errorMoreData   = 234
)

type rmUniqueProcess struct {
	ProcessID        uint32
	ProcessStartTime syscall.Filetime
}

type rmProcessInfo struct {
	Process          rmUniqueProcess
	AppName          [cchRMMaxAppName + 1]uint16
	ServiceShortName [cchRMMaxSvcName + 1]uint16
	ApplicationType  uint32
	AppStatus        uint32
	TSSessionID      uint32
	Restartable      int32
}

var (
	modRstrtmgr         = syscall.NewLazyDLL("rstrtmgr.dll")
	procRmStartSession  = modRstrtmgr.NewProc("RmStartSession")
	procRmRegisterResrc = modRstrtmgr.NewProc("RmRegisterResources")
	procRmGetList       = modRstrtmgr.NewProc("RmGetList")
	procRmEndSession    = modRstrtmgr.NewProc("RmEndSession")
)

// sessionFileHolders uses the Restart Manager to find processes holding path.
//
// Restart Manager ships with Windows and is what installers use for exactly
// this question, so it needs no third-party dependency. supported=false is
// returned when the API is unavailable, so the caller degrades to the coarse
// host-wide check instead of silently permitting a restore.
func sessionFileHolders(_ context.Context, path string) (pids []int, supported bool, err error) {
	if loadErr := modRstrtmgr.Load(); loadErr != nil {
		return nil, false, nil
	}
	for _, p := range []*syscall.LazyProc{
		procRmStartSession, procRmRegisterResrc, procRmGetList, procRmEndSession,
	} {
		if findErr := p.Find(); findErr != nil {
			return nil, false, nil
		}
	}

	var session uint32
	sessionKey := make([]uint16, rmSessionKeyLen+1)
	ret, _, _ := procRmStartSession.Call(
		uintptr(unsafe.Pointer(&session)), 0, uintptr(unsafe.Pointer(&sessionKey[0])))
	if ret != 0 {
		return nil, false, nil
	}
	defer func() { _, _, _ = procRmEndSession.Call(uintptr(session)) }()

	target, convErr := syscall.UTF16PtrFromString(path)
	if convErr != nil {
		return nil, false, nil
	}
	files := []*uint16{target}
	ret, _, _ = procRmRegisterResrc.Call(
		uintptr(session),
		uintptr(len(files)), uintptr(unsafe.Pointer(&files[0])),
		0, 0,
		0, 0,
	)
	if ret != 0 {
		return nil, false, nil
	}

	var needed, count, reasons uint32
	ret, _, _ = procRmGetList.Call(
		uintptr(session),
		uintptr(unsafe.Pointer(&needed)),
		uintptr(unsafe.Pointer(&count)),
		0,
		uintptr(unsafe.Pointer(&reasons)),
	)
	if ret != 0 && ret != errorMoreData {
		return nil, false, nil
	}
	if needed == 0 {
		return nil, true, nil
	}

	infos := make([]rmProcessInfo, needed)
	count = needed
	ret, _, _ = procRmGetList.Call(
		uintptr(session),
		uintptr(unsafe.Pointer(&needed)),
		uintptr(unsafe.Pointer(&count)),
		uintptr(unsafe.Pointer(&infos[0])),
		uintptr(unsafe.Pointer(&reasons)),
	)
	if ret != 0 {
		return nil, false, nil
	}
	for i := uint32(0); i < count && i < needed; i++ {
		pids = append(pids, int(infos[i].Process.ProcessID))
	}
	return pids, true, nil
}
