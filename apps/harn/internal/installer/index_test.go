package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
		{"match", `{"name":"a","description":"hello world"}`, "name: a\ndescription: >-\n  hello\n  world", true},
		{"name", `{"name":"b","description":"hello"}`, "name: a\ndescription: hello", false},
		{"description", `{"name":"a","description":"stale"}`, "name: a\ndescription: hello", false},
		{"missing file", `{"name":"a","description":"hello"}`, "", false},
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

func TestRepositoryCatalogFields(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "skills", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		Skills []map[string]json.RawMessage `json:"skills"`
	}
	if err := json.Unmarshal(data, &catalog); err != nil {
		t.Fatal(err)
	}
	for _, entry := range catalog.Skills {
		if len(entry) != 5 {
			t.Fatalf("unexpected fields: %s", data)
		}
		for _, key := range []string{"name", "description", "version", "updated_at", "source"} {
			if _, ok := entry[key]; !ok {
				t.Fatalf("missing %s", key)
			}
		}
		var updated, version string
		if err := json.Unmarshal(entry["updated_at"], &updated); err != nil {
			t.Fatal(err)
		}
		if _, err := time.Parse(time.RFC3339, updated); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(entry["version"], &version); err != nil || version == "" {
			t.Fatal("version required")
		}
		var source map[string]*string
		if err := json.Unmarshal(entry["source"], &source); err != nil {
			t.Fatal(err)
		}
		repo, hasRepo := source["repository"]
		path, hasPath := source["path"]
		if len(source) != 2 || !hasRepo || !hasPath || (repo == nil) != (path == nil) {
			t.Fatal("source requires repository and path, both null or both filled")
		}
		if repo != nil && (strings.TrimSpace(*repo) == "" || strings.TrimSpace(*path) == "") {
			t.Fatal("empty external source")
		}
	}
}
