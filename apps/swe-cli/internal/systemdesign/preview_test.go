package systemdesign

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewOnlyServesFreshYAMLRendering(t *testing.T) {
	_, data := fixture(t)
	root := t.TempDir()
	file := filepath.Join(root, Filename)
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	p, r, code := NewPreview(file, "pinned-test")
	if code != 0 {
		t.Fatalf("%d %+v", code, r)
	}
	h := p.Handler("127.0.0.1:12345")
	get := func(method, path, host string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "http://127.0.0.1:12345"+path, nil)
		req.Host = host
		out := httptest.NewRecorder()
		h.ServeHTTP(out, req)
		return out
	}
	first := get("GET", p.Path(), "127.0.0.1:12345")
	scope, err := FileScope(file)
	if err != nil {
		t.Fatal(err)
	}
	_, expected, code := Execute(scope, "render", "", "pinned-test")
	if code != 0 || first.Code != 200 || !bytes.Equal(first.Body.Bytes(), expected) {
		t.Fatal("preview differs from CLI rendering")
	}
	if first.Header().Get("Cache-Control") != "no-store" || !strings.Contains(first.Header().Get("Content-Security-Policy"), "script-src 'sha256-") {
		t.Fatal("missing response protections")
	}
	// A neighbouring HTML file cannot replace or influence the preview.
	if err = os.WriteFile(filepath.Join(root, "preview.html"), []byte("<html>forged</html>"), 0600); err != nil {
		t.Fatal(err)
	}
	if out := get("GET", p.Path(), "127.0.0.1:12345"); !bytes.Equal(out.Body.Bytes(), expected) {
		t.Fatal("HTML file influenced preview")
	}
	for _, tc := range []struct {
		method, path, host string
		status             int
	}{
		{"GET", "/preview.html", "127.0.0.1:12345", 404},
		{"GET", p.Path() + "?file=preview.html", "127.0.0.1:12345", 404},
		{"POST", p.Path(), "127.0.0.1:12345", 405},
		{"GET", p.Path(), "attacker.example", 403},
	} {
		if out := get(tc.method, tc.path, tc.host); out.Code != tc.status {
			t.Fatalf("%+v: %d", tc, out.Code)
		}
	}
	if out := get("HEAD", p.Path(), "127.0.0.1:12345"); out.Code != 200 || out.Body.Len() != 0 {
		t.Fatal("HEAD emitted body")
	}
	changed := bytes.Replace(data, []byte("订单处理"), []byte("更新后的订单职责"), 1)
	if err = os.WriteFile(file, changed, 0600); err != nil {
		t.Fatal(err)
	}
	if out := get("GET", p.Path(), "127.0.0.1:12345"); out.Code != 200 || !strings.Contains(out.Body.String(), "更新后的订单职责") {
		t.Fatal("refresh did not reread YAML")
	}
	if err = os.WriteFile(file, []byte("schema_version: 1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if out := get("GET", p.Path(), "127.0.0.1:12345"); out.Code != 422 || strings.Contains(out.Body.String(), "<!doctype html>") {
		t.Fatal("invalid YAML served stale page")
	}
	if err = os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if out := get("GET", p.Path(), "127.0.0.1:12345"); out.Code != 503 {
		t.Fatal("missing YAML served stale page")
	}
}

func TestPreviewRejectsNonYAMLInputs(t *testing.T) {
	for _, name := range []string{"preview.html", Filename} {
		file := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(file, []byte("<html>handwritten</html>"), 0600); err != nil {
			t.Fatal(err)
		}
		if p, _, code := NewPreview(file, "test"); code == 0 || p != nil {
			t.Fatal("accepted HTML input")
		}
	}
}

func TestPreviewCSPMatchesRenderedScripts(t *testing.T) {
	_, data := fixture(t)
	r := NewReport("render", "test", data)
	d := Parse(data, &r)
	var out bytes.Buffer
	if err := Render(&out, d, r); err != nil {
		t.Fatal(err)
	}
	// html/template may rewrite JavaScript; hashes must use the emitted bytes.
	scripts := strings.Split(out.String(), "<script type=\"application/javascript\">")[1:]
	policy := previewCSP(out.Bytes())
	if len(scripts) == 0 {
		t.Fatal("no scripts")
	}
	for _, s := range scripts {
		sum := sha256.Sum256([]byte(strings.SplitN(s, "</script>", 2)[0]))
		if !strings.Contains(policy, "'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'") {
			t.Fatal("CSP would block emitted script")
		}
	}
}
