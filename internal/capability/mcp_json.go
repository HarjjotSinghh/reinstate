package capability

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

type scopedMCPName struct {
	name      string
	scope     Scope
	transport Transport
}

type mcpDeclaration struct {
	name      string
	transport Transport
}

func scanClaudeMCPFile(c *collector, anchor, path string, scope Scope, source SourceKind) {
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
		c.addDiagnostic(Diagnostic{Agent: AgentClaude, Kind: KindMCP, Scope: scope, Code: diagnosticForStatus(status)})
		return
	}
	names, truncated, err := extractTopLevelMCP(c.ctx, raw)
	if c.cancelled() {
		return
	}
	if err != nil {
		c.addDiagnostic(Diagnostic{Agent: AgentClaude, Kind: KindMCP, Scope: scope, Code: DiagnosticMalformed})
		return
	}
	names, err = sortedUniqueDeclarations(c.ctx, names)
	if err != nil {
		return
	}
	for _, declaration := range names {
		if c.cancelled() {
			return
		}
		c.add(Item{Agent: AgentClaude, Kind: KindMCP, Name: declaration.name, Scope: scope, State: StateDeclared, SourceKind: source, Transport: declaration.transport})
	}
	if truncated {
		c.addDiagnostic(Diagnostic{Agent: AgentClaude, Kind: KindMCP, Scope: scope, Code: DiagnosticLimitReached})
	}
}

func scanClaudeStateMCP(c *collector, anchor, path string, opts Options) {
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
		c.addDiagnostic(Diagnostic{Agent: AgentClaude, Kind: KindMCP, Scope: ScopeUser, Code: diagnosticForStatus(status)})
		return
	}
	names, truncated, err := extractClaudeStateMCP(c.ctx, raw, opts)
	if c.cancelled() {
		return
	}
	if err != nil {
		c.addDiagnostic(Diagnostic{Agent: AgentClaude, Kind: KindMCP, Scope: ScopeUser, Code: DiagnosticMalformed})
		return
	}
	if err := sortScopedMCPNames(c.ctx, names); err != nil {
		return
	}
	for _, entry := range names {
		if c.cancelled() {
			return
		}
		c.add(Item{Agent: AgentClaude, Kind: KindMCP, Name: entry.name, Scope: entry.scope, State: StateDeclared, SourceKind: SourceClaudeStateJSON, Transport: entry.transport})
	}
	if truncated {
		c.addDiagnostic(Diagnostic{Agent: AgentClaude, Kind: KindMCP, Scope: ScopeUser, Code: DiagnosticLimitReached})
	}
}

func extractTopLevelMCP(ctx context.Context, raw []byte) ([]mcpDeclaration, bool, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	names, truncated, err := extractObjectMCPField(ctx, dec)
	if err != nil {
		return nil, false, err
	}
	if err := requireJSONEOF(ctx, dec); err != nil {
		return nil, false, err
	}
	return names, truncated, nil
}

func extractClaudeStateMCP(ctx context.Context, raw []byte, opts Options) ([]scopedMCPName, bool, error) {
	if err := checkContext(ctx); err != nil {
		return nil, false, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return nil, false, errMalformedJSON
	}
	var names []scopedMCPName
	truncated := false
	for dec.More() {
		if err := checkContext(ctx); err != nil {
			return nil, false, err
		}
		key, err := jsonObjectKey(ctx, dec)
		if err != nil {
			return nil, false, err
		}
		switch key {
		case "mcpServers":
			got, hitLimit, err := extractMCPServerObject(ctx, dec)
			if err != nil {
				return nil, false, err
			}
			for _, declaration := range got {
				if err := checkContext(ctx); err != nil {
					return nil, false, err
				}
				names = appendBoundedScoped(names, scopedMCPName{name: declaration.name, scope: ScopeUser, transport: declaration.transport}, &truncated)
			}
			truncated = truncated || hitLimit
		case "projects":
			got, hitLimit, err := extractProjectMCP(ctx, dec, opts)
			if err != nil {
				return nil, false, err
			}
			for _, name := range got {
				if err := checkContext(ctx); err != nil {
					return nil, false, err
				}
				names = appendBoundedScoped(names, name, &truncated)
			}
			truncated = truncated || hitLimit
		default:
			if err := skipJSONValue(ctx, dec, 0); err != nil {
				return nil, false, err
			}
		}
	}
	if tok, err = dec.Token(); err != nil || tok != json.Delim('}') {
		return nil, false, errMalformedJSON
	}
	if err := requireJSONEOF(ctx, dec); err != nil {
		return nil, false, err
	}
	return names, truncated, nil
}

