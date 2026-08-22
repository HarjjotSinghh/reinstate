package conformance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
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
		if err := checkNativeArgv(d); err != nil {
			return err
		}
	}
	// NativeSpec.NewSession is deliberately not required at T4. A destination
	// only needs it when the vendor lets the caller pin the new session's
	// identifier; Codex CLI assigns its own and reconciles it afterwards, so
	// requiring the field would make an existing, evidenced T5 descriptor
	// non-conformant. checkNativeArgv still validates the template when a
	// descriptor does declare one.
	return nil
}

// checkNativeArgv asserts that a session-addressing template actually addresses
// the session, and that a declared session-ID shape is enforceable.
//
// A template missing {{.SessionID}} would launch the vendor's most recent
// session instead of the resolved one, which is a wrong-session resume that no
// other check would notice.
func checkNativeArgv(d agents.Descriptor) error {
	const placeholder = "{{.SessionID}}"
	for name, template := range map[string][]string{
		"Resume":     d.Native.Resume,
		"Fork":       d.Native.Fork,
		"NewSession": d.Native.NewSession,
	} {
		if len(template) == 0 {
			continue
		}
		if !slices.ContainsFunc(template, func(part string) bool {
			return strings.Contains(part, placeholder)
		}) {
			return fmt.Errorf("NativeSpec.%s does not substitute %s", name, placeholder)
		}
	}
	if len(d.Native.Resume) == 0 {
		return errors.New("NativeSpec.Resume required at T3 and above")
	}
	pattern := strings.TrimSpace(d.Native.SessionIDPattern)
	if pattern == "" {
		return nil
	}
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("NativeSpec.SessionIDPattern does not compile: %w", err)
	}
	if !strings.HasPrefix(pattern, "^") || !strings.HasSuffix(pattern, "$") {
		return errors.New("NativeSpec.SessionIDPattern must be anchored with ^ and $")
	}
	// An unanchored or permissive pattern would let a session title through the
	// one gate that exists to stop it.
	for _, title := range []string{"", "fix the parser", "recent work", "../../etc/passwd"} {
		if compiled.MatchString(title) {
			return fmt.Errorf("NativeSpec.SessionIDPattern accepts the non-identifier %q", title)
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
		} else if gap := probePlatformGap(d.Evidence.ProbeReports); gap != "" {
			missing = append(missing, "ProbeReports on "+gap)
		}
		if len(d.Evidence.Fixtures) == 0 {
			missing = append(missing, "Fixtures")
		}
	}
	if d.Tier >= agents.TierResume {
		if len(d.Evidence.DeviceReports) == 0 {
			missing = append(missing, "DeviceReports")
		} else if gap := devicePlatformGap(d.Evidence.DeviceReports); gap != "" {
			missing = append(missing, "DeviceReports on "+gap)
		} else if gap := tierJourneyGap(d, repo); gap != "" {
			missing = append(missing, "a device journey for "+gap)
		}
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

// probePlatformGap names the platforms a T1 claim still lacks, or "" when both
// are present.
//
// docs/agent-support-tiers.md has always required a probe from macOS *and*
// native Windows at T1, but the check only counted reports, so a single macOS
// artifact satisfied it. That is the one place the ladder must not be an
// honour system: layouts diverge across platforms precisely where a scanner
// gets them wrong, and a swarm promoting agents in parallel will take whatever
// the code allows.
//
// Probe reports are named <date>-<macos|windows|wsl>-<agent>.json. WSL2 is a
// separate device with a separate tree and never substitutes for native
// Windows.
func probePlatformGap(reports []string) string {
	return platformGap(reports)
}

// legacyTierEvidence are whole-matrix device reports written before
// AGENT-TIER-JOURNEY-V1 existed.
//
// They carry no per-agent tier vocabulary at all: no T3, no T5, no per-agent
// matrix row identifiers. Nothing can read a rung out of them, so no rule
// applied to them would be measuring anything. They were the evidence standard
// when Claude Code and Codex CLI reached T5, and they are accepted as they
// stand rather than rewritten.
//
// Nothing may be added here. Every report written since names its agent and its
// rung in its own first heading, which is what tierJourneyGap reads.
var legacyTierEvidence = map[string]struct{}{
	"docs/testing/results/2026-08-11-macos-phase3-V030.md":       {},
	"docs/testing/results/2026-08-11-windows-phase3-V030.md":     {},
	"docs/testing/results/2026-08-15-macos-phase4-V040RC11.md":   {},
	"docs/testing/results/2026-08-15-windows-phase4-V040RC11.md": {},
}

// tierRungs lists every rung a claim at tier must evidence, from T3 upward.
// Below T3 no device journey is required at all.
func tierRungs(tier agents.Tier) []agents.Tier {
	var rungs []agents.Tier
	for rung := agents.TierResume; rung <= tier; rung++ {
		rungs = append(rungs, rung)
	}
	return rungs
}

// tierJourneyGap names what a T3+ claim cannot evidence, or "" when every rung
// from T3 up to the declared tier has a journey on both platforms.
//
// devicePlatformGap already refuses a claim whose reports do not span both
// platforms. It cannot tell what those reports are *about*, and that gap is not
// theoretical: Grok cited two release-acceptance reports that mention it only
// in index and handoff-source rows, and Qwen's T4 claim passed while its only
// Windows report covered T3, because a Windows filename was present either way.
//
// A journey report names its agent and its rung in its first heading, so the
// claim and the evidence can be compared directly.
func tierJourneyGap(d agents.Descriptor, repo string) string {
	if d.Tier < agents.TierResume {
		return ""
	}
	type coverage struct{ macOS, windows bool }
	covered := map[agents.Tier]*coverage{}
	for _, rung := range tierRungs(d.Tier) {
		covered[rung] = &coverage{}
	}

	for _, rel := range d.Evidence.DeviceReports {
		if _, legacy := legacyTierEvidence[rel]; legacy {
			// Accepted for every rung; see legacyTierEvidence.
			for _, c := range covered {
				name := strings.ToLower(filepath.Base(rel))
				switch {
				case strings.Contains(name, "-macos-"):
					c.macOS = true
				case strings.Contains(name, "-windows-"):
					c.windows = true
				}
			}
			continue
		}
		heading := firstHeading(repo, rel)
		if heading == "" || !strings.Contains(strings.ToLower(heading), strings.ToLower(d.DisplayName)) {
			continue
		}
		name := strings.ToLower(filepath.Base(rel))
		if strings.Contains(name, "-wsl-") {
			continue
		}
		for rung, c := range covered {
			if !strings.Contains(heading, rung.String()) {
				continue
			}
			switch {
			case strings.Contains(name, "-macos-"):
				c.macOS = true
			case strings.Contains(name, "-windows-"):
				c.windows = true
			}
		}
	}

	var missing []string
	for _, rung := range tierRungs(d.Tier) {
		c := covered[rung]
		switch {
		case !c.macOS && !c.windows:
			missing = append(missing, rung.String()+" on macos and windows")
		case !c.macOS:
			missing = append(missing, rung.String()+" on macos")
		case !c.windows:
			missing = append(missing, rung.String()+" on windows")
		}
	}
	return strings.Join(missing, ", ")
}

// firstHeading returns the report's first markdown heading, which is where a
// journey names its agent and its rung.
func firstHeading(repo, rel string) string {
	body, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rel)))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

