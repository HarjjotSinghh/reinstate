package doctor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/adapter/claude"
	"github.com/HarjjotSinghh/reinstate/internal/backend/memory"
	"github.com/HarjjotSinghh/reinstate/internal/crypto"
	"github.com/HarjjotSinghh/reinstate/internal/fsx"
	syncengine "github.com/HarjjotSinghh/reinstate/internal/sync"
)

// SelfTest runs a synthetic encryption + atomic write check using only temp data.
// It never reads real vendor sessions or credentials.
func SelfTest(home string) error {
	return selfTest(home, nil)
}

func selfTest(home string, codec syncengine.EnvelopeCodec) error {
	dir := filepath.Join(home, "cache", "selftest")
	if err := fsx.EnsureOwnerOnlyDir(dir); err != nil {
		// fall back to system temp if home not writable
		dir = filepath.Join(os.TempDir(), "reinstate-selftest")
		if err := fsx.EnsureOwnerOnlyDir(dir); err != nil {
			return err
		}
	}
	plain := []byte("reinstate-synthetic-self-test-payload-v1")
	pass := "test-passphrase-not-real"
	var buf bytes.Buffer
	var encryptErr error
	if codec == nil {
		encryptErr = crypto.Encrypt(bytes.NewReader(plain), &buf, pass)
	} else {
		encryptErr = codec.Encrypt(bytes.NewReader(plain), &buf, pass)
	}
	if encryptErr != nil {
		return fmt.Errorf("encrypt: %w", encryptErr)
	}
	cipher := buf.Bytes()
	if bytes.Contains(cipher, plain) {
		return fmt.Errorf("ciphertext contains plaintext")
	}
	var out bytes.Buffer
	var decryptErr error
	if codec == nil {
		decryptErr = crypto.Decrypt(bytes.NewReader(cipher), &out, pass)
	} else {
		var reader io.Reader
		reader, decryptErr = codec.DecryptReader(bytes.NewReader(cipher), pass)
		if decryptErr == nil {
			_, decryptErr = io.Copy(&out, reader)
		}
	}
	if decryptErr != nil {
		return fmt.Errorf("decrypt: %w", decryptErr)
	}
	if !bytes.Equal(out.Bytes(), plain) {
		return fmt.Errorf("round-trip mismatch")
	}

	sourceRoot := filepath.Join(dir, "source-claude")
	sourceProject := filepath.Join(sourceRoot, "projects", "synthetic")
	if err := os.MkdirAll(sourceProject, 0o700); err != nil {
		return err
	}
	sourcePath := filepath.Join(sourceProject, "selftest-session.jsonl")
	if err := fsx.WriteFileAtomic(sourcePath, []byte(
		`{"type":"meta","cwd":"/synthetic/project"}`+"\n"+
			`{"type":"user","message":{"content":"reinstate self-test"}}`+"\n",
	), 0o600); err != nil {
		return err
	}
	sourceAdapter := &claude.Adapter{Root: sourceRoot}
	sessions, err := sourceAdapter.Discover(context.Background(), adapter.DiscoverOptions{})
	if err != nil || len(sessions) != 1 {
		return fmt.Errorf("adapter discovery: sessions=%d err=%w", len(sessions), err)
	}
	exportPath := filepath.Join(dir, "selftest-export.tar")
	exportFile, err := os.OpenFile(exportPath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	exportPlan, err := sourceAdapter.PlanExport(context.Background(), sessions[0], adapter.ExportOptions{})
	if err != nil {
		_ = exportFile.Close()
		return err
	}
	if err := sourceAdapter.Export(context.Background(), exportPlan, exportFile); err != nil {
		_ = exportFile.Close()
		return fmt.Errorf("adapter export: %w", err)
	}
	if err := exportFile.Close(); err != nil {
		return err
	}

	store := memory.New()
	engine := &syncengine.Engine{Backend: store, Passphrase: pass, Codec: codec}
	snapshotID, err := engine.PushSession(context.Background(), syncengine.PushItem{
		Agent: "claude", SessionID: sessions[0].ID, ProjectID: sessions[0].ProjectID,
		LocalPath: exportPath, RelativePath: sessions[0].RelativePath,
	}, false)
	if err != nil {
		return fmt.Errorf("sync push: %w", err)
	}
	pullDir := filepath.Join(dir, "pull")
	envelope, artifactPath, err := engine.PullArtifact(context.Background(), syncengine.PullItem{
		Agent: "claude", SessionID: sessions[0].ID,
		SnapshotID: snapshotID, ProjectID: sessions[0].ProjectID,
	}, pullDir, false)
	if err != nil {
		return fmt.Errorf("sync pull: %w", err)
	}

	targetRoot := filepath.Join(dir, "target-claude")
	if err := os.MkdirAll(filepath.Join(targetRoot, "projects"), 0o700); err != nil {
		return err
	}
	targetAdapter := &claude.Adapter{Root: targetRoot}
	restorePlan, err := targetAdapter.PlanRestore(context.Background(), adapter.Snapshot{
		ID: envelope.SnapshotID, Agent: envelope.Agent, SessionID: envelope.SessionID,
		ProjectID: envelope.ProjectID, RelativePath: envelope.Files[0].Path,
	}, adapter.RestoreOptions{BackupRoot: filepath.Join(dir, "backups")})
	if err != nil {
		return fmt.Errorf("restore plan: %w", err)
	}
	artifact, err := os.Open(artifactPath)
	if err != nil {
		return err
	}
	restoreErr := targetAdapter.Restore(context.Background(), restorePlan, artifact)
	closeErr := artifact.Close()
	if restoreErr != nil {
		return fmt.Errorf("adapter restore: %w", restoreErr)
	}
	if closeErr != nil {
		return closeErr
	}
	restored, err := targetAdapter.Discover(context.Background(), adapter.DiscoverOptions{})
	if err != nil || len(restored) != 1 || restored[0].ID != sessions[0].ID {
		return fmt.Errorf("post-restore discovery failed")
	}
	_ = os.Remove(exportPath)
	_ = os.Remove(artifactPath)
	return nil
}
