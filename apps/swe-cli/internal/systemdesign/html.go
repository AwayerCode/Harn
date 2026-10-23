package systemdesign

import (
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"math"
	"sort"
	"strings"
)

type viewModule struct {
	Module
	Path                string
	X, Y, Width, Height int
	Lines               []string
}
type viewLayer struct {
	Layer
	Modules                   []viewModule
	Y, Height, LabelX, LabelY int
	Lines                     []string
	Color                     int
}
type viewEdge struct {
	From, To, Path string
	Kind           string
}
type diagramView struct {
	Layers                          []viewLayer
	Edges                           []viewEdge
	Report                          Report
	Width, Height, BandX, BandWidth int
}

// Wrap by display cells so Chinese descriptions and long identifiers remain inside nodes.
func diagramLines(s string, limit int) []string {
	var lines []string
	var line strings.Builder
	width := 0
	for _, c := range s {
		w := 1
		if c > 127 {
			w = 2
		}
		if c == '\n' || width+w > limit {
			lines = append(lines, line.String())
			line.Reset()
			width = 0
		}
		if c != '\n' {
			line.WriteRune(c)
			width += w
		}
	}
	if line.Len() > 0 {
		lines = append(lines, line.String())
	}
	return lines
}

// Order modules to reduce crossings between layers while retaining layer order.
func orderDiagramModules(d *Design) [][]Module {
	rows := make([][]Module, len(d.Layers))
	for i, l := range d.Layers {
		for _, m := range d.Modules {
			if m.LayerID == l.ID {
				rows[i] = append(rows[i], m)
			}
		}
	}
	cost := func() float64 {
		type pos struct{ row, col int }
		positions := map[string]pos{}
		for i, row := range rows {
			for j, m := range row {
				positions[m.ID] = pos{i, j}
			}
		}
		type link struct{ a, b pos }
		links := []link{}
		score := 0.0
		for _, m := range d.Modules {
			for _, dep := range m.Allowed {
				a, b := positions[m.ID], positions[dep]
				if a.row == b.row {
					score += math.Abs(float64(a.col-b.col)) * 0.4
					continue
				}
				score += math.Abs(float64(a.col-b.col)) * 0.15
				if a.row > b.row {
					a, b = b, a
				}
				links = append(links, link{a, b})
			}
		}
		for i, e := range links {
			for _, f := range links[i+1:] {
				if e.a.row == f.a.row && e.b.row == f.b.row && (e.a.col-f.a.col)*(e.b.col-f.b.col) < 0 {
					score += 10
				}
			}
		}
		return score
	}
	best := cost()
	for pass := 0; pass < 12; pass++ {
		changed := false
		for i := range rows {
			for j := 0; j+1 < len(rows[i]); j++ {
				rows[i][j], rows[i][j+1] = rows[i][j+1], rows[i][j]
				candidate := cost()
				if candidate < best-0.001 {
					best = candidate
					changed = true
				} else {
					rows[i][j], rows[i][j+1] = rows[i][j+1], rows[i][j]
				}
			}
		}
		if !changed {
			break
		}
	}
	return rows
}

