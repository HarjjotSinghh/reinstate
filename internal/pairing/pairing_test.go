// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package pairing

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// samplePayload is a complete, realistic coordinate set: every optional field
// populated, so a round trip that silently drops one is visible.
func samplePayload() Payload {
	return Payload{
		Version:   Version,
		Endpoint:  "https://a1b2c3d4e5f6.r2.cloudflarestorage.com",
		Bucket:    "reinstate",
		Region:    "auto",
		Prefix:    "profiles/6f1d2a8e-4c3b-4f5a-9d2e-7b8c9a0d1e2f",
		ProfileID: "6f1d2a8e-4c3b-4f5a-9d2e-7b8c9a0d1e2f",
	}
}

// rawCode builds a pairing string out of arbitrary bytes, which is the only way
// to produce the malformed and future-version codes a decoder has to refuse.
func rawCode(body []byte) string { return Prefix + encoding.EncodeToString(body) }

func mustEncode(t *testing.T, payload Payload) string {
	t.Helper()
	code, err := Encode(payload)
	if err != nil {
		t.Fatalf("Encode(%+v) failed: %v", payload, err)
	}
	return code
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		payload Payload
	}{
		{name: "every field", payload: samplePayload()},
		{
			name: "no region and no prefix",
			payload: Payload{
				Endpoint:  "https://s3.us-east-1.amazonaws.com",
				Bucket:    "team-sessions",
				ProfileID: "0f9e8d7c-6b5a-4938-8271-605f4e3d2c1b",
			},
		},
		{
			name: "region without prefix",
			payload: Payload{
				Endpoint:  "https://s3.us-west-004.backblazeb2.com",
				Bucket:    "b",
				Region:    "us-west-004",
				ProfileID: "11111111-2222-4333-8444-555555555555",
			},
		},
		{
			name: "prefix without region",
			payload: Payload{
				Endpoint:  "http://minio.internal:9000",
				Bucket:    "reinstate-dev",
				Prefix:    "profiles/shared",
				ProfileID: "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
			},
		},
		{
			name: "values that need quoting in JSON",
			payload: Payload{
				Endpoint:  "https://héllo.example.com:9000/base",
				Bucket:    "bucket.with.dots",
				Region:    "eu-central-1",
				Prefix:    `profiles/"quoted"/path`,
				ProfileID: "12345678-90ab-4cde-8f01-234567890abc",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code := mustEncode(t, test.payload)

			decoded, err := Decode(code)
			if err != nil {
				t.Fatalf("Decode(%q) failed: %v", code, err)
			}

			want := test.payload
			want.Version = Version
			if decoded != want {
				t.Fatalf("round trip changed the payload\n got: %+v\nwant: %+v", decoded, want)
			}
		})
	}
}

// TestDecodeIsForgivingAboutTransport is the reason the format exists at all. A
// pairing code is read aloud, wrapped by a chat client, and retyped by hand, so
// every mangling that survives the content has to decode.
func TestDecodeIsForgivingAboutTransport(t *testing.T) {
	payload := samplePayload()
	code := mustEncode(t, payload)

	// A line-wrapped rendering, as printed by rein init --link.
	wrapped := strings.Join(Format(code, 24), "\n")

	tests := []struct {
		name    string
		mangled string
	}{
		{name: "verbatim", mangled: code},
		{name: "lowercased", mangled: strings.ToLower(code)},
		{name: "mixed case", mangled: strings.ToLower(code[:12]) + strings.ToUpper(code[12:])},
		{name: "leading and trailing whitespace", mangled: "\n\t  " + code + "  \n"},
		{name: "spaces inserted", mangled: spaceEvery(code, 4)},
		{name: "tabs inserted", mangled: strings.ReplaceAll(spaceEvery(code, 6), " ", "\t")},
		{name: "wrapped across lines", mangled: wrapped},
		{name: "wrapped and lowercased", mangled: strings.ToLower(wrapped)},
		{name: "windows line endings", mangled: strings.ReplaceAll(wrapped, "\n", "\r\n")},
		{name: "extra dashes", mangled: spaceEvery(code, 5, "-")},
		{name: "dashes and spaces and case", mangled: strings.ToLower(spaceEvery(code, 7, " - "))},
		{name: "every dash removed", mangled: strings.ReplaceAll(code, "-", "")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decoded, err := Decode(test.mangled)
			if err != nil {
				t.Fatalf("Decode(%q) failed: %v", test.mangled, err)
			}
			if decoded != payload {
				t.Fatalf("payload changed\n got: %+v\nwant: %+v", decoded, payload)
			}
		})
	}
}

