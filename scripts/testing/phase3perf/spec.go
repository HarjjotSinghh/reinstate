package main

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	workspacePlaceholder = "${WORKSPACE}"
	manifestFileName     = "manifest.json"
)

//go:embed spec.json
var embeddedSpec []byte

type corpusSpec struct {
	Name                  string `json:"name"`
	ClaudeRecords         int    `json:"claude_records"`
	CodexRecords          int    `json:"codex_records"`
	ClaudeCapabilityNames int    `json:"claude_capability_names"`
	CodexCapabilityNames  int    `json:"codex_capability_names"`
	Query                 string `json:"query"`
	Limit                 int    `json:"limit"`
	TimestampBase         string `json:"timestamp_base"`
	TimestampStepSeconds  int64  `json:"timestamp_step_seconds"`
	AnchorIndex           int    `json:"anchor_index"`
}

type fixtureSpec struct {
	SchemaVersion               int        `json:"schema_version"`
	Generator                   string     `json:"generator"`
	CanonicalDigest             string     `json:"canonical_digest"`
	EventCountPerRecord         int        `json:"event_count_per_record"`
	MessageCountPerRecord       int        `json:"message_count_per_record"`
	FileReferenceCountPerRecord int        `json:"file_reference_count_per_record"`
	SessionIDBytes              int        `json:"session_id_bytes"`
	TitleBytes                  int        `json:"title_bytes"`
	UserMessageBytes            int        `json:"user_message_bytes"`
	AssistantMessageBytes       int        `json:"assistant_message_bytes"`
	FileReferenceBytes          int        `json:"file_reference_bytes"`
	CapabilityNameBytes         int        `json:"capability_name_bytes"`
	WorkspaceFileBytes          int        `json:"workspace_file_bytes"`
	GitBranch                   string     `json:"git_branch"`
	GitAuthorName               string     `json:"git_author_name"`
	GitAuthorEmail              string     `json:"git_author_email"`
	GitCommitTimestamp          string     `json:"git_commit_timestamp"`
	GitCommitMessage            string     `json:"git_commit_message"`
	GitExpectedHead             string     `json:"git_expected_head"`
	Normal                      corpusSpec `json:"normal"`
	Large                       corpusSpec `json:"large"`
}

type fixtureFile struct {
	RelativePath string
	Canonical    []byte
	Materialized []byte
	Mode         fs.FileMode
	ModTime      time.Time
}

type corpusManifest struct {
	SchemaVersion              int      `json:"schema_version"`
	Generator                  string   `json:"generator"`
	Corpus                     string   `json:"corpus"`
	CanonicalDigest            string   `json:"canonical_digest"`
	MaterializedDigest         string   `json:"materialized_digest"`
	ClaudeRecords              int      `json:"claude_records"`
	CodexRecords               int      `json:"codex_records"`
	TotalRecords               int      `json:"total_records"`
	EventCountPerRecord        int      `json:"event_count_per_record"`
	TotalEvents                int      `json:"total_events"`
	MessageCountPerRecord      int      `json:"message_count_per_record"`
	TotalMessages              int      `json:"total_messages"`
	FileReferencesPerRecord    int      `json:"file_references_per_record"`
	TotalFileReferences        int      `json:"total_file_references"`
	ClaudeCapabilityNames      int      `json:"claude_capability_names"`
	CodexCapabilityNames       int      `json:"codex_capability_names"`
	TotalCapabilityNames       int      `json:"total_capability_names"`
	SessionIDBytes             int      `json:"session_id_bytes"`
	TitleBytes                 int      `json:"title_bytes"`
	UserMessageBytes           int      `json:"user_message_bytes"`
	AssistantMessageBytes      int      `json:"assistant_message_bytes"`
	CapabilityNameBytes        int      `json:"capability_name_bytes"`
	FileReferenceBytes         int      `json:"file_reference_bytes"`
	WorkspaceFileBytes         int      `json:"workspace_file_bytes"`
	TimestampBase              string   `json:"timestamp_base"`
	TimestampStepSeconds       int64    `json:"timestamp_step_seconds"`
	Query                      string   `json:"query"`
	Limit                      int      `json:"limit"`
	ClaudeReference            string   `json:"claude_reference"`
	CodexReference             string   `json:"codex_reference"`
	RelativeReinstateHome      string   `json:"relative_reinstate_home"`
	RelativeClaudeConfigDir    string   `json:"relative_claude_config_dir"`
	RelativeCodexHome          string   `json:"relative_codex_home"`
	RelativeGeminiCLIHome      string   `json:"relative_gemini_cli_home"`
	RelativeProcessHome        string   `json:"relative_process_home"`
	RelativeWorkingDirectory   string   `json:"relative_working_directory"`
	WorkspaceGitBranch         string   `json:"workspace_git_branch"`
	WorkspaceGitExpectedHead   string   `json:"workspace_git_expected_head"`
	WorkspaceGitCommitTime     string   `json:"workspace_git_commit_time"`
	OpenCodePolicy             string   `json:"opencode_policy"`
	AmbientCapabilityPolicy    string   `json:"ambient_capability_policy"`
	SourceFileCount            int      `json:"source_file_count"`
	SourceByteCount            int64    `json:"source_byte_count"`
	FixtureFileCount           int      `json:"fixture_file_count"`
	CanonicalRelativeFilePaths []string `json:"canonical_relative_file_paths"`
}

