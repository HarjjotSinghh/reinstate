package capability

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
)

func FuzzClaudeMCPConfigNames(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`{"mcpServers":{"safe":{"command":"agent"}}}`),
		[]byte(`{"mcpServers":{"../../escape":{"type":"stdio"},"\u001b[31mred\u001b[0m\u202eserver":{"url":"https://example.invalid"}}}`),
		[]byte(`{"mcpServers":{"` + strings.Repeat("界", maxNameRunes+8) + `":{"type":"http"}}}`),
		[]byte(`{"mcpServers":{"unterminated":`),
		{0xff, 0xfe, '{', '}'},
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > int(maxConfigBytes) {
			return
		}
		first, firstTruncated, err := extractTopLevelMCP(context.Background(), raw)
		if err != nil {
			return
		}
		second, secondTruncated, err := extractTopLevelMCP(context.Background(), raw)
		if err != nil || firstTruncated != secondTruncated || !reflect.DeepEqual(first, second) {
			t.Fatalf("Claude MCP parser is nondeterministic: first=%+v/%t second=%+v/%t err=%v", first, firstTruncated, second, secondTruncated, err)
		}
		if len(first) > maxEntries {
			t.Fatalf("Claude MCP parser returned %d names, maximum is %d", len(first), maxEntries)
		}

		firstInventory := inventoryFromFuzzDeclarations(first, AgentClaude, SourceClaudeMCPJSON)
		secondInventory := inventoryFromFuzzDeclarations(second, AgentClaude, SourceClaudeMCPJSON)
		if !reflect.DeepEqual(firstInventory, secondInventory) {
			t.Fatalf("Claude MCP inventory is nondeterministic: first=%+v second=%+v", firstInventory, secondInventory)
		}
		assertFuzzCapabilityNames(t, firstInventory)
	})
}

func FuzzCodexMCPConfigNames(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte("[mcp_servers.safe]\ncommand = \"agent\"\n"),
		[]byte("[mcp_servers.\"../../escape\"]\ntype = \"stdio\"\n[mcp_servers.\"\\u001b[31mred\\u001b[0m\\u202eserver\"]\nurl = \"https://example.invalid\"\n"),
		[]byte("[mcp_servers.\"" + strings.Repeat("界", maxNameRunes+8) + "\"]\ntransport = \"http\"\n"),
		[]byte("[mcp_servers.unterminated\ncommand = \"secret\""),
		{0xff, 0xfe, '[', ']'},
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > int(maxConfigBytes) {
			return
		}
		first, ok := parseFuzzCodexDeclarations(raw)
		if !ok {
			return
		}
		second, ok := parseFuzzCodexDeclarations(raw)
		if !ok || !reflect.DeepEqual(first, second) {
			t.Fatalf("Codex MCP inventory is nondeterministic: first=%+v second=%+v", first, second)
		}
		assertFuzzCapabilityNames(t, first)
	})
}

func parseFuzzCodexDeclarations(raw []byte) (Inventory, bool) {
	var config codexMCPConfig
	if err := toml.Unmarshal(raw, &config); err != nil {
		return Inventory{}, false
	}
	names, _, err := boundedSortedMapKeys(context.Background(), config.MCPServers)
	if err != nil {
		return Inventory{}, false
	}
	declarations := make([]mcpDeclaration, 0, len(names))
	for _, name := range names {
		declarations = append(declarations, mcpDeclaration{name: name, transport: inferCodexTransport(config.MCPServers[name])})
	}
	return inventoryFromFuzzDeclarations(declarations, AgentCodex, SourceCodexMCPConfigTOML), true
}

func inventoryFromFuzzDeclarations(declarations []mcpDeclaration, agent Agent, source SourceKind) Inventory {
	collector := newCollector()
	for _, declaration := range declarations {
		collector.add(Item{
			Agent: agent, Kind: KindMCP, Name: declaration.name,
			Scope: ScopeUser, State: StateDeclared,
			SourceKind: source, Transport: declaration.transport,
		})
	}
	return collector.inventory()
}

func assertFuzzCapabilityNames(t *testing.T, inventory Inventory) {
	t.Helper()
	if len(inventory.Items) > maxEntries {
		t.Fatalf("inventory returned %d items, maximum is %d", len(inventory.Items), maxEntries)
	}
	for _, item := range inventory.Items {
		if item.Name == "" || !utf8.ValidString(item.Name) {
			t.Fatalf("invalid capability name %q", item.Name)
		}
		if utf8.RuneCountInString(item.Name) > maxNameRunes {
			t.Fatalf("capability name exceeds %d runes: %q", maxNameRunes, item.Name)
		}
		if item.Name != strings.TrimSpace(item.Name) || sanitizeName(item.Name) != item.Name {
			t.Fatalf("capability name is not canonically sanitized: %q", item.Name)
		}
		for _, current := range item.Name {
			if unicode.IsControl(current) || isBidiControl(current) || current == '/' || current == '\\' {
				t.Fatalf("unsafe rune %U survived in capability name %q", current, item.Name)
			}
		}
		switch item.Transport {
		case TransportUnknown, TransportStdio, TransportHTTP, TransportSSE:
		default:
			t.Fatalf("invalid transport %q", item.Transport)
		}
	}
}
