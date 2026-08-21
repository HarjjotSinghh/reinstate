// Package claude implements the Claude Code session adapter.
package claude

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/fsx"
	"github.com/HarjjotSinghh/reinstate/internal/pathmap"
)

const maxJSONLRecordBytes = 16 << 20

const (
	minimumVerifiedClaudeVersion = "2.1.219"
	maximumVerifiedClaudeVersion = "2.1.238"
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
	root := a.Root
	explicitRoot := root != ""
	if root == "" {
		if configured := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")); configured != "" {
			root = configured
			explicitRoot = true
		}
	}
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
	projects := filepath.Join(root, "projects")
	if st, err := os.Stat(projects); err != nil || !st.IsDir() {
		if !explicitRoot {
			return inst, adapter.CompatibilityUntested, nil
		}
		if inst.Version == "unknown" {
			inst.Version = "layout-projects-jsonl-v1"
		}
		return inst, adapter.CompatibilitySupported, nil
	}
	if inst.Version == "unknown" {
		inst.Version = "layout-projects-jsonl-v1"
	}
	if !explicitRoot && !isSupportedVersion(inst.Version) {
		output, versionErr := adapter.RunVersionCommand(ctx, "claude")
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
	if !explicitRoot && !isSupportedVersion(inst.Version) {
		return inst, adapter.CompatibilityUntested, nil
	}
	return inst, adapter.CompatibilitySupported, nil
}

func isSupportedVersion(version string) bool {
	return adapter.StableVersionInRange(version, minimumVerifiedClaudeVersion, maximumVerifiedClaudeVersion)
}