type aggregateManifest struct {
	SchemaVersion   int            `json:"schema_version"`
	Generator       string         `json:"generator"`
	CanonicalDigest string         `json:"canonical_digest"`
	Normal          corpusManifest `json:"normal"`
	Large           corpusManifest `json:"large"`
}

func loadSpec() (fixtureSpec, error) {
	var spec fixtureSpec
	decoder := json.NewDecoder(bytes.NewReader(embeddedSpec))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&spec); err != nil {
		return fixtureSpec{}, fmt.Errorf("decode embedded performance corpus spec: %w", err)
	}
	if spec.SchemaVersion != 1 || spec.Generator != "phase3perf-v1" {
		return fixtureSpec{}, errors.New("unsupported performance corpus specification")
	}
	for _, corpus := range []corpusSpec{spec.Normal, spec.Large} {
		if corpus.Name == "" || corpus.ClaudeRecords <= 0 || corpus.CodexRecords <= 0 ||
			corpus.Query == "" || corpus.Limit <= 0 || corpus.AnchorIndex < 0 ||
			corpus.AnchorIndex >= corpus.ClaudeRecords || corpus.AnchorIndex >= corpus.CodexRecords {
			return fixtureSpec{}, fmt.Errorf("invalid %q corpus specification", corpus.Name)
		}
		if _, err := time.Parse(time.RFC3339, corpus.TimestampBase); err != nil {
			return fixtureSpec{}, fmt.Errorf("parse %q timestamp base: %w", corpus.Name, err)
		}
	}
	if spec.GitBranch != "main" || spec.GitAuthorName == "" || spec.GitAuthorEmail == "" ||
		spec.GitCommitMessage == "" || spec.WorkspaceFileBytes <= 0 {
		return fixtureSpec{}, errors.New("invalid controlled Git workspace specification")
	}
	if _, err := time.Parse(time.RFC3339, spec.GitCommitTimestamp); err != nil {
		return fixtureSpec{}, fmt.Errorf("parse Git commit timestamp: %w", err)
	}
	return spec, nil
}

