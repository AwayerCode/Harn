package checks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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
