package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AwayerCode/Harness/CLI/harness/internal/state"
)

func TestInstallAndUpdatePreserveLocalFiles(t *testing.T) {
	t.Parallel()
	archive := newArchiveServer(t, archiveFiles{
		"Skills/swe/SKILL.md":               {body: "version one", mode: 0o644},
		"Skills/swe/scripts/run.sh":         {body: "#!/bin/sh\n", mode: 0o755},
		"Skills/swe/references/obsolete.md": {body: "old", mode: 0o644},
		"Skills/other/SKILL.md":             {body: "other", mode: 0o644},
	})
	project := t.TempDir()
	manager := Manager{ProjectRoot: project, Client: archive.Client()}

	results, err := manager.Install(context.Background(), archive.URL, "tag:v1", []string{"Skills/swe"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Changed || results[0].Files != 3 {
		t.Fatalf("unexpected install results: %#v", results)
	}
	assertFile(t, project, "Skills/swe/SKILL.md", "version one")
	assertFile(t, project, "Skills/swe/scripts/run.sh", "#!/bin/sh\n")
	if _, err := os.Stat(filepath.Join(project, ".harness", "Skills", "other")); !os.IsNotExist(err) {
		t.Fatalf("unselected path should not be installed: %v", err)
	}

	info, err := os.Stat(filepath.Join(project, ".harness", "Skills", "swe", "scripts", "run.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("expected executable file, got %o", info.Mode().Perm())
	}

	writeProjectFile(t, project, "Skills/swe/LOCAL.md", "keep me", 0o644)
	archive.Set(archiveFiles{
		"Skills/swe/SKILL.md":              {body: "version two", mode: 0o644},
		"Skills/swe/scripts/run.sh":        {body: "#!/bin/sh\n", mode: 0o755},
		"Skills/swe/references/current.md": {body: "new", mode: 0o644},
	})

	results, err = manager.Update(context.Background(), []string{"Skills/swe"}, "tag:v2")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Changed || results[0].Version != "tag:v2" {
		t.Fatalf("unexpected update results: %#v", results)
	}
	assertFile(t, project, "Skills/swe/SKILL.md", "version two")
	assertFile(t, project, "Skills/swe/references/current.md", "new")
	assertFile(t, project, "Skills/swe/LOCAL.md", "keep me")
	if _, err := os.Stat(filepath.Join(project, ".harness", "Skills", "swe", "references", "obsolete.md")); !os.IsNotExist(err) {
		t.Fatalf("stale recorded file should be removed: %v", err)
	}

	record, err := state.Load(filepath.Join(project, ".harness"))
	if err != nil {
		t.Fatal(err)
	}
	download := record.Downloads["Skills/swe"]
	if download.Version != "tag:v2" || download.Kind != "directory" {
		t.Fatalf("unexpected record: %#v", download)
	}
	if _, exists := download.Files["Skills/swe/LOCAL.md"]; exists {
		t.Fatal("local file must not become part of the download record")
	}
}

func TestUpdateConflictWritesNothing(t *testing.T) {
	t.Parallel()
	archive := newArchiveServer(t, archiveFiles{
		"Skills/swe/SKILL.md":   {body: "one", mode: 0o644},
		"Skills/other/SKILL.md": {body: "one", mode: 0o644},
	})
	project := t.TempDir()
	manager := Manager{ProjectRoot: project, Client: archive.Client()}
	if _, err := manager.Install(context.Background(), archive.URL, "tag:v1", []string{"Skills/swe", "Skills/other"}); err != nil {
		t.Fatal(err)
	}
	recordBefore, err := os.ReadFile(filepath.Join(project, ".harness", state.FileName))
	if err != nil {
		t.Fatal(err)
	}
	writeProjectFile(t, project, "Skills/swe/SKILL.md", "local edit", 0o644)
	archive.Set(archiveFiles{
		"Skills/swe/SKILL.md":   {body: "two", mode: 0o644},
		"Skills/other/SKILL.md": {body: "two", mode: 0o644},
	})

	_, err = manager.Update(context.Background(), []string{"Skills/other", "Skills/swe"}, "tag:v2")
	if err == nil || !strings.Contains(err.Error(), "local changes prevent update") {
		t.Fatalf("expected conflict, got %v", err)
	}
	assertFile(t, project, "Skills/swe/SKILL.md", "local edit")
	assertFile(t, project, "Skills/other/SKILL.md", "one")
	recordAfter, err := os.ReadFile(filepath.Join(project, ".harness", state.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(recordBefore, recordAfter) {
		t.Fatal("download record changed after a failed preflight")
	}
}

func TestUpdateDoesNotOverwriteLocalCollision(t *testing.T) {
	t.Parallel()
	archive := newArchiveServer(t, archiveFiles{
		"Skills/swe/SKILL.md": {body: "one", mode: 0o644},
	})
	project := t.TempDir()
	manager := Manager{ProjectRoot: project, Client: archive.Client()}
	if _, err := manager.Install(context.Background(), archive.URL, "tag:v1", []string{"Skills/swe"}); err != nil {
		t.Fatal(err)
	}
	writeProjectFile(t, project, "Skills/swe/new.md", "local", 0o644)
	archive.Set(archiveFiles{
		"Skills/swe/SKILL.md": {body: "two", mode: 0o644},
		"Skills/swe/new.md":   {body: "upstream", mode: 0o644},
	})

	_, err := manager.Update(context.Background(), []string{"Skills/swe"}, "")
	if err == nil || !strings.Contains(err.Error(), "would be overwritten") {
		t.Fatalf("expected collision, got %v", err)
	}
	assertFile(t, project, "Skills/swe/SKILL.md", "one")
	assertFile(t, project, "Skills/swe/new.md", "local")
}

func TestInstallRefusesUnrecordedPath(t *testing.T) {
	t.Parallel()
	archive := newArchiveServer(t, archiveFiles{
		"Skills/swe/SKILL.md": {body: "upstream", mode: 0o644},
	})
	project := t.TempDir()
	writeProjectFile(t, project, "Skills/swe/SKILL.md", "local", 0o644)
	manager := Manager{ProjectRoot: project, Client: archive.Client()}
	_, err := manager.Install(context.Background(), archive.URL, "tag:v1", []string{"Skills/swe"})
	if err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("expected refusal, got %v", err)
	}
	assertFile(t, project, "Skills/swe/SKILL.md", "local")
}

func TestInstallAndUpdateSingleFile(t *testing.T) {
	t.Parallel()
	archive := newArchiveServer(t, archiveFiles{
		"HARNESS.md": {body: "# Harness", mode: 0o644},
	})
	project := t.TempDir()
	manager := Manager{ProjectRoot: project, Client: archive.Client()}
	if _, err := manager.Install(context.Background(), archive.URL, "tag:v1", []string{"HARNESS.md"}); err != nil {
		t.Fatal(err)
	}
	record, err := state.Load(filepath.Join(project, ".harness"))
	if err != nil {
		t.Fatal(err)
	}
	if record.Downloads["HARNESS.md"].Kind != "file" {
		t.Fatalf("unexpected kind: %s", record.Downloads["HARNESS.md"].Kind)
	}
	archive.Set(archiveFiles{
		"HARNESS.md": {body: "# Harness v2", mode: 0o644},
	})
	results, err := manager.Update(context.Background(), []string{"HARNESS.md"}, "tag:v2")
	if err != nil {
		t.Fatal(err)
	}
	if !results[0].Changed {
		t.Fatal("expected file update")
	}
	assertFile(t, project, "HARNESS.md", "# Harness v2")
}

func TestConcurrentInstallIsRejectedByProjectLock(t *testing.T) {
	t.Parallel()
	archive := newArchiveServer(t, archiveFiles{
		"Skills/swe/SKILL.md":   {body: "swe", mode: 0o644},
		"Skills/other/SKILL.md": {body: "other", mode: 0o644},
	})
	project := t.TempDir()
	started := make(chan struct{})
	release := make(chan struct{})
	var startOnce sync.Once
	blockingClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		startOnce.Do(func() { close(started) })
		<-release
		return archive.Client().Transport.RoundTrip(request)
	})}

	firstResult := make(chan error, 1)
	go func() {
		_, err := (&Manager{ProjectRoot: project, Client: blockingClient}).Install(
			context.Background(), archive.URL, "tag:v1", []string{"Skills/swe"},
		)
		firstResult <- err
	}()
	waitForSignal(t, started, release)

	_, err := (&Manager{ProjectRoot: project, Client: archive.Client()}).Install(
		context.Background(), archive.URL, "tag:v1", []string{"Skills/other"},
	)
	if !errors.Is(err, ErrOperationInProgress) {
		close(release)
		<-firstResult
		t.Fatalf("expected lock conflict, got %v", err)
	}
	close(release)
	if err := <-firstResult; err != nil {
		t.Fatal(err)
	}

	record, err := state.Load(filepath.Join(project, ".harness"))
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := record.Downloads["Skills/swe"]; !exists {
		t.Fatal("first install was not recorded")
	}
	if _, exists := record.Downloads["Skills/other"]; exists {
		t.Fatal("lock-conflicting install changed the record")
	}
}

func TestConcurrentUpdateIsRejectedAndCanBeRetried(t *testing.T) {
	t.Parallel()
	archive := newArchiveServer(t, archiveFiles{
		"Skills/swe/SKILL.md":   {body: "one", mode: 0o644},
		"Skills/other/SKILL.md": {body: "one", mode: 0o644},
	})
	project := t.TempDir()
	manager := &Manager{ProjectRoot: project, Client: archive.Client()}
	if _, err := manager.Install(context.Background(), archive.URL, "tag:v1", []string{"Skills/swe", "Skills/other"}); err != nil {
		t.Fatal(err)
	}
	archive.Set(archiveFiles{
		"Skills/swe/SKILL.md":   {body: "two", mode: 0o644},
		"Skills/other/SKILL.md": {body: "two", mode: 0o644},
	})

	started := make(chan struct{})
	release := make(chan struct{})
	var startOnce sync.Once
	blockingClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		startOnce.Do(func() { close(started) })
		<-release
		return archive.Client().Transport.RoundTrip(request)
	})}
	firstResult := make(chan error, 1)
	go func() {
		_, err := (&Manager{ProjectRoot: project, Client: blockingClient}).Update(
			context.Background(), []string{"Skills/swe"}, "tag:v2",
		)
		firstResult <- err
	}()
	waitForSignal(t, started, release)

	_, err := manager.Update(context.Background(), []string{"Skills/other"}, "tag:v2")
	if !errors.Is(err, ErrOperationInProgress) {
		close(release)
		<-firstResult
		t.Fatalf("expected lock conflict, got %v", err)
	}
	close(release)
	if err := <-firstResult; err != nil {
		t.Fatal(err)
	}

	if _, err := manager.Update(context.Background(), []string{"Skills/other"}, "tag:v2"); err != nil {
		t.Fatalf("retry after lock release: %v", err)
	}
	record, err := state.Load(filepath.Join(project, ".harness"))
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"Skills/swe", "Skills/other"} {
		if record.Downloads[target].Version != "tag:v2" {
			t.Fatalf("%s version = %q", target, record.Downloads[target].Version)
		}
	}
}

