package installer

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/AwayerCode/Harn/CLI/harn/internal/source"
	"github.com/AwayerCode/Harn/CLI/harn/internal/state"
)

type Manager struct {
	ProjectRoot string
	Client      *http.Client
	DryRun      bool
}

type Result struct {
	Path    string
	Version string
	Changed bool
	Files   int
}

type Entry struct {
	Path    string
	Source  string
	Version string
	Kind    string
	Files   int
	Status  string
}

type Removal struct {
	Path  string
	Files int
}

type targetPlan struct {
	target   string
	download state.Download
	writes   map[string]write
	deletes  []string
}

type write struct {
	source string
	mode   string
	sha256 string
}

type stagedTarget struct {
	download state.Download
	writes   map[string]write
}

func (m *Manager) Install(ctx context.Context, rawSource, version string, requested []string) ([]Result, error) {
	if m.Client == nil {
		return nil, errors.New("HTTP client is required")
	}
	targets, err := normalizeTargets(requested)
	if err != nil {
		return nil, err
	}
	if rawSource == "" {
		return nil, errors.New("source is required")
	}
	if version == "" {
		return nil, errors.New("version is required")
	}

	if !m.DryRun {
		lock, err := acquireProjectLock(m.ProjectRoot)
		if err != nil {
			return nil, err
		}
		defer lock.Close()
	}

	harnessRoot := filepath.Join(m.ProjectRoot, ".harness")
	current, err := state.Load(harnessRoot)
	if err != nil {
		return nil, err
	}
	if err := rejectOverlaps(targets, current); err != nil {
		return nil, err
	}
	for _, target := range targets {
		if _, exists := current.Downloads[target]; exists {
			return nil, fmt.Errorf("%q is already recorded; use update", target)
		}
		if _, err := os.Lstat(filepath.Join(harnessRoot, filepath.FromSlash(target))); err == nil {
			return nil, fmt.Errorf("%q already exists and is not recorded; refusing to overwrite it", target)
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("inspect %q: %w", target, err)
		}
		if err := checkParents(m.ProjectRoot, filepath.Join(harnessRoot, filepath.FromSlash(target))); err != nil {
			return nil, err
		}
	}

	archiveURL, err := source.ResolveArchiveURL(rawSource, version)
	if err != nil {
		return nil, err
	}
	stage, err := os.MkdirTemp("", "harn-install-*")
	if err != nil {
		return nil, fmt.Errorf("create staging directory: %w", err)
	}
	defer os.RemoveAll(stage)

	contentRoot := filepath.Join(stage, "content")
	if err := source.Fetch(ctx, m.Client, archiveURL, targets, contentRoot); err != nil {
		return nil, err
	}
	staged := make(map[string]stagedTarget, len(targets))
	plans := make([]targetPlan, 0, len(targets))
	for _, target := range targets {
		download, writes, err := snapshotTarget(contentRoot, target)
		if err != nil {
			return nil, fmt.Errorf("read downloaded resource %q: %w", target, err)
		}
		download.Source = rawSource
		download.Version = version
		staged[target] = stagedTarget{download: download, writes: writes}

		plan := targetPlan{target: target, download: download, writes: writes}
		for destination := range writes {
			if _, err := os.Lstat(filepath.Join(harnessRoot, filepath.FromSlash(destination))); err == nil {
				return nil, fmt.Errorf("local path %q appeared during installation; no files were written", destination)
			} else if !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("inspect %q: %w", destination, err)
			}
			if err := checkParents(m.ProjectRoot, filepath.Join(harnessRoot, filepath.FromSlash(destination))); err != nil {
				return nil, err
			}
		}
		plans = append(plans, plan)
	}

	results := make([]Result, 0, len(targets))
	for _, target := range targets {
		download := staged[target].download
		results = append(results, Result{
			Path:    target,
			Version: download.Version,
			Changed: true,
			Files:   len(download.Files),
		})
	}
	if m.DryRun {
		return results, nil
	}

	if err := applyPlans(m.ProjectRoot, harnessRoot, stage, plans); err != nil {
		return nil, err
	}
	for _, target := range targets {
		current.Downloads[target] = staged[target].download
	}
	if err := current.Save(harnessRoot); err != nil {
		rollbackErr := rollbackPlans(m.ProjectRoot, harnessRoot, stage, plans)
		if rollbackErr != nil {
			return nil, fmt.Errorf("%v; rollback also failed: %v", err, rollbackErr)
		}
		return nil, err
	}

	return results, nil
}

