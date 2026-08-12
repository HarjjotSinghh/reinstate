package capsule

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/pathmap"
)

// CanonicalBytes returns the deterministic encoding used for hashing:
// sorted object keys, no insignificant whitespace, RFC3339 UTC timestamps with
// no sub-second component, and no wall-clock field anywhere in the output.
//
// Absolute filesystem paths are rejected in the capsule's path-typed and other
// structural fields; portable pathmap tokens (${REPO:…}, ${HOME}…, ${WORK:…},
// ${EXTERNAL:…}) are allowed. Free text is never judged as a path — see
// rejectAbsolutePathFields.
func CanonicalBytes(c Capsule) ([]byte, error) {
	c = normalizeCapsule(c)
	if err := rejectAbsolutePathFields(c); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("capsule: marshal: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("capsule: decode: %w", err)
	}
	var buf bytes.Buffer
	if err := writeCanonical(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ComputeID returns the first 32 hex chars of sha256 over the capsule's
// canonical identity preimage. The preimage excludes Identity.ID, a
// self-referential LineageRoot equal to that ID, and projection size/hash
// fields because they are derived from the ID or from artifacts rendered after
// the ID is assigned. A distinct ancestor LineageRoot remains identity-bearing,
// as do policy, included event IDs, sidecar selection, and source/task/workspace
// content.
func ComputeID(c Capsule) (string, error) {
	if c.Identity.LineageRoot == c.Identity.ID {
		c.Identity.LineageRoot = ""
	}
	c.Identity.ID = ""
	c.Projection.EstimatedBytes = 0
	c.Projection.EstimatedTokens = 0
	c.Projection.BootstrapSHA256 = ""
	c.Projection.MarkdownSHA256 = ""
	b, err := CanonicalBytes(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:16]), nil
}

// EventID derives a stable event identity from its SourcePointer.
func EventID(p SourcePointer) string {
	var buf bytes.Buffer
	_ = writeCanonical(&buf, map[string]any{
		"agent":       p.Agent,
		"byte_offset": json.Number(strconv.FormatInt(p.ByteOffset, 10)),
		"index":       json.Number(strconv.Itoa(p.Index)),
		"record_key":  p.RecordKey,
		"session_id":  p.SessionID,
	})
	sum := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(sum[:16])
}

func normalizeCapsule(c Capsule) Capsule {
	for i := range c.Conversation.Events {
		t := c.Conversation.Events[i].Timestamp
		if t.IsZero() {
			continue
		}
		c.Conversation.Events[i].Timestamp = t.UTC().Truncate(time.Second)
	}
	return c
}

func writeCanonical(buf *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case nil:
		buf.WriteString("null")
		return nil
	case bool:
		if x {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
		return nil
	case string:
		enc, err := json.Marshal(x)
		if err != nil {
			return err
		}
		buf.Write(enc)
		return nil
	case json.Number:
		buf.WriteString(string(x))
		return nil
	case float64:
		// Fallback if UseNumber was not applied.
		buf.WriteString(strconv.FormatFloat(x, 'g', -1, 64))
		return nil
	case []any:
		buf.WriteByte('[')
		for i, elem := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeCanonical(buf, elem); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
		return nil
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			enc, err := json.Marshal(k)
			if err != nil {
				return err
			}
			buf.Write(enc)
			buf.WriteByte(':')
			if err := writeCanonical(buf, x[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
		return nil
	default:
		return fmt.Errorf("capsule: unsupported canonical type %T", v)
	}
}

// rejectAbsolutePathFields reports the first absolute filesystem path found in
// a field that may not hold one.
//
// Why this is structural and not a walk over every string in the document:
//
// The rule being enforced is that a path-typed field must arrive as a pathmap
// token. That is what keeps a capsule portable — an absolute path does not
// survive a move between devices or operating systems, and it carries the
// operator's account name to the destination. A reader that forgets to
// tokenize a path it emits must fail loudly here rather than leak, so every
// field that holds a path is checked, and so is every structural scalar
// (identifier, digest, enum, vendor name, command evidence line) that could
// only contain an absolute path by mistake.
//
// Free text is not a path and must never be validated as one. Conversation
// message bodies and the task fields derived from them are prose: a user who
// types "/init do the thing" wrote a slash command, and a user who writes
// "look at /etc/hosts" wrote a sentence. Judging those lexically aborted the
// handoff for a large share of real Claude Code sessions (v0.4.0-rc.1). This
// mirrors the contract docs/compatibility.md already states for the rewriting
// half of the same problem: known structural path fields are rewritten, and
// prose and unknown fields are left unchanged. Capsules are local-only in
// v0.4.0 and the destination is a local process, so prose that names a local
// path is the exposure the product already accepts elsewhere.
//
// Do not turn this back into a walk over every decoded string: that is exactly
// the defect this replaced. Classify new fields instead — canonical_test.go
// fails when a serialized string field has no classification.
func rejectAbsolutePathFields(c Capsule) error {
	var p pathCheck

	p.opaque("schema", c.Schema)
	p.opaque("identity.id", c.Identity.ID)
	p.opaque("identity.lineage_root", c.Identity.LineageRoot)
	p.opaque("identity.parent_session.agent", c.Identity.Parent.Agent)
	p.opaque("identity.parent_session.id", c.Identity.Parent.SessionID)
	p.opaque("identity.parent_session.artifact_sha256", c.Identity.Parent.ArtifactSHA256)
	p.opaque("identity.parent_session.adapter_version", c.Identity.Parent.AdapterVersion)
	p.opaque("raw_source.agent", c.RawSource.Agent)
	p.opaque("raw_source.session_id", c.RawSource.SessionID)
	p.opaque("raw_source.artifact_sha256", c.RawSource.ArtifactSHA256)
	p.opaque("raw_source.adapter_version", c.RawSource.AdapterVersion)
	// RawSource.Path and Workspace.Path are private (json:"-"). They hold the
	// operator's real absolute paths, are never serialized or hashed, and are
	// therefore never checked.

	p.task(c.Task)
	p.workspace(c.Workspace)
	p.conversation(c.Conversation)
	p.capabilities(c.Capabilities)
	p.security(c.Security)
	p.fidelity(c.Fidelity)
	p.projection(c.Projection)

	return p.err
}

// pathCheck keeps the first violation found so the error names one field.
type pathCheck struct{ err error }

// path checks a field whose value is a filesystem path. It must be a portable
// pathmap token.
func (p *pathCheck) path(field, value string) { p.reject(field, value) }

// paths checks a list of path-typed values.
func (p *pathCheck) paths(field string, values []string) {
	for i, value := range values {
		p.reject(fmt.Sprintf("%s[%d]", field, i), value)
	}
}

// opaque checks a structural scalar that is not a path: an identifier, digest,
// enum, vendor tool name, portable reference, or derived evidence line. None of
// them can legitimately be an absolute path, so an absolute path in one is a
// leak from a mis-wired reader and is still rejected.
func (p *pathCheck) opaque(field, value string) { p.reject(field, value) }

// opaques checks a list of structural scalars.
func (p *pathCheck) opaques(field string, values []string) {
	for i, value := range values {
		p.reject(fmt.Sprintf("%s[%d]", field, i), value)
	}
}

func (p *pathCheck) reject(field, value string) {
	if p.err != nil || !isForbiddenAbsolutePath(value) {
		return
	}
	p.err = fmt.Errorf("capsule: absolute filesystem path is not allowed in %s: %q", field, value)
}

func (p *pathCheck) fieldMeta(field string, portability Portability, reason, label string) {
	p.opaque(field+".portability", string(portability))
	p.opaque(field+".reason", reason)
	p.opaque(field+".label", label)
}

func (p *pathCheck) task(t Task) {
	// Prose. goal, latest_user_intent and next_action restate what the user
	// asked for; their metadata is still structural.
	for _, f := range []struct {
		name  string
		field TextField
	}{
		{"task.goal", t.Goal},
		{"task.latest_user_intent", t.LatestUserIntent},
		{"task.next_action", t.NextAction},
	} {
		p.fieldMeta(f.name, f.field.Portability, f.field.Reason, f.field.Label)
	}

	// Prose lists: verbatim or model-derived sentences, never paths.
	for _, f := range []struct {
		name  string
		field ListField
	}{
		{"task.recent_user_messages", t.RecentUserMessages},
		{"task.constraints", t.Constraints},
		{"task.decisions", t.Decisions},
		{"task.rejected_approaches", t.RejectedApproaches},
		{"task.open_questions", t.OpenQuestions},
	} {
		p.fieldMeta(f.name, f.field.Portability, f.field.Reason, f.field.Label)
	}

	// Evidence lines: "<tool> call_id=<id>" and "<runner> · <command> · <state>".
	for _, f := range []struct {
		name  string
		field ListField
	}{
		{"task.completed", t.Completed},
		{"task.pending", t.Pending},
		{"task.tests", t.Tests},
	} {
		p.fieldMeta(f.name, f.field.Portability, f.field.Reason, f.field.Label)
		p.opaques(f.name+".items", f.field.Items)
	}

	// Path-typed: workspace changes and the files a transcript claims it touched.
	for _, f := range []struct {
		name  string
		field ListField
	}{
		{"task.changed_files", t.ChangedFiles},
		{"task.files_touched_per_transcript", t.FilesTouchedPerTranscript},
	} {
		p.fieldMeta(f.name, f.field.Portability, f.field.Reason, f.field.Label)
		p.paths(f.name+".items", f.field.Items)
	}
}

func (p *pathCheck) workspace(w Workspace) {
	p.opaque("workspace.project_id", w.ProjectID)
	p.path("workspace.root", w.Root)
	p.opaque("workspace.branch", w.Branch)
	p.opaque("workspace.head", w.Head)
	p.opaque("workspace.working_tree_digest", w.WorkingTreeDigest)
	p.paths("workspace.changed_files", w.ChangedFiles)
	p.opaques("workspace.tests", w.Tests)
}

func (p *pathCheck) conversation(c Conversation) {
	// A relative artifact reference beside the capsule, never an absolute path.
	p.path("conversation.full_history_ref", c.FullHistoryRef)
	for i, e := range c.Events {
		p.event(fmt.Sprintf("conversation.events[%d]", i), e)
	}
}

func (p *pathCheck) event(field string, e Event) {
	p.opaque(field+".id", e.ID)
	p.opaque(field+".actor", string(e.Actor))
	p.opaque(field+".kind", string(e.Kind))
	p.opaque(field+".native_type", e.NativeType)
	p.opaque(field+".native_name", e.NativeName)
	p.opaque(field+".call_id", e.CallID)
	p.opaque(field+".linked_call_id", e.LinkedCallID)
	p.opaque(field+".portability", string(e.Portability))
	p.opaque(field+".reason", e.Reason)
	p.opaque(field+".content_hash", e.ContentHash)
	p.opaque(field+".source.agent", e.Source.Agent)
	p.opaque(field+".source.session_id", e.Source.SessionID)
	p.opaque(field+".source.record_key", e.Source.RecordKey)
	p.redactions(field+".redactions", e.Redactions)
	for i, b := range e.Blocks {
		p.block(fmt.Sprintf("%s.blocks[%d]", field, i), b)
	}
}

func (p *pathCheck) block(field string, b Block) {
	p.opaque(field+".type", string(b.Type))
	p.opaque(field+".mime", b.MIME)
	p.opaque(field+".sha256", b.SHA256)
	p.path(field+".ref", b.Ref)
	p.path(field+".path", b.Path)
	// Block metadata is reader-supplied structure (attachment name, source kind,
	// error flag); transcript readers tokenize every value in it.
	for _, key := range sortedKeys(b.Meta) {
		p.path(field+".meta."+key, b.Meta[key])
	}

	switch b.Type {
	case BlockTypeText:
		// Prose: the message body exactly as the user or the model wrote it.
	case BlockTypeToolInput, BlockTypeJSON:
		p.toolArguments(field+".text", b.Text)
	default:
		// tool_output, attachment and ref payloads are machine-emitted; readers
		// tokenize their leading path token.
		p.opaque(field+".text", b.Text)
	}
}

// toolArguments rejects an absolute path carried by a path-typed key inside a
// tool call's arguments — file_path, path, paths[], … — which is how a session
// that read a file under the operator's home used to smuggle that home into the
// capsule (v0.4.0-rc.1). Argument values under other keys are left alone: a
// tool argument is often prose (a prompt, a replacement string, a search
// pattern), and judging those as paths is the same mistake as judging a
// message.
func (p *pathCheck) toolArguments(field, text string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return
	}
	if trimmed[0] != '{' && trimmed[0] != '[' {
		// Not a JSON document: the whole value is one argument.
		p.opaque(field, text)
		return
	}
	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		// Truncated or malformed arguments fall back to the single-value rule.
		p.opaque(field, text)
		return
	}
	p.toolArgumentValue(field, decoded, false)
}

func (p *pathCheck) toolArgumentValue(field string, value any, underPathKey bool) {
	switch typed := value.(type) {
	case string:
		if underPathKey {
			p.path(field, typed)
		}
	case []any:
		for i, item := range typed {
			p.toolArgumentValue(fmt.Sprintf("%s[%d]", field, i), item, underPathKey)
		}
	case map[string]any:
		for _, key := range sortedKeys(typed) {
			p.toolArgumentValue(field+"."+key, typed[key], IsPathFieldName(key))
		}
	}
}

func (p *pathCheck) capabilities(d CapabilityDiff) {
	// Privacy-safe summary maps: counts, published tool families, tri-state
	// flags. They never carry paths, secrets, or command lines.
	for _, side := range []struct {
		name    string
		summary map[string]any
	}{
		{"capabilities.source", d.Source},
		{"capabilities.destination", d.Destination},
	} {
		for _, key := range sortedKeys(side.summary) {
			if s, ok := side.summary[key].(string); ok {
				p.opaque(side.name+"."+key, s)
			}
		}
	}
	for i, m := range d.Missing {
		field := fmt.Sprintf("capabilities.missing[%d]", i)
		p.opaque(field+".kind", m.Kind)
		p.opaque(field+".name", m.Name)
		p.opaque(field+".impact", m.Impact)
	}
}

func (p *pathCheck) security(s Security) {
	p.redactions("security.redactions", s.Redactions)
	// security.destination_warning is an operator-facing sentence: prose.
}

func (p *pathCheck) redactions(field string, redactions []Redaction) {
	for i, r := range redactions {
		p.opaque(fmt.Sprintf("%s[%d].category", field, i), string(r.Category))
		p.opaque(fmt.Sprintf("%s[%d].digest", field, i), r.Digest)
	}
}

func (p *pathCheck) fidelity(f Fidelity) {
	p.opaque("fidelity.overall", string(f.Overall))
	p.opaque("fidelity.mode", f.Mode)
	for i, comp := range f.Components {
		field := fmt.Sprintf("fidelity.components[%d]", i)
		p.opaque(field+".name", comp.Name)
		p.opaque(field+".portability", string(comp.Portability))
		p.opaque(field+".reason", comp.Reason)
	}
	p.opaques("fidelity.unsupported", f.Unsupported)
}

func (p *pathCheck) projection(pr Projection) {
	p.opaque("projection.policy", pr.Policy)
	p.opaques("projection.included_event_ids", pr.IncludedEventIDs)
	// A relative artifact reference beside the capsule, never an absolute path.
	p.path("projection.sidecar_ref", pr.SidecarRef)
	p.opaque("projection.bootstrap_sha256", pr.BootstrapSHA256)
	p.opaque("projection.markdown_sha256", pr.MarkdownSHA256)
}

// sortedKeys keeps violation reporting deterministic for map-valued fields.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// IsPathFieldName reports whether a tool-argument key names a filesystem path.
//
// It is exported so the capsule's path check and the checkpoint derivation that
// lifts file references out of tool arguments share exactly one vocabulary; a
// key added here is validated and extracted in the same change.
func IsPathFieldName(key string) bool {
	switch strings.NewReplacer("_", "", "-", "").Replace(strings.ToLower(key)) {
	case "path", "file", "filename", "filepath", "files", "paths",
		"targetpath", "sourcepath", "destinationpath":
		return true
	default:
		return false
	}
}

// AbsolutePathForbidden reports whether s is an absolute filesystem path rather
// than a portable pathmap token, and therefore whether CanonicalBytes rejects it
// in a field that carries one.
//
// Transcript readers use this predicate at their emit boundary so a reader can
// never produce a value the capsule refuses. It is exported to keep exactly one
// definition of "absolute path" in the codebase; it does not relax the rule.
func AbsolutePathForbidden(s string) bool {
	return isForbiddenAbsolutePath(s)
}

func isForbiddenAbsolutePath(p string) bool {
	if p == "" || isPortablePathToken(p) {
		return false
	}
	return pathmap.IsAbsolutePlatform(p)
}

// isPortablePathToken defers to pathmap so the token vocabulary and the
// definition of an absolute path have exactly one owner.
func isPortablePathToken(p string) bool {
	return pathmap.IsToken(p)
}
