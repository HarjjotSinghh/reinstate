// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package ui

import (
	"io"
	"runtime"
	"testing"
)

// The capability tests never read the real environment and never probe a real
// terminal: every input the ladder consults arrives through Options, so the
// matrix is identical on a developer laptop, in CI, and on the Windows bench.

// fakeEnv builds a Getenv over a fixed map. An absent name reads as empty,
// which is exactly what os.Getenv does for an unset variable.
func fakeEnv(values map[string]string) func(string) string {
	return func(name string) string { return values[name] }
}

// interactiveEnv is the baseline environment that reaches the interactive end
// of the ladder: a colour-capable TERM and a UTF-8 locale. Overrides are
// applied on top, and an override to "" reads as unset.
func interactiveEnv(overrides map[string]string) map[string]string {
	env := map[string]string{
		"TERM": "xterm-256color",
		"LANG": "en_US.UTF-8",
	}
	for name, value := range overrides {
		env[name] = value
	}
	return env
}

func fakeTerminalCheck(isTerminal bool) func(io.Reader, io.Writer) bool {
	return func(io.Reader, io.Writer) bool { return isTerminal }
}

func fakeSize(width, height int, err error) func(io.Writer) (int, int, error) {
	return func(io.Writer) (int, int, error) { return width, height, err }
}

type probeError string

func (e probeError) Error() string { return string(e) }

const errProbeFailed = probeError("ioctl failed")

// detectCase is one rung of the ladder. cols and rows are the terminal size the
// probe reports; both zero (with no probe error) means the standard 120x40
// window, which is comfortably interactive.
type detectCase struct {
	name       string
	json       bool
	plain      bool
	notTTY     bool
	env        map[string]string
	cols, rows int
	probeErr   error

	wantMode   Mode
	wantReason Reason
	// wantWidth and wantHeight are asserted only when wantSize is true, so rows
	// that care about the ladder alone stay readable.
	wantSize              bool
	wantWidth, wantHeight int
}

func (tc detectCase) run(t *testing.T) {
	t.Helper()
	cols, rows := tc.cols, tc.rows
	if cols == 0 && rows == 0 && tc.probeErr == nil {
		cols, rows = 120, 40
	}
	env := tc.env
	if env == nil {
		env = interactiveEnv(nil)
	}
	got := Detect(nil, nil, Options{
		JSON:          tc.json,
		Plain:         tc.plain,
		Getenv:        fakeEnv(env),
		TerminalCheck: fakeTerminalCheck(!tc.notTTY),
		Size:          fakeSize(cols, rows, tc.probeErr),
	})
	if got.Mode != tc.wantMode {
		t.Errorf("Detect().Mode = %s, want %s", got.Mode, tc.wantMode)
	}
	if got.Reason != tc.wantReason {
		t.Errorf("Detect().Reason = %q, want %q", got.Reason, tc.wantReason)
	}
	if tc.wantSize && (got.Width != tc.wantWidth || got.Height != tc.wantHeight) {
		t.Errorf("Detect() size = %dx%d, want %dx%d", got.Width, got.Height, tc.wantWidth, tc.wantHeight)
	}
	if got.Mode == ModePlain {
		if got.Color != ColorNone {
			t.Errorf("plain Detect().Color = %s, want none", got.Color)
		}
		if got.Unicode {
			t.Error("plain Detect().Unicode = true, want false")
		}
		if got.Split() {
			t.Error("plain Detect().Split() = true, want false")
		}
	} else if got.Reason != "" {
		t.Errorf("interactive Detect().Reason = %q, want empty", got.Reason)
	}
}