func (m *Manager) Update(ctx context.Context, requested []string, versionOverride string) ([]Result, error) {
	if m.Client == nil {
		return nil, errors.New("HTTP client is required")
	}
	if len(requested) == 0 {
		return nil, errors.New("at least one path is required")
	}
	targets, err := normalizeTargets(requested)
	if err != nil {
		return nil, err
	}

	if !m.DryRun {
		lock, err := acquireProjectLock(m.ProjectRoot)
		if err != nil {
			return nil, err
		}
		defer lock.Close()
	}

	harnessRoot := filepath.Join(m.ProjectRoot, ".harness")
	current, err := state.Load(harnessRoot)
	if err != nil {
		return nil, err
	}
	for _, target := range targets {
		if _, exists := current.Downloads[target]; !exists {
			return nil, fmt.Errorf("%q is not recorded under .harness", target)
		}
		download := current.Downloads[target]
		if err := verifyOwnedFiles(harnessRoot, target, download); err != nil {
			return nil, fmt.Errorf("local changes prevent update for %q: %w; no files were written", target, err)
		}
		if err := checkParents(m.ProjectRoot, filepath.Join(harnessRoot, filepath.FromSlash(target))); err != nil {
			return nil, err
		}
	}

	type group struct {
		source  string
		version string
		targets []string
	}
	groupsByKey := make(map[string]*group)
	for _, target := range targets {
		download := current.Downloads[target]
		version := download.Version
		if versionOverride != "" {
			version = versionOverride
		}
		key := download.Source + "\x00" + version
		if groupsByKey[key] == nil {
			groupsByKey[key] = &group{source: download.Source, version: version}
		}
		groupsByKey[key].targets = append(groupsByKey[key].targets, target)
	}
	groupKeys := make([]string, 0, len(groupsByKey))
	for key := range groupsByKey {
		groupKeys = append(groupKeys, key)
	}
	sort.Strings(groupKeys)

	stage, err := os.MkdirTemp("", "harn-update-*")
	if err != nil {
		return nil, fmt.Errorf("create staging directory: %w", err)
	}
	defer os.RemoveAll(stage)

	staged := make(map[string]stagedTarget, len(targets))
	for index, key := range groupKeys {
		item := groupsByKey[key]
		archiveURL, err := source.ResolveArchiveURL(item.source, item.version)
		if err != nil {
			return nil, err
		}
		contentRoot := filepath.Join(stage, "content-"+strconv.Itoa(index))
		if err := source.Fetch(ctx, m.Client, archiveURL, item.targets, contentRoot); err != nil {
			return nil, err
		}
		for _, target := range item.targets {
			download, writes, err := snapshotTarget(contentRoot, target)
			if err != nil {
				return nil, fmt.Errorf("read downloaded resource %q: %w", target, err)
			}
			download.Source = item.source
			download.Version = item.version
			staged[target] = stagedTarget{download: download, writes: writes}
		}
	}

	plans := make([]targetPlan, 0, len(targets))
	for _, target := range targets {
		oldDownload := current.Downloads[target]
		next := staged[target]
		if oldDownload.Kind != next.download.Kind {
			return nil, fmt.Errorf("resource %q changed from %s to %s; refusing to replace its kind; no files were written", target, oldDownload.Kind, next.download.Kind)
		}
		oldOwned := make(map[string]bool, len(oldDownload.Files))
		for file := range oldDownload.Files {
			oldOwned[file] = true
		}
		for destination := range next.writes {
			if _, err := os.Lstat(filepath.Join(harnessRoot, filepath.FromSlash(destination))); err == nil {
				if !oldOwned[destination] {
					return nil, fmt.Errorf("local path %q would be overwritten; no files were written", destination)
				}
			} else if !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("inspect %q: %w", destination, err)
			}
			if err := checkParents(m.ProjectRoot, filepath.Join(harnessRoot, filepath.FromSlash(destination))); err != nil {
				return nil, err
			}
		}

		deletes := make([]string, 0)
		for file := range oldDownload.Files {
			if _, remains := next.download.Files[file]; !remains {
				deletes = append(deletes, file)
			}
		}
		sort.Strings(deletes)

		writeSet := make(map[string]write)
		for destination, nextWrite := range next.writes {
			oldFile, existed := oldDownload.Files[destination]
			if !existed || oldFile != (state.FileState{SHA256: nextWrite.sha256, Mode: nextWrite.mode}) {
				writeSet[destination] = nextWrite
			}
		}
		plans = append(plans, targetPlan{
			target:   target,
			download: next.download,
			writes:   writeSet,
			deletes:  deletes,
		})
	}

	results := make([]Result, 0, len(targets))
	for _, target := range targets {
		oldDownload := current.Downloads[target]
		next := staged[target].download
		results = append(results, Result{
			Path:    target,
			Version: next.Version,
			Changed: !sameDownload(oldDownload, next),
			Files:   len(next.Files),
		})
	}
	if m.DryRun {
		return results, nil
	}

	if err := applyPlans(m.ProjectRoot, harnessRoot, stage, plans); err != nil {
		return nil, err
	}
	for _, target := range targets {
		current.Downloads[target] = staged[target].download
	}
	if err := current.Save(harnessRoot); err != nil {
		rollbackErr := rollbackPlans(m.ProjectRoot, harnessRoot, stage, plans)
		if rollbackErr != nil {
			return nil, fmt.Errorf("%v; rollback also failed: %v", err, rollbackErr)
		}
		return nil, err
	}
	return results, nil
}