func generateFixture(root string, spec fixtureSpec) (aggregateManifest, error) {
	if strings.TrimSpace(root) == "" || !filepath.IsAbs(root) {
		return aggregateManifest{}, errors.New("fixture root must be an absolute path")
	}
	if _, err := os.Lstat(root); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			return aggregateManifest{}, errors.New("fixture root already exists")
		}
		return aggregateManifest{}, fmt.Errorf("inspect fixture root: %w", err)
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return aggregateManifest{}, fmt.Errorf("create fixture root: %w", err)
	}

	normalFiles, normalManifest, err := buildCorpus(filepath.Join(root, "normal"), spec, spec.Normal)
	if err != nil {
		return aggregateManifest{}, err
	}
	largeFiles, largeManifest, err := buildCorpus(filepath.Join(root, "large"), spec, spec.Large)
	if err != nil {
		return aggregateManifest{}, err
	}
	allFiles := append(append([]fixtureFile{}, normalFiles...), largeFiles...)
	canonicalDigest, err := digestSpecAndFiles(spec, allFiles, true)
	if err != nil {
		return aggregateManifest{}, err
	}
	if spec.CanonicalDigest != "PENDING" && spec.CanonicalDigest != canonicalDigest {
		return aggregateManifest{}, fmt.Errorf("embedded canonical digest mismatch: got %s want %s", canonicalDigest, spec.CanonicalDigest)
	}
	normalManifest.CanonicalDigest = canonicalDigest
	largeManifest.CanonicalDigest = canonicalDigest

	for _, file := range allFiles {
		path := filepath.Join(root, filepath.FromSlash(file.RelativePath))
		if err := writeFixtureFile(path, file.Materialized, file.Mode, file.ModTime); err != nil {
			return aggregateManifest{}, err
		}
	}
	normalManifest.MaterializedDigest = digestMaterializedFiles(normalFiles)
	largeManifest.MaterializedDigest = digestMaterializedFiles(largeFiles)
	manifest := aggregateManifest{
		SchemaVersion:   1,
		Generator:       spec.Generator,
		CanonicalDigest: canonicalDigest,
		Normal:          normalManifest,
		Large:           largeManifest,
	}
	if err := writeJSON(filepath.Join(root, "normal", manifestFileName), normalManifest); err != nil {
		return aggregateManifest{}, err
	}
	if err := writeJSON(filepath.Join(root, "large", manifestFileName), largeManifest); err != nil {
		return aggregateManifest{}, err
	}
	if err := writeJSON(filepath.Join(root, manifestFileName), manifest); err != nil {
		return aggregateManifest{}, err
	}
	return manifest, nil
}

