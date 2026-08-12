package handoff

// EstimateTokens returns the declared size heuristic ceil(utf8_bytes / 4).
// It is not a tokenizer and must not grow a tokenizer dependency.
func EstimateTokens(b []byte) int {
	n := len(b)
	if n == 0 {
		return 0
	}
	return (n + 3) / 4
}
