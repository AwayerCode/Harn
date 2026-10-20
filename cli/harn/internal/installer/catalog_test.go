package installer

import (
	"context"
	"strings"
	"testing"
)

func TestSearchFiltersCatalog(t *testing.T) {
	t.Parallel()
	index := `{"schema_version":1,"skills":[
		{"name":"swe","path":"skills/swe","summary":"engineering design","tags":["review"]},
		{"name":"concise-response","path":"skills/concise-response","summary":"writing style","tags":["communication"]}
	]}`
	archive := newArchiveServer(t, archiveFiles{
		"skills/index.json": {body: index, mode: 0o644},
	})
	manager := Manager{Client: archive.Client()}

	skills, err := manager.Search(context.Background(), archive.URL, "tag:v1", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 2 || skills[0].Name != "concise-response" {
		t.Fatalf("unexpected catalog: %#v", skills)
	}

	skills, err = manager.Search(context.Background(), archive.URL, "tag:v1", "review")
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 || skills[0].Name != "swe" {
		t.Fatalf("unexpected tag match: %#v", skills)
	}

	skills, err = manager.Search(context.Background(), archive.URL, "tag:v1", "missing")
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 0 {
		t.Fatalf("expected no matches, got %#v", skills)
	}
}

func TestParseCatalogRejectsInvalidIndex(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data string
		want string
	}{
		{name: "schema", data: `{"schema_version":2,"skills":[]}`, want: "unsupported"},
		{name: "missing name", data: `{"schema_version":1,"skills":[{"path":"skills/swe"}]}`, want: "require name and path"},
		{name: "outside skills", data: `{"schema_version":1,"skills":[{"name":"swe","path":"cli/swe"}]}`, want: "must live under skills/"},
		{name: "duplicate", data: `{"schema_version":1,"skills":[{"name":"swe","path":"skills/swe"},{"name":"swe","path":"skills/other"}]}`, want: "duplicate"},
		{name: "invalid json", data: `{`, want: "parse resource index"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseCatalog([]byte(test.data))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}