// TestDetectLadderOrder walks the degradation ladder rung by rung, including
// the pairs that prove the order: when two rungs both match, the earlier one
// must own the Reason.
func TestDetectLadderOrder(t *testing.T) {
	cases := []detectCase{
		{
			name:     "json forces plain on a real terminal",
			json:     true,
			wantMode: ModePlain, wantReason: ReasonJSON,
			wantSize: true, wantWidth: 0, wantHeight: 0,
		},
		{
			name: "json outranks every later rung",
			json: true, notTTY: true, plain: true,
			env:  interactiveEnv(map[string]string{"REINSTATE_NO_TUI": "1", "TERM": "dumb", "CI": "true"}),
			cols: 10, rows: 2,
			wantMode: ModePlain, wantReason: ReasonJSON,
		},
		{
			name:     "a pipe is not a terminal",
			notTTY:   true,
			wantMode: ModePlain, wantReason: ReasonNotTTY,
			wantSize: true, wantWidth: 0, wantHeight: 0,
		},
		{
			name:   "not a terminal outranks the plain flag",
			notTTY: true, plain: true,
			wantMode: ModePlain, wantReason: ReasonNotTTY,
		},
		{
			name:     "not a terminal outranks the env opt-out",
			notTTY:   true,
			env:      interactiveEnv(map[string]string{"REINSTATE_NO_TUI": "1"}),
			wantMode: ModePlain, wantReason: ReasonNotTTY,
		},
		{
			name:     "plain flag",
			plain:    true,
			wantMode: ModePlain, wantReason: ReasonPlainFlag,
		},
		{
			name:     "plain flag outranks the env opt-out",
			plain:    true,
			env:      interactiveEnv(map[string]string{"REINSTATE_NO_TUI": "1"}),
			wantMode: ModePlain, wantReason: ReasonPlainFlag,
		},
		{
			name:     "env opt-out",
			env:      interactiveEnv(map[string]string{"REINSTATE_NO_TUI": "1"}),
			wantMode: ModePlain, wantReason: ReasonEnvDisabled,
		},
		{
			name:     "env opt-out outranks a dumb terminal",
			env:      interactiveEnv(map[string]string{"REINSTATE_NO_TUI": "yes", "TERM": "dumb"}),
			wantMode: ModePlain, wantReason: ReasonEnvDisabled,
		},
		{
			name:     "dumb terminal",
			env:      interactiveEnv(map[string]string{"TERM": "dumb"}),
			wantMode: ModePlain, wantReason: ReasonDumbTerm,
		},
		{
			name:     "dumb terminal spelled in mixed case",
			env:      interactiveEnv(map[string]string{"TERM": "  DUMB "}),
			wantMode: ModePlain, wantReason: ReasonDumbTerm,
		},
		{
			// An unset TERM does not mean the same thing everywhere. On Unix it
			// means there is no terminfo entry and nothing can be assumed. On
			// Windows TERM is simply not part of the environment, and every
			// console there supports virtual terminal sequences, so treating
			// unset as dumb would mean the interactive UI never appears on the
			// platform that is half of the flagship multi-device case.
			name:       "unset TERM is dumb only where TERM is meaningful",
			env:        interactiveEnv(map[string]string{"TERM": ""}),
			wantMode:   unsetTermMode(),
			wantReason: unsetTermReason(),
		},
		{
			name:     "dumb terminal outranks CI",
			env:      interactiveEnv(map[string]string{"TERM": "dumb", "CI": "true"}),
			wantMode: ModePlain, wantReason: ReasonDumbTerm,
		},
		{
			name:     "CI",
			env:      interactiveEnv(map[string]string{"CI": "true"}),
			wantMode: ModePlain, wantReason: ReasonCI,
		},
		{
			name: "CI outranks a too-small window",
			env:  interactiveEnv(map[string]string{"CI": "1"}),
			cols: 20, rows: 4,
			wantMode: ModePlain, wantReason: ReasonCI,
			wantSize: true, wantWidth: 0, wantHeight: 0,
		},
		{
			name: "too short keeps the probed size for diagnostics",
			cols: 200, rows: MinInteractiveHeight - 1,
			wantMode: ModePlain, wantReason: ReasonTooSmall,
			wantSize: true, wantWidth: 200, wantHeight: MinInteractiveHeight - 1,
		},
		{
			name: "too narrow keeps the probed size for diagnostics",
			cols: 39, rows: 40,
			wantMode: ModePlain, wantReason: ReasonTooSmall,
			wantSize: true, wantWidth: 39, wantHeight: 40,
		},
		{
			name: "one cell too small in both directions",
			cols: 39, rows: 9,
			wantMode: ModePlain, wantReason: ReasonTooSmall,
			wantSize: true, wantWidth: 39, wantHeight: 9,
		},
		{
			name: "the narrowest interactive width",
			cols: 40, rows: MinInteractiveHeight,
			wantMode: ModeCompact, wantReason: "",
			wantSize: true, wantWidth: 40, wantHeight: MinInteractiveHeight,
		},
		{
			name: "one cell below the split width is compact",
			cols: MinSplitWidth - 1, rows: 24,
			wantMode: ModeCompact, wantReason: "",
			wantSize: true, wantWidth: MinSplitWidth - 1, wantHeight: 24,
		},
		{
			name: "exactly the split width is full",
			cols: MinSplitWidth, rows: 24,
			wantMode: ModeFull, wantReason: "",
			wantSize: true, wantWidth: MinSplitWidth, wantHeight: 24,
		},
		{
			name: "a wide window is full",
			cols: 200, rows: 60,
			wantMode: ModeFull, wantReason: "",
			wantSize: true, wantWidth: 200, wantHeight: 60,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { tc.run(t) })
	}
}

