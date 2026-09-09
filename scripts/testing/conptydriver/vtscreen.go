package main

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// Screen is a minimal VT/xterm screen model: a grid of cells and a cursor,
// fed raw bytes from a real terminal (conhost via ConPTY, in this program's
// case) and updated the way a terminal emulator would update its own
// framebuffer.
//
// It exists because a regex strip of "ANSI-looking" byte runs cannot render
// a frame correctly. conhost's own repaint strategy frequently rewrites a
// run of unchanged cells as a cursor-forward move (CUF, "ESC [ n C")
// instead of writing literal space bytes, so a strip-and-join approach
// either drops those columns or double-counts them depending on which way
// it guesses. A real cursor model does not need to guess: CUF moves the
// cursor without touching cell content, and every cell starts (and is
// cleared to) a space, so the columns a CUF skips over already read as
// blank -- which is what they are.
//
// Supported: printable text (UTF-8), CR, LF, backspace, CUP/CUU/CUD/CUF/CUB
// (H/f, A, B, C, D), CHA (G) and VPA (d), EL (K) and ED (J), and two device
// queries a Bubble Tea program sends at startup: OSC 11 (current background
// colour) and CSI 6n (cursor position report) -- both answered so the
// program under test does not stall waiting for a reply no real terminal
// would ever send it here. SGR (m) is parsed (so its parameter bytes do not
// leak into the cell stream) and otherwise ignored: colour and style play no
// part in what a snapshot records. OSC and DCS/SOS/PM/APC strings are
// consumed up to their terminator and otherwise ignored.
type Screen struct {
	cols, rows       int
	cells            [][]rune
	cursorRow        int // 0-indexed
	cursorCol        int // 0-indexed
	state            parseState
	csi              []byte // parameter bytes collected since "ESC ["
	strBody          []byte // OSC/DCS/SOS/PM/APC body collected since its introducer
	strKind          byte   // ']' (OSC), 'P' (DCS), 'X'/'^'/'_' (SOS/PM/APC)
	sawEscInStrBody  bool   // last byte of strBody was ESC (watching for ST = ESC \)
	pendingBackColor string // response body sent for an OSC 11 "?" query
}

type parseState int

const (
	stateNormal parseState = iota
	stateEsc
	stateCSI
	stateStr     // OSC/DCS/SOS/PM/APC body, ends at BEL or ST (ESC \)
	stateCharset // ESC ( / ) / * / + X -- one more byte then back to normal
)

// NewScreen builds a blank cols x rows grid with the cursor at the origin.
func NewScreen(cols, rows int) *Screen {
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 25
	}
	s := &Screen{cols: cols, rows: rows, pendingBackColor: "rgb:0000/0000/0000"}
	s.cells = make([][]rune, rows)
	for i := range s.cells {
		s.cells[i] = blankRow(cols)
	}
	return s
}

func blankRow(cols int) []rune {
	row := make([]rune, cols)
	for i := range row {
		row[i] = ' '
	}
	return row
}

// Feed processes raw output bytes from the child and returns any bytes that
// must be written back to the child's input to answer a device query it is
// waiting on (OSC 11, CSI 6n). The caller is responsible for actually
// writing the return value to the pseudo console's input side; Feed never
// writes anywhere itself; it only observes and renders.
func (s *Screen) Feed(data []byte) []byte {
	var reply []byte
	i := 0
	for i < len(data) {
		switch s.state {
		case stateNormal:
			n := s.feedNormal(data[i:], &reply)
			i += n
		case stateEsc:
			i += s.feedEsc(data[i:])
		case stateCSI:
			n, resp := s.feedCSI(data[i])
			i += n
			if resp != nil {
				reply = append(reply, resp...)
			}
		case stateStr:
			i += s.feedStr(data[i:], &reply)
		case stateCharset:
			i++ // consume the single designator byte
			s.state = stateNormal
		default:
			s.state = stateNormal
			i++
		}
	}
	return reply
}

// feedNormal consumes as many plain bytes as it can starting at data[0] and
// returns how many bytes it consumed (at least 1, unless data is empty).
func (s *Screen) feedNormal(data []byte, reply *[]byte) int {
	b := data[0]
	switch {
	case b == 0x1b: // ESC
		s.state = stateEsc
		return 1
	case b == '\r':
		s.cursorCol = 0
		return 1
	case b == '\n':
		s.lineFeed()
		return 1
	case b == '\b':
		if s.cursorCol > 0 {
			s.cursorCol--
		}
		return 1
	case b == '\t':
		next := ((s.cursorCol / 8) + 1) * 8
		if next >= s.cols {
			next = s.cols - 1
		}
		s.cursorCol = next
		return 1
	case b < 0x20:
		// Other C0 controls (BEL outside a string, SO/SI, ...): no visible
		// effect on the grid.
		return 1
	default:
		r, size := utf8.DecodeRune(data)
		s.put(r)
		return size
	}
}

