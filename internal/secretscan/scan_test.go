package secretscan

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"
)

// Synthetic samples are built at runtime so static secret scanners never see
// a complete credential literal in source.

func synthAWS() string {
	return "AKIA" + strings.Repeat("2", 16)
}

func synthGCPKey() string {
	// AIza + exactly 35 charset chars.
	return "AIza" + strings.Repeat("x", 35)
}

func synthGitHub() string {
	return "ghp_" + strings.Repeat("A", 36)
}

func synthOpenAI() string {
	return "sk-" + strings.Repeat("b", 48)
}

func synthAnthropic() string {
	return "sk-ant-" + strings.Repeat("c", 40)
}

func synthPrivateKey() string {
	body := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("priv", 16)))
	return "-----BEGIN PRIVATE KEY-----\n" + body + "\n-----END PRIVATE KEY-----"
}

func synthJWT() string {
	h := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
	p := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"fixture"}`))
	s := base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("sig", 8)))
	return h + "." + p + "." + s
}

func synthBearer() string {
	return "Bearer " + strings.Repeat("t", 32)
}

func synthURLCred() string {
	return "https://fixture-user:fixture-pass@example.com/path"
}

func synthHighEntropy() string {
	// Deterministic mixed base64 of a fixed digest — above length and entropy cutoffs.
	sum := sha256.Sum256([]byte("reinstate-fixture-entropy"))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func TestScanCategoryPositiveAndNegative(t *testing.T) {
	cases := []struct {
		name     string
		category Category
		positive string
		negative string
	}{
		{"aws_key", CategoryAWSKey, "id=" + synthAWS(), "id=AKIASHORT"},
		{"gcp_key", CategoryGCPKey, "key=" + synthGCPKey(), "key=AIzaTooShort"},
		{"github_token", CategoryGitHubToken, "tok=" + synthGitHub(), "tok=ghp_short"},
		{"openai_key", CategoryOpenAIKey, "k=" + synthOpenAI(), "k=sk-short"},
		{"anthropic_key", CategoryAnthropicKey, "k=" + synthAnthropic(), "k=sk-ant-short"},
		{"private_key", CategoryPrivateKey, synthPrivateKey(), "-----BEGIN CERTIFICATE-----\nabc\n-----END CERTIFICATE-----"},
		{"jwt", CategoryJWT, "auth " + synthJWT(), "eyJhbGciOiJIUzI1NiJ9.partial"},
		{"bearer", CategoryBearer, synthBearer(), "Bearer short"},
		{"url_credential", CategoryURLCred, "see " + synthURLCred(), "https://example.com/no-user"},
		{"high_entropy", CategoryHighEntropy, "secret=" + synthHighEntropy(), "secret=lowentropyaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}
	for _, tc := range cases {
		t.Run(tc.name+"/positive", func(t *testing.T) {
			ms := Scan(tc.positive)
			found := false
			for _, m := range ms {
				if m.Category == tc.category {
					found = true
					if m.Start < 0 || m.End <= m.Start || m.End > len(tc.positive) {
						t.Fatalf("bad offsets: %+v", m)
					}
					if len(m.Digest) != 12 {
						t.Fatalf("digest len=%d want 12", len(m.Digest))
					}
					// Never retain the matched value on the Match.
					raw := tc.positive[m.Start:m.End]
					if strings.Contains(fmt.Sprintf("%#v", m), raw) && len(raw) > 12 {
						t.Fatalf("Match appears to retain raw value")
					}
				}
			}
			if !found {
				t.Fatalf("expected category %s in %v", tc.category, categoriesOf(ms))
			}
		})
		t.Run(tc.name+"/negative", func(t *testing.T) {
			ms := Scan(tc.negative)
			for _, m := range ms {
				if m.Category == tc.category {
					t.Fatalf("unexpected %s match in %q", tc.category, tc.negative)
				}
			}
		})
	}
}

func categoriesOf(ms []Match) []Category {
	out := make([]Category, len(ms))
	for i, m := range ms {
		out[i] = m.Category
	}
	return out
}

func TestScanOverlappingCandidatesDeterministic(t *testing.T) {
	// Anthropic key is also an sk- prefix; structured overlap must pick anthropic.
	text := "key=" + synthAnthropic()
	ms := Scan(text)
	if len(ms) != 1 {
		t.Fatalf("got %d matches want 1: %v", len(ms), ms)
	}
	if ms[0].Category != CategoryAnthropicKey {
		t.Fatalf("got %s want %s", ms[0].Category, CategoryAnthropicKey)
	}

	// AWS key embedded beside a longer private-key-like span: earliest+longest wins.
	aws := synthAWS()
	pem := synthPrivateKey()
	combined := "before " + aws + " " + pem + " after"
	ms = Scan(combined)
	if len(ms) < 2 {
		t.Fatalf("expected both aws and private_key, got %v", categoriesOf(ms))
	}
	// Source order: aws first, then private_key.
	if ms[0].Category != CategoryAWSKey {
		t.Fatalf("first=%s want aws_key", ms[0].Category)
	}
	if ms[1].Category != CategoryPrivateKey {
		t.Fatalf("second=%s want private_key", ms[1].Category)
	}
	// Same input twice → identical matches.
	ms2 := Scan(combined)
	if len(ms2) != len(ms) {
		t.Fatalf("nondeterministic length")
	}
	for i := range ms {
		if ms[i] != ms2[i] {
			t.Fatalf("match %d differs: %+v vs %+v", i, ms[i], ms2[i])
		}
	}
}

func TestRedactStableUnderSecondPass(t *testing.T) {
	text := strings.Join([]string{
		"aws=" + synthAWS(),
		"gh=" + synthGitHub(),
		"oa=" + synthOpenAI(),
	}, " ")
	once, ms := Redact(text)
	if len(ms) == 0 {
		t.Fatal("expected matches")
	}
	for _, m := range ms {
		want := "[redacted:" + string(m.Category) + ":" + m.Digest + "]"
		if !strings.Contains(once, want) {
			t.Fatalf("missing marker %s in %q", want, once)
		}
	}
	twice, ms2 := Redact(once)
	if twice != once {
		t.Fatalf("second Redact changed text\n once=%q\ntwice=%q", once, twice)
	}
	if len(ms2) != 0 {
		t.Fatalf("second Redact reported matches: %v", ms2)
	}
}

func TestSummaryNeverContainsValue(t *testing.T) {
	secret := synthAWS()
	ms := Scan("x=" + secret)
	sum := Summary(ms)
	if sum[CategoryAWSKey] != 1 {
		t.Fatalf("summary=%v", sum)
	}
	enc := fmt.Sprintf("%#v", sum)
	if strings.Contains(enc, secret) {
		t.Fatalf("summary leaked value: %s", enc)
	}
	for cat := range sum {
		if strings.Contains(string(cat), secret) {
			t.Fatalf("category leaked value")
		}
	}
}

func TestScanLargeInputBudget(t *testing.T) {
	const size = 1 << 20 // 1 MiB
	var b strings.Builder
	b.Grow(size)
	chunk := strings.Repeat("lorem ipsum dolor sit amet ", 40)
	for b.Len() < size-200 {
		b.WriteString(chunk)
	}
	b.WriteString(" aws=")
	b.WriteString(synthAWS())
	b.WriteString(" end")
	text := b.String()
	if len(text) < size/2 {
		t.Fatalf("fixture too small: %d", len(text))
	}
	start := time.Now()
	ms := Scan(text)
	elapsed := time.Since(start)
	if elapsed > 5*time.Second {
		t.Fatalf("1 MiB Scan took %s (budget 5s)", elapsed)
	}
	found := false
	for _, m := range ms {
		if m.Category == CategoryAWSKey {
			found = true
		}
	}
	if !found {
		t.Fatal("expected aws_key in large input")
	}
}

func TestNearMissProseZeroFalsePositives(t *testing.T) {
	prose := []string{
		"Please rotate your API key carefully when deploying to production.",
		"The AWS docs mention access key IDs that look like AKIA followed by letters.",
		"Use the Authorization header with a scheme and credential, but this sentence has none.",
		"Bearer alone is not a credential; neither is the word tokens beside it.",
		"A JWT has three base64 segments separated by dots; this is only prose.",
		"sk- is a common OpenAI prefix; sk-ant- is Anthropic; neither appears here complete.",
		"Clone https://github.com/example/project.git without embedded credentials.",
		"Shannon entropy is high for random strings, low for english sentences like this one.",
		"BEGIN PRIVATE KEY is incomplete without the PEM armor lines around a body.",
		"ghp_ tokens are GitHub PATs; a short mention of ghp_ is not enough.",
		"AIza keys are Google API keys; AIza alone must not match.",
		"Contact fixture-user@example.com about the runbook, not about secrets.",
		"Commit abcdef0123456789abcdef0123456789abcdef0 is a git object, not a secret.",
	}
	for _, line := range prose {
		ms := Scan(line)
		if len(ms) != 0 {
			t.Fatalf("false positive on %q → %v", line, ms)
		}
	}
}

func TestDigestIsSHA256Prefix(t *testing.T) {
	secret := synthAWS()
	ms := Scan(secret)
	if len(ms) != 1 {
		t.Fatalf("got %d", len(ms))
	}
	want := digestOf(secret)
	if ms[0].Digest != want {
		t.Fatalf("digest=%s want %s", ms[0].Digest, want)
	}
}

func TestScanEmptyAndNoMatch(t *testing.T) {
	if ms := Scan(""); len(ms) != 0 {
		t.Fatalf("empty: %v", ms)
	}
	if ms := Scan("hello world"); len(ms) != 0 {
		t.Fatalf("prose: %v", ms)
	}
	out, ms := Redact("clean text")
	if out != "clean text" || len(ms) != 0 {
		t.Fatalf("redact clean: %q %v", out, ms)
	}
	if sum := Summary(nil); len(sum) != 0 {
		t.Fatalf("summary nil: %v", sum)
	}
}
