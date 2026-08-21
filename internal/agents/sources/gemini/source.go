// Package gemini ports Gemini CLI's F1 index source onto the shared hometree scanner.
package gemini

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/agents/scan/hometree"
	"github.com/HarjjotSinghh/reinstate/internal/agents/sources"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

const (
	sessionGlob     = "tmp/*/chats/session-*.json*"
	readOnlyReason  = "Gemini CLI sessions are read-only in Phase 2"
	errSubagentText = "gemini subagent session"
)

var errSubagent = errors.New(errSubagentText)

// Excluded skips Gemini subagent trees.
var Excluded = []string{"subagents"}

// maxProjectsBytes bounds the projects.json read. The file is a flat path
// index; anything larger is not it.
const maxProjectsBytes = 4 << 20

// Source discovers Gemini CLI chat records through hometree.
type Source struct {
	env agents.Env
}

// New constructs a Gemini index source from a catalog environment.
func New(env agents.Env) (sessionindex.Source, error) {
	return &Source{env: env}, nil
}

// Name returns the stable agent key.
func (s *Source) Name() string { return sessionindex.AgentGemini }

// Scan walks Gemini chat files and maps each to a record.
func (s *Source) Scan(ctx context.Context) (sessionindex.ScanResult, error) {
	root, files, err := hometree.Discover(ctx, s.config())
	if err != nil {
		return sessionindex.ScanResult{}, err
	}
	if root == "" {
		return sessionindex.ScanResult{}, nil
	}
	tmpRoot := filepath.Join(root, "tmp")
	workspaces := projectWorkspaces(root)
	var result sessionindex.ScanResult
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return sessionindex.ScanResult{}, err
		}
		if !geminiSessionFile(file.Path) {
			continue
		}
		record, warnings, parseErr := parseSession(file.Path, tmpRoot, workspaces)
		if errors.Is(parseErr, errSubagent) {
			continue
		}
		if parseErr != nil {
			result.Warnings = append(result.Warnings, sessionindex.Warning{
				Agent:   sessionindex.AgentGemini,
				Source:  file.Path,
				Code:    "session_read_failed",
				Message: "Gemini session could not be read; other sessions remain available",
			})
			continue
		}
		result.Records = append(result.Records, record)
		result.Warnings = append(result.Warnings, warnings...)
	}
	sources.SortRecordsBySourcePath(result.Records)
	return result, nil
}

// Fingerprint summarises the source without opening any file, so an
// unchanged refresh can skip parsing entirely.
func (s *Source) Fingerprint(ctx context.Context) (string, bool, error) {
	return hometree.Fingerprint(ctx, s.config())
}

func (s *Source) config() hometree.Config {
	cfg := hometree.Config{
		Explicit:    s.env.FixtureRoot,
		RootEnv:     "GEMINI_CLI_HOME",
		LookupEnv:   s.env.LookupEnv,
		Marker:      "tmp",
		SessionGlob: sessionGlob,
		Excluded:    Excluded,
	}
	if home, err := s.env.HomeDir(); err == nil {
		cfg.Candidates = []string{home.Join(".gemini")}
	}
	return cfg
}

func geminiSessionFile(path string) bool {
	name := filepath.Base(path)
	extension := strings.ToLower(filepath.Ext(name))
	return strings.HasPrefix(strings.ToLower(name), "session-") &&
		(extension == ".json" || extension == ".jsonl") &&
		strings.EqualFold(filepath.Base(filepath.Dir(path)), "chats")
}

type message struct {
	id     string
	kind   string
	prompt string
	files  []string
}

type state struct {
	id          string
	projectHash string
	title       string
	workspace   string
	kind        string
	latest      int64
	messages    []message
}

