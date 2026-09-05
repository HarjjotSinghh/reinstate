package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

// startOptions configures `hoplab start`.
type startOptions struct {
	Root       string
	HopdAddr   string
	LockerAddr string
	HopdBin    string // REINSTATE_HOPD_BIN; built from HostedDir when empty
	HostedDir  string // REINSTATE_HOSTED_DIR, default D:\Projects\reinstate-hosted
}

// startResult is what a successful start produced, kept in-process for the
// foreground wait loop and also written out as LabState for other
// processes.
type startResult struct {
	state    LabState
	hopdCmd  *exec.Cmd
	lockCmd  *exec.Cmd
	hopdLogF *os.File
}

func runStart(o startOptions) (*startResult, error) {
	root, err := filepath.Abs(o.Root)
	if err != nil {
		return nil, err
	}
	if err := refuseInsideCheckout(root); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	repoRoot, err := findRepoRoot("")
	if err != nil {
		return nil, err
	}

	hopdAddr := nonEmpty(o.HopdAddr, "127.0.0.1:8082")
	lockerAddr := nonEmpty(o.LockerAddr, "127.0.0.1:9002")
	// Refused before any build or process starts: a stale hopd or
	// fakelocker from an earlier, uncleanly-stopped lab left listening on
	// one of these addresses would otherwise answer /healthz just fine in
	// the new lab's place, and everything after start would silently talk
	// to the wrong control plane and locker (the dated repro in
	// docs/testing/windows-acceptance-host.md's Hop lab section).
	if err := refuseListeningPort(hopdAddr, "hopd"); err != nil {
		return nil, err
	}
	if err := refuseListeningPort(lockerAddr, "fakelocker"); err != nil {
		return nil, err
	}

	hopdBin := o.HopdBin
	if hopdBin == "" {
		hopdBin, err = buildHopd(root, o.HostedDir)
		if err != nil {
			return nil, err
		}
	}
	lockerBin, err := buildFakelocker(root, repoRoot)
	if err != nil {
		return nil, err
	}

	baseURL := "http://" + hopdAddr
	lockerURL := "http://" + lockerAddr
	dbPath := filepath.Join(root, "hopd.db")
	_ = os.Remove(dbPath)
	logPath := filepath.Join(root, "hopd.log")
	logF, err := os.Create(logPath)
	if err != nil {
		return nil, err
	}

	lockCmd := exec.Command(lockerBin, "-addr", lockerAddr)
	lockCmd.Stdout = logF
	lockCmd.Stderr = logF
	if err := lockCmd.Start(); err != nil {
		logF.Close()
		return nil, fmt.Errorf("start fakelocker: %w", err)
	}

	hopdCmd := exec.Command(hopdBin)
	hopdCmd.Env = append(os.Environ(),
		"HOPD_ADDR="+hopdAddr,
		"HOPD_BASE_URL="+baseURL,
		"HOPD_DB_PATH="+dbPath,
		"HOPD_EMAIL_SENDER=log",
		"HOPD_STORAGE=fake",
		"HOPD_S3_ENDPOINT="+lockerURL,
	)
	hopdCmd.Stdout = logF
	hopdCmd.Stderr = logF
	if err := hopdCmd.Start(); err != nil {
		_ = lockCmd.Process.Kill()
		logF.Close()
		return nil, fmt.Errorf("start hopd: %w", err)
	}

	if err := waitHealthy(baseURL+"/healthz", 20*time.Second); err != nil {
		_ = hopdCmd.Process.Kill()
		_ = lockCmd.Process.Kill()
		logF.Close()
		return nil, fmt.Errorf("hopd did not become healthy: %w (see %s)", err, logPath)
	}

	state := LabState{
		Root: root, HopdPID: hopdCmd.Process.Pid, HopdAddr: hopdAddr, HopdBaseURL: baseURL,
		HopdLog: logPath, HopdDB: dbPath, LockerPID: lockCmd.Process.Pid, LockerAddr: lockerAddr,
	}
	if err := state.save(); err != nil {
		return nil, err
	}
	// Recorded outside root too (registry.go), so `hoplab ps`/`hoplab stop
	// -all` can find these processes even from a terminal that never knew
	// this -root, and even if root itself is later deleted.
	if err := registerLab(registryEntry{
		Root: root, HopdPID: hopdCmd.Process.Pid, HopdAddr: hopdAddr,
		LockerPID: lockCmd.Process.Pid, LockerAddr: lockerAddr, StartedAt: time.Now().UTC(),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "hoplab: warning: could not record this lab in the process registry (%v); `hoplab ps`/`hoplab stop -all` will not see it\n", err)
	}
	return &startResult{state: state, hopdCmd: hopdCmd, lockCmd: lockCmd, hopdLogF: logF}, nil
}

