package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/capability"
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func TestControlledGitWorkspaceIdentity(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git is unavailable")
	}
	spec, err := loadSpec()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "corpus")
	manifest, err := generateFixture(root, spec)
	if err != nil {
		t.Fatal(err)
	}
	config := runConfig{Root: root, CuratedPath: os.Getenv("PATH")}
	head, err := initializeWorkspaceRepository(config, spec, manifest.Normal)
	if err != nil {
		t.Fatal(err)
	}
	if spec.GitExpectedHead == "PENDING" {
		t.Logf("controlled Git HEAD: %s", head)
	} else if head != spec.GitExpectedHead {
		t.Fatalf("controlled Git HEAD = %s want %s", head, spec.GitExpectedHead)
	}
}

func TestWorkspaceValidationRejectsExecutableLocalConfigWithoutRunningIt(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("Git is unavailable")
	}
	spec, err := loadSpec()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "corpus")
	manifest, err := generateFixture(root, spec)
	if err != nil {
		t.Fatal(err)
	}
	config := runConfig{Root: root, CuratedPath: os.Getenv("PATH")}
	if _, err := initializeWorkspaceRepository(config, spec, manifest.Normal); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "fsmonitor-marker")
	hookDirectory := t.TempDir()
	hook := filepath.Join(hookDirectory, "marker-hook")
	if runtime.GOOS == "windows" {
		hook += ".cmd"
		if err := os.WriteFile(hook, []byte("@echo touched> \"%~1\"\r\n"), 0o700); err != nil {
			t.Fatal(err)
		}
		control := exec.Command("cmd.exe", "/c", hook, marker)
		if output, err := control.CombinedOutput(); err != nil {
			t.Fatalf("marker control: %v: %s", err, output)
		}
	} else {
		if err := os.WriteFile(hook, []byte("#!/bin/sh\nprintf touched > \"$1\"\n"), 0o700); err != nil {
			t.Fatal(err)
		}
		control := exec.Command(hook, marker)
		if output, err := control.CombinedOutput(); err != nil {
			t.Fatalf("marker control: %v: %s", err, output)
		}
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("marker control did not execute")
	}
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	hookCommand := fmt.Sprintf("\"%s\" \"%s\"", strings.ReplaceAll(hook, "\"", "\\\""), strings.ReplaceAll(marker, "\"", "\\\""))
	workspace := filepath.Join(root, manifest.Normal.Corpus, manifest.Normal.RelativeWorkingDirectory)
	command := exec.Command(gitPath, "config", "--local", "core.fsmonitor", hookCommand)
	command.Dir = workspace
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("install hostile local config: %v: %s", err, output)
	}
	if _, err := validateWorkspaceRepository(config, spec, manifest.Normal); err == nil {
		t.Fatal("hostile local Git configuration was accepted")
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("workspace validation executed hostile core.fsmonitor command")
	}
}

func TestFrozenCanonicalDigestAndCorpusShape(t *testing.T) {
	spec, err := loadSpec()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "corpus")
	manifest, err := generateFixture(root, spec)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.CanonicalDigest != spec.CanonicalDigest {
		t.Fatalf("canonical digest = %s want %s", manifest.CanonicalDigest, spec.CanonicalDigest)
	}
	for _, test := range []struct {
		name         string
		manifest     corpusManifest
		records      int
		capabilities int
		events       int
		messages     int
	}{
		{"normal", manifest.Normal, 8, 16, 32, 16},
		{"large", manifest.Large, 1000, 256, 4000, 2000},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.manifest.TotalRecords != test.records || test.manifest.TotalCapabilityNames != test.capabilities ||
				test.manifest.TotalEvents != test.events || test.manifest.TotalMessages != test.messages {
				t.Fatalf("unexpected manifest counts: %+v", test.manifest)
			}
			if test.manifest.TotalRecords > test.manifest.Limit {
				t.Fatalf("records %d exceed limit %d", test.manifest.TotalRecords, test.manifest.Limit)
			}
			if _, err := os.Stat(filepath.Join(root, test.name, "claude", "projects", "phase3-performance", strings.TrimPrefix(test.manifest.ClaudeReference, "claude:")+".jsonl")); err != nil {
				t.Fatal("Claude anchor is missing")
			}
			if !codexAnchorExists(t, root, test.manifest) {
				t.Fatal("Codex anchor is missing")
			}
		})
	}
	if err := verifyGeneratedFixture(root, spec, manifest); err != nil {
		t.Fatal(err)
	}
}