func buildCorpus(root string, spec fixtureSpec, corpus corpusSpec) ([]fixtureFile, corpusManifest, error) {
	base, err := time.Parse(time.RFC3339, corpus.TimestampBase)
	if err != nil {
		return nil, corpusManifest{}, err
	}
	rootName := corpus.Name
	workspace := filepath.Join(root, "workspace")
	workspaceCanonical := workspacePlaceholder
	files := []fixtureFile{
		directoryFixture(rootName, "reinstate-home/cache"),
		directoryFixture(rootName, "claude/projects/phase3-performance"),
		directoryFixture(rootName, "claude/skills"),
		directoryFixture(rootName, "codex/sessions"),
		directoryFixture(rootName, "gemini/tmp"),
		directoryFixture(rootName, "process-home/.agents/skills"),
		directoryFixture(rootName, "workspace"),
		directoryFixture(rootName, "tmp"),
		directoryFixture(rootName, "cold-evidence"),
	}
	workspaceContent := []byte(fixedASCII("controlled-phase3-performance-workspace", spec.WorkspaceFileBytes-1, 'w') + "\n")
	files = append(files, fixtureFile{
		RelativePath: filepath.ToSlash(filepath.Join(rootName, "workspace", "phase3-performance.txt")),
		Canonical:    workspaceContent,
		Materialized: workspaceContent,
		Mode:         0o600,
		ModTime:      base,
	})
	var sourceBytes int64

	for index := 0; index < corpus.ClaudeRecords; index++ {
		id := fixtureID(1, index)
		timestamp := base.Add(time.Duration(index*int(corpus.TimestampStepSeconds)) * time.Second)
		content, err := claudeFixture(spec, corpus, index, id, timestamp, workspaceCanonical)
		if err != nil {
			return nil, corpusManifest{}, err
		}
		rel := filepath.ToSlash(filepath.Join(rootName, "claude", "projects", "phase3-performance", id+".jsonl"))
		materialized, err := materializeWorkspace(content, workspace)
		if err != nil {
			return nil, corpusManifest{}, err
		}
		files = append(files, fixtureFile{RelativePath: rel, Canonical: content, Materialized: materialized, Mode: 0o600, ModTime: timestamp})
		sourceBytes += int64(len(materialized))
	}
	for index := 0; index < corpus.CodexRecords; index++ {
		id := fixtureID(2, index)
		timestamp := base.Add(time.Duration(index*int(corpus.TimestampStepSeconds)) * time.Second)
		content, err := codexFixture(spec, corpus, index, id, timestamp, workspaceCanonical)
		if err != nil {
			return nil, corpusManifest{}, err
		}
		date := timestamp.UTC().Format("2006/01/02")
		name := "rollout-" + timestamp.UTC().Format("2006-01-02T15-04-05") + "-" + id + ".jsonl"
		rel := filepath.ToSlash(filepath.Join(rootName, "codex", "sessions", filepath.FromSlash(date), name))
		materialized, err := materializeWorkspace(content, workspace)
		if err != nil {
			return nil, corpusManifest{}, err
		}
		files = append(files, fixtureFile{RelativePath: rel, Canonical: content, Materialized: materialized, Mode: 0o600, ModTime: timestamp})
		sourceBytes += int64(len(materialized))
	}
	for index := 0; index < corpus.ClaudeCapabilityNames; index++ {
		name := capabilityName("claude", index, spec.CapabilityNameBytes)
		content := capabilityFixture(name)
		rel := filepath.ToSlash(filepath.Join(rootName, "claude", "skills", name, "SKILL.md"))
		files = append(files, fixtureFile{RelativePath: rel, Canonical: content, Materialized: content, Mode: 0o600, ModTime: base})
	}
	for index := 0; index < corpus.CodexCapabilityNames; index++ {
		name := capabilityName("codex", index, spec.CapabilityNameBytes)
		content := capabilityFixture(name)
		rel := filepath.ToSlash(filepath.Join(rootName, "process-home", ".agents", "skills", name, "SKILL.md"))
		files = append(files, fixtureFile{RelativePath: rel, Canonical: content, Materialized: content, Mode: 0o600, ModTime: base})
	}

	paths := make([]string, 0, len(files))
	for _, file := range files {
		if file.Mode.IsRegular() {
			paths = append(paths, file.RelativePath)
		}
	}
	sort.Strings(paths)
	manifest := corpusManifest{
		SchemaVersion:              1,
		Generator:                  spec.Generator,
		Corpus:                     corpus.Name,
		ClaudeRecords:              corpus.ClaudeRecords,
		CodexRecords:               corpus.CodexRecords,
		TotalRecords:               corpus.ClaudeRecords + corpus.CodexRecords,
		EventCountPerRecord:        spec.EventCountPerRecord,
		TotalEvents:                spec.EventCountPerRecord * (corpus.ClaudeRecords + corpus.CodexRecords),
		MessageCountPerRecord:      spec.MessageCountPerRecord,
		TotalMessages:              spec.MessageCountPerRecord * (corpus.ClaudeRecords + corpus.CodexRecords),
		FileReferencesPerRecord:    spec.FileReferenceCountPerRecord,
		TotalFileReferences:        spec.FileReferenceCountPerRecord * (corpus.ClaudeRecords + corpus.CodexRecords),
		ClaudeCapabilityNames:      corpus.ClaudeCapabilityNames,
		CodexCapabilityNames:       corpus.CodexCapabilityNames,
		TotalCapabilityNames:       corpus.ClaudeCapabilityNames + corpus.CodexCapabilityNames,
		SessionIDBytes:             spec.SessionIDBytes,
		TitleBytes:                 spec.TitleBytes,
		UserMessageBytes:           spec.UserMessageBytes,
		AssistantMessageBytes:      spec.AssistantMessageBytes,
		CapabilityNameBytes:        spec.CapabilityNameBytes,
		FileReferenceBytes:         spec.FileReferenceBytes,
		WorkspaceFileBytes:         spec.WorkspaceFileBytes,
		TimestampBase:              corpus.TimestampBase,
		TimestampStepSeconds:       corpus.TimestampStepSeconds,
		Query:                      corpus.Query,
		Limit:                      corpus.Limit,
		ClaudeReference:            "claude:" + fixtureID(1, corpus.AnchorIndex),
		CodexReference:             "codex:" + fixtureID(2, corpus.AnchorIndex),
		RelativeReinstateHome:      "reinstate-home",
		RelativeClaudeConfigDir:    "claude",
		RelativeCodexHome:          "codex",
		RelativeGeminiCLIHome:      "gemini",
		RelativeProcessHome:        "process-home",
		RelativeWorkingDirectory:   "workspace",
		WorkspaceGitBranch:         spec.GitBranch,
		WorkspaceGitExpectedHead:   spec.GitExpectedHead,
		WorkspaceGitCommitTime:     spec.GitCommitTimestamp,
		OpenCodePolicy:             "FAIL_IF_RESOLVABLE_IN_CURATED_PATH",
		AmbientCapabilityPolicy:    "FAIL_IF_DISCOVERED_COUNT_OR_NAMES_DIFFER",
		SourceFileCount:            corpus.ClaudeRecords + corpus.CodexRecords,
		SourceByteCount:            sourceBytes,
		FixtureFileCount:           len(paths),
		CanonicalRelativeFilePaths: paths,
	}
	return files, manifest, nil
}