func parseSession(path, tmpRoot string, workspaces map[string]string) (sessionindex.Record, []sessionindex.Warning, error) {
	info, err := os.Stat(path)
	if err != nil {
		return sessionindex.Record{}, nil, err
	}
	st := state{
		id:          strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
		projectHash: projectHash(path, tmpRoot),
	}
	var warnings []sessionindex.Warning
	if strings.EqualFold(filepath.Ext(path), ".jsonl") {
		warnings, err = parseJSONL(path, &st)
	} else {
		err = parseLegacyJSON(path, &st)
	}
	if err != nil {
		return sessionindex.Record{}, nil, err
	}
	if strings.EqualFold(st.kind, "subagent") {
		return sessionindex.Record{}, nil, errSubagent
	}

	var prompts sources.BoundedText
	firstPrompt := ""
	fileSet := make(map[string]struct{})
	messageCount := 0
	for _, item := range st.messages {
		if item.kind == "user" || item.kind == "gemini" {
			messageCount++
		}
		if item.kind == "user" {
			prompts.Add(item.prompt)
			if firstPrompt == "" {
				firstPrompt = sessionindex.SafePreview(item.prompt)
			}
		}
		for _, file := range item.files {
			sources.AddFilePath(file, fileSet)
		}
	}
	if st.latest == 0 {
		st.latest = info.ModTime().Unix()
	}
	// A chat record often carries only projectHash. Gemini derives that hash
	// from the absolute project path, and projects.json lists those paths, so
	// the workspace is recoverable instead of surfacing a bare digest as the
	// project name.
	if st.workspace == "" && st.projectHash != "" {
		if resolved, ok := workspaces[strings.ToLower(st.projectHash)]; ok {
			st.workspace = resolved
		}
	}
	project := st.projectHash
	if st.workspace != "" {
		project = sources.PortableBase(st.workspace)
	}
	if project == "" {
		project = "unknown"
	}
	preview := firstPrompt
	title := sessionindex.SafePreview(st.title)
	if title == "" {
		title = st.id
	}
	files := sources.NormalizedFileMap(fileSet)
	for index := range warnings {
		warnings[index].Agent = sessionindex.AgentGemini
		warnings[index].SessionID = st.id
		warnings[index].Source = path
	}

	return sessionindex.Record{
		Key:            sessionindex.CompositeReference(sessionindex.AgentGemini, st.id),
		ID:             st.id,
		Agent:          sessionindex.AgentGemini,
		Title:          title,
		Project:        project,
		Workspace:      st.workspace,
		UpdatedAt:      time.Unix(st.latest, 0).UTC(),
		SizeBytes:      info.Size(),
		MessageCount:   messageCount,
		PromptPreview:  preview,
		Files:          files,
		ReadOnlyReason: readOnlyReason,
		SourcePath:     path,
		SourceModTime:  info.ModTime().UnixNano(),
		SourceSize:     info.Size(),
		SearchText:     sessionindex.BuildSearchText(st.id, title, project, st.workspace, prompts.String(), strings.Join(files, " ")),
	}, warnings, nil
}

func parseJSONL(path string, st *state) ([]sessionindex.Warning, error) {
	return hometree.ReadJSONL(path, func(_ int, line []byte) error {
		var event map[string]any
		if json.Unmarshal(line, &event) != nil {
			return nil
		}
		if update, ok := event["$set"].(map[string]any); ok {
			applyMetadata(update, st)
			return nil
		}
		if rewindID := sources.FirstString(event, "$rewindTo"); rewindID != "" {
			for index := len(st.messages) - 1; index >= 0; index-- {
				if st.messages[index].id == rewindID {
					st.messages = st.messages[:index+1]
					break
				}
			}
			return nil
		}
		if eventType := strings.ToLower(sources.FirstString(event, "type")); eventType != "" {
			st.messages = append(st.messages, messageFromMap(event))
			st.latest = sources.MaxInt64(st.latest, sources.EventTimestamp(event))
			return nil
		}
		applyMetadata(event, st)
		return nil
	})
}

