package doctest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestPublicSEOCommandsStayScopedAndClaimsStayQualified(t *testing.T) {
	root := repoRoot(t)
	paths := []string{"README.md", "CITATION.cff"}
	for _, directory := range []string{
		"website/src/components",
		"website/src/content",
		"website/src/pages",
	} {
		err := filepath.WalkDir(filepath.Join(root, directory), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || (!strings.HasSuffix(path, ".astro") && !strings.HasSuffix(path, ".md")) {
				return nil
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			paths = append(paths, relative)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", directory, err)
		}
	}

	for _, path := range paths {
		body := read(t, path)
		for _, forbidden := range []string{
			"Credentials are never synced",
			"across every coding agent",
			"every machine you own",
			"will always own *one* ecosystem",
			"DIY file sync breaks",
			"Secrets are not part of the sync surface",
			"S3, GCS, S3-compatible, or WebDAV",
		} {
			if strings.Contains(body, forbidden) {
				t.Errorf("%s contains unsupported public claim %q", path, forbidden)
			}
		}

		for _, unsafe := range []string{"rein push --all", "rein pull --all"} {
			if !strings.Contains(body, unsafe) {
				continue
			}
			if path == "website/src/content/docs/sync-a-session.md" &&
				strings.Contains(body, "neither Reinstate nor a coding agent should\nselect every discovered session") {
				continue
			}
			t.Errorf("%s contains unscoped mutating example %q", path, unsafe)
		}
	}
}

func TestWebsiteReleaseTruthStaysSynchronized(t *testing.T) {
	var compatibility struct {
		ReinstateVersion string `json:"reinstateVersion"`
	}
	if err := json.Unmarshal(
		[]byte(read(t, "website/src/data/compatibility.json")),
		&compatibility,
	); err != nil {
		t.Fatalf("parse compatibility.json: %v", err)
	}

	productSource := read(t, "website/src/data/product.ts")
	match := regexp.MustCompile(`currentRelease:\s*'([^']+)'`).FindStringSubmatch(productSource)
	if len(match) != 2 {
		t.Fatal("website product source must declare currentRelease")
	}
	if match[1] != compatibility.ReinstateVersion {
		t.Fatalf(
			"product currentRelease %q differs from compatibility version %q",
			match[1],
			compatibility.ReinstateVersion,
		)
	}

	citation := read(t, "CITATION.cff")
	if !strings.Contains(citation, "version: "+strings.TrimPrefix(match[1], "v")) {
		t.Fatalf("CITATION.cff is not synchronized to %s", match[1])
	}
}