// TestDetectSplitFollowsMode pins Split to the full mode rather than to the
// width, so a caller can never draw a preview pane in compact mode.
func TestDetectSplitFollowsMode(t *testing.T) {
	cases := []struct {
		name string
		cols int
		want bool
	}{
		{name: "compact does not split", cols: 60, want: false},
		{name: "full splits", cols: 120, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Detect(nil, nil, Options{
				Getenv:        fakeEnv(interactiveEnv(nil)),
				TerminalCheck: fakeTerminalCheck(true),
				Size:          fakeSize(tc.cols, 40, nil),
			})
			if got.Split() != tc.want {
				t.Fatalf("Detect(cols=%d).Split() = %t, want %t", tc.cols, got.Split(), tc.want)
			}
		})
	}
}

// TestDetectEnvDisableSpellings pins the documented truthy spellings of
// REINSTATE_NO_TUI, and just as importantly the spellings that must NOT disable
// the UI: exporting the name empty or zero has to leave the TUI on.
func TestDetectEnvDisableSpellings(t *testing.T) {
	cases := []struct {
		value       string
		wantDisable bool
	}{
		{value: "1", wantDisable: true},
		{value: "true", wantDisable: true},
		{value: "TRUE", wantDisable: true},
		{value: "yes", wantDisable: true},
		{value: "Yes", wantDisable: true},
		{value: "on", wantDisable: true},
		{value: "  on  ", wantDisable: true},
		{value: "", wantDisable: false},
		{value: "0", wantDisable: false},
		{value: "false", wantDisable: false},
		{value: "no", wantDisable: false},
		{value: "off", wantDisable: false},
		{value: "2", wantDisable: false},
		{value: "garbage", wantDisable: false},
		{value: "enabled", wantDisable: false},
	}
	for _, tc := range cases {
		name := tc.value
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			got := Detect(nil, nil, Options{
				Getenv:        fakeEnv(interactiveEnv(map[string]string{"REINSTATE_NO_TUI": tc.value})),
				TerminalCheck: fakeTerminalCheck(true),
				Size:          fakeSize(120, 40, nil),
			})
			disabled := got.Mode == ModePlain
			if disabled != tc.wantDisable {
				t.Fatalf("REINSTATE_NO_TUI=%q -> mode %s (reason %q), want disabled=%t",
					tc.value, got.Mode, got.Reason, tc.wantDisable)
			}
			if disabled && got.Reason != ReasonEnvDisabled {
				t.Fatalf("REINSTATE_NO_TUI=%q -> reason %q, want %q", tc.value, got.Reason, ReasonEnvDisabled)
			}
		})
	}
}

// TestDetectCIEnvironments covers both CI shapes: CI itself is truthy-parsed,
// while a provider-specific variable counts whenever it is present at all.
func TestDetectCIEnvironments(t *testing.T) {
	cases := []struct {
		name   string
		env    map[string]string
		wantCI bool
	}{
		{name: "CI=true", env: map[string]string{"CI": "true"}, wantCI: true},
		{name: "CI=1", env: map[string]string{"CI": "1"}, wantCI: true},
		{name: "CI=yes", env: map[string]string{"CI": "yes"}, wantCI: true},
		{name: "CI=on", env: map[string]string{"CI": "on"}, wantCI: true},
		{name: "CI=0", env: map[string]string{"CI": "0"}, wantCI: false},
		{name: "CI=false", env: map[string]string{"CI": "false"}, wantCI: false},
		{name: "CI empty", env: map[string]string{"CI": ""}, wantCI: false},
		{name: "CI garbage", env: map[string]string{"CI": "maybe"}, wantCI: false},
		{name: "GITHUB_ACTIONS", env: map[string]string{"GITHUB_ACTIONS": "true"}, wantCI: true},
		{name: "GITLAB_CI", env: map[string]string{"GITLAB_CI": "true"}, wantCI: true},
		{name: "BUILDKITE", env: map[string]string{"BUILDKITE": "true"}, wantCI: true},
		{name: "CIRCLECI", env: map[string]string{"CIRCLECI": "true"}, wantCI: true},
		{name: "TF_BUILD", env: map[string]string{"TF_BUILD": "True"}, wantCI: true},
		{
			name:   "a provider variable counts even when it reads false",
			env:    map[string]string{"GITHUB_ACTIONS": "0"},
			wantCI: true,
		},
		{
			name:   "a whitespace-only provider variable does not count",
			env:    map[string]string{"GITHUB_ACTIONS": "   "},
			wantCI: false,
		},
		{
			name:   "an unrelated variable does not count",
			env:    map[string]string{"JENKINS_HOME": "/var/jenkins"},
			wantCI: false,
		},
		{name: "no CI variables", env: nil, wantCI: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Detect(nil, nil, Options{
				Getenv:        fakeEnv(interactiveEnv(tc.env)),
				TerminalCheck: fakeTerminalCheck(true),
				Size:          fakeSize(120, 40, nil),
			})
			inCI := got.Mode == ModePlain && got.Reason == ReasonCI
			if inCI != tc.wantCI {
				t.Fatalf("Detect() = %s/%q, want CI detection = %t", got.Mode, got.Reason, tc.wantCI)
			}
		})
	}
}

