// Package codex implements the OpenAI Codex CLI session adapter.
package codex

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/fsx"
	"github.com/HarjjotSinghh/reinstate/internal/pathmap"
)

const maxJSONLRecordBytes = 16 << 20

const (
	minimumVerifiedCodexVersion = "0.133.0"
	maximumVerifiedCodexVersion = "0.146.0"
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
	root := a.Root
	explicitRoot := root != ""
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
	sessions := filepath.Join(root, "sessions")
	if st, err := os.Stat(sessions); err != nil || !st.IsDir() {
		return inst, adapter.CompatibilityUntested, nil
	}
	inst.Version = "layout-sessions-jsonl-v1"
	if !explicitRoot {
		output, versionErr := exec.CommandContext(ctx, "codex", "--version").Output()
		if versionErr != nil {
			return inst, adapter.CompatibilityUntested, nil
		}
		reported := adapter.StableVersionFromOutput(string(output))
		if reported == "" {
			return inst, adapter.CompatibilityUntested, nil
		}
		inst.Version = reported
		if !isSupportedVersion(inst.Version) {
			return inst, adapter.CompatibilityUntested, nil
		}
	}
	return inst, adapter.CompatibilitySupported, nil
}

func isSupportedVersion(version string) bool {
	return adapter.StableVersionInRange(version, minimumVerifiedCodexVersion, maximumVerifiedCodexVersion)
}