func materializeWorkspace(content []byte, workspace string) ([]byte, error) {
	encoded, err := json.Marshal(workspace)
	if err != nil || len(encoded) < 2 || encoded[0] != '"' || encoded[len(encoded)-1] != '"' {
		return nil, errors.New("encode materialized workspace path")
	}
	// The placeholder already occurs inside JSON string values. Replace it with
	// the encoded string payload, not raw filesystem bytes: Windows separators,
	// quotes, and controls must retain JSON escaping.
	replacement := encoded[1 : len(encoded)-1]
	return bytes.ReplaceAll(content, []byte(workspacePlaceholder), replacement), nil
}

func directoryFixture(rootName, relative string) fixtureFile {
	rel := filepath.ToSlash(filepath.Join(rootName, relative))
	return fixtureFile{RelativePath: rel, Mode: fs.ModeDir | 0o700}
}

func claudeFixture(spec fixtureSpec, corpus corpusSpec, index int, id string, timestamp time.Time, workspace string) ([]byte, error) {
	title := fixedASCII(fmt.Sprintf("phase3-%s-claude-%04d", corpus.Name, index), spec.TitleBytes, 't')
	user := fixedASCII(corpus.Query, spec.UserMessageBytes, 'u')
	assistant := fixedASCII("controlled-synthetic-assistant", spec.AssistantMessageBytes, 'a')
	events := []any{
		map[string]any{"type": "summary", "sessionId": id, "cwd": workspace, "title": title, "timestamp": timestamp.UTC().Format(time.RFC3339)},
		map[string]any{"type": "user", "sessionId": id, "cwd": workspace, "timestamp": timestamp.UTC().Format(time.RFC3339), "message": map[string]any{"role": "user", "content": user}},
		map[string]any{"type": "assistant", "sessionId": id, "cwd": workspace, "timestamp": timestamp.UTC().Format(time.RFC3339), "message": map[string]any{"role": "assistant", "content": assistant}},
		map[string]any{"type": "tool", "sessionId": id, "timestamp": timestamp.UTC().Format(time.RFC3339), "input": map[string]any{"paths": fixtureFileReferences()}},
	}
	return jsonLines(events)
}

