// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package tui

import (
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

// ClipboardFunc copies text to the system clipboard.
type ClipboardFunc func(string) error

// OSC52Clipboard returns a ClipboardFunc that copies through the terminal
// itself using the OSC 52 escape sequence.
//
// This is deliberately not a native clipboard binding. A session reference is
// most often needed on a machine reached over SSH, where a host-side clipboard
// call would put the text on the wrong computer. OSC 52 asks the terminal
// emulator that the human is actually looking at to do the copy, so it works
// locally, over SSH, and inside tmux and screen alike.
func OSC52Clipboard(out io.Writer) ClipboardFunc {
	return func(text string) error {
		if out == nil {
			return nil
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(text))
		_, err := io.WriteString(out, wrapForMultiplexer("\x1b]52;c;"+encoded+"\a"))
		return err
	}
}

// wrapForMultiplexer lets an OSC sequence reach the outer terminal when the
// program is running inside tmux or GNU screen. Both intercept escape
// sequences, and both provide a documented passthrough form.
//
// The wrapping is applied unconditionally rather than sniffing $TMUX, because a
// terminal that does not need it simply ignores an unknown DCS string, whereas
// guessing wrong means a silent no-op the user cannot diagnose.
func wrapForMultiplexer(sequence string) string {
	// tmux passthrough: DCS tmux; <sequence with ESC doubled> ST
	tmuxWrapped := "\x1bPtmux;" + strings.ReplaceAll(sequence, "\x1b", "\x1b\x1b") + "\x1b\\"
	return sequence + tmuxWrapped
}

// ClipboardMsg reports the outcome of a copy so a surface can show a status.
type ClipboardMsg struct {
	Text string
	Err  error
}

// String renders a human status line for a copy result.
func (c ClipboardMsg) String() string {
	if c.Err != nil {
		return fmt.Sprintf("could not copy: %v", c.Err)
	}
	return "copied " + c.Text
}
