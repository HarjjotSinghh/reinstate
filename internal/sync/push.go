package sync

import (
	"bufio"
	"bytes"
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

	"github.com/google/uuid"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
	"github.com/HarjjotSinghh/reinstate/internal/device"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

const (
	maxManifestBytes   = 4 << 20
	maxMetadataBytes   = 1 << 20
	defaultMaxPayload  = int64(32 << 30)
	maxManifestRetries = 4
)

var (
	// ErrConflict means the remote head moved beyond the caller's known parent.
	ErrConflict = errors.New("sync: conflict")
	// ErrRemoteProfileNotFound means configured storage has no manifest.
	ErrRemoteProfileNotFound = errors.New("sync: remote profile manifest not found")
)

// Engine orchestrates immutable snapshots and the encrypted remote manifest.
type Engine struct {
	Backend               backend.Backend
	Passphrase            string
	Prefix                string
	Platform              string
	MaxPayloadSize        int64
	RequireRemoteManifest bool
	// Codec overrides the age envelope implementation for deterministic tests.
	// Production callers leave it nil.
	Codec EnvelopeCodec
}

// EnvelopeCodec encrypts and authenticates sync envelopes.
type EnvelopeCodec interface {
	Encrypt(io.Reader, io.Writer, string) error
	DecryptReader(io.Reader, string) (io.Reader, error)
}

type ageEnvelopeCodec struct{}

func (ageEnvelopeCodec) Encrypt(source io.Reader, dest io.Writer, passphrase string) error {
	return crypto.Encrypt(source, dest, passphrase)
}

func (ageEnvelopeCodec) DecryptReader(source io.Reader, passphrase string) (io.Reader, error) {
	return crypto.DecryptReader(source, passphrase)
}

func (e *Engine) envelopeCodec() EnvelopeCodec {
	if e.Codec != nil {
		return e.Codec
	}
	return ageEnvelopeCodec{}
}

// PushSession streams a local artifact into an immutable encrypted snapshot,
// then conditionally advances the session head in the remote manifest.
func (e *Engine) PushSession(ctx context.Context, item PushItem, dryRun bool) (snapshotID string, err error) {
	if e.Backend == nil {
		return "", fmt.Errorf("backend required")
	}
	if e.Passphrase == "" {
		return "", fmt.Errorf("passphrase required")
	}
	if item.Agent == "" || item.SessionID == "" || item.LocalPath == "" {
		return "", fmt.Errorf("agent, session id, and local path are required")
	}
	if isCredentialName(filepath.Base(item.LocalPath)) || containsHardExcludedPath(item.RelativePath) || containsHardExcludedPath(item.LocalPath) {
		return "", fmt.Errorf("refusing to push hard-excluded path")
	}

	source, err := os.Open(item.LocalPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = source.Close() }()

	hash, size, err := crypto.SHA256Reader(source)
	if err != nil {
		return "", err
	}
	if size > e.maxPayloadSize() {
		return "", fmt.Errorf("snapshot payload exceeds maximum size")
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	current, _, err := e.loadManifest(ctx, !e.RequireRemoteManifest)
	if err != nil {
		return "", err
	}
	sessionKey := SessionKey(item.Agent, item.SessionID)
	parent := ""
	if existing, ok := current.Sessions[sessionKey]; ok {
		parent = existing.SnapshotID
	}
	if item.BaseKnown && parent != item.BaseRevision {
		return "", fmt.Errorf("%w: remote session head changed from %q to %q", ErrConflict, item.BaseRevision, parent)
	}

	snapshotID = uuid.NewString()
	relativePath := item.RelativePath
	if relativePath == "" {
		relativePath = filepath.Base(item.LocalPath)
	}
	env := schema.Envelope{
		SchemaVersion:  schema.EnvelopeSchemaVersion,
		Kind:           schema.EnvelopeKind,
		SnapshotID:     snapshotID,
		ParentRevision: parent,
		Agent:          item.Agent,
		AdapterSchema:  1,
		SourcePlatform: e.platform(),
		ProjectID:      item.ProjectID,
		SessionID:      item.SessionID,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		Files: []schema.EnvelopeFile{{
			Path:   filepath.ToSlash(relativePath),
			Mode:   0o600,
			Size:   size,
			SHA256: hash,
		}},
	}
	metaJSON, err := json.Marshal(env)
	if err != nil {
		return "", err
	}
	if dryRun {
		return snapshotID, nil
	}

	cipher, err := os.CreateTemp("", ".reinstate-snapshot-*.age")
	if err != nil {
		return "", err
	}
	cipherPath := cipher.Name()
	defer func() {
		_ = cipher.Close()
		_ = os.Remove(cipherPath)
	}()
	if err := os.Chmod(cipherPath, 0o600); err != nil {
		return "", err
	}
	plain := io.MultiReader(bytes.NewReader(append(metaJSON, '\n')), source)
	if err := e.envelopeCodec().Encrypt(plain, cipher, e.Passphrase); err != nil {
		return "", err
	}
	if err := cipher.Sync(); err != nil {
		return "", err
	}
	info, err := cipher.Stat()
	if err != nil {
		return "", err
	}
	if _, err := cipher.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	snapshotKey := e.key("snapshots/" + snapshotID + ".age")
	if _, err := e.Backend.Put(ctx, snapshotKey, cipher, info.Size(), backend.PutOptions{
		IfNoneMatch: true,
		ContentType: "application/octet-stream",
	}); err != nil {
		return "", err
	}

	entry := schema.ManifestSession{
		Agent:      item.Agent,
		SessionID:  item.SessionID,
		SnapshotID: snapshotID,
		ProjectID:  item.ProjectID,
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	if err := e.updateManifest(ctx, sessionKey, entry, parent); err != nil {
		// The immutable snapshot intentionally remains as an orphan. A future
		// explicit maintenance command may garbage-collect it.
		return snapshotID, fmt.Errorf("snapshot uploaded but manifest update failed: %w", err)
	}
	return snapshotID, nil
}

func (e *Engine) platform() string {
	if e.Platform != "" {
		return e.Platform
	}
	return device.PlatformID()
}

func (e *Engine) maxPayloadSize() int64 {
	if e.MaxPayloadSize > 0 {
		return e.MaxPayloadSize
	}
	return defaultMaxPayload
}

func (e *Engine) key(relative string) string {
	relative = strings.TrimPrefix(relative, "/")
	prefix := strings.Trim(e.Prefix, "/")
	if prefix == "" {
		return relative
	}
	return prefix + "/" + relative
}

func (e *Engine) loadManifest(ctx context.Context, allowMissing bool) (*schema.Manifest, string, error) {
	rc, meta, err := e.Backend.Get(ctx, e.key("manifest.age"))
	if errors.Is(err, backend.ErrNotFound) {
		if !allowMissing {
			return nil, "", fmt.Errorf("%w at configured storage coordinates", ErrRemoteProfileNotFound)
		}
		return schema.NewManifest(""), "", nil
	}
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = rc.Close() }()

	plain, err := e.envelopeCodec().DecryptReader(rc, e.Passphrase)
	if err != nil {
		return nil, "", err
	}
	raw, err := io.ReadAll(io.LimitReader(plain, maxManifestBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(raw) > maxManifestBytes {
		return nil, "", fmt.Errorf("manifest exceeds maximum size")
	}
	var manifest schema.Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, "", fmt.Errorf("invalid manifest: %w", err)
	}
	if manifest.SchemaVersion != schema.ManifestSchemaVersion {
		return nil, "", fmt.Errorf("unsupported manifest schema_version %d", manifest.SchemaVersion)
	}
	if manifest.Sessions == nil {
		manifest.Sessions = map[string]schema.ManifestSession{}
	}
	return &manifest, meta.ETag, nil
}

func (e *Engine) saveManifest(ctx context.Context, manifest *schema.Manifest, ifMatch string) error {
	raw, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	var cipher bytes.Buffer
	if err := e.envelopeCodec().Encrypt(bytes.NewReader(raw), &cipher, e.Passphrase); err != nil {
		return err
	}
	opts := backend.PutOptions{ContentType: "application/octet-stream"}
	if ifMatch == "" {
		opts.IfNoneMatch = true
	} else {
		opts.IfMatch = ifMatch
	}
	_, err = e.Backend.Put(ctx, e.key("manifest.age"), bytes.NewReader(cipher.Bytes()), int64(cipher.Len()), opts)
	return err
}

func (e *Engine) updateManifest(ctx context.Context, sessionKey string, entry schema.ManifestSession, expectedParent string) error {
	for attempt := 0; attempt < maxManifestRetries; attempt++ {
		manifest, etag, err := e.loadManifest(ctx, !e.RequireRemoteManifest)
		if err != nil {
			return err
		}
		currentParent := ""
		if current, ok := manifest.Sessions[sessionKey]; ok {
			currentParent = current.SnapshotID
		}
		if currentParent != expectedParent {
			return fmt.Errorf("%w: remote session head changed from %q to %q", ErrConflict, expectedParent, currentParent)
		}

		manifest.Sessions[sessionKey] = entry
		manifest.Revision = entry.SnapshotID
		manifest.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		err = e.saveManifest(ctx, manifest, etag)
		if errors.Is(err, backend.ErrPrecondition) || errors.Is(err, backend.ErrAlreadyExists) {
			continue
		}
		return err
	}
	return fmt.Errorf("%w: manifest changed too many times", ErrConflict)
}

// PullArtifact authenticates, validates, and streams a snapshot artifact into a
// private destination directory. Dry-run performs every read and validation but
// does not create the artifact.
func (e *Engine) PullArtifact(ctx context.Context, item PullItem, destDir string, dryRun bool) (schema.Envelope, string, error) {
	var env schema.Envelope
	if e.Backend == nil {
		return env, "", fmt.Errorf("backend required")
	}
	rc, _, err := e.Backend.Get(ctx, e.key("snapshots/"+item.SnapshotID+".age"))
	if err != nil {
		return env, "", err
	}
	defer func() { _ = rc.Close() }()

	plain, err := e.envelopeCodec().DecryptReader(rc, e.Passphrase)
	if err != nil {
		return env, "", err
	}
	reader := bufio.NewReaderSize(plain, maxMetadataBytes+1)
	metaLine, err := reader.ReadSlice('\n')
	if errors.Is(err, bufio.ErrBufferFull) || len(metaLine) > maxMetadataBytes {
		return env, "", fmt.Errorf("snapshot metadata exceeds maximum size")
	}
	if err != nil {
		return env, "", fmt.Errorf("invalid snapshot metadata: %w", err)
	}
	if err := json.Unmarshal(bytes.TrimSuffix(metaLine, []byte{'\n'}), &env); err != nil {
		return env, "", fmt.Errorf("invalid snapshot metadata: %w", err)
	}
	if err := e.validateEnvelope(env, item); err != nil {
		return env, "", err
	}

	file := env.Files[0]
	if dryRun {
		if err := validatePayload(reader, io.Discard, file); err != nil {
			return env, "", err
		}
		return env, "", nil
	}
	if destDir == "" {
		return env, "", fmt.Errorf("destination directory required")
	}
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return env, "", err
	}
	dest := filepath.Join(destDir, filepath.Base(filepath.FromSlash(file.Path)))
	if err := writePayloadAtomic(dest, reader, file); err != nil {
		return env, "", err
	}
	return env, dest, nil
}

// PullSession preserves the original byte-returning API for focused tests.
// Production CLI code uses PullArtifact to avoid buffering session payloads.
func (e *Engine) PullSession(ctx context.Context, item PullItem, destDir string, dryRun bool) (schema.Envelope, []byte, error) {
	env, path, err := e.PullArtifact(ctx, item, destDir, dryRun)
	if err != nil || dryRun {
		return env, nil, err
	}
	payload, err := os.ReadFile(path)
	return env, payload, err
}

func (e *Engine) validateEnvelope(env schema.Envelope, item PullItem) error {
	if env.SchemaVersion != schema.EnvelopeSchemaVersion {
		return fmt.Errorf("unsupported envelope schema_version %d", env.SchemaVersion)
	}
	if env.Kind != schema.EnvelopeKind {
		return fmt.Errorf("unexpected snapshot kind %q", env.Kind)
	}
	if env.SnapshotID == "" || env.SnapshotID != item.SnapshotID {
		return fmt.Errorf("snapshot identity mismatch")
	}
	if item.Agent != "" && env.Agent != item.Agent {
		return fmt.Errorf("snapshot agent mismatch")
	}
	if item.SessionID != "" && env.SessionID != item.SessionID {
		return fmt.Errorf("snapshot session mismatch")
	}
	if item.ProjectID != "" && env.ProjectID != item.ProjectID {
		return fmt.Errorf("snapshot project mismatch")
	}
	if len(env.Files) != 1 {
		return fmt.Errorf("snapshot must contain exactly one artifact")
	}
	file := env.Files[0]
	clean := filepath.Clean(filepath.FromSlash(file.Path))
	if file.Path == "" || filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("unsafe snapshot path")
	}
	if containsHardExcludedPath(file.Path) {
		return fmt.Errorf("snapshot path is hard-excluded")
	}
	if file.Size < 0 || file.Size > e.maxPayloadSize() {
		return fmt.Errorf("invalid snapshot payload size")
	}
	decoded, err := hex.DecodeString(file.SHA256)
	if err != nil || len(decoded) != sha256.Size {
		return fmt.Errorf("invalid snapshot payload hash")
	}
	return nil
}

func validatePayload(source io.Reader, dest io.Writer, file schema.EnvelopeFile) error {
	hash := sha256.New()
	n, err := io.Copy(io.MultiWriter(dest, hash), io.LimitReader(source, file.Size+1))
	if err != nil {
		return err
	}
	if n != file.Size {
		return fmt.Errorf("snapshot payload size mismatch: got %d want %d", n, file.Size)
	}
	got := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(got, file.SHA256) {
		return fmt.Errorf("snapshot payload hash mismatch")
	}
	return nil
}

func writePayloadAtomic(dest string, source io.Reader, file schema.EnvelopeFile) error {
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".reinstate-pull-*")
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
	if err := validatePayload(source, tmp, file); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, dest); err != nil {
		return err
	}
	cleanup = false
	return nil
}

