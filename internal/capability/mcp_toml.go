package capability

import (
	"context"
	"path/filepath"
	"sort"

	"github.com/BurntSushi/toml"
)

type codexMCPConfig struct {
	MCPServers map[string]codexMCPServer `toml:"mcp_servers"`
}

type codexMCPServer struct {
	Type      tomlTransportDiscriminator `toml:"type"`
	Transport tomlTransportDiscriminator `toml:"transport"`
	Command   tomlPresence               `toml:"command"`
	URL       tomlPresence               `toml:"url"`
}

type tomlPresence bool

func (p *tomlPresence) UnmarshalTOML(any) error {
	*p = true
	return nil
}

type tomlTransportDiscriminator struct {
	present   bool
	valid     bool
	transport Transport
}

func (d *tomlTransportDiscriminator) UnmarshalTOML(value any) error {
	d.present = true
	text, ok := value.(string)
	if !ok || len(text) > len(TransportUnknown) {
		return nil
	}
	switch text {
	case string(TransportStdio):
		d.transport, d.valid = TransportStdio, true
	case string(TransportHTTP):
		d.transport, d.valid = TransportHTTP, true
	case string(TransportSSE):
		d.transport, d.valid = TransportSSE, true
	}
	return nil
}

func scanCodexMCPFile(c *collector, anchor, path string, scope Scope) {
	if !filepath.IsAbs(anchor) {
		return
	}
	raw, status := readBounded(c.ctx, anchor, path)
	if c.cancelled() {
		return
	}
	if status == pathMissing {
		return
	}
	if status != pathRegular {
		c.addDiagnostic(Diagnostic{Agent: AgentCodex, Kind: KindMCP, Scope: scope, Code: diagnosticForStatus(status)})
		return
	}
	var config codexMCPConfig
	if c.cancelled() {
		return
	}
	if err := toml.Unmarshal(raw, &config); err != nil {
		c.addDiagnostic(Diagnostic{Agent: AgentCodex, Kind: KindMCP, Scope: scope, Code: DiagnosticMalformed})
		return
	}
	if c.cancelled() {
		return
	}
	names, truncated, err := boundedSortedMapKeys(c.ctx, config.MCPServers)
	if err != nil {
		return
	}
	for _, name := range names {
		if c.cancelled() {
			return
		}
		c.add(Item{Agent: AgentCodex, Kind: KindMCP, Name: name, Scope: scope, State: StateDeclared, SourceKind: SourceCodexMCPConfigTOML, Transport: inferCodexTransport(config.MCPServers[name])})
	}
	if truncated {
		c.addDiagnostic(Diagnostic{Agent: AgentCodex, Kind: KindMCP, Scope: scope, Code: DiagnosticLimitReached})
	}
}

// boundedSortedMapKeys deterministically retains the lexicographically lowest
// maxEntries keys without allocating a slice proportional to an adversarial
// TOML table. The decoder's typed map retains names plus transport shape bits
// only; command, URL, argument, environment, and authentication values are
// never retained.
func boundedSortedMapKeys[V any](ctx context.Context, values map[string]V) ([]string, bool, error) {
	names := make([]string, 0, min(len(values), maxEntries))
	for name := range values {
		if err := checkContext(ctx); err != nil {
			return nil, false, err
		}
		if len(names) < maxEntries {
			names = append(names, name)
			continue
		}
		maxIndex := 0
		for i := 1; i < len(names); i++ {
			if err := checkContext(ctx); err != nil {
				return nil, false, err
			}
			if names[i] > names[maxIndex] {
				maxIndex = i
			}
		}
		if name < names[maxIndex] {
			names[maxIndex] = name
		}
	}
	sort.Strings(names)
	if err := checkContext(ctx); err != nil {
		return nil, false, err
	}
	return names, len(values) > maxEntries, nil
}

func inferCodexTransport(server codexMCPServer) Transport {
	declared := server.Type
	if server.Transport.present {
		if declared.present && (!declared.valid || !server.Transport.valid || declared.transport != server.Transport.transport) {
			return TransportUnknown
		}
		declared = server.Transport
	}
	return inferTransport(declared.transport, declared.present, declared.valid, bool(server.Command), bool(server.URL), true)
}
