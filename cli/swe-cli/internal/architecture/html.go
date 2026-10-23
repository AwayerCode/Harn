package architecture

import (
	"html/template"
	"io"
)

type viewModule struct {
	Module
	Path string
}
type viewLayer struct {
	Layer
	Modules []viewModule
}

func Render(w io.Writer, d *Design, r Report) error {
	layers := []viewLayer{}
	for _, l := range d.Layers {
		v := viewLayer{Layer: l}
		for _, m := range d.Modules {
			if m.LayerID == l.ID {
				v.Modules = append(v.Modules, viewModule{m, d.ModulePath(m)})
			}
		}
		layers = append(layers, v)
	}
	return page.Execute(w, struct {
		Layers []viewLayer
		Report Report
	}{layers, r})
}

var page = template.Must(template.New("architecture").Parse(`<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>模块架构</title>
<style>body{font:16px system-ui,sans-serif;background:#f3f6fa;color:#19283a;max-width:1100px;margin:40px auto;padding:0 24px}header,section{margin:24px 0}section{border-left:5px solid #3678bc;background:white;padding:20px;border-radius:10px}.modules{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:16px}details{border:1px solid #ced9e5;border-radius:8px;padding:16px;scroll-margin-top:24px}details:target{outline:3px solid #e4a43b}summary{cursor:pointer;font-weight:700}code{overflow-wrap:anywhere;font-size:13px}a{color:#245b99}small{color:#4b5f73}li{margin:6px 0}.flow{padding:12px;background:#edf3f9;border-radius:6px}</style></head>
<body><header><h1>模块架构</h1><p>只读设计视图 · 修改 YAML 后重新生成 · 箭头表示允许的源码依赖</p><small>设计 SHA256：<code>{{.Report.Digest}}</code><br>生成器：{{.Report.Version}} · 仅设计校验，不代表实现已验收</small></header>
{{range .Layers}}<section id="layer-{{.ID}}"><h2>{{.ID}}</h2><p>{{.Responsibility}}</p><p>层目录：<code>{{.Directory}}/</code></p><p>允许跨层依赖：{{range .Allowed}}<a href="#layer-{{.}}">→ {{.}}</a> {{else}}无{{end}}</p><div class="modules">{{range .Modules}}<details open id="module-{{.ID}}"><summary>{{.ID}}</summary><p>{{.Responsibility}}</p><code>{{.Path}}/</code><h3>必要文件</h3><ul>{{range .Required}}<li><code>{{.}}</code></li>{{else}}<li>未指定</li>{{end}}</ul><div class="flow">{{.ID}} {{range .Allowed}}<a href="#module-{{.}}">→ {{.}}</a> {{else}}· 无跨模块依赖{{end}}</div></details>{{end}}</div></section>{{end}}
</body></html>`))
