package main

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// StepKind names one step-script verb.
type StepKind int

const (
	StepWait StepKind = iota
	StepSend
	StepKey
	StepSnapshot
	StepSleep
	StepKill
)

// Step is one parsed line of a conptydriver step script. The grammar (one
// verb per line, blank lines and lines starting with "#" ignored):
//
//	wait /regex/ 10s        block until the rendered frame matches regex,
//	                        or fail after the timeout
//	send "text"             write text to the child's input, literally (Go
//	                        string-literal escapes such as \n and \" apply)
//	key enter|esc|tab|ctrl+k|up|down|space|<char>
//	                        write one named keystroke; ctrl+X sends the
//	                        control byte for letter X; a single character
//	                        not otherwise named is sent literally
//	snapshot path           render the current frame through the VT model
//	                        and write it to path (parent directories made
//	                        as needed)
//	sleep 500ms              a plain pause, for a step that needs no signal
//	                        to wait on
//	kill                     terminate the child immediately
type Step struct {
	Kind    StepKind
	Line    int // 1-indexed source line, for error messages
	Pattern *regexp.Regexp
	Timeout time.Duration
	Text    string
	Key     string
	Path    string
}

// ParseScript reads a step script. It never runs anything; it only parses.
func ParseScript(r io.Reader) ([]Step, error) {
	var steps []Step
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 4096), 1<<20)
	line := 0
	for sc.Scan() {
		line++
		raw := strings.TrimSpace(sc.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		verb, rest, _ := strings.Cut(raw, " ")
		rest = strings.TrimSpace(rest)
		step, err := parseStep(verb, rest)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		step.Line = line
		steps = append(steps, step)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return steps, nil
}

func parseStep(verb, rest string) (Step, error) {
	switch verb {
	case "wait":
		pattern, remainder, err := takeSlashed(rest)
		if err != nil {
			return Step{}, fmt.Errorf("wait: %w", err)
		}
		remainder = strings.TrimSpace(remainder)
		if remainder == "" {
			return Step{}, fmt.Errorf("wait: missing timeout, want \"wait /regex/ 10s\"")
		}
		d, err := time.ParseDuration(remainder)
		if err != nil {
			return Step{}, fmt.Errorf("wait: timeout %q: %w", remainder, err)
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return Step{}, fmt.Errorf("wait: regex %q: %w", pattern, err)
		}
		return Step{Kind: StepWait, Pattern: re, Timeout: d}, nil
	case "send":
		text, err := takeQuoted(rest)
		if err != nil {
			return Step{}, fmt.Errorf("send: %w", err)
		}
		return Step{Kind: StepSend, Text: text}, nil
	case "key":
		if rest == "" {
			return Step{}, fmt.Errorf("key: missing key name")
		}
		return Step{Kind: StepKey, Key: rest}, nil
	case "snapshot":
		if rest == "" {
			return Step{}, fmt.Errorf("snapshot: missing path")
		}
		return Step{Kind: StepSnapshot, Path: rest}, nil
	case "sleep":
		d, err := time.ParseDuration(rest)
		if err != nil {
			return Step{}, fmt.Errorf("sleep: duration %q: %w", rest, err)
		}
		return Step{Kind: StepSleep, Timeout: d}, nil
	case "kill":
		return Step{Kind: StepKill}, nil
	default:
		return Step{}, fmt.Errorf("unknown step %q", verb)
	}
}

// takeSlashed reads a leading "/pattern/" (with "\/" as an escaped slash)
// from s and returns the unescaped pattern plus whatever follows.
func takeSlashed(s string) (pattern, rest string, err error) {
	if !strings.HasPrefix(s, "/") {
		return "", "", fmt.Errorf("want /regex/, got %q", s)
	}
	var b strings.Builder
	i := 1
	for i < len(s) {
		c := s[i]
		if c == '\\' && i+1 < len(s) && s[i+1] == '/' {
			b.WriteByte('/')
			i += 2
			continue
		}
		if c == '/' {
			return b.String(), s[i+1:], nil
		}
		b.WriteByte(c)
		i++
	}
	return "", "", fmt.Errorf("unterminated /regex/ in %q", s)
}

// takeQuoted reads a whole Go-style double-quoted string literal from s
// (the whole remainder is expected to be exactly one such literal).
func takeQuoted(s string) (string, error) {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return "", fmt.Errorf("want a quoted string, got %q", s)
	}
	text, err := strconv.Unquote(s)
	if err != nil {
		return "", fmt.Errorf("%q: %w", s, err)
	}
	return text, nil
}

// KeyBytes maps a step-script key name to the bytes conptydriver writes to
// the child's input. ctrl+X (X a single letter) is generic; the rest are
// the fixed names the grammar promises, plus a fallback that sends any
// other single character literally so "key f" (an ordinary letter, no
// different from `send "f"`) needs no special case of its own.
func KeyBytes(name string) ([]byte, error) {
	switch strings.ToLower(name) {
	case "enter", "return":
		return []byte("\r"), nil
	case "esc", "escape":
		return []byte("\x1b"), nil
	case "tab":
		return []byte("\t"), nil
	case "up":
		return []byte("\x1b[A"), nil
	case "down":
		return []byte("\x1b[B"), nil
	case "right":
		return []byte("\x1b[C"), nil
	case "left":
		return []byte("\x1b[D"), nil
	case "space":
		return []byte(" "), nil
	case "backspace":
		return []byte{0x7f}, nil
	}
	if lower := strings.ToLower(name); strings.HasPrefix(lower, "ctrl+") {
		letter := strings.TrimPrefix(lower, "ctrl+")
		if len(letter) != 1 || letter[0] < 'a' || letter[0] > 'z' {
			return nil, fmt.Errorf("ctrl+%s: want a single letter a-z", letter)
		}
		return []byte{letter[0] - 'a' + 1}, nil
	}
	if len([]rune(name)) == 1 {
		return []byte(name), nil
	}
	return nil, fmt.Errorf("unknown key %q", name)
}
