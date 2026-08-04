package capability

import (
	"path/filepath"
	"sort"

	"github.com/BurntSushi/toml"
)

type codexMCPConfig struct {
	MCPServers map[string]struct {
		Enabled  *bool `toml:"enabled"`
		Required *bool `toml:"required"`
	} `toml:"mcp_servers"`
}

func scanCodexMCPFile(c *collector, anchor, path string, scope Scope) {
	if !filepath.IsAbs(anchor) {
		return
	}
	raw, status := readBounded(anchor, path)
	if status == pathMissing {
		return
	}
	if status != pathRegular {
		c.addDiagnostic(Diagnostic{Agent: AgentCodex, Kind: KindMCP, Scope: scope, Code: diagnosticForStatus(status)})
		return
	}
	var config codexMCPConfig
	if err := toml.Unmarshal(raw, &config); err != nil {
		c.addDiagnostic(Diagnostic{Agent: AgentCodex, Kind: KindMCP, Scope: scope, Code: DiagnosticMalformed})
		return
	}
	names, truncated := boundedSortedMapKeys(config.MCPServers)
	for _, name := range names {
		c.add(Item{Agent: AgentCodex, Kind: KindMCP, Name: name, Scope: scope, State: StateDeclared, SourceKind: SourceCodexMCPConfigTOML})
	}
	if truncated {
		c.addDiagnostic(Diagnostic{Agent: AgentCodex, Kind: KindMCP, Scope: scope, Code: DiagnosticLimitReached})
	}
}

// boundedSortedMapKeys deterministically retains the lexicographically lowest
// maxEntries keys without allocating a slice proportional to an adversarial
// TOML table. The decoder's typed map contains names and two optional booleans
// only; command, URL, argument, environment, and authentication values are
// never retained.
func boundedSortedMapKeys[V any](values map[string]V) ([]string, bool) {
	names := make([]string, 0, min(len(values), maxEntries))
	for name := range values {
		if len(names) < maxEntries {
			names = append(names, name)
			continue
		}
		maxIndex := 0
		for i := 1; i < len(names); i++ {
			if names[i] > names[maxIndex] {
				maxIndex = i
			}
		}
		if name < names[maxIndex] {
			names[maxIndex] = name
		}
	}
	sort.Strings(names)
	return names, len(values) > maxEntries
}
