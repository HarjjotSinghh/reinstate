package conformance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/agents"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

var families = map[agents.StorageFamily]struct{}{
	agents.FamilyHomeTree:    {},
	agents.FamilyCLIQuery:    {},
	agents.FamilyEmbeddedDB:  {},
	agents.FamilyProjectFile: {},
	agents.FamilyRemote:      {},
}

func checkStructure(d agents.Descriptor) error {
	if !keyPattern.MatchString(d.Key) {
		return fmt.Errorf("key %q is not a stable lowercase catalog key", d.Key)
	}
	if strings.TrimSpace(d.DisplayName) == "" || strings.TrimSpace(d.Vendor) == "" {
		return errors.New("display name and vendor must be non-empty")
	}
	if _, ok := families[d.Family]; !ok {
		return fmt.Errorf("family %q is not F1–F5", d.Family)
	}
	if d.Tier == agents.TierKnown {
		if d.T0Reason == "" {
			return errors.New("T0Reason required at T0")
		}
	} else if d.T0Reason != "" {
		return fmt.Errorf("T0Reason %q is only valid at T0", d.T0Reason)
	}
	if d.Tier >= agents.TierResume {
		if d.Native == nil || strings.TrimSpace(d.Native.Executable) == "" {
			return errors.New("NativeSpec required at T3 and above")
		}
		if d.Version == nil || d.Version.Min == "" || d.Version.Max == "" {
			return errors.New("VersionSpec required at T3 and above")
		}
	}
	return nil
}

