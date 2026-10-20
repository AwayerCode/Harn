package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveLoadAndSort(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	record := New()
	record.Downloads["Skills/swe"] = Download{
		Source:  "https://github.com/AwayerCode/Harness",
		Version: "branch:main",
		Kind:    "directory",
		Files: map[string]FileState{
			"Skills/swe/SKILL.md": {SHA256: "abc", Mode: "0644"},
		},
	}
	record.Downloads["Agents"] = Download{
		Source:  "https://github.com/AwayerCode/Harness",
		Version: "tag:v1",
		Kind:    "file",
		Files: map[string]FileState{
			"Agents": {SHA256: "def", Mode: "0755"},
		},
	}
	if err := record.Save(root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"schema_version": 1`) {
		t.Fatalf("missing schema version:\n%s", data)
	}
	if strings.Index(string(data), `"Agents"`) > strings.Index(string(data), `"Skills/swe"`) {
		t.Fatalf("record keys should be sorted:\n%s", data)
	}

	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Downloads) != 2 {
		t.Fatalf("got %d downloads", len(loaded.Downloads))
	}
	if loaded.Downloads["Skills/swe"].Files["Skills/swe/SKILL.md"].Mode != "0644" {
		t.Fatal("file state did not round trip")
	}
}

func TestLoadRejectsInvalidRecord(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data string
	}{
		{
			name: "schema",
			data: `{"schema_version":2,"downloads":{}}`,
		},
		{
			name: "reserved path",
			data: `{"schema_version":1,"downloads":{"bin":{"source":"x","version":"y","kind":"file","files":{"bin":"a"}}}}`,
		},
		{
			name: "outside file",
			data: `{"schema_version":1,"downloads":{"Skills/swe":{"source":"x","version":"y","kind":"directory","files":{"Other/a":"a"}}}}`,
		},
		{
			name: "overlap",
			data: `{"schema_version":1,"downloads":{"Skills":{"source":"x","version":"y","kind":"directory","files":{}},"Skills/swe":{"source":"x","version":"y","kind":"directory","files":{}}}}`,
		},
		{
			name: "trailing data",
			data: `{"schema_version":1,"downloads":{}} {"extra":true}`,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, FileName), []byte(test.data), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(root); err == nil {
				t.Fatal("expected load error")
			}
		})
	}
}
