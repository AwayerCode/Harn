package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/AwayerCode/Harn/cli/swe-cli/internal/checks"
	"github.com/AwayerCode/Harn/cli/swe-cli/internal/workspace"
)

type values []string

func (v *values) String() string     { return strings.Join(*v, ",") }
func (v *values) Set(s string) error { *v = append(*v, s); return nil }
func Scope(args []string, out io.Writer) int {
	if len(args) == 0 {
		return unavailable(out, errors.New("scope requires snapshot, check or readonly"))
	}
	mode := args[0]
	f := flag.NewFlagSet("scope", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	root := f.String("root", "", "Git root")
	baseline := f.String("baseline", "", "external baseline")
	expected := f.String("expected", "", "versioned fingerprint")
	var files, dirs values
	f.Var(&files, "allow-file", "writable file")
	f.Var(&dirs, "allow-dir", "writable directory")
	if e := f.Parse(args[1:]); e != nil {
		return unavailable(out, e)
	}
	if f.NArg() != 0 || *root == "" {
		return unavailable(out, errors.New("scope requires --root and no positional arguments"))
	}
	r, e := workspace.Root(*root)
	if e != nil {
		return unavailable(out, e)
	}
	switch mode {
	case "readonly":
		if *baseline != "" || len(files)+len(dirs) > 0 {
			return unavailable(out, errors.New("readonly cannot accept baseline or writable scope"))
		}
		if *expected != "" {
			if e := workspace.ValidateFingerprint(*expected); e != nil {
				return unavailable(out, e)
			}
		}
		hash, count, e := workspace.Fingerprint(r)
		if e != nil {
			return unavailable(out, e)
		}
		changed := *expected != "" && *expected != hash
		code := 0
		if changed {
			code = 1
		}
		return encode(out, map[string]any{"format": workspace.Format, "sha256": hash, "files": count, "changed": changed, "baseline": *expected == "", "coverage": workspace.Coverage}, code)
	case "snapshot":
		if *expected != "" || *baseline == "" {
			return unavailable(out, errors.New("snapshot requires --baseline and no --expected"))
		}
		if e := workspace.Snapshot(r, *baseline, files, dirs); e != nil {
			return unavailable(out, e)
		}
		return encode(out, map[string]any{"format": workspace.Format, "baseline": "created", "coverage": workspace.Coverage}, 0)
	case "check":
		if *expected != "" || *baseline == "" || len(files)+len(dirs) > 0 {
			return unavailable(out, errors.New("check uses only the original baseline and cannot expand scope"))
		}
		changed, unexpected, e := workspace.Check(r, *baseline)
		if e != nil {
			return unavailable(out, e)
		}
		code := 0
		if len(unexpected) > 0 {
			code = 1
		}
		return encode(out, map[string]any{"format": workspace.Format, "changed": changed, "unexpected": unexpected, "coverage": workspace.Coverage}, code)
	default:
		return unavailable(out, errors.New("unknown scope mode"))
	}
}

// Convenience commands build the same request as JSON callers, with no second checker implementation.
func Direct(command string, args []string, out io.Writer) int {
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	root := f.String("root", ".", "project root")
	kind := f.String("kind", "", "artifact kind")
	file := f.String("file", "", "file")
	profile := f.String("profile", "", "profile")
	system := f.String("system-design", "", "system design")
	protocol := f.String("interface-protocol", "", "interface protocol")
	f.StringVar(protocol, "protocol", "", "interface protocol")
	design := f.String("design", "", "system design")
	var scopes values
	f.Var(&scopes, "scope", "relative scope")
	if e := f.Parse(args); e != nil {
		return unavailable(out, e)
	}
	allowed := map[string][]string{
		"documents": {"root", "kind", "file", "profile"}, "design": {"root", "system-design", "interface-protocol", "protocol"},
		"artifacts": {"root", "kind", "file", "protocol", "interface-protocol"}, "structure": {"root", "design", "scope"},
	}
	var optionError error
	f.Visit(func(option *flag.Flag) {
		if !slices.Contains(allowed[command], option.Name) {
			optionError = fmt.Errorf("option --%s is not valid for %s", option.Name, command)
		}
	})
	if optionError != nil {
		return unavailable(out, optionError)
	}
	if f.NArg() != 0 {
		return unavailable(out, errors.New("unexpected positional arguments"))
	}
	absolute, e := filepath.Abs(*root)
	if e != nil {
		return unavailable(out, e)
	}
	absolute, e = filepath.EvalSymlinks(absolute)
	if e != nil {
		return unavailable(out, e)
	}
	input := map[string]any{}
	switch command {
	case "documents":
		input = map[string]any{"kind": *kind, "file": *file, "profile": *profile}
	case "design":
		input = map[string]any{"systemDesign": *system, "protocol": *protocol}
	case "artifacts":
		input = map[string]any{"kind": *kind, "file": *file, "protocol": *protocol}
	case "structure":
		input = map[string]any{"design": *design, "scopes": []string(scopes)}
	}
	raw, _ := json.Marshal(input)
	required := true
	req := Request{ProtocolVersion: ProtocolVersion, TaskID: command, Round: 1, Mode: "review", ProjectRoot: absolute, Checks: []CheckRequest{{CheckID: command, Required: &required, Input: raw}}}
	data, _ := json.Marshal(req)
	return Run([]string{"check", "--request", "-"}, bytes.NewReader(data), out)
}
func absoluteInput(root, value string) string {
	if value == "" {
		return ""
	}
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Join(root, value)
}
func related(id string, raw json.RawMessage, root string, schemas map[string]checks.Schema, reader *Observer) (checks.Report, error) {
	r := checks.NewReport()
	switch id {
	case "design":
		var input struct {
			SystemDesign string `json:"systemDesign"`
			Protocol     string `json:"protocol"`
		}
		if e := strict(raw, &input); e != nil {
			return r, e
		}
		if input.SystemDesign == "" || input.Protocol == "" {
			return r, errors.New("design requires systemDesign and protocol")
		}
		return checks.Design(absoluteInput(root, input.SystemDesign), absoluteInput(root, input.Protocol), schemas, reader)
	case "artifacts":
		var input struct {
			Kind     string `json:"kind"`
			File     string `json:"file"`
			Protocol string `json:"protocol"`
		}
		if e := strict(raw, &input); e != nil {
			return r, e
		}
		if input.File == "" {
			return r, errors.New("artifacts requires file")
		}
		return checks.Artifacts(input.Kind, absoluteInput(root, input.File), absoluteInput(root, input.Protocol), schemas, reader)
	case "structure":
		var input struct {
			Design string   `json:"design"`
			Scopes []string `json:"scopes"`
		}
		if e := strict(raw, &input); e != nil {
			return r, e
		}
		if input.Design == "" {
			return r, errors.New("structure requires design")
		}
		return checks.Structure(root, absoluteInput(root, input.Design), input.Scopes, schemas, reader)
	default:
		return r, fmt.Errorf("check not implemented: %s", id)
	}
}
