// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Why every demo workspace is a real Git repository:
//
// internal/workspace compares four things against what the session recorded —
// repository identity, branch, HEAD, and working-tree state. Three of the four
// degrade to a hard BLOCK outside a repository ("the recorded Git branch cannot
// be verified outside a Git repository"), so a plain directory can never be
// anything but CANNOT RESUME. Every readiness state above blocked therefore
// requires `git init` plus at least one commit.
//
// Rules for any row that is meant to reach READY (all four are load-bearing):
//
//   - The tree must be genuinely clean. UNTRACKED FILES COUNT AS DIRTY — the
//     working-tree digest is computed from porcelain=v2 records, and '?' lines
//     contribute just like '1'/'2'/'u' lines do.
//   - No runtime declaration file in the workspace root: no go.mod, .nvmrc,
//     .node-version, or package.json with engines.node. An UNPARSEABLE one is
//     just as bad as a mismatched one — it yields declarationUnknown, which is
//     a warning, not silence.
//   - No capability files: no CLAUDE.md, AGENTS.md, .mcp.json, .claude/**,
//     .agents/skills/**. Capability discovery would find them, and a baseline
//     seeded before they existed would report them as newly appeared.
//   - No Git remote. Without one, repository identity derives from the root
//     commit (roots-sha256:<64 hex>), which the comparison accepts. Adding a
//     remote is not "realism": if a Codex session also records a
//     repository_url and the two disagree, the row BLOCKS with exit 7. A
//     remote belongs on exactly one row in this bench — the one whose entire
//     purpose is that block.
//
// The workspace must also equal the repository root. `rein handoff` sends the
// destination agent to report.Workspace.Git.Root, not to the recorded
// workspace, so a workspace nested inside a repo hands off into the repo root
// instead.

// gitEnv keeps repository creation independent of the developer's own Git
// configuration and identity. GIT_CONFIG_GLOBAL/SYSTEM are neutralized so a
// host commit.gpgsign, init.defaultBranch, or core.hooksPath cannot leak in,
// and no commit carries the developer's name. Dates are pinned so re-running
// the generator produces identical commit hashes.
func gitEnv() []string {
	return append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_AUTHOR_NAME=Reinstate Sandbox",
		"GIT_AUTHOR_EMAIL=sandbox@example.invalid",
		"GIT_COMMITTER_NAME=Reinstate Sandbox",
		"GIT_COMMITTER_EMAIL=sandbox@example.invalid",
		"GIT_AUTHOR_DATE=2026-01-01T00:00:00+00:00",
		"GIT_COMMITTER_DATE=2026-01-01T00:00:00+00:00",
	)
}

func git(dir string, args ...string) error {
	command := exec.Command("git", args...)
	command.Dir = dir
	command.Env = gitEnv()
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s in %s: %w: %s",
			strings.Join(args, " "), dir, err, strings.TrimSpace(string(output)))
	}
	return nil
}

// gitAvailable reports whether a usable git is on PATH. Without it the bench
// cannot produce anything but blocked rows, so the generator refuses outright
// rather than emitting nine misleading fixtures.
func gitAvailable() bool {
	command := exec.Command("git", "--version")
	command.Env = gitEnv()
	return command.Run() == nil
}

// buildWorkspace materializes the workspace for one demo row. Every lever it
// applies maps to exactly one check in the environment report; see the comments
// on the session table in main.go for which row uses which.
func buildWorkspace(path string, s session) error {
	if s.missingWS {
		// Never created. The session still records this path, so
		// `workspace.available` reports StatusMissing / SeverityBlock.
		return os.RemoveAll(path)
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	readme := "# " + s.project + "\n\nSynthetic Reinstate bench workspace. Nothing here is real.\n"
	if err := os.WriteFile(filepath.Join(path, "README.md"), []byte(readme), 0o600); err != nil {
		return err
	}
	// Committed, not ignored-and-untracked: an untracked .gitignore would make
	// the tree dirty. It exists because `handoff.Plan` runs three times per
	// launch and all three results are compared with reflect.DeepEqual — a
	// stray .DS_Store appearing between them aborts the launch with "source,
	// environment, or structured handoff plan changed at the execution
	// boundary".
	tracked := []string{"README.md", ".gitignore"}
	if err := os.WriteFile(filepath.Join(path, ".gitignore"), []byte(".DS_Store\nThumbs.db\n"), 0o600); err != nil {
		return err
	}
	// A runtime declaration turns into a `runtime.<name>.declaration` warning
	// whenever the installed toolchain does not satisfy it. .nvmrc pins a Node
	// the host almost certainly does not have; go.mod's `go` directive is a
	// MINIMUM, so it only warns when the version is in the future — `go 1.19.0`
	// is satisfied by any modern toolchain and produces nothing.
	if s.nvmrc != "" {
		if err := os.WriteFile(filepath.Join(path, ".nvmrc"), []byte(s.nvmrc), 0o600); err != nil {
			return err
		}
		tracked = append(tracked, ".nvmrc")
	}
	if s.gomod != "" {
		if err := os.WriteFile(filepath.Join(path, "go.mod"), []byte(s.gomod), 0o600); err != nil {
			return err
		}
		tracked = append(tracked, "go.mod")
	}

	// -b names the initial branch explicitly so the result never depends on the
	// host's init.defaultBranch. liveBranch is the branch actually checked out;
	// when it differs from the branch the session recorded, `git.branch` warns.
	live := s.branch
	if s.liveBranch != "" {
		live = s.liveBranch
	}
	if err := git(path, "init", "-q", "-b", live); err != nil {
		return err
	}
	if err := git(path, append([]string{"add", "--"}, tracked...)...); err != nil {
		return err
	}
	if err := git(path, "commit", "-q", "-m", "sandbox: initial commit"); err != nil {
		return err
	}
	if s.detach {
		// A detached HEAD warns on `git.branch` and — unlike every other
		// warning in this bench — can never be cleared by re-baselining, because
		// the branch expectation falls back to the vendor-recorded value.
		if err := git(path, "checkout", "-q", "--detach", "HEAD"); err != nil {
			return err
		}
	}
	// Deliberately no `git remote add`, on every row including the one that
	// blocks on repository identity. Without a remote the live identity is
	// derived from the root commit (roots-sha256:…); the blocked row gets its
	// mismatch purely from the repository_url its session file records, so the
	// block is metadata-driven and the rest of its report reads clean.
	if s.untracked != "" {
		// Written after the commit and never added: the working tree is
		// modified, so `git.working_tree` warns.
		if err := os.WriteFile(filepath.Join(path, s.untracked), []byte("scratch, uncommitted\n"), 0o600); err != nil {
			return err
		}
	}
	return nil
}
