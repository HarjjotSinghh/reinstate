// Command hoplab runs a disposable Hop lab on this host: a real hopd (the
// private control plane, built from REINSTATE_HOSTED_DIR or named by
// REINSTATE_HOPD_BIN) plus scripts/testing/fakelocker standing in for the
// bucket, both on loopback with fake/log providers, and an approver that
// clicks the sign-in links hopd's log email sender prints -- the same shape
// the 2026-08-24 lab used (docs/testing/results/2026-08-24-first-push-acceptance-lab.md).
//
// It never commits anything from the private control-plane repository into
// this one; that repository is referred to only by path, through
// REINSTATE_HOSTED_DIR / REINSTATE_HOPD_BIN.
//
// See README.md next to this file for full usage and the -tags
// hopacceptance suites this unblocks.
//
// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	if len(argv) == 0 {
		usage()
		return 2
	}
	sub, rest := argv[0], argv[1:]
	var err error
	switch sub {
	case "start":
		err = cmdStart(rest)
	case "stop":
		err = cmdStop(rest)
	case "approve":
		err = cmdApprove(rest)
	case "homes":
		err = cmdHomes(rest)
	case "env":
		err = cmdEnv(rest)
	case "keyring":
		err = cmdKeyring(rest)
	case "-h", "--help", "help":
		usage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "hoplab: unknown command %q\n", sub)
		usage()
		return 2
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "hoplab %s: %v\n", sub, err)
		return 1
	}
	return 0
}

func usage() {
	fmt.Fprint(os.Stderr, `usage: hoplab <command> [flags]

commands:
  start    build/locate hopd and fakelocker, run them, print the env block
  stop     stop a lab started with -background (or from another terminal)
  approve  approve (or, with -refuse, decline) sign-in emails as they appear
  homes    seed two (or more) isolated device homes under -root
  env      print the env block for one seeded device
  keyring  save/load/clear the OS keyring's Hop device token, to swap which
           device the real rein binary acts as

See README.md next to this program for full usage.
`)
}

func cmdStart(argv []string) error {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	root := fs.String("root", "", "lab root directory (required; must be outside any Git checkout)")
	hopdAddr := fs.String("hopd-addr", "127.0.0.1:8082", "hopd listen address")
	lockerAddr := fs.String("locker-addr", "127.0.0.1:9002", "fakelocker listen address")
	hopdBin := fs.String("hopd-bin", os.Getenv("REINSTATE_HOPD_BIN"), "a prebuilt hopd binary (env REINSTATE_HOPD_BIN); built from -hosted-dir when empty")
	hostedDir := fs.String("hosted-dir", os.Getenv("REINSTATE_HOSTED_DIR"), "the private control-plane checkout (env REINSTATE_HOSTED_DIR; default D:\\Projects\\reinstate-hosted)")
	background := fs.Bool("background", false, "start and return immediately, leaving hopd and fakelocker running; stop later with `hoplab stop`")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if strings.TrimSpace(*root) == "" {
		return fmt.Errorf("-root is required")
	}
	res, err := runStart(startOptions{Root: *root, HopdAddr: *hopdAddr, LockerAddr: *lockerAddr, HopdBin: *hopdBin, HostedDir: *hostedDir})
	if err != nil {
		return err
	}
	fmt.Printf("export REINSTATE_HOP_URL=%q\n", res.state.HopdBaseURL)
	fmt.Fprintf(os.Stderr, "hoplab: hopd %s (pid %d), fakelocker %s (pid %d), db %s, log %s\n",
		res.state.HopdAddr, res.hopdCmd.Process.Pid, res.state.LockerAddr, res.lockCmd.Process.Pid, res.state.HopdDB, res.state.HopdLog)
	if *background {
		fmt.Fprintf(os.Stderr, "hoplab: running in the background; stop with `hoplab stop -root %s`\n", res.state.Root)
		return nil
	}
	return res.runForeground(os.Stderr)
}

func cmdStop(argv []string) error {
	fs := flag.NewFlagSet("stop", flag.ExitOnError)
	root := fs.String("root", "", "lab root directory (required)")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if strings.TrimSpace(*root) == "" {
		return fmt.Errorf("-root is required")
	}
	if err := runStop(*root); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "hoplab: stopped")
	return nil
}

