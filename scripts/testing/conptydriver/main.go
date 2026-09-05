// Command conptydriver is the Windows twin of scripts/testing/vendor-tty-driver.py:
// it runs a command under a real Windows pseudo console (ConPTY), drives it
// with a small step script, and snapshots what a person would actually see
// through a real VT parser -- not a regex strip, which conhost's own
// repaint strategy (rewriting a run of unchanged cells as a cursor-forward
// move rather than literal spaces) defeats.
//
// Usage:
//
//	conptydriver [-cols 80] [-rows 25] [-dir WORKDIR] -script steps.txt [-raw raw.log] -- <command> [args...]
//	conptydriver -- <command> [args...]     # interactive passthrough, no script
//
// See README.md next to this file for the step-script grammar and two
// traps worth knowing before writing one.
//
// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	fs := flag.NewFlagSet("conptydriver", flag.ContinueOnError)
	cols := fs.Int("cols", 80, "pseudo console width in columns")
	rows := fs.Int("rows", 25, "pseudo console height in rows")
	dir := fs.String("dir", "", "child working directory (default: this process's)")
	scriptPath := fs.String("script", "", "step-script file to run (see README.md); omitted means interactive passthrough")
	rawPath := fs.String("raw", "", "also write every raw output byte from the child to this file")
	exitTimeout := fs.Duration("exit-timeout", 15*time.Second, "how long to wait for the child to exit after the script finishes, before killing it")
	bg := fs.String("bg", "rgb:0000/0000/0000", "the XParseColor rgb: string this console reports for an OSC 11 background-colour query")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: conptydriver [flags] -- <command> [args...]\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(argv); err != nil {
		return 2
	}
	args := fs.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "conptydriver: a command is required after --")
		fs.Usage()
		return 2
	}

	var rawFile *os.File
	if *rawPath != "" {
		f, err := os.Create(*rawPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "conptydriver: %v\n", err)
			return 1
		}
		defer f.Close()
		rawFile = f
	}

	console, err := StartConsole(args, *cols, *rows, *dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "conptydriver: %v\n", err)
		return 1
	}
	defer console.Close()

	screen := NewScreen(*cols, *rows)
	screen.SetBackgroundColorReply(*bg)
	var rawWriter io.Writer
	if rawFile != nil {
		rawWriter = rawFile
	}
	runner := NewRunner(console, screen, rawWriter)
	go runner.Pump()

	if *scriptPath == "" {
		return runInteractive(console)
	}

	f, err := os.Open(*scriptPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "conptydriver: %v\n", err)
		return 1
	}
	steps, err := ParseScript(f)
	f.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "conptydriver: parsing %s: %v\n", *scriptPath, err)
		return 1
	}

	scriptErr := runner.Run(steps)

	code, waitErr := waitOrKill(console, *exitTimeout)
	switch {
	case scriptErr != nil:
		fmt.Fprintf(os.Stderr, "conptydriver: script failed: %v\n", scriptErr)
		return 1
	case waitErr != nil:
		fmt.Fprintf(os.Stderr, "conptydriver: %v\n", waitErr)
		return 1
	default:
		fmt.Fprintf(os.Stdout, "conptydriver: child exited %d\n", code)
		if code != 0 {
			return 1
		}
		return 0
	}
}

// waitOrKill waits for the child up to timeout, then kills it. A script
// that already sent "kill" or watched the child exit on its own returns
// immediately either way; this only matters for a script that finished
// without either.
func waitOrKill(console Console, timeout time.Duration) (int, error) {
	done := make(chan struct {
		code int
		err  error
	}, 1)
	go func() {
		code, err := console.Wait()
		done <- struct {
			code int
			err  error
		}{code, err}
	}()
	select {
	case r := <-done:
		return r.code, r.err
	case <-time.After(timeout):
		_ = console.Kill()
		r := <-done
		if r.err != nil {
			return r.code, fmt.Errorf("child did not exit within %s and was killed: %w", timeout, r.err)
		}
		return r.code, fmt.Errorf("child did not exit within %s and was killed", timeout)
	}
}

// runInteractive is a best-effort manual mode with no step script: stdin
// lines are sent to the child (with a trailing newline turned into "\r",
// the way pressing Enter reads to a terminal program) and the child's
// rendered frame is reprinted to stdout after each line. It is
// line-buffered, not a full keystroke-by-keystroke passthrough -- for that,
// use a real Windows Terminal session or write a step script.
func runInteractive(console Console) int {
	fmt.Fprintln(os.Stderr, "conptydriver: interactive mode (line-buffered; Ctrl-D / Ctrl-Z to send EOF and exit)")
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		if _, err := console.Write([]byte(sc.Text() + "\r")); err != nil {
			fmt.Fprintf(os.Stderr, "conptydriver: write: %v\n", err)
			break
		}
	}
	code, err := waitOrKillSoon(console)
	if err != nil {
		fmt.Fprintf(os.Stderr, "conptydriver: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "conptydriver: child exited %d\n", code)
	if code != 0 {
		return 1
	}
	return 0
}

func waitOrKillSoon(console Console) (int, error) {
	return waitOrKill(console, 5*time.Second)
}