// TestDetectSizeResolution covers the size probe, its fallback, and the
// deterministic overrides the acceptance harness relies on.
func TestDetectSizeResolution(t *testing.T) {
	cases := []struct {
		name                  string
		cols, rows            int
		probeErr              error
		env                   map[string]string
		wantWidth, wantHeight int
		wantMode              Mode
		wantReason            Reason
	}{
		{
			name: "the probe is used when it succeeds",
			cols: 132, rows: 43,
			wantWidth: 132, wantHeight: 43, wantMode: ModeFull,
		},
		{
			name:      "a failing probe falls back to 80x24",
			probeErr:  errProbeFailed,
			wantWidth: fallbackWidth, wantHeight: fallbackHeight, wantMode: ModeFull,
		},
		{
			name: "a zero-sized probe falls back to 80x24",
			cols: 0, rows: 0, probeErr: nil,
			wantWidth: fallbackWidth, wantHeight: fallbackHeight, wantMode: ModeFull,
		},
		{
			name: "a negative probe falls back to 80x24",
			cols: -1, rows: -1,
			wantWidth: fallbackWidth, wantHeight: fallbackHeight, wantMode: ModeFull,
		},
		{
			name: "the column override wins over the probe",
			cols: 200, rows: 60,
			env:       map[string]string{"REINSTATE_TUI_COLS": "60"},
			wantWidth: 60, wantHeight: 60, wantMode: ModeCompact,
		},
		{
			name: "the row override wins over the probe",
			cols: 200, rows: 60,
			env:       map[string]string{"REINSTATE_TUI_ROWS": "30"},
			wantWidth: 200, wantHeight: 30, wantMode: ModeFull,
		},
		{
			name: "both overrides win over the probe",
			cols: 200, rows: 60,
			env:       map[string]string{"REINSTATE_TUI_COLS": "100", "REINSTATE_TUI_ROWS": "28"},
			wantWidth: 100, wantHeight: 28, wantMode: ModeFull,
		},
		{
			name:      "overrides win over the fallback too",
			probeErr:  errProbeFailed,
			env:       map[string]string{"REINSTATE_TUI_COLS": "50", "REINSTATE_TUI_ROWS": "12"},
			wantWidth: 50, wantHeight: 12, wantMode: ModeCompact,
		},
		{
			name: "a padded override is still parsed",
			cols: 200, rows: 60,
			env:       map[string]string{"REINSTATE_TUI_COLS": "  90  "},
			wantWidth: 90, wantHeight: 60, wantMode: ModeFull,
		},
		{
			name: "a zero override is ignored",
			cols: 120, rows: 40,
			env:       map[string]string{"REINSTATE_TUI_COLS": "0", "REINSTATE_TUI_ROWS": "0"},
			wantWidth: 120, wantHeight: 40, wantMode: ModeFull,
		},
		{
			name: "a negative override is ignored",
			cols: 120, rows: 40,
			env:       map[string]string{"REINSTATE_TUI_COLS": "-10"},
			wantWidth: 120, wantHeight: 40, wantMode: ModeFull,
		},
		{
			name: "a non-numeric override is ignored",
			cols: 120, rows: 40,
			env:       map[string]string{"REINSTATE_TUI_COLS": "wide", "REINSTATE_TUI_ROWS": ""},
			wantWidth: 120, wantHeight: 40, wantMode: ModeFull,
		},
		{
			name: "an override can push the terminal below the interactive floor",
			cols: 200, rows: 60,
			env:        map[string]string{"REINSTATE_TUI_ROWS": "5"},
			wantWidth:  200,
			wantHeight: 5,
			wantMode:   ModePlain,
			wantReason: ReasonTooSmall,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Detect(nil, nil, Options{
				Getenv:        fakeEnv(interactiveEnv(tc.env)),
				TerminalCheck: fakeTerminalCheck(true),
				Size:          fakeSize(tc.cols, tc.rows, tc.probeErr),
			})
			if got.Width != tc.wantWidth || got.Height != tc.wantHeight {
				t.Errorf("Detect() size = %dx%d, want %dx%d", got.Width, got.Height, tc.wantWidth, tc.wantHeight)
			}
			if got.Mode != tc.wantMode {
				t.Errorf("Detect().Mode = %s, want %s", got.Mode, tc.wantMode)
			}
			if got.Reason != tc.wantReason {
				t.Errorf("Detect().Reason = %q, want %q", got.Reason, tc.wantReason)
			}
		})
	}
}

