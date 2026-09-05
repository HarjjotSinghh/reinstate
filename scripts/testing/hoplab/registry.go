package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// registry.go tracks every hopd/fakelocker pair `hoplab start` has ever
// started, in a file OUTSIDE any lab root -- a lab root can be deleted (or
// never even exist yet if -root itself is wrong) without losing the record
// of what is still running. This is what `hoplab ps` and `hoplab stop
// -all` read, and what lets a fresh `hoplab start` refuse a port another
// still-running lab already owns instead of silently reusing it.
//
// See docs/testing/windows-acceptance-host.md's Hop lab section for the
// dated repro an orphaned lab like this produced: a "fresh" lab's hopd
// process failing to bind its chosen port (because an earlier lab's hopd
// was still listening on it) went unnoticed because runStart only waits
// for /healthz to answer -- which the orphan answered just fine -- so the
// caller talked to the wrong control plane and locker without ever being
// told.

// registryEntry is one hoplab-started lab: the two process ids and
// addresses, and the lab root they belong to (informational -- ps/stop-all
// key everything else off the registry file itself, not this field).
type registryEntry struct {
	Root       string    `json:"root"`
	HopdPID    int       `json:"hopd_pid"`
	HopdAddr   string    `json:"hopd_addr"`
	LockerPID  int       `json:"locker_pid"`
	LockerAddr string    `json:"locker_addr"`
	StartedAt  time.Time `json:"started_at"`
}

// registryDir is where every lab's registry entry lives: a fixed,
// per-user cache directory, never under a lab root or this repository.
func registryDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("locate the per-user cache directory for the hoplab registry: %w", err)
	}
	dir := filepath.Join(base, "reinstate-hoplab", "labs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// registryFile is the one registry entry belonging to root -- named by a
// hash of root's cleaned absolute path so two different -root values never
// collide and the same -root always finds its own prior entry (to replace
// it, on a restart) without listing the whole directory.
func registryFile(root string) (string, error) {
	dir, err := registryDir()
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	key := strings.ToLower(filepath.Clean(abs))
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(dir, hex.EncodeToString(sum[:8])+".json"), nil
}

// registerLab records e, replacing any earlier entry for the same root.
func registerLab(e registryEntry) error {
	path, err := registryFile(e.Root)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

// unregisterLab removes root's registry entry, if any.
func unregisterLab(root string) error {
	path, err := registryFile(root)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// listRegistry returns every recorded lab, across every -root this user
// has ever started with `hoplab start` on this machine. A file that fails
// to parse is skipped rather than failing the whole listing -- ps is a
// diagnostic, and one corrupt entry should not hide the rest.
func listRegistry() ([]registryEntry, error) {
	dir, err := registryDir()
	if err != nil {
		return nil, err
	}
	des, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []registryEntry
	for _, de := range des {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, de.Name()))
		if err != nil {
			continue
		}
		var e registryEntry
		if json.Unmarshal(b, &e) == nil {
			out = append(out, e)
		}
	}
	return out, nil
}

// processAlive reports whether pid names a currently running process, on
// this OS. Best-effort: a false negative (reporting a live process as
// gone) only means ps/stop-all under-report, never that they kill or wait
// on something that no longer exists. The real implementation
// (processAliveNative) is split by GOOS -- registry_windows.go uses
// OpenProcess directly rather than shelling out to `tasklist`, which
// depends on WMI/performance counters unavailable in some restricted
// Windows environments (observed on this development host: a bare
// `tasklist` failed with "ERROR: Critical error"); registry_other.go uses
// signal 0.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return processAliveNative(pid)
}

// portOwner best-effort identifies which pid is listening on addr, for a
// message a person can act on ("stop it, or take the port"). Returning
// ok=false (command missing, output not parseable, nothing found) is not
// an error -- the caller already knows the port refused a bind either way.
func portOwner(addr string) (pid int, ok bool) {
	_, port, err := net.SplitHostPort(addr)
	if err != nil || port == "" {
		return 0, false
	}
	if runtime.GOOS == "windows" {
		out, err := exec.Command("netstat", "-ano").Output()
		if err != nil {
			return 0, false
		}
		for _, line := range strings.Split(string(out), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 5 || !strings.EqualFold(fields[0], "TCP") {
				continue
			}
			if !strings.EqualFold(fields[3], "LISTENING") {
				continue
			}
			if !strings.HasSuffix(fields[1], ":"+port) {
				continue
			}
			if p, err := strconv.Atoi(fields[len(fields)-1]); err == nil {
				return p, true
			}
		}
		return 0, false
	}
	out, err := exec.Command("lsof", "-nP", "-iTCP:"+port, "-sTCP:LISTEN").Output()
	if err != nil {
		return 0, false
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return 0, false
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 2 {
		return 0, false
	}
	p, err := strconv.Atoi(fields[1])
	return p, err == nil
}

// refuseListeningPort refuses to proceed when addr already has a listener
// -- run before hoplab starts hopd/fakelocker, so a stale process from an
// earlier, uncleanly-stopped lab is reported by name and pid instead of
// silently answering in the new lab's place (the orphan-hopd repro in
// docs/testing/windows-acceptance-host.md's Hop lab section).
func refuseListeningPort(addr, label string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		if pid, ok := portOwner(addr); ok {
			return fmt.Errorf("%s address %s is already in use by pid %d; `hoplab ps` lists hoplab-started processes, `hoplab stop -all` stops every one still running, or stop it yourself (Windows: taskkill /PID %d /F)", label, addr, pid, pid)
		}
		return fmt.Errorf("%s address %s is already in use (could not identify the owning process): %w", label, addr, err)
	}
	_ = ln.Close()
	return nil
}
