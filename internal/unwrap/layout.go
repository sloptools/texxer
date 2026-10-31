package unwrap

import (
	"math"
	"slices"
	"sort"
	"sync"

	"texxer/internal/mesh"
)

// orient scales the chart to world size, turns it so its tightest bounding rectangle
// lines up with the pixel grid, upright where it has an up, and moves it to the origin.
func (g *geometry) orient(c *chart) {
	flat := 0.0
	for _, f := range c.faces {
		flat += c.signedArea(g, f)
	}
	if flat > 0 {
		s := math.Sqrt(c.area / flat)
		for i := range c.uv {
			c.uv[i][0] *= s
			c.uv[i][1] *= s
		}
	}

	angle := minRectAngle(hull(c.uv))
	up := g.upGradient(c)
	up = rotate(up, angle)
	// Of the four quarter turns of the rectangle, the one with up pointing up.
	switch {
	case math.Abs(up[0]) > math.Abs(up[1]) && up[0] > 0:
		angle += math.Pi / 2
	case math.Abs(up[0]) > math.Abs(up[1]):
		angle -= math.Pi / 2
	case up[1] < 0:
		angle += math.Pi
	}

	lo, hi := [2]float64{math.Inf(1), math.Inf(1)}, [2]float64{math.Inf(-1), math.Inf(-1)}
	for i, p := range c.uv {
		q := rotate(p, angle)
		c.uv[i] = q
		lo = [2]float64{min(lo[0], q[0]), min(lo[1], q[1])}
		hi = [2]float64{max(hi[0], q[0]), max(hi[1], q[1])}
	}
	for i := range c.uv {
		c.uv[i][0] -= lo[0]
		c.uv[i][1] -= lo[1]
	}
	c.w, c.h = hi[0]-lo[0], hi[1]-lo[1]
}

// upGradient is the direction in the chart along which the world goes up. Floors and
// roofs take world -z as their up, like a top view.
func (g *geometry) upGradient(c *chart) [2]float64 {
	var normal mesh.Vec
	for _, f := range c.faces {
		normal = normal.Add(g.normal[f].Scale(g.area[f]))
	}
	axis, sign := 1, 1.0
	if math.Abs(normal.Norm()[1]) > 0.7 {
		axis, sign = 2, -1
	}

	var sum [2]float64
	for _, f := range c.faces {
		t := g.tri[f]
		a, b, d := c.uv[c.local[t[0]]], c.uv[c.local[t[1]]], c.uv[c.local[t[2]]]
		e1 := [2]float64{b[0] - a[0], b[1] - a[1]}
		e2 := [2]float64{d[0] - a[0], d[1] - a[1]}
		det := e1[0]*e2[1] - e2[0]*e1[1]
		if math.Abs(det) < 1e-20 {
			continue
		}
		h1 := (g.pos[t[1]][axis] - g.pos[t[0]][axis]) * sign
		h2 := (g.pos[t[2]][axis] - g.pos[t[0]][axis]) * sign
		// Solve grad·e1 = h1, grad·e2 = h2; weighted by area, which is det/2.
		gx := (h1*e2[1] - h2*e1[1]) / det
		gy := (e1[0]*h2 - e2[0]*h1) / det
		sum[0] += gx * math.Abs(det)
		sum[1] += gy * math.Abs(det)
	}
	return sum
}

func rotate(p [2]float64, a float64) [2]float64 {
	s, c := math.Sin(a), math.Cos(a)
	return [2]float64{p[0]*c - p[1]*s, p[0]*s + p[1]*c}
}

// hull is the convex hull of the points (monotone chain).
func hull(points [][2]float64) [][2]float64 {
	p := slices.Clone(points)
	sort.Slice(p, func(i, j int) bool {
		return p[i][0] < p[j][0] || p[i][0] == p[j][0] && p[i][1] < p[j][1]
	})
	cross := func(o, a, b [2]float64) float64 {
		return (a[0]-o[0])*(b[1]-o[1]) - (a[1]-o[1])*(b[0]-o[0])
	}

	var h [][2]float64
	for pass := range 2 {
		start := len(h)
		for _, q := range p {
			for len(h) >= start+2 && cross(h[len(h)-2], h[len(h)-1], q) <= 0 {
				h = h[:len(h)-1]
			}
			h = append(h, q)
		}
		h = h[:len(h)-1]
		if pass == 0 {
			slices.Reverse(p)
		}
	}
	return h
}

// minRectAngle is the rotation that lays the hull's tightest bounding rectangle along the axes.
func minRectAngle(h [][2]float64) float64 {
	best, bestArea := 0.0, math.Inf(1)
	for i := range h {
		a, b := h[i], h[(i+1)%len(h)]
		angle := -math.Atan2(b[1]-a[1], b[0]-a[0])
		lo, hi := [2]float64{math.Inf(1), math.Inf(1)}, [2]float64{math.Inf(-1), math.Inf(-1)}
		for _, p := range h {
			q := rotate(p, angle)
			lo = [2]float64{min(lo[0], q[0]), min(lo[1], q[1])}
			hi = [2]float64{max(hi[0], q[0]), max(hi[1], q[1])}
		}
		// Ties go to the smaller turn, so axis-aligned charts stay as they are.
		area := (hi[0] - lo[0]) * (hi[1] - lo[1])
		if area < bestArea*(1-1e-9) || area <= bestArea*(1+1e-9) && math.Abs(angle) < math.Abs(best) {
			best, bestArea = angle, area
		}
	}
	return best
}

// pack finds the highest texel density at which all charts fit the texture and places
// them there, by their pixel masks. Charts keep their orientation so up stays up.
func (g *geometry) pack(charts []*chart, size, padding int) float64 {
	total := 0.0
	for _, c := range charts {
		total += c.area
	}
	if total == 0 {
		return 0
	}

	// Several densities at once, each round narrowing to the gap between the best that
	// fit and the next that didn't. Above full coverage nothing fits.
	const tries = 6
	lo, hi := 0.0, float64(size)/math.Sqrt(total)
	var at [][2]int
	for round := 0; round < 3 || at == nil && round < 12; round++ {
		var wg sync.WaitGroup
		results := make([][][2]int, tries)
		for k := range tries {
			wg.Go(func() {
				results[k], _ = g.rasterLayout(charts, size, padding, lo+(hi-lo)*float64(k+1)/(tries+1))
			})
		}
		wg.Wait()

		next := hi
		for k := tries - 1; k >= 0; k-- {
			dk := lo + (hi-lo)*float64(k+1)/(tries+1)
			if results[k] != nil {
				lo, at = dk, results[k]
				break
			}
			next = dk
		}
		hi = next
	}
	if at == nil {
		return 0
	}

	for i, c := range charts {
		c.x, c.y = at[i][0], at[i][1]
	}
	return lo
}

func pixels(world, density float64) int {
	return max(1, int(math.Ceil(world*density)))
}
