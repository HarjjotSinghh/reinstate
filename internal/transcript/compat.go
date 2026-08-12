package transcript

import (
	"context"
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/agentcheck"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// The shared reader compatibility contract
//
// Every transcript reader answers Probe with the same two-part rule. Claude and
// Codex used to disagree — Claude read a `<agent-root>/version` file that real
// installations never create and failed closed when it was missing, while Codex
// checked no version at all — so identical situations produced different
// answers in the same invocation.
//
//  1. Layout is authoritative. An unrecognized layout is UNSUPPORTED, because
//     the reader genuinely cannot interpret those bytes.
//
//  2. Version is advisory and fails open. It is resolved from the installed
//     executable through internal/agentcheck, the single source of truth that
//     also backs `rein inspect`:
//
//     - determinable and inside the verified range  -> SUPPORTED
//     - determinable and outside the verified range -> UNTESTED
//     - not determinable                            -> SUPPORTED (layout only)
//
// The third case is the important one. A structured handoff reads a file that
// already exists on disk; it must keep working when the source agent is closed,
// logged out, rate limited, or uninstalled, which is precisely when a user
// reaches for one. Absence of version information is absence of evidence, not
// evidence of incompatibility, so it must not fail closed.
//
// This is deliberately the opposite of the sync adapters in internal/adapter,
// which fail closed on an unknown version: those write into a vendor tree, and
// writing blind into an unknown layout can destroy session state. Reading never
// can.
//
// Agents with no agentcheck definition (Gemini, OpenCode, Grok) can never
// resolve a version and are therefore always judged on layout alone. That is
// the same rule, not an exception.

// VersionResolver reports the installed source-agent version for a record.
// known=false means the version is not determinable on this device.
type VersionResolver func(ctx context.Context, rec sessionindex.Record) (version string, known bool)

// InstalledVersion is the production VersionResolver. It only probes when the
// record resolves to a real agent root, so a reader can never be tricked into
// describing some unrelated install — and never probes a contributor's ambient
// agent tree from a fixture-backed record.
func InstalledVersion(ctx context.Context, rec sessionindex.Record) (string, bool) {
	root := sessionindex.AgentRoot(rec)
	if root == "" {
		return "", false
	}
	return agentcheck.InstalledVersion(ctx, strings.ToLower(strings.TrimSpace(rec.Agent)), agentcheck.Options{
		Root:      root,
		Workspace: rec.Workspace,
	})
}

// probeCompatibility applies the shared contract documented above.
func probeCompatibility(
	ctx context.Context,
	rec sessionindex.Record,
	layoutRecognized bool,
	resolve VersionResolver,
) Compatibility {
	if !layoutRecognized {
		return CompatibilityUnsupported
	}
	if resolve == nil {
		resolve = InstalledVersion
	}
	version, known := resolve(ctx, rec)
	if !known {
		return CompatibilitySupported
	}
	if !agentcheck.SupportedVersion(rec.Agent, version) {
		return CompatibilityUntested
	}
	return CompatibilitySupported
}
