package opencode

import (
	"bytes"
	"encoding/json"
	"strings"
)

// rewriteJSONPaths walks a message or part data blob and rewrites the values of
// recognised path keys through mapPath, leaving prose and every other field
// untouched. An unparseable blob is returned unchanged rather than guessed at.
func rewriteJSONPaths(data []byte, mapPath func(string) string) json.RawMessage {
	if len(strings.TrimSpace(string(data))) == 0 {
		return json.RawMessage(data)
	}
	v, err := decodeJSON(data)
	if err != nil {
		return json.RawMessage(data)
	}
	if !walkPaths(v, mapPath) {
		// Nothing was rewritten: keep the vendor's bytes rather than re-keying
		// the blob (sorted keys, HTML-escaped runes) for no reason.
		return json.RawMessage(data)
	}
	out, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(data)
	}
	return json.RawMessage(out)
}

// decodeJSON parses a blob with numbers kept as their literal text, so a
// re-marshalled body carries integers above 2^53 (ids, nanosecond stamps)
// byte-for-byte rather than rounded through float64.
func decodeJSON(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

// walkPaths rewrites every absolute path value under a path key in place and
// reports whether any value actually changed.
func walkPaths(v any, mapPath func(string) string) bool {
	changed := false
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if s, ok := child.(string); ok {
				if isPathKey(k) && isAbs(s) {
					if mapped := mapPath(s); mapped != s {
						t[k] = mapped
						changed = true
					}
				}
				continue
			}
			if walkPaths(child, mapPath) {
				changed = true
			}
		}
	case []any:
		for _, c := range t {
			if walkPaths(c, mapPath) {
				changed = true
			}
		}
	}
	return changed
}

func isPathKey(key string) bool {
	switch key {
	case "cwd", "root", "directory", "worktree", "path", "filePath", "file_path":
		return true
	default:
		return false
	}
}

func isAbs(s string) bool {
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, "${") {
		return true
	}
	return len(s) > 2 && s[1] == ':' && (s[2] == '\\' || s[2] == '/')
}

// rewriteSessionRef replaces a source session id with the target id inside a
// message or part data blob, so a forked copy references its own session rather
// than the original. Only exact whole-string "sessionID" fields are rewritten.
func rewriteSessionRef(data json.RawMessage, sourceID, targetID string) json.RawMessage {
	if sourceID == "" || targetID == "" || sourceID == targetID || len(data) == 0 {
		return data
	}
	v, err := decodeJSON(data)
	if err != nil {
		return data
	}
	walkSessionRef(v, sourceID, targetID)
	out, err := json.Marshal(v)
	if err != nil {
		return data
	}
	return json.RawMessage(out)
}

func walkSessionRef(v any, sourceID, targetID string) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if s, ok := child.(string); ok {
				if isSessionRefKey(k) && s == sourceID {
					t[k] = targetID
				}
				continue
			}
			walkSessionRef(child, sourceID, targetID)
		}
	case []any:
		for _, c := range t {
			walkSessionRef(c, sourceID, targetID)
		}
	}
}

func isSessionRefKey(key string) bool {
	switch key {
	case "sessionID", "session_id", "sessionId", "parentID", "parent_id":
		return true
	default:
		return false
	}
}

// messageIDKeys are the data fields that hold a message id: a message's own id,
// the user message an assistant message answers, and the message a part
// belongs to.
var messageIDKeys = map[string]bool{
	"id": true, "parentID": true, "parent_id": true, "messageID": true, "message_id": true,
}

// partIDKeys are the data fields that hold a part's own id.
var partIDKeys = map[string]bool{"id": true}

// rewriteIDRefs replaces, at the top level of a data blob, every string value
// under one of keys that is an exact key of idMap with the mapped id. Only
// whole-string matches are rewritten so prose mentioning an id is untouched.
// Nested objects are left alone: OpenCode keeps identity fields at the top of
// each message and part row.
func rewriteIDRefs(data json.RawMessage, idMap map[string]string, keys map[string]bool) json.RawMessage {
	if len(data) == 0 || len(idMap) == 0 {
		return data
	}
	v, err := decodeJSON(data)
	if err != nil {
		return data
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return data
	}
	changed := false
	for k, child := range obj {
		s, isString := child.(string)
		if !isString || !keys[k] {
			continue
		}
		if mapped, found := idMap[s]; found && mapped != s {
			obj[k] = mapped
			changed = true
		}
	}
	if !changed {
		return data
	}
	out, err := json.Marshal(obj)
	if err != nil {
		return data
	}
	return json.RawMessage(out)
}
