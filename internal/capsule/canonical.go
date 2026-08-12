package capsule

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CanonicalBytes returns the deterministic encoding used for hashing:
// sorted object keys, no insignificant whitespace, RFC3339 UTC timestamps with
// no sub-second component, and no wall-clock field anywhere in the output.
//
// Absolute filesystem paths are rejected. Portable pathmap tokens (${REPO:…},
// ${HOME}…, ${WORK:…}) are allowed.
func CanonicalBytes(c Capsule) ([]byte, error) {
	c = normalizeCapsule(c)
	raw, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("capsule: marshal: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("capsule: decode: %w", err)
	}
	if err := rejectAbsolutePaths(v); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := writeCanonical(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ComputeID returns the first 32 hex chars of sha256(CanonicalBytes) with
// Identity.ID cleared, so the ID is a fixed point over its own content.
func ComputeID(c Capsule) (string, error) {
	c.Identity.ID = ""
	b, err := CanonicalBytes(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:16]), nil
}

// EventID derives a stable event identity from its SourcePointer.
func EventID(p SourcePointer) string {
	var buf bytes.Buffer
	_ = writeCanonical(&buf, map[string]any{
		"agent":       p.Agent,
		"byte_offset": json.Number(strconv.FormatInt(p.ByteOffset, 10)),
		"index":       json.Number(strconv.Itoa(p.Index)),
		"record_key":  p.RecordKey,
		"session_id":  p.SessionID,
	})
	sum := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(sum[:16])
}

func normalizeCapsule(c Capsule) Capsule {
	for i := range c.Conversation.Events {
		t := c.Conversation.Events[i].Timestamp
		if t.IsZero() {
			continue
		}
		c.Conversation.Events[i].Timestamp = t.UTC().Truncate(time.Second)
	}
	return c
}

func writeCanonical(buf *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case nil:
		buf.WriteString("null")
		return nil
	case bool:
		if x {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
		return nil
	case string:
		enc, err := json.Marshal(x)
		if err != nil {
			return err
		}
		buf.Write(enc)
		return nil
	case json.Number:
		buf.WriteString(string(x))
		return nil
	case float64:
		// Fallback if UseNumber was not applied.
		buf.WriteString(strconv.FormatFloat(x, 'g', -1, 64))
		return nil
	case []any:
		buf.WriteByte('[')
		for i, elem := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeCanonical(buf, elem); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
		return nil
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			enc, err := json.Marshal(k)
			if err != nil {
				return err
			}
			buf.Write(enc)
			buf.WriteByte(':')
			if err := writeCanonical(buf, x[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
		return nil
	default:
		return fmt.Errorf("capsule: unsupported canonical type %T", v)
	}
}

func rejectAbsolutePaths(v any) error {
	switch x := v.(type) {
	case string:
		if isForbiddenAbsolutePath(x) {
			return fmt.Errorf("capsule: absolute filesystem path is not allowed: %q", x)
		}
		return nil
	case []any:
		for _, elem := range x {
			if err := rejectAbsolutePaths(elem); err != nil {
				return err
			}
		}
		return nil
	case map[string]any:
		for _, elem := range x {
			if err := rejectAbsolutePaths(elem); err != nil {
				return err
			}
		}
		return nil
	default:
		return nil
	}
}

func isForbiddenAbsolutePath(p string) bool {
	if p == "" {
		return false
	}
	if isPortablePathToken(p) {
		return false
	}
	if filepath.IsAbs(p) || strings.HasPrefix(p, "/") || strings.HasPrefix(p, `\`) {
		return true
	}
	return len(p) >= 3 &&
		((p[0] >= 'A' && p[0] <= 'Z') || (p[0] >= 'a' && p[0] <= 'z')) &&
		p[1] == ':' && (p[2] == '\\' || p[2] == '/')
}

func isPortablePathToken(p string) bool {
	return strings.HasPrefix(p, "${REPO:") ||
		strings.HasPrefix(p, "${HOME}") ||
		strings.HasPrefix(p, "${WORK:")
}