func (s *Screen) put(r rune) {
	if s.cursorRow < 0 || s.cursorRow >= s.rows {
		return
	}
	if s.cursorCol >= s.cols {
		s.lineFeed()
		s.cursorCol = 0
	}
	if s.cursorRow < s.rows && s.cursorCol < s.cols {
		s.cells[s.cursorRow][s.cursorCol] = r
	}
	s.cursorCol++
}

// lineFeed moves the cursor down one row, scrolling the grid up when it is
// already on the last row -- the ordinary terminal behaviour a long-running
// program's output needs, even though a single-frame snapshot rarely
// exercises it.
func (s *Screen) lineFeed() {
	if s.cursorRow < s.rows-1 {
		s.cursorRow++
		return
	}
	copy(s.cells, s.cells[1:])
	s.cells[s.rows-1] = blankRow(s.cols)
}

func (s *Screen) feedEsc(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	c := data[0]
	switch c {
	case '[':
		s.state = stateCSI
		s.csi = s.csi[:0]
		return 1
	case ']':
		s.state = stateStr
		s.strKind = ']'
		s.strBody = s.strBody[:0]
		s.sawEscInStrBody = false
		return 1
	case 'P', 'X', '^', '_':
		s.state = stateStr
		s.strKind = c
		s.strBody = s.strBody[:0]
		s.sawEscInStrBody = false
		return 1
	case '(', ')', '*', '+':
		s.state = stateCharset
		return 1
	default:
		// A single-character escape (RIS "c", IND, NEL, DECSC/DECRC "7"/"8",
		// keypad mode "="/">" , ...): no grid effect we model.
		s.state = stateNormal
		return 1
	}
}

// feedCSI consumes one byte of a "ESC [ ... final" sequence and, once the
// final byte (0x40-0x7E) is seen, applies it. It returns the number of
// input bytes consumed (always 1) and any reply bytes the applied command
// produced.
func (s *Screen) feedCSI(b byte) (int, []byte) {
	if b >= 0x40 && b <= 0x7e {
		s.state = stateNormal
		return 1, s.applyCSI(b, string(s.csi))
	}
	s.csi = append(s.csi, b)
	return 1, nil
}

func (s *Screen) applyCSI(final byte, params string) []byte {
	// The private-mode prefix ('?', '<', '=', '>') on params like
	// "?25" (cursor visibility) or "?1004" (focus reporting) does not
	// change the two-integer parsing below; nothing here reaches a branch
	// that reads it as a number.
	ps := parseParams(params)
	get := func(i, def int) int {
		if i < len(ps) && ps[i] > 0 {
			return ps[i]
		}
		return def
	}
	switch final {
	case 'H', 'f': // CUP
		s.cursorRow = clamp(get(0, 1)-1, 0, s.rows-1)
		s.cursorCol = clamp(get(1, 1)-1, 0, s.cols-1)
	case 'A': // CUU
		s.cursorRow = clamp(s.cursorRow-get(0, 1), 0, s.rows-1)
	case 'B': // CUD
		s.cursorRow = clamp(s.cursorRow+get(0, 1), 0, s.rows-1)
	case 'C': // CUF
		s.cursorCol = clamp(s.cursorCol+get(0, 1), 0, s.cols-1)
	case 'D': // CUB
		s.cursorCol = clamp(s.cursorCol-get(0, 1), 0, s.cols-1)
	case 'G': // CHA: cursor horizontal absolute
		s.cursorCol = clamp(get(0, 1)-1, 0, s.cols-1)
	case 'd': // VPA: line position absolute
		s.cursorRow = clamp(get(0, 1)-1, 0, s.rows-1)
	case 'J': // ED: erase in display
		s.eraseDisplay(get(0, 0))
	case 'K': // EL: erase in line
		s.eraseLine(get(0, 0))
	case 'X': // ECH: erase character (blank N cells at the cursor, no cursor move)
		s.eraseChars(get(0, 1))
	case 'n': // DSR: device status report
		if get(0, 0) == 6 {
			return []byte("\x1b[" + strconv.Itoa(s.cursorRow+1) + ";" + strconv.Itoa(s.cursorCol+1) + "R")
		}
	case 'm':
		// SGR: colour/style. Deliberately ignored; see the type doc.
	default:
		// Every other final byte (scroll region, insert/delete line, mode
		// sets, ...): parsed and discarded, not left in the byte stream.
	}
	return nil
}

