// Generate immutable companion bindings after building a specific SWE CLI release.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

func main() {
	version := flag.String("version", "", "SWE CLI release version without prefix")
	dist := flag.String("dist", "dist", "built assets")
	root := flag.String("root", "../..", "repository root")
	flag.Parse()
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[A-Za-z0-9.-]+)?$`).MatchString(*version) {
		panic("explicit release version required")
	}
	assets := map[string]any{}
	for _, osname := range []string{"linux", "darwin", "windows"} {
		for _, arch := range []string{"amd64", "arm64"} {
			name := "swe-cli_" + osname + "_" + arch
			if osname == "windows" {
				name += ".exe"
			}
			b, e := os.ReadFile(filepath.Join(*dist, name))
			must(e)
			assets[osname+"/"+arch] = map[string]string{"url": "https://github.com/AwayerCode/Harn/releases/download/swe-cli-v" + *version + "/" + name, "sha256": fmt.Sprintf("%x", sha256.Sum256(b))}
		}
	}
	b, e := json.MarshalIndent(map[string]any{"schema_version": 1, "name": "swe-cli", "version": *version, "assets": assets}, "", "  ")
	must(e)
	for _, skill := range []string{"swe"} {
		must(os.WriteFile(filepath.Join(*root, "skills", skill, "runtime.json"), append(b, '\n'), 0644))
	}
}
func must(e error) {
	if e != nil {
		panic(e)
	}
}