func normalizeTargets(requested []string) ([]string, error) {
	if len(requested) == 0 {
		return nil, errors.New("at least one --path is required")
	}
	seen := make(map[string]bool, len(requested))
	targets := make([]string, 0, len(requested))
	for _, raw := range requested {
		clean := path.Clean(strings.ReplaceAll(raw, "\\", "/"))
		if clean == "." || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") ||
			clean == ".harness" || strings.HasPrefix(clean, ".harness/") ||
			clean == "bin" || strings.HasPrefix(clean, "bin/") || clean == state.FileName {
			return nil, fmt.Errorf("invalid resource path %q", raw)
		}
		for _, part := range strings.Split(clean, "/") {
			if part == "" || part == "." || part == ".." || part == ".git" {
				return nil, fmt.Errorf("invalid resource path %q", raw)
			}
		}
		if !seen[clean] {
			seen[clean] = true
			targets = append(targets, clean)
		}
	}
	sort.Strings(targets)
	for left := 0; left < len(targets); left++ {
		for right := left + 1; right < len(targets); right++ {
			if overlaps(targets[left], targets[right]) {
				return nil, fmt.Errorf("resource paths %q and %q overlap", targets[left], targets[right])
			}
		}
	}
	return targets, nil
}

func rejectOverlaps(targets []string, current *state.State) error {
	for _, target := range targets {
		for installed := range current.Downloads {
			if overlaps(target, installed) {
				return fmt.Errorf("resource path %q overlaps recorded path %q", target, installed)
			}
		}
	}
	return nil
}

func overlaps(left, right string) bool {
	return left == right || strings.HasPrefix(left, right+"/") || strings.HasPrefix(right, left+"/")
}

