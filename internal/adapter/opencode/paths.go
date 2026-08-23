package opencode

import (
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
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return json.RawMessage(data)
	}
	walkPaths(v, mapPath)
	out, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(data)
	}
	return json.RawMessage(out)
}

func walkPaths(v any, mapPath func(string) string) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if s, ok := child.(string); ok {
				if isPathKey(k) && isAbs(s) {
					t[k] = mapPath(s)
				}
				continue
			}
			walkPaths(child, mapPath)
		}
	case []any:
		for _, c := range t {
			walkPaths(c, mapPath)
		}
	}
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
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
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
