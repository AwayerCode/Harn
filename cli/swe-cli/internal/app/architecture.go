package app

import (
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"

	"github.com/AwayerCode/Harn/cli/swe-cli/internal/architecture"
)

// Architecture keeps rendering separate from checks: HTML is written to stdout only after validation.
func Architecture(args []string, out io.Writer) int {
	if len(args) == 0 {
		return unavailable(out, errors.New("architecture requires validate, check or render"))
	}
	mode := args[0]
	if mode != "validate" && mode != "check" && mode != "render" {
		return unavailable(out, errors.New("unknown architecture mode"))
	}
	f := flag.NewFlagSet("architecture", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	root := f.String("root", ".", "project root")
	file := f.String("file", "", "architecture YAML")
	profile := f.String("profile", "", "architecture partial design (validate only)")
	if e := f.Parse(args[1:]); e != nil {
		return unavailable(out, e)
	}
	if *file == "" || f.NArg() != 0 {
		return unavailable(out, errors.New("architecture requires --file and no positional arguments"))
	}
	if *profile != "" && (*profile != "architecture" || mode != "validate") {
		return unavailable(out, errors.New("--profile architecture is only valid with validate"))
	}
	p := *file
	if !filepath.IsAbs(p) {
		p = filepath.Join(*root, p)
	}
	input, e := os.Open(p)
	if e != nil {
		return unavailable(out, e)
	}
	defer input.Close()
	data, e := io.ReadAll(io.LimitReader(input, 1024*1024+1))
	if e != nil {
		return unavailable(out, e)
	}
	r := architecture.NewReport(mode, Version, data)
	d := architecture.ParseProfile(data, &r, *profile == "architecture")
	if *profile != "" {
		r.Unverified = append(r.Unverified, "Partial architecture profile: not a complete implementation acceptance.")
	}
	if len(r.Diagnostics) == 0 && mode == "check" {
		if e = d.CheckGo(*root, &r); e != nil {
			r.Status = "unavailable"
			r.Diagnostics = append(r.Diagnostics, architecture.Diagnostic{Rule: "implementation.unavailable", Message: e.Error()})
			return encode(out, r, 2)
		}
	}
	if len(r.Diagnostics) > 0 {
		return encode(out, r, 1)
	}
	if mode == "render" {
		if e = architecture.Render(out, d, r); e != nil {
			return 2
		}
		return 0
	}
	return encode(out, r, 0)
}
