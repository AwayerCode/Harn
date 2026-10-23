package architecture

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/mod/modfile"
)

// regularPath rejects symlinks in every component, including source_root ancestors.
func regularPath(root, rel string) (fs.FileInfo, error) {
	if !SafePath(rel) {
		return nil, fmt.Errorf("unsafe path: %s", rel)
	}
	current := root
	var info fs.FileInfo
	for _, part := range strings.Split(rel, "/") {
		current = filepath.Join(current, part)
		var e error
		info, e = os.Lstat(current)
		if e != nil {
			return nil, e
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("symlink not supported: %s", current)
		}
	}
	return info, nil
}
func (d *Design) owner(rel string) *Module {
	for i := range d.Modules {
		p := d.ModulePath(d.Modules[i])
		if rel == p || strings.HasPrefix(rel, p+"/") {
			return &d.Modules[i]
		}
	}
	return nil
}

// CheckGo parses all Go import declarations, independently of build tags. It never runs project code.
func (d *Design) CheckGo(root string, r *Report) error {
	root, e := filepath.Abs(root)
	if e != nil {
		return e
	}
	root, e = filepath.EvalSymlinks(root)
	if e != nil {
		return e
	}
	info, e := regularPath(root, "go.mod")
	if e != nil {
		return e
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("go.mod must be a regular file")
	}
	b, e := os.ReadFile(filepath.Join(root, "go.mod"))
	if e != nil {
		return e
	}
	parsed, e := modfile.Parse("go.mod", b, nil)
	if e != nil {
		return fmt.Errorf("parse go.mod: %w", e)
	}
	if parsed.Module == nil || parsed.Module.Mod.Path == "" {
		return fmt.Errorf("go.mod requires a module directive")
	}
	module := parsed.Module.Mod.Path
	// Replacements change package ownership and require a resolver beyond v1.
	if len(parsed.Replace) > 0 {
		return fmt.Errorf("go.mod replace directives are not supported by architecture v1")
	}
	if e := checkWorkspace(root); e != nil {
		return e
	}
	for p := path.Dir(d.Source); p != "."; p = path.Dir(p) {
		if _, e := os.Lstat(filepath.Join(root, filepath.FromSlash(p), "go.mod")); e == nil {
			return fmt.Errorf("nested Go module above source_root: %s", p)
		} else if !os.IsNotExist(e) {
			return e
		}
	}
	requiredDir := func(rel string) error {
		info, e := regularPath(root, rel)
		if os.IsNotExist(e) {
			r.Add("structure.missing", rel, "required directory missing")
			return nil
		}
		if e != nil {
			return e
		}
		if !info.IsDir() {
			r.Add("structure.directory", rel, "expected directory")
		}
		return nil
	}
	if e = requiredDir(d.Source); e != nil {
		return e
	}
	for _, l := range d.Layers {
		if e = requiredDir(path.Join(d.Source, l.Directory)); e != nil {
			return e
		}
	}
	for _, m := range d.Modules {
		p := d.ModulePath(m)
		if e = requiredDir(p); e != nil {
			return e
		}
		for _, f := range m.Required {
			rel := path.Join(p, f)
			info, e := regularPath(root, rel)
			if os.IsNotExist(e) {
				r.Add("structure.missing", rel, "required file missing")
				continue
			}
			if e != nil {
				return e
			}
			if !info.Mode().IsRegular() {
				r.Add("structure.file", rel, "expected regular file")
			}
		}
	}
	if len(r.Diagnostics) > 0 {
		return nil
	}
	files := 0
	goFiles := 0
	observed := map[string][]string{}
	e = filepath.WalkDir(filepath.Join(root, filepath.FromSlash(d.Source)), func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, e := filepath.Rel(root, p)
		if e != nil {
			return e
		}
		rel = filepath.ToSlash(rel)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink not supported: %s", rel)
		}
		if entry.IsDir() {
			if entry.Name() == "vendor" {
				return fmt.Errorf("vendor directories inside source_root are unsupported: %s", rel)
			}
			if rel == d.Source {
				return nil
			}
			valid := false
			for _, l := range d.Layers {
				if rel == path.Join(d.Source, l.Directory) {
					valid = true
				}
			}
			if !valid && d.owner(rel) == nil {
				r.Add("structure.unowned", rel, "undeclared directory outside modules")
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("non-regular file: %s", rel)
		}
		if entry.Name() == "go.mod" || entry.Name() == "go.work" {
			return fmt.Errorf("nested Go modules/workspaces are unsupported: %s", rel)
		}
		files++
		owner := d.owner(rel)
		if owner == nil {
			r.Add("structure.unowned", rel, "file has no module owner")
			return nil
		}
		if !strings.HasSuffix(rel, ".go") {
			return nil
		}
		goFiles++
		set := token.NewFileSet()
		f, e := parser.ParseFile(set, p, nil, parser.ImportsOnly)
		if e != nil {
			return e
		}
		for _, imp := range f.Imports {
			target, e := strconv.Unquote(imp.Path.Value)
			if e != nil {
				return e
			}
			if target != module && !strings.HasPrefix(target, module+"/") {
				continue
			}
			local := strings.TrimPrefix(strings.TrimPrefix(target, module), "/")
			to := d.owner(local)
			if to == nil {
				r.Add("dependency.unowned", fmt.Sprintf("%s:%d", rel, set.Position(imp.Pos()).Line), "local import outside managed modules: "+target)
				continue
			}
			// A nested module or symlink cannot be accepted as an ordinary local package.
			info, e := regularPath(root, local)
			if e != nil {
				if os.IsNotExist(e) {
					r.Add("dependency.missing", rel, "local import directory missing: "+target)
					continue
				}
				return e
			}
			if !info.IsDir() {
				r.Add("dependency.missing", rel, "local import is not a directory: "+target)
				continue
			}
			if to.ID == owner.ID {
				continue
			}
			r.Edges = append(r.Edges, Edge{owner.ID, to.ID, rel, set.Position(imp.Pos()).Line})
			if !slices.Contains(observed[owner.ID], to.ID) {
				observed[owner.ID] = append(observed[owner.ID], to.ID)
			}
			if !slices.Contains(owner.Allowed, to.ID) {
				r.Add("dependency.forbidden", fmt.Sprintf("%s:%d", rel, set.Position(imp.Pos()).Line), owner.ID+" -> "+to.ID+" is not allowed")
			}
		}
		return nil
	})
	if e != nil {
		return e
	}
	cycles(observed, "observed module", r)
	r.Coverage = append(r.Coverage, fmt.Sprintf("%d regular files owned; %d Go files parsed including tests, generated files and all build tags", files, goFiles), "Single root Go module; import declarations only; external imports excluded")
	r.Unverified = append(r.Unverified, "No compiler/type resolution or build execution; build-tag alternatives are conservatively combined; external dependency internals are not inspected.")
	return nil
}

// Respect Go's workspace selection, including an explicit path outside root's ancestors.
func checkWorkspace(root string) error {
	switch setting := os.Getenv("GOWORK"); setting {
	case "off":
		return nil
	case "", "auto":
		for p := root; ; p = filepath.Dir(p) {
			if _, e := os.Lstat(filepath.Join(p, "go.work")); e == nil {
				return fmt.Errorf("Go workspaces are not supported by architecture v1: %s", p)
			} else if !os.IsNotExist(e) {
				return e
			}
			if filepath.Dir(p) == p {
				return nil
			}
		}
	default:
		if !filepath.IsAbs(setting) {
			return fmt.Errorf("GOWORK must be off, auto or an absolute path: %s", setting)
		}
		return fmt.Errorf("explicit GOWORK is not supported by architecture v1: %s", setting)
	}
}
