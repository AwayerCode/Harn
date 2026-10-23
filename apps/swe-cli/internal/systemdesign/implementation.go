package systemdesign

import (
	"errors"
	"fmt"
	"go/build"
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
	scope := Scope{Root: root, Children: []string{}}
	if r.Scope != nil {
		scope = *r.Scope
	} else {
		scopes, err := Discover(root)
		if err != nil {
			return err
		}
		for _, s := range scopes.Scopes {
			if s.Root == root {
				scope = s
			}
		}
	}
	if scope.Unavailable != "" {
		return errors.New(scope.Unavailable)
	}
	d.ValidateScope(scope, r)
	if len(r.Diagnostics) > 0 {
		return nil
	}
	moduleRoot, e := findModuleRoot(root)
	if e != nil {
		return e
	}
	info, e := regularPath(moduleRoot, "go.mod")
	if e != nil {
		return e
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("go.mod must be a regular file")
	}
	b, e := os.ReadFile(filepath.Join(moduleRoot, "go.mod"))
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
	foreignModules, uncertain, e := localScopeModules(root, moduleRoot)
	if e != nil {
		return e
	}
	for _, issue := range uncertain {
		r.Unverified = append(r.Unverified, "Local module metadata unavailable: "+issue)
	}
	// Replacements change package ownership and require a resolver beyond v1.
	if len(parsed.Replace) > 0 {
		return fmt.Errorf("go.mod replace directives are not supported by system-design v1")
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
		if entry.Name() == ".git" {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if scope.Delegated(rel) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
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
			foreign := false
			matchLength := -1
			if target == module || strings.HasPrefix(target, module+"/") {
				matchLength = len(module)
			}
			for _, prefix := range foreignModules {
				if (target == prefix || strings.HasPrefix(target, prefix+"/")) && len(prefix) > matchLength {
					foreign = true
					matchLength = len(prefix)
				}
			}
			if foreign {
				r.Add("dependency.cross-scope", rel, "import targets another local system-design scope: "+target)
				continue
			}
			if target != module && !strings.HasPrefix(target, module+"/") {
				if len(uncertain) > 0 && !standardImport(target) {
					return fmt.Errorf("cannot classify import %s while local module metadata is unavailable: %s", target, strings.Join(uncertain, "; "))
				}
				continue
			}
			moduleLocal := strings.TrimPrefix(strings.TrimPrefix(target, module), "/")
			local, err := filepath.Rel(root, filepath.Join(moduleRoot, filepath.FromSlash(moduleLocal)))
			if err != nil {
				return err
			}
			local = filepath.ToSlash(local)
			if local == ".." || strings.HasPrefix(local, "../") || scope.Delegated(local) {
				r.Add("dependency.cross-scope", rel, "import leaves this system-design scope: "+target)
				continue
			}
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
	r.Coverage = append(r.Coverage, fmt.Sprintf("%d regular files owned; %d Go files parsed including tests, generated files and all build tags", files, goFiles), "Nearest ancestor Go module; child scopes excluded; import declarations only; external imports excluded")
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
				return fmt.Errorf("Go workspaces are not supported by system-design v1: %s", p)
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
		return fmt.Errorf("explicit GOWORK is not supported by system-design v1: %s", setting)
	}
}

// A design may manage a subdirectory of a larger Go module.
var errNoModule = errors.New("no Go module")

func findModuleRoot(root string) (string, error) {
	for p := root; ; p = filepath.Dir(p) {
		info, err := os.Lstat(filepath.Join(p, "go.mod"))
		if err == nil {
			if !info.Mode().IsRegular() {
				return "", fmt.Errorf("go.mod must be regular: %s", p)
			}
			return p, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		if _, err := os.Lstat(filepath.Join(p, ".git")); err == nil {
			return "", fmt.Errorf("%w within repository for %s", errNoModule, root)
		} else if !os.IsNotExist(err) {
			return "", err
		}
		if filepath.Dir(p) == p {
			return "", fmt.Errorf("%w in ancestors for %s", errNoModule, root)
		}
	}
}

// Other independently managed Go modules in the same repository are not
// silently treated as third-party libraries. No dependency code is executed.
func localScopeModules(root, moduleRoot string) ([]string, []string, error) {
	search := moduleRoot
	for p := root; ; p = filepath.Dir(p) {
		if _, err := os.Lstat(filepath.Join(p, ".git")); err == nil {
			search = p
			break
		} else if !os.IsNotExist(err) {
			return nil, nil, err
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	discovery, err := Discover(search)
	if err != nil {
		return nil, nil, err
	}
	result := []string{}
	uncertain := []string{}
	for _, issue := range discovery.Issues {
		uncertain = append(uncertain, issue.Path+": "+issue.Message)
	}
	for _, s := range discovery.Scopes {
		other, err := findModuleRoot(s.Root)
		if err != nil {
			if !errors.Is(err, errNoModule) {
				uncertain = append(uncertain, s.Root+": "+err.Error())
			}
			continue
		}
		if other == moduleRoot {
			continue
		}
		b, err := os.ReadFile(filepath.Join(other, "go.mod"))
		if err != nil {
			uncertain = append(uncertain, other+": "+err.Error())
			continue
		}
		modulePath := modfile.ModulePath(b)
		if modulePath != "" {
			result = append(result, modulePath)
		} else {
			uncertain = append(uncertain, "missing module path: "+other)
		}
	}
	return result, uncertain, nil
}

// Classify standard-library imports without running go list or project code.
func standardImport(target string) bool {
	if target == "C" {
		return true
	} // cgo pseudo-package, not another Go module
	pkg, err := build.Default.Import(target, "", build.FindOnly)
	return err == nil && pkg.Goroot
}