// SupportedVersion reports whether a stable Claude Code version is inside the
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
	projectIDsByDirectory, err := a.projectIDsByDirectory()
	if err != nil {
		return nil, err
	}
	// layout: root/projects/<project>/session-*.jsonl or recursive *.jsonl under projects
	projects := filepath.Join(inst.Root, "projects")
	err = filepath.Walk(projects, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "subagents" {
				relativeDirectory, relErr := filepath.Rel(projects, path)
				if relErr != nil {
					return relErr
				}
				if strings.Count(filepath.ToSlash(relativeDirectory), "/") >= 2 {
					return filepath.SkipDir
				}
			}
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
		rootRel, relErr := filepath.Rel(inst.Root, path)
		if relErr != nil {
			return relErr
		}
		proj := filepath.Dir(rel)
		if proj == "." {
			proj = "unknown"
		}
		canonicalID, mapped := projectIDsByDirectory[filepath.ToSlash(proj)]
		if len(projectIDsByDirectory) > 0 && !mapped {
			return nil
		}
		if mapped {
			proj = canonicalID
		}
		if opts.ProjectID != "" && proj != opts.ProjectID {
			return nil
		}
		sessions = append(sessions, adapter.Session{
			ID:           strings.TrimSuffix(info.Name(), ".jsonl"),
			Agent:        "claude",
			ProjectID:    proj,
			Title:        info.Name(),
			UpdatedAt:    info.ModTime().Unix(),
			SizeBytes:    info.Size(),
			Path:         path,
			RelativePath: filepath.ToSlash(rootRel),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

func (a *Adapter) projectIDsByDirectory() (map[string]string, error) {
	mapped := make(map[string]string, len(a.Projects))
	for canonicalID, localRoot := range a.Projects {
		directory := claudeProjectDirectory(localRoot)
		if previous, exists := mapped[directory]; exists && previous != canonicalID {
			return nil, fmt.Errorf(
				"claude project mappings %q and %q resolve to the same vendor directory %q",
				previous,
				canonicalID,
				directory,
			)
		}
		mapped[directory] = canonicalID
	}
	return mapped, nil
}

// claudeProjectDirectory reproduces Claude Code's project-directory key:
// realpath when available, non-ASCII-alphanumeric UTF-16 units replaced with
// dashes, and a JavaScript-style signed 32-bit hash when the key exceeds 200
// characters.
func claudeProjectDirectory(projectPath string) string {
	canonicalPath := filepath.Clean(projectPath)
	if resolved, err := filepath.EvalSymlinks(canonicalPath); err == nil {
		canonicalPath = resolved
	}
	units := utf16.Encode([]rune(canonicalPath))
	var encoded strings.Builder
	encoded.Grow(len(units))
	for _, unit := range units {
		if unit >= 'a' && unit <= 'z' ||
			unit >= 'A' && unit <= 'Z' ||
			unit >= '0' && unit <= '9' {
			encoded.WriteByte(byte(unit))
		} else {
			encoded.WriteByte('-')
		}
	}
	key := encoded.String()
	if len(key) <= 200 {
		return key
	}
	return key[:200] + "-" + strconv.FormatInt(absJavaScriptStringHash(units), 36)
}

func absJavaScriptStringHash(units []uint16) int64 {
	var hash int32
	for _, unit := range units {
		hash = int32(int64(hash)*31 + int64(unit))
	}
	value := int64(hash)
	if value < 0 {
		return -value
	}
	return value
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
		return adapter.ExportPlan{}, fmt.Errorf("claude compatibility %s refuses export", compatibility)
	}
	relative := s.RelativePath
	if relative == "" {
		relativePath, relErr := filepath.Rel(inst.Root, s.Path)
		if relErr != nil {
			return adapter.ExportPlan{}, relErr
		}
		relative = filepath.ToSlash(relativePath)
	}
	expected, err := safeRestorePath(inst.Root, relative, "projects")
	if err != nil {
		return adapter.ExportPlan{}, err
	}
	if filepath.Clean(expected) != filepath.Clean(s.Path) {
		return adapter.ExportPlan{}, fmt.Errorf("claude session path escapes detected root")
	}
	s.RelativePath = filepath.ToSlash(relative)
	return adapter.ExportPlan{Session: s, Files: []string{s.Path}}, nil
}

func (a *Adapter) Export(ctx context.Context, plan adapter.ExportPlan, w io.Writer) error {
	if len(plan.Files) != 1 {
		return fmt.Errorf("claude export requires exactly one session file")
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
		transformed, err := os.CreateTemp("", ".reinstate-claude-export-*")
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
		hdr := &tar.Header{
			Name:    filepath.ToSlash(headerName),
			Mode:    0o600,
			Size:    info.Size(),
			ModTime: time.Now(),
		}
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
		archiveRelative = filepath.ToSlash(filepath.Join("projects", snap.ProjectID, snap.SessionID+".jsonl"))
	}
	if _, err := safeRestorePath(inst.Root, archiveRelative, "projects"); err != nil {
		return adapter.RestorePlan{}, err
	}
	archiveParts := strings.Split(path.Clean(filepath.ToSlash(archiveRelative)), "/")
	if len(archiveParts) < 3 {
		return adapter.RestorePlan{}, fmt.Errorf("unexpected Claude snapshot path %q", archiveRelative)
	}
	archiveProjectDirectory := archiveParts[1]
	destinationID := snap.SessionID
	if opts.ForkSessionID != "" {
		destinationID = opts.ForkSessionID
	}
	destinationRelative := archiveRelative
	if localRoot, ok := a.Projects[snap.ProjectID]; ok {
		destinationName := destinationID + ".jsonl"
		if opts.DestinationRelativePath != "" {
			if _, err := safeRestorePath(inst.Root, opts.DestinationRelativePath, "projects"); err != nil {
				return adapter.RestorePlan{}, err
			}
			if providedName := path.Base(filepath.ToSlash(opts.DestinationRelativePath)); providedName != destinationName {
				return adapter.RestorePlan{}, fmt.Errorf(
					"claude restore destination filename %q does not match session %q",
					providedName,
					destinationID,
				)
			}
		}
		destinationRelative = path.Join("projects", claudeProjectDirectory(localRoot), destinationName)
	} else {
		if len(a.Projects) > 0 || snap.ProjectID != archiveProjectDirectory {
			return adapter.RestorePlan{}, fmt.Errorf(
				"claude snapshot project %q has no local mapping on this device; configure it with rein init --project %s=/absolute/path (or the equivalent config mapping) before pull; legacy snapshots may need to be pushed again from the source device with this Reinstate release",
				snap.ProjectID,
				snap.ProjectID,
			)
		}
		if opts.DestinationRelativePath != "" {
			destinationRelative = opts.DestinationRelativePath
		}
	}
	dest, err := safeRestorePath(inst.Root, destinationRelative, "projects")
	if err != nil {
		return adapter.RestorePlan{}, err
	}
	return adapter.RestorePlan{
		Session: adapter.Session{
			ID: destinationID, Agent: "claude", ProjectID: snap.ProjectID,
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
		return fmt.Errorf("read Claude snapshot archive: %w", err)
	}
	expected := plan.ArchivePath
	if expected == "" {
		expected = filepath.ToSlash(plan.Session.RelativePath)
	}
	if expected != "" && filepath.ToSlash(hdr.Name) != expected {
		return fmt.Errorf("unexpected Claude archive entry %q", hdr.Name)
	}
	dest := plan.Session.Path
	if dest == "" && len(plan.Files) > 0 {
		dest = plan.Files[0]
	}
	if dest == "" {
		return fmt.Errorf("claude restore destination required")
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
		rewriteClaudeSessionID(v, plan.SourceSessionID, plan.Session.ID)
	}
	if err := transformJSONLStream(tr, tmp, a.mapper().Denormalize, rewriteID); err != nil {
		return err
	}
	if _, err := tr.Next(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("claude snapshot archive contains multiple entries")
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
			return fmt.Errorf("backup existing Claude session: %w", err)
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

// transformJSONL rewrites path-like strings in JSON objects without global prose replace.
func transformJSONL(data []byte, mapPath func(string) string) []byte {
	return transformJSONLWithHook(data, mapPath, nil)
}

func transformJSONLWithHook(data []byte, mapPath func(string) string, rewrite func(any)) []byte {
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
		if rewrite != nil {
			rewrite(v)
		}
		out, err := json.Marshal(v)
		if err == nil {
			lines[i] = out
		}
	}
	return bytes.Join(lines, []byte("\n"))
}

func rewriteClaudeSessionID(v any, source, destination string) {
	if source == "" || destination == "" || source == destination {
		return
	}
	switch value := v.(type) {
	case map[string]any:
		for key, child := range value {
			if text, ok := child.(string); ok && (key == "sessionId" || key == "session_id") && text == source {
				value[key] = destination
				continue
			}
			rewriteClaudeSessionID(child, source, destination)
		}
	case []any:
		for _, child := range value {
			rewriteClaudeSessionID(child, source, destination)
		}
	}
}

func rewriteValue(v any, mapPath func(string) string) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if s, ok := child.(string); ok {
				if isKnownPathKey(k) && isPathish(s) {
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

func isKnownPathKey(key string) bool {
	switch key {
	case "cwd", "path", "file", "filename", "filePath", "file_path",
		"projectPath", "project_path", "workingDirectory", "workdir":
		return true
	default:
		return false
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
