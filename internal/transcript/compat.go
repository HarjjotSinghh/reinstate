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
//  2. Version is advisory and fails open where information is genuinely
//     absent. It is resolved from the installed executable through
//     internal/agentcheck, the single source of truth that also backs
//     `rein inspect`:
//
//     - determinable and inside the verified range  -> SUPPORTED
//     - determinable and outside the verified range -> UNTESTED
//     - not determinable                            -> SUPPORTED (layout only)
//     - installed, but the version probe failed     -> UNTESTED
//
// The third case is the important one. A structured handoff reads a file that
// already exists on disk; it must keep working when the source agent is closed,
// logged out, rate limited, or uninstalled, which is precisely when a user
// reaches for one. Absence of version information is absence of evidence, not
// evidence of incompatibility, so it must not fail closed.
//
// The fourth case is not the third. When the agent is installed and its version
// probe fails — it timed out even after a retry, would not execute, or the
// executable changed underneath it — a version does exist and Reinstate failed
// to read it. Treating that as "not determinable" collapses row two into row
// three: an out-of-range install passes as unknown and is silently accepted,
// which is a fail-open on the one gate that exists to refuse untested versions.
// A failed measurement is uncertainty, so it is UNTESTED and the user can still
// proceed deliberately with --allow-untested.
//
// This is deliberately the opposite of the sync adapters in internal/adapter,
// which fail closed on an unknown version: those write into a vendor tree, and
// writing blind into an unknown layout can destroy session state. Reading never
// can.
//
// Agents with no agentcheck definition (Gemini, Grok) can never resolve a
// version and are therefore always judged on layout alone. That is the same
// rule, not an exception.
//
// The corollary bites when an agent gains a definition: OpenCode did, at T3,
// and a handoff from an OpenCode build outside the verified range is now
// UNTESTED where it was previously judged on layout alone. That is the contract
// working, not a regression in it — but the verified range starts at a single
// physically measured build, so it will report UNTESTED for most installs until
// more builds are measured on a device.

// VersionResolver reports the installed source-agent version for a record,
// together with how much the resolution actually established. The evidence is
// not a nicety: "no version to read" and "could not read the version" must lead
// to different compatibility answers, and a resolver that returns only a
// version cannot express the difference.
type VersionResolver func(ctx context.Context, rec sessionindex.Record) (version string, evidence agentcheck.VersionEvidence)

// InstalledVersion is the production VersionResolver. It only probes when the
// record resolves to a real agent root, so a reader can never be tricked into
// describing some unrelated install — and never probes a contributor's ambient
// agent tree from a fixture-backed record.
func InstalledVersion(ctx context.Context, rec sessionindex.Record) (string, agentcheck.VersionEvidence) {
	root := sessionindex.AgentRoot(rec)
	if root == "" {
		return "", agentcheck.VersionUnavailable
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
	version, evidence := resolve(ctx, rec)
	switch evidence {
	case agentcheck.VersionProbeFailed:
		// Installed, but unread. Uncertainty about a present agent is not the
		// same as an absent one and must not pass as supported.
		return CompatibilityUntested
	case agentcheck.VersionDetermined:
		if !agentcheck.SupportedVersion(rec.Agent, version) {
			return CompatibilityUntested
		}
		return CompatibilitySupported
	default:
		return CompatibilitySupported
	}
}
