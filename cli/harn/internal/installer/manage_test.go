package installer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AwayerCode/Harn/cli/harn/internal/state"
)

func TestDryRunWritesNothing(t *testing.T) {
	t.Parallel()
	archive := newArchiveServer(t, archiveFiles{
		"skills/swe/SKILL.md": {body: "one", mode: 0o644},
	})
	project := t.TempDir()
	manager := Manager{ProjectRoot: project, Client: archive.Client(), DryRun: true}

	results, err := manager.Install(context.Background(), archive.URL, "tag:v1", []string{"skills/swe"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Changed || results[0].Files != 1 {
		t.Fatalf("unexpected dry-run results: %#v", results)
	}
	if _, err := os.Stat(filepath.Join(project, ".harness")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry-run must not create .harness: %v", err)
	}

	manager.DryRun = false
	if _, err := manager.Install(context.Background(), archive.URL, "tag:v1", []string{"skills/swe"}); err != nil {
		t.Fatal(err)
	}
	archive.Set(archiveFiles{
		"skills/swe/SKILL.md": {body: "two", mode: 0o644},
	})
	manager.DryRun = true
	results, err = manager.Update(context.Background(), []string{"skills/swe"}, "tag:v2")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Changed || results[0].Version != "tag:v2" {
		t.Fatalf("unexpected dry-run update results: %#v", results)
	}
	assertFile(t, project, "skills/swe/SKILL.md", "one")
	record, err := state.Load(filepath.Join(project, ".harness"))
	if err != nil {
		t.Fatal(err)
	}
	if record.Downloads["skills/swe"].Version != "tag:v1" {
		t.Fatalf("dry-run update changed the record: %#v", record.Downloads["skills/swe"])
	}
}

func TestListAndStatusReportLocalChanges(t *testing.T) {
	t.Parallel()
	archive := newArchiveServer(t, archiveFiles{
		"skills/swe/SKILL.md":   {body: "one", mode: 0o644},
		"skills/other/SKILL.md": {body: "one", mode: 0o644},
	})
	project := t.TempDir()
	manager := Manager{ProjectRoot: project, Client: archive.Client()}
	if _, err := manager.Install(context.Background(), archive.URL, "tag:v1", []string{"skills/swe", "skills/other"}); err != nil {
		t.Fatal(err)
	}

	entries, err := manager.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Path != "skills/other" || entries[1].Files != 1 {
		t.Fatalf("unexpected entries: %#v", entries)
	}

	statuses, err := manager.Status()
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range statuses {
		if entry.Status != StatusOK {
			t.Fatalf("expected ok status, got %#v", entry)
		}
	}

	writeProjectFile(t, project, "skills/swe/SKILL.md", "local edit", 0o644)
	statuses, err = manager.Status()
	if err != nil {
		t.Fatal(err)
	}
	if statuses[1].Status != StatusModified {
		t.Fatalf("expected modified status, got %#v", statuses[1])
	}

	if err := os.RemoveAll(filepath.Join(project, ".harness", "skills", "other")); err != nil {
		t.Fatal(err)
	}
	statuses, err = manager.Status()
	if err != nil {
		t.Fatal(err)
	}
	if statuses[0].Status != StatusMissing {
		t.Fatalf("expected missing status, got %#v", statuses[0])
	}
}

func TestRemoveDeletesOnlyRecordedFiles(t *testing.T) {
	t.Parallel()
	archive := newArchiveServer(t, archiveFiles{
		"skills/swe/SKILL.md": {body: "one", mode: 0o644},
	})
	project := t.TempDir()
	manager := Manager{ProjectRoot: project, Client: archive.Client()}
	if _, err := manager.Install(context.Background(), archive.URL, "tag:v1", []string{"skills/swe"}); err != nil {
		t.Fatal(err)
	}
	writeProjectFile(t, project, "skills/swe/LOCAL.md", "keep me", 0o644)

	results, err := manager.Remove([]string{"skills/swe"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Files != 1 {
		t.Fatalf("unexpected removal results: %#v", results)
	}
	assertFile(t, project, "skills/swe/LOCAL.md", "keep me")
	if _, err := os.Stat(filepath.Join(project, ".harness", "skills", "swe", "SKILL.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("recorded file should be removed: %v", err)
	}
	record, err := state.Load(filepath.Join(project, ".harness"))
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := record.Downloads["skills/swe"]; exists {
		t.Fatal("record entry should be removed")
	}
}

func TestRemoveRefusesLocalChanges(t *testing.T) {
	t.Parallel()
	archive := newArchiveServer(t, archiveFiles{
		"skills/swe/SKILL.md": {body: "one", mode: 0o644},
	})
	project := t.TempDir()
	manager := Manager{ProjectRoot: project, Client: archive.Client()}
	if _, err := manager.Install(context.Background(), archive.URL, "tag:v1", []string{"skills/swe"}); err != nil {
		t.Fatal(err)
	}
	writeProjectFile(t, project, "skills/swe/SKILL.md", "local edit", 0o644)

	_, err := manager.Remove([]string{"skills/swe"})
	if err == nil || !strings.Contains(err.Error(), "local changes prevent removal") {
		t.Fatalf("expected local change error, got %v", err)
	}
	assertFile(t, project, "skills/swe/SKILL.md", "local edit")
}

func TestRemoveRejectsUnrecordedPath(t *testing.T) {
	t.Parallel()
	project := t.TempDir()
	manager := Manager{ProjectRoot: project}
	if _, err := manager.Remove([]string{"skills/swe"}); err == nil {
		t.Fatal("expected unrecorded path error")
	}
}

func TestRemoveAllowsReinstall(t *testing.T) {
	t.Parallel()
	archive := newArchiveServer(t, archiveFiles{
		"skills/swe/SKILL.md":                    {body: "one", mode: 0o644},
		"skills/swe/references/nested/method.md": {body: "method", mode: 0o644},
		"skills/other/SKILL.md":                  {body: "other", mode: 0o644},
	})
	project := t.TempDir()
	manager := Manager{ProjectRoot: project, Client: archive.Client()}
	if _, err := manager.Install(context.Background(), archive.URL, "tag:v1", []string{"skills/swe", "skills/other"}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Remove([]string{"skills/swe"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(project, ".harness", "skills", "swe")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("resource directory remains: %v", err)
	}
	assertFile(t, project, "skills/other/SKILL.md", "other")
	if _, err := manager.Install(context.Background(), archive.URL, "tag:v1", []string{"skills/swe"}); err != nil {
		t.Fatalf("reinstall: %v", err)
	}
	assertFile(t, project, "skills/swe/references/nested/method.md", "method")
}

func TestRemovePreservesLocalDirectories(t *testing.T) {
	t.Parallel()
	archive := newArchiveServer(t, archiveFiles{
		"skills/swe/SKILL.md":             {body: "one", mode: 0o644},
		"skills/swe/references/method.md": {body: "method", mode: 0o644},
	})
	project := t.TempDir()
	manager := Manager{ProjectRoot: project, Client: archive.Client()}
	if _, err := manager.Install(context.Background(), archive.URL, "tag:v1", []string{"skills/swe"}); err != nil {
		t.Fatal(err)
	}
	writeProjectFile(t, project, "skills/swe/references/LOCAL.md", "keep", 0o644)
	empty := filepath.Join(project, ".harness", "skills", "swe", "local-empty")
	if err := os.Mkdir(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Remove([]string{"skills/swe"}); err != nil {
		t.Fatal(err)
	}
	assertFile(t, project, "skills/swe/references/LOCAL.md", "keep")
	if info, err := os.Stat(empty); err != nil || !info.IsDir() {
		t.Fatalf("local directory lost: %v", err)
	}
	if _, err := manager.Install(context.Background(), archive.URL, "tag:v1", []string{"skills/swe"}); err == nil {
		t.Fatal("reinstall must protect remaining local files")
	}
}
