package app

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/AwayerCode/Harn/cli/swe-cli/internal/checks"
)

var Version = "0.5.0-dev"

const maxRequest = 1024 * 1024

// Version 2 deliberately rejects the removed extension protocol.
const ProtocolVersion = 2

type Input struct {
	Kind    string `json:"kind"`
	File    string `json:"file"`
	Profile string `json:"profile,omitempty"`
}
type CheckRequest struct {
	CheckID  string          `json:"checkId"`
	Required *bool           `json:"required"`
	Input    json.RawMessage `json:"input"`
}
type Request struct {
	ProtocolVersion int            `json:"protocolVersion"`
	TaskID          string         `json:"taskId"`
	Round           int64          `json:"round"`
	Mode            string         `json:"mode"`
	ProjectRoot     string         `json:"projectRoot"`
	Checks          []CheckRequest `json:"checks"`
}
type Diagnostic struct {
	RuleID   string `json:"ruleId"`
	Message  string `json:"message"`
	Location string `json:"location,omitempty"`
}
type Result struct {
	CheckID     string       `json:"checkId"`
	Source      string       `json:"source"`
	Required    bool         `json:"required"`
	Status      string       `json:"status"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Coverage    string       `json:"coverage"`
	Evidence    []string     `json:"evidence"`
	Unverified  []string     `json:"unverified,omitempty"`
}
type Summary struct {
	Status   string   `json:"status"`
	ExitCode int      `json:"exitCode"`
	Warnings []string `json:"warnings"`
}

func digest(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }
func strict(data []byte, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return errors.New("request must contain exactly one JSON value")
	}
	return nil
}
func encode(out io.Writer, v any, code int) int {
	if json.NewEncoder(out).Encode(v) != nil {
		return 2
	}
	return code
}
func unavailable(out io.Writer, err error) int {
	return encode(out, map[string]any{"protocolVersion": ProtocolVersion, "error": Diagnostic{RuleID: "request.unavailable", Message: err.Error()}, "summary": Summary{Status: "unavailable", ExitCode: 2, Warnings: []string{}}}, 2)
}
func Run(args []string, in io.Reader, out io.Writer) int {
	if len(args) > 0 && args[0] == "system-design" {
		return SystemDesign(args[1:], out)
	}
	if len(args) > 0 && args[0] == "scope" {
		return Scope(args[1:], out)
	}
	if len(args) > 0 && slices.Contains([]string{"documents", "design", "artifacts", "structure"}, args[0]) {
		return Direct(args[0], args[1:], out)
	}
	if len(args) == 0 || len(args) == 1 && (args[0] == "help" || args[0] == "--help") {
		_, err := fmt.Fprintln(out, "swe-cli check --request <file|-> [--format json]\nswe-cli version\nGo core: documents, design, artifacts, structure; scope snapshot/check/readonly; system-design list --root <dir>; system-design validate/check --file <system-design.yaml> or --root <dir> --all; system-design render --file <system-design.yaml>. No extensions.")
		if err != nil {
			return 2
		}
		return 0
	}
	if len(args) == 1 && args[0] == "version" {
		return encode(out, map[string]any{"version": Version, "protocolVersion": ProtocolVersion, "runtime": runtime.Version(), "standalone": true}, 0)
	}
	if args[0] != "check" {
		return unavailable(out, errors.New("unknown command"))
	}
	options := map[string]string{}
	for i := 1; i < len(args); i += 2 {
		if !slices.Contains([]string{"--request", "--format"}, args[i]) || options[args[i]] != "" || i+1 >= len(args) {
			return unavailable(out, errors.New("invalid command options"))
		}
		options[args[i]] = args[i+1]
	}
	if options["--request"] == "" || options["--format"] != "" && options["--format"] != "json" {
		return unavailable(out, errors.New("use --request <file|-> --format json"))
	}
	source := in
	if options["--request"] != "-" {
		file, err := os.Open(options["--request"])
		if err != nil {
			return unavailable(out, err)
		}
		defer file.Close()
		source = file
	}
	data, err := io.ReadAll(io.LimitReader(source, maxRequest+1))
	if err != nil {
		return unavailable(out, err)
	}
	if len(data) > maxRequest {
		return unavailable(out, errors.New("request exceeds 1 MiB"))
	}
	var req Request
	if err := strict(data, &req); err != nil {
		return unavailable(out, err)
	}
	root, err := validateRequest(req)
	if err != nil {
		return unavailable(out, err)
	}
	var schemas map[string]checks.Schema
	if err := json.Unmarshal([]byte(checks.SchemaJSON), &schemas); err != nil {
		return unavailable(out, err)
	}
	reader := &Observer{Root: root, Files: map[string]string{}}
	results := []Result{}
	for _, c := range req.Checks {
		r := Result{CheckID: c.CheckID, Source: "core", Required: *c.Required, Status: "unavailable", Diagnostics: []Diagnostic{}, Evidence: []string{}, Coverage: "Only requested checks; not task completion"}
		if slices.Contains([]string{"design", "artifacts", "structure"}, c.CheckID) {
			report, err := related(c.CheckID, c.Input, root, schemas, reader)
			if err != nil {
				r.Diagnostics = append(r.Diagnostics, Diagnostic{RuleID: c.CheckID + ".unavailable", Message: err.Error()})
			} else {
				r.Status = "passed"
				if len(report.Errors) > 0 {
					r.Status = "failed"
				}
				for _, message := range report.Errors {
					r.Diagnostics = append(r.Diagnostics, Diagnostic{RuleID: c.CheckID + ".relation", Message: message})
				}
				r.Evidence = report.Checked
				r.Unverified = report.Unverified
				r.Coverage = "Explicit relations only; unverified items require manual review"
			}
		} else if c.CheckID != "documents" {
			r.Diagnostics = append(r.Diagnostics, Diagnostic{RuleID: "check.unavailable", Message: "Check not implemented: " + c.CheckID})
		} else {
			var input Input
			err := strict(c.Input, &input)
			schema, ok := schemas[input.Kind]
			if err == nil && (!ok || strings.TrimSpace(input.File) == "" || input.Profile != "" && input.Profile != "architecture") {
				err = errors.New("invalid documents input")
			}
			if err == nil {
				file := input.File
				if !filepath.IsAbs(file) {
					file = filepath.Join(root, file)
				}
				var problems []string
				problems, err = checks.Validate(file, input.Kind, schema, reader, input.Profile)
				if err == nil {
					r.Status = "passed"
					if len(problems) > 0 {
						r.Status = "failed"
					}
					for _, message := range problems {
						r.Diagnostics = append(r.Diagnostics, Diagnostic{RuleID: "documents.format", Message: message, Location: file})
					}
					r.Coverage = "Document format and local links; not semantic correctness"
					if input.Profile != "" {
						r.Coverage = "Partial:" + input.Profile + "; not full design validation"
					}
					r.Evidence = append(r.Evidence, file)
				}
			}
			if err != nil {
				r.Diagnostics = append(r.Diagnostics, Diagnostic{RuleID: "check.unavailable", Message: err.Error()})
			}
		}
		results = append(results, r)
	}
	if changed := reader.Changed(); len(changed) > 0 {
		r := Result{CheckID: "input-stability", Source: "core", Required: true, Status: "unavailable", Diagnostics: []Diagnostic{}, Evidence: []string{}, Coverage: "Observed file content, path resolution and link existence"}
		for _, file := range changed {
			r.Diagnostics = append(r.Diagnostics, Diagnostic{RuleID: "input.changed", Message: "Observed input changed during execution", Location: file})
		}
		results = append(results, r)
	}
	code := 0
	warnings := []string{}
	for _, r := range results {
		if !r.Required {
			if r.Status != "passed" {
				warnings = append(warnings, r.CheckID)
			}
			continue
		}
		if r.Status == "unavailable" || r.Status == "not-run" {
			code = 2
		} else if r.Status == "failed" && code == 0 {
			code = 1
		}
	}
	status := []string{"passed", "failed", "unavailable"}[code]
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return unavailable(out, err)
	}
	identityData, _ := json.Marshal(struct {
		Request Request
		Files   map[string]string
	}{req, reader.Files})
	return encode(out, map[string]any{"protocolVersion": ProtocolVersion, "taskId": req.TaskID, "round": req.Round, "identity": map[string]any{"runId": hex.EncodeToString(id), "coreVersion": Version, "rulesDigest": digest([]byte(checks.SchemaJSON)), "inputDigest": digest(identityData), "files": reader.Files}, "checks": results, "summary": Summary{Status: status, ExitCode: code, Warnings: warnings}}, code)
}
func validateRequest(req Request) (string, error) {
	if req.ProtocolVersion != ProtocolVersion {
		return "", fmt.Errorf("unsupported protocolVersion; use %d (extensions removed)", ProtocolVersion)
	}
	if strings.TrimSpace(req.TaskID) == "" || req.Round < 1 || req.Round > 9007199254740991 || !slices.Contains([]string{"design", "implement", "test", "review"}, req.Mode) || !filepath.IsAbs(req.ProjectRoot) {
		return "", errors.New("invalid task, round, mode or absolute projectRoot")
	}
	root, err := filepath.EvalSymlinks(req.ProjectRoot)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("projectRoot must be a directory")
	}
	ids := map[string]bool{}
	required := false
	for _, c := range req.Checks {
		if strings.TrimSpace(c.CheckID) == "" || ids[c.CheckID] || c.Required == nil || len(bytes.TrimSpace(c.Input)) == 0 || bytes.TrimSpace(c.Input)[0] != '{' {
			return "", errors.New("invalid or duplicate checkId, required or input")
		}
		ids[c.CheckID] = true
		required = required || *c.Required
	}
	if !required {
		return "", errors.New("at least one required check is needed")
	}
	return root, nil
}

// Observer binds results to content and lexical-to-real path resolution. It is not a filesystem lock.
type Observer struct {
	Root  string
	Files map[string]string
}

func inside(root, file string) bool {
	rel, err := filepath.Rel(root, file)
	return err == nil && !filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
func (o *Observer) snapshot(file string) (string, []byte, error) {
	file = filepath.Clean(file)
	if !inside(o.Root, file) {
		return "", nil, errors.New("input outside projectRoot: " + file)
	}
	real, err := filepath.EvalSymlinks(file)
	if os.IsNotExist(err) {
		return "missing", nil, nil
	}
	if err != nil {
		return "", nil, err
	}
	if !inside(o.Root, real) {
		return "", nil, errors.New("resolved input outside projectRoot: " + file)
	}
	info, err := os.Stat(real)
	if err != nil {
		return "", nil, err
	}
	if info.IsDir() {
		return "directory:" + real, nil, nil
	}
	if !info.Mode().IsRegular() {
		return "", nil, errors.New("input is not a regular file: " + file)
	}
	data, err := os.ReadFile(real)
	if err != nil {
		return "", nil, err
	}
	return "file:" + real + ":" + digest(data), data, nil
}
func (o *Observer) observe(file string) (string, []byte, error) {
	state, data, err := o.snapshot(file)
	if err != nil {
		return "", nil, err
	}
	if old, ok := o.Files[file]; ok && old != state {
		return "", nil, errors.New("input changed during check: " + file)
	}
	o.Files[file] = state
	return state, data, nil
}
func (o *Observer) Read(file string) (string, error) {
	state, data, err := o.observe(file)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(state, "file:") {
		return "", fmt.Errorf("not a readable file: %s", file)
	}
	if !utf8.Valid(data) {
		return "", errors.New("input is not UTF-8: " + file)
	}
	return string(data), nil
}
func (o *Observer) Exists(file string) (bool, error) {
	state, _, err := o.observe(file)
	return state != "missing", err
}
func (o *Observer) Changed() []string {
	changed := []string{}
	for file, previous := range o.Files {
		now, _, err := o.snapshot(file)
		if err != nil || now != previous {
			changed = append(changed, file)
		}
	}
	slices.Sort(changed)
	return changed
}

func (o *Observer) PathKind(file string) (string, error) {
	state, _, err := o.observe(file)
	if err != nil {
		return "", err
	}
	kind, _, _ := strings.Cut(state, ":")
	return kind, nil
}