func checkCapability(d agents.Descriptor) error {
	want := map[string]bool{
		"NewIndexSource": d.Tier >= agents.TierDiscover,
		"NewReader":      d.Tier >= agents.TierHandoffFrom,
		"NewTarget":      d.Tier >= agents.TierHandoffTo,
		"NewSyncAdapter": d.Tier >= agents.TierSync,
	}
	have := map[string]bool{
		"NewIndexSource": d.NewIndexSource != nil,
		"NewReader":      d.NewReader != nil,
		"NewTarget":      d.NewTarget != nil,
		"NewSyncAdapter": d.NewSyncAdapter != nil,
	}
	var problems []string
	for name, required := range want {
		if required && !have[name] {
			problems = append(problems, name+" missing at "+d.Tier.String())
		}
		if !required && have[name] {
			problems = append(problems, name+" present above declared "+d.Tier.String())
		}
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func checkEvidence(d agents.Descriptor, repo string) error {
	var missing []string
	if strings.TrimSpace(d.Evidence.StoragePage) == "" {
		missing = append(missing, "StoragePage")
	} else if !exists(repo, d.Evidence.StoragePage) {
		missing = append(missing, d.Evidence.StoragePage)
	}
	if d.Tier >= agents.TierDiscover {
		if len(d.Evidence.ProbeReports) == 0 {
			missing = append(missing, "ProbeReports")
		}
		if len(d.Evidence.Fixtures) == 0 {
			missing = append(missing, "Fixtures")
		}
	}
	if d.Tier >= agents.TierResume && len(d.Evidence.DeviceReports) == 0 {
		missing = append(missing, "DeviceReports")
	}
	for _, path := range d.Evidence.ProbeReports {
		if !exists(repo, path) {
			missing = append(missing, path)
		}
	}
	for _, path := range d.Evidence.Fixtures {
		if !exists(repo, path) {
			missing = append(missing, path)
		}
	}
	for _, path := range d.Evidence.DeviceReports {
		if !exists(repo, path) {
			missing = append(missing, path)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required evidence: %s", strings.Join(missing, ", "))
	}
	return nil
}

func exists(repo, rel string) bool {
	_, err := os.Stat(filepath.Join(repo, filepath.FromSlash(rel)))
	return err == nil
}

func checkDeterminism(d agents.Descriptor, repo string, fixtures Fixtures) error {
	if d.NewIndexSource == nil {
		return nil
	}
	for _, root := range scanRoots(d, repo, fixtures) {
		first, err := scanOnce(d, root)
		if err != nil {
			return err
		}
		second, err := scanOnce(d, root)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(first.Records, second.Records) {
			return fmt.Errorf("two scans of %s produced different records", root)
		}
	}
	return nil
}

func checkIsolation(d agents.Descriptor, repo string, fixtures Fixtures) error {
	roots := scanRoots(d, repo, fixtures)
	if len(roots) == 0 {
		tmp := os.TempDir()
		wrapped, err := newIsolationFS(tmp)
		if err != nil {
			return err
		}
		return probeIsolationFS(wrapped)
	}
	for _, root := range roots {
		wrapped, err := newIsolationFS(root)
		if err != nil {
			return err
		}
		if err := probeIsolationFS(wrapped); err != nil {
			return err
		}
		result, err := scanOnce(d, root)
		if err != nil {
			return err
		}
		for _, record := range result.Records {
			if record.SourcePath == "" || strings.Contains(record.SourcePath, "://") {
				continue
			}
			abs, absErr := filepath.Abs(record.SourcePath)
			if absErr != nil {
				return absErr
			}
			if !withinRoot(root, abs) {
				return fmt.Errorf("%w: record %s source %s", errOutsideRoot, record.Key, abs)
			}
		}
	}
	return nil
}

func probeIsolationFS(wrapped *isolationFS) error {
	if _, err := wrapped.OpenFile("outside-write", os.O_WRONLY|os.O_CREATE, 0o600); !errors.Is(err, errWriteAttempt) {
		return fmt.Errorf("write open: %v", err)
	}
	if err := wrapped.Rename("a", "b"); !errors.Is(err, errWriteAttempt) {
		return fmt.Errorf("rename: %v", err)
	}
	if err := wrapped.Truncate("a", 0); !errors.Is(err, errWriteAttempt) {
		return fmt.Errorf("truncate: %v", err)
	}
	if err := wrapped.Lock("a"); !errors.Is(err, errWriteAttempt) {
		return fmt.Errorf("lock: %v", err)
	}
	outside := filepath.Join(filepath.Dir(wrapped.root), "conformance-outside")
	if _, err := wrapped.Open(outside); !errors.Is(err, errOutsideRoot) {
		return fmt.Errorf("outside open: %v", err)
	}
	return nil
}

func checkCorruption(d agents.Descriptor) error {
	if d.NewIndexSource == nil || d.Family != agents.FamilyHomeTree {
		return nil
	}
	cases := []struct {
		name string
		prep func(root string) error
		want func(sessionindex.ScanResult, error) error
	}{
		{
			name: "truncated_final_record",
			prep: func(root string) error {
				return writeSessionFile(d, root, []byte("{\"id\":\"ok\"}\n{\"id\":\"part"))
			},
			want: func(result sessionindex.ScanResult, err error) error {
				if err != nil {
					return err
				}
				for _, record := range result.Records {
					if strings.Contains(record.ID, "part") && !strings.Contains(record.ID, "ok") {
						return errors.New("partial trailing record was indexed")
					}
				}
				return nil
			},
		},
		{
			name: "invalid_utf8",
			prep: func(root string) error {
				return writeSessionFile(d, root, []byte("{\"id\":\"ok\"}\n\xff\xfe\n"))
			},
			want: func(result sessionindex.ScanResult, err error) error {
				if err != nil {
					return err
				}
				for _, record := range result.Records {
					if !utf8.ValidString(record.ID) || !utf8.ValidString(record.Title) {
						return errors.New("invalid UTF-8 leaked into a record")
					}
				}
				return nil
			},
		},
		{
			name: "empty_file",
			prep: func(root string) error {
				return writeSessionFile(d, root, nil)
			},
			want: func(_ sessionindex.ScanResult, err error) error {
				return err
			},
		},
		{
			name: "empty_directory",
			prep: func(root string) error {
				return writeMarkerOnly(d, root)
			},
			want: func(result sessionindex.ScanResult, err error) error {
				if err != nil {
					return err
				}
				if len(result.Records) != 0 {
					return fmt.Errorf("empty directory produced %d records", len(result.Records))
				}
				return nil
			},
		},
		{
			name: "absent_root",
			prep: func(string) error { return nil },
			want: func(result sessionindex.ScanResult, err error) error {
				if err != nil {
					return err
				}
				if len(result.Records) != 0 {
					return errors.New("absent root produced records")
				}
				return nil
			},
		},
		{
			name: "unknown_layout",
			prep: func(root string) error {
				return writeUnknownLayout(d, root)
			},
			want: func(result sessionindex.ScanResult, err error) error {
				if err != nil {
					return nil
				}
				if len(result.Records) != 0 {
					return errors.New("unknown layout produced records")
				}
				return nil
			},
		},
	}

	for _, tc := range cases {
		dir, err := os.MkdirTemp("", "rein-conformance-"+tc.name+"-*")
		if err != nil {
			return err
		}
		if err := tc.prep(dir); err != nil {
			_ = os.RemoveAll(dir)
			return fmt.Errorf("%s setup: %w", tc.name, err)
		}
		target := dir
		if tc.name == "absent_root" {
			target = filepath.Join(dir, "missing")
		}
		result, scanErr := scanOnce(d, target)
		checkErr := tc.want(result, scanErr)
		_ = os.RemoveAll(dir)
		if checkErr != nil {
			return fmt.Errorf("%s: %w", tc.name, checkErr)
		}
	}
	return nil
}

func checkPrivacy(d agents.Descriptor, repo string, fixtures Fixtures) error {
	if d.NewIndexSource == nil {
		return nil
	}
	for _, root := range scanRoots(d, repo, fixtures) {
		result, err := scanOnce(d, root)
		if err != nil {
			return err
		}
		for _, record := range result.Records {
			if utf8.RuneCountInString(record.PromptPreview) > sessionindex.PromptPreviewRunes {
				return fmt.Errorf("prompt preview for %s exceeds PromptPreviewRunes", record.Key)
			}
			if len(record.SearchText) > sessionindex.MaxSearchTextBytes {
				return fmt.Errorf("search text for %s exceeds MaxSearchTextBytes", record.Key)
			}
		}
	}
	return nil
}

func checkVersion(d agents.Descriptor) error {
	if d.Tier < agents.TierResume {
		return nil
	}
	if d.Version == nil || d.Version.Parse == nil {
		return errors.New("T3+ descriptor has no version parser")
	}
	outside := d.Version.Max + ".1"
	if adapter.StableVersionInRange(outside, d.Version.Min, d.Version.Max) {
		return fmt.Errorf("version %s is outside %s–%s but still in range", outside, d.Version.Min, d.Version.Max)
	}
	parsed, ok := d.Version.Parse(agents.VersionOutput{Stdout: "not-a-version\n"})
	if ok {
		return fmt.Errorf("unparseable version yielded %q", parsed)
	}
	if adapter.StableVersionInRange(parsed, d.Version.Min, d.Version.Max) {
		return errors.New("unparseable version is treated as SUPPORTED")
	}
	return nil
}

func checkReadOnly(d agents.Descriptor, repo string, fixtures Fixtures) error {
	if d.Tier >= agents.TierResume || d.NewIndexSource == nil {
		return nil
	}
	for _, root := range scanRoots(d, repo, fixtures) {
		result, err := scanOnce(d, root)
		if err != nil {
			return err
		}
		for _, record := range result.Records {
			if strings.TrimSpace(record.ReadOnlyReason) == "" {
				return fmt.Errorf("record %s below T3 has empty ReadOnlyReason", record.Key)
			}
		}
	}
	return nil
}

func scanRoots(d agents.Descriptor, repo string, fixtures Fixtures) []string {
	if d.Family == agents.FamilyCLIQuery {
		return nil
	}
	return fixtureRoots(repo, fixtures)
}

func scanOnce(d agents.Descriptor, root string) (sessionindex.ScanResult, error) {
	source, err := d.NewIndexSource(agents.Env{FixtureRoot: root})
	if err != nil {
		return sessionindex.ScanResult{}, err
	}
	return source.Scan(context.Background())
}

func writeMarkerOnly(d agents.Descriptor, root string) error {
	marker := strings.TrimSpace(d.Storage.Marker)
	if marker == "" {
		return nil
	}
	return os.MkdirAll(filepath.Join(root, filepath.FromSlash(marker)), 0o755)
}

func writeSessionFile(d agents.Descriptor, root string, body []byte) error {
	if err := writeMarkerOnly(d, root); err != nil {
		return err
	}
	path, err := sessionFilePath(d, root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func sessionFilePath(d agents.Descriptor, root string) (string, error) {
	switch d.Family {
	case agents.FamilyHomeTree:
		switch d.Storage.Marker {
		case "projects":
			return filepath.Join(root, "projects", "fixture", "session.jsonl"), nil
		case "sessions":
			if strings.Contains(d.Storage.SessionGlob, "summary.json") {
				return filepath.Join(root, "sessions", "ws", "sid", "summary.json"), nil
			}
			return filepath.Join(root, "sessions", "session.jsonl"), nil
		case "tmp":
			return filepath.Join(root, "tmp", "proj", "chats", "session-syn.jsonl"), nil
		default:
			return filepath.Join(root, "session.jsonl"), nil
		}
	default:
		return "", fmt.Errorf("no session file mapping for family %s", d.Family)
	}
}

func writeUnknownLayout(d agents.Descriptor, root string) error {
	if strings.Contains(d.Storage.SessionGlob, "summary.json") {
		body := []byte(`{"info":{"id":"x","cwd":"/tmp"},"chat_format_version":99}` + "\n")
		return writeSessionFile(d, root, body)
	}
	// Unrecognized extension / layout token so the glob does not match.
	if err := writeMarkerOnly(d, root); err != nil {
		return err
	}
	path := filepath.Join(root, d.Storage.Marker, "unknown.layout")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("layout=unknown\n"), 0o644)
}
