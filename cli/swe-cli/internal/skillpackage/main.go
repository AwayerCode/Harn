// skillpackage assembles platform Skill archives. The CLI is an input to the
// package build, never an asset fetched separately during Skill installation.
package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var platforms = []string{"linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64", "windows/amd64", "windows/arm64"}

func main() {
	version := flag.String("version", "", "version compiled into the bundled CLI")
	dist := flag.String("dist", "dist/bin", "directory containing the six built CLIs")
	root := flag.String("root", "../..", "repository root")
	out := flag.String("out", "dist", "Skill archive output directory")
	flag.Parse()
	if err := buildPackages(*root, *dist, *out, *version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildPackages(root, dist, out, version string) error {
	if strings.TrimSpace(version) == "" {
		return errors.New("explicit bundled CLI version required")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	var sums []string
	for _, platform := range platforms {
		suffix := strings.ReplaceAll(platform, "/", "_")
		executable := "swe-cli"
		if strings.HasPrefix(platform, "windows/") {
			executable += ".exe"
		}
		binary := filepath.Join(dist, "swe-cli_"+suffix)
		if strings.HasSuffix(executable, ".exe") {
			binary += ".exe"
		}
		destination := filepath.Join(out, "swe_"+suffix+".tar.gz")
		if err := buildPackage(filepath.Join(root, "skills", "swe"), binary, destination, version, platform, executable); err != nil {
			return fmt.Errorf("package %s: %w", platform, err)
		}
		data, err := os.ReadFile(destination)
		if err != nil {
			return err
		}
		sums = append(sums, fmt.Sprintf("%x  %s", sha256.Sum256(data), filepath.Base(destination)))
	}
	sort.Strings(sums)
	return os.WriteFile(filepath.Join(out, "SHA256SUMS"), []byte(strings.Join(sums, "\n")+"\n"), 0644)
}

func buildPackage(skillRoot, binary, destination, version, platform, executable string) error {
	data, err := os.ReadFile(binary)
	if err != nil {
		return err
	}
	binding, err := json.MarshalIndent(map[string]any{
		"schema_version": 1, "name": "swe-cli", "version": version,
		"assets": map[string]any{platform: map[string]string{"sha256": fmt.Sprintf("%x", sha256.Sum256(data))}},
	}, "", "  ")
	if err != nil {
		return err
	}
	if info, err := os.Stat(filepath.Join(skillRoot, "SKILL.md")); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("Skill entry missing or invalid: %s", skillRoot)
	}
	file, err := os.CreateTemp(filepath.Dir(destination), ".skill-package-*")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	defer file.Close()
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	add := func(name string, mode int64, contents []byte) error {
		if err := tw.WriteHeader(&tar.Header{Name: "skills/swe/" + name, Mode: mode, Size: int64(len(contents)), Typeflag: tar.TypeReg}); err != nil {
			return err
		}
		_, err := tw.Write(contents)
		return err
	}
	err = filepath.WalkDir(skillRoot, func(name string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(skillRoot, name)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if relative == "bin" {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if relative == "runtime.json" {
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
			return fmt.Errorf("non-regular Skill source %q", name)
		}
		contents, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		mode := int64(0644)
		if info.Mode().Perm()&0111 != 0 {
			mode = 0755
		}
		return add(filepath.ToSlash(relative), mode, contents)
	})
	if err != nil {
		return err
	}
	if err := add("runtime.json", 0644, append(binding, '\n')); err != nil {
		return err
	}
	if err := add("bin/"+executable, 0755, data); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	if err := file.Chmod(0644); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(temporary, destination)
}
