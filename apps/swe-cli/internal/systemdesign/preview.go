package systemdesign

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
)

// Preview owns only a YAML path and the running build's renderer. It never
// accepts HTML, serves a directory, loads a template, or invokes another binary.
type Preview struct {
	file, version, route string
}

func NewPreview(file, version string) (*Preview, Report, int) {
	scope, err := FileScope(file)
	if err != nil {
		return nil, UnavailableReport("preview", version, err), 2
	}
	report, _, code := Execute(scope, "render", "", version)
	if code != 0 {
		return nil, report, code
	}
	var token [24]byte
	if _, err = rand.Read(token[:]); err != nil {
		return nil, UnavailableReport("preview", version, err), 2
	}
	return &Preview{scope.File, version, "/" + hex.EncodeToString(token[:]) + "/"}, report, 0
}

func (p *Preview) Path() string { return p.route }
func (p *Preview) File() string { return p.file }

// Host is the literal loopback listener address, preventing Host-based rebinding.
// Every valid request gets a fresh scope and one read of the YAML input. Errors
// replace the document; no prior successful page is retained or served on failure.
func (p *Preview) Handler(host string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		if r.Host != host {
			http.Error(w, "invalid preview host", http.StatusForbidden)
			return
		}
		if r.URL.Path != p.route || r.URL.RawQuery != "" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "read-only preview", http.StatusMethodNotAllowed)
			return
		}
		scope, err := FileScope(p.file)
		var report Report
		var html []byte
		code := 2
		if err != nil {
			report = UnavailableReport("preview", p.version, err)
		} else {
			report, html, code = Execute(scope, "render", "", p.version)
		}
		if code != 0 {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			status := http.StatusUnprocessableEntity
			if code == 2 {
				status = http.StatusServiceUnavailable
			}
			w.WriteHeader(status)
			if r.Method == http.MethodGet {
				_ = json.NewEncoder(w).Encode(report)
			}
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", previewCSP(html))
		if r.Method == http.MethodGet {
			_, _ = w.Write(html)
		}
	})
}

// Permit only the scripts embedded in this build. Dynamic inline styles support
// pan/zoom and column resizing; no external resources or network calls are used.
func previewCSP(html []byte) string {
	hashes := []string{}
	for _, match := range regexp.MustCompile(`(?s)<script type="application/javascript">(.*?)</script>`).FindAllStringSubmatch(string(html), -1) {
		sum := sha256.Sum256([]byte(match[1]))
		hashes = append(hashes, "'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'")
	}
	return "default-src 'none'; script-src " + strings.Join(hashes, " ") + "; style-src 'unsafe-inline'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'"
}
