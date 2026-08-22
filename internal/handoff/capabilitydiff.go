package handoff

import (
	"sort"
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/capability"
	"github.com/HarjjotSinghh/reinstate/internal/capsule"
)

// Impact levels for destination capability gaps.
const (
	ImpactBlocking      = "blocking"
	ImpactDegraded      = "degraded"
	ImpactInformational = "informational"
)

// Missing kinds reported in a capability diff.
const (
	KindToolFamily  = "tool_family"
	KindMCP         = "mcp"
	KindSkill       = "skill"
	KindInstruction = "instruction"
	KindAttachment  = "attachment"
	KindContext     = "context"
)

const (
	summaryOmitted = "omitted"
	// contextCeilingReason is the R7 answer: no harness-level token ceiling is
	// published for Claude Code or Codex CLI in vendor docs Reinstate trusts.
	contextCeilingReason = "no_vendor_published_harness_token_ceiling"
)

// Missing is one destination gap produced by DiffCapabilities.
type Missing struct {
	Kind   string // tool_family | mcp | skill | instruction | attachment | context
	Name   string
	Impact string // blocking | degraded | informational
}

// WarningID returns the stable acknowledgement ID for a missing capability.
// Format: handoff.capability.<kind>.<name>
func WarningID(m Missing) string {
	return "handoff.capability." + safeCapabilityIDPart(m.Kind) + "." + safeCapabilityIDPart(m.Name)
}

// CapabilityWarningIDs returns sorted warning IDs for every Missing entry.
func CapabilityWarningIDs(diff capsule.CapabilityDiff) []string {
	ids := make([]string, 0, len(diff.Missing))
	for _, m := range diff.Missing {
		ids = append(ids, WarningID(Missing{Kind: m.Kind, Name: m.Name, Impact: m.Impact}))
	}
	sort.Strings(ids)
	return ids
}

// DiffCapabilities compares source and destination inventories plus published
// agent-level traits. Only VerifiedPresence() inventory items count as present.
//
// Missing entries are sorted by kind, then name. Warning IDs derived from them
// are therefore stable.
func DiffCapabilities(source, destination capability.Inventory, srcAgent, dstAgent string) capsule.CapabilityDiff {
	srcAgent = normalizeAgent(srcAgent)
	dstAgent = normalizeAgent(dstAgent)

	srcPresent := verifiedNames(source, srcAgent)
	dstPresent := verifiedNames(destination, dstAgent)

	// "Absent from the destination inventory" only means the destination lacks
	// it when the destination was actually enumerated. discoverInventory covers
	// Claude Code and Codex; every other destination arrives with an empty
	// inventory, and reporting each source capability as degraded there would
	// assert a gap nobody looked for.
	gapImpact := ImpactDegraded
	if !capabilityDiscoverySupported(dstAgent) {
		gapImpact = ImpactInformational
	}
	var missing []Missing
	for _, kind := range []string{KindInstruction, KindMCP, KindSkill} {
		names := sortedKeys(srcPresent[kind])
		for _, name := range names {
			if _, ok := dstPresent[kind][name]; ok {
				continue
			}
			missing = append(missing, Missing{
				Kind:   kind,
				Name:   name,
				Impact: gapImpact,
			})
		}
	}

	srcProfile := publishedProfile(srcAgent)
	dstProfile := publishedProfile(dstAgent)
	// Approval modes and multi-root appear in Source/Destination summaries when
	// published; Missing kinds are limited to the contract set (no approval /
	// multi_root kind), so gaps there stay summary-only until documented.
	missing = append(missing, diffPublishedSets(KindToolFamily, srcProfile.ToolFamilies, dstProfile.ToolFamilies)...)
	missing = append(missing, diffTriState(KindAttachment, "support", srcProfile.Attachments, dstProfile.Attachments)...)
	missing = append(missing, diffContextCeiling(srcProfile, dstProfile)...)

	sort.SliceStable(missing, func(i, j int) bool {
		if missing[i].Kind != missing[j].Kind {
			return missing[i].Kind < missing[j].Kind
		}
		return missing[i].Name < missing[j].Name
	})

	out := capsule.CapabilityDiff{
		Source:      summarize(srcAgent, srcPresent, srcProfile),
		Destination: summarize(dstAgent, dstPresent, dstProfile),
	}
	if len(missing) > 0 {
		out.Missing = make([]capsule.MissingCapability, len(missing))
		for i, m := range missing {
			out.Missing[i] = capsule.MissingCapability{Kind: m.Kind, Name: m.Name, Impact: m.Impact}
		}
	}
	return out
}

type triState int

const (
	triUnknown triState = iota
	triNo
	triYes
)

type agentProfile struct {
	ToolFamilies  map[string]struct{} // nil => unpublished
	ApprovalModes map[string]struct{} // nil => unpublished
	Attachments   triState
	MultiRoot     triState
	// ContextCeilingTokens is nil when unpublished (R7).
	ContextCeilingTokens *int
}

