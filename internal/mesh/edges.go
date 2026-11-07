package mesh

import "math"

// edge is a sharp mesh edge in world space.
type edge struct {
	a, b   Vec
	convex bool
	// weight grows with the angle between the faces.
	weight float64
}

// sharpEdges finds edges where the faces meet at more than minAngle degrees,
// after welding vertices that share a position. Open borders count as convex.
func sharpEdges(tris []Tri, minAngle, weldDist float64) []edge {
	type key [3]int64
	quant := func(p Vec) key {
		return key{int64(math.Round(p[0] / weldDist)), int64(math.Round(p[1] / weldDist)), int64(math.Round(p[2] / weldDist))}
	}

	ids := map[key]int{}
	pos := []Vec{}
	weld := func(p Vec) int {
		k := quant(p)
		if id, ok := ids[k]; ok {
			return id
		}
		ids[k] = len(pos)
		pos = append(pos, p)
		return len(pos) - 1
	}

	type pair [2]int
	faces := map[pair][]int{}
	normals := make([]Vec, len(tris))
	centers := make([]Vec, len(tris))

	for i, t := range tris {
		normals[i] = t.P[1].Sub(t.P[0]).Cross(t.P[2].Sub(t.P[0])).Norm()
		centers[i] = t.P[0].Add(t.P[1]).Add(t.P[2]).Scale(1.0 / 3)

		v := [3]int{weld(t.P[0]), weld(t.P[1]), weld(t.P[2])}
		for c := range 3 {
			a, b := v[c], v[(c+1)%3]
			if a == b {
				continue
			}
			if a > b {
				a, b = b, a
			}
			faces[pair{a, b}] = append(faces[pair{a, b}], i)
		}
	}

	cosMin := math.Cos(minAngle * math.Pi / 180)
	var out []edge
	for p, fs := range faces {
		a, b := pos[p[0]], pos[p[1]]

		if len(fs) == 1 {
			out = append(out, edge{a, b, true, 1})
			continue
		}
		if len(fs) != 2 {
			continue
		}

		n1, n2 := normals[fs[0]], normals[fs[1]]
		cos := n1.Dot(n2)
		if cos > cosMin {
			continue
		}

		// Convex when the second face falls away behind the first one's plane.
		mid := a.Add(b).Scale(0.5)
		convex := n1.Dot(centers[fs[1]].Sub(mid)) < 0
		angle := math.Acos(max(-1, min(1, cos)))
		out = append(out, edge{a, b, convex, min(1, angle/(math.Pi/2))})
	}
	return out
}

// edgeGrid finds edges near a point quickly.
type edgeGrid struct {
	cell  float64
	cells map[[3]int64][]int
	edges []edge
}

func newEdgeGrid(edges []edge, reach float64) *edgeGrid {
	g := &edgeGrid{cell: max(reach, 1e-6), cells: map[[3]int64][]int{}, edges: edges}

	for i, e := range edges {
		lo := e.a.Min(e.b).Sub(Vec{reach, reach, reach})
		hi := e.a.Max(e.b).Add(Vec{reach, reach, reach})
		c0, c1 := g.key(lo), g.key(hi)

		for x := c0[0]; x <= c1[0]; x++ {
			for y := c0[1]; y <= c1[1]; y++ {
				for z := c0[2]; z <= c1[2]; z++ {
					k := [3]int64{x, y, z}
					g.cells[k] = append(g.cells[k], i)
				}
			}
		}
	}
	return g
}

func (g *edgeGrid) key(p Vec) [3]int64 {
	return [3]int64{int64(math.Floor(p[0] / g.cell)), int64(math.Floor(p[1] / g.cell)), int64(math.Floor(p[2] / g.cell))}
}

// near returns how close p is to convex and concave edges: 1 on an edge,
// falling to 0 at width.
func (g *edgeGrid) near(p Vec, width float64) (convex, concave float64) {
	for _, i := range g.cells[g.key(p)] {
		e := g.edges[i]
		d := segmentDist(p, e.a, e.b)
		if d >= width {
			continue
		}

		v := (1 - d/width) * e.weight
		if e.convex {
			convex = max(convex, v)
		} else {
			concave = max(concave, v)
		}
	}
	return convex, concave
}

func segmentDist(p, a, b Vec) float64 {
	ab := b.Sub(a)
	t := 0.0
	if l := ab.Dot(ab); l > 0 {
		t = max(0, min(1, p.Sub(a).Dot(ab)/l))
	}
	return p.Sub(a.Add(ab.Scale(t))).Len()
}