// TestDetectColorDepth exercises detectColor through Detect, which is the only
// path production code takes. TERM is never empty or dumb here because the
// ladder turns those into plain mode before colour is ever resolved.
func TestDetectColorDepth(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want ColorDepth
	}{
		{
			name: "NO_COLOR beats COLORTERM and CLICOLOR_FORCE",
			env:  map[string]string{"NO_COLOR": "1", "COLORTERM": "truecolor", "CLICOLOR_FORCE": "1"},
			want: ColorNone,
		},
		{
			name: "any NO_COLOR value disables colour",
			env:  map[string]string{"NO_COLOR": "0"},
			want: ColorNone,
		},
		{
			name: "an empty NO_COLOR reads as unset",
			env:  map[string]string{"NO_COLOR": "", "COLORTERM": "truecolor"},
			want: ColorTrue,
		},
		{
			name: "CLICOLOR_FORCE forces truecolor",
			env:  map[string]string{"CLICOLOR_FORCE": "1", "TERM": "xterm"},
			want: ColorTrue,
		},
		{
			name: "CLICOLOR_FORCE=0 forces nothing",
			env:  map[string]string{"CLICOLOR_FORCE": "0", "TERM": "xterm"},
			want: Color16,
		},
		{
			name: "COLORTERM=truecolor",
			env:  map[string]string{"COLORTERM": "truecolor", "TERM": "xterm"},
			want: ColorTrue,
		},
		{
			name: "COLORTERM=24bit",
			env:  map[string]string{"COLORTERM": "24bit", "TERM": "xterm"},
			want: ColorTrue,
		},
		{
			name: "COLORTERM is matched case-insensitively",
			env:  map[string]string{"COLORTERM": "TrueColor", "TERM": "xterm"},
			want: ColorTrue,
		},
		{
			name: "a non-truecolor COLORTERM falls through to TERM",
			env:  map[string]string{"COLORTERM": "yes", "TERM": "xterm-256color"},
			want: Color256,
		},
		{
			name: "TERM=xterm-256color",
			env:  map[string]string{"TERM": "xterm-256color"},
			want: Color256,
		},
		{
			name: "TERM=screen-256color",
			env:  map[string]string{"TERM": "screen-256color"},
			want: Color256,
		},
		{
			name: "a plain TERM gets the base sixteen",
			env:  map[string]string{"TERM": "xterm"},
			want: Color16,
		},
		{
			name: "an unrecognised TERM still gets the base sixteen",
			env:  map[string]string{"TERM": "vt220"},
			want: Color16,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Detect(nil, nil, Options{
				Getenv:        fakeEnv(interactiveEnv(tc.env)),
				TerminalCheck: fakeTerminalCheck(true),
				Size:          fakeSize(120, 40, nil),
			})
			if !got.Mode.Interactive() {
				t.Fatalf("Detect() = %s/%q, want an interactive mode", got.Mode, got.Reason)
			}
			if got.Color != tc.want {
				t.Fatalf("Detect().Color = %s, want %s", got.Color, tc.want)
			}
		})
	}
}