func parseLegacyJSON(path string, st *state) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	limited := io.LimitReader(file, int64(sessionindex.MaxJSONLineBytes)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return err
	}
	if len(data) > sessionindex.MaxJSONLineBytes {
		return fmt.Errorf("gemini session exceeds %d-byte read limit", sessionindex.MaxJSONLineBytes)
	}
	var conversation map[string]any
	if err := json.Unmarshal(data, &conversation); err != nil {
		return err
	}
	applyMetadata(conversation, st)
	if messages, ok := conversation["messages"].([]any); ok {
		for _, raw := range messages {
			if item, ok := raw.(map[string]any); ok {
				st.messages = append(st.messages, messageFromMap(item))
			}
		}
	}
	return nil
}

func applyMetadata(values map[string]any, st *state) {
	if value := sources.FirstString(values, "sessionId", "session_id"); value != "" {
		st.id = value
	}
	if value := sources.FirstString(values, "projectHash", "project_hash"); value != "" {
		st.projectHash = value
	}
	if value := sources.FirstString(values, "summary", "title", "name"); value != "" {
		st.title = value
	}
	if value := sources.FirstString(values, "kind"); value != "" {
		st.kind = value
	}
	if directories, ok := values["directories"].([]any); ok {
		for _, raw := range directories {
			if directory, ok := raw.(string); ok && directory != "" {
				st.workspace = directory
				break
			}
		}
	}
	if value := sources.FirstString(values, "cwd", "directory", "workspace"); value != "" {
		st.workspace = value
	}
	st.latest = sources.MaxInt64(st.latest, sources.EventTimestamp(values))
}

func messageFromMap(values map[string]any) message {
	item := message{
		id:   sources.FirstString(values, "id"),
		kind: strings.ToLower(sources.FirstString(values, "type", "role")),
	}
	if item.kind == "user" {
		item.prompt = sources.ExtractTextContent(values["content"])
	}
	if item.kind == "gemini" || item.kind == "assistant" || item.kind == "model" {
		if calls, ok := values["toolCalls"].([]any); ok {
			fileSet := make(map[string]struct{})
			for _, raw := range calls {
				if call, ok := raw.(map[string]any); ok {
					collectGeminiArgs(call["args"], fileSet)
				}
			}
			item.files = sources.NormalizedFileMap(fileSet)
		}
		item.kind = "gemini"
	}
	return item
}

func collectGeminiArgs(value any, files map[string]struct{}) {
	// Reuse the same structured-file walk as the original source by wrapping
	// args in a synthetic event so CollectToolFiles sees them as input.
	sources.CollectToolFiles(map[string]any{"input": value}, files)
}

// maxProjectRootBytes bounds a .project_root read. The file holds one
// absolute path and nothing else.
const maxProjectRootBytes = 64 << 10

// maxProjectRootDirs bounds how many session directories are consulted for a
// .project_root, so a pathological tree cannot turn a scan into a walk.
const maxProjectRootDirs = 4096

