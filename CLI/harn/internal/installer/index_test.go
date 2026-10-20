package installer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPublishedIndexIsValid(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "..", "..", "Skills", "index.json")
	data, err := os.ReadFile(root)
	if err != nil {
		t.Fatal(err)
	}
	skills, err := parseCatalog(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) == 0 {
		t.Fatal("published index must list at least one skill")
	}
	for _, skill := range skills {
		if skill.Summary == "" {
			t.Fatalf("%s has no summary", skill.Name)
		}
		if _, err := os.Stat(filepath.Join("..", "..", "..", "..", filepath.FromSlash(skill.Path), "SKILL.md")); err != nil {
			t.Fatalf("%s does not resolve to a SKILL.md: %v", skill.Path, err)
		}
	}
}
