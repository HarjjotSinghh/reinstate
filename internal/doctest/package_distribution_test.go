package doctest

import (
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
		`gh attestation verify "release-assets/$asset"`,
		"./scripts/check-release-artifacts.sh release-assets",
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
