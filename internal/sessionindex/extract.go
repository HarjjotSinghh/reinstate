package sessionindex

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/HarjjotSinghh/reinstate/internal/environment"
)

const (
	// PromptPreviewRunes is the maximum human preview length.
	PromptPreviewRunes = 160
	// MaxJSONLineBytes bounds one vendor JSONL event.
	MaxJSONLineBytes = 4 << 20
	// MaxSearchTextBytes bounds private searchable content for one session.
	MaxSearchTextBytes = 256 << 10
	// MaxFileReferences bounds structured file metadata for one session.
	MaxFileReferences = 512
	// MaxFileReferenceRunes bounds one structured file reference.
	MaxFileReferenceRunes = 4096
)

const (
	maxTitleRunes     = 512
	maxMetadataRunes  = 4096
	maxReadOnlyRunes  = 512
	defaultReaderSize = 64 << 10
)

// SafePreview returns a metadata-safe prompt preview.
func SafePreview(value string) string {
	return SafeText(value, PromptPreviewRunes)
}

// SafeText removes terminal escape sequences, C0/C1 controls, Unicode format
// controls, and collapses whitespace. A positive maxRunes truncates without
// splitting UTF-8.
func SafeText(value string, maxRunes int) string {
	value = strings.ToValidUTF8(value, "")
	value = stripTerminalSequences(value)

	var output strings.Builder
	output.Grow(min(len(value), 4096))
	wroteSpace := false
	runes := 0
	for _, current := range value {
		if unicode.IsSpace(current) {
			if output.Len() > 0 {
				wroteSpace = true
			}
			continue
		}
		if unicode.IsControl(current) || unicode.In(current, unicode.Cf) {
			continue
		}
		if wroteSpace {
			if maxRunes > 0 && runes >= maxRunes {
				break
			}
			output.WriteByte(' ')
			runes++
			wroteSpace = false
		}
		if maxRunes > 0 && runes >= maxRunes {
			break
		}
		output.WriteRune(current)
		runes++
	}
	return strings.TrimSpace(output.String())
}

// BuildSearchText constructs bounded private literal-search content.
func BuildSearchText(parts ...string) string {
	var output strings.Builder
	for _, part := range parts {
		part = SafeText(part, 0)
		if part == "" {
			continue
		}
		if output.Len() > 0 {
			if output.Len()+1 > MaxSearchTextBytes {
				break
			}
			output.WriteByte('\n')
		}
		remaining := MaxSearchTextBytes - output.Len()
		if remaining <= 0 {
			break
		}
		part = truncateUTF8Bytes(part, remaining)
		output.WriteString(part)
		if len(part) == remaining {
			break
		}
	}
	return output.String()
}

// NormalizeFiles sanitizes, bounds, deduplicates, and sorts structured file
// references for deterministic storage and output.
func NormalizeFiles(files []string) []string {
	if len(files) == 0 {
		return nil
	}
	unique := make(map[string]struct{}, min(len(files), MaxFileReferences))
	for _, file := range files {
		if len(unique) >= MaxFileReferences {
			break
		}
		file = SafeText(file, MaxFileReferenceRunes)
		if file == "" {
			continue
		}
		unique[file] = struct{}{}
	}
	normalized := make([]string, 0, len(unique))
	for file := range unique {
		normalized = append(normalized, file)
	}
	sort.Strings(normalized)
	return normalized
}

