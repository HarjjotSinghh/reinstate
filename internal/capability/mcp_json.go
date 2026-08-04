package capability

import (
	"bytes"
	"encoding/json"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

type scopedMCPName struct {
	name  string
	scope Scope
}

func scanClaudeMCPFile(c *collector, anchor, path string, scope Scope, source SourceKind) {
	if !filepath.IsAbs(anchor) {
		return
	}
	raw, status := readBounded(anchor, path)
	if status == pathMissing {
		return
	}
	if status != pathRegular {
		c.addDiagnostic(Diagnostic{Agent: AgentClaude, Kind: KindMCP, Scope: scope, Code: diagnosticForStatus(status)})
		return
	}
	names, truncated, err := extractTopLevelMCP(raw)
	if err != nil {
		c.addDiagnostic(Diagnostic{Agent: AgentClaude, Kind: KindMCP, Scope: scope, Code: DiagnosticMalformed})
		return
	}
	for _, name := range sortedUnique(names) {
		c.add(Item{Agent: AgentClaude, Kind: KindMCP, Name: name, Scope: scope, State: StateDeclared, SourceKind: source})
	}
	if truncated {
		c.addDiagnostic(Diagnostic{Agent: AgentClaude, Kind: KindMCP, Scope: scope, Code: DiagnosticLimitReached})
	}
}

func scanClaudeStateMCP(c *collector, anchor, path string, opts Options) {
	if !filepath.IsAbs(anchor) {
		return
	}
	raw, status := readBounded(anchor, path)
	if status == pathMissing {
		return
	}
	if status != pathRegular {
		c.addDiagnostic(Diagnostic{Agent: AgentClaude, Kind: KindMCP, Scope: ScopeUser, Code: diagnosticForStatus(status)})
		return
	}
	names, truncated, err := extractClaudeStateMCP(raw, opts)
	if err != nil {
		c.addDiagnostic(Diagnostic{Agent: AgentClaude, Kind: KindMCP, Scope: ScopeUser, Code: DiagnosticMalformed})
		return
	}
	sortScopedMCPNames(names)
	for _, entry := range names {
		c.add(Item{Agent: AgentClaude, Kind: KindMCP, Name: entry.name, Scope: entry.scope, State: StateDeclared, SourceKind: SourceClaudeStateJSON})
	}
	if truncated {
		c.addDiagnostic(Diagnostic{Agent: AgentClaude, Kind: KindMCP, Scope: ScopeUser, Code: DiagnosticLimitReached})
	}
}

func extractTopLevelMCP(raw []byte) ([]string, bool, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	names, truncated, err := extractObjectMCPField(dec)
	if err != nil {
		return nil, false, err
	}
	if err := requireJSONEOF(dec); err != nil {
		return nil, false, err
	}
	return names, truncated, nil
}

func extractClaudeStateMCP(raw []byte, opts Options) ([]scopedMCPName, bool, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return nil, false, errMalformedJSON
	}
	var names []scopedMCPName
	truncated := false
	for dec.More() {
		key, err := jsonObjectKey(dec)
		if err != nil {
			return nil, false, err
		}
		switch key {
		case "mcpServers":
			got, hitLimit, err := extractStringKeyObject(dec)
			if err != nil {
				return nil, false, err
			}
			for _, name := range got {
				names = appendBoundedScoped(names, scopedMCPName{name: name, scope: ScopeUser}, &truncated)
			}
			truncated = truncated || hitLimit
		case "projects":
			got, hitLimit, err := extractProjectMCP(dec, opts)
			if err != nil {
				return nil, false, err
			}
			for _, name := range got {
				names = appendBoundedScoped(names, name, &truncated)
			}
			truncated = truncated || hitLimit
		default:
			if err := skipJSONValue(dec, 0); err != nil {
				return nil, false, err
			}
		}
	}
	if tok, err = dec.Token(); err != nil || tok != json.Delim('}') {
		return nil, false, errMalformedJSON
	}
	if err := requireJSONEOF(dec); err != nil {
		return nil, false, err
	}
	return names, truncated, nil
}