func codexFixture(spec fixtureSpec, corpus corpusSpec, index int, id string, timestamp time.Time, workspace string) ([]byte, error) {
	title := fixedASCII(fmt.Sprintf("phase3-%s-codex-%04d", corpus.Name, index), spec.TitleBytes, 't')
	user := fixedASCII(corpus.Query, spec.UserMessageBytes, 'u')
	assistant := fixedASCII("controlled-synthetic-assistant", spec.AssistantMessageBytes, 'a')
	argumentBytes, err := json.Marshal(map[string]any{"paths": fixtureFileReferences()})
	if err != nil {
		return nil, err
	}
	events := []any{
		map[string]any{"type": "session_meta", "timestamp": timestamp.UTC().Format(time.RFC3339), "payload": map[string]any{"type": "session_meta", "id": id, "cwd": workspace, "title": title, "timestamp": timestamp.UTC().Format(time.RFC3339)}},
		map[string]any{"type": "event_msg", "timestamp": timestamp.UTC().Format(time.RFC3339), "payload": map[string]any{"type": "user_message", "message": user}},
		map[string]any{"type": "event_msg", "timestamp": timestamp.UTC().Format(time.RFC3339), "payload": map[string]any{"type": "agent_message", "message": assistant}},
		map[string]any{"type": "response_item", "timestamp": timestamp.UTC().Format(time.RFC3339), "payload": map[string]any{"type": "function_call", "arguments": string(argumentBytes)}},
	}
	return jsonLines(events)
}

func jsonLines(values []any) ([]byte, error) {
	var output bytes.Buffer
	for _, value := range values {
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		output.Write(encoded)
		output.WriteByte('\n')
	}
	return output.Bytes(), nil
}

func fixtureFileReferences() []string {
	return []string{"src/perf-aa.go", "src/perf-bb.go"}
}

func fixtureID(agent int, index int) string {
	return fmt.Sprintf("%08d-0000-4000-8000-%012d", agent, index+1)
}

func capabilityName(agent string, index, width int) string {
	return fixedASCII(fmt.Sprintf("phase3-%s-cap-%03d", agent, index), width, 'c')
}

func capabilityFixture(name string) []byte {
	return []byte(fmt.Sprintf("---\nname: %s\ndescription: Deterministic synthetic performance capability.\n---\n\n# Controlled performance capability\n", name))
}

func fixedASCII(prefix string, width int, fill byte) string {
	if len(prefix) > width {
		panic("fixture field exceeds fixed width")
	}
	return prefix + strings.Repeat(string(fill), width-len(prefix))
}

func digestSpecAndFiles(spec fixtureSpec, files []fixtureFile, canonical bool) (string, error) {
	spec.CanonicalDigest = ""
	encoded, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	writeDigestEntry(hash, "spec.json", encoded)
	sorted := append([]fixtureFile(nil), files...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].RelativePath < sorted[j].RelativePath })
	for _, file := range sorted {
		if !file.Mode.IsRegular() {
			continue
		}
		content := file.Materialized
		if canonical {
			content = file.Canonical
		}
		writeDigestEntry(hash, file.RelativePath, content)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func digestMaterializedFiles(files []fixtureFile) string {
	hash := sha256.New()
	sorted := append([]fixtureFile(nil), files...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].RelativePath < sorted[j].RelativePath })
	for _, file := range sorted {
		if file.Mode.IsRegular() {
			writeDigestEntry(hash, file.RelativePath, file.Materialized)
		}
	}
	return hex.EncodeToString(hash.Sum(nil))
}

type digestWriter interface {
	Write([]byte) (int, error)
}

func writeDigestEntry(hash digestWriter, path string, content []byte) {
	_, _ = hash.Write([]byte(path))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(strconv.Itoa(len(content))))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write(content)
	_, _ = hash.Write([]byte{0})
}

func writeFixtureFile(path string, content []byte, mode fs.FileMode, modTime time.Time) error {
	if mode.IsDir() {
		if err := os.MkdirAll(path, mode.Perm()); err != nil {
			return fmt.Errorf("create fixture directory: %w", err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create fixture parent: %w", err)
	}
	if err := os.WriteFile(path, content, mode.Perm()); err != nil {
		return fmt.Errorf("write fixture file: %w", err)
	}
	if !modTime.IsZero() {
		if err := os.Chtimes(path, modTime, modTime); err != nil {
			return fmt.Errorf("set fixture modification time: %w", err)
		}
	}
	return nil
}

func writeJSON(path string, value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}
