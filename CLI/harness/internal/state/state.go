package state

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const (
	SchemaVersion = 1
	FileName      = "downloads.json"
)

type State struct {
	SchemaVersion int                 `json:"schema_version"`
	Downloads     map[string]Download `json:"downloads"`
}

type Download struct {
	Source  string               `json:"source"`
	Version string               `json:"version"`
	Kind    string               `json:"kind"`
	Files   map[string]FileState `json:"files"`
}

type FileState struct {
	SHA256 string `json:"sha256"`
	Mode   string `json:"mode"`
}

func New() *State {
	return &State{SchemaVersion: SchemaVersion, Downloads: make(map[string]Download)}
}

func Load(root string) (*State, error) {
	data, err := os.ReadFile(filepath.Join(root, FileName))
	if errors.Is(err, os.ErrNotExist) {
		return New(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read download record: %w", err)
	}

	state := New()
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(state); err != nil {
		return nil, fmt.Errorf("parse download record: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, errors.New("parse download record: trailing JSON data")
	}
	if state.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("unsupported download record schema %d", state.SchemaVersion)
	}
	if state.Downloads == nil {
		state.Downloads = make(map[string]Download)
	}
	if err := state.Validate(); err != nil {
		return nil, fmt.Errorf("invalid download record: %w", err)
	}
	return state, nil
}

func (s *State) Save(root string) error {
	if s.SchemaVersion != SchemaVersion {
		s.SchemaVersion = SchemaVersion
	}
	if s.Downloads == nil {
		s.Downloads = make(map[string]Download)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode download record: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("create .harness directory: %w", err)
	}
	target := filepath.Join(root, FileName)
	temp, err := os.CreateTemp(root, ".downloads-*.json")
	if err != nil {
		return fmt.Errorf("create download record: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return fmt.Errorf("write download record: %w", err)
	}
	if err := temp.Chmod(0o644); err != nil {
		temp.Close()
		return fmt.Errorf("set download record permissions: %w", err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return fmt.Errorf("sync download record: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close download record: %w", err)
	}
	if err := os.Rename(tempName, target); err != nil {
		return fmt.Errorf("replace download record: %w", err)
	}
	return nil
}

func (s *State) Validate() error {
	for target, download := range s.Downloads {
		if err := validateTarget(target); err != nil {
			return fmt.Errorf("%q: %w", target, err)
		}
		if download.Source == "" {
			return fmt.Errorf("%q has no source", target)
		}
		if download.Version == "" {
			return fmt.Errorf("%q has no version", target)
		}
		if download.Kind != "file" && download.Kind != "directory" {
			return fmt.Errorf("%q has invalid kind %q", target, download.Kind)
		}
		for file := range download.Files {
			if err := validateRecordedFile(target, file); err != nil {
				return fmt.Errorf("%q: %w", target, err)
			}
		}
	}
	targets := Targets(s)
	for left := 0; left < len(targets); left++ {
		for right := left + 1; right < len(targets); right++ {
			if targets[left] == targets[right] ||
				strings.HasPrefix(targets[left], targets[right]+"/") ||
				strings.HasPrefix(targets[right], targets[left]+"/") {
				return fmt.Errorf("recorded paths %q and %q overlap", targets[left], targets[right])
			}
		}
	}
	return nil
}

func Targets(s *State) []string {
	targets := make([]string, 0, len(s.Downloads))
	for target := range s.Downloads {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	return targets
}

func validateTarget(target string) error {
	if target == "" || target == "." || strings.Contains(target, "\\") {
		return fmt.Errorf("invalid resource path %q", target)
	}
	clean := path.Clean(target)
	if clean != target || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("invalid resource path %q", target)
	}
	if clean == ".harness" || strings.HasPrefix(clean, ".harness/") {
		return fmt.Errorf("resource path %q is reserved", target)
	}
	if clean == "bin" || strings.HasPrefix(clean, "bin/") || clean == "downloads.json" {
		return fmt.Errorf("resource path %q conflicts with the Harness directory", target)
	}
	for _, part := range strings.Split(clean, "/") {
		if part == "" || part == "." || part == ".." || part == ".git" {
			return fmt.Errorf("invalid resource path %q", target)
		}
	}
	return nil
}

func validateRecordedFile(target, file string) error {
	if file == "" || strings.Contains(file, "\\") || path.IsAbs(file) {
		return fmt.Errorf("invalid recorded file %q", file)
	}
	clean := path.Clean(file)
	if clean != file || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("invalid recorded file %q", file)
	}
	if clean != target && !strings.HasPrefix(clean, target+"/") {
		return fmt.Errorf("recorded file %q is outside %q", file, target)
	}
	return nil
}
