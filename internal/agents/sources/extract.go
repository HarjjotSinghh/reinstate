// Package sources holds shared record-mapping helpers used by per-agent
// index ports. Agent-specific mapping lives in sources/<key>/.
package sources

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// BoundedText accumulates searchable prompt text up to MaxSearchTextBytes.
type BoundedText struct {
	value  strings.Builder
	sealed bool
}

// Add appends sanitized text without exceeding the shared search-text ceiling.
func (b *BoundedText) Add(value string) {
	if b.sealed {
		return
	}
	value = sessionindex.SafeText(value, 0)
	if value == "" {
		return
	}
	remaining := sessionindex.MaxSearchTextBytes - b.value.Len()
	if b.value.Len() > 0 {
		if remaining <= 1 {
			b.sealed = true
			return
		}
		value = truncateUTF8Bytes(value, remaining-1)
		if value == "" {
			return
		}
		b.value.WriteByte(' ')
	} else {
		value = truncateUTF8Bytes(value, remaining)
	}
	b.value.WriteString(value)
	if b.value.Len() == sessionindex.MaxSearchTextBytes {
		b.sealed = true
	}
}

// String returns the accumulated text.
func (b *BoundedText) String() string { return b.value.String() }

// FirstString returns the first non-empty string field among keys.
func FirstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key].(string); ok && value != "" {
			return value
		}
	}
	return ""
}

// BoolValue reports whether value is a true boolean.
func BoolValue(value any) bool {
	result, _ := value.(bool)
	return result
}

// EventTimestamp reads the first recognized timestamp field.
func EventTimestamp(values map[string]any) int64 {
	for _, key := range []string{"timestamp", "updatedAt", "updated_at", "lastUpdated", "createdAt", "created_at"} {
		if timestamp := ParseTimestamp(values[key]); timestamp != 0 {
			return timestamp
		}
	}
	return 0
}

// ParseTimestamp normalizes vendor timestamp encodings to unix seconds.
func ParseTimestamp(value any) int64 {
	switch typed := value.(type) {
	case string:
		if parsed, err := time.Parse(time.RFC3339Nano, typed); err == nil {
			return parsed.Unix()
		}
		if number, err := strconv.ParseInt(typed, 10, 64); err == nil {
			return normalizeUnix(number)
		}
	case float64:
		return normalizeUnix(int64(typed))
	case json.Number:
		if number, err := typed.Int64(); err == nil {
			return normalizeUnix(number)
		}
	case int64:
		return normalizeUnix(typed)
	case int:
		return normalizeUnix(int64(typed))
	}
	return 0
}

// ExtractTextContent flattens vendor text / content-block payloads.
func ExtractTextContent(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		var texts []string
		for _, raw := range typed {
			if text, ok := raw.(string); ok {
				texts = append(texts, text)
				continue
			}
			block, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			blockType := strings.ToLower(FirstString(block, "type"))
			if blockType != "" && blockType != "text" && blockType != "input_text" {
				continue
			}
			if text := FirstString(block, "text"); text != "" {
				texts = append(texts, text)
			}
		}
		return strings.Join(texts, "\n")
	case map[string]any:
		return FirstString(typed, "text")
	default:
		return ""
	}
}

// CollectToolFiles records structured file paths from a vendor event.
func CollectToolFiles(event map[string]any, files map[string]struct{}) {
	collectStructuredFileFields(event["input"], files)
	collectStructuredFileFields(event["arguments"], files)
	if message, ok := event["message"].(map[string]any); ok {
		if blocks, ok := message["content"].([]any); ok {
			for _, raw := range blocks {
				block, ok := raw.(map[string]any)
				if !ok {
					continue
				}
				if strings.EqualFold(FirstString(block, "type"), "tool_use") {
					collectStructuredFileFields(block["input"], files)
				}
			}
		}
	}
	if payload, ok := event["payload"].(map[string]any); ok {
		collectStructuredFileFields(payload["input"], files)
		collectStructuredFileFields(payload["arguments"], files)
	}
}

// AddFilePath records one sanitized file path.
func AddFilePath(value string, files map[string]struct{}) {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\r\n\x00") || len(files) >= sessionindex.MaxFileReferences {
		return
	}
	files[value] = struct{}{}
}

// NormalizedFileMap bounds, sanitizes, and sorts a file set.
func NormalizedFileMap(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	return sessionindex.NormalizeFiles(result)
}

// PortableBase returns the last path element using slash separators.
func PortableBase(path string) string {
	path = strings.TrimRight(strings.ReplaceAll(path, `\`, "/"), "/")
	if index := strings.LastIndex(path, "/"); index >= 0 {
		path = path[index+1:]
	}
	if path == "" {
		return "unknown"
	}
	return path
}

// MaxInt64 returns the larger of a and b.
func MaxInt64(a, b int64) int64 {
	if b > a {
		return b
	}
	return a
}

// SortRecordsBySourcePath sorts records for deterministic scan output.
func SortRecordsBySourcePath(records []sessionindex.Record) {
	sort.Slice(records, func(i, j int) bool {
		return records[i].SourcePath < records[j].SourcePath
	})
}

// MapsFromAny keeps object entries from a JSON array.
func MapsFromAny(values []any) []map[string]any {
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		if mapped, ok := value.(map[string]any); ok {
			result = append(result, mapped)
		}
	}
	return result
}

// FirstInt reads the first integer-like field among keys.
func FirstInt(values map[string]any, keys ...string) int {
	for _, key := range keys {
		switch value := values[key].(type) {
		case int:
			return value
		case float64:
			return int(value)
		case json.Number:
			number, err := strconv.Atoi(value.String())
			if err == nil {
				return number
			}
		case string:
			number, err := strconv.Atoi(strings.TrimSpace(value))
			if err == nil {
				return number
			}
		}
	}
	return 0
}

func collectStructuredFileFields(value any, files map[string]struct{}) {
	if len(files) >= sessionindex.MaxFileReferences || value == nil {
		return
	}
	if encoded, ok := value.(string); ok {
		var decoded any
		if json.Unmarshal([]byte(encoded), &decoded) != nil {
			return
		}
		value = decoded
	}
	collectStructuredFileValue(value, files)
}

func collectStructuredFileValue(value any, files map[string]struct{}) {
	if len(files) >= sessionindex.MaxFileReferences || value == nil {
		return
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if isFileField(key) {
				switch fileValue := nested.(type) {
				case string:
					AddFilePath(fileValue, files)
				case []any:
					for _, item := range fileValue {
						if path, ok := item.(string); ok {
							AddFilePath(path, files)
						}
					}
				}
			}
			collectStructuredFileValue(nested, files)
		}
	case []any:
		for _, nested := range typed {
			collectStructuredFileValue(nested, files)
		}
	}
}

func isFileField(key string) bool {
	normalized := strings.NewReplacer("_", "", "-", "").Replace(strings.ToLower(key))
	switch normalized {
	case "path", "file", "filename", "filepath", "files", "paths", "targetpath", "sourcepath", "destinationpath":
		return true
	default:
		return false
	}
}

func normalizeUnix(value int64) int64 {
	switch {
	case value > 1_000_000_000_000_000:
		return value / 1_000_000_000
	case value > 1_000_000_000_000:
		return value / 1_000
	default:
		return value
	}
}

func truncateUTF8Bytes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if len(value) <= limit {
		return value
	}
	value = value[:limit]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}
