package installer

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCompanionLifecycle(t *testing.T) {
	payload := "companion v1"
	status := http.StatusOK
	binary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status); fmt.Fprint(w, payload) }))
	defer binary.Close()
	binding := func() string {
		data, _ := json.Marshal(RuntimeBinding{SchemaVersion: 1, Name: "swe-cli", Version: "test", Assets: map[string]RuntimeAsset{runtime.GOOS + "/" + runtime.GOARCH: {URL: binary.URL, SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(payload)))}}})
		return string(data)
	}
	archive := newArchiveServer(t, archiveFiles{"skills/swe/SKILL.md": {body: "one", mode: 0644}, "skills/swe/runtime.json": {body: binding(), mode: 0644}})
	project := t.TempDir()
	client := archive.Client()
	client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.HasPrefix(r.URL.String(), binary.URL) {
			return http.DefaultTransport.RoundTrip(r)
		}
		return archive.Client().Transport.RoundTrip(r)
	})
	manager := Manager{ProjectRoot: project, Client: client}
	name := "skills/swe/bin/swe-cli"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if _, e := manager.Install(context.Background(), archive.URL, "tag:v1", []string{"skills/swe"}); e != nil {
		t.Fatal(e)
	}
	assertFile(t, project, name, payload)
	payload = "companion v2"
	archive.Set(archiveFiles{"skills/swe/SKILL.md": {body: "two", mode: 0644}, "skills/swe/runtime.json": {body: binding(), mode: 0644}})
	status = 404
	if _, e := manager.Update(context.Background(), []string{"skills/swe"}, "tag:v2"); e == nil {
		t.Fatal("missing asset accepted")
	}
	assertFile(t, project, name, "companion v1")
	assertFile(t, project, "skills/swe/SKILL.md", "one")
	status = 200
	payload = "wrong bytes"
	if _, e := manager.Update(context.Background(), []string{"skills/swe"}, "tag:v2"); e == nil || !strings.Contains(e.Error(), "SHA256") {
		t.Fatal(e)
	}
	assertFile(t, project, name, "companion v1")
	payload = "companion v2"
	if _, e := manager.Update(context.Background(), []string{"skills/swe"}, "tag:v2"); e != nil {
		t.Fatal(e)
	}
	assertFile(t, project, name, payload)
	writeProjectFile(t, project, name, "local", 0755)
	if _, e := manager.Update(context.Background(), []string{"skills/swe"}, "tag:v2"); e == nil {
		t.Fatal("modified companion overwritten")
	}
}
func TestCompanionFailedInstallWritesNoResource(t *testing.T) {
	data, _ := json.Marshal(RuntimeBinding{SchemaVersion: 1, Name: "swe-cli", Version: "test", Assets: map[string]RuntimeAsset{}})
	archive := newArchiveServer(t, archiveFiles{"skills/swe/runtime.json": {body: string(data), mode: 0644}})
	project := t.TempDir()
	manager := Manager{ProjectRoot: project, Client: archive.Client()}
	if _, e := manager.Install(context.Background(), archive.URL, "tag:v1", []string{"skills/swe"}); e == nil {
		t.Fatal("missing platform accepted")
	}
	if _, e := os.Stat(filepath.Join(project, ".harness", "skills", "swe")); !os.IsNotExist(e) {
		t.Fatal("partial install", e)
	}
}
