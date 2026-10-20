package app

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type archiveEntry struct {
	body string
	mode int64
}

func newArchiveServer(t *testing.T, files map[string]archiveEntry) *httptest.Server {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, entry := range files {
		header := &tar.Header{
			Name:     "Harn-main/" + name,
			Mode:     entry.mode,
			Size:     int64(len(entry.body)),
			Typeflag: tar.TypeReg,
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write([]byte(entry.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	payload := buffer.Bytes()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/gzip")
		_, _ = writer.Write(payload)
	}))
	t.Cleanup(server.Close)
	return server
}

func run(t *testing.T, runner *Runner, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	runner.stdout = &stdout
	runner.stderr = &stderr
	code := runner.Run(context.Background(), args)
	return code, stdout.String(), stderr.String()
}

func TestInstallListStatusRemoveFlow(t *testing.T) {
	t.Parallel()
	server := newArchiveServer(t, map[string]archiveEntry{
		"skills/index.json":   {body: `{"schema_version":1,"skills":[{"name":"swe","path":"skills/swe","description":"engineering review"}]}`, mode: 0o644},
		"skills/swe/SKILL.md": {body: "---\nname: swe\ndescription: engineering review\n---\n", mode: 0o644},
	})
	project := t.TempDir()
	runner := New(Options{Version: "v0.2.0", Client: server.Client()})

	code, stdout, stderr := run(t, runner, "search",
		"--source", server.URL+"/archive.tar.gz",
		"--version", "tag:v0.2.0", "review")
	if code != 0 || !strings.Contains(stdout, "swe") {
		t.Fatalf("search failed: %d %q %q", code, stdout, stderr)
	}

	code, stdout, stderr = run(t, runner, "install",
		"--source", server.URL+"/archive.tar.gz",
		"--version", "tag:v0.2.0",
		"--path", "skills/swe",
		"--root", project, "--dry-run")
	if code != 0 || !strings.Contains(stdout, "would download") {
		t.Fatalf("dry-run failed: %d %q %q", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(project, ".harness")); !os.IsNotExist(err) {
		t.Fatalf("dry-run wrote state: %v", err)
	}

	code, _, stderr = run(t, runner, "install",
		"--source", server.URL+"/archive.tar.gz",
		"--version", "tag:v0.2.0",
		"--path", "skills/swe",
		"--root", project)
	if code != 0 {
		t.Fatalf("install failed: %s", stderr)
	}

	code, stdout, _ = run(t, runner, "list", "--root", project)
	if code != 0 || !strings.Contains(stdout, "skills/swe") || !strings.Contains(stdout, "tag:v0.2.0") {
		t.Fatalf("list failed: %d %q", code, stdout)
	}

	code, stdout, _ = run(t, runner, "status", "--root", project)
	if code != 0 || !strings.Contains(stdout, "ok skills/swe") {
		t.Fatalf("status failed: %d %q", code, stdout)
	}

	target := filepath.Join(project, ".harness", "skills", "swe", "SKILL.md")
	if err := os.WriteFile(target, []byte("local"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ = run(t, runner, "status", "--root", project)
	if code != 1 || !strings.Contains(stdout, "modified skills/swe") {
		t.Fatalf("status should report a change: %d %q", code, stdout)
	}

	code, _, stderr = run(t, runner, "remove", "--path", "skills/swe", "--root", project)
	if code != 1 || !strings.Contains(stderr, "local changes prevent removal") {
		t.Fatalf("remove should refuse local changes: %d %q", code, stderr)
	}

	if err := os.WriteFile(target, []byte("---\nname: swe\ndescription: engineering review\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr = run(t, runner, "remove", "--path", "skills/swe", "--root", project)
	if code != 0 || !strings.Contains(stdout, "removed skills/swe") {
		t.Fatalf("remove failed: %d %q %q", code, stdout, stderr)
	}
}

func TestUpdateAllDryRun(t *testing.T) {
	t.Parallel()
	server := newArchiveServer(t, map[string]archiveEntry{
		"skills/swe/SKILL.md":   {body: "one", mode: 0o644},
		"skills/other/SKILL.md": {body: "one", mode: 0o644},
	})
	project := t.TempDir()
	runner := New(Options{Version: "v0.2.0", Client: server.Client()})
	code, _, stderr := run(t, runner, "install",
		"--source", server.URL+"/archive.tar.gz",
		"--version", "tag:v0.2.0",
		"--path", "skills/swe",
		"--path", "skills/other",
		"--root", project)
	if code != 0 {
		t.Fatalf("install failed: %s", stderr)
	}

	code, stdout, stderr := run(t, runner, "update", "--all", "--dry-run", "--root", project)
	if code != 0 {
		t.Fatalf("update --all failed: %s", stderr)
	}
	if strings.Count(stdout, "unchanged") != 2 {
		t.Fatalf("expected two unchanged entries: %q", stdout)
	}
}
