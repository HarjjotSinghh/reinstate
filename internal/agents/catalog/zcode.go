package catalog

import "github.com/HarjjotSinghh/reinstate/internal/agents"

func init() { agents.MustRegister(ZCode()) }

// ZCode is the official Z.ai desktop ADE (T0, desktop_only).
//
// Z.ai ships ZCode as a desktop app from zcode.z.ai, not as a terminal CLI.
// There is no documented session-file layout and no vendor session-export API,
// so the descriptor claims no Storage.Layout and no index constructor.
// The npm package zcode-app-cli is unaffiliated and is not this catalog key.
func ZCode() agents.Descriptor {
	return agents.Descriptor{
		Key:         "zcode",
		DisplayName: "ZCode",
		Vendor:      "Z.ai",
		DocsURL:     "https://zcode.z.ai/en",
		Tier:        agents.TierKnown,
		Family:      agents.FamilyRemote,
		T0Reason:    agents.T0DesktopOnly,
		Storage: agents.StorageSpec{
			// ~/.zcode is documented config, credentials, logs, and command
			// output — not a session store. Do not treat it as one.
			ProjectKey: agents.ProjectKeyNone,
			Excluded: []string{
				"v2/credentials.json",
				"v2/config.json",
				"v2/telemetry-state.json",
				"cli/exec",
				"logs",
			},
		},
		Process: agents.ProcessSpec{
			// Linux .deb installs a `zcode` GUI binary. It is not a session CLI.
			Images: []string{"zcode"},
		},
		Evidence: agents.Evidence{
			StoragePage: "docs/session-storage/zcode.md",
		},
	}
}
