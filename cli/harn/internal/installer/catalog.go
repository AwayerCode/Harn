package installer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AwayerCode/Harn/cli/harn/internal/source"
)

type Skill struct {
	Name    string
	Path    string
	Summary string
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
	if err := source.Fetch(ctx, m.Client, archiveURL, []string{"skills/index.json"}, contentRoot); err != nil {
		return nil, fmt.Errorf("read skills/index.json: %w", err)
	}
	skills, err := readCatalog(contentRoot)
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

// Search reads only the catalog; repository tests verify it against SKILL.md.
func readCatalog(root string) ([]Skill, error) {
	data, err := os.ReadFile(filepath.Join(root, "skills", "index.json"))
	if err != nil {
		return nil, err
	}
	var catalog struct {
		SchemaVersion int `json:"schema_version"`
		Skills        []struct {
			Name        string `json:"name"`
			Path        string `json:"path"`
			Description string `json:"description"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("parse skills/index.json: %w", err)
	}
	if catalog.SchemaVersion != 1 || catalog.Skills == nil {
		return nil, errors.New("skills/index.json requires schema_version 1 and a skills array")
	}
	skills := make([]Skill, 0, len(catalog.Skills))
	names, paths := map[string]bool{}, map[string]bool{}
	for _, entry := range catalog.Skills {
		parts := strings.Split(entry.Path, "/")
		if strings.TrimSpace(entry.Name) == "" || strings.TrimSpace(entry.Description) == "" || len(parts) != 2 || parts[0] != "skills" || parts[1] == "" || strings.HasPrefix(parts[1], ".") || strings.HasPrefix(parts[1], "_") || strings.Contains(entry.Path, "\\") {
			return nil, fmt.Errorf("invalid skill entry %q: name, description and public top-level skills path required", entry.Name)
		}
		if names[entry.Name] || paths[entry.Path] {
			return nil, fmt.Errorf("duplicate skill name or path %q", entry.Name)
		}
		names[entry.Name], paths[entry.Path] = true, true
		skills = append(skills, Skill{Name: entry.Name, Path: entry.Path, Summary: strings.Join(strings.Fields(entry.Description), " ")})
	}
	return skills, nil
}

func parseSkill(data []byte, path string) (Skill, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return Skill{}, errors.New("missing YAML front matter")
	}
	end := 1
	for end < len(lines) && lines[end] != "---" {
		end++
	}
	if end == len(lines) {
		return Skill{}, errors.New("unclosed YAML front matter")
	}
	var metadata struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &metadata); err != nil {
		return Skill{}, fmt.Errorf("parse YAML front matter: %w", err)
	}
	name := strings.TrimSpace(metadata.Name)
	summary := strings.Join(strings.Fields(metadata.Description), " ")
	if name == "" || summary == "" {
		return Skill{}, errors.New("front matter requires name and description")
	}
	return Skill{Name: name, Path: path, Summary: summary}, nil
}

func matchesQuery(skill Skill, needle string) bool {
	return strings.Contains(strings.ToLower(skill.Name), needle) ||
		strings.Contains(strings.ToLower(skill.Path), needle) ||
		strings.Contains(strings.ToLower(skill.Summary), needle)
}
