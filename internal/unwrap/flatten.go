package unwrap

import (
	"math"

	"texxer/internal/mesh"
)

// chart is a group of faces laid flat. uv is per chart vertex, in world units until packed.
type chart struct {
	faces []int
	verts []int
	local map[int]int
	uv    [][2]float64
	area  float64
	// Size after orienting, in world units, and the packed position in pixels.
	w, h float64
	x, y int
}

func (g *geometry) newChart(faces []int) *chart {
	c := &chart{faces: faces, local: map[int]int{}}
	for _, f := range faces {
		c.area += g.area[f]
		for _, v := range g.tri[f] {
			if _, ok := c.local[v]; !ok {
				c.local[v] = len(c.verts)
				c.verts = append(c.verts, v)
			}
		}
	}
	c.uv = make([][2]float64, len(c.verts))
	return c
}

// flatten lays faces flat; where that stretches them too much, it splits them
// into charts with a tighter cone and tries again.
func (g *geometry) flatten(faces []int, cone, sharp float64) []*chart {
	c := g.newChart(faces)
	if g.project(c) {
		return []*chart{c}
	}
	g.lscm(c)
	if len(faces) == 1 || cone < 8 || g.even(c) && !g.overlaps(c) {
		return []*chart{c}
	}

	var out []*chart
	for _, sub := range g.charts(faces, cone*0.6, sharp) {
		out = append(out, g.flatten(sub, cone*0.6, sharp)...)
	}
	return out
}

// project lays the chart onto the plane of its mean normal, faces keeping their winding.
// It reports whether the chart is flat, so the projection is exact.
func (g *geometry) project(c *chart) bool {
	var sum mesh.Vec
	for _, f := range c.faces {
		sum = sum.Add(g.normal[f].Scale(g.area[f]))
	}
	n := sum.Norm()
	e1 := perpendicular(n)
	e2 := n.Cross(e1)
	for i, v := range c.verts {
		c.uv[i] = [2]float64{g.pos[v].Dot(e1), g.pos[v].Dot(e2)}
	}

	flat := math.Cos(0.5 * math.Pi / 180)
	for _, f := range c.faces {
		if !g.degenerate(f) && g.normal[f].Dot(n) < flat {
			return false
		}
	}
	return true
}

func perpendicular(n mesh.Vec) mesh.Vec {
	a := mesh.Vec{1, 0, 0}
	if math.Abs(n[0]) > 0.9 {
		a = mesh.Vec{0, 1, 0}
	}
	return a.Sub(n.Scale(a.Dot(n))).Norm()
}

// even reports whether no face is flipped and every face keeps roughly its share of the area.
func (g *geometry) even(c *chart) bool {
	total := 0.0
	for _, f := range c.faces {
		total += c.signedArea(g, f)
	}
	if total <= 0 {
		return false
	}
	ratio := total / c.area
	for _, f := range c.faces {
		if g.degenerate(f) {
			continue
		}
		r := c.signedArea(g, f) / (g.area[f] * ratio)
		if r < 1.0/3 || r > 3 {
			return false
		}
	}
	return true
}

// overlaps reports whether the chart folds over itself, by drawing it on a fine grid.
func (g *geometry) overlaps(c *chart) bool {
	lo, hi := [2]float64{math.Inf(1), math.Inf(1)}, [2]float64{math.Inf(-1), math.Inf(-1)}
	for _, p := range c.uv {
		lo = [2]float64{min(lo[0], p[0]), min(lo[1], p[1])}
		hi = [2]float64{max(hi[0], p[0]), max(hi[1], p[1])}
	}
	// Half a texel at the highest density, so any overlap that could cover a texel shows.
	cell := max(g.texel/2, max(hi[0]-lo[0], hi[1]-lo[1])/4096)
	if cell == 0 {
		return false
	}

	hit := newBitmap(int((hi[0]-lo[0])/cell)+2, int((hi[1]-lo[1])/cell)+2)
	for _, f := range c.faces {
		t := g.tri[f]
		var p [3][2]float64
		for k, v := range t {
			q := c.uv[c.local[v]]
			p[k] = [2]float64{(q[0] - lo[0]) / cell, (q[1] - lo[1]) / cell}
		}
		for _, xy := range cells(p) {
			if hit.get(xy[0], xy[1]) {
				return true
			}
			hit.set(xy[0], xy[1])
		}
	}
	return false
}

// cells lists the grid cells whose centers lie strictly inside the triangle.
func cells(p [3][2]float64) [][2]int {
	edge := func(a, b [2]float64, x, y float64) float64 {
		return (b[0]-a[0])*(y-a[1]) - (b[1]-a[1])*(x-a[0])
	}
	var out [][2]int
	x0 := int(min(p[0][0], p[1][0], p[2][0]))
	x1 := int(max(p[0][0], p[1][0], p[2][0]))
	y0 := int(min(p[0][1], p[1][1], p[2][1]))
	y1 := int(max(p[0][1], p[1][1], p[2][1]))
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			cx, cy := float64(x)+0.5, float64(y)+0.5
			e0, e1, e2 := edge(p[0], p[1], cx, cy), edge(p[1], p[2], cx, cy), edge(p[2], p[0], cx, cy)
			if e0 < 0 && e1 < 0 && e2 < 0 || e0 > 0 && e1 > 0 && e2 > 0 {
				out = append(out, [2]int{x, y})
			}
		}
	}
	return out
}

