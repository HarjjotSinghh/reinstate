package agents

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/adapter"
	"github.com/HarjjotSinghh/reinstate/internal/handoff"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

// Existing read ceilings, reused so a new agent cannot introduce an unbounded
// read. Values are owned by sessionindex; do not invent new ones here.
const (
	MaxJSONLineBytes   = sessionindex.MaxJSONLineBytes
	MaxSearchTextBytes = sessionindex.MaxSearchTextBytes
	MaxFileReferences  = sessionindex.MaxFileReferences
	// DefaultMaxArgvBytes is the conservative Windows-safe argv ceiling.
	// NativeSpec.MaxArgvBytes of 0 uses this value.
	DefaultMaxArgvBytes = handoff.DefaultMaxArgvBytes
)

// Descriptor is the complete, self-contained description of one coding agent.
// Exactly one Descriptor exists per agent, in internal/agents/catalog/<key>.go.
type Descriptor struct {
	Key         string // stable lowercase key, e.g. "kimi"; used in agent:session refs
	DisplayName string // "Kimi Code CLI"
	Vendor      string // "Moonshot AI"
	DocsURL     string // vendor documentation root

	Tier     Tier          // highest tier this agent has evidence for
	Family   StorageFamily // F1..F5
	T0Reason T0Reason      // required when Tier == TierKnown, empty otherwise

	Storage StorageSpec
	Native  *NativeSpec  // required at T3 and above
	Version *VersionSpec // required at T3 and above
	Process ProcessSpec

	Evidence Evidence

	// Capability constructors. A nil constructor means the agent does not
	// provide that capability. The conformance suite asserts these agree
	// exactly with Tier: no capability above the declared tier, and no
	// missing capability below it.
	NewIndexSource func(Env) (sessionindex.Source, error)   // T1+
	NewReader      func(Env) (transcript.Reader, error)     // T2+
	NewTarget      func(Env) (handoff.HandoffTarget, error) // T4+
	NewSyncAdapter func(Env) (adapter.Adapter, error)       // T5
}

// Tier is one rung on the support ladder. Tiers are cumulative.
type Tier int

const (
	TierKnown       Tier = iota // T0
	TierDiscover                // T1
	TierHandoffFrom             // T2
	TierResume                  // T3
	TierHandoffTo               // T4
	TierSync                    // T5
)

// String returns the public tier token (T0..T5).
func (t Tier) String() string {
	switch t {
	case TierKnown:
		return "T0"
	case TierDiscover:
		return "T1"
	case TierHandoffFrom:
		return "T2"
	case TierResume:
		return "T3"
	case TierHandoffTo:
		return "T4"
	case TierSync:
		return "T5"
	default:
		return "T?"
	}
}

// StorageFamily is how an agent's sessions are stored.
type StorageFamily string

const (
	FamilyHomeTree    StorageFamily = "F1" // JSON/JSONL tree under a home root
	FamilyCLIQuery    StorageFamily = "F2" // vendor CLI is the read API
	FamilyEmbeddedDB  StorageFamily = "F3" // SQLite or editor extension storage
	FamilyProjectFile StorageFamily = "F4" // per-repository files
	FamilyRemote      StorageFamily = "F5" // server-backed or desktop-only
)

// T0Reason is a closed enumeration so rein doctor output stays machine readable.
type T0Reason string

const (
	T0NoLocalHistory             T0Reason = "no_local_history"
	T0ServerBacked               T0Reason = "server_backed"
	T0DesktopOnly                T0Reason = "desktop_only"
	T0UnidentifiedProduct        T0Reason = "unidentified_product"
	T0UnofficialDistributionOnly T0Reason = "unofficial_distribution_only"
	T0LayoutUnverified           T0Reason = "layout_unverified"
)

// StorageSpec describes how to find an agent's local session artifacts.
type StorageSpec struct {
	RootEnv     string                    // "KIMI_CODE_HOME"; empty when none
	Roots       func(home HomeDir) []Root // ordered candidates, first match wins
	Marker      string                    // relative path that must exist for the root to count
	Layout      string                    // stable layout id, e.g. "sessions-workdir-wire-jsonl"
	SessionGlob string                    // relative glob below the root
	ProjectKey  ProjectKeyKind            // how the vendor derives its project bucket
	Excluded    []string                  // subtrees never read (credentials, caches, subagents)
}

// HomeDir is a resolved user-home path used to expand per-OS root candidates.
type HomeDir string

// String returns the home path.
func (h HomeDir) String() string { return string(h) }

// Join appends path elements under the home directory.
func (h HomeDir) Join(elem ...string) string {
	parts := make([]string, 0, 1+len(elem))
	parts = append(parts, string(h))
	parts = append(parts, elem...)
	return filepath.Join(parts...)
}

// Root is one candidate storage root. Native Windows and WSL2 are separate
// devices with separate trees.
type Root struct {
	Path string
	// OS is "macos", "windows", "wsl", or empty to match every OS.
	OS string
}

// Matches reports whether this candidate applies to osName.
func (r Root) Matches(osName string) bool {
	return r.OS == "" || r.OS == osName
}

// CurrentOS is the catalog's OS token for this process: macos, windows, or
// linux. WSL is a separate device and is never inferred from GOOS alone.
func CurrentOS() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos"
	case "windows":
		return "windows"
	default:
		return runtime.GOOS
	}
}

