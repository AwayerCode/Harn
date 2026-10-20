package installer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// RuntimeBinding pins companion bytes to the selected Skill revision. Binaries
// are staged inside the resource and use the same ownership/rollback transaction.
type RuntimeBinding struct {
	SchemaVersion int                     `json:"schema_version"`
	Name          string                  `json:"name"`
	Version       string                  `json:"version"`
	Assets        map[string]RuntimeAsset `json:"assets"`
}
type RuntimeAsset struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

const maxRuntimeBytes = 64 << 20

func (m *Manager) stageRuntime(ctx context.Context, contentRoot, target string) error {
	root := filepath.Join(contentRoot, filepath.FromSlash(target))
	info, e := os.Stat(root)
	if e != nil {
		return e
	}
	if !info.IsDir() {
		return nil
	}
	data, e := os.ReadFile(filepath.Join(root, "runtime.json"))
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	var binding RuntimeBinding
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if e = dec.Decode(&binding); e != nil {
		return fmt.Errorf("invalid runtime binding: %w", e)
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return errors.New("trailing runtime binding data")
	}
	if binding.SchemaVersion != 1 || binding.Name != "swe-cli" || strings.TrimSpace(binding.Version) == "" {
		return errors.New("unsupported runtime binding")
	}
	asset, ok := binding.Assets[runtime.GOOS+"/"+runtime.GOARCH]
	if !ok {
		return fmt.Errorf("runtime %s has no asset for %s/%s", binding.Version, runtime.GOOS, runtime.GOARCH)
	}
	sum, e := hex.DecodeString(asset.SHA256)
	if e != nil || len(sum) != sha256.Size {
		return errors.New("runtime binding requires a SHA256 digest")
	}
	u, e := url.Parse(asset.URL)
	if e != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return errors.New("invalid runtime asset URL")
	}
	name := "swe-cli"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	bin := filepath.Join(root, "bin")
	if _, e := os.Lstat(bin); e == nil {
		return errors.New("resource already contains bin; refusing to replace companion files")
	} else if !os.IsNotExist(e) {
		return e
	}
	request, e := http.NewRequestWithContext(ctx, http.MethodGet, asset.URL, nil)
	if e != nil {
		return e
	}
	response, e := m.Client.Do(request)
	if e != nil {
		return fmt.Errorf("download companion %s: %w", binding.Version, e)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download companion %s: HTTP %s", binding.Version, response.Status)
	}
	if response.ContentLength > maxRuntimeBytes {
		return errors.New("runtime asset too large")
	}
	b, e := io.ReadAll(io.LimitReader(response.Body, maxRuntimeBytes+1))
	if e != nil {
		return e
	}
	if len(b) > maxRuntimeBytes {
		return errors.New("runtime asset too large")
	}
	got := sha256.Sum256(b)
	if !bytes.Equal(sum, got[:]) {
		return errors.New("companion SHA256 mismatch; nothing installed")
	}
	if e = os.Mkdir(bin, 0755); e != nil {
		return e
	}
	return os.WriteFile(filepath.Join(bin, name), b, 0755)
}
