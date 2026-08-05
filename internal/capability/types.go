// Package capability discovers privacy-safe, name-only declarations for
// coding-agent instructions, skills, and MCP servers.
//
// Discovery is deliberately passive: it never executes agent configuration,
// follows symlinks, or returns configuration contents and filesystem paths.
package capability

const (
	maxConfigBytes = int64(1 << 20)
	maxEntries     = 256
	maxDepth       = 8
	maxNameRunes   = 128
)

// Agent identifies a supported coding agent.
type Agent string

const (
	AgentClaude Agent = "claude"
	AgentCodex  Agent = "codex"
)

// Kind identifies the declaration family.
type Kind string

const (
	KindInstruction Kind = "instruction"
	KindSkill       Kind = "skill"
	KindMCP         Kind = "mcp"
)

// Scope describes the non-sensitive logical location of a declaration.
type Scope string

const (
	ScopeManaged Scope = "managed"
	ScopeUser    Scope = "user"
	ScopeProject Scope = "project"
	ScopeLocal   Scope = "local"
)

// State is intentionally conservative. Static discovery never claims that a
// declaration was loaded, approved, authenticated, connected, or healthy.
type State string

const (
	StateCandidate  State = "candidate"
	StateDeclared   State = "declared"
	StateUnverified State = "unverified"
)

// Transport is a privacy-safe MCP transport classification. It never contains
// a command, URL, argument, header, environment value, or parser output.
type Transport string

const (
	TransportUnknown Transport = "unknown"
	TransportStdio   Transport = "stdio"
	TransportHTTP    Transport = "http"
	TransportSSE     Transport = "sse"
)

// SourceKind is a path-free description of the declaration format.
type SourceKind string

const (
	SourceClaudeMemory       SourceKind = "claude_memory"
	SourceClaudeRule         SourceKind = "claude_rule"
	SourceClaudeSkill        SourceKind = "claude_skill"
	SourceClaudeLegacyCmd    SourceKind = "claude_legacy_command"
	SourceClaudeMCPJSON      SourceKind = "claude_mcp_json"
	SourceClaudeStateJSON    SourceKind = "claude_state_json"
	SourceClaudeManagedMCP   SourceKind = "claude_managed_mcp_json"
	SourceCodexInstruction   SourceKind = "codex_instruction"
	SourceCodexSkill         SourceKind = "codex_skill"
	SourceCodexMCPConfigTOML SourceKind = "codex_config_toml"
)

// Item is the complete public discovery record. It contains no source path,
// content, description, command, URL, environment, authentication data, or
// parser output.
type Item struct {
	Agent      Agent      `json:"agent"`
	Kind       Kind       `json:"kind"`
	Name       string     `json:"name"`
	Scope      Scope      `json:"scope"`
	State      State      `json:"state"`
	SourceKind SourceKind `json:"source_kind"`
	Transport  Transport  `json:"transport,omitempty"`
	Lazy       bool       `json:"lazy"`
}

// VerifiedPresence reports whether discovery verified a regular local file or
// parsed declaration. Unverified symlink names are useful diagnostics, but
// must never satisfy capability preflight.
func (i Item) VerifiedPresence() bool {
	return i.State == StateCandidate || i.State == StateDeclared
}

// DiagnosticCode is a bounded, path-free reason why discovery was incomplete.
type DiagnosticCode string

const (
	DiagnosticInvalidRoot   DiagnosticCode = "invalid_root"
	DiagnosticMalformed     DiagnosticCode = "malformed"
	DiagnosticOversized     DiagnosticCode = "oversized"
	DiagnosticReadFailed    DiagnosticCode = "read_failed"
	DiagnosticSymlink       DiagnosticCode = "symlink_skipped"
	DiagnosticUnsafePath    DiagnosticCode = "unsafe_path"
	DiagnosticLimitReached  DiagnosticCode = "limit_reached"
	DiagnosticUnsupportedOS DiagnosticCode = "unsupported_os"
	DiagnosticCancelled     DiagnosticCode = "cancelled"
)

// Diagnostic never contains an error string or filesystem path. In
// particular, malformed input cannot echo secrets through parser errors.
type Diagnostic struct {
	Agent Agent          `json:"agent,omitempty"`
	Kind  Kind           `json:"kind,omitempty"`
	Scope Scope          `json:"scope,omitempty"`
	Code  DiagnosticCode `json:"code"`
}

// Inventory is deterministic: Items and Diagnostics are sorted and deduped.
type Inventory struct {
	Items       []Item       `json:"items"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}

// Options supplies verified roots without consulting process environment.
//
// ManagedRoot is the filesystem root used for platform-managed locations. It
// is normally "/" on macOS and a volume root such as `C:\` on Windows. Tests
// can point it at a disposable directory while injecting GOOS.
//
// ProjectRoot and WorkingDir must be absolute, and WorkingDir must be within
// ProjectRoot. Project declarations are not scanned otherwise. ClaudeHome and
// CodexHome may be omitted to use UserHome/.claude and UserHome/.codex.
type Options struct {
	GOOS        string `json:"goos,omitempty"`
	UserHome    string `json:"-"`
	ClaudeHome  string `json:"-"`
	CodexHome   string `json:"-"`
	ProjectRoot string `json:"-"`
	WorkingDir  string `json:"-"`
	ManagedRoot string `json:"-"`
}