// NormalizeRecord validates identity and applies all derived-data bounds before
// a record reaches SQLite.
func NormalizeRecord(record Record) (Record, error) {
	record.Agent = strings.ToLower(SafeText(strings.TrimSpace(record.Agent), 64))
	record.ID = SafeText(strings.TrimSpace(record.ID), maxMetadataRunes)
	if record.Agent == "" {
		return Record{}, errors.New("session agent must not be empty")
	}
	if strings.Contains(record.Agent, ":") {
		return Record{}, errors.New("session agent must not contain ':'")
	}
	if record.ID == "" {
		return Record{}, errors.New("session ID must not be empty")
	}

	expectedKey := CompositeReference(record.Agent, record.ID)
	if record.Key == "" {
		record.Key = expectedKey
	} else {
		record.Key = SafeText(strings.TrimSpace(record.Key), maxMetadataRunes+65)
		if record.Key != expectedKey {
			return Record{}, fmt.Errorf("session key %q does not match %q", record.Key, expectedKey)
		}
	}

	record.Title = SafeText(record.Title, maxTitleRunes)
	record.Project = SafeText(record.Project, maxMetadataRunes)
	record.Workspace = SafeText(record.Workspace, maxMetadataRunes)
	record.Branch = SafeText(record.Branch, maxMetadataRunes)
	record.PromptPreview = SafePreview(record.PromptPreview)
	record.ReadOnlyReason = SafeText(record.ReadOnlyReason, maxReadOnlyRunes)
	record.Files = NormalizeFiles(record.Files)
	record.UpdatedAt = record.UpdatedAt.UTC()
	var err error
	record.RecordedEnvironment, err = environment.NormalizeRecordedEnvironment(record.RecordedEnvironment)
	if err != nil {
		return Record{}, fmt.Errorf("normalize recorded environment: %w", err)
	}

	if record.SizeBytes < 0 || record.SourceSize < 0 {
		return Record{}, errors.New("session sizes must not be negative")
	}
	if record.MessageCount < 0 {
		return Record{}, errors.New("session message count must not be negative")
	}
	if record.CanFork && !record.CanResume {
		return Record{}, errors.New("a fork-capable session must also be resumable")
	}
	if !record.CanResume && record.ReadOnlyReason == "" {
		record.ReadOnlyReason = "read-only session source"
	}

	fileParts := make([]string, 0, len(record.Files)+7)
	fileParts = append(fileParts, record.SearchText, record.Key, record.ID, record.Agent)
	fileParts = append(fileParts, record.Title, record.Project, record.Workspace, record.Branch)
	fileParts = append(fileParts, record.Files...)
	record.SearchText = BuildSearchText(fileParts...)
	return record, nil
}

// CoalesceRecords folds multiple local files for one native session ID into a
// single deterministic index record. Vendors may create a fresh rollout file
// when a native session is resumed; that is one session, not an ambiguity or a
// source-wide refresh failure.
func CoalesceRecords(records []Record) ([]Record, []Warning) {
	if len(records) < 2 {
		return records, nil
	}
	groups := make(map[string][]Record, len(records))
	var keys []string
	for _, record := range records {
		key := record.Reference()
		if _, exists := groups[key]; !exists {
			keys = append(keys, key)
		}
		groups[key] = append(groups[key], record)
	}
	sort.Strings(keys)

	coalesced := make([]Record, 0, len(keys))
	var warnings []Warning
	for _, key := range keys {
		segments := groups[key]
		if len(segments) == 1 {
			coalesced = append(coalesced, segments[0])
			continue
		}
		sort.SliceStable(segments, func(i, j int) bool {
			if !segments[i].UpdatedAt.Equal(segments[j].UpdatedAt) {
				return segments[i].UpdatedAt.Before(segments[j].UpdatedAt)
			}
			if segments[i].SourceModTime != segments[j].SourceModTime {
				return segments[i].SourceModTime < segments[j].SourceModTime
			}
			return segments[i].SourcePath < segments[j].SourcePath
		})
		record := mergeRecordSegments(segments)
		coalesced = append(coalesced, record)
		warnings = append(warnings, Warning{
			Agent:     record.Agent,
			SessionID: record.ID,
			Code:      "coalesced_session_segments",
			Message: fmt.Sprintf(
				"coalesced %d local records for one native session",
				len(segments),
			),
		})
	}
	SortRecords(coalesced)
	return coalesced, warnings
}