func extractProjectMCP(dec *json.Decoder, opts Options) ([]scopedMCPName, bool, error) {
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return nil, false, errMalformedJSON
	}
	var names []scopedMCPName
	truncated := false
	for dec.More() {
		projectKey, err := jsonObjectKey(dec)
		if err != nil {
			return nil, false, err
		}
		if !matchesActiveProject(projectKey, opts) {
			if err := skipJSONValue(dec, 0); err != nil {
				return nil, false, err
			}
			continue
		}
		got, hitLimit, err := extractObjectMCPField(dec)
		if err != nil {
			return nil, false, err
		}
		for _, name := range got {
			if len(names) >= maxEntries {
				truncated = true
				break
			}
			names = append(names, scopedMCPName{name: name, scope: ScopeLocal})
		}
		truncated = truncated || hitLimit
	}
	if tok, err = dec.Token(); err != nil || tok != json.Delim('}') {
		return nil, false, errMalformedJSON
	}
	return names, truncated, nil
}

func extractObjectMCPField(dec *json.Decoder) ([]string, bool, error) {
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return nil, false, errMalformedJSON
	}
	var names []string
	truncated := false
	for dec.More() {
		key, err := jsonObjectKey(dec)
		if err != nil {
			return nil, false, err
		}
		if key != "mcpServers" {
			if err := skipJSONValue(dec, 0); err != nil {
				return nil, false, err
			}
			continue
		}
		got, hitLimit, err := extractStringKeyObject(dec)
		if err != nil {
			return nil, false, err
		}
		for _, name := range got {
			if len(names) >= maxEntries {
				truncated = true
				break
			}
			names = append(names, name)
		}
		truncated = truncated || hitLimit
	}
	if tok, err = dec.Token(); err != nil || tok != json.Delim('}') {
		return nil, false, errMalformedJSON
	}
	return names, truncated, nil
}

func extractStringKeyObject(dec *json.Decoder) ([]string, bool, error) {
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return nil, false, errMalformedJSON
	}
	names := make([]string, 0, 8)
	truncated := false
	for dec.More() {
		key, err := jsonObjectKey(dec)
		if err != nil {
			return nil, false, err
		}
		if len(names) < maxEntries {
			names = append(names, key)
		} else {
			truncated = true
		}
		if err := skipJSONValue(dec, 0); err != nil {
			return nil, false, err
		}
	}
	if tok, err = dec.Token(); err != nil || tok != json.Delim('}') {
		return nil, false, errMalformedJSON
	}
	return names, truncated, nil
}

func jsonObjectKey(dec *json.Decoder) (string, error) {
	tok, err := dec.Token()
	if err != nil {
		return "", err
	}
	key, ok := tok.(string)
	if !ok {
		return "", errMalformedJSON
	}
	return key, nil
}

func skipJSONValue(dec *json.Decoder, depth int) error {
	if depth >= maxDepth {
		return errMalformedJSON
	}
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		for dec.More() {
			if _, err := jsonObjectKey(dec); err != nil {
				return err
			}
			if err := skipJSONValue(dec, depth+1); err != nil {
				return err
			}
		}
		end, err := dec.Token()
		if err != nil || end != json.Delim('}') {
			return errMalformedJSON
		}
	case '[':
		for dec.More() {
			if err := skipJSONValue(dec, depth+1); err != nil {
				return err
			}
		}
		end, err := dec.Token()
		if err != nil || end != json.Delim(']') {
			return errMalformedJSON
		}
	default:
		return errMalformedJSON
	}
	return nil
}

func requireJSONEOF(dec *json.Decoder) error {
	if _, err := dec.Token(); err == io.EOF {
		return nil
	}
	return errMalformedJSON
}

func matchesActiveProject(candidate string, opts Options) bool {
	for _, active := range []string{opts.ProjectRoot, opts.WorkingDir} {
		if active == "" {
			continue
		}
		if opts.GOOS == "windows" {
			cleanCandidate := strings.TrimRight(strings.ReplaceAll(candidate, "\\", "/"), "/")
			cleanActive := strings.TrimRight(strings.ReplaceAll(active, "\\", "/"), "/")
			if strings.EqualFold(cleanCandidate, cleanActive) {
				return true
			}
			continue
		}
		if filepath.Clean(candidate) == filepath.Clean(active) {
			return true
		}
	}
	return false
}

func appendBoundedScoped(names []scopedMCPName, name scopedMCPName, truncated *bool) []scopedMCPName {
	if len(names) >= maxEntries {
		*truncated = true
		return names
	}
	return append(names, name)
}

func sortScopedMCPNames(names []scopedMCPName) {
	sort.Slice(names, func(i, j int) bool {
		if names[i].scope != names[j].scope {
			return names[i].scope < names[j].scope
		}
		return names[i].name < names[j].name
	})
}

type staticJSONError string

func (e staticJSONError) Error() string { return string(e) }

const errMalformedJSON staticJSONError = "malformed JSON"
