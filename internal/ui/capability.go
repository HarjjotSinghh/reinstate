// Package ui holds presentation-layer decisions shared by the plain and
// interactive Reinstate surfaces: what a terminal can render, which theme and
// glyphs to use, and how agents and timestamps are labelled.
//
// ui never reads session data, never talks to a vendor, and never decides what
// an action does. It decides only how something looks. internal/tui builds on
// it; engine packages never import either.
//
// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.
package ui

import (
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// Mode is the rendering mode selected by the degradation ladder.
type Mode int

const (
	// ModePlain is the frozen non-interactive output that scripts depend on.
	// It is byte-identical to what Reinstate emitted before the TUI existed.
	ModePlain Mode = iota
	// ModeCompact is the interactive TUI without a side preview pane, chosen
	// when the terminal is too narrow to split.
	ModeCompact
	// ModeFull is the interactive TUI with every pane.
	ModeFull
)

// String returns a stable lowercase token for logs and tests.
func (m Mode) String() string {
	switch m {
	case ModePlain:
		return "plain"
	case ModeCompact:
		return "compact"
	case ModeFull:
		return "full"
	default:
		return "unknown"
	}
}

// Interactive reports whether the mode runs a Bubble Tea program.
func (m Mode) Interactive() bool { return m == ModeCompact || m == ModeFull }

// ColorDepth is how much colour the stream accepts.
type ColorDepth int

const (
	ColorNone ColorDepth = iota // monochrome; NO_COLOR or a dumb terminal
	Color16                     // ANSI base colours
	Color256                    // xterm-256
	ColorTrue                   // 24-bit
)

// String returns a stable lowercase token.
func (c ColorDepth) String() string {
	switch c {
	case ColorNone:
		return "none"
	case Color16:
		return "16"
	case Color256:
		return "256"
	case ColorTrue:
		return "truecolor"
	default:
		return "unknown"
	}
}

// MinSplitWidth is the narrowest terminal that still gets a preview pane.
// Below it the switcher renders a single column.
const MinSplitWidth = 80

// MinInteractiveHeight is the shortest terminal that can host a full-screen
// program with a header, a body, and a key bar without thrashing.
const MinInteractiveHeight = 10

// fallbackWidth and fallbackHeight are used when a size probe fails but the
// stream is still a terminal.
const (
	fallbackWidth  = 80
	fallbackHeight = 24
)

// Reason explains why the ladder produced a non-interactive mode. It is empty
// when the mode is interactive. Callers surface it only in diagnostics.
type Reason string

const (
	ReasonJSON        Reason = "json_output"
	ReasonPlainFlag   Reason = "plain_flag"
	ReasonEnvDisabled Reason = "reinstate_no_tui"
	ReasonNotTTY      Reason = "not_a_terminal"
	ReasonDumbTerm    Reason = "dumb_terminal"
	ReasonCI          Reason = "ci_environment"
	ReasonTooSmall    Reason = "terminal_too_small"
)

// Capability is the resolved rendering environment for one command run.
type Capability struct {
	Mode    Mode
	Reason  Reason
	Color   ColorDepth
	Unicode bool
	Width   int
	Height  int
}

// Split reports whether a side-by-side layout fits.
func (c Capability) Split() bool { return c.Mode == ModeFull }

// Options are the caller-supplied inputs to Detect. Everything the ladder
// consults is injected so the matrix is testable without a real terminal.
type Options struct {
	// JSON is the resolved --json flag. It forces plain unconditionally.
	JSON bool
	// Plain is the resolved --plain flag.
	Plain bool
	// Getenv reads the environment. nil selects os.Getenv.
	Getenv func(string) string
	// TerminalCheck reports whether both streams are terminals. nil selects
	// the real golang.org/x/term probe.
	TerminalCheck func(io.Reader, io.Writer) bool
	// Size returns the terminal dimensions. nil selects the real probe.
	Size func(io.Writer) (width, height int, err error)
}

// Detect applies the degradation ladder and returns the resolved capability.
//
// The ladder is checked in order; the first match wins:
//
//  1. --json                                   -> plain
//  2. not a TTY on either stream               -> plain
//  3. --plain, REINSTATE_NO_TUI, TERM=dumb, CI -> plain
//  4. terminal too small                       -> plain
//  5. NO_COLOR                                 -> interactive, no colour
//  6. width below MinSplitWidth                -> compact
//  7. otherwise                                -> full
//
// Plain mode must remain byte-identical to the pre-TUI output, so callers treat
// ModePlain as "render exactly what we always rendered".
func Detect(in io.Reader, out io.Writer, opts Options) Capability {
	getenv := opts.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	terminalCheck := opts.TerminalCheck
	if terminalCheck == nil {
		terminalCheck = defaultTerminalCheck
	}
	sizeProbe := opts.Size
	if sizeProbe == nil {
		sizeProbe = defaultSize
	}

	plain := func(reason Reason) Capability {
		return Capability{
			Mode:    ModePlain,
			Reason:  reason,
			Color:   ColorNone,
			Unicode: false,
			Width:   0,
			Height:  0,
		}
	}

	// 1. JSON output can never be interleaved with a redraw.
	if opts.JSON {
		return plain(ReasonJSON)
	}
	// 2. A pipe or a file is not something we can drive.
	if !terminalCheck(in, out) {
		return plain(ReasonNotTTY)
	}
	// 3. Explicit opt-outs and environments that lie about being a terminal.
	if opts.Plain {
		return plain(ReasonPlainFlag)
	}
	if envTruthy(getenv("REINSTATE_NO_TUI")) {
		return plain(ReasonEnvDisabled)
	}
	if isDumbTerm(getenv("TERM")) {
		return plain(ReasonDumbTerm)
	}
	if isCI(getenv) {
		return plain(ReasonCI)
	}

	width, height, err := sizeProbe(out)
	if err != nil || width <= 0 || height <= 0 {
		width, height = fallbackWidth, fallbackHeight
	}
	// Deterministic acceptance evidence needs a fixed frame size regardless of
	// the console the harness happens to get.
	if override, ok := parseDimension(getenv("REINSTATE_TUI_COLS")); ok {
		width = override
	}
	if override, ok := parseDimension(getenv("REINSTATE_TUI_ROWS")); ok {
		height = override
	}
	// 4. Too small to host a full-screen program without thrashing.
	if height < MinInteractiveHeight || width < 40 {
		capability := plain(ReasonTooSmall)
		capability.Width = width
		capability.Height = height
		return capability
	}

	capability := Capability{
		Color:   detectColor(getenv),
		Unicode: detectUnicode(getenv),
		Width:   width,
		Height:  height,
	}
	// 5 and 6. Colour and split are independent degradations.
	if width < MinSplitWidth {
		capability.Mode = ModeCompact
	} else {
		capability.Mode = ModeFull
	}
	return capability
}

func defaultTerminalCheck(in io.Reader, out io.Writer) bool {
	inputFile, inputOK := in.(*os.File)
	outputFile, outputOK := out.(*os.File)
	if !inputOK || !outputOK {
		return false
	}
	return term.IsTerminal(int(inputFile.Fd())) && term.IsTerminal(int(outputFile.Fd()))
}

func defaultSize(out io.Writer) (int, int, error) { return TerminalSize(out) }

// TerminalSize reports the width and height of out when it is a real terminal.
// It is exported so callers that must build their own size probe still measure
// the console the same way the ladder does.
func TerminalSize(out io.Writer) (int, int, error) {
	file, ok := out.(*os.File)
	if !ok {
		return 0, 0, errNotFile
	}
	return term.GetSize(int(file.Fd()))
}

type sizeError string

func (e sizeError) Error() string { return string(e) }

const errNotFile = sizeError("output stream is not a file")

// envTruthy treats the documented truthy spellings as enabled. An unset or
// empty value is false, so exporting REINSTATE_NO_TUI= does not disable the UI.
func envTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// isDumbTerm reports whether TERM names a terminal that cannot host a
// full-screen program.
//
// An explicit TERM=dumb always means dumb. An *unset* TERM does not mean the
// same thing on every platform: on Unix it means there is no terminfo entry and
// nothing can be assumed, but on Windows TERM is simply not part of the
// environment — cmd.exe, PowerShell, Windows Terminal, and ConPTY all leave it
// unset while supporting virtual terminal sequences perfectly well. Treating
// unset as dumb there would mean the interactive UI never appears on Windows at
// all, which is the platform half of the flagship multi-device case.
func isDumbTerm(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "dumb" {
		return true
	}
	return normalized == "" && unsetTermIsDumb
}

// isCI matches the CI variable that every major provider sets, plus the
// provider-specific ones that do not always set it.
func isCI(getenv func(string) string) bool {
	if envTruthy(getenv("CI")) {
		return true
	}
	for _, name := range []string{
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"BUILDKITE",
		"CIRCLECI",
		"TF_BUILD",
	} {
		if strings.TrimSpace(getenv(name)) != "" {
			return true
		}
	}
	return false
}

// detectColor resolves colour depth. NO_COLOR wins over every positive signal,
// per the no-color.org contract: any non-empty value disables colour.
func detectColor(getenv func(string) string) ColorDepth {
	if _, set := lookup(getenv, "NO_COLOR"); set {
		return ColorNone
	}
	if forced := strings.TrimSpace(getenv("CLICOLOR_FORCE")); forced != "" && forced != "0" {
		return ColorTrue
	}
	colorTerm := strings.ToLower(getenv("COLORTERM"))
	if strings.Contains(colorTerm, "truecolor") || strings.Contains(colorTerm, "24bit") {
		return ColorTrue
	}
	termValue := strings.ToLower(strings.TrimSpace(getenv("TERM")))
	switch {
	case strings.Contains(termValue, "256color"):
		return Color256
	case termValue == "dumb":
		return ColorNone
	case termValue == "":
		// Same reasoning as isDumbTerm: on Windows an unset TERM says nothing
		// about the console's capability. Anything that got past the ladder
		// there is a virtual-terminal console, which handles 256 colours.
		if unsetTermIsDumb {
			return ColorNone
		}
		return Color256
	default:
		return Color16
	}
}

// lookup reports whether a variable is present, distinguishing unset from
// empty. NO_COLOR is presence-sensitive, unlike the truthy variables.
func lookup(getenv func(string) string, name string) (string, bool) {
	value := getenv(name)
	if value != "" {
		return value, true
	}
	// A Getenv-only interface cannot distinguish unset from empty, so treat a
	// present-but-empty NO_COLOR as unset. That matches os.Getenv semantics and
	// avoids disabling colour for callers that export the name blank.
	return "", false
}

// detectUnicode reports whether box-drawing and status glyphs are safe.
//
// Windows is the reason this is not simply "true": legacy conhost code pages
// render multi-byte glyphs as mojibake. Windows Terminal and any UTF-8 locale
// are fine.
func detectUnicode(getenv func(string) string) bool {
	for _, name := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		value := strings.ToLower(getenv(name))
		if value == "" {
			continue
		}
		return strings.Contains(value, "utf-8") || strings.Contains(value, "utf8")
	}
	// No locale variables at all is the normal Windows case. Trust the
	// terminal identity instead.
	if strings.TrimSpace(getenv("WT_SESSION")) != "" {
		return true
	}
	return windowsUnicodeDefault(getenv)
}

// parseDimension is a small helper for environment-provided overrides used by
// the deterministic frame tests.
func parseDimension(value string) (int, bool) {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number <= 0 {
		return 0, false
	}
	return number, true
}
