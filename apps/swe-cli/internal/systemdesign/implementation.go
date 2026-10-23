package systemdesign

import (
	"errors"
	"fmt"
	"path/filepath"
)

// CheckGo coordinates environment, structure, import resolution and dependency checks.
// It parses all build-tag branches and never executes project code.
func (d *Design) CheckGo(root string, r *Report) error {
	return d.checkGo(root, r, NewSession())
}

func (d *Design) checkGo(root string, r *Report, session *Session) error {
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
		scopes, err := session.Discover(root)
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
	env, e := session.loadGoEnvironment(root, d.Source, r)
	if e != nil {
		return e
	}
	if e = d.checkRequiredPaths(root, r); e != nil {
		return e
	}
	if len(r.Diagnostics) > 0 {
		return nil
	}
	observed := map[string][]string{}
	files, goFiles, e := d.scanGoFiles(root, scope, r, func(file, rel string, owner *Module) error {
		imports, err := readGoImports(file)
		if err != nil {
			return err
		}
		return d.checkGoImports(imports, rel, owner, scope, env, observed, r)
	})
	if e != nil {
		return e
	}
	cycles(observed, "observed module", r)
	r.Coverage = append(r.Coverage, fmt.Sprintf("%d regular files owned; %d Go files parsed including tests, generated files and all build tags", files, goFiles), "Nearest ancestor Go module; child scopes excluded; import declarations only; external imports excluded")
	r.Unverified = append(r.Unverified, "No compiler/type resolution or build execution; build-tag alternatives are conservatively combined; external dependency internals are not inspected.")
	return nil
}
