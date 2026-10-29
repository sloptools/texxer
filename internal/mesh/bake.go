package mesh

import (
	"math"
	"runtime"
	"slices"
	"sync"
)

type BakeOptions struct {
	W, H int
	// Material limits baking to triangles of this material; empty bakes all.
	Material string
	AORays   int
	// AODist, EdgeWidth and CavityWidth are fractions of the model's diagonal.
	AODist      float64
	EdgeWidth   float64
	CavityWidth float64
	EdgeAngle   float64
	// FixNormals turns normals around that point into the solid, for models
	// with flipped faces. It can misjudge real interiors, so it is opt-in.
	FixNormals bool
	// Select names the parts that make up the Selected map.
	Select  []string
	Padding int
}

// Maps are baked per texel. Gray maps hold one value per texel, Normal three.
type Maps struct {
	W, H                         int
	AO, Edges, Cavity, Up        []float32
	Height, Part, Selected, Mask []float32
	Normal                       []float32
	// Position holds world positions as (p - Min) / diagonal, three per texel,
	// so decals can be placed in space. Diag is the model's diagonal.
	Position []float32
	Min      Vec
	Diag     float64
	// PartID holds (part index + 0.5) / 256 of the face that covers a texel.
	PartID []float32
}

type sample struct {
	texel    int
	p, n, ng Vec
	part     int
}

// Bake rasterizes the model's UV layout and computes geometry maps per texel.
// Texels shared by several faces, as with mirrored UVs, get the average.
func Bake(m *Model, o BakeOptions) *Maps {
	diag := m.Max.Sub(m.Min).Len()
	samples := rasterize(m, o)

	bvh := NewBVH(m.Tris)
	edges := sharpEdges(m.Tris, o.EdgeAngle, diag*1e-5)
	reach := diag * max(o.EdgeWidth, o.CavityWidth)
	grid := newEdgeGrid(edges, reach)

	selected := map[int]bool{}
	for i, name := range m.Parts {
		selected[i] = slices.Contains(o.Select, name)
	}

	n := o.W * o.H
	sum := struct{ ao, edges, cavity, up, height, part, sel, count []float64 }{
		make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n),
		make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n),
	}
	normals := make([]Vec, n)
	positions := make([]Vec, n)
	partID := make([]float32, n)

	results := make([][3]float64, len(samples))
	parallel(len(samples), func(i int) {
		s := samples[i]
		ao, flipped := occlusion(bvh, s, o.AORays, o.AODist*diag, o.FixNormals)
		if flipped {
			samples[i].n = s.n.Scale(-1)
		}
		convex, concave := grid.near(s.p, o.EdgeWidth*diag)
		if o.CavityWidth != o.EdgeWidth {
			_, concave = grid.near(s.p, o.CavityWidth*diag)
		}
		results[i] = [3]float64{ao, convex, concave}
	})

	for i, s := range samples {
		t := s.texel
		sum.ao[t] += results[i][0]
		sum.edges[t] += results[i][1]
		sum.cavity[t] += results[i][2]
		sum.up[t] += s.n[1]*0.5 + 0.5
		sum.height[t] += (s.p[1] - m.Min[1]) / max(m.Max[1]-m.Min[1], 1e-9)
		sum.part[t] += partValue(s.part)
		if selected[s.part] {
			sum.sel[t]++
		}
		normals[t] = normals[t].Add(s.n)
		positions[t] = positions[t].Add(s.p)
		partID[t] = (float32(s.part) + 0.5) / 256
		sum.count[t]++
	}

	maps := &Maps{W: o.W, H: o.H}
	avg := func(src []float64) []float32 {
		out := make([]float32, n)
		for i, c := range sum.count {
			if c > 0 {
				out[i] = float32(src[i] / c)
			}
		}
		return out
	}
	maps.AO, maps.Edges, maps.Cavity = avg(sum.ao), avg(sum.edges), avg(sum.cavity)
	maps.Up, maps.Height, maps.Part, maps.Selected = avg(sum.up), avg(sum.height), avg(sum.part), avg(sum.sel)

	maps.PartID = partID
	maps.Mask = make([]float32, n)
	maps.Normal = make([]float32, n*3)
	maps.Position = make([]float32, n*3)
	maps.Min, maps.Diag = m.Min, max(diag, 1e-9)
	for i, c := range sum.count {
		if c == 0 {
			continue
		}
		maps.Mask[i] = 1
		nv := normals[i].Norm()
		pv := positions[i].Scale(1 / c).Sub(m.Min).Scale(1 / maps.Diag)
		for k := range 3 {
			maps.Normal[i*3+k] = float32(nv[k]*0.5 + 0.5)
			maps.Position[i*3+k] = float32(pv[k])
		}
	}

	maps.pad(o.Padding)
	return maps
}