// ProjectKeyKind records the vendor's project-bucketing scheme so pathmap can
// recompute the destination key rather than reuse the source device's.
type ProjectKeyKind string

const (
	ProjectKeyNone        ProjectKeyKind = "none"
	ProjectKeyPathSlug    ProjectKeyKind = "path_slug"
	ProjectKeyPathHash    ProjectKeyKind = "path_hash"
	ProjectKeyURLEncoding ProjectKeyKind = "url_encoding"
	ProjectKeyOpaqueID    ProjectKeyKind = "opaque_id"
)

// NativeSpec is the vendor's documented launch argv, required at T3.
type NativeSpec struct {
	Executable    string   // "kimi"
	Resume        []string // argv template, {{.SessionID}} substituted
	Fork          []string // nil when the vendor has no fork
	Continue      []string // nil when the vendor has no continue
	NewSession    []string // T4 only
	InitialPrompt PromptMode
	MaxArgvBytes  int // 0 uses DefaultMaxArgvBytes
}

// PromptMode is how a destination accepts an initial prompt.
type PromptMode string

const (
	PromptNone  PromptMode = ""
	PromptArgv  PromptMode = "argv"
	PromptStdin PromptMode = "stdin"
	PromptFile  PromptMode = "file"
)

// VersionSpec is the existing agentcheck definition shape, moved into the descriptor.
type VersionSpec struct {
	Args     []string // {"--version"}
	Parse    func(VersionOutput) (string, bool)
	Min, Max string // inclusive, fail-closed range
}

// VersionOutput keeps the two process streams separate so a warning written to
// stderr cannot be mistaken for authoritative version output.
type VersionOutput struct {
	Stdout string
	Stderr string
}

// ProcessSpec describes how to recognize a running vendor process.
type ProcessSpec struct {
	// Images are executable basenames (without .exe) that identify the agent.
	Images []string
	// NodeMarkers are command-line fragments that identify a node-hosted
	// distribution of the agent.
	NodeMarkers []string
	// Identify is vendor self-identification environment variables. Prefer
	// these over image heuristics when any entry is set.
	Identify []EnvIdentity
}

// EnvIdentity is one vendor-declared process environment pair.
type EnvIdentity struct {
	Name  string
	Value string // empty means any non-empty value
}

// Evidence is the committed artifacts that justify the declared tier.
// Every path must exist in the repository; conformance enforces this.
type Evidence struct {
	StoragePage   string   // docs/session-storage/<key>.md — always required
	ProbeReports  []string // docs/testing/results/agent-probes/… — required at T1+
	Fixtures      []string // testdata/… roots — required at T1+
	DeviceReports []string // docs/testing/results/… — required at T3+
}

// Env carries the resolved home directory, environment lookups, and fixture
// root overrides, so every constructor is testable against testdata/ without
// touching a real machine.
type Env struct {
	Home        string
	LookupEnv   func(string) string
	FixtureRoot string
}

// Lookup returns an environment value. Nil LookupEnv uses the process environment.
func (e Env) Lookup(key string) string {
	if e.LookupEnv != nil {
		return e.LookupEnv(key)
	}
	return os.Getenv(key)
}

// HomeDir returns the resolved home, or the process user home when unset.
func (e Env) HomeDir() (HomeDir, error) {
	if strings.TrimSpace(e.Home) != "" {
		return HomeDir(e.Home), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return HomeDir(home), nil
}

// Capability is one catalog-derived consumer surface.
type Capability string

const (
	CapabilityIndex       Capability = "index"
	CapabilityHandoffFrom Capability = "handoff_from"
	CapabilityResume      Capability = "resume"
	CapabilityHandoffTo   Capability = "handoff_to"
	CapabilitySync        Capability = "sync"
)

// ArgvBudget returns NativeSpec.MaxArgvBytes, or DefaultMaxArgvBytes when unset.
func (n *NativeSpec) ArgvBudget() int {
	if n == nil || n.MaxArgvBytes <= 0 {
		return DefaultMaxArgvBytes
	}
	return n.MaxArgvBytes
}

func (d Descriptor) hasCapability(c Capability) bool {
	switch c {
	case CapabilityIndex:
		return d.NewIndexSource != nil
	case CapabilityHandoffFrom:
		return d.NewReader != nil
	case CapabilityResume:
		return d.Native != nil
	case CapabilityHandoffTo:
		return d.NewTarget != nil
	case CapabilitySync:
		return d.NewSyncAdapter != nil
	default:
		return false
	}
}

func constructorMinTier(name string) (Tier, bool) {
	switch name {
	case "NewIndexSource":
		return TierDiscover, true
	case "NewReader":
		return TierHandoffFrom, true
	case "NewTarget":
		return TierHandoffTo, true
	case "NewSyncAdapter":
		return TierSync, true
	default:
		return 0, false
	}
}

func (d Descriptor) constructorsAboveTier() []string {
	var above []string
	for _, name := range []string{"NewIndexSource", "NewReader", "NewTarget", "NewSyncAdapter"} {
		min, ok := constructorMinTier(name)
		if !ok {
			continue
		}
		present := false
		switch name {
		case "NewIndexSource":
			present = d.NewIndexSource != nil
		case "NewReader":
			present = d.NewReader != nil
		case "NewTarget":
			present = d.NewTarget != nil
		case "NewSyncAdapter":
			present = d.NewSyncAdapter != nil
		}
		if present && d.Tier < min {
			above = append(above, name)
		}
	}
	return above
}
