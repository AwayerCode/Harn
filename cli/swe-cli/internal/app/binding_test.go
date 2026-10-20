package app

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRuntimeBinding(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if e := os.Mkdir(bin, 0700); e != nil {
		t.Fatal(e)
	}
	file := filepath.Join(bin, "swe-cli")
	write(t, file, "binary")
	binding := func(version, digest string) {
		b, _ := json.Marshal(map[string]any{"schema_version": 1, "name": "swe-cli", "version": version, "assets": map[string]any{runtime.GOOS + "/" + runtime.GOARCH: map[string]string{"sha256": digest}}})
		write(t, filepath.Join(root, "runtime.json"), string(b))
	}
	sum := fmt.Sprintf("%x", sha256.Sum256([]byte("binary")))
	binding(Version, sum)
	if e := verifyBinding(file); e != nil {
		t.Fatal(e)
	}
	binding("old", sum)
	if e := verifyBinding(file); e == nil {
		t.Fatal("wrong version accepted")
	}
	binding(Version, sum)
	write(t, file, "tampered")
	if e := verifyBinding(file); e == nil {
		t.Fatal("tampered executable accepted")
	}
}
