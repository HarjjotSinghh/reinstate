package doctest

import (
	"regexp"
	"strings"
	"testing"
)

func TestPackagePromotionStartsFromPublishedVerifiedRelease(t *testing.T) {
	workflow := read(t, ".github/workflows/package-publish.yml")
	for _, required := range []string{
		"release:",
		"types: [published]",
		"workflow_dispatch:",
		`git verify-tag "$TAG"`,
		`git merge-base --is-ancestor "$TAG_COMMIT" origin/main`,
		`gh attestation verify "$1"`,
		"./scripts/check-release-artifacts.sh release-assets",
		"Checkout reviewed promotion tools",
		`ref: ${{ github.workflow_sha }}`,
		`--dist "$GITHUB_WORKSPACE/release-assets"`,
		"environment: package-publish",
		"vars.PUBLISH_NPM == 'true'",
		"vars.PUBLISH_JSR == 'true'",
		"vars.PUBLISH_HOMEBREW == 'true'",
		"vars.PUBLISH_CHOCOLATEY == 'true'",
		"vars.PUBLISH_SCOOP == 'true'",
		"vars.PUBLISH_WINGET == 'true'",
		"vars.PUBLISH_AUR == 'true'",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("package promotion workflow is missing %q", required)
		}
	}

	if strings.Contains(workflow, "push:\n    tags:") {
		t.Error("package managers must not publish directly from a tag push")
	}
}

func TestPackagePromotionBindsEveryAttestationToReleaseProvenance(t *testing.T) {
	workflow := read(t, ".github/workflows/package-publish.yml")
	activeLines := make([]string, 0, strings.Count(workflow, "\n")+1)
	for _, line := range strings.Split(workflow, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		activeLines = append(activeLines, line)
	}
	active := strings.Join(activeLines, "\n")

	function := regexp.MustCompile(`(?ms)^          verify_attestation\(\) \{\n            gh attestation verify "\$1" \\\n              --repo "\$GITHUB_REPOSITORY" \\\n              --source-ref "refs/tags/\$TAG" \\\n              --source-digest "\$TAG_COMMIT" \\\n              --predicate-type "https://slsa\.dev/provenance/v1" >/dev/null\n          \}$`)
	if !function.MatchString(active) {
		t.Fatal("package promotion must bind attestation verification to repository, exact tag ref, full tag commit, and SLSA v1 provenance")
	}

	for _, required := range []string{
		`TAG="$RELEASE_TAG"`,
		`TAG_COMMIT="$(git rev-list -n 1 "$TAG")"`,
		`verify_attestation "release-assets/$asset"`,
		`verify_attestation release-assets/checksums.txt`,
	} {
		line := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(required) + `\s*$`)
		if !line.MatchString(active) {
			t.Errorf("package promotion is missing active release-attestation command %q", required)
		}
	}

	if got := strings.Count(active, `gh attestation verify`); got != 1 {
		t.Fatalf("all package-promotion attestations must use the one provenance-bound verifier; found %d active gh invocations", got)
	}
}

func TestPackagePromotionReleaseChannelMatchesTagSemVer(t *testing.T) {
	workflow := read(t, ".github/workflows/package-publish.yml")
	activeLines := make([]string, 0, strings.Count(workflow, "\n")+1)
	for _, line := range strings.Split(workflow, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		activeLines = append(activeLines, line)
	}
	active := strings.Join(activeLines, "\n")

	for _, required := range []string{
		`EXPECTED_PRERELEASE=false`,
		`if [[ "$VERSION" == *-* ]]; then EXPECTED_PRERELEASE=true; fi`,
		`ACTUAL_PRERELEASE="$(jq -r .isPrerelease <<<"$RELEASE_JSON")"`,
		`test "$ACTUAL_PRERELEASE" = "$EXPECTED_PRERELEASE"`,
		`echo "prerelease=$ACTUAL_PRERELEASE"`,
	} {
		line := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(required) + `\s*$`)
		if !line.MatchString(active) {
			t.Errorf("package promotion is missing active release-channel assertion %q", required)
		}
	}

	for _, stableOnly := range []string{
		"vars.PUBLISH_HOMEBREW == 'true' && needs.verify.outputs.prerelease == 'false'",
		"vars.PUBLISH_SCOOP == 'true' && needs.verify.outputs.prerelease == 'false'",
		"vars.PUBLISH_CHOCOLATEY == 'true' && needs.verify.outputs.prerelease == 'false'",
		"vars.PUBLISH_WINGET == 'true' && needs.verify.outputs.prerelease == 'false'",
		"vars.PUBLISH_AUR == 'true' && needs.verify.outputs.prerelease == 'false'",
	} {
		if !strings.Contains(active, stableOnly) {
			t.Errorf("stable-only package lane lost prerelease guard %q", stableOnly)
		}
	}
}

