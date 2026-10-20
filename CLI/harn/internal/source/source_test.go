package source

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveArchiveURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		source  string
		version string
		want    string
		wantErr bool
	}{
		{
			name:    "branch",
			source:  "https://github.com/AwayerCode/Harn.git",
			version: "branch:main",
			want:    "https://github.com/AwayerCode/Harn/archive/refs/heads/main.tar.gz",
		},
		{
			name:    "tag with slash",
			source:  "https://github.com/AwayerCode/Harn",
			version: "tag:release/2026.10",
			want:    "https://github.com/AwayerCode/Harn/archive/refs/tags/release/2026.10.tar.gz",
		},
		{
			name:    "branch requiring escaping",
			source:  "https://github.com/AwayerCode/Harn",
			version: "branch:feature/new skill",
			want:    "https://github.com/AwayerCode/Harn/archive/refs/heads/feature/new%20skill.tar.gz",
		},
		{
			name:    "commit",
			source:  "https://github.com/AwayerCode/Harn",
			version: "commit:0123456789abcdef",
			want:    "https://github.com/AwayerCode/Harn/archive/0123456789abcdef.tar.gz",
		},
		{
			name:    "direct archive",
			source:  "https://example.com/content.tar.gz?token=one",
			version: "tag:v1",
			want:    "https://example.com/content.tar.gz?token=one",
		},
		{
			name:    "unqualified version",
			source:  "https://github.com/AwayerCode/Harn",
			version: "main",
			wantErr: true,
		},
		{
			name:    "unsupported host",
			source:  "https://example.com/AwayerCode/Harn",
			version: "branch:main",
			wantErr: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := ResolveArchiveURL(test.source, test.version)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestFetchSelectsPathsAndPreservesExecutableMode(t *testing.T) {
	t.Parallel()
	archive := testArchive(t, []testFile{
		{name: "._Harn-main", body: "apple double", mode: 0o644},
		{name: "Harn-main/Skills/swe/SKILL.md", body: "skill", mode: 0o644},
		{name: "Harn-main/Skills/swe/scripts/check.py", body: "#!/usr/bin/env python3\n", mode: 0o755},
		{name: "Harn-main/Skills/other/SKILL.md", body: "other", mode: 0o644},
	})
	client := archiveClient(archive)

	destination := t.TempDir()
	err := Fetch(context.Background(), client, "https://example.test/archive.tar.gz", []string{"Skills/swe"}, destination)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(destination, "Skills", "swe", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "skill" {
		t.Fatalf("unexpected content %q", body)
	}
	info, err := os.Stat(filepath.Join(destination, "Skills", "swe", "scripts", "check.py"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("expected executable mode, got %o", info.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(destination, "Skills", "other")); !os.IsNotExist(err) {
		t.Fatalf("unselected path should not be extracted: %v", err)
	}
}

func TestFetchIgnoresTopLevelArchiveMetadata(t *testing.T) {
	t.Parallel()
	archive := testArchive(t, []testFile{
		{name: "pax_global_header", body: "github archive metadata", mode: 0o644},
		{name: "Harn-main/Skills/swe/SKILL.md", body: "skill", mode: 0o644},
	})

	destination := t.TempDir()
	err := Fetch(context.Background(), archiveClient(archive), "https://example.test/archive.tar.gz", []string{"Skills/swe"}, destination)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(destination, "Skills", "swe", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "skill" {
		t.Fatalf("unexpected content %q", body)
	}
}

func TestFetchSupportsRootlessArchive(t *testing.T) {
	t.Parallel()
	archive := testArchive(t, []testFile{
		{name: "Skills/swe/SKILL.md", body: "skill", mode: 0o644},
		{name: "Skills/swe/scripts/check.py", body: "#!/usr/bin/env python3\n", mode: 0o755},
	})

	destination := t.TempDir()
	err := Fetch(context.Background(), archiveClient(archive), "https://example.test/archive.tar.gz", []string{"Skills/swe"}, destination)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(destination, "Skills", "swe", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "skill" {
		t.Fatalf("unexpected content %q", body)
	}
	info, err := os.Stat(filepath.Join(destination, "Skills", "swe", "scripts", "check.py"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("expected executable mode, got %o", info.Mode().Perm())
	}
}

func TestFetchSupportsSingleFileArchive(t *testing.T) {
	t.Parallel()
	archive := testArchive(t, []testFile{
		{name: "HARN.md", body: "# Harn", mode: 0o644},
	})

	destination := t.TempDir()
	err := Fetch(context.Background(), archiveClient(archive), "https://example.test/archive.tar.gz", []string{"HARN.md"}, destination)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(destination, "HARN.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "# Harn" {
		t.Fatalf("unexpected content %q", body)
	}
}

func TestFetchRejectsUnsafeArchivePath(t *testing.T) {
	t.Parallel()
	archive := testArchive(t, []testFile{
		{name: "../outside", body: "bad", mode: 0o644},
	})
	err := Fetch(context.Background(), archiveClient(archive), "https://example.test/archive.tar.gz", []string{"Skills/swe"}, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "unsafe archive path") {
		t.Fatalf("expected unsafe path error, got %v", err)
	}
}

func TestFetchRejectsSelectedSymlink(t *testing.T) {
	t.Parallel()
	archive := testArchive(t, []testFile{
		{name: "Harn-main/Skills/swe/link", linkname: "../../outside", mode: 0o777, typeflag: tar.TypeSymlink},
	})
	err := Fetch(context.Background(), archiveClient(archive), "https://example.test/archive.tar.gz", []string{"Skills/swe"}, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "not a regular file or directory") {
		t.Fatalf("expected symlink error, got %v", err)
	}
}

func TestFetchReportsMissingTarget(t *testing.T) {
	t.Parallel()
	archive := testArchive(t, []testFile{
		{name: "Harn-main/Skills/swe/SKILL.md", body: "skill", mode: 0o644},
	})
	err := Fetch(context.Background(), archiveClient(archive), "https://example.test/archive.tar.gz", []string{"Skills/missing"}, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "was not found") {
		t.Fatalf("expected missing target error, got %v", err)
	}
}

type testFile struct {
	name     string
	body     string
	mode     int64
	typeflag byte
	linkname string
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func archiveClient(payload []byte) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    http.StatusOK,
			Status:        "200 OK",
			Body:          io.NopCloser(bytes.NewReader(payload)),
			ContentLength: int64(len(payload)),
			Header:        make(http.Header),
			Request:       request,
		}, nil
	})}
}

func testArchive(t *testing.T, files []testFile) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, file := range files {
		typeflag := file.typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}
		header := &tar.Header{
			Name:     file.name,
			Mode:     file.mode,
			Typeflag: typeflag,
			Linkname: file.linkname,
		}
		if typeflag == tar.TypeReg || typeflag == byte(0) {
			header.Size = int64(len(file.body))
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if typeflag == tar.TypeReg || typeflag == byte(0) {
			if _, err := tarWriter.Write([]byte(file.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
