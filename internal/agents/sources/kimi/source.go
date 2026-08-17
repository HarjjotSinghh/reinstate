// Package kimi discovers Kimi Code CLI sessions on the shared hometree scanner.
//
// Layout, from the 2026-08-16 macOS probe
// (docs/testing/results/agent-probes/2026-08-16-macos-kimi.json):
//
//	~/.kimi-code/sessions/wd_<slug>_<12-hex>/session_<uuid>/
//	  state.json                 id, title, cwd, createdAt, updatedAt, lastPrompt, version
//	  agents/main/wire.jsonl     append-only event log, first record type "metadata"
//
// The vendor's global session_index.jsonl is deliberately *not* used. It would
// be the cheaper discovery path, but nothing yet shows what happens to it when
// a session directory is removed by hand, and an index that outlives its
// sessions would list threads the user cannot open. The directory walk is the
// on-disk truth; the index becomes an optimisation once its staleness
// behaviour is observed.
package kimi

import (
	"context"
	"encoding/json"
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

// SessionGlob matches one session's metadata file below the root.
const SessionGlob = "sessions/**/state.json"

// stateSchemaVersion is the only state.json version observed on a device.
// Anything else fails closed: a changed schema is exactly when a tolerant
// parser starts inventing records.
const stateSchemaVersion = 2

// wireProtocolMajor is the accepted major of wire.jsonl's metadata record.
// The probe observed "1.5".
const wireProtocolMajor = "1"

// Excluded keeps the walk out of subagent trees, caches, and credentials.
// "agents" matters most: a subagent directory carries its own state.json, and
// without this the glob would report subagents as top-level sessions.
var Excluded = []string{
	"agents",
	"subagents",
	"credentials",
	"mcp-oauth",
	"cache",
	"user-history",
	"workspace-trust",
	"logs",
	"updates",
	"telemetry",
	"skills",
}

// Source discovers Kimi Code CLI sessions through hometree.
type Source struct {
	env agents.Env
}

// New constructs a Kimi index source from a catalog environment.
func New(env agents.Env) (sessionindex.Source, error) {
	return &Source{env: env}, nil
}

// Name returns the stable agent key.
func (s *Source) Name() string { return sessionindex.AgentKimi }

// Scan maps every readable session directory to one record.
func (s *Source) Scan(ctx context.Context) (sessionindex.ScanResult, error) {
	root, files, err := hometree.Discover(ctx, s.config())
	if err != nil {
		return sessionindex.ScanResult{}, err
	}
	if root == "" {
		return sessionindex.ScanResult{}, nil
	}
	var result sessionindex.ScanResult
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return sessionindex.ScanResult{}, err
		}
		sessionDir := filepath.Dir(file.Path)
		record, parseErr := parseSession(sessionDir)
		if parseErr != nil {
			result.Warnings = append(result.Warnings, sessionindex.Warning{
				Agent:   sessionindex.AgentKimi,
				Source:  sessionDir,
				Code:    "session_read_failed",
				Message: "Kimi session could not be read; other sessions remain available",
			})
			continue
		}
		result.Records = append(result.Records, record)
	}
	sources.SortRecordsBySourcePath(result.Records)
	return result, nil
}

func (s *Source) config() hometree.Config {
	cfg := hometree.Config{
		Explicit:    s.env.FixtureRoot,
		RootEnv:     "KIMI_CODE_HOME",
		LookupEnv:   s.env.LookupEnv,
		Marker:      "sessions",
		SessionGlob: SessionGlob,
		Excluded:    Excluded,
	}
	if home, err := s.env.HomeDir(); err == nil {
		// ~/.kimi-code is the root the macOS probe resolved. ~/.kimi is the
		// conflicting mirror's claim; it did not exist on that device and is
		// kept as a second candidate rather than dropped.
		cfg.Candidates = []string{home.Join(".kimi-code"), home.Join(".kimi")}
	}
	return cfg
}

