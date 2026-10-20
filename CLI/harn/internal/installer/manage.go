package installer

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AwayerCode/Harn/CLI/harn/internal/state"
)

const (
	StatusOK       = "ok"
	StatusModified = "modified"
	StatusMissing  = "missing"
)

func (m Manager) List() ([]Entry, error) {
	harnessRoot := filepath.Join(m.ProjectRoot, ".harness")
	current, err := state.Load(harnessRoot)
	if err != nil {
		return nil, err
	}
	targets := state.Targets(current)
	entries := make([]Entry, 0, len(targets))
	for _, target := range targets {
		download := current.Downloads[target]
		entries = append(entries, Entry{
			Path:    target,
			Source:  download.Source,
			Version: download.Version,
			Kind:    download.Kind,
			Files:   len(download.Files),
		})
	}
	return entries, nil
}

func (m Manager) Status() ([]Entry, error) {
	harnessRoot := filepath.Join(m.ProjectRoot, ".harness")
	current, err := state.Load(harnessRoot)
	if err != nil {
		return nil, err
	}
	targets := state.Targets(current)
	entries := make([]Entry, 0, len(targets))
	for _, target := range targets {
		download := current.Downloads[target]
		entries = append(entries, Entry{
			Path:    target,
			Source:  download.Source,
			Version: download.Version,
			Kind:    download.Kind,
			Files:   len(download.Files),
			Status:  inspectStatus(harnessRoot, target, download),
		})
	}
	return entries, nil
}

func inspectStatus(harnessRoot, target string, download state.Download) string {
	root := filepath.Join(harnessRoot, filepath.FromSlash(target))
	if _, err := os.Lstat(root); errors.Is(err, os.ErrNotExist) {
		return StatusMissing
	} else if err != nil {
		return StatusModified
	}
	if err := verifyOwnedFiles(harnessRoot, target, download); err != nil {
		if strings.Contains(err.Error(), "is missing") || strings.Contains(err.Error(), "(missing)") {
			return StatusMissing
		}
		return StatusModified
	}
	return StatusOK
}

func (m Manager) Remove(requested []string) ([]Removal, error) {
	targets, err := normalizeTargets(requested)
	if err != nil {
		return nil, err
	}

	lock, err := acquireProjectLock(m.ProjectRoot)
	if err != nil {
		return nil, err
	}
	defer lock.Close()

	harnessRoot := filepath.Join(m.ProjectRoot, ".harness")
	current, err := state.Load(harnessRoot)
	if err != nil {
		return nil, err
	}
	plans := make([]targetPlan, 0, len(targets))
	for _, target := range targets {
		download, exists := current.Downloads[target]
		if !exists {
			return nil, fmt.Errorf("%q is not recorded under .harness", target)
		}
		if err := verifyOwnedFiles(harnessRoot, target, download); err != nil {
			return nil, fmt.Errorf("local changes prevent removal for %q: %w; nothing was removed", target, err)
		}
		if err := checkParents(m.ProjectRoot, filepath.Join(harnessRoot, filepath.FromSlash(target))); err != nil {
			return nil, err
		}
		deletes := make([]string, 0, len(download.Files))
		for file := range download.Files {
			deletes = append(deletes, file)
		}
		sort.Strings(deletes)
		plans = append(plans, targetPlan{target: target, download: download, deletes: deletes})
	}

	stage, err := os.MkdirTemp("", "harn-remove-*")
	if err != nil {
		return nil, fmt.Errorf("create staging directory: %w", err)
	}
	defer os.RemoveAll(stage)

	if err := applyPlans(m.ProjectRoot, harnessRoot, stage, plans); err != nil {
		return nil, err
	}
	for _, plan := range plans {
		for _, file := range plan.deletes {
			pruneEmptyDirs(harnessRoot, file)
		}
		delete(current.Downloads, plan.target)
	}
	if err := current.Save(harnessRoot); err != nil {
		rollbackErr := rollbackPlans(m.ProjectRoot, harnessRoot, stage, plans)
		if rollbackErr != nil {
			return nil, fmt.Errorf("%v; rollback also failed: %v", err, rollbackErr)
		}
		return nil, err
	}

	results := make([]Removal, 0, len(plans))
	for _, plan := range plans {
		results = append(results, Removal{Path: plan.target, Files: len(plan.deletes)})
	}
	return results, nil
}

func pruneEmptyDirs(harnessRoot, target string) {
	start := filepath.Join(harnessRoot, filepath.FromSlash(path.Dir(target)))
	for {
		relative, err := filepath.Rel(harnessRoot, start)
		if err != nil || relative == "." || relative == ".." ||
			strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return
		}
		entries, err := os.ReadDir(start)
		if err != nil || len(entries) > 0 {
			return
		}
		if err := os.Remove(start); err != nil {
			return
		}
		start = filepath.Dir(start)
	}
}