func snapshotTarget(contentRoot, target string) (state.Download, map[string]write, error) {
	root := filepath.Join(contentRoot, filepath.FromSlash(target))
	info, err := os.Lstat(root)
	if err != nil {
		return state.Download{}, nil, err
	}
	kind := ""
	switch {
	case info.IsDir():
		kind = "directory"
	case info.Mode().IsRegular():
		kind = "file"
	default:
		return state.Download{}, nil, fmt.Errorf("resource root is not a regular file or directory")
	}

	files := make(map[string]state.FileState)
	writes := make(map[string]write)
	if kind == "file" {
		file, err := digestFile(root)
		if err != nil {
			return state.Download{}, nil, err
		}
		files[target] = file
		writes[target] = write{source: root, mode: file.Mode, sha256: file.SHA256}
		return state.Download{Kind: kind, Files: files}, writes, nil
	}

	err = filepath.WalkDir(root, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == root {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%q is not a regular file", current)
		}
		relative, err := filepath.Rel(contentRoot, current)
		if err != nil {
			return err
		}
		destination := filepath.ToSlash(relative)
		file, err := digestFile(current)
		if err != nil {
			return err
		}
		files[destination] = file
		writes[destination] = write{source: current, mode: file.Mode, sha256: file.SHA256}
		return nil
	})
	if err != nil {
		return state.Download{}, nil, err
	}
	if len(files) == 0 {
		return state.Download{}, nil, errors.New("resource directory is empty; Git cannot preserve an empty downloaded directory")
	}
	return state.Download{Kind: kind, Files: files}, writes, nil
}

func digestFile(filename string) (state.FileState, error) {
	file, err := os.Open(filename)
	if err != nil {
		return state.FileState{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return state.FileState{}, err
	}
	if !info.Mode().IsRegular() {
		return state.FileState{}, fmt.Errorf("%q is not a regular file", filename)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return state.FileState{}, err
	}
	return state.FileState{SHA256: fmt.Sprintf("%x", hash.Sum(nil)), Mode: normalizeMode(info.Mode())}, nil
}

func normalizeMode(mode os.FileMode) string {
	if mode.Perm()&0o111 != 0 {
		return "0755"
	}
	return "0644"
}

func verifyOwnedFiles(harnessRoot, target string, download state.Download) error {
	root := filepath.Join(harnessRoot, filepath.FromSlash(target))
	rootInfo, err := os.Lstat(root)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s is missing", target)
	}
	if err != nil {
		return err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symbolic link", target)
	}
	if download.Kind == "directory" && !rootInfo.IsDir() {
		return fmt.Errorf("%s is no longer a directory", target)
	}
	if download.Kind == "file" && !rootInfo.Mode().IsRegular() {
		return fmt.Errorf("%s is no longer a regular file", target)
	}

	files := make([]string, 0, len(download.Files))
	for file := range download.Files {
		files = append(files, file)
	}
	sort.Strings(files)
	if len(files) == 0 && download.Kind == "directory" {
		return nil
	}

	conflicts := make([]string, 0)
	for _, file := range files {
		actual, err := digestFile(filepath.Join(harnessRoot, filepath.FromSlash(file)))
		if errors.Is(err, os.ErrNotExist) {
			conflicts = append(conflicts, file+" (missing)")
			continue
		}
		if err != nil {
			conflicts = append(conflicts, file+" (not readable as a regular file)")
			continue
		}
		expected := download.Files[file]
		if actual.SHA256 != expected.SHA256 {
			conflicts = append(conflicts, file+" (content changed)")
			continue
		}
		if runtime.GOOS != "windows" && actual.Mode != expected.Mode {
			conflicts = append(conflicts, file+" (permissions changed)")
		}
	}
	if len(conflicts) > 0 {
		const limit = 5
		if len(conflicts) > limit {
			return fmt.Errorf("%s and %d more", strings.Join(conflicts[:limit], ", "), len(conflicts)-limit)
		}
		return errors.New(strings.Join(conflicts, ", "))
	}
	return nil
}

func sameDownload(left, right state.Download) bool {
	if left.Source != right.Source || left.Version != right.Version || left.Kind != right.Kind || len(left.Files) != len(right.Files) {
		return false
	}
	for file, leftFile := range left.Files {
		if right.Files[file] != leftFile {
			return false
		}
	}
	return true
}
