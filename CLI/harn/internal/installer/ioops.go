package installer

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func applyPlans(projectRoot, harnessRoot, stage string, plans []targetPlan) error {
	affected := affectedPaths(plans)
	if err := backupPaths(stage, harnessRoot, affected); err != nil {
		return err
	}
	rollback := func(cause error) error {
		if rollbackErr := rollbackPlans(projectRoot, harnessRoot, stage, plans); rollbackErr != nil {
			return fmt.Errorf("%v; rollback also failed: %v", cause, rollbackErr)
		}
		return cause
	}

	if _, err := mkdirAllTracked(projectRoot, harnessRoot); err != nil {
		return err
	}
	for _, plan := range plans {
		for _, destination := range plan.deletes {
			fullPath := filepath.Join(harnessRoot, filepath.FromSlash(destination))
			if err := os.Remove(fullPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return rollback(fmt.Errorf("remove replaced file %q: %w", destination, err))
			}
		}
		if plan.download.Kind == "directory" {
			if _, err := mkdirAllTracked(projectRoot, filepath.Join(harnessRoot, filepath.FromSlash(plan.target))); err != nil {
				return rollback(err)
			}
		}

		destinations := make([]string, 0, len(plan.writes))
		for destination := range plan.writes {
			destinations = append(destinations, destination)
		}
		sort.Strings(destinations)
		for _, destination := range destinations {
			item := plan.writes[destination]
			target := filepath.Join(harnessRoot, filepath.FromSlash(destination))
			if _, err := mkdirAllTracked(projectRoot, filepath.Dir(target)); err != nil {
				return rollback(err)
			}
			if err := atomicWrite(item.source, target, item.mode); err != nil {
				return rollback(fmt.Errorf("write %q: %w", destination, err))
			}
		}
	}
	return nil
}

func rollbackPlans(projectRoot, harnessRoot, stage string, plans []targetPlan) error {
	affected := affectedPaths(plans)
	paths := make([]string, 0, len(affected))
	for destination := range affected {
		paths = append(paths, destination)
	}
	sort.Strings(paths)

	var failures []string
	for _, destination := range paths {
		backup := filepath.Join(stage, "rollback", filepath.FromSlash(destination))
		target := filepath.Join(harnessRoot, filepath.FromSlash(destination))
		info, err := os.Lstat(backup)
		if err == nil {
			if !info.Mode().IsRegular() {
				failures = append(failures, destination+": backup is not a regular file")
				continue
			}
			if _, err := mkdirAllTracked(projectRoot, filepath.Dir(target)); err != nil {
				failures = append(failures, destination+": "+err.Error())
				continue
			}
			if err := atomicWrite(backup, target, normalizeMode(info.Mode())); err != nil {
				failures = append(failures, destination+": "+err.Error())
			}
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			failures = append(failures, destination+": "+err.Error())
			continue
		}
		if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
			failures = append(failures, destination+": "+err.Error())
		}
	}
	if len(failures) > 0 {
		return errors.New(strings.Join(failures, "; "))
	}
	return nil
}

func affectedPaths(plans []targetPlan) map[string]bool {
	affected := make(map[string]bool)
	for _, plan := range plans {
		for destination := range plan.writes {
			affected[destination] = true
		}
		for _, destination := range plan.deletes {
			affected[destination] = true
		}
	}
	return affected
}

func backupPaths(stage, harnessRoot string, affected map[string]bool) error {
	paths := make([]string, 0, len(affected))
	for destination := range affected {
		paths = append(paths, destination)
	}
	sort.Strings(paths)
	for _, destination := range paths {
		sourcePath := filepath.Join(harnessRoot, filepath.FromSlash(destination))
		info, err := os.Lstat(sourcePath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect existing file %q: %w", destination, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("existing path %q is not a regular file", destination)
		}
		backup := filepath.Join(stage, "rollback", filepath.FromSlash(destination))
		if err := os.MkdirAll(filepath.Dir(backup), 0o700); err != nil {
			return fmt.Errorf("create rollback directory: %w", err)
		}
		if err := copyFile(sourcePath, backup, info.Mode()); err != nil {
			return fmt.Errorf("back up %q: %w", destination, err)
		}
	}
	return nil
}

func copyFile(source, destination string, mode os.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func atomicWrite(source, destination, mode string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("source is not a regular file")
	}

	perm, err := strconv.ParseUint(mode, 8, 32)
	if err != nil {
		return fmt.Errorf("invalid file mode %q", mode)
	}
	temp, err := os.CreateTemp(filepath.Dir(destination), ".harness-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if _, err := io.Copy(temp, input); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Chmod(os.FileMode(perm).Perm()); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, destination)
}

func checkParents(projectRoot, target string) error {
	relative, err := filepath.Rel(projectRoot, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path %q is outside the project", target)
	}
	parts := strings.Split(relative, string(filepath.Separator))
	cursor := projectRoot
	for index := 0; index < len(parts)-1; index++ {
		cursor = filepath.Join(cursor, parts[index])
		info, err := os.Lstat(cursor)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect parent %q: %w", cursor, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("parent %q is a symbolic link; refusing to write outside the project", cursor)
		}
		if !info.IsDir() {
			return fmt.Errorf("parent %q is not a directory", cursor)
		}
	}
	return nil
}

func mkdirAllTracked(projectRoot, directory string) ([]string, error) {
	relative, err := filepath.Rel(projectRoot, directory)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("directory %q is outside the project", directory)
	}
	cursor := projectRoot
	created := make([]string, 0)
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		cursor = filepath.Join(cursor, part)
		info, err := os.Lstat(cursor)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(cursor, 0o755); err != nil {
				return created, fmt.Errorf("create directory %q: %w", cursor, err)
			}
			created = append(created, cursor)
			continue
		}
		if err != nil {
			return created, fmt.Errorf("inspect directory %q: %w", cursor, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return created, fmt.Errorf("directory %q is a symbolic link", cursor)
		}
		if !info.IsDir() {
			return created, fmt.Errorf("path %q is not a directory", cursor)
		}
	}
	return created, nil
}