func TestNormalizeTargetsAcceptsWindowsSeparators(t *testing.T) {
	t.Parallel()
	targets, err := normalizeTargets([]string{`Skills\swe`, "Skills/swe/"})
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || targets[0] != "Skills/swe" {
		t.Fatalf("unexpected targets: %#v", targets)
	}
}

type archiveFile struct {
	body string
	mode int64
}

type archiveFiles map[string]archiveFile

type archiveServer struct {
	mu      sync.RWMutex
	payload []byte
	URL     string
}

func newArchiveServer(t *testing.T, files archiveFiles) *archiveServer {
	t.Helper()
	return &archiveServer{
		payload: makeArchive(t, files),
		URL:     "https://example.test/archive.tar.gz",
	}
}

func (s *archiveServer) Client() *http.Client {
	return &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		s.mu.RLock()
		current := append([]byte(nil), s.payload...)
		s.mu.RUnlock()
		return &http.Response{
			StatusCode:    http.StatusOK,
			Status:        "200 OK",
			Body:          io.NopCloser(bytes.NewReader(current)),
			ContentLength: int64(len(current)),
			Header:        make(http.Header),
			Request:       request,
		}, nil
	})}
}

func (s *archiveServer) Set(files archiveFiles) {
	s.mu.Lock()
	s.payload = makeArchiveWithoutTest(files)
	s.mu.Unlock()
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func makeArchive(t *testing.T, files archiveFiles) []byte {
	t.Helper()
	return makeArchiveWithoutTest(files)
}

func makeArchiveWithoutTest(files archiveFiles) []byte {
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sortStrings(names)
	for _, name := range names {
		file := files[name]
		header := &tar.Header{
			Name:     "Harness-main/" + name,
			Mode:     file.mode,
			Size:     int64(len(file.body)),
			Typeflag: tar.TypeReg,
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			panic(err)
		}
		if _, err := tarWriter.Write([]byte(file.body)); err != nil {
			panic(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		panic(err)
	}
	if err := gzipWriter.Close(); err != nil {
		panic(err)
	}
	return buffer.Bytes()
}

func sortStrings(values []string) {
	for left := 0; left < len(values); left++ {
		for right := left + 1; right < len(values); right++ {
			if values[right] < values[left] {
				values[left], values[right] = values[right], values[left]
			}
		}
	}
}

func assertFile(t *testing.T, project, name, want string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(project, ".harness", filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("%s = %q, want %q", name, data, want)
	}
}

func writeProjectFile(t *testing.T, project, name, body string, mode os.FileMode) {
	t.Helper()
	target := filepath.Join(project, ".harness", filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
}

func waitForSignal(t *testing.T, signal <-chan struct{}, release chan<- struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("timed out waiting for the first operation to hold the project lock")
	}
}
