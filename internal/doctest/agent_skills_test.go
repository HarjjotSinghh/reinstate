package doctest

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

var reinstateSEOSkills = []string{
	"reinstate-ai-search",
	"reinstate-answer-optimization",
	"reinstate-content-brief",
	"reinstate-product-truth",
	"reinstate-release-discoverability",
	"reinstate-seo-ci",
	"reinstate-site-audit",
	"reinstate-structured-data",
	"reinstate-technical-seo",
}

func TestSEOAgentSkillsStayPortableAndInSync(t *testing.T) {
	root := repoRoot(t)
	codexRoot := filepath.Join(root, ".agents", "skills")
	claudeRoot := filepath.Join(root, ".claude", "skills")

	for _, directory := range []string{codexRoot, claudeRoot} {
		entries, err := os.ReadDir(directory)
		if err != nil {
			t.Fatal(err)
		}

		var discovered []string
		for _, entry := range entries {
			if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "reinstate-") {
				continue
			}
			discovered = append(discovered, entry.Name())
		}
		sort.Strings(discovered)
		if !reflect.DeepEqual(discovered, reinstateSEOSkills) {
			t.Errorf("%s contains skills %v; want %v", directory, discovered, reinstateSEOSkills)
		}
	}

	for _, name := range reinstateSEOSkills {
		codexPath := filepath.Join(codexRoot, name, "SKILL.md")
		claudePath := filepath.Join(claudeRoot, name, "SKILL.md")
		codexSkill, err := os.ReadFile(codexPath)
		if err != nil {
			t.Fatal(err)
		}
		claudeSkill, err := os.ReadFile(claudePath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(codexSkill, claudeSkill) {
			t.Errorf("%s differs between .agents/skills and .claude/skills", name)
		}

		frontmatter := "---\nname: " + name + "\ndescription: "
		if !bytes.HasPrefix(codexSkill, []byte(frontmatter)) {
			t.Errorf("%s has invalid or mismatched Agent Skills frontmatter", codexPath)
		}
	}
}
