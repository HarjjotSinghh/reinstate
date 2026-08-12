package secretscan

import (
	"math"
	"regexp"
	"unicode"
)

// Category classifies a detected secret shape.
type Category string

// Known secret categories. Values are stable API identifiers.
const (
	CategoryAWSKey       Category = "aws_key"
	CategoryGCPKey       Category = "gcp_key"
	CategoryGitHubToken  Category = "github_token"
	CategoryOpenAIKey    Category = "openai_key"
	CategoryAnthropicKey Category = "anthropic_key"
	CategoryPrivateKey   Category = "private_key"
	CategoryJWT          Category = "jwt"
	CategoryBearer       Category = "bearer"
	CategoryURLCred      Category = "url_credential"
	CategoryHighEntropy  Category = "high_entropy"
)

// Fixed entropy heuristic parameters. Changing any of these changes Scan output.
const (
	highEntropyMinLen = 40
	hexEntropyMinLen  = 64
	highEntropyCutoff = 4.5 // bits/char for mixed base64-ish tokens
	hexEntropyCutoff  = 3.5 // bits/char for hex-only tokens
	base64Charset     = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
	hexCharset        = "0123456789abcdefABCDEF"
)

// patternSpec is a structured detector. Lower priority wins on equal spans.
type patternSpec struct {
	category Category
	re       *regexp.Regexp
	priority int
}

// Structured detectors ordered for documentation; priority values drive ties.
var structuredPatterns = []patternSpec{
	{
		category: CategoryPrivateKey,
		priority: 1,
		re:       regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |DSA |ENCRYPTED )?PRIVATE KEY-----[\s\S]*?-----END (?:RSA |EC |OPENSSH |DSA |ENCRYPTED )?PRIVATE KEY-----`),
	},
	{
		category: CategoryAWSKey,
		priority: 2,
		re:       regexp.MustCompile(`\b(?:AKIA|ASIA)[A-Z0-9]{16}\b`),
	},
	{
		category: CategoryGCPKey,
		priority: 3,
		re:       regexp.MustCompile(`\bAIza[0-9A-Za-z_\-]{35}\b`),
	},
	{
		category: CategoryGitHubToken,
		priority: 4,
		re:       regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9_]{36,}|github_pat_[A-Za-z0-9_]{22,})\b`),
	},
	{
		category: CategoryAnthropicKey,
		priority: 5,
		re:       regexp.MustCompile(`\bsk-ant-[A-Za-z0-9_\-]{20,}\b`),
	},
	{
		category: CategoryOpenAIKey,
		priority: 6,
		// Broader than Anthropic; same-span ties prefer Anthropic (priority 5).
		re: regexp.MustCompile(`\bsk-[A-Za-z0-9_\-]{20,}\b`),
	},
	{
		category: CategoryJWT,
		priority: 7,
		re:       regexp.MustCompile(`\beyJ[A-Za-z0-9_\-]{8,}\.eyJ[A-Za-z0-9_\-]{8,}\.[A-Za-z0-9_\-]{8,}\b`),
	},
	{
		category: CategoryBearer,
		priority: 8,
		re:       regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9\-._~+/]{20,}=*`),
	},
	{
		category: CategoryURLCred,
		priority: 9,
		re:       regexp.MustCompile(`https?://[^/\s:@]+:[^/\s:@]+@[^\s/]+`),
	},
}

var (
	base64Set = charsetSet(base64Charset)
	hexSet    = charsetSet(hexCharset)
)

func charsetSet(s string) map[rune]struct{} {
	m := make(map[rune]struct{}, len(s))
	for _, r := range s {
		m[r] = struct{}{}
	}
	return m
}

// shannonEntropy returns bits-per-character Shannon entropy of s.
func shannonEntropy(s string) float64 {
	if s == "" {
		return 0
	}
	var freq [256]int
	n := 0
	for i := 0; i < len(s); i++ {
		freq[s[i]]++
		n++
	}
	var h float64
	fn := float64(n)
	for _, c := range freq {
		if c == 0 {
			continue
		}
		p := float64(c) / fn
		h -= p * math.Log2(p)
	}
	return h
}

// isHexToken reports whether every byte is in the fixed hex charset.
func isHexToken(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if _, ok := hexSet[r]; !ok {
			return false
		}
	}
	return true
}

// isBase64Token reports whether every byte is in the fixed base64 charset.
func isBase64Token(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if _, ok := base64Set[r]; !ok {
			return false
		}
	}
	return true
}

// findHighEntropyCandidates returns high-entropy token spans using the fixed
// charset, length, and Shannon-entropy cutoffs.
func findHighEntropyCandidates(text string) []candidate {
	var out []candidate
	runStart := -1
	for i := 0; i <= len(text); i++ {
		inSet := false
		if i < len(text) {
			r := rune(text[i])
			// ASCII-only charset membership; multi-byte runes break the run.
			if r <= unicode.MaxASCII {
				_, inSet = base64Set[r]
			}
		}
		if inSet {
			if runStart < 0 {
				runStart = i
			}
			continue
		}
		if runStart >= 0 {
			if c, ok := classifyEntropyRun(text, runStart, i); ok {
				out = append(out, c)
			}
			runStart = -1
		}
	}
	return out
}

func classifyEntropyRun(text string, start, end int) (candidate, bool) {
	tok := text[start:end]
	n := end - start
	if isHexToken(tok) {
		if n < hexEntropyMinLen {
			return candidate{}, false
		}
		if shannonEntropy(tok) < hexEntropyCutoff {
			return candidate{}, false
		}
		return candidate{category: CategoryHighEntropy, start: start, end: end, priority: 100}, true
	}
	if !isBase64Token(tok) || n < highEntropyMinLen {
		return candidate{}, false
	}
	if shannonEntropy(tok) < highEntropyCutoff {
		return candidate{}, false
	}
	// Require mixed letter+digit so uniform padding or prose-like runs drop out.
	hasLetter, hasDigit := false, false
	for i := 0; i < len(tok); i++ {
		switch {
		case tok[i] >= 'A' && tok[i] <= 'Z', tok[i] >= 'a' && tok[i] <= 'z':
			hasLetter = true
		case tok[i] >= '0' && tok[i] <= '9':
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return candidate{}, false
	}
	return candidate{category: CategoryHighEntropy, start: start, end: end, priority: 100}, true
}
