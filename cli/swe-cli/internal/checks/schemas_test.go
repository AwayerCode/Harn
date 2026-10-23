package checks

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestGeneratedSchemasMatchSource(t *testing.T) {
	var generated map[string]any
	if e := json.Unmarshal([]byte(SchemaJSON), &generated); e != nil {
		t.Fatal(e)
	}
	files, e := filepath.Glob("../../../../skills/swe/references/artifact-schemas/*.json")
	if e != nil || len(files) == 0 {
		t.Fatal("missing canonical schemas", e)
	}
	canonical := map[string]any{}
	for _, p := range files {
		b, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		var s any
		if e = json.Unmarshal(b, &s); e != nil {
			t.Fatal(e)
		}
		name := filepath.Base(p)
		canonical[name[:len(name)-5]] = s
	}
	if !reflect.DeepEqual(generated, canonical) {
		t.Fatal("run go generate ./... to refresh compiled schemas")
	}
}

func TestArchitectureTemplate(t *testing.T) {
	root := "../../../../skills/swe"
	var schemas map[string]Schema
	if e := json.Unmarshal([]byte(SchemaJSON), &schemas); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(filepath.Join(root, "system-design/assets/architecture-design.md"))
	if e != nil {
		t.Fatal(e)
	}
	headings := []string{}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "## ") {
			headings = append(headings, line[3:])
		}
	}
	if !reflect.DeepEqual(headings, schemas["system-design"].Profiles["architecture"]) {
		t.Fatal("architecture template differs from owned sections")
	}
}

func TestSkillLinksResolve(t *testing.T) {
	root := "../../../../skills"
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		b, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		ignored := []string{}
		for _, m := range linkRE.FindAllStringSubmatch(strings.Join(Visible(string(b), &ignored), "\n"), -1) {
			u, e := url.Parse(strings.Trim(m[1], "<>"))
			if e != nil {
				return e
			}
			if u.Scheme != "" || u.Host != "" || u.Path == "" {
				continue
			}
			target := filepath.Join(filepath.Dir(path), filepath.FromSlash(u.Path))
			if _, e = os.Stat(target); e != nil {
				t.Errorf("%s: %s: %v", path, m[1], e)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestUnifiedSWERuntimeAndEntries(t *testing.T) {
	root := "../../../../skills/swe"
	entries, err := filepath.Glob(filepath.Join(root, "*", "SKILL.md"))
	if err != nil || len(entries) != 5 {
		t.Fatalf("expected five domain entries, got %v: %v", entries, err)
	}
	runtimes := []string{}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Name() == "runtime.json" {
			runtimes = append(runtimes, path)
		}
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "swe-") {
			t.Errorf("obsolete domain directory: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(runtimes) != 0 {
		t.Fatalf("runtime bindings are generated only inside platform Skill packages, found in source: %v", runtimes)
	}
}

func TestUnifiedSWEPackageClosure(t *testing.T) {
	for _, domain := range []string{"business-model", "system-design", "implementation", "testing", "delivery"} {
		t.Run(domain, func(t *testing.T) {
			root := t.TempDir()
			for _, name := range []string{"swe"} {
				if err := os.CopyFS(filepath.Join(root, name), os.DirFS(filepath.Join("../../../../skills", name))); err != nil {
					t.Fatal(err)
				}
			}
			var contract struct {
				Version int               `json:"schema_version"`
				Shared  string            `json:"shared_resource"`
				Inputs  []string          `json:"inputs"`
				Outputs []string          `json:"outputs"`
				Schemas map[string]string `json:"schemas"`
			}
			b, err := os.ReadFile(filepath.Join(root, "swe", domain, "contract.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err = json.Unmarshal(b, &contract); err != nil {
				t.Fatal(err)
			}
			if contract.Version != 1 || contract.Shared != ".." {
				t.Fatal("unsupported domain contract")
			}
			for _, kind := range append(contract.Inputs, contract.Outputs...) {
				ref, ok := contract.Schemas[kind]
				if !ok {
					t.Fatalf("missing schema: %s", kind)
				}
				data, err := os.ReadFile(filepath.Join(root, "swe", domain, ref))
				if err != nil {
					t.Fatal(err)
				}
				if !json.Valid(data) {
					t.Fatalf("invalid schema: %s", ref)
				}
			}
			seen := map[string]bool{}
			var visit func(string)
			visit = func(path string) {
				path = filepath.Clean(path)
				if seen[path] {
					return
				}
				seen[path] = true
				data, err := os.ReadFile(path)
				if err != nil {
					t.Error(err)
					return
				}
				ignored := []string{}
				for _, match := range linkRE.FindAllStringSubmatch(strings.Join(Visible(string(data), &ignored), "\n"), -1) {
					u, err := url.Parse(strings.Trim(match[1], "<>"))
					if err != nil {
						t.Error(err)
						continue
					}
					if u.Scheme != "" || u.Host != "" || u.Path == "" {
						continue
					}
					target := filepath.Join(filepath.Dir(path), filepath.FromSlash(u.Path))
					if _, err := os.Stat(target); err != nil {
						t.Errorf("%s: unavailable installed reference %s", path, match[1])
						continue
					}
					if filepath.Ext(target) == ".md" {
						visit(target)
					}
				}
			}
			visit(filepath.Join(root, "swe", domain, "SKILL.md"))
			visit(filepath.Join(root, "swe", "SKILL.md"))
		})
	}
}
