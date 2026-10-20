package installer

import (
	"context"
	"strings"
	"testing"
)

func TestSearchFiltersCatalog(t *testing.T) {
	t.Parallel()
	archive := newArchiveServer(t, archiveFiles{
		"skills/swe/SKILL.md":              {body: "---\nname: swe\ndescription: >-\n  engineering design\n  review\n---\n", mode: 0o644},
		"skills/concise-response/SKILL.md": {body: "---\nname: 'concise-response'\ndescription: writing style\n---\n", mode: 0o644},
		"skills/_template/SKILL.md":        {body: "invalid", mode: 0o644},
		"skills/swe/nested/SKILL.md":       {body: "invalid", mode: 0o644},
		"skills/swe/script.py":             {body: "ignored", mode: 0o644},
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
		t.Fatalf("unexpected description match: %#v", skills)
	}

	skills, err = manager.Search(context.Background(), archive.URL, "tag:v1", "missing")
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 0 {
		t.Fatalf("expected no matches, got %#v", skills)
	}
}

func TestParseSkill(t *testing.T) {
	tests := []struct{ name, data, want string }{
		{"missing front matter", "hello", "missing YAML"},
		{"unclosed", "---\nname: swe", "unclosed"},
		{"missing description", "---\nname: swe\n---", "requires name and description"},
		{"invalid YAML", "---\nname: [\n---", "parse YAML"},
		{"duplicate key", "---\nname: swe\nname: other\ndescription: hi\n---", "parse YAML"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseSkill([]byte(test.data), "skills/swe")
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("got %v, want %s", err, test.want)
			}
		})
	}
	skill, err := parseSkill([]byte("\xef\xbb\xbf---\r\nname: swe\r\ndescription: \"design: review\"\r\n---\r\n"), "skills/swe")
	if err != nil || skill.Summary != "design: review" {
		t.Fatalf("%#v: %v", skill, err)
	}
}

func TestSearchRejectsDuplicateNames(t *testing.T) {
	archive := newArchiveServer(t, archiveFiles{
		"skills/a/SKILL.md": {body: "---\nname: same\ndescription: one\n---", mode: 0o644},
		"skills/b/SKILL.md": {body: "---\nname: same\ndescription: two\n---", mode: 0o644},
	})
	_, err := (Manager{Client: archive.Client()}).Search(context.Background(), archive.URL, "tag:v1", "")
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("got %v", err)
	}
}
