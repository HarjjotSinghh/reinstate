//go:build windows

package ui

import "strings"

// windowsUnicodeDefault decides whether to trust box-drawing glyphs on Windows
// when no locale variable is set.
//
// Legacy conhost under a non-UTF-8 code page renders multi-byte glyphs as
// mojibake, and the acceptance bench runs there, so the default is no. A
// terminal that advertises a modern identity is trusted.
func windowsUnicodeDefault(getenv func(string) string) bool {
	if strings.TrimSpace(getenv("WT_SESSION")) != "" {
		return true
	}
	// ConEmu and the VS Code integrated terminal both set TERM_PROGRAM.
	if strings.TrimSpace(getenv("TERM_PROGRAM")) != "" {
		return true
	}
	return false
}

// unsetTermIsDumb is false on Windows: TERM is not part of that platform's
// environment, so its absence carries no information about the console.
const unsetTermIsDumb = false