func TestCanonicalDigestDoesNotDependOnPrivateRoot(t *testing.T) {
	spec, err := loadSpec()
	if err != nil {
		t.Fatal(err)
	}
	firstRoot := filepath.Join(t.TempDir(), "a")
	first, err := generateFixture(firstRoot, spec)
	if err != nil {
		t.Fatal(err)
	}
	second, err := generateFixture(filepath.Join(t.TempDir(), "a-much-longer-private-root"), spec)
	if err != nil {
		t.Fatal(err)
	}
	if first.CanonicalDigest != second.CanonicalDigest {
		t.Fatal("canonical digest depends on private root")
	}
	if first.Large.MaterializedDigest == second.Large.MaterializedDigest {
		t.Fatal("materialized digest did not bind exact workspace bytes")
	}
	encoded, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte(firstRoot)) {
		t.Fatal("manifest serialized a private absolute path")
	}
}

func TestEveryGeneratedSessionHasFrozenSchema(t *testing.T) {
	spec, err := loadSpec()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "corpus")
	manifest, err := generateFixture(root, spec)
	if err != nil {
		t.Fatal(err)
	}
	for _, corpus := range []corpusManifest{manifest.Normal, manifest.Large} {
		for _, relative := range corpus.CanonicalRelativeFilePaths {
			if !strings.HasSuffix(relative, ".jsonl") {
				continue
			}
			content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
			if err != nil {
				t.Fatal(err)
			}
			lines := bytes.Split(bytes.TrimSuffix(content, []byte{'\n'}), []byte{'\n'})
			if len(lines) != spec.EventCountPerRecord {
				t.Fatalf("%s event count = %d", relative, len(lines))
			}
			for _, line := range lines {
				var value map[string]any
				if json.Unmarshal(line, &value) != nil {
					t.Fatalf("%s contains malformed JSON", relative)
				}
			}
		}
	}
}