type state struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	CWD        string `json:"cwd"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
	LastPrompt string `json:"lastPrompt"`
	Archived   bool   `json:"archived"`
	Version    *int   `json:"version"`
}

func parseSession(sessionDir string) (sessionindex.Record, error) {
	item, err := readState(filepath.Join(sessionDir, "state.json"))
	if err != nil {
		return sessionindex.Record{}, err
	}
	if item.Version == nil || *item.Version != stateSchemaVersion {
		return sessionindex.Record{}, fmt.Errorf("unsupported state.json version %v", item.Version)
	}

	id := strings.TrimSpace(item.ID)
	if id == "" {
		id = filepath.Base(sessionDir)
	}
	workspace := strings.TrimSpace(item.CWD)
	project := "unknown"
	if workspace != "" {
		project = sources.PortableBase(workspace)
	}

	authorityPath, authorityInfo, err := authorityFile(sessionDir)
	if err != nil {
		return sessionindex.Record{}, err
	}

	transcript, err := readWire(authorityPath)
	if err != nil {
		return sessionindex.Record{}, err
	}

	updatedAt := parseTime(item.UpdatedAt)
	if updatedAt.IsZero() {
		updatedAt = parseTime(item.CreatedAt)
	}
	if updatedAt.IsZero() {
		updatedAt = authorityInfo.ModTime().UTC()
	}

	title := sessionindex.SafePreview(item.Title)
	if title == "" {
		title = id
	}
	preview := transcript.firstPrompt
	if preview == "" {
		preview = sessionindex.SafePreview(item.LastPrompt)
	}

	return sessionindex.Record{
		Key:            sessionindex.CompositeReference(sessionindex.AgentKimi, id),
		ID:             id,
		Agent:          sessionindex.AgentKimi,
		Title:          title,
		Project:        project,
		Workspace:      workspace,
		UpdatedAt:      updatedAt,
		SizeBytes:      authorityInfo.Size(),
		MessageCount:   transcript.messages,
		PromptPreview:  preview,
		Files:          transcript.files,
		CanResume:      false,
		CanFork:        false,
		ReadOnlyReason: sessionindex.KimiReadOnlyReason,
		SourcePath:     sessionDir,
		SourceModTime:  authorityInfo.ModTime().UnixNano(),
		SourceSize:     authorityInfo.Size(),
		SearchText: sessionindex.BuildSearchText(
			id, title, project, workspace, transcript.prompts.String(), strings.Join(transcript.files, " "),
		),
	}, nil
}

func readState(path string) (state, error) {
	file, err := os.Open(path)
	if err != nil {
		return state{}, err
	}
	defer func() { _ = file.Close() }()
	limited := io.LimitReader(file, int64(sessionindex.MaxJSONLineBytes)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return state{}, err
	}
	if len(data) > sessionindex.MaxJSONLineBytes {
		return state{}, fmt.Errorf("kimi state.json exceeds %d-byte read limit", sessionindex.MaxJSONLineBytes)
	}
	var item state
	if err := json.Unmarshal(data, &item); err != nil {
		return state{}, err
	}
	return item, nil
}

// authorityFile is the file whose size and mtime represent the session. The
// wire log is preferred because state.json is rewritten on metadata-only
// changes such as a rename.
func authorityFile(sessionDir string) (string, os.FileInfo, error) {
	wire := filepath.Join(sessionDir, "agents", "main", "wire.jsonl")
	if info, err := os.Stat(wire); err == nil && !info.IsDir() {
		return wire, info, nil
	}
	statePath := filepath.Join(sessionDir, "state.json")
	info, err := os.Stat(statePath)
	if err != nil {
		return "", nil, err
	}
	return statePath, info, nil
}

type wireContent struct {
	prompts     sources.BoundedText
	firstPrompt string
	messages    int
	files       []string
}

// readWire walks the append-only event log. A wire log that is present but
// announces an unknown protocol is a layout change, so it fails closed rather
// than being parsed on the assumption that the records still mean what they
// used to.
func readWire(path string) (wireContent, error) {
	var out wireContent
	if !strings.HasSuffix(path, "wire.jsonl") {
		return out, nil
	}
	fileSet := make(map[string]struct{})
	var protocolErr error
	first := true

	_, _ = hometree.ReadJSONL(path, func(_ int, line []byte) error {
		var item map[string]any
		if json.Unmarshal(line, &item) != nil {
			return nil
		}
		kind := sources.FirstString(item, "type")
		if first {
			first = false
			if kind == "metadata" {
				version := sources.FirstString(item, "protocol_version")
				if major, _, _ := strings.Cut(version, "."); major != wireProtocolMajor {
					protocolErr = fmt.Errorf("unsupported wire protocol_version %q", version)
					return protocolErr
				}
			}
		}
		switch kind {
		case "turn.prompt":
			out.messages++
			text := sources.ExtractTextContent(item["input"])
			out.prompts.Add(text)
			if out.firstPrompt == "" {
				out.firstPrompt = sessionindex.SafePreview(text)
			}
		case "context.append_message":
			message, ok := item["message"].(map[string]any)
			if !ok {
				return nil
			}
			// turn.prompt already counted the user side of the turn.
			if !strings.EqualFold(sources.FirstString(message, "role"), "user") {
				out.messages++
			}
			// The shared collector reads event["message"]["content"], so it
			// takes the wrapping record rather than the message itself.
			sources.CollectToolFiles(item, fileSet)
			if calls, ok := message["toolCalls"].([]any); ok {
				for _, raw := range calls {
					if call, ok := raw.(map[string]any); ok {
						sources.CollectToolFiles(call, fileSet)
					}
				}
			}
		}
		return nil
	})
	if protocolErr != nil {
		return wireContent{}, protocolErr
	}
	out.files = sources.NormalizedFileMap(fileSet)
	return out, nil
}

func parseTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.UTC()
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC()
	}
	return time.Time{}
}
