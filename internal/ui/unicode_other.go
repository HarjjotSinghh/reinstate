//go:build !windows

package ui

// windowsUnicodeDefault is the non-Windows fallback. Every supported Unix
// terminal renders box-drawing and status glyphs, so the absence of locale
// variables is not evidence against Unicode there.
func windowsUnicodeDefault(func(string) string) bool { return true }

// unsetTermIsDumb is true on Unix, where an unset TERM means there is no
// terminfo entry and no capability can be assumed.
const unsetTermIsDumb = true
