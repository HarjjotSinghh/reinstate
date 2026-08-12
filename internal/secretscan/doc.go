// Package secretscan provides deterministic secret detection and redaction
// for transcript and capsule content.
//
// Matches are identified by category and a short SHA-256 digest of the matched
// bytes. Matched values are never logged, returned, or stored — only category,
// byte offsets, and digests leave this package.
//
// Scan results are deterministic for identical input: fixed pattern set, fixed
// overlap resolution, and a fixed Shannon-entropy heuristic for high-entropy
// candidates.
package secretscan