// runForeground blocks until Ctrl+C or either child process exits on its
// own, then stops the other and returns.
func (r *startResult) runForeground(out io.Writer) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	hopdDone := make(chan error, 1)
	lockDone := make(chan error, 1)
	go func() { hopdDone <- r.hopdCmd.Wait() }()
	go func() { lockDone <- r.lockCmd.Wait() }()

	fmt.Fprintf(out, "hoplab: hopd pid %d on %s, fakelocker pid %d on %s; log at %s\n",
		r.hopdCmd.Process.Pid, r.state.HopdAddr, r.lockCmd.Process.Pid, r.state.LockerAddr, r.state.HopdLog)
	fmt.Fprintln(out, "hoplab: Ctrl+C, or `hoplab stop --root "+r.state.Root+"` from another terminal, stops both.")

	select {
	case <-ctx.Done():
		fmt.Fprintln(out, "hoplab: stopping (Ctrl+C)...")
	case err := <-hopdDone:
		fmt.Fprintf(out, "hoplab: hopd exited on its own: %v\n", err)
	case err := <-lockDone:
		fmt.Fprintf(out, "hoplab: fakelocker exited on its own: %v\n", err)
	}
	r.stop()
	_ = unregisterLab(r.state.Root)
	return removeState(r.state.Root)
}

func (r *startResult) stop() {
	_ = r.hopdCmd.Process.Kill()
	_ = r.lockCmd.Process.Kill()
	_, _ = r.hopdCmd.Process.Wait()
	_, _ = r.lockCmd.Process.Wait()
	if r.hopdLogF != nil {
		r.hopdLogF.Close()
	}
}

// runStop kills the processes a `hoplab start --background` (or a foreground
// run in another terminal) recorded, from its LabState.
func runStop(root string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	s, err := loadState(root)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no hoplab-state.json under %s; nothing to stop", root)
		}
		return err
	}
	for _, pid := range []int{s.HopdPID, s.LockerPID} {
		if pid <= 0 {
			continue
		}
		if p, err := os.FindProcess(pid); err == nil {
			_ = p.Kill()
		}
	}
	_ = unregisterLab(root)
	return removeState(root)
}

// runStopAll kills every process the registry still lists as running,
// across every -root this user has ever started `hoplab start` with on
// this machine, and clears each entry (and its lab root's state.json,
// best-effort -- the root may already be gone). It reports how many it
// stopped and how many entries it removed because the process behind them
// was already gone.
func runStopAll(out io.Writer) error {
	entries, err := listRegistry()
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Fprintln(out, "hoplab: no labs recorded in the process registry")
		return nil
	}
	stopped, stale := 0, 0
	for _, e := range entries {
		live := false
		for _, pid := range []int{e.HopdPID, e.LockerPID} {
			if pid <= 0 || !processAlive(pid) {
				continue
			}
			live = true
			if p, err := os.FindProcess(pid); err == nil {
				_ = p.Kill()
			}
		}
		if live {
			stopped++
			fmt.Fprintf(out, "hoplab: stopped lab at %s (hopd pid %d, fakelocker pid %d)\n", e.Root, e.HopdPID, e.LockerPID)
		} else {
			stale++
		}
		_ = unregisterLab(e.Root)
		_ = removeState(e.Root)
	}
	fmt.Fprintf(out, "hoplab: stopped %d lab(s); removed %d stale registry entry(ies) whose processes were already gone\n", stopped, stale)
	return nil
}

func waitHealthy(url string, timeout time.Duration) error {
	client := &http.Client{Timeout: 3 * time.Second}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("status %s", resp.Status)
		} else {
			lastErr = err
		}
		time.Sleep(250 * time.Millisecond)
	}
	return lastErr
}

func buildFakelocker(root, repoRoot string) (string, error) {
	out := filepath.Join(root, exeName("fakelocker"))
	cmd := exec.Command("go", "build", "-o", out, "./scripts/testing/fakelocker")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if b, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("go build fakelocker: %w\n%s", err, b)
	}
	return out, nil
}

func buildHopd(root, hostedDir string) (string, error) {
	hostedDir = nonEmpty(hostedDir, os.Getenv("REINSTATE_HOSTED_DIR"))
	hostedDir = nonEmpty(hostedDir, `D:\Projects\reinstate-hosted`)
	if _, err := os.Stat(filepath.Join(hostedDir, "go.mod")); err != nil {
		return "", fmt.Errorf("%s does not look like the control-plane checkout (no go.mod): %w; set REINSTATE_HOPD_BIN to a prebuilt hopd instead", hostedDir, err)
	}
	out := filepath.Join(root, exeName("hopd"))
	cmd := exec.Command("go", "build", "-o", out, "./cmd/hopd")
	cmd.Dir = hostedDir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if b, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("go build hopd (from %s): %w\n%s", hostedDir, err, b)
	}
	return out, nil
}

func exeName(base string) string {
	if strings.EqualFold(os.Getenv("OS"), "Windows_NT") || filepath.Separator == '\\' {
		return base + ".exe"
	}
	return base
}

func nonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// refuseInsideCheckout mirrors scripts/tuisandbox's own refusal: a lab root
// under a Git checkout risks hopd.db, logs, and keyring snapshots landing
// in a place `git add -A` could pick up.
func refuseInsideCheckout(root string) error {
	current := filepath.Clean(root)
	for {
		if _, err := os.Lstat(filepath.Join(current, ".git")); err == nil {
			return fmt.Errorf("-root %s is inside a Git checkout (found %s\\.git); use a path outside any checkout, e.g. D:\\ReinstateAcceptanceProjects\\hoplab", root, current)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}
