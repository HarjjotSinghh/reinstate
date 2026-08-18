package probe

import (
	"fmt"
	"strings"
	"time"
)

// Validate checks the artifact against the AGENT-PROBE-V1 contract.
func Validate(a Artifact) error {
	if a.Schema != Schema {
		return fmt.Errorf("schema %q, want %s", a.Schema, Schema)
	}
	if _, err := time.Parse(time.RFC3339, a.GeneratedAt); err != nil {
		return fmt.Errorf("generated_at: %w", err)
	}
	if a.Platform.OS == "" || a.Platform.Arch == "" || a.Platform.DeviceClass == "" {
		return fmt.Errorf("platform incomplete: %+v", a.Platform)
	}
	if a.ReinstateVersion == "" {
		return fmt.Errorf("reinstate_version is empty")
	}
	if a.Agents == nil {
		return fmt.Errorf("agents must be a present array")
	}
	for i, agent := range a.Agents {
		if agent.Key == "" || agent.DisplayName == "" || agent.DeclaredTier == "" {
			return fmt.Errorf("agents[%d] missing identity", i)
		}
		if agent.CandidateRoots == nil {
			return fmt.Errorf("agents[%d] candidate_roots must be present", i)
		}
		if agent.Tree == nil {
			return fmt.Errorf("agents[%d] tree must be present", i)
		}
		if agent.NameShapes == nil {
			return fmt.Errorf("agents[%d] name_shapes must be present", i)
		}
		if agent.FirstLineKeys == nil {
			return fmt.Errorf("agents[%d] first_line_keys must be present", i)
		}
		for _, root := range agent.CandidateRoots {
			if root.RelativeTo == "" {
				return fmt.Errorf("agents[%d] candidate root missing relative_to", i)
			}
		}
		if agent.ResolvedRoot != nil && agent.ResolvedRoot.RelativeTo == "" {
			return fmt.Errorf("agents[%d] resolved_root missing relative_to", i)
		}
	}
	return nil
}

func containsForbidden(raw []byte, needles []string) []string {
	text := string(raw)
	var hits []string
	for _, needle := range needles {
		if needle != "" && strings.Contains(text, needle) {
			hits = append(hits, needle)
		}
	}
	return hits
}
