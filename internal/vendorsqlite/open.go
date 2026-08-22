// Package vendorsqlite opens a coding agent's embedded SQLite store for reading
// without writing anything under the agent's own directory.
//
// The two constraints are in tension. A vendor that journals in write-ahead
// mode keeps recent commits in a -wal sidecar and may not checkpoint them into
// the main database for a long time, so a reader that ignores the -wal cannot
// see the sessions the user just worked in. But opening a WAL database normally
// makes SQLite create a -shm sidecar next to it, and writing under a vendor's
// root is something this project does not do.
//
// Reading a private copy satisfies both.
package vendorsqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// walSuffix is the write-ahead log SQLite keeps beside a journalled database.
// The -shm file is deliberately never copied: it is a derived index into the
// -wal that SQLite rebuilds, and a stale one describes frames that may not be
// in the copy.
const walSuffix = "-wal"

// copyAttempts bounds the retry when the vendor writes during a copy. A vendor
// that is checkpointing continuously will exhaust this and fall back to the
// in-place read, which is correct but cannot see the log.
const copyAttempts = 3

// Handle is an open read-only view of a vendor store.
type Handle struct {
	DB *sql.DB
	// SawWAL reports whether this view includes an un-checkpointed write-ahead
	// log. False means the store had none, not that one was ignored.
	SawWAL bool
	// Incomplete reports that a write-ahead log exists but could not be read,
	// so records committed since the vendor last checkpointed are missing from
	// this view. A caller that lists sessions should say so rather than present
	// a short list as the whole store.
	Incomplete bool
	tempDir    string
}

// Close releases the handle and removes any private copy it made.
func (h *Handle) Close() error {
	var err error
	if h.DB != nil {
		err = h.DB.Close()
	}
	if h.tempDir != "" {
		if removeErr := os.RemoveAll(h.tempDir); err == nil {
			err = removeErr
		}
	}
	return err
}

// Open returns a read-only view of the store at path.
//
// With no write-ahead log the store is opened in place and immutable, which is
// both the cheapest option and the one that provably touches nothing. With a
// log present the database and its log are copied to a private directory and
// the copy is opened normally, so the log's contents are visible and the
// vendor's directory is still never written to.
func Open(path string) (*Handle, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("vendorsqlite: %s is not a regular file", path)
	}

	if walInfo, walErr := os.Stat(path + walSuffix); walErr != nil || walInfo.Size() == 0 {
		db, openErr := sql.Open("sqlite", immutableDSN(path))
		if openErr != nil {
			return nil, openErr
		}
		return &Handle{DB: db}, nil
	}

	handle, err := openCopy(path)
	if err == nil {
		return handle, nil
	}
	// A store being written to faster than it can be copied still has to be
	// readable. Falling back to the in-place view loses the log's contents,
	// which is exactly the behaviour that shipped before this package existed,
	// and never risks reporting a database assembled from two different moments.
	db, openErr := sql.Open("sqlite", immutableDSN(path))
	if openErr != nil {
		return nil, errors.Join(err, openErr)
	}
	return &Handle{DB: db, Incomplete: true}, nil
}

// openCopy copies the database and its log to a private directory.
//
// The copy is not atomic, and the hazard that creates is specific: if the
// vendor checkpoints between the two files being read, the log describes pages
// that the database copy does not have, and SQLite would assemble a database
// that never existed. The main file is therefore stat-ed before and after, and
// a change means the pair cannot be trusted together.
func openCopy(path string) (*Handle, error) {
	var lastErr error
	for attempt := 0; attempt < copyAttempts; attempt++ {
		before, err := os.Stat(path)
		if err != nil {
			return nil, err
		}

		tempDir, err := os.MkdirTemp("", "reinstate-vendorstore-")
		if err != nil {
			return nil, err
		}
		copyPath := filepath.Join(tempDir, filepath.Base(path))
		if err := copyFile(path, copyPath); err != nil {
			_ = os.RemoveAll(tempDir)
			return nil, err
		}
		if err := copyFile(path+walSuffix, copyPath+walSuffix); err != nil {
			_ = os.RemoveAll(tempDir)
			return nil, err
		}

		after, statErr := os.Stat(path)
		if statErr != nil {
			_ = os.RemoveAll(tempDir)
			return nil, statErr
		}
		if after.ModTime() != before.ModTime() || after.Size() != before.Size() {
			_ = os.RemoveAll(tempDir)
			lastErr = errors.New("vendorsqlite: store changed while it was being copied")
			continue
		}

		db, err := sql.Open("sqlite", copyDSN(copyPath))
		if err != nil {
			_ = os.RemoveAll(tempDir)
			return nil, err
		}
		return &Handle{DB: db, SawWAL: true, tempDir: tempDir}, nil
	}
	return nil, lastErr
}

func copyFile(source, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// immutableDSN opens a store in place. immutable=1 promises SQLite the file
// cannot change, which is what stops it taking a lock or creating a sidecar —
// and is also why it ignores the write-ahead log entirely.
func immutableDSN(path string) string {
	return "file:" + url.PathEscape(filepath.ToSlash(path)) + "?mode=ro&immutable=1&_pragma=query_only(1)"
}

// copyDSN opens a private copy. immutable is deliberately absent: reading the
// log is the whole point of having made the copy. The copy lives in a directory
// this process owns, so the -shm SQLite creates there harms nothing.
func copyDSN(path string) string {
	return "file:" + url.PathEscape(filepath.ToSlash(path)) + "?mode=ro&_pragma=query_only(1)"
}