func TestCanonicalReleaseStagesAndAttestsExpandedArtifacts(t *testing.T) {
	workflow := read(t, ".github/workflows/release.yml")
	stageAt := strings.Index(workflow, "./scripts/stage-release-assets.sh dist")
	checkAt := strings.Index(workflow, "./scripts/check-release-artifacts.sh dist")
	if stageAt < 0 || checkAt < 0 || stageAt > checkAt {
		t.Fatal("release workflow must stage raw binaries before validating artifacts")
	}
	for _, required := range []string{
		"dist/*.apk",
		"dist/*.deb",
		"dist/*.pkg.tar.zst",
		"dist/*.rpm",
		"dist/reinstate_*_windows_amd64.exe",
		"--output dist/package-manager",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("canonical release workflow is missing %q", required)
		}
	}
}

func TestGoReleaserBuildsRawAndNativePackageArtifacts(t *testing.T) {
	config := read(t, ".goreleaser.yml")
	for _, required := range []string{
		"id: rein-windows-alias",
		"id: raw",
		"formats: [binary]",
		"id: linux-packages",
		"- apk",
		"- archlinux",
		"- deb",
		"- rpm",
		"dst: /usr/bin/rein",
		`type: symlink`,
	} {
		if !strings.Contains(config, required) {
			t.Errorf("GoReleaser distribution config is missing %q", required)
		}
	}
}

func TestPackagePublishingGuideKeepsStableRolloutReminders(t *testing.T) {
	guide := read(t, "docs/package-manager-publishing.md")
	for _, required := range []string{
		"## Rollout status",
		"## Stable v0.2.0 publication reminder",
		"## Post-publication documentation reminder",
		"limited-platform reconciliation",
		"README.md",
		"docs/getting-started.md",
		"website/src/content/docs/installation.md",
		"CHANGELOG.md",
		"PUBLISH_AUR",
	} {
		if !strings.Contains(guide, required) {
			t.Errorf("package publishing guide is missing stable rollout reminder %q", required)
		}
	}
}

func TestWinGetManifestSchemaIsValidatedBeforePublication(t *testing.T) {
	generator := read(t, "scripts/prepare-package-manager-assets.mjs")
	for _, required := range []string{
		"$schema=https://aka.ms/winget-manifest.version.1.10.0.schema.json",
		"$schema=https://aka.ms/winget-manifest.installer.1.10.0.schema.json",
		"$schema=https://aka.ms/winget-manifest.defaultLocale.1.10.0.schema.json",
		"Installers:\n  - Architecture: x64",
	} {
		if !strings.Contains(generator, required) {
			t.Errorf("WinGet generator is missing schema contract %q", required)
		}
	}

	workflow := read(t, ".github/workflows/ci.yml")
	for _, required := range []string{
		"Validate WinGet manifests without publication",
		"wingetcreate.exe submit package-manager/winget --no-open",
		"Manifest validation succeeded: True",
		"Read-only WinGet validation token unexpectedly published a manifest",
		"24042bd37915805615e6cf969ac57c6439124c3fe85823327f5f3fb24bd9ffea",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("Windows release-packaging gate is missing %q", required)
		}
	}
}