// projectWorkspaces maps a Gemini project hash to the absolute path it was
// derived from. Gemini records only the hash on each chat and keeps the paths
// elsewhere, so the two are joined here. A missing or malformed file yields no
// mappings rather than an error: this only enriches a record.
//
// Two sources are consulted. projects.json is the flat index Gemini has always
// kept. Current Gemini also writes tmp/<name>/.project_root beside the chats
// themselves, which is authoritative for that directory and survives a pruned
// projects.json.
func projectWorkspaces(root string) map[string]string {
	out := make(map[string]string)
	// One directory listing serves every workspace that shares a parent.
	listing := make(map[string][]string)
	add := func(workspace string) {
		workspace = strings.TrimSpace(workspace)
		if workspace == "" {
			return
		}
		register(out, workspace)
		// Gemini hashes the path as the process saw it. On a case-insensitive
		// filesystem it records a lower-cased path in projects.json and
		// .project_root but hashes the real on-disk case, so the verbatim
		// digest never matches. Both corrections are registered alongside the
		// verbatim spelling, so a path that needs none is unaffected.
		if resolved, err := filepath.EvalSymlinks(workspace); err == nil && resolved != workspace {
			register(out, resolved)
		}
		if cased, ok := trueCasePath(workspace, listing); ok && cased != workspace {
			register(out, cased)
		}
	}
	for _, workspace := range declaredProjects(root) {
		add(workspace)
	}
	for _, workspace := range recordedProjectRoots(root) {
		add(workspace)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// maxPathComponents bounds the case walk so a pathological path cannot turn
// one lookup into an unbounded number of directory reads.
const maxPathComponents = 64

// trueCasePath rewrites a path into the spelling the filesystem actually
// holds, one component at a time. On a case-insensitive filesystem a path that
// exists may be spelled differently from how it is stored, and Gemini hashes
// the stored spelling. Reports false when any component cannot be resolved,
// which includes every case-sensitive filesystem where the question does not
// arise.
func trueCasePath(path string, listing map[string][]string) (string, bool) {
	if !filepath.IsAbs(path) {
		return "", false
	}
	volume := filepath.VolumeName(path)
	rest := strings.TrimPrefix(path, volume)
	parts := make([]string, 0, 8)
	for _, part := range strings.Split(filepath.ToSlash(rest), "/") {
		if part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 || len(parts) > maxPathComponents {
		return "", false
	}
	// A drive letter is part of what Gemini hashes, and it is recorded in
	// upper case by every Windows API that reports a real path.
	current := strings.ToUpper(volume) + string(filepath.Separator)
	for _, part := range parts {
		names, cached := listing[current]
		if !cached {
			entries, err := os.ReadDir(current)
			if err != nil {
				return "", false
			}
			names = make([]string, 0, len(entries))
			for _, entry := range entries {
				names = append(names, entry.Name())
			}
			listing[current] = names
		}
		match := ""
		for _, name := range names {
			if name == part {
				match = name
				break
			}
			if match == "" && strings.EqualFold(name, part) {
				match = name
			}
		}
		if match == "" {
			return "", false
		}
		current = filepath.Join(current, match)
	}
	return current, true
}

// register indexes one workspace under its Gemini project hash. The first
// spelling seen wins, so a verbatim match is never displaced by a derived one.
func register(out map[string]string, workspace string) {
	sum := sha256.Sum256([]byte(workspace))
	key := hex.EncodeToString(sum[:])
	if _, exists := out[key]; !exists {
		out[key] = workspace
	}
}

// declaredProjects reads the flat path index Gemini keeps at the root.
func declaredProjects(root string) []string {
	path := filepath.Join(root, "projects.json")
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxProjectsBytes {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var doc struct {
		Projects map[string]json.RawMessage `json:"projects"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil
	}
	out := make([]string, 0, len(doc.Projects))
	for workspace := range doc.Projects {
		out = append(out, workspace)
	}
	return out
}

// recordedProjectRoots reads tmp/<name>/.project_root, which current Gemini
// writes beside each session directory.
func recordedProjectRoots(root string) []string {
	entries, err := os.ReadDir(filepath.Join(root, "tmp"))
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(entries))
	for index, entry := range entries {
		if index >= maxProjectRootDirs {
			break
		}
		if !entry.IsDir() {
			continue
		}
		marker := filepath.Join(root, "tmp", entry.Name(), ".project_root")
		info, err := os.Stat(marker)
		if err != nil || !info.Mode().IsRegular() || info.Size() > maxProjectRootBytes {
			continue
		}
		data, err := os.ReadFile(marker)
		if err != nil {
			continue
		}
		out = append(out, strings.TrimSpace(string(data)))
	}
	return out
}

func projectHash(path, tmpRoot string) string {
	relative, err := filepath.Rel(tmpRoot, path)
	if err != nil {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(relative), "/")
	if len(parts) < 3 {
		return ""
	}
	return parts[0]
}
