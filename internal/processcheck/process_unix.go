//go:build !windows

package processcheck

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

func listProcesses(ctx context.Context) ([]Process, error) {
	output, err := exec.CommandContext(ctx, "ps", "-axo", "pid=,comm=,args=").Output()
	if err != nil {
		return nil, err
	}
	return parsePSOutput(string(output)), nil
}

func parsePSOutput(output string) []Process {
	var procs []Process
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		procs = append(procs, Process{
			PID:         pid,
			Image:       fields[1],
			CommandLine: strings.Join(fields[2:], " "),
		})
	}
	return procs
}

// sessionFileHolders returns the PIDs holding path open.
//
// It reports supported=false when lsof is unavailable, so the caller can fall
// back to the coarse host-wide check rather than silently allowing a restore.
func sessionFileHolders(ctx context.Context, path string) (pids []int, supported bool, err error) {
	if _, lookErr := exec.LookPath("lsof"); lookErr != nil {
		return nil, false, nil
	}
	// +c 0 disables command-name truncation; -F p emits machine-readable
	// "p<pid>" records. lsof exits 1 when simply nothing matches.
	cmd := exec.CommandContext(ctx, "lsof", "+c", "0", "-F", "p", "--", path)
	output, runErr := cmd.Output()
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) && exitErr.ExitCode() == 1 && len(output) == 0 {
			return nil, true, nil
		}
		return nil, false, nil
	}
	return parseLsofPIDs(string(output)), true, nil
}

func parseLsofPIDs(output string) []int {
	var pids []int
	seen := map[int]bool{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "p") {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimPrefix(line, "p"))
		if err != nil || seen[pid] {
			continue
		}
		seen[pid] = true
		pids = append(pids, pid)
	}
	return pids
}
