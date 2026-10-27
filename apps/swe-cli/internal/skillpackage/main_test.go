package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlatformPackagesContainCompleteSkillAndLocalCLI(t *testing.T) {
	root, dist, out := t.TempDir(), t.TempDir(), t.TempDir()
	skill := filepath.Join(root, "skills", "swe")
	write := func(name, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, "skills", "3rdparty", "external", "SKILL.md"), "third-party must not ship in SWE")
	write(filepath.Join(skill, "SKILL.md"), "---\nname: swe\ndescription: test\n---\n")
	write(filepath.Join(skill, "testing", "SKILL.md"), "testing entry")
	write(filepath.Join(skill, "adoption", "SKILL.md"), "adoption entry")
	write(filepath.Join(skill, "adoption", "references", "lifecycle.md"), "lifecycle")
	write(filepath.Join(skill, "references", "rule.md"), "rule")
	write(filepath.Join(skill, "runtime.json"), `{"url":"https://old-release.invalid"}`)
	write(filepath.Join(skill, "bin", "stale"), "must not ship")
	for _, platform := range platforms {
		name := "swe-cli_" + strings.ReplaceAll(platform, "/", "_")
		if strings.HasPrefix(platform, "windows/") {
			name += ".exe"
		}
		write(filepath.Join(dist, name), "binary "+platform)
	}
	if err := buildPackages(root, dist, out, "test-version"); err != nil {
		t.Fatal(err)
	}
	for _, platform := range platforms {
		t.Run(platform, func(t *testing.T) {
			archive := filepath.Join(out, "swe_"+strings.ReplaceAll(platform, "/", "_")+".tar.gz")
			file, err := os.Open(archive)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			gz, err := gzip.NewReader(file)
			if err != nil {
				t.Fatal(err)
			}
			defer gz.Close()
			tr := tar.NewReader(gz)
			files := map[string][]byte{}
			modes := map[string]int64{}
			for {
				header, err := tr.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				data, err := io.ReadAll(tr)
				if err != nil {
					t.Fatal(err)
				}
				files[header.Name] = data
				modes[header.Name] = header.Mode
			}
			executable := "skills/swe/bin/swe-cli"
			if strings.HasPrefix(platform, "windows/") {
				executable += ".exe"
			}
			if string(files[executable]) != "binary "+platform || modes[executable]&0111 == 0 {
				t.Fatal("missing executable or mode")
			}
			if len(files) != 7 || string(files["skills/swe/adoption/SKILL.md"]) != "adoption entry" || string(files["skills/swe/adoption/references/lifecycle.md"]) != "lifecycle" || string(files["skills/swe/testing/SKILL.md"]) != "testing entry" || string(files["skills/swe/references/rule.md"]) != "rule" {
				t.Fatalf("incomplete package: %v", files)
			}
			var binding struct {
				Version string                       `json:"version"`
				Assets  map[string]map[string]string `json:"assets"`
			}
			if err := json.Unmarshal(files["skills/swe/runtime.json"], &binding); err != nil {
				t.Fatal(err)
			}
			if binding.Version != "test-version" || len(binding.Assets) != 1 || binding.Assets[platform]["sha256"] != fmt.Sprintf("%x", sha256.Sum256(files[executable])) {
				t.Fatalf("binding: %+v", binding)
			}
			if strings.Contains(string(files["skills/swe/runtime.json"]), "url") {
				t.Fatal("download URL leaked into package")
			}
			sums, err := os.ReadFile(filepath.Join(out, "SHA256SUMS"))
			if err != nil {
				t.Fatal(err)
			}
			packed, err := os.ReadFile(archive)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(sums), fmt.Sprintf("%x  %s", sha256.Sum256(packed), filepath.Base(archive))) {
				t.Fatal("package checksum mismatch")
			}
		})
	}
	// Source files are never rewritten by packaging.
	data, _ := os.ReadFile(filepath.Join(skill, "runtime.json"))
	if !strings.Contains(string(data), "old-release") {
		t.Fatal("source binding modified")
	}
}

func TestMissingBinaryDoesNotProduceSkillArchive(t *testing.T) {
	out := t.TempDir()
	if err := buildPackage(t.TempDir(), filepath.Join(t.TempDir(), "missing"), filepath.Join(out, "swe.tar.gz"), "v1", "darwin/arm64", "swe-cli"); err == nil {
		t.Fatal("missing binary accepted")
	}
	entries, _ := os.ReadDir(out)
	if len(entries) != 0 {
		t.Fatalf("partial package: %v", entries)
	}
}
