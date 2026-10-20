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

func TestModuleContractsAndArchitectureTemplate(t *testing.T) {
	root := "../../../../skills/swe"
	var schemas map[string]Schema
	if e := json.Unmarshal([]byte(SchemaJSON), &schemas); e != nil {
		t.Fatal(e)
	}
	files, e := filepath.Glob(filepath.Join(root, "*/references/*contract.json"))
	if e != nil || len(files) == 0 {
		t.Fatal(e)
	}
	for _, p := range files {
		b, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		var c struct {
			Version   int                          `json:"schema_version"`
			Documents map[string]map[string]string `json:"documents"`
		}
		if e = json.Unmarshal(b, &c); e != nil {
			t.Fatal(e)
		}
		if c.Version != 1 {
			t.Fatal(p, "unsupported schema")
		}
		for kind, spec := range c.Documents {
			if len(spec) != 1 || spec["$ref"] != kind+".json" {
				t.Fatal(p, kind, spec)
			}
			if _, ok := schemas[kind]; !ok {
				t.Fatal("missing canonical schema", kind)
			}
		}
	}
	b, e := os.ReadFile(filepath.Join(root, "swe-architecture-design/assets/architecture-design.md"))
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
