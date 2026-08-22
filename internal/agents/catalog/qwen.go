package catalog

import (
	"regexp"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	qwensrc "github.com/HarjjotSinghh/reinstate/internal/agents/sources/qwen"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/transcript"
)

func init() { agents.MustRegister(Qwen()) }

// `qwen --version` prints the package version as one bare line on stdout, with
// nothing on stderr — measured on macOS with and without QWEN_HOME set. A
// pre-release or nightly suffix is not a version this range has been verified
// against, so it fails closed.
var qwenVersionPattern = regexp.MustCompile(`^((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*))$`)

// Qwen is the Qwen Code descriptor (T3, verified resume).
//
// Promoted to T1 on 2026-08-19 from dual-platform AGENT-PROBE-V1. Both
// platforms write JSONL conversations under
// ~/.qwen/projects/<sanitized-cwd>/chats/<uuid-v4>.jsonl with matching
// first-line keys. Runtime status sidecars are JSON, not JSONL, and never
// match the session glob.
//
// T2 since 2026-08-22: transcript.QwenReader turns those conversations into
// capsule events, so `rein handoff --from qwen` works.
//
// T3 since 2026-08-22: `rein resume qwen:<id>` and `rein fork` launch the
// vendor's own CLI against the vendor's own session, gated by the version
// range below. Qwen is still not a handoff destination — that is T4.
//
// The Claude reader is not reusable here. Qwen's top-level record keys match
// Claude Code's, but the body is a Gemini Content value
// ({"role":…,"parts":[…]}), and Qwen encodes /rewind by re-rooting the
// parentUuid chain rather than by writing a marker.
func Qwen() agents.Descriptor {
	return agents.Descriptor{
		Key:         sessionindex.AgentQwen,
		DisplayName: "Qwen Code",
		Vendor:      "Alibaba",
		DocsURL:     "https://qwenlm.github.io/qwen-code-docs/",
		Tier:        agents.TierResume,
		Family:      agents.FamilyHomeTree,
		Storage: agents.StorageSpec{
			RootEnv: "QWEN_HOME",
			Roots: func(home agents.HomeDir) []agents.Root {
				return []agents.Root{{Path: home.Join(".qwen")}}
			},
			Marker:      "projects",
			SessionGlob: qwensrc.SessionGlob,
			Layout:      "projects-slug-chats-jsonl",
			ProjectKey:  agents.ProjectKeyPathSlug,
			Excluded:    qwensrc.Excluded,
		},
		Native: &agents.NativeSpec{
			Executable: "qwen",
			// Measured on macOS with qwen 0.21.13. `--resume` is a yargs
			// string option, so both `--resume <id>` and `--resume=<id>` are
			// accepted; the separated form is used because it is the one the
			// vendor's own help text shows. An unknown id exits 1 with
			// "No saved session found with ID …", so a stale reference fails
			// loudly rather than silently opening a picker.
			Resume: []string{"--resume", "{{.SessionID}}"},
			// `--fork-session` is documented as "Create a new forked session
			// from the resumed session. Must be used with --resume or
			// --continue", and was observed writing a second chats/<uuid>.jsonl
			// whose records carry forkedFrom {sessionId, messageUuid}.
			Fork:     []string{"--resume", "{{.SessionID}}", "--fork-session"},
			Continue: []string{"--continue"},
		},
		Version: &agents.VersionSpec{
			Args:  []string{"--version"},
			Parse: parseQwenVersion,
			// Both ends are measured on this host, and the spread is not
			// slack. Qwen ships a managed self-updater that installs into
			// <QWEN_HOME>/updates/npm and then execs the updated copy, so one
			// machine answers `--version` differently depending on which root
			// is in scope: 0.21.12 is the bundled npm install, 0.21.13 is the
			// managed update in the default root. 0.21.12 is the version that
			// wrote every fixture in testdata/, and 0.21.13 is the version the
			// documented flag surface was read from. A single-version range
			// would refuse every install that has not self-updated.
			//
			// A newer managed update (0.21.15 was observed installing itself
			// mid-journey) is deliberately outside this range: only its
			// `--version` output has been seen, not its storage layout, so it
			// reports UNTESTED and the operator can still proceed with
			// --allow-untested. See the device report for the drift this
			// updater causes between the probed version and the running one.
			Min: "0.21.12",
			Max: "0.21.13",
		},
		Process: agents.ProcessSpec{
			// Measured from `ps -axo pid=,comm=,args=` against a live
			// `qwen --resume`: the launcher runs as `node …/bin/qwen`, and the
			// two workers it relaunches run as
			// `node --expose-gc …/@qwen-code/qwen-code/cli.js`. The workers are
			// what the node marker recognises.
			Images:      []string{"qwen"},
			NodeMarkers: []string{"/@qwen-code/qwen-code/"},
			Identify: []agents.EnvIdentity{
				{Name: "QWEN_CODE", Value: "1"},
			},
		},
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/qwen.md",
			ProbeReports: []string{
				"docs/testing/results/agent-probes/2026-08-17-macos-qwen.json",
				"docs/testing/results/agent-probes/2026-08-17-windows-qwen.json",
			},
			Fixtures: []string{
				"testdata/sessionindex/qwen/macos",
				"testdata/sessionindex/qwen/windows",
				"testdata/handoff/qwen",
			},
			DeviceReports: []string{
				"docs/testing/results/2026-08-22-macos-qwen-t3.md",
				"docs/testing/results/2026-08-22-windows-qwen-t3.md",
			},
		},
		NewIndexSource: qwensrc.New,
		NewReader: func(agents.Env) (transcript.Reader, error) {
			return transcript.NewQwenReader(), nil
		},
	}
}

func parseQwenVersion(output agents.VersionOutput) (string, bool) {
	return parseVersionLine(output, qwenVersionPattern)
}