func makeDiagram(d *Design, r Report) diagramView {
	v := diagramView{Report: r}
	rows := orderDiagramModules(d)
	layerEdges, columns := 0, 1
	rowOf := map[string]int{}
	for i, l := range d.Layers {
		layerEdges += len(l.Allowed)
		columns = max(columns, len(rows[i]))
		for _, m := range rows[i] {
			rowOf[m.ID] = i
		}
	}
	sameCount := make([]int, len(rows))
	for _, m := range d.Modules {
		for _, dep := range m.Allowed {
			if rowOf[m.ID] == rowOf[dep] {
				sameCount[rowOf[m.ID]]++
			}
		}
	}
	v.BandX = 44 + layerEdges*14
	v.BandWidth = 206 + columns*280
	v.Width = v.BandX + v.BandWidth + 24
	nodes := map[string]viewModule{}
	layerNodes := map[string]viewLayer{}
	y := 24
	for i, l := range d.Layers {
		vl := viewLayer{Layer: l, Y: y, LabelX: v.BandX + 24, Color: i % 4}
		vl.Lines = append(diagramLines(l.ID, 18), diagramLines(l.Responsibility, 20)...)
		height := max(90, 30+len(vl.Lines)*23)
		for _, m := range rows[i] {
			lines := append(diagramLines(m.ID, 24), diagramLines(m.Responsibility, 26)...)
			height = max(height, 36+len(lines)*23)
			vl.Modules = append(vl.Modules, viewModule{Module: m, Path: d.ModulePath(m), Lines: lines})
		}
		vl.Height = 24 + height + 24 + sameCount[i]*14
		vl.LabelY = y + 24
		for j := range vl.Modules {
			m := &vl.Modules[j]
			m.X, m.Y, m.Width, m.Height = v.BandX+202+j*280, y+24, 224, height
			nodes[m.ID] = *m
		}
		v.Layers = append(v.Layers, vl)
		layerNodes[l.ID] = vl
		y += vl.Height + 110
	}
	v.Height = y - 86
	lane := 0

	for _, l := range d.Layers {
		for _, to := range l.Allowed {
			a, b := layerNodes[l.ID], layerNodes[to]
			x := v.BandX - 20 - lane*14
			sy, ty := a.Y+60, b.Y+60
			v.Edges = append(v.Edges, viewEdge{l.ID, to, fmt.Sprintf("M %d %d H %d V %d H %d", v.BandX, sy, x, ty, v.BandX), "layer"})
			lane++
		}
	}
	// Distinct ports ordered by the other endpoint avoid shared arrowheads and stems.
	incoming, outgoing := map[string][]string{}, map[string][]string{}
	for _, m := range d.Modules {
		for _, dep := range m.Allowed {
			outgoing[m.ID] = append(outgoing[m.ID], dep)
			incoming[dep] = append(incoming[dep], m.ID)
		}
	}
	for _, ports := range []map[string][]string{incoming, outgoing} {
		for _, ids := range ports {
			sort.SliceStable(ids, func(i, j int) bool { return nodes[ids[i]].X < nodes[ids[j]].X })
		}
	}
	port := func(m viewModule, other string, ids []string) int {
		for i, id := range ids {
			if id == other {
				return m.X + 20 + (m.Width-40)*(i+1)/(len(ids)+1)
			}
		}
		return m.X + m.Width/2
	}
	sameLane := make([]int, len(rows))
	gapUse := make([]int, columns+1)
	for _, m := range d.Modules {
		for _, to := range m.Allowed {
			a, b := nodes[m.ID], nodes[to]
			ar, br := rowOf[m.ID], rowOf[to]
			sx, tx := port(a, to, outgoing[m.ID]), port(b, m.ID, incoming[to])
			outIndex, inIndex := 0, 0
			for i, id := range outgoing[m.ID] {
				if id == to {
					outIndex = i
				}
			}
			for i, id := range incoming[to] {
				if id == m.ID {
					inIndex = i
				}
			}
			// Reserve the left bottom ports for same-row arrivals, the right for departures.
			sx = a.X + 84 + (a.Width-104)*(outIndex+1)/(len(outgoing[m.ID])+1)
			sy, ty := a.Y+a.Height, b.Y
			path := ""
			if ar == br {
				// Same-layer dependencies occupy only this layer's lower routing strip.
				ty = b.Y + b.Height
				tx = b.X + 12 + 60*(inIndex+1)/(len(incoming[to])+1)
				lower := sy + 18 + sameLane[ar]*14
				path = fmt.Sprintf("M %d %d V %d H %d V %d", sx, sy, lower, tx, ty)
				sameLane[ar]++
			} else {
				direction := 1
				if ar > br {
					direction = -1
					sy = a.Y
					ty = b.Y + b.Height
				}
				if ar-br == 1 || br-ar == 1 {
					// Adjacent rows connect locally; curves fan out between endpoint ports.
					middle := (sy + ty) / 2
					path = fmt.Sprintf("M %d %d C %d %d %d %d %d %d", sx, sy, sx, middle, tx, middle, tx, ty)
				} else {
					// All rows share module columns. Their gutters are obstacle-free, even across skipped rows.
					chosen, best := 0, math.MaxFloat64
					for k := 0; k <= columns; k++ {
						x := v.BandX + 174 + k*280
						score := math.Abs(float64(sx-x)) + math.Abs(float64(tx-x)) + float64(gapUse[k]*60)
						if score < best {
							best = score
							chosen = k
						}
					}
					x := v.BandX + 174 + chosen*280 + (gapUse[chosen]%5-2)*6
					gapUse[chosen]++
					startY := sy + direction*(46+min(outIndex, 10)*6)
					if direction > 0 {
						startY += sameCount[ar] * 14
					}
					endY := ty - direction*(30+min(inIndex, 10)*6)
					path = fmt.Sprintf("M %d %d V %d H %d V %d H %d V %d", sx, sy, startY, x, endY, tx, ty)
				}
			}
			v.Edges = append(v.Edges, viewEdge{m.ID, to, path, "module"})
		}
	}
	return v
}

func Render(w io.Writer, d *Design, r Report) error {
	return page.Execute(w, makeDiagram(d, r))
}

//go:embed architecture.html.tmpl
var pageSource string
var page = template.Must(template.New("architecture").Funcs(template.FuncMap{
	"add":   func(a, b int) int { return a + b },
	"lineY": func(y, i int) int { return y + 28 + i*23 },
}).Parse(pageSource))