// TestDetectUnicode covers glyph safety. The locale variables are decisive on
// every platform; only the case with no locale at all is platform-specific,
// because legacy conhost is the reason the check exists.
func TestDetectUnicode(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{name: "LANG carries UTF-8", env: map[string]string{"LANG": "en_US.UTF-8"}, want: true},
		{name: "LANG carries utf8", env: map[string]string{"LANG": "en_US.utf8"}, want: true},
		{name: "LANG is C", env: map[string]string{"LANG": "C"}, want: false},
		{name: "LANG is a legacy code page", env: map[string]string{"LANG": "en_US.ISO-8859-1"}, want: false},
		{
			name: "LC_ALL outranks LANG",
			env:  map[string]string{"LC_ALL": "C", "LANG": "en_US.UTF-8"},
			want: false,
		},
		{
			name: "LC_CTYPE outranks LANG",
			env:  map[string]string{"LC_CTYPE": "en_US.UTF-8", "LANG": "C"},
			want: true,
		},
		{
			name: "Windows Terminal is trusted without a locale",
			env:  map[string]string{"LANG": "", "WT_SESSION": "b0f0-1"},
			want: true,
		},
		{
			name: "a locale still outranks the terminal identity",
			env:  map[string]string{"LANG": "C", "WT_SESSION": "b0f0-1"},
			want: false,
		},
		{
			name: "no locale and no terminal identity",
			env:  map[string]string{"LANG": ""},
			// Unix terminals all render box drawing; legacy conhost does not, so
			// Windows withholds trust unless the terminal identifies itself.
			want: runtime.GOOS != "windows",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Detect(nil, nil, Options{
				Getenv:        fakeEnv(interactiveEnv(tc.env)),
				TerminalCheck: fakeTerminalCheck(true),
				Size:          fakeSize(120, 40, nil),
			})
			if !got.Mode.Interactive() {
				t.Fatalf("Detect() = %s/%q, want an interactive mode", got.Mode, got.Reason)
			}
			if got.Unicode != tc.want {
				t.Fatalf("Detect().Unicode = %t, want %t", got.Unicode, tc.want)
			}
		})
	}
}

func TestModeString(t *testing.T) {
	cases := []struct {
		mode Mode
		want string
	}{
		{mode: ModePlain, want: "plain"},
		{mode: ModeCompact, want: "compact"},
		{mode: ModeFull, want: "full"},
		{mode: Mode(42), want: "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.mode.String(); got != tc.want {
				t.Fatalf("Mode(%d).String() = %q, want %q", int(tc.mode), got, tc.want)
			}
		})
	}
}

func TestModeInteractive(t *testing.T) {
	cases := []struct {
		mode Mode
		want bool
	}{
		{mode: ModePlain, want: false},
		{mode: ModeCompact, want: true},
		{mode: ModeFull, want: true},
		{mode: Mode(42), want: false},
	}
	for _, tc := range cases {
		t.Run(tc.mode.String(), func(t *testing.T) {
			if got := tc.mode.Interactive(); got != tc.want {
				t.Fatalf("Mode(%d).Interactive() = %t, want %t", int(tc.mode), got, tc.want)
			}
		})
	}
}

func TestColorDepthString(t *testing.T) {
	cases := []struct {
		depth ColorDepth
		want  string
	}{
		{depth: ColorNone, want: "none"},
		{depth: Color16, want: "16"},
		{depth: Color256, want: "256"},
		{depth: ColorTrue, want: "truecolor"},
		{depth: ColorDepth(9), want: "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.depth.String(); got != tc.want {
				t.Fatalf("ColorDepth(%d).String() = %q, want %q", int(tc.depth), got, tc.want)
			}
		})
	}
}

func TestCapabilitySplit(t *testing.T) {
	cases := []struct {
		name string
		mode Mode
		want bool
	}{
		{name: "plain", mode: ModePlain, want: false},
		{name: "compact", mode: ModeCompact, want: false},
		{name: "full", mode: ModeFull, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			capability := Capability{Mode: tc.mode, Width: 200, Height: 60}
			if got := capability.Split(); got != tc.want {
				t.Fatalf("Capability{Mode:%s}.Split() = %t, want %t", tc.mode, got, tc.want)
			}
		})
	}
}

// unsetTermMode is the mode the ladder selects for an unset TERM on this
// platform. See the "unset TERM" case in TestDetectLadderOrder.
func unsetTermMode() Mode {
	if unsetTermIsDumb {
		return ModePlain
	}
	return ModeFull
}

// unsetTermReason is the matching reason, empty when the mode is interactive.
func unsetTermReason() Reason {
	if unsetTermIsDumb {
		return ReasonDumbTerm
	}
	return ""
}
