package installer

import (
	"bytes"
	"context"
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
	if err := source.FetchSkillEntries(ctx, m.Client, archiveURL, contentRoot); err != nil {
		return nil, fmt.Errorf("read skill entries: %w", err)
	}
	skills, err := scanSkills(contentRoot)
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

func scanSkills(root string) ([]Skill, error) {
	files, err := filepath.Glob(filepath.Join(root, "skills", "*", "SKILL.md"))
	if err != nil {
		return nil, err
	}
	skills := make([]Skill, 0, len(files))
	seen := make(map[string]bool)
	for _, file := range files {
		directory := filepath.Base(filepath.Dir(file))
		if strings.HasPrefix(directory, "_") || strings.HasPrefix(directory, ".") {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		skill, err := parseSkill(data, "skills/"+directory)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		if seen[skill.Name] {
			return nil, fmt.Errorf("duplicate skill name %q", skill.Name)
		}
		seen[skill.Name] = true
		skills = append(skills, skill)
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
