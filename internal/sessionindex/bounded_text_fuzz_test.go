package sessionindex

import (
	"bytes"
	"testing"
)

// FuzzBoundedPromptAccumulatorMatchesBuildSearchText protects the issue #96
// optimization against semantic drift. Vendor messages arrive as independent
// JSON events, so arbitrary chunk boundaries (including invalid UTF-8) must
// produce the exact private search text that repeated BuildSearchText calls
// produced before the linear accumulator was introduced.
func FuzzBoundedPromptAccumulatorMatchesBuildSearchText(f *testing.F) {
	f.Add([]byte("first\x00second\x00third"))
	f.Add([]byte("prefix\x00\xf0\x9f\x99\x82\x00suffix"))
	f.Add([]byte("controls\x1b[31m\x00\xff\xfe\x00tail"))
	f.Add(bytes.Repeat([]byte("bounded\x00"), 1_024))

	f.Fuzz(func(t *testing.T, encoded []byte) {
		// Keep the legacy oracle bounded: its repeated rebuild is deliberately
		// quadratic and exists here only to prove behavioral equivalence.
		if len(encoded) > 64<<10 {
			t.Skip()
		}

		var optimized boundedText
		legacy := ""
		for _, part := range bytes.Split(encoded, []byte{0}) {
			value := string(part)
			optimized.Add(value)
			legacy = BuildSearchText(legacy, value)
		}

		got := BuildSearchText("controlled-prefix", optimized.String(), "controlled-suffix")
		want := BuildSearchText("controlled-prefix", legacy, "controlled-suffix")
		if got != want {
			t.Fatalf("optimized accumulator differs from legacy oracle: got %d bytes, want %d", len(got), len(want))
		}
	})
}
