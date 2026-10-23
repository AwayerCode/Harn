package systemdesign

import (
	"math"
)

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
