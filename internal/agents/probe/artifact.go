// Package probe emits the AGENT-PROBE-V1 evidence artifact.
package probe

// Schema is the committable probe artifact identifier.
const Schema = "AGENT-PROBE-V1"

// Artifact is one complete probe report.
type Artifact struct {
	Schema           string   `json:"schema"`
	GeneratedAt      string   `json:"generated_at"`
	Platform         Platform `json:"platform"`
	ReinstateVersion string   `json:"reinstate_version"`
	Agents           []Agent  `json:"agents"`
}

// Platform is the host the probe ran on.
type Platform struct {
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	DeviceClass string `json:"device_class"`
}

// Agent is one catalog agent's redacted storage observation.
type Agent struct {
	Key              string              `json:"key"`
	DisplayName      string              `json:"display_name"`
	DeclaredTier     string              `json:"declared_tier"`
	RootEnv          string              `json:"root_env,omitempty"`
	RootEnvSet       bool                `json:"root_env_set"`
	CandidateRoots   []CandidateRoot     `json:"candidate_roots"`
	ResolvedRoot     *RelativeRoot       `json:"resolved_root"`
	ExecutableOnPath bool                `json:"executable_on_path"`
	VersionRaw       string              `json:"version_raw,omitempty"`
	Tree             []TreeNode          `json:"tree"`
	NameShapes       []NameShape         `json:"name_shapes"`
	FirstLineKeys    map[string][]string `json:"first_line_keys"`
}

// CandidateRoot is one declared storage root, never an absolute path.
type CandidateRoot struct {
	RelativeTo    string `json:"relative_to"`
	Suffix        string `json:"suffix"`
	Exists        bool   `json:"exists"`
	MarkerPresent bool   `json:"marker_present"`
}

// RelativeRoot is a resolved root as a {relative_to, suffix} pair.
type RelativeRoot struct {
	RelativeTo string `json:"relative_to"`
	Suffix     string `json:"suffix"`
}

// TreeNode is one aggregated, shape-normalized path under the resolved root.
type TreeNode struct {
	Path        string `json:"path"`
	Kind        string `json:"kind"`
	Children    int    `json:"children,omitempty"`
	Count       int    `json:"count,omitempty"`
	MedianBytes int64  `json:"median_bytes,omitempty"`
	SampleCount int    `json:"sample_count,omitempty"`
}

// NameShape records how variable path components look after normalization.
type NameShape struct {
	Path    string `json:"path"`
	Shape   string `json:"shape"`
	Samples int    `json:"samples"`
}