// spaceEvery inserts a separator every n runes, defaulting to a single space.
func spaceEvery(s string, n int, separator ...string) string {
	sep := " "
	if len(separator) > 0 {
		sep = separator[0]
	}
	var builder strings.Builder
	for index, r := range []rune(s) {
		if index > 0 && index%n == 0 {
			builder.WriteString(sep)
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

// TestDecodeRejections checks that every refusal says something a person can
// act on. "Invalid input" would be true and useless.
func TestDecodeRejections(t *testing.T) {
	valid := mustEncode(t, samplePayload())

	futureVersion, err := json.Marshal(map[string]any{
		"v": Version + 1,
		"e": "https://s3.us-east-1.amazonaws.com",
		"b": "reinstate",
		"i": "6f1d2a8e-4c3b-4f5a-9d2e-7b8c9a0d1e2f",
	})
	if err != nil {
		t.Fatalf("build future-version payload: %v", err)
	}

	missing := func(without string) []byte {
		fields := map[string]any{
			"v": Version,
			"e": "https://s3.us-east-1.amazonaws.com",
			"b": "reinstate",
			"i": "6f1d2a8e-4c3b-4f5a-9d2e-7b8c9a0d1e2f",
		}
		delete(fields, without)
		body, marshalErr := json.Marshal(fields)
		if marshalErr != nil {
			t.Fatalf("build payload without %q: %v", without, marshalErr)
		}
		return body
	}

	tests := []struct {
		name string
		code string
		want string
	}{
		{
			name: "empty string",
			code: "",
			want: "not a Reinstate pairing code",
		},
		{
			name: "some other identifier entirely",
			code: "6f1d2a8e-4c3b-4f5a-9d2e-7b8c9a0d1e2f",
			want: "not a Reinstate pairing code",
		},
		{
			name: "a code from a different tool",
			code: "OTHER1-MFRGGZDF",
			want: "not a Reinstate pairing code",
		},
		{
			name: "the prefix alone",
			code: Prefix,
			want: "damaged or incomplete",
		},
		{
			name: "a character outside the alphabet",
			code: valid[:len(Prefix)+4] + "0" + valid[len(Prefix)+5:],
			want: "damaged or incomplete",
		},
		{
			name: "truncated mid-payload",
			code: valid[:len(valid)-9],
			want: "damaged or incomplete",
		},
		{
			name: "valid base32 that is not JSON",
			code: rawCode([]byte("this is not JSON at all")),
			want: "damaged or incomplete",
		},
		{
			name: "valid JSON that is not an object",
			code: rawCode([]byte(`["endpoint","bucket"]`)),
			want: "damaged or incomplete",
		},
		{
			name: "a newer format version",
			code: rawCode(futureVersion),
			want: "format version 2",
		},
		{
			name: "no endpoint",
			code: rawCode(missing("e")),
			want: "requires an endpoint",
		},
		{
			name: "no bucket",
			code: rawCode(missing("b")),
			want: "requires a bucket",
		},
		{
			name: "no profile ID",
			code: rawCode(missing("i")),
			want: "requires a profile ID",
		},
		{
			name: "a blank endpoint is the same as none",
			code: rawCode([]byte(`{"v":1,"e":"   ","b":"reinstate","i":"6f1d2a8e"}`)),
			want: "requires an endpoint",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload, err := Decode(test.code)
			if err == nil {
				t.Fatalf("Decode(%q) accepted the code and returned %+v", test.code, payload)
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %q, want it to mention %q", err.Error(), test.want)
			}
			if payload != (Payload{}) {
				t.Fatalf("a refused decode returned %+v, want the zero payload", payload)
			}
		})
	}

	// The prefix message has to show the reader what a real code looks like,
	// and the version message has to name both versions or it cannot be acted
	// on: one tells them to upgrade, the other tells them which build to trust.
	t.Run("the prefix refusal names the prefix", func(t *testing.T) {
		_, err := Decode("hello")
		if err == nil || !strings.Contains(err.Error(), Prefix) {
			t.Fatalf("error = %v, want it to quote %q", err, Prefix)
		}
	})

	t.Run("the version refusal names both versions", func(t *testing.T) {
		_, err := Decode(rawCode(futureVersion))
		if err == nil {
			t.Fatal("a future version was accepted")
		}
		message := err.Error()
		for _, want := range []string{"format version 2", "version 1"} {
			if !strings.Contains(message, want) {
				t.Fatalf("error = %q, want it to mention %q", message, want)
			}
		}
	})
}

func TestEncodeRejectsIncompletePayloads(t *testing.T) {
	complete := samplePayload()

	tests := []struct {
		name    string
		payload Payload
		want    string
	}{
		{
			name:    "no endpoint",
			payload: Payload{Bucket: complete.Bucket, ProfileID: complete.ProfileID},
			want:    "requires an endpoint",
		},
		{
			name:    "whitespace endpoint",
			payload: Payload{Endpoint: " \t ", Bucket: complete.Bucket, ProfileID: complete.ProfileID},
			want:    "requires an endpoint",
		},
		{
			name:    "no bucket",
			payload: Payload{Endpoint: complete.Endpoint, ProfileID: complete.ProfileID},
			want:    "requires a bucket",
		},
		{
			name:    "whitespace bucket",
			payload: Payload{Endpoint: complete.Endpoint, Bucket: "  ", ProfileID: complete.ProfileID},
			want:    "requires a bucket",
		},
		{
			name:    "no profile ID",
			payload: Payload{Endpoint: complete.Endpoint, Bucket: complete.Bucket},
			want:    "requires a profile ID",
		},
		{
			name:    "whitespace profile ID",
			payload: Payload{Endpoint: complete.Endpoint, Bucket: complete.Bucket, ProfileID: "\n"},
			want:    "requires a profile ID",
		},
		{
			name:    "nothing at all",
			payload: Payload{},
			want:    "requires an endpoint",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code, err := Encode(test.payload)
			if err == nil {
				t.Fatalf("Encode(%+v) produced %q, want a refusal", test.payload, code)
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %q, want it to mention %q", err.Error(), test.want)
			}
			if code != "" {
				t.Fatalf("a refused encode returned %q, want an empty string", code)
			}
		})
	}
}

// TestEncodeStampsTheCurrentVersion stops a caller from minting a code that
// claims to be a format it is not. The version is the decoder's only guard
// against reading fields that mean something different.
func TestEncodeStampsTheCurrentVersion(t *testing.T) {
	for _, claimed := range []int{0, Version, Version + 1, -7, 999} {
		payload := samplePayload()
		payload.Version = claimed

		code := mustEncode(t, payload)
		decoded, err := Decode(code)
		if err != nil {
			t.Fatalf("Decode of a code encoded with v=%d failed: %v", claimed, err)
		}
		if decoded.Version != Version {
			t.Fatalf("encoded v=%d, decoded v=%d, want %d", claimed, decoded.Version, Version)
		}
	}
}

// TestPayloadCarriesNoSecretMaterial is the security contract of this package,
// and the reason it is asserted structurally rather than by reading the docs.
//
// A pairing code is designed to be read aloud on a call and pasted into a chat
// window. Adding a field to Payload — an access key, a passphrase, an
// encryption identity, a signed token — would silently turn every printed code
// into a credential, and nothing else in the codebase would notice. If this
// test fails because a field was added, the field is the thing to reconsider.
func TestPayloadCarriesNoSecretMaterial(t *testing.T) {
	want := []struct {
		name string
		kind reflect.Kind
		tag  string
	}{
		{name: "Version", kind: reflect.Int, tag: "v"},
		{name: "Endpoint", kind: reflect.String, tag: "e"},
		{name: "Bucket", kind: reflect.String, tag: "b"},
		{name: "Region", kind: reflect.String, tag: "r,omitempty"},
		{name: "Prefix", kind: reflect.String, tag: "p,omitempty"},
		{name: "ProfileID", kind: reflect.String, tag: "i"},
	}

	payloadType := reflect.TypeOf(Payload{})
	if got := payloadType.NumField(); got != len(want) {
		var names []string
		for index := 0; index < got; index++ {
			names = append(names, payloadType.Field(index).Name)
		}
		t.Fatalf(
			"Payload has %d fields %v, want exactly %d.\n"+
				"A pairing code is read aloud and pasted into chat. If the new field "+
				"carries a key, a passphrase, or anything else secret, it must not "+
				"travel in a pairing code; if it is genuinely non-secret coordinate "+
				"data, add it here and to the round-trip table.",
			got, names, len(want))
	}
	for index, expected := range want {
		field := payloadType.Field(index)
		if field.Name != expected.name {
			t.Fatalf("field %d is %q, want %q", index, field.Name, expected.name)
		}
		if field.Type.Kind() != expected.kind {
			t.Fatalf("field %s is a %s, want a %s", field.Name, field.Type.Kind(), expected.kind)
		}
		if got := field.Tag.Get("json"); got != expected.tag {
			t.Fatalf("field %s has json tag %q, want %q", field.Name, got, expected.tag)
		}
		if !field.IsExported() {
			t.Fatalf("field %s is not exported; it would not survive a round trip", field.Name)
		}
	}

	// The wire form is checked too: an unexported or embedded field would not
	// show up above, but would still reach the encoded string.
	body, err := json.Marshal(samplePayload())
	if err != nil {
		t.Fatalf("marshal sample payload: %v", err)
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatalf("unmarshal sample payload: %v", err)
	}
	if len(wire) != len(want) {
		t.Fatalf("the encoded form has %d keys %v, want %d", len(wire), wire, len(want))
	}
	for _, expected := range want {
		key := strings.Split(expected.tag, ",")[0]
		if _, ok := wire[key]; !ok {
			t.Fatalf("the encoded form has no %q key: %v", key, wire)
		}
	}
}

func TestFormat(t *testing.T) {
	code := mustEncode(t, samplePayload())

	t.Run("lines never exceed the requested width", func(t *testing.T) {
		for _, width := range []int{1, 2, 3, 7, 16, 24, 47, 48, 56, 200} {
			lines := Format(code, width)
			for index, line := range lines {
				if got := len([]rune(line)); got > width {
					t.Fatalf("width %d: line %d is %d runes: %q", width, index, got, line)
				}
			}
			if got := strings.Join(lines, ""); got != code {
				t.Fatalf("width %d: rejoined lines = %q, want %q", width, got, code)
			}
			// Only the last line may be short, or the wrapping wastes a terminal.
			for index, line := range lines[:max(len(lines)-1, 0)] {
				if len([]rune(line)) != width {
					t.Fatalf("width %d: line %d is short at %d runes: %q",
						width, index, len([]rune(line)), line)
				}
			}
		}
	})

	t.Run("a non-positive width falls back to a sane default", func(t *testing.T) {
		for _, width := range []int{0, -1, -48} {
			lines := Format(code, width)
			if len(lines) == 0 {
				t.Fatalf("width %d produced no lines", width)
			}
			for index, line := range lines {
				if got := len([]rune(line)); got > 48 {
					t.Fatalf("width %d: line %d is %d runes, want at most the 48-rune default",
						width, index, got)
				}
			}
			if got := strings.Join(lines, ""); got != code {
				t.Fatalf("width %d: rejoined lines = %q, want %q", width, got, code)
			}
		}
	})

	t.Run("a short code stays on one line", func(t *testing.T) {
		lines := Format("REIN1-ABCD", 48)
		if len(lines) != 1 {
			t.Fatalf("Format produced %d lines, want 1: %q", len(lines), lines)
		}
		if lines[0] != "REIN1-ABCD" {
			t.Fatalf("line = %q, want the code unchanged", lines[0])
		}
	})

	t.Run("a code exactly one line long is not split", func(t *testing.T) {
		exact := strings.Repeat("A", 24)
		if lines := Format(exact, 24); len(lines) != 1 || lines[0] != exact {
			t.Fatalf("Format(%d runes, width 24) = %q, want one line", len(exact), lines)
		}
	})

	t.Run("an empty code produces nothing to print", func(t *testing.T) {
		if lines := Format("", 24); len(lines) != 0 {
			t.Fatalf("Format(\"\") = %q, want no lines", lines)
		}
	})

	// The wrapped rendering is what a reader retypes, so it must survive the
	// trip back through Decode with the newlines the terminal put in.
	t.Run("wrapped output decodes again", func(t *testing.T) {
		for _, width := range []int{8, 24, 56} {
			joined := strings.Join(Format(code, width), "\n")
			decoded, err := Decode(joined)
			if err != nil {
				t.Fatalf("width %d: Decode of the wrapped code failed: %v", width, err)
			}
			if decoded != samplePayload() {
				t.Fatalf("width %d: decoded %+v, want %+v", width, decoded, samplePayload())
			}
		}
	})
}

// TestEncodingAvoidsAmbiguousCharacters is why the alphabet is custom. A code is
// dictated over a call: 0 against O and 1 against l are the two mistakes a
// listener makes, and 8 and 9 are excluded by the same RFC 4648 alphabet.
func TestEncodingAvoidsAmbiguousCharacters(t *testing.T) {
	payloads := []Payload{
		samplePayload(),
		{
			Endpoint:  "https://s3.eu-west-2.amazonaws.com",
			Bucket:    "0123456789",
			Region:    "eu-west-2",
			Prefix:    "0/1/8/9",
			ProfileID: "01189998-8199-4911-8197-253000000000",
		},
		{
			Endpoint:  "http://localhost:9000",
			Bucket:    "l1O0",
			ProfileID: "88888888-9999-4000-8111-000000000000",
		},
	}
	for _, payload := range payloads {
		code := mustEncode(t, payload)
		body := strings.TrimPrefix(code, Prefix)
		if body == code {
			t.Fatalf("code %q does not start with %q", code, Prefix)
		}
		if index := strings.IndexAny(body, "0189"); index >= 0 {
			t.Fatalf("code body %q contains the ambiguous character %q at %d",
				body, body[index], index)
		}
		for _, r := range body {
			isUpper := r >= 'A' && r <= 'Z'
			isDigit := r >= '2' && r <= '7'
			if !isUpper && !isDigit {
				t.Fatalf("code body %q contains %q, which is outside the alphabet", body, r)
			}
		}
		// Padding would add '=', which no one wants to read aloud either.
		if strings.Contains(code, "=") {
			t.Fatalf("code %q is padded", code)
		}
	}
}