// rasterize finds the texels whose centers lie inside each triangle's UVs.
func rasterize(m *Model, o BakeOptions) []sample {
	var out []sample
	w, h := float64(o.W), float64(o.H)

	for _, t := range m.Tris {
		if o.Material != "" && t.Material != o.Material {
			continue
		}

		var u, v [3]float64
		for c := range 3 {
			u[c], v[c] = t.UV[c][0]*w, t.UV[c][1]*h
		}
		area := (u[1]-u[0])*(v[2]-v[0]) - (u[2]-u[0])*(v[1]-v[0])
		if math.Abs(area) < 1e-12 {
			continue
		}
		ng := t.P[1].Sub(t.P[0]).Cross(t.P[2].Sub(t.P[0])).Norm()

		x0 := max(0, int(math.Floor(min(u[0], u[1], u[2]))))
		x1 := min(o.W-1, int(math.Ceil(max(u[0], u[1], u[2]))))
		y0 := max(0, int(math.Floor(min(v[0], v[1], v[2]))))
		y1 := min(o.H-1, int(math.Ceil(max(v[0], v[1], v[2]))))

		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				px, py := float64(x)+0.5, float64(y)+0.5
				b0 := ((u[1]-px)*(v[2]-py) - (u[2]-px)*(v[1]-py)) / area
				b1 := ((u[2]-px)*(v[0]-py) - (u[0]-px)*(v[2]-py)) / area
				b2 := 1 - b0 - b1
				if b0 < 0 || b1 < 0 || b2 < 0 {
					continue
				}

				p := t.P[0].Scale(b0).Add(t.P[1].Scale(b1)).Add(t.P[2].Scale(b2))
				n := t.N[0].Scale(b0).Add(t.N[1].Scale(b1)).Add(t.N[2].Scale(b2)).Norm()
				out = append(out, sample{y*o.W + x, p, n, ng, t.Part})
			}
		}
	}
	return out
}

// occlusion is the fraction of cosine-weighted rays that escape, 1 fully
// open. With fix, mostly occluded surfaces are tested from their other side: when
// that side is much more open, the normal points into the solid, as with
// flipped faces, and flipped reports it so lighting can turn it around.
func occlusion(b *BVH, s sample, rays int, dist float64, fix bool) (open float64, flipped bool) {
	if rays <= 0 || dist <= 0 {
		return 1, false
	}

	open = hemisphere(b, s, s.n, rays, dist)
	if fix && open < 0.5 {
		if back := hemisphere(b, s, s.n.Scale(-1), rays, dist); back > open+0.3 {
			return back, true
		}
	}
	return open, false
}

func hemisphere(b *BVH, s sample, n Vec, rays int, dist float64) float64 {
	// Start just off the surface on the side n points to.
	ng := s.ng
	if ng.Dot(n) < 0 {
		ng = ng.Scale(-1)
	}
	origin := s.p.Add(ng.Scale(dist * 1e-3))

	t, bt := basis(n)
	open := 0
	for k := range rays {
		r1 := fract(float64(k)*0.618034 + hash01(s.texel, 1))
		r2 := (float64(k) + hash01(s.texel, 2)) / float64(rays)
		phi := 2 * math.Pi * r1
		sr := math.Sqrt(r2)
		d := t.Scale(math.Cos(phi) * sr).Add(bt.Scale(math.Sin(phi) * sr)).Add(n.Scale(math.Sqrt(1 - r2)))

		if !b.Occluded(origin, d.Norm(), dist) {
			open++
		}
	}
	return float64(open) / float64(rays)
}

func basis(n Vec) (Vec, Vec) {
	up := Vec{0, 1, 0}
	if math.Abs(n[1]) > 0.9 {
		up = Vec{1, 0, 0}
	}
	t := up.Cross(n).Norm()
	return t, n.Cross(t)
}

// pad grows the maps a few texels past their UV islands, so filtering and
// mipmaps don't pull in the empty background at seams.
func (m *Maps) pad(texels int) {
	covered := slices.Clone(m.Mask)
	gray := [][]float32{m.AO, m.Edges, m.Cavity, m.Up, m.Height, m.Part, m.Selected}

	for range texels {
		next := slices.Clone(covered)
		for y := range m.H {
			for x := range m.W {
				i := y*m.W + x
				if covered[i] > 0 {
					continue
				}

				var nb []int
				for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					nx, ny := x+d[0], y+d[1]
					if nx >= 0 && nx < m.W && ny >= 0 && ny < m.H && covered[ny*m.W+nx] > 0 {
						nb = append(nb, ny*m.W+nx)
					}
				}
				if len(nb) == 0 {
					continue
				}

				// Ids can't be averaged, so padding copies one neighbor.
				m.PartID[i] = m.PartID[nb[0]]
				for _, buf := range gray {
					var s float32
					for _, j := range nb {
						s += buf[j]
					}
					buf[i] = s / float32(len(nb))
				}
				for _, buf := range [][]float32{m.Normal, m.Position} {
					for k := range 3 {
						var s float32
						for _, j := range nb {
							s += buf[j*3+k]
						}
						buf[i*3+k] = s / float32(len(nb))
					}
				}
				next[i] = 1
			}
		}
		covered = next
	}
}

func partValue(part int) float64 {
	return hash01(part, 7)
}

func hash01(x, seed int) float64 {
	h := uint32(x)*374761393 + uint32(seed)*668265263
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h) / (1 << 32)
}

func fract(v float64) float64 {
	return v - math.Floor(v)
}

func parallel(n int, fn func(i int)) {
	workers := runtime.GOMAXPROCS(0)
	var wg sync.WaitGroup
	chunk := (n + workers - 1) / workers

	for w := range workers {
		start, end := w*chunk, min(n, (w+1)*chunk)
		if start >= end {
			break
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := start; i < end; i++ {
				fn(i)
			}
		}()
	}
	wg.Wait()
}
