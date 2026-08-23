// Package opencode implements the OpenCode embedded-store session adapter.
//
// OpenCode keeps every session in one embedded SQLite database (opencode.db)
// rather than a file per session, so this adapter is shaped differently from
// the file-per-session Claude and Codex adapters:
//
//   - The synced unit is a single session extracted from the shared store, not
//     a standalone vendor file. Export reads only the session, project, message
//     and part rows for one session id and serialises them into a portable,
//     deterministic JSON document. The credential and account tables in the
//     same database are never opened.
//   - Restore writes those rows back into the destination's own opencode.db —
//     the first tier at which Reinstate writes into a vendor store, per
//     docs/agent-support-tiers.md. It never invents the store's schema: a
//     destination without an initialised opencode.db is refused rather than
//     fabricated.
//   - Because no session owns a file to hash, the adapter implements
//     adapter.SessionRevisioner: the per-session revision is the digest of the
//     normalised portable document, which is stable across devices and so lets
//     the sync engine detect a real content change instead of every session
//     appearing to change whenever the shared database file does.
//
// Reads go through internal/vendorsqlite, which copies the database and its
// write-ahead log into a private directory before opening, so a session the
// vendor has only just written is visible and nothing is written beside the
// vendor's database.
package opencode

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/fsx"
	"github.com/HarjjotSinghh/reinstate/internal/pathmap"
	"github.com/HarjjotSinghh/reinstate/internal/vendorsqlite"

	_ "modernc.org/sqlite"
)

// DatabaseName is the embedded session store inside the resolved data root.
const DatabaseName = "opencode.db"

// exportSchema versions the portable document so a future shape change fails
// closed instead of being silently misread.
const exportSchema = "opencode-session/1"

// maxExportMessages bounds one session's export so a pathological store cannot
// make an export unbounded.
const maxExportMessages = 20000

// archiveTop is the required first segment of a snapshot's relative path.
const archiveTop = "sessions"

// Adapter implements adapter.Adapter for OpenCode.
type Adapter struct {
	// Root overrides the resolved OpenCode data root (the directory holding
	// opencode.db). Empty resolves from XDG_DATA_HOME or the user home. Tests
	// must set this to a synthetic fixture root, never a real store.
	Root string
	// Home is the local user home used for portable path remapping.
	Home string
	// Projects maps canonical project IDs to configured local roots for
	// portable path remapping and discovery filtering.
	Projects map[string]string
	// GOOS overrides path style for cross-OS fixture tests
	// ("windows" | "darwin" | "linux"). Empty uses runtime.GOOS.
	GOOS string
	// ForceCompat overrides installation detection (tests only).
	ForceCompat adapter.Compatibility
}

func (a *Adapter) Name() string { return "opencode" }

// Exclusions lists content that must never leave the device. OpenCode keeps
// credentials and account tokens in the same database as sessions; the export
// path only ever selects the session, project, message and part tables, so
// those tables are excluded by construction, and these patterns state the
// contract the code keeps.
func (a *Adapter) Exclusions() []adapter.Exclusion {
	return []adapter.Exclusion{
		{Pattern: "**/auth.json", Reason: "credentials"},
		{Pattern: "table:credential", Reason: "credentials table is never exported"},
		{Pattern: "table:account", Reason: "account tokens are never exported"},
		{Pattern: "table:control_account", Reason: "account tokens are never exported"},
		{Pattern: "table:account_state", Reason: "account state is never exported"},
	}
}

func (a *Adapter) resolveRoot() (string, bool) {
	if strings.TrimSpace(a.Root) != "" {
		return a.Root, true
	}
	if configured := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); configured != "" {
		return filepath.Join(configured, "opencode"), false
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "opencode"), false
	}
	return "", false
}

func (a *Adapter) databasePath() (string, error) {
	root, _ := a.resolveRoot()
	if root == "" {
		return "", fmt.Errorf("opencode data root is unavailable")
	}
	return filepath.Join(root, DatabaseName), nil
}

