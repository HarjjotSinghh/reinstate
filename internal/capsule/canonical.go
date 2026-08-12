package capsule

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/pathmap"
)

// CanonicalBytes returns the deterministic encoding used for hashing:
// sorted object keys, no insignificant whitespace, RFC3339 UTC timestamps with
// no sub-second component, and no wall-clock field anywhere in the output.
//
// Absolute filesystem paths are rejected. Portable pathmap tokens (${REPO:…},
// ${HOME}…, ${WORK:…}, ${EXTERNAL:…}) are allowed.
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

// ComputeID returns the first 32 hex chars of sha256 over the capsule's
// canonical identity preimage. The preimage excludes Identity.ID, a
// self-referential LineageRoot equal to that ID, and projection size/hash
// fields because they are derived from the ID or from artifacts rendered after
// the ID is assigned. A distinct ancestor LineageRoot remains identity-bearing,
// as do policy, included event IDs, sidecar selection, and source/task/workspace
// content.
func ComputeID(c Capsule) (string, error) {
	if c.Identity.LineageRoot == c.Identity.ID {
		c.Identity.LineageRoot = ""
	}
	c.Identity.ID = ""
	c.Projection.EstimatedBytes = 0
	c.Projection.EstimatedTokens = 0
	c.Projection.BootstrapSHA256 = ""
	c.Projection.MarkdownSHA256 = ""
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

// AbsolutePathForbidden reports whether CanonicalBytes rejects s because it is
// an absolute filesystem path rather than a portable pathmap token.
//
// Transcript readers use this predicate at their emit boundary so a reader can
// never produce a value the capsule refuses. It is exported to keep exactly one
// definition of "absolute path" in the codebase; it does not relax the rule.
func AbsolutePathForbidden(s string) bool {
	return isForbiddenAbsolutePath(s)
}

func isForbiddenAbsolutePath(p string) bool {
	if p == "" || isPortablePathToken(p) {
		return false
	}
	return pathmap.IsAbsolutePlatform(p)
}

// isPortablePathToken defers to pathmap so the token vocabulary and the
// definition of an absolute path have exactly one owner.
func isPortablePathToken(p string) bool {
	return pathmap.IsToken(p)
}
