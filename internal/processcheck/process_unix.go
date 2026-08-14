//go:build !windows

package processcheck

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const listProcessesTimeout = 5 * time.Second

func listProcesses(ctx context.Context) ([]Process, error) {
	runCtx, cancel := context.WithTimeout(ctx, listProcessesTimeout)
	defer cancel()
	output, err := exec.CommandContext(runCtx, "ps", "-axo", "pid=,comm=,args=").Output()
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

// agentWorkingDirectories maps matching agent PIDs to their working directory.
//
// An empty result is not an error: it only means project affinity cannot
// contribute a signal, and the remaining signals still apply.
func agentWorkingDirectories(ctx context.Context, agent string, procs []Process) map[int]string {
	var pids []string
	for _, p := range procs {
		if matchesAgentProcess(agent, p.Image, p.CommandLine) {
			pids = append(pids, strconv.Itoa(p.PID))
		}
	}
	if len(pids) == 0 {
		return nil
	}
	if _, lookErr := exec.LookPath("lsof"); lookErr != nil {
		return nil
	}
	// -d cwd restricts output to the working-directory entry; -F pn emits
	// machine-readable "p<pid>" and "n<path>" records.
	output, err := exec.CommandContext(ctx, "lsof",
		"+c", "0", "-a", "-d", "cwd", "-F", "pn", "-p", strings.Join(pids, ",")).Output()
	if err != nil && len(output) == 0 {
		return nil
	}
	return parseLsofWorkingDirectories(string(output))
}

func parseLsofWorkingDirectories(output string) map[int]string {
	dirs := map[int]string{}
	current := 0
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "p"):
			pid, err := strconv.Atoi(strings.TrimPrefix(line, "p"))
			if err != nil {
				current = 0
				continue
			}
			current = pid
		case strings.HasPrefix(line, "n") && current != 0:
			dirs[current] = strings.TrimPrefix(line, "n")
		}
	}
	return dirs
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