func mergeRecordSegments(segments []Record) Record {
	record := segments[len(segments)-1]
	var (
		files       []string
		searchParts []string
		fingerprint = sha256.New()
	)
	record.SizeBytes = 0
	record.MessageCount = 0
	record.SourceSize = 0
	record.PromptPreview = ""
	record.CanResume = false
	record.CanFork = false
	record.RecordedEnvironment = environment.RecordedEnvironment{}

	for _, segment := range segments {
		record.SizeBytes = saturatingAddInt64(record.SizeBytes, segment.SizeBytes)
		record.SourceSize = saturatingAddInt64(record.SourceSize, segment.SourceSize)
		record.MessageCount = saturatingAddInt(record.MessageCount, segment.MessageCount)
		if record.PromptPreview == "" && segment.PromptPreview != "" {
			record.PromptPreview = segment.PromptPreview
		}
		record.CanResume = record.CanResume || segment.CanResume
		record.CanFork = record.CanFork || segment.CanFork
		files = append(files, segment.Files...)
		searchParts = append(searchParts, segment.SearchText)
		if segment.SourceModTime > record.SourceModTime {
			record.SourceModTime = segment.SourceModTime
		}
		_, _ = fingerprint.Write([]byte(strconv.Quote(segment.SourcePath)))
		_, _ = fingerprint.Write([]byte{
			':',
		})
		_, _ = fingerprint.Write([]byte(strconv.FormatInt(segment.SourceModTime, 10)))
		_, _ = fingerprint.Write([]byte{
			':',
		})
		_, _ = fingerprint.Write([]byte(strconv.FormatInt(segment.SourceSize, 10)))
		_, _ = fingerprint.Write([]byte{
			'\n',
		})
	}
	record.Files = NormalizeFiles(files)
	record.SearchText = BuildSearchText(searchParts...)
	record.SourcePath = "aggregate://" + record.Agent + "/" + record.ID + "/" +
		fmt.Sprintf("%x", fingerprint.Sum(nil))
	if record.CanFork && !record.CanResume {
		record.CanResume = true
	}
	if record.CanResume {
		record.ReadOnlyReason = ""
	}

	// Prefer the newest non-empty metadata, but do not replace a useful vendor
	// title with the native-ID fallback from a later segment.
	for index := len(segments) - 1; index >= 0; index-- {
		segment := segments[index]
		mergeRecordedEnvironment(&record.RecordedEnvironment, segment.RecordedEnvironment)
		if record.Workspace == "" && segment.Workspace != "" {
			record.Workspace = segment.Workspace
		}
		if record.Project == "" && segment.Project != "" {
			record.Project = segment.Project
		}
		if record.Branch == "" && segment.Branch != "" {
			record.Branch = segment.Branch
		}
		if segment.Title != "" && segment.Title != segment.ID {
			record.Title = segment.Title
			break
		}
	}
	return record
}

func mergeRecordedEnvironment(target *environment.RecordedEnvironment, source environment.RecordedEnvironment) {
	if target.RepositoryID.Value == "" && source.RepositoryID.Value != "" {
		target.RepositoryID = source.RepositoryID
	}
	if target.Branch.Value == "" && source.Branch.Value != "" {
		target.Branch = source.Branch
	}
	if target.GitHead.Value == "" && source.GitHead.Value != "" {
		target.GitHead = source.GitHead
	}
	if len(source.Requirements) != 0 {
		target.Requirements = append(target.Requirements, source.Requirements...)
		normalized, err := environment.NormalizeRecordedEnvironment(*target)
		if err == nil {
			*target = normalized
		}
	}
}

func saturatingAddInt64(left, right int64) int64 {
	const maxInt64 = int64(^uint64(0) >> 1)
	if right > 0 && left > maxInt64-right {
		return maxInt64
	}
	return left + right
}

func saturatingAddInt(left, right int) int {
	maxInt := int(^uint(0) >> 1)
	if right > 0 && left > maxInt-right {
		return maxInt
	}
	return left + right
}

