package app

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"

	"github.com/AwayerCode/Harn/cli/swe-cli/internal/systemdesign"
)

type designBatch struct {
	Format  string                        `json:"format"`
	Mode    string                        `json:"mode"`
	Status  string                        `json:"status"`
	Issues  []systemdesign.DiscoveryIssue `json:"discovery_issues"`
	Scopes  []systemdesign.Scope          `json:"scopes"`
	Results []systemdesign.Report         `json:"results"`
}

func designUnavailable(mode string, err error) systemdesign.Report {
	r := systemdesign.NewReport(mode, Version, nil)
	r.Digest = ""
	r.Status = "unavailable"
	r.Diagnostics = append(r.Diagnostics, systemdesign.Diagnostic{Rule: "system-design.unavailable", Message: err.Error()})
	return r
}

func SystemDesign(args []string, out io.Writer) int {
	mode := ""
	if len(args) > 0 {
		mode = args[0]
	}
	fail := func(err error) int { return encode(out, designUnavailable(mode, err), 2) }
	if mode != "list" && mode != "validate" && mode != "check" && mode != "render" {
		return fail(errors.New("system-design requires list, validate, check or render"))
	}
	f := flag.NewFlagSet("system-design", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	root := f.String("root", ".", "discovery root or base for a relative file")
	file := f.String("file", "", "system-design.yaml path; its directory is the scope")
	all := f.Bool("all", false, "check all discovered designs")
	profile := f.String("profile", "", "layers: partial design, validate only")
	if err := f.Parse(args[1:]); err != nil {
		return fail(err)
	}
	if f.NArg() != 0 {
		return fail(errors.New("positional arguments are not supported"))
	}
	if *profile != "" && (*profile != "layers" || mode != "validate") {
		return fail(errors.New("--profile layers is only valid with validate"))
	}
	if mode == "list" {
		if *file != "" || *all || *profile != "" {
			return fail(errors.New("list accepts only --root"))
		}
	} else if (*file == "") == (!*all) || (mode == "render" && *all) {
		return fail(errors.New("choose exactly one of --file or --all; render requires --file"))
	}
	if mode == "list" || *all {
		discovery, err := systemdesign.Discover(*root)
		if err != nil {
			return fail(err)
		}
		scopes := discovery.Scopes
		if len(scopes) == 0 && len(discovery.Issues) == 0 {
			return fail(errors.New("no system-design.yaml found"))
		}
		result := designBatch{Format: "swe-system-design-batch-v1", Mode: mode, Status: "passed", Scopes: scopes, Issues: discovery.Issues, Results: []systemdesign.Report{}}
		code := 0
		if len(discovery.Issues) > 0 {
			code = 2
		}
		if mode == "list" {
			for _, s := range scopes {
				if s.Unavailable != "" {
					code = 2
				}
			}
		}
		if mode != "list" {
			for _, s := range scopes {
				r, _, c := runDesign(s, mode, *profile)
				result.Results = append(result.Results, r)
				if c > code {
					code = c
				}
			}
		}
		if code == 1 {
			result.Status = "violations"
		}
		if code == 2 {
			result.Status = "unavailable"
		}
		return encode(out, result, code)
	}
	p := *file
	if !filepath.IsAbs(p) {
		p = filepath.Join(*root, p)
	}
	scope, err := systemdesign.FileScope(p)
	if err != nil {
		return fail(err)
	}
	r, html, code := runDesign(scope, mode, *profile)
	if mode == "render" && code == 0 {
		if _, err := out.Write(html); err != nil {
			return 2
		}
		return 0
	}
	return encode(out, r, code)
}

func runDesign(s systemdesign.Scope, mode, profile string) (systemdesign.Report, []byte, int) {
	fail := func(err error) (systemdesign.Report, []byte, int) {
		r := designUnavailable(mode, err)
		r.Scope = &s
		return r, nil, 2
	}
	if s.Unavailable != "" {
		return fail(errors.New(s.Unavailable))
	}
	info, err := os.Lstat(s.File)
	if err != nil {
		return fail(err)
	}
	if !info.Mode().IsRegular() {
		return fail(errors.New("design must be a regular file"))
	}
	input, err := os.Open(s.File)
	if err != nil {
		return fail(err)
	}
	defer input.Close()
	data, err := io.ReadAll(io.LimitReader(input, 1024*1024+1))
	if err != nil {
		return fail(err)
	}
	r := systemdesign.NewReport(mode, Version, data)
	r.Scope = &s
	d := systemdesign.ParseProfile(data, &r, profile == "layers")
	if profile != "" {
		r.Unverified = append(r.Unverified, "Partial layers profile: not a complete implementation acceptance.")
	}
	if len(r.Diagnostics) == 0 {
		d.ValidateScope(s, &r)
	}
	if len(r.Diagnostics) == 0 && mode == "check" {
		if err = d.CheckGo(s.Root, &r); err != nil {
			r.Status = "unavailable"
			r.Diagnostics = append(r.Diagnostics, systemdesign.Diagnostic{Rule: "implementation.unavailable", Message: err.Error()})
			return r, nil, 2
		}
	}
	if len(r.Diagnostics) > 0 {
		return r, nil, 1
	}
	if mode == "render" {
		var html bytes.Buffer
		if err = systemdesign.Render(&html, d, r); err != nil {
			r.Status = "unavailable"
			r.Diagnostics = append(r.Diagnostics, systemdesign.Diagnostic{Rule: "render.unavailable", Message: err.Error()})
			return r, nil, 2
		}
		return r, html.Bytes(), 0
	}
	return r, nil, 0
}