// capabilityDiscoverySupported reports whether Reinstate can enumerate an
// agent's instructions, MCP servers, and skills. It must stay in step with
// discoverInventory in pipeline.go.
func capabilityDiscoverySupported(agent string) bool {
	switch normalizeAgent(agent) {
	case "claude", "codex":
		return true
	default:
		return false
	}
}

func publishedProfile(agent string) agentProfile {
	// Inventory discovery covers MCP/skills/instructions. Agent-level traits
	// below are published only when Reinstate has a vendor-documented claim.
	// Unknown traits stay omitted — never guessed (R7 and related rules).
	switch agent {
	case "claude":
		// Attachments: Claude Code image blocks are documented in
		// docs/session-storage-map.md (R8). Tool families, approval modes,
		// multi-root, and harness context ceilings are not published here.
		return agentProfile{Attachments: triYes}
	case "codex":
		// No vendor-published harness attachment / multi-root / tool-family /
		// approval-mode / context-ceiling tables are recorded for Codex yet.
		return agentProfile{}
	case "qwen":
		// Qwen Code accepts images interactively, but Reinstate has no
		// vendor-published table for attachments, tool families, approval
		// modes, multi-root, or a harness context ceiling — and the capsule
		// never re-embeds attachment bytes anyway. Every trait stays omitted
		// rather than guessed, which is what an empty profile means.
		return agentProfile{}
	default:
		return agentProfile{}
	}
}

func verifiedNames(inv capability.Inventory, agent string) map[string]map[string]struct{} {
	out := map[string]map[string]struct{}{
		KindInstruction: {},
		KindMCP:         {},
		KindSkill:       {},
	}
	for _, item := range inv.Items {
		if !item.VerifiedPresence() {
			continue
		}
		if agent != "" && !strings.EqualFold(string(item.Agent), agent) {
			continue
		}
		kind := string(item.Kind)
		name := normalizeCapabilityName(item.Name)
		if name == "" {
			continue
		}
		if _, ok := out[kind]; !ok {
			continue
		}
		out[kind][name] = struct{}{}
	}
	return out
}

func diffPublishedSets(kind string, source, destination map[string]struct{}) []Missing {
	if source == nil {
		return nil
	}
	var missing []Missing
	for _, name := range sortedKeys(source) {
		if destination != nil {
			if _, ok := destination[name]; ok {
				continue
			}
		}
		impact := ImpactDegraded
		if destination == nil {
			impact = ImpactInformational
		}
		missing = append(missing, Missing{Kind: kind, Name: name, Impact: impact})
	}
	return missing
}

func diffTriState(kind, name string, source, destination triState) []Missing {
	if source != triYes {
		return nil
	}
	switch destination {
	case triYes:
		return nil
	case triNo:
		return []Missing{{Kind: kind, Name: name, Impact: ImpactDegraded}}
	default:
		return []Missing{{Kind: kind, Name: name, Impact: ImpactInformational}}
	}
}

func diffContextCeiling(source, destination agentProfile) []Missing {
	if source.ContextCeilingTokens == nil {
		return nil
	}
	if destination.ContextCeilingTokens == nil {
		return []Missing{{Kind: KindContext, Name: "ceiling", Impact: ImpactInformational}}
	}
	if *destination.ContextCeilingTokens < *source.ContextCeilingTokens {
		return []Missing{{Kind: KindContext, Name: "ceiling", Impact: ImpactDegraded}}
	}
	return nil
}

func summarize(agent string, present map[string]map[string]struct{}, profile agentProfile) map[string]any {
	out := map[string]any{
		"agent":             agent,
		"mcp_count":         len(present[KindMCP]),
		"skill_count":       len(present[KindSkill]),
		"instruction_count": len(present[KindInstruction]),
		"tool_families":     joinPublished(profile.ToolFamilies),
		"approval_modes":    joinPublished(profile.ApprovalModes),
		"attachments":       triStateValue(profile.Attachments),
		"multi_root":        triStateValue(profile.MultiRoot),
	}
	if profile.ContextCeilingTokens != nil {
		out["context_ceiling"] = *profile.ContextCeilingTokens
	} else {
		out["context_ceiling"] = summaryOmitted
		out["context_ceiling_reason"] = contextCeilingReason
	}
	return out
}

func joinPublished(set map[string]struct{}) any {
	if set == nil {
		return summaryOmitted
	}
	names := sortedKeys(set)
	if len(names) == 0 {
		return ""
	}
	return strings.Join(names, ",")
}

func triStateValue(v triState) any {
	switch v {
	case triYes:
		return true
	case triNo:
		return false
	default:
		return summaryOmitted
	}
}

func sortedKeys(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func normalizeAgent(agent string) string {
	return strings.ToLower(strings.TrimSpace(agent))
}

func normalizeCapabilityName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func safeCapabilityIDPart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		case r == ':' || r == '.' || r == '/':
			b.WriteByte('-')
		}
	}
	if b.Len() == 0 {
		return "unknown"
	}
	return b.String()
}