func (c *chart) signedArea(g *geometry, f int) float64 {
	t := g.tri[f]
	a, b, d := c.uv[c.local[t[0]]], c.uv[c.local[t[1]]], c.uv[c.local[t[2]]]
	return ((b[0]-a[0])*(d[1]-a[1]) - (d[0]-a[0])*(b[1]-a[1])) / 2
}

// lscm finds the most angle-preserving layout (least squares conformal maps, Lévy 2002),
// starting from the projection, with the two vertices furthest apart along it pinned.
func (g *geometry) lscm(c *chart) {
	pinA, pinB := c.extremes()
	col := make([]int, len(c.verts))
	free := 0
	for i := range c.verts {
		if i == pinA || i == pinB {
			col[i] = -1
			continue
		}
		col[i] = free
		free++
	}

	// Two rows per face: real and imaginary part of sum W_j (u_j + i v_j) / sqrt(2A).
	type entry struct {
		col int
		val float64
	}
	var rows [][]entry
	var rhs []float64
	for _, f := range c.faces {
		if g.degenerate(f) {
			continue
		}
		t := g.tri[f]
		z := localTriangle(g.pos[t[0]], g.pos[t[1]], g.pos[t[2]])
		s := 1 / math.Sqrt(2*g.area[f])
		var re, im []entry
		var bre, bim float64
		for j := range 3 {
			k, l := (j+1)%3, (j+2)%3
			a := (z[l][0] - z[k][0]) * s
			b := (z[l][1] - z[k][1]) * s
			i := c.local[t[j]]
			if col[i] < 0 {
				u, v := c.uv[i][0], c.uv[i][1]
				bre -= a*u - b*v
				bim -= b*u + a*v
				continue
			}
			re = append(re, entry{2 * col[i], a}, entry{2*col[i] + 1, -b})
			im = append(im, entry{2 * col[i], b}, entry{2*col[i] + 1, a})
		}
		rows = append(rows, re, im)
		rhs = append(rhs, bre, bim)
	}

	x := make([]float64, 2*free)
	for i, k := range col {
		if k >= 0 {
			x[2*k], x[2*k+1] = c.uv[i][0], c.uv[i][1]
		}
	}

	mul := func(x []float64) []float64 {
		out := make([]float64, len(rows))
		for r, row := range rows {
			for _, e := range row {
				out[r] += e.val * x[e.col]
			}
		}
		return out
	}
	mulT := func(y []float64) []float64 {
		out := make([]float64, len(x))
		for r, row := range rows {
			for _, e := range row {
				out[e.col] += e.val * y[r]
			}
		}
		return out
	}
	cgls(mul, mulT, rhs, x, max(200, 4*len(x)))

	for i, k := range col {
		if k >= 0 {
			c.uv[i] = [2]float64{x[2*k], x[2*k+1]}
		}
	}
}

// extremes are the chart vertices furthest apart along the longer side of the projection.
func (c *chart) extremes() (int, int) {
	lo, hi := [2]float64{math.Inf(1), math.Inf(1)}, [2]float64{math.Inf(-1), math.Inf(-1)}
	for _, p := range c.uv {
		lo = [2]float64{min(lo[0], p[0]), min(lo[1], p[1])}
		hi = [2]float64{max(hi[0], p[0]), max(hi[1], p[1])}
	}
	axis := 0
	if hi[1]-lo[1] > hi[0]-lo[0] {
		axis = 1
	}
	a, b := 0, 0
	for i, p := range c.uv {
		if p[axis] < c.uv[a][axis] {
			a = i
		}
		if p[axis] > c.uv[b][axis] {
			b = i
		}
	}
	if a == b {
		b = (a + 1) % len(c.uv)
	}
	return a, b
}

// localTriangle puts a face into its own plane, counterclockwise seen from its front.
func localTriangle(p0, p1, p2 mesh.Vec) [3][2]float64 {
	e1 := p1.Sub(p0)
	l := e1.Len()
	e1 = e1.Norm()
	n := e1.Cross(p2.Sub(p0)).Norm()
	e2 := n.Cross(e1)
	d := p2.Sub(p0)
	return [3][2]float64{{0, 0}, {l, 0}, {d.Dot(e1), d.Dot(e2)}}
}

// cgls solves min |A x - b| by conjugate gradients on the normal equations, from x on.
func cgls(mul, mulT func([]float64) []float64, b, x []float64, iterations int) {
	if len(x) == 0 {
		return
	}
	ax := mul(x)
	r := make([]float64, len(b))
	for i := range r {
		r[i] = b[i] - ax[i]
	}
	s := mulT(r)
	p := append([]float64(nil), s...)
	gamma := dot(s, s)
	start := gamma

	for range iterations {
		if gamma <= start*1e-20 || gamma == 0 {
			return
		}
		q := mul(p)
		qq := dot(q, q)
		if qq == 0 {
			return
		}
		alpha := gamma / qq
		for i := range x {
			x[i] += alpha * p[i]
		}
		for i := range r {
			r[i] -= alpha * q[i]
		}
		s = mulT(r)
		next := dot(s, s)
		beta := next / gamma
		gamma = next
		for i := range p {
			p[i] = s[i] + beta*p[i]
		}
	}
}

func dot(a, b []float64) float64 {
	s := 0.0
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}
