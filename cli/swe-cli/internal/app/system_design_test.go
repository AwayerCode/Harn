package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchitectureCommands(t *testing.T) {
	root := "../systemdesign/testdata/project"
	for _, mode := range []string{"validate", "check", "render"} {
		t.Run(mode, func(t *testing.T) {
			var out bytes.Buffer
			code := Run([]string{"system-design", mode, "--root", root, "--file", "system-design.yaml"}, nil, &out)
			if code != 0 {
				t.Fatalf("%d %s", code, out.String())
			}
			if mode == "render" {
				if !strings.HasPrefix(out.String(), "<!doctype html>") {
					t.Fatal(out.String())
				}
			} else {
				var result map[string]any
				if e := json.Unmarshal(out.Bytes(), &result); e != nil || result["status"] != "passed" {
					t.Fatal(out.String(), e)
				}
			}
		})
	}
}
func TestArchitectureInvalidNeverRenders(t *testing.T) {
	p := filepath.Join(t.TempDir(), "system-design.yaml")
	if e := os.WriteFile(p, []byte("schema_version: 1\n"), 0644); e != nil {
		t.Fatal(e)
	}
	var out bytes.Buffer
	if code := Run([]string{"system-design", "render", "--file", p}, nil, &out); code != 1 || strings.Contains(out.String(), "<!doctype") {
		t.Fatalf("%d %s", code, out.String())
	}
}
func TestArchitectureInvalidArguments(t *testing.T) {
	for _, args := range [][]string{{}, {"unknown"}, {"check"}, {"validate", "--other", "a"}, {"check", "--file", "/missing/system-design.yaml"}} {
		var out bytes.Buffer
		if code := SystemDesign(args, &out); code != 2 {
			t.Fatalf("%v: %d %s", args, code, out.String())
		}
	}
}

func TestArchitectureWorkspaceReturnsUnavailable(t *testing.T) {
	t.Setenv("GOWORK", filepath.Join(t.TempDir(), "external.work"))
	var out bytes.Buffer
	code := Run([]string{"system-design", "check", "--root", "../systemdesign/testdata/project", "--file", "system-design.yaml"}, nil, &out)
	var report struct {
		Status string `json:"status"`
	}
	if e := json.Unmarshal(out.Bytes(), &report); e != nil {
		t.Fatal(e)
	}
	if code != 2 || report.Status != "unavailable" {
		t.Fatalf("%d %s", code, out.String())
	}
}

func TestArchitectureConflictingFilesCannotRender(t *testing.T) {
	b, e := os.ReadFile("../systemdesign/testdata/project/system-design.yaml")
	if e != nil {
		t.Fatal(e)
	}
	file := filepath.Join(t.TempDir(), "system-design.yaml")
	write(t, file, strings.Replace(string(b), "[service.go]", "[service.go, service.go/child.go]", 1))
	for _, mode := range []string{"validate", "render"} {
		var out bytes.Buffer
		code := Run([]string{"system-design", mode, "--file", file}, nil, &out)
		var report struct {
			Status string `json:"status"`
		}
		if e := json.Unmarshal(out.Bytes(), &report); e != nil {
			t.Fatal(e)
		}
		if code != 1 || report.Status != "violations" {
			t.Fatalf("%s: %d %s", mode, code, out.String())
		}
	}
}