func cmdApprove(argv []string) error {
	fs := flag.NewFlagSet("approve", flag.ExitOnError)
	root := fs.String("root", "", "lab root directory (reads hoplab-state.json for the log path)")
	logPath := fs.String("log", "", "hopd log path, instead of reading it from -root's state")
	email := fs.String("email", "", "only approve links addressed to this address (default: any)")
	count := fs.Int("count", 1, "stop after this many sign-ins are decided")
	timeout := fs.Duration("timeout", 2*time.Minute, "give up after this long")
	refuse := fs.Bool("refuse", false, "see each link but withhold approval (exercises the refused/expired sign-in path)")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if strings.TrimSpace(*root) == "" && strings.TrimSpace(*logPath) == "" {
		return fmt.Errorf("-root or -log is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout+5*time.Second)
	defer cancel()
	return runApprove(ctx, approveOptions{Root: *root, LogPath: *logPath, Email: *email, Count: *count, Timeout: *timeout, Refuse: *refuse}, os.Stdout, &Approver{})
}

func cmdHomes(argv []string) error {
	fs := flag.NewFlagSet("homes", flag.ExitOnError)
	root := fs.String("root", "", "lab root directory (required)")
	devices := fs.String("devices", "device-a,device-b", "comma-separated device names to seed")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if strings.TrimSpace(*root) == "" {
		return fmt.Errorf("-root is required")
	}
	repoRoot, err := findRepoRoot("")
	if err != nil {
		return err
	}
	for _, name := range strings.Split(*devices, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		h := BuildDeviceHome(*root, name)
		if err := h.Seed(repoRoot); err != nil {
			return fmt.Errorf("seed %s: %w", name, err)
		}
		fmt.Fprintf(os.Stderr, "hoplab: seeded %s (project %s) under %s\n", name, h.Project, h.Root)
	}
	return nil
}

func cmdEnv(argv []string) error {
	fs := flag.NewFlagSet("env", flag.ExitOnError)
	root := fs.String("root", "", "lab root directory (required)")
	device := fs.String("device", "device-a", "which seeded device's home to print")
	shell := fs.String("shell", defaultShell(), "sh or powershell")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if strings.TrimSpace(*root) == "" {
		return fmt.Errorf("-root is required")
	}
	s, err := loadState(*root)
	if err != nil {
		if os.IsNotExist(err) {
			// env can still be useful before/without a running lab (the
			// caller may only want the isolated-home variables); leave
			// REINSTATE_HOP_URL out rather than guess a URL.
			s = LabState{}
		} else {
			return err
		}
	}
	h := BuildDeviceHome(*root, *device)
	pairs := hopLabEnv(s, h)
	if s.HopdBaseURL == "" {
		pairs = pairs[1:] // drop the empty REINSTATE_HOP_URL entry
	}
	printEnv(os.Stdout, *shell, pairs)
	return nil
}

func cmdKeyring(argv []string) error {
	if len(argv) == 0 {
		return fmt.Errorf("want save|load|clear")
	}
	action, rest := argv[0], argv[1:]
	fs := flag.NewFlagSet("keyring "+action, flag.ExitOnError)
	root := fs.String("root", "", "lab root directory")
	device := fs.String("device", "", "device name (matches a name given to `hoplab homes -devices`)")
	if err := fs.Parse(rest); err != nil {
		return err
	}
	switch action {
	case "save":
		if *root == "" || *device == "" {
			return fmt.Errorf("save needs -root and -device")
		}
		if err := keyringSave(*root, *device); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "hoplab: saved the current OS keyring device token as %s\n", *device)
	case "load":
		if *root == "" || *device == "" {
			return fmt.Errorf("load needs -root and -device")
		}
		if err := keyringLoad(*root, *device); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "hoplab: the OS keyring now holds %s's device token; `rein whoami` acts as it\n", *device)
	case "clear":
		if err := keyringClear(); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "hoplab: cleared the OS keyring device token")
	default:
		return fmt.Errorf("want save|load|clear, got %q", action)
	}
	return nil
}

func defaultShell() string {
	if strings.EqualFold(os.Getenv("OS"), "Windows_NT") {
		return "powershell"
	}
	return "sh"
}