// ScanJSONLines visits complete, valid JSON values while bounding individual
// records. A malformed or oversized record is skipped with a sanitized warning;
// an invalid final partial line is treated as a concurrent append and ignored.
func ScanJSONLines(
	reader io.Reader,
	maxLineBytes int,
	visit func(lineNumber int, line []byte) error,
) ([]Warning, error) {
	if reader == nil {
		return nil, errors.New("JSONL reader must not be nil")
	}
	if visit == nil {
		return nil, errors.New("JSONL visitor must not be nil")
	}
	if maxLineBytes <= 0 {
		maxLineBytes = MaxJSONLineBytes
	}

	buffered := bufio.NewReaderSize(reader, min(maxLineBytes+1, defaultReaderSize))
	var warnings []Warning
	lineNumber := 0
	for {
		lineNumber++
		line, complete, oversized, err := readBoundedLine(buffered, maxLineBytes)
		if err != nil && !errors.Is(err, io.EOF) {
			return warnings, err
		}
		line = bytes.TrimSpace(line)
		if oversized {
			warnings = append(warnings, Warning{
				Code:    "oversized_record",
				Message: fmt.Sprintf("ignored JSONL record %d larger than %d bytes", lineNumber, maxLineBytes),
			})
		} else if len(line) > 0 {
			if !json.Valid(line) {
				code := "malformed_record"
				message := fmt.Sprintf("ignored malformed JSONL record %d", lineNumber)
				if !complete && errors.Is(err, io.EOF) {
					code = "incomplete_trailing_record"
					message = fmt.Sprintf("ignored incomplete trailing JSONL record %d", lineNumber)
				}
				warnings = append(warnings, Warning{Code: code, Message: message})
			} else if visitErr := visit(lineNumber, line); visitErr != nil {
				return warnings, visitErr
			}
		}
		if errors.Is(err, io.EOF) {
			return warnings, nil
		}
	}
}

func readBoundedLine(reader *bufio.Reader, limit int) ([]byte, bool, bool, error) {
	var line []byte
	discarded := false
	for {
		fragment, err := reader.ReadSlice('\n')
		if len(line) <= limit {
			available := limit + 1 - len(line)
			if available > 0 {
				if len(fragment) > available {
					line = append(line, fragment[:available]...)
					discarded = true
				} else {
					line = append(line, fragment...)
				}
			}
			if available <= 0 && len(fragment) > 0 {
				discarded = true
			}
		} else if len(fragment) > 0 {
			discarded = true
		}
		switch {
		case err == nil:
			line = bytes.TrimSuffix(line, []byte{'\n'})
			return line, true, discarded || len(line) > limit, nil
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case errors.Is(err, io.EOF):
			return line, false, discarded || len(line) > limit, io.EOF
		default:
			return line, false, discarded || len(line) > limit, err
		}
	}
}

func truncateUTF8Bytes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if len(value) <= limit {
		return value
	}
	value = value[:limit]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}

func stripTerminalSequences(value string) string {
	const (
		stateText = iota
		stateEscape
		stateCSI
		stateString
		stateStringEscape
	)

	state := stateText
	var output strings.Builder
	output.Grow(len(value))
	for _, current := range value {
		switch state {
		case stateText:
			switch current {
			case '\x1b':
				state = stateEscape
			case '\u009b':
				state = stateCSI
			case '\u0090', '\u009d', '\u009e', '\u009f':
				state = stateString
			default:
				output.WriteRune(current)
			}
		case stateEscape:
			switch current {
			case '[':
				state = stateCSI
			case ']', 'P', 'X', '^', '_':
				state = stateString
			default:
				state = stateText
			}
		case stateCSI:
			if current >= 0x40 && current <= 0x7e {
				state = stateText
			}
		case stateString:
			switch current {
			case '\a':
				state = stateText
			case '\x1b':
				state = stateStringEscape
			}
		case stateStringEscape:
			if current == '\\' {
				state = stateText
			} else if current != '\x1b' {
				state = stateString
			}
		}
	}
	return output.String()
}
