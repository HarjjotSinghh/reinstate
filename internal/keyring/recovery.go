package keyring

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"strings"
)

// Recovery-code shape: eight groups of four Crockford base32 characters.
// The first seven groups carry 140 random bits (above the 128-bit floor); the
// last group is a 20-bit checksum over the data groups, so a typo in any
// position is rejected before a key derivation is attempted.
const (
	recoveryGroupLen    = 4
	recoveryDataGroups  = 7
	recoveryTotalGroups = 8
	recoveryDataChars   = recoveryGroupLen * recoveryDataGroups
	recoveryTotalChars  = recoveryGroupLen * recoveryTotalGroups
	recoveryDataBits    = recoveryDataChars * 5
	recoveryDataBytes   = (recoveryDataBits + 7) / 8
)

// Crockford base32: no I, L, O, or U, so the code survives handwriting and
// reading aloud. Decoding folds the confusable letters back.
const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// RecoveryCodeFormat is the human description printed in help text.
const RecoveryCodeFormat = "XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX"

// GenerateRecoveryCode draws a fresh recovery code from crypto/rand and
// returns it grouped for display.
func GenerateRecoveryCode() (string, error) {
	raw := make([]byte, recoveryDataBytes)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return "", fmt.Errorf("generate recovery code: %w", err)
	}
	data := encodeCrockford(raw, recoveryDataChars)
	return formatRecoveryCode(data + recoveryChecksum(data)), nil
}

// NormalizeRecoveryCode accepts a code as a person would type it (any case,
// with or without separators, with O/I/L folded to 0/1) and returns the
// canonical grouped form, or an error when the length or checksum is wrong.
// The canonical form is the only string ever fed to the key derivation.
func NormalizeRecoveryCode(typed string) (string, error) {
	var compact strings.Builder
	for i, r := range strings.ToUpper(typed) {
		switch {
		case r == '-' || r == ' ' || r == '\t' || r == '\r' || r == '\n':
			continue
		case r == 'O':
			compact.WriteByte('0')
		case r == 'I' || r == 'L':
			compact.WriteByte('1')
		case strings.ContainsRune(crockford, r):
			compact.WriteRune(r)
		default:
			return "", fmt.Errorf("recovery code contains an invalid character at position %d (only Crockford base32 letters and digits, dashes, and spaces are allowed)", i+1)
		}
	}
	code := compact.String()
	if len(code) != recoveryTotalChars {
		return "", fmt.Errorf("recovery code must have %d groups of %d characters (%s)", recoveryTotalGroups, recoveryGroupLen, RecoveryCodeFormat)
	}
	data, check := code[:recoveryDataChars], code[recoveryDataChars:]
	if check != recoveryChecksum(data) {
		return "", fmt.Errorf("recovery code checksum does not match; check it for typos")
	}
	return formatRecoveryCode(code), nil
}

func recoveryChecksum(data string) string {
	sum := sha256.Sum256([]byte("reinstate/recovery-code/v1:" + data))
	return encodeCrockford(sum[:], recoveryGroupLen)
}

func encodeCrockford(raw []byte, chars int) string {
	var out strings.Builder
	acc := uint32(0)
	bits := 0
	for _, b := range raw {
		acc = acc<<8 | uint32(b)
		bits += 8
		for bits >= 5 && out.Len() < chars {
			bits -= 5
			out.WriteByte(crockford[(acc>>bits)&31])
		}
		if out.Len() == chars {
			break
		}
	}
	for out.Len() < chars {
		// Only reachable when raw is shorter than chars needs; pad with
		// the remaining low bits so the encoding stays total.
		acc <<= 5 - bits
		out.WriteByte(crockford[acc&31])
		bits = 0
	}
	return out.String()
}

func formatRecoveryCode(compact string) string {
	groups := make([]string, 0, recoveryTotalGroups)
	for i := 0; i < len(compact); i += recoveryGroupLen {
		groups = append(groups, compact[i:i+recoveryGroupLen])
	}
	return strings.Join(groups, "-")
}
