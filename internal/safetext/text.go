// Package safetext removes terminal/control injection from bounded metadata.
package safetext

import (
	"strings"
	"unicode"
)

// Text strips terminal sequences, controls, and Unicode format controls,
// collapses whitespace, and bounds Unicode code points. A non-positive limit
// keeps all sanitized code points.
func Text(value string, maxRunes int) string {
	value = strings.ToValidUTF8(value, "")
	value = stripTerminalSequences(value)
	var output strings.Builder
	output.Grow(min(len(value), 4096))
	wroteSpace := false
	runes := 0
	for _, current := range value {
		if unicode.IsSpace(current) {
			if output.Len() > 0 {
				wroteSpace = true
			}
			continue
		}
		if unicode.IsControl(current) || unicode.In(current, unicode.Cf) {
			continue
		}
		if wroteSpace {
			if maxRunes > 0 && runes >= maxRunes {
				break
			}
			output.WriteByte(' ')
			runes++
			wroteSpace = false
		}
		if maxRunes > 0 && runes >= maxRunes {
			break
		}
		output.WriteRune(current)
		runes++
	}
	return strings.TrimSpace(output.String())
}

func stripTerminalSequences(value string) string {
	const (
		stateText = iota
		stateEscape
		stateCSI
		stateString
		stateStringEscape
	)
	state := stateText
	var output strings.Builder
	output.Grow(len(value))
	for _, current := range value {
		switch state {
		case stateText:
			switch current {
			case '\x1b':
				state = stateEscape
			case '\u009b':
				state = stateCSI
			case '\u0090', '\u009d', '\u009e', '\u009f':
				state = stateString
			default:
				output.WriteRune(current)
			}
		case stateEscape:
			switch current {
			case '[':
				state = stateCSI
			case ']', 'P', 'X', '^', '_':
				state = stateString
			default:
				state = stateText
			}
		case stateCSI:
			if current >= 0x40 && current <= 0x7e {
				state = stateText
			}
		case stateString:
			switch current {
			case '\a':
				state = stateText
			case '\x1b':
				state = stateStringEscape
			}
		case stateStringEscape:
			if current == '\\' {
				state = stateText
			} else if current != '\x1b' {
				state = stateString
			}
		}
	}
	return output.String()
}
