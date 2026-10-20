package installer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AwayerCode/Harn/CLI/harn/internal/source"
)

const indexTarget = "Skills/index.json"

type Skill struct {
	Name    string   `json:"name"`
	Path    string   `json:"path"`
	Summary string   `json:"summary"`
	Tags    []string `json:"tags"`
}

type catalog struct {
	SchemaVersion int     `json:"schema_version"`
	Skills        []Skill `json:"skills"`
}

func (m Manager) Search(ctx context.Context, rawSource, version, query string) ([]Skill, error) {
	if m.Client == nil {
		return nil, errors.New("HTTP client is required")
	}
	if rawSource == "" {
		return nil, errors.New("source is required")
	}
	if version == "" {
		version = "branch:main"
	}
	archiveURL, err := source.ResolveArchiveURL(rawSource, version)
	if err != nil {
		return nil, err
	}
	stage, err := os.MkdirTemp("", "harn-search-*")
	if err != nil {
		return nil, fmt.Errorf("create staging directory: %w", err)
	}
	defer os.RemoveAll(stage)
	contentRoot := filepath.Join(stage, "content")
	if err := source.Fetch(ctx, m.Client, archiveURL, []string{indexTarget}, contentRoot); err != nil {
		return nil, fmt.Errorf("read resource index (does the source publish %s?): %w", indexTarget, err)
	}
	data, err := os.ReadFile(filepath.Join(contentRoot, filepath.FromSlash(indexTarget)))
	if err != nil {
		return nil, fmt.Errorf("read resource index: %w", err)
	}
	skills, err := parseCatalog(data)
	if err != nil {
		return nil, err
	}

	needle := strings.ToLower(strings.TrimSpace(query))
	matches := make([]Skill, 0, len(skills))
	for _, skill := range skills {
		if needle == "" || matchesQuery(skill, needle) {
			matches = append(matches, skill)
		}
	}
	sort.Slice(matches, func(left, right int) bool {
		return matches[left].Name < matches[right].Name
	})
	return matches, nil
}

func parseCatalog(data []byte) ([]Skill, error) {
	var parsed catalog
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("parse resource index: %w", err)
	}
	if parsed.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported resource index schema %d", parsed.SchemaVersion)
	}
	seen := make(map[string]bool, len(parsed.Skills))
	for _, skill := range parsed.Skills {
		if skill.Name == "" || skill.Path == "" {
			return nil, errors.New("resource index entries require name and path")
		}
		if !strings.HasPrefix(skill.Path, "Skills/") {
			return nil, fmt.Errorf("resource index path %q must live under Skills/", skill.Path)
		}
		if seen[skill.Name] {
			return nil, fmt.Errorf("duplicate skill name %q in resource index", skill.Name)
		}
		seen[skill.Name] = true
	}
	return parsed.Skills, nil
}

func matchesQuery(skill Skill, needle string) bool {
	if strings.Contains(strings.ToLower(skill.Name), needle) ||
		strings.Contains(strings.ToLower(skill.Path), needle) ||
		strings.Contains(strings.ToLower(skill.Summary), needle) {
		return true
	}
	for _, tag := range skill.Tags {
		if strings.Contains(strings.ToLower(tag), needle) {
			return true
		}
	}
	return false
}
