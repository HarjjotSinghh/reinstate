package transcript

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// MaxJSONLineBytes bounds one vendor JSONL record during boundary discovery.
// It matches sessionindex.MaxJSONLineBytes so readers share one ceiling.
const MaxJSONLineBytes = sessionindex.MaxJSONLineBytes

const defaultReaderSize = 64 << 10

// Boundary freezes an immutable, complete-record prefix of a vendor transcript.
//
// ByteOffset is the end of the last newline-terminated parseable JSONL record.
// SHA256 digests exactly bytes [0, ByteOffset). Path is private and never
// serialized — use Path() for local re-reads of the frozen prefix.
//
// Snapshot semantics: opening is read-only; the source is never locked, renamed,
// truncated, or written. Appending to the live file after Snapshot does not
// mutate a returned Boundary value; PrefixReader and DigestPrefix always honor
// the frozen ByteOffset, so the digest remains stable for that prefix.
type Boundary struct {
	Agent      string `json:"agent"`
	SessionID  string `json:"session_id"`
	ByteOffset int64  `json:"byte_offset"` // end of the last complete record
	SizeBytes  int64  `json:"size_bytes"`  // full file size at snapshot time
	SHA256     string `json:"sha256"`      // digest of bytes [0, ByteOffset)
	ModTimeNS  int64  `json:"mod_time_ns"`
	Partial    bool   `json:"partial,omitempty"` // true when SizeBytes > ByteOffset
	path       string // never serialized
	paths      PathContext
}

// Path returns the private absolute source path frozen into the boundary.
func (b Boundary) Path() string {
	return b.path
}

// WithPathContext freezes the roots used to tokenize vendor paths during Parse.
// Readers attach it in Snapshot, where the source record is still available.
func (b Boundary) WithPathContext(paths PathContext) Boundary {
	b.paths = paths
	return b
}

// PathContext returns the tokenization roots frozen into the boundary.
func (b Boundary) PathContext() PathContext {
	return b.paths
}

// SnapshotJSONL opens path read-only and freezes the last complete JSONL record
// boundary. A trailing partial line (no terminating newline, or bytes after the
// last parseable record) is excluded and sets Partial.
//
// maxLineBytes bounds each record; values <= 0 use MaxJSONLineBytes.
func SnapshotJSONL(path, agent, sessionID string, maxLineBytes int) (Boundary, error) {
	if path == "" {
		return Boundary{}, errors.New("transcript: snapshot path must not be empty")
	}
	if maxLineBytes <= 0 {
		maxLineBytes = MaxJSONLineBytes
	}

	file, err := os.Open(path) // read-only; never locks or writes
	if err != nil {
		return Boundary{}, fmt.Errorf("transcript: open snapshot: %w", err)
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return Boundary{}, fmt.Errorf("transcript: stat snapshot: %w", err)
	}
	size := info.Size()
	modTimeNS := info.ModTime().UnixNano()

	offset, err := completeJSONLOffset(file, maxLineBytes)
	if err != nil {
		return Boundary{}, err
	}

	digest, err := digestPrefix(path, offset)
	if err != nil {
		return Boundary{}, err
	}

	return Boundary{
		Agent:      agent,
		SessionID:  sessionID,
		ByteOffset: offset,
		SizeBytes:  size,
		SHA256:     digest,
		ModTimeNS:  modTimeNS,
		Partial:    size > offset,
		path:       path,
	}, nil
}

// DigestPrefix recomputes the SHA-256 of bytes [0, b.ByteOffset) at b.Path().
// Appending after the frozen offset does not change this digest.
func DigestPrefix(b Boundary) (string, error) {
	if b.path == "" {
		return "", errors.New("transcript: boundary path is empty")
	}
	if b.ByteOffset < 0 {
		return "", fmt.Errorf("transcript: negative byte offset %d", b.ByteOffset)
	}
	return digestPrefix(b.path, b.ByteOffset)
}

// PrefixReader returns a read-closer limited to bytes [0, ByteOffset).
// Callers must close it. The live file may grow; only the frozen prefix is read.
func PrefixReader(b Boundary) (io.ReadCloser, error) {
	if b.path == "" {
		return nil, errors.New("transcript: boundary path is empty")
	}
	if b.ByteOffset < 0 {
		return nil, fmt.Errorf("transcript: negative byte offset %d", b.ByteOffset)
	}
	file, err := os.Open(b.path)
	if err != nil {
		return nil, fmt.Errorf("transcript: open prefix: %w", err)
	}
	return &prefixReadCloser{
		Reader: io.LimitReader(file, b.ByteOffset),
		closer: file,
	}, nil
}

