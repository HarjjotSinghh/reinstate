package handoff

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"

	"github.com/HarjjotSinghh/reinstate/internal/fsx"
)

const (
	claudeDestConfigName = ".claude.json"
	codexDestConfigName  = "config.toml"
)

func acceptClaudeWorkspaceTrust(configDir, workspace string) error {
	root, workspace, err := destTrustArgs(configDir, workspace)
	if err != nil {
		return err
	}
	path := filepath.Join(root, claudeDestConfigName)
	doc := map[string]any{}
	if data, readErr := os.ReadFile(path); readErr == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("handoff: parse dest Claude config: %w", err)
		}
	} else if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return readErr
	}
	projects := nestedObject(doc, "projects")
	for _, key := range workspaceTrustKeys(runtime.GOOS, workspace) {
		proj := nestedObject(projects, key)
		proj["hasTrustDialogAccepted"] = true
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return writeDestTrustFile(path, out)
}

func acceptCodexProjectTrust(codexHome, workspace string) error {
	root, workspace, err := destTrustArgs(codexHome, workspace)
	if err != nil {
		return err
	}
	path := filepath.Join(root, codexDestConfigName)
	doc := map[string]any{}
	if data, readErr := os.ReadFile(path); readErr == nil && len(data) > 0 {
		if err := toml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("handoff: parse dest Codex config: %w", err)
		}
	} else if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return readErr
	}
	projects := nestedObject(doc, "projects")
	for _, key := range workspaceTrustKeys(runtime.GOOS, workspace) {
		proj := nestedObject(projects, key)
		proj["trust_level"] = "trusted"
	}
	out, err := marshalCodexTOML(doc)
	if err != nil {
		return err
	}
	return writeDestTrustFile(path, out)
}

func destTrustArgs(configRoot, workspace string) (string, string, error) {
	root := strings.TrimSpace(configRoot)
	workspace = strings.TrimSpace(workspace)
	if root == "" {
		return "", "", errors.New("handoff: dest config root is required for workspace trust")
	}
	if workspace == "" {
		return "", "", errors.New("handoff: dest workspace is required for workspace trust")
	}
	if !filepath.IsAbs(workspace) {
		abs, err := filepath.Abs(workspace)
		if err != nil {
			return "", "", err
		}
		workspace = abs
	}
	workspace = filepath.Clean(workspace)
	if err := fsx.EnsureOwnerOnlyDir(root); err != nil {
		return "", "", err
	}
	return filepath.Clean(root), workspace, nil
}

// workspaceTrustKeys returns dest-cwd aliases Codex/Claude may look up.
// Windows dest-ack (rc.10) needed both slash styles and a lowercased key;
// double-quoted TOML `"C:\Users\..."` is unsafe because `\U` is a unicode escape.
func workspaceTrustKeys(goos, workspace string) []string {
	native := strings.TrimSpace(workspace)
	if native == "" {
		return nil
	}
	fwd := strings.ReplaceAll(native, `\`, "/")
	keys := []string{native}
	add := func(s string) {
		if s == "" {
			return
		}
		for _, existing := range keys {
			if existing == s {
				return
			}
		}
		keys = append(keys, s)
	}
	add(fwd)
	if goos == "windows" {
		add(strings.ToLower(native))
		add(strings.ToLower(fwd))
	}
	return keys
}

func nestedObject(parent map[string]any, key string) map[string]any {
	if parent == nil {
		return map[string]any{}
	}
	if existing, ok := parent[key].(map[string]any); ok && existing != nil {
		return existing
	}
	child := map[string]any{}
	parent[key] = child
	return child
}

func writeDestTrustFile(path string, data []byte) error {
	if err := fsx.WriteFileAtomic(path, data, 0o600); err != nil {
		return err
	}
	return fsx.ProtectOwnerOnly(path, false)
}

func marshalCodexTOML(doc map[string]any) ([]byte, error) {
	projects, _ := doc["projects"].(map[string]any)
	rest := make(map[string]any, len(doc))
	for k, v := range doc {
		if k == "projects" {
			continue
		}
		rest[k] = v
	}
	var buf bytes.Buffer
	if len(rest) > 0 {
		head, err := toml.Marshal(rest)
		if err != nil {
			return nil, err
		}
		buf.Write(head)
		if buf.Len() > 0 && buf.Bytes()[buf.Len()-1] != '\n' {
			buf.WriteByte('\n')
		}
	}
	keys := make([]string, 0, len(projects))
	for k := range projects {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		proj, ok := projects[key].(map[string]any)
		if !ok || proj == nil {
			continue
		}
		fmt.Fprintf(&buf, "[projects.%s]\n", tomlTableKey(key))
		fields := make([]string, 0, len(proj))
		for field := range proj {
			fields = append(fields, field)
		}
		sort.Strings(fields)
		for _, field := range fields {
			fmt.Fprintf(&buf, "%s = %s\n", tomlBareOrQuote(field), tomlScalar(proj[field]))
		}
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}

func tomlTableKey(key string) string {
	if isBareTOMLKey(key) {
		return key
	}
	if strings.ContainsRune(key, '\\') || strings.ContainsRune(key, '\'') {
		return "'" + strings.ReplaceAll(key, "'", "''") + "'"
	}
	return strconv.Quote(key)
}

func tomlBareOrQuote(key string) string {
	if isBareTOMLKey(key) {
		return key
	}
	return strconv.Quote(key)
}

func isBareTOMLKey(key string) bool {
	if key == "" {
		return false
	}
	for _, r := range key {
		if r > unicode.MaxASCII || (!unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-') {
			return false
		}
	}
	return true
}

func tomlScalar(v any) string {
	switch t := v.(type) {
	case string:
		return strconv.Quote(t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return strconv.Quote(fmt.Sprint(v))
	}
}
