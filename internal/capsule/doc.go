// Package capsule defines the continuity-capsule v1 model and its deterministic
// canonical encoding.
//
// A capsule is the portable record used for a structured cross-agent handoff.
// It is not a native resume, not a session transfer, and never claims lossless
// fidelity. Portability is always exact, normalized, summarized, referenced, or
// omitted — with a reason when not exact.
//
// Absolute filesystem paths are never serialized. Workspace paths must use
// ${REPO:<id>} / ${HOME} (and related) tokens from package pathmap. Private
// absolute path fields are tagged json:"-" and omitted from CanonicalBytes.
package capsule