// VisitCompleteJSONL visits each newline-terminated JSON value inside the
// frozen prefix. Trailing partial content outside ByteOffset is never surfaced.
func VisitCompleteJSONL(b Boundary, maxLineBytes int, visit func(lineNumber int, line []byte) error) ([]Warning, error) {
	if visit == nil {
		return nil, errors.New("transcript: JSONL visitor must not be nil")
	}
	if maxLineBytes <= 0 {
		maxLineBytes = MaxJSONLineBytes
	}
	reader, err := PrefixReader(b)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	// Reuse the shared JSONL scanner; the LimitReader already excludes the
	// partial tail, so incomplete_trailing_record should not appear for a
	// correctly frozen boundary that ends on a record newline.
	return sessionindex.ScanJSONLines(reader, maxLineBytes, visit)
}

type prefixReadCloser struct {
	io.Reader
	closer io.Closer
}

func (p *prefixReadCloser) Close() error {
	return p.closer.Close()
}

func digestPrefix(path string, offset int64) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("transcript: open for digest: %w", err)
	}
	defer func() { _ = file.Close() }()

	h := sha256.New()
	written, err := io.Copy(h, io.LimitReader(file, offset))
	if err != nil {
		return "", fmt.Errorf("transcript: digest prefix: %w", err)
	}
	if written != offset {
		return "", fmt.Errorf("transcript: digest read %d bytes, want %d", written, offset)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// completeJSONLOffset returns the byte end of the last newline-terminated
// parseable JSON record. Oversized or malformed newline-terminated lines are
// skipped without updating the offset; a trailing partial line is excluded.
func completeJSONLOffset(r io.Reader, maxLineBytes int) (int64, error) {
	buffered := bufio.NewReaderSize(r, min(maxLineBytes+1, defaultReaderSize))
	var (
		pos          int64
		lastComplete int64
	)
	for {
		line, complete, oversized, n, err := readBoundedLineAt(buffered, maxLineBytes)
		pos += int64(n)
		if n == 0 && errors.Is(err, io.EOF) {
			return lastComplete, nil
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return 0, err
		}

		if complete {
			trimmed := trimJSONLSpace(line)
			if !oversized && len(trimmed) > 0 && json.Valid(trimmed) {
				lastComplete = pos
			}
		}
		// Incomplete trailing bytes (EOF without '\n') never advance lastComplete.

		if errors.Is(err, io.EOF) {
			return lastComplete, nil
		}
	}
}

// readBoundedLineAt mirrors sessionindex line bounding while also returning how
// many source bytes were consumed (including a terminating newline when present).
func readBoundedLineAt(reader *bufio.Reader, limit int) (line []byte, complete, oversized bool, consumed int, err error) {
	var buf []byte
	discarded := false
	for {
		fragment, readErr := reader.ReadSlice('\n')
		consumed += len(fragment)
		if len(buf) <= limit {
			available := limit + 1 - len(buf)
			if available > 0 {
				if len(fragment) > available {
					buf = append(buf, fragment[:available]...)
					discarded = true
				} else {
					buf = append(buf, fragment...)
				}
			}
			if available <= 0 && len(fragment) > 0 {
				discarded = true
			}
		} else if len(fragment) > 0 {
			discarded = true
		}
		switch {
		case readErr == nil:
			if len(buf) > 0 && buf[len(buf)-1] == '\n' {
				buf = buf[:len(buf)-1]
			}
			return buf, true, discarded || len(buf) > limit, consumed, nil
		case errors.Is(readErr, bufio.ErrBufferFull):
			continue
		case errors.Is(readErr, io.EOF):
			return buf, false, discarded || len(buf) > limit, consumed, io.EOF
		default:
			return buf, false, discarded || len(buf) > limit, consumed, readErr
		}
	}
}

func trimJSONLSpace(line []byte) []byte {
	start, end := 0, len(line)
	for start < end {
		c := line[start]
		if c != ' ' && c != '\t' && c != '\r' {
			break
		}
		start++
	}
	for end > start {
		c := line[end-1]
		if c != ' ' && c != '\t' && c != '\r' {
			break
		}
		end--
	}
	return line[start:end]
}