func (s *Screen) eraseDisplay(mode int) {
	switch mode {
	case 0:
		s.eraseLine(0)
		for r := s.cursorRow + 1; r < s.rows; r++ {
			s.cells[r] = blankRow(s.cols)
		}
	case 1:
		s.eraseLine(1)
		for r := 0; r < s.cursorRow; r++ {
			s.cells[r] = blankRow(s.cols)
		}
	default: // 2 and 3: entire screen
		for r := range s.cells {
			s.cells[r] = blankRow(s.cols)
		}
	}
}

// eraseChars implements ECH (CSI Ps X): blank n cells starting at the
// cursor, without moving the cursor. Local test-harness fix, uncommitted:
// this CSI final byte previously had no case in applyCSI's switch (fell
// into "parsed and discarded"), so a real terminal's blanked cells stayed
// stale in this renderer's own grid — see acceptance report evidence for
// CLI row 9 (part C, 2026-09-09-windows-v060rc8-part-c.md) for the capture
// artifact this produced and traced back to here.
func (s *Screen) eraseChars(n int) {
	if s.cursorRow < 0 || s.cursorRow >= s.rows || n <= 0 {
		return
	}
	row := s.cells[s.cursorRow]
	for c := s.cursorCol; c < s.cols && c < s.cursorCol+n; c++ {
		row[c] = ' '
	}
}

func (s *Screen) eraseLine(mode int) {
	if s.cursorRow < 0 || s.cursorRow >= s.rows {
		return
	}
	row := s.cells[s.cursorRow]
	switch mode {
	case 0:
		for c := s.cursorCol; c < s.cols; c++ {
			row[c] = ' '
		}
	case 1:
		for c := 0; c <= s.cursorCol && c < s.cols; c++ {
			row[c] = ' '
		}
	default: // 2: entire line
		for c := range row {
			row[c] = ' '
		}
	}
}

// feedStr consumes bytes of an OSC/DCS/SOS/PM/APC body until BEL or the
// two-byte ST ("ESC \"), then, for an OSC body, answers a background-colour
// query. It returns the number of input bytes consumed.
func (s *Screen) feedStr(data []byte, reply *[]byte) int {
	b := data[0]
	if s.sawEscInStrBody {
		s.sawEscInStrBody = false
		if b == '\\' {
			s.finishStr(reply)
			return 1
		}
		// Not actually a string terminator; the ESC starts a new
		// escape sequence of its own (some senders interleave). Replay it
		// through the normal escape entry point rather than swallowing it.
		s.finishStr(reply)
		s.state = stateEsc
		return 0
	}
	switch b {
	case 0x07: // BEL: also terminates an OSC body
		s.finishStr(reply)
		return 1
	case 0x1b:
		s.sawEscInStrBody = true
		return 1
	default:
		s.strBody = append(s.strBody, b)
		return 1
	}
}

func (s *Screen) finishStr(reply *[]byte) {
	s.state = stateNormal
	if s.strKind == ']' {
		body := string(s.strBody)
		// RequestBackgroundColor is "ESC ] 11 ; ? BEL"; Bubble Tea and
		// other termenv-based programs send this at startup to decide
		// whether to run a light or dark theme, and block their first
		// render on the reply.
		if body == "11;?" {
			*reply = append(*reply, []byte("\x1b]11;"+s.pendingBackColor+"\x07")...)
		}
	}
	s.strBody = s.strBody[:0]
}

// SetBackgroundColorReply overrides the colour this screen reports for an
// OSC 11 query; the default is black.
func (s *Screen) SetBackgroundColorReply(xrgb string) { s.pendingBackColor = xrgb }

// Render returns the current frame as rows joined by "\n", each row
// right-trimmed of trailing blanks (the padding a fixed-width grid carries
// past the last thing actually drawn on that row, not content).
func (s *Screen) Render() string {
	lines := make([]string, s.rows)
	for i, row := range s.cells {
		lines[i] = strings.TrimRight(string(row), " ")
	}
	return strings.Join(lines, "\n")
}

// Cursor returns the current cursor position, 0-indexed (row, col).
func (s *Screen) Cursor() (row, col int) { return s.cursorRow, s.cursorCol }

func parseParams(s string) []int {
	if s == "" {
		return nil
	}
	// Strip a leading private-mode marker; none of the sequences this
	// screen acts on take one, but leaving it in would make the first
	// field fail to parse as a number and read as "default" anyway, which
	// is harmless -- this just avoids the wasted Atoi error.
	if len(s) > 0 && (s[0] == '?' || s[0] == '<' || s[0] == '=' || s[0] == '>') {
		s = s[1:]
	}
	parts := strings.Split(s, ";")
	out := make([]int, len(parts))
	for i, p := range parts {
		if p == "" {
			out[i] = 0
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			out[i] = 0
			continue
		}
		out[i] = n
	}
	return out
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
