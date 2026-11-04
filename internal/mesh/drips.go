package mesh

import "math"

// dripGrid holds the sharp edges that lie rather than stand, which water runs off.
func dripGrid(edges []edge, length float64) *edgeGrid {
	var lying []edge
	for _, e := range edges {
		d := e.b.Sub(e.a)
		if l := d.Len(); l > 0 && math.Abs(d[1])/l < 0.7 {
			lying = append(lying, e)
		}
	}
	return newEdgeGrid(lying, length)
}

// drip is 1 on a wall just below an edge right above it, fading to 0 at length
// below; reach is how far out from the wall the edge may sit.
func (g *edgeGrid) drip(p, n Vec, length, reach float64) float64 {
	if math.Abs(n[1]) > 0.8 {
		return 0
	}

	best := 0.0
	for _, i := range g.cells[g.key(p)] {
		e := g.edges[i]
		ab := e.b.Sub(e.a)
		flat := Vec{ab[0], 0, ab[2]}
		t := 0.0
		if l := flat.Dot(flat); l > 0 {
			t = max(0, min(1, Vec{p[0] - e.a[0], 0, p[2] - e.a[2]}.Dot(flat)/l))
		}

		q := e.a.Add(ab.Scale(t))
		dy := q[1] - p[1]
		if dy <= 0 || dy >= length || math.Hypot(q[0]-p[0], q[2]-p[2]) >= reach {
			continue
		}
		best = max(best, 1-dy/length)
	}
	return best
}