// SupportedVersion reports whether a stable Codex CLI version is inside the
// physically verified native session range. Native preflight uses the same
// fail-closed contract as the sync adapter without executing an export or
// restore.
func SupportedVersion(version string) bool {
	return isSupportedVersion(version)
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
	projectIDsByRoot, err := a.projectIDsByRoot()
	if err != nil {
		return nil, err
	}
	sessionsDir := filepath.Join(inst.Root, "sessions")
	err = filepath.Walk(sessionsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".jsonl") {
			return nil
		}
		id := strings.TrimSuffix(info.Name(), ".jsonl")
		projectID := "unknown"
		// Read only the first record. Current rollouts keep session identity in
		// a session_meta payload; older fixtures used top-level fields.
		if file, openErr := os.Open(path); openErr == nil {
			reader := bufio.NewReader(file)
			line, readErr := reader.ReadBytes('\n')
			_ = file.Close()
			if readErr != nil && readErr != io.EOF {
				return readErr
			}
			var meta map[string]any
			if json.Unmarshal(bytes.TrimSpace(line), &meta) == nil {
				source := meta
				if payload, ok := meta["payload"].(map[string]any); ok {
					source = payload
				}
				if cwd, ok := source["cwd"].(string); ok && cwd != "" {
					projectID = cwd
				}
				if id2, ok := source["id"].(string); ok && id2 != "" {
					id = id2
				}
			}
		}
		if len(projectIDsByRoot) > 0 {
			canonicalID, mapped := projectIDsByRoot[canonicalProjectRoot(projectID)]
			if !mapped {
				return nil
			}
			projectID = canonicalID
		}
		if opts.ProjectID != "" && projectID != opts.ProjectID {
			return nil
		}
		relativePath, relErr := filepath.Rel(inst.Root, path)
		if relErr != nil {
			return relErr
		}
		sessions = append(sessions, adapter.Session{
			ID:           id,
			Agent:        "codex",
			ProjectID:    projectID,
			Title:        info.Name(),
			UpdatedAt:    info.ModTime().Unix(),
			SizeBytes:    info.Size(),
			Path:         path,
			RelativePath: filepath.ToSlash(relativePath),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

func (a *Adapter) projectIDsByRoot() (map[string]string, error) {
	mapped := make(map[string]string, len(a.Projects))
	for canonicalID, localRoot := range a.Projects {
		root := canonicalProjectRoot(localRoot)
		if previous, exists := mapped[root]; exists && previous != canonicalID {
			return nil, fmt.Errorf(
				"codex project mappings %q and %q resolve to the same local root %q",
				previous,
				canonicalID,
				root,
			)
		}
		mapped[root] = canonicalID
	}
	return mapped, nil
}

func canonicalProjectRoot(projectPath string) string {
	canonicalPath := filepath.Clean(projectPath)
	if resolved, err := filepath.EvalSymlinks(canonicalPath); err == nil {
		canonicalPath = resolved
	}
	canonicalPath = filepath.ToSlash(canonicalPath)
	if runtime.GOOS == "windows" {
		canonicalPath = strings.ToLower(canonicalPath)
	}
	return canonicalPath
}

func (a *Adapter) mapper() pathmap.Mapper {
	normalizationProjects := make(map[string]string, len(a.Projects))
	for canonicalID, localRoot := range a.Projects {
		resolvedRoot := filepath.Clean(localRoot)
		if resolved, err := filepath.EvalSymlinks(resolvedRoot); err == nil {
			resolvedRoot = resolved
		}
		normalizationProjects[canonicalID] = resolvedRoot
	}
	return pathmap.Mapper{
		Home:              a.Home,
		Projects:          a.Projects,
		NormalizeProjects: normalizationProjects,
	}
}

func (a *Adapter) PlanExport(ctx context.Context, s adapter.Session, opts adapter.ExportOptions) (adapter.ExportPlan, error) {
	_ = opts
	inst, compatibility, err := a.Detect(ctx)
	if err != nil {
		return adapter.ExportPlan{}, err
	}
	if compatibility != adapter.CompatibilitySupported {
		return adapter.ExportPlan{}, fmt.Errorf("codex compatibility %s refuses export", compatibility)
	}
	relative := s.RelativePath
	if relative == "" {
		relativePath, relErr := filepath.Rel(inst.Root, s.Path)
		if relErr != nil {
			return adapter.ExportPlan{}, relErr
		}
		relative = filepath.ToSlash(relativePath)
	}
	expected, err := safeRestorePath(inst.Root, relative, "sessions")
	if err != nil {
		return adapter.ExportPlan{}, err
	}
	if filepath.Clean(expected) != filepath.Clean(s.Path) {
		return adapter.ExportPlan{}, fmt.Errorf("codex session path escapes detected root")
	}
	s.RelativePath = filepath.ToSlash(relative)
	return adapter.ExportPlan{Session: s, Files: []string{s.Path}}, nil
}

func (a *Adapter) Export(ctx context.Context, plan adapter.ExportPlan, w io.Writer) error {
	if len(plan.Files) != 1 {
		return fmt.Errorf("codex export requires exactly one session file")
	}
	tw := tar.NewWriter(w)
	for _, f := range plan.Files {
		if err := ctx.Err(); err != nil {
			return err
		}
		source, err := os.Open(f)
		if err != nil {
			return err
		}
		transformed, err := os.CreateTemp("", ".reinstate-codex-export-*")
		if err != nil {
			_ = source.Close()
			return err
		}
		transformedPath := transformed.Name()
		defer func() {
			_ = transformed.Close()
			_ = os.Remove(transformedPath)
		}()
		if err := os.Chmod(transformedPath, 0o600); err != nil {
			_ = source.Close()
			return err
		}
		if err := transformJSONLStream(source, transformed, a.mapper().Normalize, nil); err != nil {
			_ = source.Close()
			return err
		}
		if err := source.Close(); err != nil {
			return err
		}
		info, err := transformed.Stat()
		if err != nil {
			return err
		}
		if _, err := transformed.Seek(0, io.SeekStart); err != nil {
			return err
		}
		headerName := plan.Session.RelativePath
		if headerName == "" {
			headerName = filepath.Base(f)
		}
		hdr := &tar.Header{Name: filepath.ToSlash(headerName), Mode: 0o600, Size: info.Size(), ModTime: time.Now()}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := io.Copy(tw, transformed); err != nil {
			return err
		}
	}
	return tw.Close()
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
	archiveRelative := snap.RelativePath
	if archiveRelative == "" {
		archiveRelative = filepath.ToSlash(filepath.Join("sessions", snap.SessionID+".jsonl"))
	}
	destinationRelative := archiveRelative
	if opts.DestinationRelativePath != "" {
		destinationRelative = opts.DestinationRelativePath
	}
	destinationID := snap.SessionID
	if opts.ForkSessionID != "" {
		destinationID = opts.ForkSessionID
	}
	dest, err := safeRestorePath(inst.Root, destinationRelative, "sessions")
	if err != nil {
		return adapter.RestorePlan{}, err
	}
	return adapter.RestorePlan{
		Session: adapter.Session{
			ID: destinationID, Agent: "codex", ProjectID: snap.ProjectID,
			Path: dest, RelativePath: filepath.ToSlash(destinationRelative),
		},
		Files:           []string{dest},
		BackupRoot:      opts.BackupRoot,
		ArchivePath:     filepath.ToSlash(archiveRelative),
		SourceSessionID: snap.SessionID,
	}, nil
}

func safeRestorePath(root, relative, requiredTop string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == "." || clean == ".." || filepath.IsAbs(clean) ||
		strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe snapshot path %q", relative)
	}
	parts := strings.Split(filepath.ToSlash(clean), "/")
	if len(parts) < 2 || parts[0] != requiredTop || !strings.HasSuffix(parts[len(parts)-1], ".jsonl") {
		return "", fmt.Errorf("unexpected %s snapshot path %q", requiredTop, relative)
	}
	return filepath.Join(root, clean), nil
}

func (a *Adapter) Restore(ctx context.Context, plan adapter.RestorePlan, r io.Reader) error {
	if plan.Refuse != "" {
		return fmt.Errorf("%s", plan.Refuse)
	}
	tr := tar.NewReader(r)
	hdr, err := tr.Next()
	if err != nil {
		return fmt.Errorf("read Codex snapshot archive: %w", err)
	}
	expected := plan.ArchivePath
	if expected == "" {
		expected = filepath.ToSlash(plan.Session.RelativePath)
	}
	if expected != "" && filepath.ToSlash(hdr.Name) != expected {
		return fmt.Errorf("unexpected Codex archive entry %q", hdr.Name)
	}
	dest := plan.Session.Path
	if dest == "" && len(plan.Files) > 0 {
		dest = plan.Files[0]
	}
	if dest == "" {
		return fmt.Errorf("codex restore destination required")
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return err
	}
	// Record the target before doing any slow work so a concurrent agent write
	// is caught before the rename discards it.
	before, err := fsx.FingerprintFile(dest)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".reinstate-restore-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		_ = tmp.Close()
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return err
	}
	rewriteID := func(v any) {
		rewriteCodexSessionID(v, plan.SourceSessionID, plan.Session.ID)
	}
	if err := transformJSONLStream(tr, tmp, a.mapper().Denormalize, rewriteID); err != nil {
		return err
	}
	if _, err := tr.Next(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("codex snapshot archive contains multiple entries")
		}
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := fsx.VerifyUnchanged(dest, before); err != nil {
		return err
	}
	if _, err := os.Stat(dest); err == nil {
		relative := plan.Session.RelativePath
		if relative == "" {
			relative = filepath.Base(dest)
		}
		if _, err := fsx.BackupFile(dest, plan.BackupRoot, relative); err != nil {
			return fmt.Errorf("backup existing Codex session: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := fsx.VerifyUnchanged(dest, before); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, dest); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func transformJSONLStream(source io.Reader, dest io.Writer, mapPath func(string) string, rewrite func(any)) error {
	reader := bufio.NewReaderSize(source, 64<<10)
	var record bytes.Buffer
	for {
		fragment, err := reader.ReadSlice('\n')
		if record.Len()+len(fragment) > maxJSONLRecordBytes {
			return fmt.Errorf("JSONL record exceeds %d bytes", maxJSONLRecordBytes)
		}
		if _, writeErr := record.Write(fragment); writeErr != nil {
			return writeErr
		}
		if err == bufio.ErrBufferFull {
			continue
		}
		line := record.Bytes()
		if len(line) > 0 {
			hasNewline := line[len(line)-1] == '\n'
			body := line
			if hasNewline {
				body = line[:len(line)-1]
			}
			if _, writeErr := dest.Write(transformJSONLWithHook(body, mapPath, rewrite)); writeErr != nil {
				return writeErr
			}
			if hasNewline {
				if _, writeErr := dest.Write([]byte{'\n'}); writeErr != nil {
					return writeErr
				}
			}
		}
		record.Reset()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func transformJSONL(data []byte, mapPath func(string) string) []byte {
	return transformJSONLWithHook(data, mapPath, nil)
}

func transformJSONLWithHook(data []byte, mapPath func(string) string, hook func(any)) []byte {
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
		if hook != nil {
			hook(v)
		}
		if out, err := json.Marshal(v); err == nil {
			lines[i] = out
		}
	}
	return bytes.Join(lines, []byte("\n"))
}

func rewriteCodexSessionID(v any, source, destination string) {
	if source == "" || destination == "" || source == destination {
		return
	}
	record, ok := v.(map[string]any)
	if !ok || record["type"] != "session_meta" {
		return
	}
	payload, ok := record["payload"].(map[string]any)
	if !ok {
		return
	}
	if id, ok := payload["id"].(string); ok && id == source {
		payload["id"] = destination
	}
}

func rewrite(v any, mapPath func(string) string) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if s, ok := child.(string); ok {
				if isCodexPathKey(k) && isAbs(s) {
					t[k] = mapPath(s)
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

func isCodexPathKey(key string) bool {
	switch key {
	case "cwd", "path", "workdir", "workingDirectory", "filePath", "file_path":
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
