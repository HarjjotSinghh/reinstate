// Package pairing encodes the non-secret storage coordinates that a second
// device needs in order to join an existing Reinstate profile.
//
// Setting up a second device currently means carrying four things by hand: an
// endpoint, a bucket, a prefix, and a profile UUID. Three of them are long and
// one is a UUID, and getting any of them wrong produces a failure that looks
// like an authentication problem. A pairing code carries all four as one
// string.
//
// # What is deliberately not in here
//
// No access key, no secret key, no passphrase, no encryption identity. A
// pairing code is not a credential and must not become one: it is designed to
// be readable over a video call or pasted into a chat window, and anything
// secret in it would then be compromised. The receiving device still asks for
// storage keys and a passphrase through the normal hardened prompts.
//
// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.
package pairing

import (
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Version is the pairing payload schema version. A decoder refuses a version it
// does not know rather than guessing at the fields.
const Version = 1

// Prefix marks a pairing string so a human can recognise one, and so a decoder
// can refuse anything else with a clear message.
const Prefix = "REIN1-"

// Payload is the non-secret coordinate set.
type Payload struct {
	Version   int    `json:"v"`
	Endpoint  string `json:"e"`
	Bucket    string `json:"b"`
	Region    string `json:"r,omitempty"`
	Prefix    string `json:"p,omitempty"`
	ProfileID string `json:"i"`
}

// encoding is unpadded base32 without ambiguous characters in the alphabet, so
// a code can be read aloud and typed back without confusing 0 and O or 1 and l.
var encoding = base32.NewEncoding("ABCDEFGHIJKLMNOPQRSTUVWXYZ234567").WithPadding(base32.NoPadding)

// Encode renders a payload as a pairing string.
func Encode(payload Payload) (string, error) {
	payload.Version = Version
	if err := validate(payload); err != nil {
		return "", err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode pairing payload: %w", err)
	}
	return Prefix + encoding.EncodeToString(body), nil
}

// Decode parses a pairing string.
//
// Whitespace and casing are forgiving because these strings get wrapped by
// chat clients and read aloud; the content is not.
func Decode(code string) (Payload, error) {
	cleaned := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' || r == '-' {
			return -1
		}
		return r
	}, code)
	cleaned = strings.ToUpper(strings.TrimSpace(cleaned))

	trimmedPrefix := strings.ToUpper(strings.ReplaceAll(Prefix, "-", ""))
	if !strings.HasPrefix(cleaned, trimmedPrefix) {
		return Payload{}, errors.New("this is not a Reinstate pairing code; it should start with " + Prefix)
	}
	cleaned = strings.TrimPrefix(cleaned, trimmedPrefix)

	body, err := encoding.DecodeString(cleaned)
	if err != nil {
		return Payload{}, errors.New("pairing code is damaged or incomplete")
	}
	var payload Payload
	if err := json.Unmarshal(body, &payload); err != nil {
		return Payload{}, errors.New("pairing code is damaged or incomplete")
	}
	if payload.Version != Version {
		return Payload{}, fmt.Errorf(
			"pairing code uses format version %d; this build understands version %d",
			payload.Version, Version,
		)
	}
	if err := validate(payload); err != nil {
		return Payload{}, err
	}
	return payload, nil
}

func validate(payload Payload) error {
	switch {
	case strings.TrimSpace(payload.Endpoint) == "":
		return errors.New("pairing payload requires an endpoint")
	case strings.TrimSpace(payload.Bucket) == "":
		return errors.New("pairing payload requires a bucket")
	case strings.TrimSpace(payload.ProfileID) == "":
		return errors.New("pairing payload requires a profile ID")
	}
	return nil
}

// Format wraps a pairing code into fixed-width lines for display.
func Format(code string, width int) []string {
	if width <= 0 {
		width = 48
	}
	runes := []rune(code)
	lines := make([]string, 0, len(runes)/width+1)
	for start := 0; start < len(runes); start += width {
		end := start + width
		if end > len(runes) {
			end = len(runes)
		}
		lines = append(lines, string(runes[start:end]))
	}
	return lines
}
