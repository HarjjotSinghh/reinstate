package main

import (
	"strings"
	"testing"
)

func firstLine(render string) string {
	line, _, _ := strings.Cut(render, "\n")
	return line
}

func TestScreenPlainText(t *testing.T) {
	s := NewScreen(10, 3)
	s.Feed([]byte("hi"))
	if got := firstLine(s.Render()); got != "hi" {
		t.Fatalf("Render() first line = %q, want %q", got, "hi")
	}
}

func TestScreenCursorForwardLeavesBlanks(t *testing.T) {
	// This is the trap the type doc names: conhost may skip over unchanged
	// cells with a cursor-forward move (CUF) instead of writing literal
	// spaces. A regex strip cannot tell that apart from nothing having
	// happened; a real cursor model renders it as blank, which is correct.
	s := NewScreen(10, 1)
	s.Feed([]byte("ab"))
	s.Feed([]byte("\x1b[3C")) // CUF 3: skip to column 6 (0-indexed 5)
	s.Feed([]byte("Z"))
	got := s.Render()
	want := "ab   Z"
	if got != want {
		t.Fatalf("Render() = %q, want %q", got, want)
	}
}

func TestScreenCUP(t *testing.T) {
	s := NewScreen(10, 3)
	s.Feed([]byte("\x1b[2;3Hx")) // row 2, col 3 (1-indexed) -> 0-indexed (1,2)
	row, col := s.Cursor()
	if row != 1 || col != 3 {
		t.Fatalf("cursor after CUP+1 write = (%d,%d), want (1,3)", row, col)
	}
	lines := strings.Split(s.Render(), "\n")
	if len(lines) < 2 || lines[1] != "  x" {
		t.Fatalf("Render() row 1 = %q, want \"  x\"", lines[1])
	}
}

func TestScreenCUU_CUD_CUB(t *testing.T) {
	s := NewScreen(10, 5)
	s.Feed([]byte("\x1b[3;3H")) // start at row 3, col 3 (1-indexed)
	s.Feed([]byte("\x1b[1A"))   // up 1
	s.Feed([]byte("\x1b[2C"))   // forward 2
	row, col := s.Cursor()
	if row != 1 || col != 4 {
		t.Fatalf("cursor = (%d,%d), want (1,4)", row, col)
	}
	s.Feed([]byte("\x1b[3D")) // back 3
	_, col = s.Cursor()
	if col != 1 {
		t.Fatalf("cursor col after CUB 3 = %d, want 1", col)
	}
	s.Feed([]byte("\x1b[2B")) // down 2
	row, _ = s.Cursor()
	if row != 3 {
		t.Fatalf("cursor row after CUD 2 = %d, want 3", row)
	}
}

func TestScreenCRLF(t *testing.T) {
	s := NewScreen(10, 3)
	s.Feed([]byte("one\r\ntwo"))
	lines := strings.Split(s.Render(), "\n")
	if lines[0] != "one" || lines[1] != "two" {
		t.Fatalf("Render() = %q", s.Render())
	}
}

func TestScreenEraseLine(t *testing.T) {
	s := NewScreen(10, 1)
	s.Feed([]byte("abcdefgh"))
	s.Feed([]byte("\x1b[1;4H")) // column 4 (0-indexed 3)
	s.Feed([]byte("\x1b[K"))    // EL 0: cursor to end of line
	if got := s.Render(); got != "abc" {
		t.Fatalf("Render() = %q, want %q", got, "abc")
	}
}

func TestScreenEraseDisplay(t *testing.T) {
	s := NewScreen(5, 3)
	s.Feed([]byte("aaaaa\r\nbbbbb\r\nccccc"))
	s.Feed([]byte("\x1b[2;1H")) // row 2 col 1
	s.Feed([]byte("\x1b[2J"))   // ED 2: whole screen
	if got := s.Render(); got != "\n\n" {
		t.Fatalf("Render() after ED2 = %q, want blank grid", got)
	}
}

func TestScreenSGRIgnoredButConsumed(t *testing.T) {
	s := NewScreen(10, 1)
	s.Feed([]byte("\x1b[1;31mred\x1b[0m plain"))
	if got := s.Render(); got != "red plain" {
		t.Fatalf("Render() = %q, want %q (SGR bytes must not leak into cells)", got, "red plain")
	}
}

func TestScreenOSCBackgroundColorQuery(t *testing.T) {
	s := NewScreen(10, 1)
	s.SetBackgroundColorReply("rgb:1234/5678/9abc")
	reply := s.Feed([]byte("\x1b]11;?\x07"))
	want := "\x1b]11;rgb:1234/5678/9abc\x07"
	if string(reply) != want {
		t.Fatalf("OSC 11 reply = %q, want %q", reply, want)
	}
}

func TestScreenOSCBackgroundColorQuery_STTerminated(t *testing.T) {
	s := NewScreen(10, 1)
	reply := s.Feed([]byte("\x1b]11;?\x1b\\"))
	if !strings.HasPrefix(string(reply), "\x1b]11;") {
		t.Fatalf("OSC 11 (ST-terminated) reply = %q, want an OSC 11 response", reply)
	}
}

func TestScreenOtherOSCIgnored(t *testing.T) {
	s := NewScreen(10, 1)
	reply := s.Feed([]byte("\x1b]0;window title\x07visible"))
	if len(reply) != 0 {
		t.Fatalf("OSC 0 (window title) produced a reply %q, want none", reply)
	}
	if got := s.Render(); got != "visible" {
		t.Fatalf("Render() = %q, want %q", got, "visible")
	}
}

func TestScreenCursorPositionReport(t *testing.T) {
	s := NewScreen(10, 5)
	s.Feed([]byte("\x1b[3;5H"))
	reply := s.Feed([]byte("\x1b[6n"))
	want := "\x1b[3;5R"
	if string(reply) != want {
		t.Fatalf("CSI 6n reply = %q, want %q", reply, want)
	}
}

func TestScreenDCSSkipped(t *testing.T) {
	s := NewScreen(10, 1)
	reply := s.Feed([]byte("\x1bPsome dcs body\x1b\\after"))
	if len(reply) != 0 {
		t.Fatalf("DCS produced a reply %q, want none", reply)
	}
	if got := s.Render(); got != "after" {
		t.Fatalf("Render() = %q, want %q", got, "after")
	}
}

func TestScreenSplitAcrossFeeds(t *testing.T) {
	// A real ConPTY read loop delivers arbitrary chunk boundaries, including
	// mid-escape-sequence. Feed must resume state across calls.
	s := NewScreen(10, 5)
	s.Feed([]byte("\x1b["))
	s.Feed([]byte("2"))
	s.Feed([]byte(";3Hx"))
	row, col := s.Cursor()
	if row != 1 || col != 3 {
		t.Fatalf("cursor after split CUP+1 write = (%d,%d), want (1,3)", row, col)
	}
}

func TestScreenUTF8(t *testing.T) {
	s := NewScreen(10, 1)
	s.Feed([]byte("→ done"))
	if got := firstLine(s.Render()); got != "→ done" {
		t.Fatalf("Render() = %q, want %q", got, "→ done")
	}
}

func TestScreenLineWrap(t *testing.T) {
	s := NewScreen(3, 2)
	s.Feed([]byte("abcd"))
	lines := strings.Split(s.Render(), "\n")
	if lines[0] != "abc" || lines[1] != "d" {
		t.Fatalf("Render() = %q", s.Render())
	}
}