func extractProjectMCP(ctx context.Context, dec *json.Decoder, opts Options) ([]scopedMCPName, bool, error) {
	if err := checkContext(ctx); err != nil {
		return nil, false, err
	}
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return nil, false, errMalformedJSON
	}
	var names []scopedMCPName
	truncated := false
	for dec.More() {
		if err := checkContext(ctx); err != nil {
			return nil, false, err
		}
		projectKey, err := jsonObjectKey(ctx, dec)
		if err != nil {
			return nil, false, err
		}
		if !matchesActiveProject(projectKey, opts) {
			if err := skipJSONValue(ctx, dec, 0); err != nil {
				return nil, false, err
			}
			continue
		}
		got, hitLimit, err := extractObjectMCPField(ctx, dec)
		if err != nil {
			return nil, false, err
		}
		for _, declaration := range got {
			if err := checkContext(ctx); err != nil {
				return nil, false, err
			}
			if len(names) >= maxEntries {
				truncated = true
				break
			}
			names = append(names, scopedMCPName{name: declaration.name, scope: ScopeLocal, transport: declaration.transport})
		}
		truncated = truncated || hitLimit
	}
	if tok, err = dec.Token(); err != nil || tok != json.Delim('}') {
		return nil, false, errMalformedJSON
	}
	return names, truncated, nil
}

