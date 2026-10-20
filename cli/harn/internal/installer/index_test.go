package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validateCatalog(root string) error {
	skills, err := readCatalog(root)
	if err != nil {
		return err
	}
	entries, err := filepath.Glob(filepath.Join(root, "skills", "*", "SKILL.md"))
	if err != nil {
		return err
	}
	actual := map[string]Skill{}
	for _, entry := range entries {
		dir := filepath.Base(filepath.Dir(entry))
		if strings.HasPrefix(dir, ".") || strings.HasPrefix(dir, "_") {
			continue
		}
		data, err := os.ReadFile(entry)
		if err != nil {
			return err
		}
		skill, err := parseSkill(data, "skills/"+dir)
		if err != nil {
			return err
		}
		actual[skill.Path] = skill
	}
	for _, skill := range skills {
		if entry, ok := actual[skill.Path]; !ok || entry != skill {
			return fmt.Errorf("catalog and SKILL.md differ: %s", skill.Path)
		}
		delete(actual, skill.Path)
	}
	if len(actual) > 0 {
		return fmt.Errorf("unregistered skills: %v", actual)
	}
	return nil
}

func TestRepositoryCatalogConsistency(t *testing.T) {
	if err := validateCatalog(filepath.Join("..", "..", "..", "..")); err != nil {
		t.Fatal(err)
	}
}

func TestCatalogConsistencyDetectsDrift(t *testing.T) {
	for _, test := range []struct {
		name, index, entry string
		valid              bool
	}{
		{"match", `{"name":"a","path":"skills/a","description":"hello world"}`, "name: a\ndescription: >-\n  hello\n  world", true},
		{"name", `{"name":"b","path":"skills/a","description":"hello"}`, "name: a\ndescription: hello", false},
		{"description", `{"name":"a","path":"skills/a","description":"stale"}`, "name: a\ndescription: hello", false},
		{"missing file", `{"name":"a","path":"skills/a","description":"hello"}`, "", false},
		{"unregistered", "", "name: a\ndescription: hello", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "skills", "a"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "skills", "index.json"), []byte(`{"schema_version":1,"skills":[`+test.index+`]}`), 0644); err != nil {
				t.Fatal(err)
			}
			if test.entry != "" {
				if err := os.WriteFile(filepath.Join(root, "skills", "a", "SKILL.md"), []byte("---\n"+test.entry+"\n---\n"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			err := validateCatalog(root)
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v error=%v", test.valid, err)
			}
		})
	}
}