func TestMoveColdIndexFamilyPreservesOnlyExactFiles(t *testing.T) {
	root := t.TempDir()
	cache := filepath.Join(root, "reinstate-home", "cache")
	if err := os.MkdirAll(cache, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range coldIndexFamily {
		if err := os.WriteFile(filepath.Join(cache, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	unrelated := filepath.Join(cache, "session-index-v1.sqlite")
	if err := os.WriteFile(unrelated, []byte("unrelated"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := moveColdIndexFamily(root, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(unrelated); err != nil {
		t.Fatal("cold reset moved unrelated state")
	}
	for _, name := range coldIndexFamily {
		if _, err := os.Stat(filepath.Join(root, "cold-evidence", "sample-1", name)); err != nil {
			t.Fatalf("cold evidence missing %s", name)
		}
	}
}

func TestCalculateMetricsUsesNearestRankP95(t *testing.T) {
	values := make([]time.Duration, 20)
	for index := range values {
		values[index] = time.Duration(index+1) * time.Millisecond
	}
	metrics := calculateMetrics(values)
	if metrics.MedianNS != (10*time.Millisecond + 500*time.Microsecond).Nanoseconds() {
		t.Fatalf("median = %d", metrics.MedianNS)
	}
	if metrics.P95NS != (19 * time.Millisecond).Nanoseconds() {
		t.Fatalf("p95 = %d", metrics.P95NS)
	}
	if metrics.MaximumNS != (20*time.Millisecond).Nanoseconds() || !metrics.Validated {
		t.Fatalf("metrics = %+v", metrics)
	}
}

func TestSmallCommandRejectsOversizedOutput(t *testing.T) {
	environment := append(os.Environ(), "PHASE3PERF_OVERSIZED_HELPER=1")
	if _, err := runSmallCommand("", environment, os.Args[0], "-test.run=^TestPhase3PerfOversizedOutputHelper$"); err == nil {
		t.Fatal("oversized helper output was accepted")
	}
}

func TestPhase3PerfOversizedOutputHelper(t *testing.T) {
	if os.Getenv("PHASE3PERF_OVERSIZED_HELPER") != "1" {
		return
	}
	_, _ = os.Stdout.Write(bytes.Repeat([]byte{'x'}, maxSmallCommandStdoutBytes+1))
}

func TestPerformanceCommandsAreFrozen(t *testing.T) {
	manifest := corpusManifest{
		Limit: 1000, Query: "REINSTATE-V030RC1-PERF-LARGE",
		ClaudeReference: "claude:controlled", CodexReference: "codex:controlled",
	}
	want := [][]string{
		{"sessions", "--limit", "1000", "--json"},
		{"search", manifest.Query, "--limit", "1000", "--json"},
		{"inspect", manifest.ClaudeReference, "--json"},
		{"resume", manifest.ClaudeReference, "--dry-run", "--json"},
		{"resume", manifest.CodexReference, "--dry-run", "--json"},
		{"fork", manifest.ClaudeReference, "--dry-run", "--json"},
		{"fork", manifest.CodexReference, "--dry-run", "--json"},
	}
	commands := performanceCommands(manifest)
	if len(commands) != len(want) {
		t.Fatalf("command count = %d", len(commands))
	}
	for index := range want {
		if strings.Join(commands[index].Args, "\x00") != strings.Join(want[index], "\x00") {
			t.Fatalf("command %d = %q", index, commands[index].Args)
		}
	}
}

func TestStartupResultValidationIsCommandSpecific(t *testing.T) {
	config := runConfig{
		Root:            "/private/evidence",
		ExpectedVersion: "0.3.0-rc.1",
		ExpectedCommit:  "0123456789abcdef0123456789abcdef01234567",
	}
	version := []byte(`{"name":"reinstate","version":"0.3.0-rc.1","commit":"0123456789abcdef0123456789abcdef01234567","date":"2030-01-01T00:00:00Z"}`)
	if err := validateStartupResult(commandDefinition{Label: "version"}, version, config); err != nil {
		t.Fatal(err)
	}
	help := []byte("Usage:\nAvailable Commands:\n  sessions\n  search\n  inspect\n  resume\n  fork\n  version\n")
	if err := validateStartupResult(commandDefinition{Label: "help"}, help, config); err != nil {
		t.Fatal(err)
	}
	if err := validateStartupResult(commandDefinition{Label: "help"}, version, config); err == nil {
		t.Fatal("version JSON unexpectedly passed help validation")
	}
}

func TestOutputValidatorsRejectUnknownFieldsRecursively(t *testing.T) {
	manifest := corpusManifest{
		Corpus: "normal", ClaudeRecords: 1, TotalRecords: 1,
		ClaudeCapabilityNames: 1, CapabilityNameBytes: 32,
		Limit: 1, TimestampBase: "2030-01-01T00:00:00Z", TimestampStepSeconds: 1,
		TitleBytes: 64,
	}
	manifest.ClaudeReference = "claude:" + fixtureID(1, 0)
	// A one-record fixture reuses its only controlled reference for the second
	// visibility assertion; production manifests use distinct Claude/Codex refs.
	manifest.CodexReference = manifest.ClaudeReference
	expected, err := expectedRecordValues(manifest)
	if err != nil {
		t.Fatal(err)
	}
	want := expected[manifest.ClaudeReference]
	sessionsJSON := mustMarshalTestJSON(t, strictSessionsOutput{
		Sessions: []strictSessionSummary{{
			Key: manifest.ClaudeReference, ID: fixtureID(1, 0), Agent: "claude",
			Title: want.Title, UpdatedAt: want.UpdatedAt, MessageCount: 2,
			Files:        []string{"src/perf-aa.go", "src/perf-bb.go"},
			Capabilities: sessionindex.Capabilities{Resume: true, Fork: true},
		}},
		Warnings: []strictWarning{{Agent: "opencode", Code: "agent_not_installed", Message: "controlled"}},
	})
	for _, label := range []string{"sessions", "search"} {
		definition := commandDefinition{Label: label}
		assertRejectsMutatedJSON(t, label+"_top", sessionsJSON, func(root map[string]any) {
			root["unexpected_top_level_secret"] = "CONTROLLED-LEAK"
		}, func(value []byte) error { return validateCommandResult(definition, value, manifest) })
		assertRejectsMutatedJSON(t, label+"_session", sessionsJSON, func(root map[string]any) {
			root["sessions"].([]any)[0].(map[string]any)["unexpected_nested_secret"] = "CONTROLLED-LEAK"
		}, func(value []byte) error { return validateCommandResult(definition, value, manifest) })
		assertRejectsMutatedJSON(t, label+"_warning", sessionsJSON, func(root map[string]any) {
			root["warnings"].([]any)[0].(map[string]any)["unexpected_warning_secret"] = "CONTROLLED-LEAK"
		}, func(value []byte) error { return validateCommandResult(definition, value, manifest) })
	}

	capabilityItem := capability.Item{
		Agent: capability.AgentClaude, Kind: capability.KindSkill,
		Name:  capabilityName("claude", 0, manifest.CapabilityNameBytes),
		Scope: capability.ScopeUser, State: capability.StateCandidate,
		SourceKind: capability.SourceClaudeSkill,
	}
	report := preflight.Report{
		SchemaVersion: preflight.SchemaVersion, SessionRef: manifest.ClaudeReference,
		Decision: preflight.DecisionReady,
		Checks: []preflight.Check{{
			ID: "source.fresh", Status: preflight.StatusMatch, Severity: preflight.SeverityInfo,
			Actual: true, Provenance: "current_observation", Message: "controlled",
		}},
		Capabilities: capability.Inventory{Items: []capability.Item{capabilityItem}},
	}
	record := sessionindex.Record{
		Key: manifest.ClaudeReference, ID: fixtureID(1, 0), Agent: "claude",
		Title: want.Title, UpdatedAt: want.UpdatedAt, MessageCount: 2,
		Files: []string{"src/perf-aa.go", "src/perf-bb.go"}, CanResume: true, CanFork: true,
	}
	inspectJSON := mustMarshalTestJSON(t, strictInspectOutput{Session: record, Environment: report})
	inspectValidator := func(value []byte) error {
		return validateCommandResult(commandDefinition{Label: "inspect_claude"}, value, manifest)
	}
	assertRejectsMutatedJSON(t, "inspect_top", inspectJSON, func(root map[string]any) {
		root["unexpected_top_level_secret"] = "CONTROLLED-LEAK"
	}, inspectValidator)
	assertRejectsMutatedJSON(t, "inspect_session", inspectJSON, func(root map[string]any) {
		root["session"].(map[string]any)["unexpected_session_secret"] = "CONTROLLED-LEAK"
	}, inspectValidator)
	assertRejectsMutatedJSON(t, "inspect_preflight", inspectJSON, func(root map[string]any) {
		root["environment"].(map[string]any)["capabilities"].(map[string]any)["items"].([]any)[0].(map[string]any)["unexpected_capability_secret"] = "CONTROLLED-LEAK"
	}, inspectValidator)
	assertRejectsMutatedJSON(t, "inspect_dynamic_check", inspectJSON, func(root map[string]any) {
		root["environment"].(map[string]any)["checks"].([]any)[0].(map[string]any)["actual"] = map[string]any{"unexpected_secret": "CONTROLLED-LEAK"}
	}, inspectValidator)

	launchJSON := mustMarshalTestJSON(t, strictLaunchOutput{
		LaunchPlan: sessionindex.LaunchPlan{
			Agent: "claude", SessionRef: manifest.ClaudeReference, Operation: "resume",
			Executable: "claude", Args: []string{"--resume", fixtureID(1, 0)}, Dir: "/controlled",
		},
		Environment: report,
	})
	launchValidator := func(value []byte) error {
		return validateCommandResult(commandDefinition{Label: "resume_claude_dry_run"}, value, manifest)
	}
	assertRejectsMutatedJSON(t, "dry_run_plan", launchJSON, func(root map[string]any) {
		root["unexpected_plan_secret"] = "CONTROLLED-LEAK"
	}, launchValidator)
	assertRejectsMutatedJSON(t, "dry_run_preflight", launchJSON, func(root map[string]any) {
		root["environment"].(map[string]any)["agent"].(map[string]any)["unexpected_agent_secret"] = "CONTROLLED-LEAK"
	}, launchValidator)

	versionConfig := runConfig{
		Root: "/private/evidence", ExpectedVersion: "0.3.0-rc.1",
		ExpectedCommit: "0123456789abcdef0123456789abcdef01234567",
	}
	versionJSON := mustMarshalTestJSON(t, strictVersionOutput{
		Name: "reinstate", Version: versionConfig.ExpectedVersion,
		Commit: versionConfig.ExpectedCommit, Date: "2030-01-01T00:00:00Z",
	})
	versionValidator := func(value []byte) error {
		return validateStartupResult(commandDefinition{Label: "version"}, value, versionConfig)
	}
	assertRejectsMutatedJSON(t, "version_top", versionJSON, func(root map[string]any) {
		root["unexpected_version_secret"] = "CONTROLLED-LEAK"
	}, versionValidator)
	assertRejectsMutatedJSON(t, "version_nested_value", versionJSON, func(root map[string]any) {
		root["commit"] = map[string]any{"unexpected_secret": "CONTROLLED-LEAK"}
	}, versionValidator)
}

func mustMarshalTestJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func assertRejectsMutatedJSON(t *testing.T, name string, original []byte, mutate func(map[string]any), validate func([]byte) error) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		var root map[string]any
		if err := json.Unmarshal(original, &root); err != nil {
			t.Fatal(err)
		}
		mutate(root)
		if err := validate(mustMarshalTestJSON(t, root)); err == nil {
			t.Fatal("validator accepted an unknown or schema-invalid sensitive field")
		}
	})
}

func TestImmutableFingerprintDetectsAddedSourceFile(t *testing.T) {
	spec, err := loadSpec()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "corpus")
	manifest, err := generateFixture(root, spec)
	if err != nil {
		t.Fatal(err)
	}
	before, err := fingerprintImmutableFixture(root, manifest.Normal)
	if err != nil {
		t.Fatal(err)
	}
	added := filepath.Join(root, "normal", "claude", "projects", "phase3-performance", "unexpected.jsonl")
	if err := os.WriteFile(added, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	after, err := fingerprintImmutableFixture(root, manifest.Normal)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("added fixture source did not change fingerprint")
	}
}

func TestInstalledAliasesAreRehashed(t *testing.T) {
	root := t.TempDir()
	rein := filepath.Join(root, "rein")
	reinstate := filepath.Join(root, "reinstate")
	for _, path := range []string{rein, reinstate} {
		if err := os.WriteFile(path, []byte("controlled-installed-binary"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	expected, err := fileSHA256(rein)
	if err != nil {
		t.Fatal(err)
	}
	config := runConfig{Rein: rein, Reinstate: reinstate}
	if err := requireInstalledAliasHashes(config, expected); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reinstate, []byte("changed-installed-binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := requireInstalledAliasHashes(config, expected); err == nil {
		t.Fatal("changed installed alias was accepted")
	}
}

func TestTaggedHarnessVerificationUsesExactBlobBytesAndPathSet(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("Git is unavailable")
	}
	source := filepath.Join(t.TempDir(), "source")
	harness := filepath.Join(source, "scripts", "testing", "phase3perf")
	if err := os.MkdirAll(harness, 0o700); err != nil {
		t.Fatal(err)
	}
	tracked := filepath.Join(harness, "main.go")
	original := []byte("package main\n")
	if err := os.WriteFile(tracked, original, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "--quiet", "--initial-branch", "main", "--object-format", "sha1"},
		{"add", "--", "scripts/testing/phase3perf/main.go"},
		{"-c", "user.name=Phase3 Test", "-c", "user.email=phase3@example.invalid", "-c", "commit.gpgSign=false", "commit", "--quiet", "-m", "fixture"},
	} {
		command := exec.Command(gitPath, args...)
		command.Dir = source
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	headCommand := exec.Command(gitPath, "rev-parse", "HEAD")
	headCommand.Dir = source
	headOutput, err := headCommand.Output()
	if err != nil {
		t.Fatal(err)
	}
	config := runConfig{
		SourceRoot: source, ExpectedCommit: strings.TrimSpace(string(headOutput)),
		CuratedPath: filepath.Dir(gitPath),
	}
	if err := verifyTaggedHarnessSource(config); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tracked, []byte("package changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyTaggedHarnessSource(config); err == nil {
		t.Fatal("changed tracked harness bytes were accepted")
	}
	if err := os.WriteFile(tracked, bytes.Repeat([]byte{'x'}, maxTaggedHarnessFileBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyTaggedHarnessSource(config); err == nil {
		t.Fatal("oversized tracked harness bytes were accepted")
	}
	if err := os.WriteFile(tracked, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(harness, "untracked.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyTaggedHarnessSource(config); err == nil {
		t.Fatal("untracked harness source was accepted")
	}
}

func TestTaggedHarnessVerificationIgnoresGitReplaceObjects(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("Git is unavailable")
	}
	source := filepath.Join(t.TempDir(), "source")
	harness := filepath.Join(source, "scripts", "testing", "phase3perf")
	if err := os.MkdirAll(harness, 0o700); err != nil {
		t.Fatal(err)
	}
	tracked := filepath.Join(harness, "main.go")
	if err := os.WriteFile(tracked, []byte("package original\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit := func(args ...string) string {
		t.Helper()
		command := exec.Command(gitPath, args...)
		command.Dir = source
		output, commandErr := command.CombinedOutput()
		if commandErr != nil {
			t.Fatalf("git %v: %v: %s", args, commandErr, output)
		}
		return strings.TrimSpace(string(output))
	}
	runGit("init", "--quiet", "--initial-branch", "main", "--object-format", "sha1")
	runGit("add", "--", "scripts/testing/phase3perf/main.go")
	runGit("-c", "user.name=Phase3 Test", "-c", "user.email=phase3@example.invalid", "-c", "commit.gpgSign=false", "commit", "--quiet", "-m", "original")
	originalCommit := runGit("rev-parse", "HEAD")
	if err := os.WriteFile(tracked, []byte("package replacement\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit("add", "--", "scripts/testing/phase3perf/main.go")
	runGit("-c", "user.name=Phase3 Test", "-c", "user.email=phase3@example.invalid", "-c", "commit.gpgSign=false", "commit", "--quiet", "-m", "replacement")
	replacementCommit := runGit("rev-parse", "HEAD")
	runGit("replace", originalCommit, replacementCommit)
	// Set HEAD to the requested original object while retaining replacement
	// worktree bytes. Without GIT_NO_REPLACE_OBJECTS, ls-tree/cat-file can be
	// redirected to the replacement commit and falsely accept these bytes.
	if err := os.WriteFile(filepath.Join(source, ".git", "HEAD"), []byte(originalCommit+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	config := runConfig{
		SourceRoot: source, ExpectedCommit: originalCommit,
		CuratedPath: filepath.Dir(gitPath),
	}
	if err := verifyTaggedHarnessSource(config); err == nil {
		t.Fatal("Git replace object redirected tagged harness verification")
	}
}

func codexAnchorExists(t *testing.T, root string, manifest corpusManifest) bool {
	t.Helper()
	wanted := strings.TrimPrefix(manifest.CodexReference, "codex:") + ".jsonl"
	found := false
	err := filepath.WalkDir(filepath.Join(root, manifest.Corpus, "codex", "sessions"), func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && strings.HasSuffix(entry.Name(), wanted) {
			found = true
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return found
}
