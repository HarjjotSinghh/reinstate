// Package codex implements the OpenAI Codex CLI session adapter.
package codex

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/pathmap"
)

// Adapter implements adapter.Adapter for Codex.
type Adapter struct {
	Root        string
	Home        string
	Projects    map[string]string
	ForceCompat adapter.Compatibility
}

func (a *Adapter) Name() string { return "codex" }

func (a *Adapter) Exclusions() []adapter.Exclusion {
	return []adapter.Exclusion{
		{Pattern: "**/auth.json", Reason: "credentials"},
		{Pattern: "**/.codex/auth.json", Reason: "credentials"},
		{Pattern: "**/.env", Reason: "secrets"},
		{Pattern: "**/cache/**", Reason: "regenerable"},
	}
}

func (a *Adapter) Detect(ctx context.Context) (adapter.Install, adapter.Compatibility, error) {
	_ = ctx
	root := a.Root
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return adapter.Install{}, adapter.CompatibilityNotInstalled, err
		}
		for _, c := range []string{filepath.Join(home, ".codex"), filepath.Join(home, ".config", "codex")} {
			if st, err := os.Stat(c); err == nil && st.IsDir() {
				root = c
				break
			}
		}
	}
	if root == "" {
		return adapter.Install{Agent: "codex"}, adapter.CompatibilityNotInstalled, nil
	}
	inst := adapter.Install{Agent: "codex", Root: root, Layout: "sessions-rollout-jsonl", Version: "unknown"}
	if a.ForceCompat != "" {
		return inst, a.ForceCompat, nil
	}
	return inst, adapter.CompatibilitySupported, nil
}

func (a *Adapter) Discover(ctx context.Context, opts adapter.DiscoverOptions) ([]adapter.Session, error) {
	_ = ctx
	inst, compat, err := a.Detect(context.Background())
	if err != nil {
		return nil, err
	}
	if compat == adapter.CompatibilityNotInstalled {
		return nil, nil
	}
	var sessions []adapter.Session
	sessionsDir := filepath.Join(inst.Root, "sessions")
	_ = filepath.Walk(sessionsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".jsonl") {
			return nil
		}
		id := strings.TrimSuffix(info.Name(), ".jsonl")
		projectID := "unknown"
		// try first line session_meta cwd
		if b, err := os.ReadFile(path); err == nil {
			line, _, _ := strings.Cut(string(b), "\n")
			var meta map[string]any
			if json.Unmarshal([]byte(line), &meta) == nil {
				if cwd, ok := meta["cwd"].(string); ok && cwd != "" {
					projectID = cwd
				}
				if id2, ok := meta["id"].(string); ok && id2 != "" {
					id = id2
				}
			}
		}
		if opts.ProjectID != "" && projectID != opts.ProjectID {
			return nil
		}
		sessions = append(sessions, adapter.Session{
			ID:        id,
			Agent:     "codex",
			ProjectID: projectID,
			Title:     info.Name(),
			UpdatedAt: info.ModTime().Unix(),
			SizeBytes: info.Size(),
			Path:      path,
		})
		return nil
	})
	return sessions, nil
}

func (a *Adapter) mapper() pathmap.Mapper {
	return pathmap.Mapper{Home: a.Home, Projects: a.Projects}
}

func (a *Adapter) PlanExport(ctx context.Context, s adapter.Session, opts adapter.ExportOptions) (adapter.ExportPlan, error) {
	_ = ctx
	_ = opts
	return adapter.ExportPlan{Session: s, Files: []string{s.Path}}, nil
}

func (a *Adapter) Export(ctx context.Context, plan adapter.ExportPlan, w io.Writer) error {
	_ = ctx
	tw := tar.NewWriter(w)
	defer tw.Close()
	for _, f := range plan.Files {
		b, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		transformed := transformJSONL(b, a.mapper().Normalize)
		hdr := &tar.Header{Name: filepath.Base(f), Mode: 0o600, Size: int64(len(transformed)), ModTime: time.Now()}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(transformed); err != nil {
			return err
		}
	}
	return nil
}

func (a *Adapter) PlanRestore(ctx context.Context, snap adapter.Snapshot, opts adapter.RestoreOptions) (adapter.RestorePlan, error) {
	_ = ctx
	inst, compat, err := a.Detect(context.Background())
	if err != nil {
		return adapter.RestorePlan{}, err
	}
	if !adapter.CanRestore(compat, opts.CompatibilityOK) {
		return adapter.RestorePlan{Refuse: fmt.Sprintf("compatibility %s refuses restore", compat)}, nil
	}
	dest := filepath.Join(inst.Root, "sessions", snap.SessionID+".jsonl")
	return adapter.RestorePlan{
		Session: adapter.Session{ID: snap.SessionID, Agent: "codex", ProjectID: snap.ProjectID, Path: dest},
		Files:   []string{dest},
	}, nil
}

func (a *Adapter) Restore(ctx context.Context, plan adapter.RestorePlan, r io.Reader) error {
	_ = ctx
	if plan.Refuse != "" {
		return fmt.Errorf("%s", plan.Refuse)
	}
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		b, err := io.ReadAll(tr)
		if err != nil {
			return err
		}
		b = transformJSONL(b, a.mapper().Denormalize)
		dest := plan.Session.Path
		if dest == "" {
			dest = hdr.Name
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(dest, b, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func transformJSONL(data []byte, mapPath func(string) string) []byte {
	lines := bytes.Split(data, []byte("\n"))
	for i, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var v any
		if err := json.Unmarshal(line, &v); err != nil {
			continue
		}
		rewrite(v, mapPath)
		if out, err := json.Marshal(v); err == nil {
			lines[i] = out
		}
	}
	return bytes.Join(lines, []byte("\n"))
}

func rewrite(v any, mapPath func(string) string) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if s, ok := child.(string); ok {
				if k == "cwd" || k == "path" || k == "workdir" || isAbs(s) {
					if isAbs(s) || k == "cwd" || k == "path" || k == "workdir" {
						if isAbs(s) {
							t[k] = mapPath(s)
						}
					}
				}
			} else {
				rewrite(child, mapPath)
			}
		}
	case []any:
		for _, c := range t {
			rewrite(c, mapPath)
		}
	}
}

func isAbs(s string) bool {
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, "${") {
		return true
	}
	return len(s) > 2 && s[1] == ':' && (s[2] == '\\' || s[2] == '/')
}