// devicePlatformGap names the platforms a T3+ device-journey claim still lacks.
//
// docs/agent-support-tiers.md requires a physical device journey on macOS *and*
// native Windows before T3, and the project's non-negotiables say a tier is
// never claimed on one platform's evidence. checkEvidence enforced neither: it
// asserted only that DeviceReports was non-empty and that each named file
// existed, so a verified-resume or handoff-destination claim could rest on a
// single macOS journey and still pass.
//
// That is the same hole probePlatformGap was written to close one rung lower,
// and it matters more here, not less. T1 discovery misreads a layout; T3 hands
// a live session to a vendor binary and T4 writes a new one. Those are exactly
// the places platforms diverge — argv quoting, path separators, process
// identity — and none of it is visible from one device.
//
// Device reports are named <date>-<macos|windows|wsl>-<...>.md. WSL2 is a
// separate device with a separate tree and never substitutes for native
// Windows.
func devicePlatformGap(reports []string) string {
	return platformGap(reports)
}

// platformGap is the single implementation behind both callers. Two copies of
// this rule would be free to drift, and the drift would stay invisible until
// something was promoted on one platform's evidence — which is the failure this
// file exists to prevent.
func platformGap(reports []string) string {
	var macOS, windows bool
	for _, report := range reports {
		name := strings.ToLower(filepath.Base(report))
		if strings.Contains(name, "-wsl-") {
			// Checked before the platform tokens rather than left to fall
			// through them: a WSL report is a report about a third device, and
			// a name that happens to carry another token as well must not be
			// read as evidence from a device it did not come from.
			continue
		}
		switch {
		case strings.Contains(name, "-macos-"):
			macOS = true
		case strings.Contains(name, "-windows-"):
			windows = true
		}
	}
	switch {
	case !macOS && !windows:
		return "macos and windows"
	case !macOS:
		return "macos"
	case !windows:
		return "windows"
	}
	return ""
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
