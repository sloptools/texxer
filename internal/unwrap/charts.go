package unwrap

import (
	"math"
	"slices"

	"texxer/internal/mesh"
)

// geometry is the model welded by position, so faces know their neighbors.
type geometry struct {
	pos    []mesh.Vec
	tri    [][3]int
	normal []mesh.Vec
	area   []float64
	// adj holds the neighbor across the edge from corner c to c+1, or -1.
	adj [][3]int
	// smooth marks edges whose vertex normals are shared, so the model shades them round.
	smooth [][3]bool
	// texel is the finest texel the layout can get, in world units.
	texel float64
}

func newGeometry(tris []mesh.Tri) *geometry {
	g := &geometry{
		tri:    make([][3]int, len(tris)),
		normal: make([]mesh.Vec, len(tris)),
		area:   make([]float64, len(tris)),
		adj:    make([][3]int, len(tris)),
		smooth: make([][3]bool, len(tris)),
	}

	lo, hi := mesh.Vec{math.Inf(1), math.Inf(1), math.Inf(1)}, mesh.Vec{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, t := range tris {
		for _, p := range t.P {
			lo, hi = lo.Min(p), hi.Max(p)
		}
	}
	eps := max(hi.Sub(lo).Len()*1e-6, 1e-12)

	// Parts and materials stay apart even where they touch.
	type key struct {
		x, y, z int64
		part    int
		mat     string
	}
	ids := map[key]int{}
	for i, t := range tris {
		for c, p := range t.P {
			k := key{int64(math.Round(p[0] / eps)), int64(math.Round(p[1] / eps)), int64(math.Round(p[2] / eps)), t.Part, t.Material}
			id, ok := ids[k]
			if !ok {
				id = len(g.pos)
				ids[k] = id
				g.pos = append(g.pos, p)
			}
			g.tri[i][c] = id
		}
		cross := t.P[1].Sub(t.P[0]).Cross(t.P[2].Sub(t.P[0]))
		g.area[i] = cross.Len() / 2
		g.normal[i] = cross.Norm()
	}

	// Two faces are neighbors when they share an edge with opposite winding and nothing else does.
	type edge struct{ a, b int }
	owners := map[edge][]int{}
	for i, t := range g.tri {
		for c := range 3 {
			owners[edge{t[c], t[(c+1)%3]}] = append(owners[edge{t[c], t[(c+1)%3]}], i)
		}
	}
	for i, t := range g.tri {
		for c := range 3 {
			a, b := t[c], t[(c+1)%3]
			g.adj[i][c] = -1
			back := owners[edge{b, a}]
			if len(back) == 1 && len(owners[edge{a, b}]) == 1 && back[0] != i {
				n := back[0]
				g.adj[i][c] = n
				g.smooth[i][c] = sameNormal(tris[i], c, tris[n], a, g.tri[n]) && sameNormal(tris[i], (c+1)%3, tris[n], b, g.tri[n])
			}
		}
	}
	return g
}

// sameNormal compares the normal at corner c of t with the one u has at the same vertex.
func sameNormal(t mesh.Tri, c int, u mesh.Tri, v int, ids [3]int) bool {
	for k, id := range ids {
		if id == v {
			return t.N[c].Dot(u.N[k]) > 0.999
		}
	}
	return false
}

// charts grows faces into charts from the largest face on: a neighbor joins unless the
// edge between them is hard (split normals, bent more than sharp) or it turns too far
// from the chart's mean normal.
func (g *geometry) charts(faces []int, cone, sharp float64) [][]int {
	in := make(map[int]bool, len(faces))
	for _, f := range faces {
		in[f] = true
	}
	order := slices.Clone(faces)
	slices.SortStableFunc(order, func(a, b int) int {
		if g.area[a] > g.area[b] {
			return -1
		}
		if g.area[a] < g.area[b] {
			return 1
		}
		return 0
	})

	cosCone := math.Cos(cone * math.Pi / 180)
	cosSharp := math.Cos(sharp * math.Pi / 180)
	done := map[int]bool{}
	var out [][]int

	for _, seed := range order {
		if done[seed] {
			continue
		}
		done[seed] = true
		chart := []int{seed}
		sum := g.normal[seed].Scale(g.area[seed])

		for i := 0; i < len(chart); i++ {
			f := chart[i]
			mean := sum.Norm()
			for e, n := range g.adj[f] {
				if n < 0 || !in[n] || done[n] || !g.joins(f, e, mean, cosSharp, cosCone) {
					continue
				}
				done[n] = true
				chart = append(chart, n)
				sum = sum.Add(g.normal[n].Scale(g.area[n]))
			}
		}
		out = append(out, chart)
	}
	return out
}

// Slivers have no normal to judge by; they go with whoever reaches them first.
func (g *geometry) joins(f, e int, mean mesh.Vec, cosSharp, cosCone float64) bool {
	n := g.adj[f][e]
	if g.degenerate(n) {
		return true
	}
	if !g.degenerate(f) && !g.smooth[f][e] && g.normal[f].Dot(g.normal[n]) < cosSharp {
		return false
	}
	return mean.Dot(g.normal[n]) >= cosCone
}

func (g *geometry) degenerate(f int) bool {
	return g.area[f] < 1e-14
}