// Detect reports whether an OpenCode store is present. A regular opencode.db
// under the resolved root is a supported install; its absence is
// NOT_INSTALLED. No session body is read.
func (a *Adapter) Detect(ctx context.Context) (adapter.Install, adapter.Compatibility, error) {
	_ = ctx
	root, _ := a.resolveRoot()
	inst := adapter.Install{Agent: "opencode", Root: root, Layout: "embedded-sqlite-session-store"}
	if a.ForceCompat != "" {
		return inst, a.ForceCompat, nil
	}
	if root == "" {
		return adapter.Install{Agent: "opencode"}, adapter.CompatibilityNotInstalled, nil
	}
	db := filepath.Join(root, DatabaseName)
	st, err := os.Stat(db)
	if err != nil || !st.Mode().IsRegular() {
		return inst, adapter.CompatibilityNotInstalled, nil
	}
	inst.Version = "layout-embedded-sqlite-v1"
	return inst, adapter.CompatibilitySupported, nil
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
		GOOS:              a.GOOS,
	}
}

// Discover lists every session in the store as adapter.Session metadata. Every
// session shares the store path; RelativePath is the virtual per-session
// archive name so the sync engine can address one session at a time.
func (a *Adapter) Discover(ctx context.Context, opts adapter.DiscoverOptions) ([]adapter.Session, error) {
	inst, compat, err := a.Detect(ctx)
	if err != nil {
		return nil, err
	}
	if compat == adapter.CompatibilityNotInstalled {
		return nil, nil
	}
	dbPath := filepath.Join(inst.Root, DatabaseName)
	handle, err := vendorsqlite.Open(dbPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = handle.Close() }()

	rows, err := handle.DB.QueryContext(ctx, `
SELECT id, COALESCE(directory, ''), COALESCE(title, ''),
       COALESCE(time_updated, 0), COALESCE(time_created, 0)
  FROM session
 ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var sessions []adapter.Session
	for rows.Next() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var (
			id, directory, title string
			updated, created     int64
		)
		if err := rows.Scan(&id, &directory, &title, &updated, &created); err != nil {
			return nil, err
		}
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		projectID := strings.TrimSpace(directory)
		if opts.ProjectID != "" && opts.ProjectID != projectID {
			continue
		}
		stamp := updated
		if stamp == 0 {
			stamp = created
		}
		sessions = append(sessions, adapter.Session{
			ID:           id,
			Agent:        "opencode",
			ProjectID:    projectID,
			Title:        title,
			UpdatedAt:    unixFromMillisOrSeconds(stamp),
			Path:         dbPath,
			RelativePath: archiveRelativePath(id),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(sessions, func(i, j int) bool { return sessions[i].ID < sessions[j].ID })
	return sessions, nil
}

// SessionRevision returns a stable per-session content revision: the digest of
// the normalised portable document. It is independent of the device's paths, so
// the same session on two machines has the same revision and a genuine edit
// changes it.
func (a *Adapter) SessionRevision(ctx context.Context, s adapter.Session) (string, error) {
	dbPath := s.Path
	if strings.TrimSpace(dbPath) == "" {
		p, err := a.databasePath()
		if err != nil {
			return "", err
		}
		dbPath = p
	}
	doc, err := a.readSessionDocument(ctx, dbPath, s.ID)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func (a *Adapter) PlanExport(ctx context.Context, s adapter.Session, opts adapter.ExportOptions) (adapter.ExportPlan, error) {
	_ = opts
	inst, compat, err := a.Detect(ctx)
	if err != nil {
		return adapter.ExportPlan{}, err
	}
	if compat != adapter.CompatibilitySupported {
		return adapter.ExportPlan{}, fmt.Errorf("opencode compatibility %s refuses export", compat)
	}
	if strings.TrimSpace(s.ID) == "" {
		return adapter.ExportPlan{}, fmt.Errorf("opencode export requires a session id")
	}
	dbPath := filepath.Join(inst.Root, DatabaseName)
	s.Path = dbPath
	s.RelativePath = archiveRelativePath(s.ID)
	return adapter.ExportPlan{Session: s, Files: []string{dbPath}}, nil
}

// Export serialises one session into a portable JSON document and writes it as
// the single entry of a tar stream.
func (a *Adapter) Export(ctx context.Context, plan adapter.ExportPlan, w io.Writer) error {
	if len(plan.Files) != 1 {
		return fmt.Errorf("opencode export requires exactly one store path")
	}
	doc, err := a.readSessionDocument(ctx, plan.Files[0], plan.Session.ID)
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	name := plan.Session.RelativePath
	if name == "" {
		name = archiveRelativePath(plan.Session.ID)
	}
	tw := tar.NewWriter(w)
	hdr := &tar.Header{Name: filepath.ToSlash(name), Mode: 0o600, Size: int64(len(encoded)), ModTime: time.Now()}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := tw.Write(encoded); err != nil {
		return err
	}
	return tw.Close()
}

func (a *Adapter) PlanRestore(ctx context.Context, snap adapter.Snapshot, opts adapter.RestoreOptions) (adapter.RestorePlan, error) {
	inst, compat, err := a.Detect(ctx)
	if err != nil {
		return adapter.RestorePlan{}, err
	}
	if !adapter.CanRestore(compat, opts.CompatibilityOK) {
		return adapter.RestorePlan{Refuse: fmt.Sprintf("compatibility %s refuses restore", compat)}, nil
	}
	archiveRelative := snap.RelativePath
	if archiveRelative == "" {
		archiveRelative = archiveRelativePath(snap.SessionID)
	}
	if _, err := archiveSessionID(archiveRelative); err != nil {
		return adapter.RestorePlan{}, err
	}
	destinationRelative := archiveRelative
	if opts.DestinationRelativePath != "" {
		destinationRelative = opts.DestinationRelativePath
	}
	if _, err := archiveSessionID(destinationRelative); err != nil {
		return adapter.RestorePlan{}, err
	}
	destinationID := snap.SessionID
	if opts.ForkSessionID != "" {
		destinationID = opts.ForkSessionID
	}
	dbPath := filepath.Join(inst.Root, DatabaseName)
	return adapter.RestorePlan{
		Session: adapter.Session{
			ID: destinationID, Agent: "opencode", ProjectID: snap.ProjectID,
			Path: dbPath, RelativePath: filepath.ToSlash(destinationRelative),
		},
		Files:           []string{dbPath},
		BackupRoot:      opts.BackupRoot,
		ArchivePath:     filepath.ToSlash(archiveRelative),
		SourceSessionID: snap.SessionID,
	}, nil
}

// Restore writes the session in the archive back into the destination store.
//
// The write is atomic and reversible: a checkpointed working copy of the store
// is edited, the live store is fingerprinted before and after so a concurrent
// vendor write aborts the restore, the live store is backed up, and only then
// is the working copy renamed over it. The stale write-ahead sidecars are
// removed so the vendor reopens the merged database rather than a database plus
// an out-of-date log.
func (a *Adapter) Restore(ctx context.Context, plan adapter.RestorePlan, r io.Reader) error {
	if plan.Refuse != "" {
		return fmt.Errorf("%s", plan.Refuse)
	}
	tr := tar.NewReader(r)
	hdr, err := tr.Next()
	if err != nil {
		return fmt.Errorf("read OpenCode snapshot archive: %w", err)
	}
	expected := plan.ArchivePath
	if expected == "" {
		expected = filepath.ToSlash(plan.Session.RelativePath)
	}
	if expected != "" && filepath.ToSlash(hdr.Name) != expected {
		return fmt.Errorf("unexpected OpenCode archive entry %q", hdr.Name)
	}
	body, err := io.ReadAll(io.LimitReader(tr, maxSnapshotBytes+1))
	if err != nil {
		return err
	}
	if int64(len(body)) > maxSnapshotBytes {
		return fmt.Errorf("opencode snapshot exceeds %d bytes", maxSnapshotBytes)
	}
	if _, err := tr.Next(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("opencode snapshot archive contains multiple entries")
		}
		return err
	}
	var doc sessionDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return fmt.Errorf("decode OpenCode snapshot: %w", err)
	}
	if doc.Schema != exportSchema {
		return fmt.Errorf("unrecognised OpenCode snapshot schema %q", doc.Schema)
	}
	if strings.TrimSpace(doc.Session.ID) == "" {
		return fmt.Errorf("opencode snapshot carries no session id")
	}
	if plan.SourceSessionID != "" && doc.Session.ID != plan.SourceSessionID {
		return fmt.Errorf("opencode snapshot is session %q, not the planned %q", doc.Session.ID, plan.SourceSessionID)
	}
	if strings.TrimSpace(plan.Session.ID) == "" {
		return fmt.Errorf("opencode restore requires a destination session id")
	}

	dest := plan.Session.Path
	if dest == "" && len(plan.Files) > 0 {
		dest = plan.Files[0]
	}
	if dest == "" {
		return fmt.Errorf("opencode restore destination required")
	}
	if info, statErr := os.Stat(dest); statErr != nil || !info.Mode().IsRegular() {
		// Restore writes into the vendor's own store and never invents its
		// schema; a store that has not been initialised by OpenCode has nowhere
		// to receive the session.
		return fmt.Errorf("opencode store is not initialised at %s; run opencode once before pulling", dest)
	}

	before, err := fsx.FingerprintFile(dest)
	if err != nil {
		return err
	}

	working, cleanup, err := checkpointedCopy(dest)
	if err != nil {
		return err
	}
	defer cleanup()

	if err := a.applyDocument(ctx, working, doc, plan.SourceSessionID, plan.Session.ID); err != nil {
		return err
	}

	if err := fsx.VerifyUnchanged(dest, before); err != nil {
		return err
	}
	// The backup is the whole store, not one session, so it is labelled by the
	// store file name rather than the virtual sessions/<id>.json path.
	if _, err := fsx.BackupFile(dest, plan.BackupRoot, filepath.Base(dest)); err != nil {
		return fmt.Errorf("backup existing OpenCode store: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := fsx.VerifyUnchanged(dest, before); err != nil {
		return err
	}
	if err := os.Rename(working, dest); err != nil {
		return err
	}
	// The old write-ahead sidecars describe the pre-rename inode; leaving them
	// would let the vendor reopen a database plus a stale log.
	_ = os.Remove(dest + "-wal")
	_ = os.Remove(dest + "-shm")
	return nil
}

const maxSnapshotBytes = 256 << 20

func archiveRelativePath(id string) string {
	return archiveTop + "/" + id + ".json"
}

// archiveSessionID validates a snapshot relative path and returns its session
// id. It refuses anything that escapes the store's session namespace, and an
// entry with no id at all. Backslashes are refused outright so the contract is
// identical on every host rather than depending on the platform separator.
func archiveSessionID(relative string) (string, error) {
	if strings.Contains(relative, `\`) {
		return "", fmt.Errorf("unsafe snapshot path %q", relative)
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == "." || clean == ".." || filepath.IsAbs(clean) ||
		strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe snapshot path %q", relative)
	}
	parts := strings.Split(filepath.ToSlash(clean), "/")
	if len(parts) != 2 || parts[0] != archiveTop || !strings.HasSuffix(parts[1], ".json") {
		return "", fmt.Errorf("unexpected opencode snapshot path %q", relative)
	}
	id := strings.TrimSuffix(parts[1], ".json")
	if strings.TrimSpace(id) == "" || id == "." {
		return "", fmt.Errorf("opencode snapshot path %q carries no session id", relative)
	}
	return id, nil
}

func unixFromMillisOrSeconds(value int64) int64 {
	if value <= 0 {
		return 0
	}
	if value > 1e12 {
		return value / 1000
	}
	return value
}
