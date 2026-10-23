package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Execute the published examples so documentation cannot silently drift from
// accepted arguments and output contracts. Templates intentionally fail content
// validation, but must produce a normal check result rather than a request error.
func TestSkillCommandContracts(t *testing.T) {
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := os.ReadFile(filepath.Join(root, "skills/swe/references/cli.md"))
	if err != nil {
		t.Fatal(err)
	}
	field := regexp.MustCompile("结果数组为 `([^`]+)`").FindSubmatch(doc)
	if len(field) != 2 {
		t.Fatal("missing documented check result field")
	}
	fixture, err := filepath.Abs("../architecture/testdata/project")
	if err != nil {
		t.Fatal(err)
	}
	repl := strings.NewReplacer(
		"<项目根目录>", root,
		"<模型路径>", "skills/swe/business-model/assets/business-model.md",
		"<Markdown路径>", "skills/swe/system-design/assets/system-design.md",
		"<协议路径>", "skills/swe/system-design/assets/interface-protocol.md",
		"<测试设计路径>", "skills/swe/testing/assets/test-design.md",
		"<相对路径>", "cli/swe-cli",
	)
	count := 0
	for _, line := range strings.Split(string(doc), "\n") {
		if !strings.HasPrefix(line, "<CLI> ") || strings.HasPrefix(line, "<CLI> scope ") {
			continue
		}
		count++
		t.Run(strings.TrimPrefix(line, "<CLI> "), func(t *testing.T) {
			command := strings.TrimPrefix(line, "<CLI> ")
			if strings.HasPrefix(command, "architecture ") {
				command = strings.ReplaceAll(command, "<项目根目录>", fixture)
				command = strings.ReplaceAll(command, "<YAML路径>", "architecture.yaml")
			}
			args := strings.Fields(repl.Replace(command))
			var out bytes.Buffer
			code := Run(args, nil, &out)
			if code != 0 && code != 1 {
				t.Fatalf("example unavailable: %d %s", code, &out)
			}
			if len(args) > 1 && args[0] == "architecture" && args[1] == "render" {
				if code != 0 || !strings.HasPrefix(out.String(), "<!doctype html>") {
					t.Fatal(out.String())
				}
				return
			}
			var result map[string]json.RawMessage
			if err := json.Unmarshal(out.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if args[0] == "version" {
				if len(result["version"]) == 0 {
					t.Fatal(out.String())
				}
				return
			}
			if args[0] == "architecture" {
				if string(result["format"]) != `"swe-architecture-v1"` || string(result["status"]) != `"passed"` {
					t.Fatal(out.String())
				}
				return
			}
			var checks []map[string]json.RawMessage
			if err := json.Unmarshal(result[string(field[1])], &checks); err != nil || len(checks) == 0 {
				t.Fatalf("documented field missing: %s", &out)
			}
			for _, check := range checks {
				for _, key := range []string{"status", "diagnostics", "evidence", "coverage"} {
					if _, ok := check[key]; !ok {
						t.Fatalf("missing %s: %s", key, &out)
					}
				}
			}
		})
	}
	if count == 0 {
		t.Fatal("no CLI examples exercised")
	}
}

func TestArchitectureUnavailableContract(t *testing.T) {
	root := t.TempDir()
	for _, args := range [][]string{
		{}, {"unknown"}, {"check"}, {"validate", "--unknown"},
		{"check", "--file", "unused", "--profile", "architecture"},
		{"validate", "--file", filepath.Join(root, "missing.yaml")},
		{"render", "--file", root}, // opening succeeds on Unix, reading a directory fails
	} {
		var out bytes.Buffer
		code := Run(append([]string{"architecture"}, args...), nil, &out)
		var r struct {
			Format      string `json:"format"`
			Status      string `json:"status"`
			Digest      string `json:"design_sha256"`
			Diagnostics []any  `json:"diagnostics"`
		}
		if err := json.Unmarshal(out.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		if code != 2 || r.Format != "swe-architecture-v1" || r.Status != "unavailable" || r.Digest != "" || len(r.Diagnostics) == 0 {
			t.Fatalf("%v: %d %s", args, code, &out)
		}
	}
}
