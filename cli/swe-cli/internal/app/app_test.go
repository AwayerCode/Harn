package app

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func repository(t *testing.T) string {
	t.Helper()
	p, e := filepath.Abs("../../../..")
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func fixture(t *testing.T) (string, string) {
	t.Helper()
	root, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(filepath.Join(repository(t), "skills/swe/scripts/tests/fixtures/business-model.md"))
	if e != nil {
		t.Fatal(e)
	}
	write(t, filepath.Join(root, "model.md"), string(b))
	return root, string(b)
}
func write(t *testing.T, p, s string) {
	t.Helper()
	if e := os.WriteFile(p, []byte(s), 0600); e != nil {
		t.Fatal(e)
	}
}
func request(root string) map[string]any {
	return map[string]any{"protocolVersion": 2, "taskId": "test", "round": 1, "mode": "design", "projectRoot": root, "checks": []any{map[string]any{"checkId": "documents", "required": true, "input": map[string]any{"kind": "business-model", "file": "model.md"}}}}
}
func run(t *testing.T, r map[string]any) (int, map[string]any) {
	t.Helper()
	b, _ := json.Marshal(r)
	var out bytes.Buffer
	code := Run([]string{"check", "--request", "-"}, bytes.NewReader(b), &out)
	var result map[string]any
	if e := json.Unmarshal(out.Bytes(), &result); e != nil {
		t.Fatal(e, out.String())
	}
	return code, result
}
func TestRequestValidation(t *testing.T) {
	root, _ := fixture(t)
	mutations := []func(map[string]any){func(r map[string]any) { r["protocolVersion"] = 1 }, func(r map[string]any) { r["extensions"] = []any{} }, func(r map[string]any) { r["checks"] = []any{} }, func(r map[string]any) { r["checks"] = append(r["checks"].([]any), r["checks"].([]any)[0]) }, func(r map[string]any) { r["round"] = 0 }, func(r map[string]any) { r["checks"].([]any)[0].(map[string]any)["required"] = false }}
	for i, m := range mutations {
		r := request(root)
		m(r)
		if c, _ := run(t, r); c != 2 {
			t.Fatalf("case %d: %d", i, c)
		}
	}
	for _, s := range []string{"{} {}", strings.Repeat(" ", maxRequest+1), "{"} {
		var out bytes.Buffer
		if c := Run([]string{"check", "--request", "-"}, strings.NewReader(s), &out); c != 2 {
			t.Fatal(c)
		}
	}
}
func TestResultAndOptionalChecks(t *testing.T) {
	root, _ := fixture(t)
	r := request(root)
	c, result := run(t, r)
	if c != 0 {
		t.Fatal(result)
	}
	identity := result["identity"].(map[string]any)
	if len(identity["inputDigest"].(string)) != 64 {
		t.Fatal(identity)
	}
	r["checks"] = append(r["checks"].([]any), map[string]any{"checkId": "scope", "required": false, "input": map[string]any{}})
	if c, result = run(t, r); c != 0 || len(result["summary"].(map[string]any)["warnings"].([]any)) != 1 {
		t.Fatal(result)
	}
	write(t, filepath.Join(root, "model.md"), "# broken\n")
	if c, _ = run(t, r); c != 1 {
		t.Fatal(c)
	}
	r["checks"].([]any)[1].(map[string]any)["required"] = true
	if c, _ = run(t, r); c != 2 {
		t.Fatal(c)
	}
}
func TestObserver(t *testing.T) {
	root, _ := fixture(t)
	p := filepath.Join(root, "model.md")
	o := &Observer{Root: root, Files: map[string]string{}}
	if _, e := o.Read(p); e != nil {
		t.Fatal(e)
	}
	write(t, p, "changed")
	if len(o.Changed()) != 1 {
		t.Fatal("change undetected")
	}
	o = &Observer{Root: root, Files: map[string]string{}}
	missing := filepath.Join(root, "missing")
	if exists, e := o.Exists(missing); exists || e != nil {
		t.Fatal(exists, e)
	}
	write(t, missing, "new")
	if len(o.Changed()) != 1 {
		t.Fatal("new file undetected")
	}
	outside := filepath.Join(t.TempDir(), "external")
	write(t, outside, "external")
	link := filepath.Join(root, "link")
	if e := os.Symlink(outside, link); e == nil {
		if _, e = o.Read(link); e == nil {
			t.Fatal("external symlink accepted")
		}
	}
	write(t, p, string([]byte{255}))
	if _, e := (&Observer{Root: root, Files: map[string]string{}}).Read(p); e == nil {
		t.Fatal("invalid utf8 accepted")
	}
}
func TestPythonParity(t *testing.T) {
	python, e := exec.LookPath("python3")
	if e != nil {
		t.Skip("Python only required for migration parity tests")
	}
	root, text := fixture(t)
	repo := repository(t)
	variants := []string{text, strings.TrimRight(text, "\n"), text + "\n", strings.Replace(text, "# 业务模型", "# 错误标题", 1), strings.Replace(text, "## 业务行为", "## 未知章节", 1), text + "\n```\nx\n```\n", text + "\n[缺失](missing.md)\n"}
	for _, v := range variants {
		write(t, filepath.Join(root, "model.md"), v)
		cmd := exec.Command(python, "-B", filepath.Join(repo, "skills/swe/scripts/validate_documents.py"), "--kind", "business-model", "--file", filepath.Join(root, "model.md"))
		output, err := cmd.CombinedOutput()
		expected := 0
		if err != nil {
			if ex, ok := err.(*exec.ExitError); ok {
				expected = ex.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		if c, result := run(t, request(root)); c != expected {
			t.Fatalf("parity: %d != %d: %s %v", c, expected, output, result)
		}
	}
	for _, kind := range []string{"business-model", "system-design", "interface-protocol", "interaction-standards", "test-design"} {
		r := request(repo)
		file := "skills/swe/scripts/tests/fixtures/" + kind + ".md"
		r["checks"].([]any)[0].(map[string]any)["input"] = map[string]any{"kind": kind, "file": file}
		cmd := exec.Command(python, "-B", filepath.Join(repo, "skills/swe/scripts/validate_documents.py"), "--kind", kind, "--file", filepath.Join(repo, file))
		output, err := cmd.CombinedOutput()
		expected := 0
		if err != nil {
			if ex, ok := err.(*exec.ExitError); ok {
				expected = ex.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		if c, result := run(t, r); c != expected {
			t.Fatalf("%s: %d != %d %s %v", kind, c, expected, output, result)
		}
	}
}

func TestArchitectureProfileAndLinks(t *testing.T) {
	root, text := fixture(t)
	repo := repository(t)
	b, e := os.ReadFile(filepath.Join(repo, "skills/swe/scripts/tests/fixtures/system-design.md"))
	if e != nil {
		t.Fatal(e)
	}
	// A malformed module table is outside the architecture profile's ownership.
	design := strings.Replace(string(b), "| 模块 |", "| 错误 |", 1)
	write(t, filepath.Join(root, "design.md"), design)
	r := request(root)
	input := map[string]any{"kind": "system-design", "file": "design.md", "profile": "architecture"}
	r["checks"].([]any)[0].(map[string]any)["input"] = input
	if c, result := run(t, r); c != 0 {
		t.Fatal(result)
	}
	delete(input, "profile")
	if c, _ := run(t, r); c != 1 {
		t.Fatalf("full validation must detect malformed module table: %d", c)
	}
	write(t, filepath.Join(root, "目标 文档.md"), "# 本地标题\n")
	write(t, filepath.Join(root, "model.md"), text+"\n[链接](<目标 文档.md#本地标题>)\n")
	if c, result := run(t, request(root)); c != 0 {
		t.Fatal(result)
	}
	write(t, filepath.Join(root, "model.md"), text+"\n[链接](<目标 文档.md#错误标题>)\n")
	if c, _ := run(t, request(root)); c != 1 {
		t.Fatal("missing anchor accepted", c)
	}
}
