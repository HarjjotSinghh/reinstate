// Package claude implements the Claude Code session adapter.
package claude

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

// Adapter implements adapter.Adapter for Claude Code.
type Adapter struct {
	// Root overrides detection root (tests/fixtures).
	Root string
	// Home for path mapping.
	Home string
	// Projects for pathmap.
	Projects map[string]string
	// ForceCompat forces a compatibility state (tests).
	ForceCompat adapter.Compatibility
}

func (a *Adapter) Name() string { return "claude" }

func (a *Adapter) Exclusions() []adapter.Exclusion {
	return []adapter.Exclusion{
		{Pattern: "**/auth.json", Reason: "credentials"},
		{Pattern: "**/.credentials.json", Reason: "credentials"},
		{Pattern: "**/credentials.json", Reason: "credentials"},
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
		for _, c := range []string{filepath.Join(home, ".claude"), filepath.Join(home, ".config", "claude")} {
			if st, err := os.Stat(c); err == nil && st.IsDir() {
				root = c
				break
			}
		}
	}
	if root == "" {
		return adapter.Install{Agent: "claude"}, adapter.CompatibilityNotInstalled, nil
	}
	inst := adapter.Install{Agent: "claude", Root: root, Layout: "projects-jsonl", Version: "unknown"}
	// version file if present
	if b, err := os.ReadFile(filepath.Join(root, "version")); err == nil {
		inst.Version = strings.TrimSpace(string(b))
	}
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
	// layout: root/projects/<project>/session-*.jsonl or recursive *.jsonl under projects
	projects := filepath.Join(inst.Root, "projects")
	_ = filepath.Walk(projects, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".jsonl") {
			return nil
		}
		// skip excluded names
		if info.Name() == "auth.json" {
			return nil
		}
		rel, _ := filepath.Rel(projects, path)
		proj := filepath.Dir(rel)
		if proj == "." {
			proj = "unknown"
		}
		if opts.ProjectID != "" && proj != opts.ProjectID {
			return nil
		}
		sessions = append(sessions, adapter.Session{
			ID:        strings.TrimSuffix(info.Name(), ".jsonl"),
			Agent:     "claude",
			ProjectID: proj,
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
	// tar of transformed files
	tw := tar.NewWriter(w)
	defer tw.Close()
	for _, f := range plan.Files {
		b, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		// schema-aware path rewrite on JSON string values that look like absolute paths
		transformed := transformJSONL(b, a.mapper().Normalize)
		hdr := &tar.Header{
			Name:    filepath.Base(f),
			Mode:    0o600,
			Size:    int64(len(transformed)),
			ModTime: time.Now(),
		}
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
	dest := filepath.Join(inst.Root, "projects", snap.ProjectID, snap.SessionID+".jsonl")
	return adapter.RestorePlan{
		Session: adapter.Session{ID: snap.SessionID, Agent: "claude", ProjectID: snap.ProjectID, Path: dest},
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
			// raw payload without tar
			b, rerr := io.ReadAll(io.MultiReader(bytes.NewReader(nil), r))
			_ = b
			_ = rerr
			return err
		}
		b, err := io.ReadAll(tr)
		if err != nil {
			return err
		}
		// denormalize paths
		b = transformJSONL(b, a.mapper().Denormalize)
		dest := plan.Session.Path
		if dest == "" && len(plan.Files) > 0 {
			dest = plan.Files[0]
		}
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

// transformJSONL rewrites path-like strings in JSON objects without global prose replace.
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
		rewriteValue(v, mapPath)
		out, err := json.Marshal(v)
		if err == nil {
			lines[i] = out
		}
	}
	return bytes.Join(lines, []byte("\n"))
}

func rewriteValue(v any, mapPath func(string) string) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			// only transform known structural keys and string values that look like absolute paths
			if s, ok := child.(string); ok {
				if isPathish(s) && (k == "path" || k == "cwd" || k == "file" || k == "filename" || strings.Contains(k, "path") || strings.HasPrefix(s, "/") || (len(s) > 2 && s[1] == ':')) {
					t[k] = mapPath(s)
					continue
				}
				// also rewrite paths embedded in content carefully only if full string is a path
				if isPathish(s) && (strings.HasPrefix(s, "/") || (len(s) > 2 && s[1] == ':')) && !strings.Contains(s, " ") {
					t[k] = mapPath(s)
				}
			} else {
				rewriteValue(child, mapPath)
			}
		}
	case []any:
		for _, child := range t {
			rewriteValue(child, mapPath)
		}
	}
}

func isPathish(s string) bool {
	if s == "" || strings.Contains(s, "\n") {
		return false
	}
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, "${") {
		return true
	}
	if len(s) > 2 && s[1] == ':' && (s[2] == '\\' || s[2] == '/') {
		return true
	}
	return false
}