// FetchManifest returns the authenticated remote manifest.
func (e *Engine) FetchManifest(ctx context.Context) (*schema.Manifest, error) {
	manifest, _, err := e.loadManifest(ctx, !e.RequireRemoteManifest)
	return manifest, err
}

func isCredentialName(name string) bool {
	lower := strings.ToLower(name)
	if lower == ".env" || strings.HasPrefix(lower, ".env.") {
		return true
	}
	switch lower {
	case "auth.json", ".credentials.json", "credentials.json", "oauth.json",
		"tokens.json", "token.json", "secrets.json", "secret.json":
		return true
	default:
		return false
	}
}

func containsCredentialPath(path string) bool {
	for _, part := range strings.FieldsFunc(filepath.ToSlash(path), func(r rune) bool { return r == '/' }) {
		if isCredentialName(part) {
			return true
		}
	}
	return false
}

// containsHardExcludedPath reports credential material and the local-only
// handoffs/ store (never in push/pull scope for v0.4.0).
func containsHardExcludedPath(path string) bool {
	if containsCredentialPath(path) {
		return true
	}
	for _, part := range strings.FieldsFunc(filepath.ToSlash(path), func(r rune) bool { return r == '/' }) {
		if strings.EqualFold(part, "handoffs") {
			return true
		}
	}
	return false
}
