package main

import (
	"strings"
	"testing"
	"time"
)

func TestParseScriptAllVerbs(t *testing.T) {
	src := `
# a comment, and a blank line above

wait /ready/ 10s
send "hello\n"
key enter
key ctrl+k
key f
snapshot out/frame.txt
sleep 500ms
kill
`
	steps, err := ParseScript(strings.NewReader(src))
	if err != nil {
		t.Fatalf("ParseScript: %v", err)
	}
	if len(steps) != 8 {
		t.Fatalf("got %d steps, want 8: %+v", len(steps), steps)
	}
	if steps[0].Kind != StepWait || steps[0].Pattern.String() != "ready" || steps[0].Timeout != 10*time.Second {
		t.Fatalf("wait step = %+v", steps[0])
	}
	if steps[1].Kind != StepSend || steps[1].Text != "hello\n" {
		t.Fatalf("send step = %+v", steps[1])
	}
	if steps[2].Kind != StepKey || steps[2].Key != "enter" {
		t.Fatalf("key step = %+v", steps[2])
	}
	if steps[3].Key != "ctrl+k" {
		t.Fatalf("key step 3 = %+v", steps[3])
	}
	if steps[4].Key != "f" {
		t.Fatalf("key step 4 = %+v", steps[4])
	}
	if steps[5].Kind != StepSnapshot || steps[5].Path != "out/frame.txt" {
		t.Fatalf("snapshot step = %+v", steps[5])
	}
	if steps[6].Kind != StepSleep || steps[6].Timeout != 500*time.Millisecond {
		t.Fatalf("sleep step = %+v", steps[6])
	}
	if steps[7].Kind != StepKill {
		t.Fatalf("kill step = %+v", steps[7])
	}
}

func TestParseScriptLineNumbers(t *testing.T) {
	src := "wait /a/ 1s\nsend \"x\"\nkey enter\n"
	steps, err := ParseScript(strings.NewReader(src))
	if err != nil {
		t.Fatalf("ParseScript: %v", err)
	}
	for i, want := range []int{1, 2, 3} {
		if steps[i].Line != want {
			t.Fatalf("step %d line = %d, want %d", i, steps[i].Line, want)
		}
	}
}

func TestParseScriptWaitRegexWithEscapedSlash(t *testing.T) {
	steps, err := ParseScript(strings.NewReader(`wait /a\/b/ 1s`))
	if err != nil {
		t.Fatalf("ParseScript: %v", err)
	}
	if steps[0].Pattern.String() != "a/b" {
		t.Fatalf("pattern = %q, want %q", steps[0].Pattern.String(), "a/b")
	}
}

func TestParseScriptErrors(t *testing.T) {
	cases := []string{
		"wait ready 10s",     // missing slashes
		"wait /ready/",       // missing timeout
		"wait /ready/ soon",  // bad duration
		"wait /(/ 1s",        // bad regex
		"send hello",         // not quoted
		`send "unterminated`, // bad quoting
		"key",                // missing key name
		"snapshot",           // missing path
		"sleep",              // missing duration
		"sleep tomorrow",     // bad duration
		"frobnicate",         // unknown verb
	}
	for _, src := range cases {
		if _, err := ParseScript(strings.NewReader(src)); err == nil {
			t.Errorf("ParseScript(%q): want an error, got none", src)
		}
	}
}

func TestParseScriptBlankAndComments(t *testing.T) {
	steps, err := ParseScript(strings.NewReader("\n  \n# comment\nkill\n"))
	if err != nil {
		t.Fatalf("ParseScript: %v", err)
	}
	if len(steps) != 1 || steps[0].Kind != StepKill {
		t.Fatalf("steps = %+v, want just kill", steps)
	}
}

func TestKeyBytes(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"enter", "\r"},
		{"Enter", "\r"},
		{"esc", "\x1b"},
		{"tab", "\t"},
		{"up", "\x1b[A"},
		{"down", "\x1b[B"},
		{"space", " "},
		{"ctrl+k", "\x0b"},
		{"ctrl+a", "\x01"},
		{"f", "f"},
		{"q", "q"},
	}
	for _, c := range cases {
		got, err := KeyBytes(c.name)
		if err != nil {
			t.Errorf("KeyBytes(%q): %v", c.name, err)
			continue
		}
		if string(got) != c.want {
			t.Errorf("KeyBytes(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestKeyBytesErrors(t *testing.T) {
	for _, name := range []string{"ctrl+", "ctrl+1", "ctrl+ab", "pageup"} {
		if _, err := KeyBytes(name); err == nil {
			t.Errorf("KeyBytes(%q): want an error, got none", name)
		}
	}
}
