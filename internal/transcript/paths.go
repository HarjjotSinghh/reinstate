package transcript

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/pathmap"
	"github.com/HarjjotSinghh/reinstate/internal/project"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// Path tokenization at the reader boundary
//
// Vendor transcripts are full of absolute paths: a Read tool's file_path, a
// shell call's workdir, a pwd in tool output. Capsule content may hold none of
// them — they do not survive a move between devices or operating systems, and
// they embed the operator's account name. Readers, not capsule validation, are
// responsible for that: a reader converts every structural path it emits into a
// pathmap token, so an absolute path can never reach the capsule.
//
// What is rewritten:
//
//   - tool-call inputs, walked as decoded JSON so each string value is judged
//     on its own (file_path, workdir, paths[], argv entries, …)
//   - tool-result output text
//   - attachment path and reference values
//
// What is not: user, assistant, summary, and harness message bodies. Those are
// prose, and docs/compatibility.md keeps prose unmodified.
//
// A path inside a known root becomes ${REPO:<id>}/… or ${HOME}/…. A path
// outside every known root becomes pathmap.ExternalToken — see that function
// for why the location is dropped rather than carried. Both outcomes are
// portable; neither is an absolute path, and neither aborts the handoff.
//
// Only the leading whitespace-delimited token of a value is rewritten, so
// `/usr/bin/env go test ./...` keeps its arguments while its executable path
// becomes portable. Values that are already tokens, and values capsule
// canonicalization accepts as-is, are returned untouched.

// PathContext holds the roots a reader tokenizes against. It is derived once,
// frozen into the Boundary at Snapshot time, and never reads the filesystem.
type PathContext struct {
	// ProjectID is the canonical project id used for ${REPO:<id>}.
	ProjectID string
	// Root is the absolute workspace root the session ran in.
	Root string
	// Home is the absolute user home used for ${HOME}.
	Home string
}

// PathContextFor derives the tokenization roots for a record. The project id
// matches internal/handoff BindWorkspace so a capsule's workspace root and its
// event paths always name the same project.
func PathContextFor(rec sessionindex.Record) PathContext {
	ctx := PathContext{
		ProjectID: strings.TrimSpace(rec.Project),
		Root:      strings.TrimSpace(rec.Workspace),
	}
	if ctx.ProjectID == "" && ctx.Root != "" {
		ctx.ProjectID = project.OpaqueID(ctx.Root)
	}
	if home, err := os.UserHomeDir(); err == nil {
		ctx.Home = strings.TrimSpace(home)
	}
	return ctx
}

func (p PathContext) mapper() pathmap.Mapper {
	mapper := pathmap.Mapper{Home: p.Home}
	if p.ProjectID != "" && p.Root != "" {
		mapper.Projects = map[string]string{p.ProjectID: p.Root}
	}
	return mapper
}

// Tokenize returns a value that capsule canonicalization accepts, rewriting a
// leading absolute path into its portable token and leaving everything else —
// including the remainder of the value — untouched.
func (p PathContext) Tokenize(value string) string {
	if value == "" || pathmap.IsToken(value) || !capsule.AbsolutePathForbidden(value) {
		return value
	}
	head, rest := splitLeadingToken(value)
	return p.mapper().NormalizePortable(head) + rest
}

// TokenizeJSON rewrites every absolute path inside a decoded JSON value and
// reports whether anything changed. Callers keep the vendor's original encoding
// when nothing changed, so untouched transcripts keep byte-stable digests.
func (p PathContext) TokenizeJSON(value any) (any, bool) {
	switch typed := value.(type) {
	case string:
		rewritten := p.Tokenize(typed)
		return rewritten, rewritten != typed
	case []any:
		out := make([]any, len(typed))
		changed := false
		for i, item := range typed {
			rewrittenItem, itemChanged := p.TokenizeJSON(item)
			out[i] = rewrittenItem
			changed = changed || itemChanged
		}
		return out, changed
	case map[string]any:
		out := make(map[string]any, len(typed))
		changed := false
		for key, item := range typed {
			rewrittenItem, itemChanged := p.TokenizeJSON(item)
			out[key] = rewrittenItem
			changed = changed || itemChanged
		}
		return out, changed
	default:
		return value, false
	}
}

// TokenizeJSONText rewrites absolute paths inside a JSON document carried as
// text, such as a Codex function_call arguments string. Text that is not JSON
// falls back to the single-value rule; text with no path is returned verbatim.
func (p PathContext) TokenizeJSONText(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return text
	}
	if trimmed[0] != '{' && trimmed[0] != '[' {
		return p.Tokenize(text)
	}
	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return p.Tokenize(text)
	}
	rewritten, changed := p.TokenizeJSON(decoded)
	if !changed {
		return text
	}
	encoded, err := json.Marshal(rewritten)
	if err != nil {
		return p.Tokenize(text)
	}
	return string(encoded)
}

// TokenizeBlocks applies the boundary rule in place to already-built blocks and
// reports whether anything changed. It is the backstop that keeps the reader
// invariant true for every emitted structural value; callers recompute content
// hashes only when it reports a change, so untouched transcripts keep their
// existing digests.
func (p PathContext) TokenizeBlocks(blocks []capsule.Block) bool {
	changed := false
	rewrite := func(value string) string {
		rewritten := p.Tokenize(value)
		changed = changed || rewritten != value
		return rewritten
	}
	for i := range blocks {
		blocks[i].Ref = rewrite(blocks[i].Ref)
		blocks[i].Path = rewrite(blocks[i].Path)
		if blocks[i].Type == capsule.BlockTypeToolInput ||
			blocks[i].Type == capsule.BlockTypeToolOutput ||
			blocks[i].Type == capsule.BlockTypeJSON {
			// Tool payloads are usually a JSON document carried as text, so a
			// path sits on a field inside it. The single-value rule cannot see
			// those, and the capsule validator walks the decoded structure, so
			// it rejected what this backstop had left untouched.
			rewritten := p.TokenizeJSONText(blocks[i].Text)
			changed = changed || rewritten != blocks[i].Text
			blocks[i].Text = rewritten
		}
		for key, value := range blocks[i].Meta {
			blocks[i].Meta[key] = rewrite(value)
		}
	}
	return changed
}

func splitLeadingToken(value string) (head, rest string) {
	for i, r := range value {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return value[:i], value[i:]
		}
	}
	return value, ""
}
