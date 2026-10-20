package installer

import (
	"path/filepath"
	"testing"
)

func TestRepositorySkillsAreSearchable(t *testing.T) {
	skills, err := scanSkills(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) == 0 {
		t.Fatal("repository must contain searchable skills")
	}
}