func extractObjectMCPField(ctx context.Context, dec *json.Decoder) ([]mcpDeclaration, bool, error) {
	if err := checkContext(ctx); err != nil {
		return nil, false, err
	}
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return nil, false, errMalformedJSON
	}
	var names []mcpDeclaration
	truncated := false
	for dec.More() {
		if err := checkContext(ctx); err != nil {
			return nil, false, err
		}
		key, err := jsonObjectKey(ctx, dec)
		if err != nil {
			return nil, false, err
		}
		if key != "mcpServers" {
			if err := skipJSONValue(ctx, dec, 0); err != nil {
				return nil, false, err
			}
			continue
		}
		got, hitLimit, err := extractMCPServerObject(ctx, dec)
		if err != nil {
			return nil, false, err
		}
		for _, name := range got {
			if err := checkContext(ctx); err != nil {
				return nil, false, err
			}
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

func extractMCPServerObject(ctx context.Context, dec *json.Decoder) ([]mcpDeclaration, bool, error) {
	if err := checkContext(ctx); err != nil {
		return nil, false, err
	}
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return nil, false, errMalformedJSON
	}
	names := make([]mcpDeclaration, 0, 8)
	truncated := false
	for dec.More() {
		if err := checkContext(ctx); err != nil {
			return nil, false, err
		}
		key, err := jsonObjectKey(ctx, dec)
		if err != nil {
			return nil, false, err
		}
		transport, err := extractClaudeTransport(ctx, dec)
		if err != nil {
			return nil, false, err
		}
		if len(names) < maxEntries {
			names = append(names, mcpDeclaration{name: key, transport: transport})
		} else {
			truncated = true
		}
	}
	if tok, err = dec.Token(); err != nil || tok != json.Delim('}') {
		return nil, false, errMalformedJSON
	}
	return names, truncated, nil
}

// extractClaudeTransport consumes one server declaration while retaining only
// bounded shape signals. A URL without an explicit type is deliberately
// unknown because Claude supports both HTTP and SSE URL transports.
func extractClaudeTransport(ctx context.Context, dec *json.Decoder) (Transport, error) {
	if err := checkContext(ctx); err != nil {
		return TransportUnknown, err
	}
	tok, err := dec.Token()
	if err != nil {
		return TransportUnknown, err
	}
	if tok != json.Delim('{') {
		if err := skipJSONToken(ctx, dec, tok, 0); err != nil {
			return TransportUnknown, err
		}
		return TransportUnknown, nil
	}

	var declared Transport
	typeSeen := false
	typeValid := false
	hasCommand := false
	hasURL := false
	ambiguous := false
	fields := 0
	for dec.More() {
		if err := checkContext(ctx); err != nil {
			return TransportUnknown, err
		}
		key, err := jsonObjectKey(ctx, dec)
		if err != nil {
			return TransportUnknown, err
		}
		value, err := dec.Token()
		if err != nil {
			return TransportUnknown, err
		}
		fields++
		if fields > maxEntries {
			ambiguous = true
			if err := skipJSONToken(ctx, dec, value, 0); err != nil {
				return TransportUnknown, err
			}
			continue
		}

		switch key {
		case "type":
			candidate, valid := recognizedTransport(value)
			if typeSeen && (!valid || !typeValid || candidate != declared) {
				ambiguous = true
			}
			typeSeen = true
			declared = candidate
			typeValid = valid
		case "command":
			hasCommand = true
		case "url":
			hasURL = true
		}
		if err := skipJSONToken(ctx, dec, value, 0); err != nil {
			return TransportUnknown, err
		}
	}
	end, err := dec.Token()
	if err != nil || end != json.Delim('}') {
		return TransportUnknown, errMalformedJSON
	}
	if ambiguous {
		return TransportUnknown, nil
	}
	return inferTransport(declared, typeSeen, typeValid, hasCommand, hasURL, false), nil
}

func recognizedTransport(value json.Token) (Transport, bool) {
	text, ok := value.(string)
	if !ok || len(text) > len(TransportUnknown) {
		return TransportUnknown, false
	}
	switch text {
	case string(TransportStdio):
		return TransportStdio, true
	case string(TransportHTTP):
		return TransportHTTP, true
	case string(TransportSSE):
		return TransportSSE, true
	default:
		return TransportUnknown, false
	}
}

func inferTransport(declared Transport, typeSeen, typeValid, hasCommand, hasURL, urlMeansHTTP bool) Transport {
	if typeSeen {
		if !typeValid || (hasCommand && declared != TransportStdio) || (hasURL && declared == TransportStdio) {
			return TransportUnknown
		}
		return declared
	}
	if hasCommand && !hasURL {
		return TransportStdio
	}
	if hasURL && !hasCommand && urlMeansHTTP {
		return TransportHTTP
	}
	return TransportUnknown
}

func jsonObjectKey(ctx context.Context, dec *json.Decoder) (string, error) {
	if err := checkContext(ctx); err != nil {
		return "", err
	}
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

func skipJSONValue(ctx context.Context, dec *json.Decoder, depth int) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if depth >= maxDepth {
		return errMalformedJSON
	}
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	return skipJSONToken(ctx, dec, tok, depth)
}

func skipJSONToken(ctx context.Context, dec *json.Decoder, tok json.Token, depth int) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		for dec.More() {
			if err := checkContext(ctx); err != nil {
				return err
			}
			if _, err := jsonObjectKey(ctx, dec); err != nil {
				return err
			}
			if err := skipJSONValue(ctx, dec, depth+1); err != nil {
				return err
			}
		}
		end, err := dec.Token()
		if err != nil || end != json.Delim('}') {
			return errMalformedJSON
		}
	case '[':
		for dec.More() {
			if err := checkContext(ctx); err != nil {
				return err
			}
			if err := skipJSONValue(ctx, dec, depth+1); err != nil {
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

func requireJSONEOF(ctx context.Context, dec *json.Decoder) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
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

func sortScopedMCPNames(ctx context.Context, names []scopedMCPName) error {
	sort.Slice(names, func(i, j int) bool {
		if names[i].scope != names[j].scope {
			return names[i].scope < names[j].scope
		}
		if names[i].name != names[j].name {
			return names[i].name < names[j].name
		}
		return names[i].transport < names[j].transport
	})
	return checkContext(ctx)
}

func sortedUniqueDeclarations(ctx context.Context, values []mcpDeclaration) ([]mcpDeclaration, error) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].name != values[j].name {
			return values[i].name < values[j].name
		}
		return values[i].transport < values[j].transport
	})
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	out := values[:0]
	for _, value := range values {
		if err := checkContext(ctx); err != nil {
			return nil, err
		}
		if len(out) == 0 || value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out, nil
}

func checkContext(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

type staticJSONError string

func (e staticJSONError) Error() string { return string(e) }

const errMalformedJSON staticJSONError = "malformed JSON"
