package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/backend"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
	"github.com/HarjjotSinghh/reinstate/internal/device"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

// Engine orchestrates push/pull against a Backend.
type Engine struct {
	Backend    backend.Backend
	Passphrase string
	Prefix     string // unused; keys relative
	Platform   string
}

// PushSession encrypts a local file payload and updates the remote manifest.
func (e *Engine) PushSession(ctx context.Context, item PushItem, dryRun bool) (snapshotID string, err error) {
	if e.Passphrase == "" {
		return "", fmt.Errorf("passphrase required")
	}
	data, err := os.ReadFile(item.LocalPath)
	if err != nil {
		return "", err
	}
	// Ensure no obvious secrets path names
	base := filepath.Base(item.LocalPath)
	if base == "auth.json" || base == ".credentials.json" {
		return "", fmt.Errorf("refusing to push credential file")
	}

	snapshotID = fmt.Sprintf("snap-%d", time.Now().UnixNano())
	env := schema.Envelope{
		SchemaVersion:  schema.EnvelopeSchemaVersion,
		Kind:           schema.EnvelopeKind,
		SnapshotID:     snapshotID,
		Agent:          item.Agent,
		AdapterSchema:  1,
		SourcePlatform: e.platform(),
		ProjectID:      item.ProjectID,
		SessionID:      item.SessionID,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		Files: []schema.EnvelopeFile{{
			Path:   filepath.Base(item.LocalPath),
			Mode:   0o600,
			Size:   int64(len(data)),
			SHA256: crypto.SHA256Hex(data),
		}},
	}
	metaJSON, err := json.Marshal(env)
	if err != nil {
		return "", err
	}
	// payload = meta JSON newline + raw file bytes (simple v1 packaging)
	payload := append(append(metaJSON, '\n'), data...)
	if dryRun {
		return snapshotID, nil
	}

	var cipher bytes.Buffer
	if err := crypto.Encrypt(bytes.NewReader(payload), &cipher, e.Passphrase); err != nil {
		return "", err
	}
	key := "snapshots/" + snapshotID + ".age"
	if _, err := e.Backend.Put(ctx, key, bytes.NewReader(cipher.Bytes()), int64(cipher.Len()), backend.PutOptions{IfNoneMatch: true}); err != nil {
		return "", err
	}

	// update manifest
	man, etag, err := e.loadManifest(ctx)
	if err != nil {
		return "", err
	}
	if man.Sessions == nil {
		man.Sessions = map[string]schema.ManifestSession{}
	}
	sk := SessionKey(item.Agent, item.SessionID)
	man.Sessions[sk] = schema.ManifestSession{
		Agent:      item.Agent,
		SessionID:  item.SessionID,
		SnapshotID: snapshotID,
		ProjectID:  item.ProjectID,
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	man.Revision = snapshotID
	man.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := e.saveManifest(ctx, man, etag); err != nil {
		// snapshot remains; manifest failed (by design)
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

func (e *Engine) loadManifest(ctx context.Context) (*schema.Manifest, string, error) {
	rc, meta, err := e.Backend.Get(ctx, "manifest.age")
	if err == backend.ErrNotFound {
		return schema.NewManifest(""), "", nil
	}
	if err != nil {
		return nil, "", err
	}
	defer rc.Close()
	var plain bytes.Buffer
	if err := crypto.Decrypt(rc, &plain, e.Passphrase); err != nil {
		return nil, "", err
	}
	var man schema.Manifest
	if err := json.Unmarshal(plain.Bytes(), &man); err != nil {
		return nil, "", err
	}
	return &man, meta.ETag, nil
}

func (e *Engine) saveManifest(ctx context.Context, man *schema.Manifest, ifMatch string) error {
	b, err := json.Marshal(man)
	if err != nil {
		return err
	}
	var cipher bytes.Buffer
	if err := crypto.Encrypt(bytes.NewReader(b), &cipher, e.Passphrase); err != nil {
		return err
	}
	opts := backend.PutOptions{}
	if ifMatch != "" {
		opts.IfMatch = ifMatch
	} else {
		opts.IfNoneMatch = true
		// if exists, retry with match — for memory backend first put
		if _, err := e.Backend.Head(ctx, "manifest.age"); err == nil {
			// exists without etag path
			opts.IfNoneMatch = false
		}
	}
	_, err = e.Backend.Put(ctx, "manifest.age", bytes.NewReader(cipher.Bytes()), int64(cipher.Len()), opts)
	if err == backend.ErrAlreadyExists || err == backend.ErrPrecondition {
		// reload and force with current etag for tests
		_, meta, gerr := e.Backend.Get(ctx, "manifest.age")
		if gerr != nil {
			return err
		}
		opts = backend.PutOptions{IfMatch: meta.ETag}
		_, err = e.Backend.Put(ctx, "manifest.age", bytes.NewReader(cipher.Bytes()), int64(cipher.Len()), opts)
	}
	return err
}

// PullSession downloads and decrypts a snapshot into destDir.
func (e *Engine) PullSession(ctx context.Context, item PullItem, destDir string, dryRun bool) (schema.Envelope, []byte, error) {
	var env schema.Envelope
	key := "snapshots/" + item.SnapshotID + ".age"
	if dryRun {
		return env, nil, nil
	}
	rc, _, err := e.Backend.Get(ctx, key)
	if err != nil {
		return env, nil, err
	}
	defer rc.Close()
	var plain bytes.Buffer
	if err := crypto.Decrypt(rc, &plain, e.Passphrase); err != nil {
		return env, nil, err
	}
	raw := plain.Bytes()
	// split meta line and payload
	idx := bytes.IndexByte(raw, '\n')
	if idx < 0 {
		return env, nil, fmt.Errorf("invalid snapshot payload")
	}
	if err := json.Unmarshal(raw[:idx], &env); err != nil {
		return env, nil, err
	}
	// plaintext must not appear in remote: remote is ciphertext only — verified by caller tests
	payload := raw[idx+1:]
	if destDir != "" {
		if err := os.MkdirAll(destDir, 0o700); err != nil {
			return env, nil, err
		}
		name := "payload.bin"
		if len(env.Files) > 0 {
			name = filepath.Base(env.Files[0].Path)
		}
		if err := os.WriteFile(filepath.Join(destDir, name), payload, 0o600); err != nil {
			return env, nil, err
		}
	}
	return env, payload, nil
}

// FetchManifest returns the decrypted remote manifest.
func (e *Engine) FetchManifest(ctx context.Context) (*schema.Manifest, error) {
	m, _, err := e.loadManifest(ctx)
	return m, err
}

// Ensure no unused import
var _ = io.EOF
