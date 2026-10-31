// Package unwrap lays a model out in UV space for pixel textures: charts cut at sharp
// edges, flattened, turned to the pixel grid, all at one texel density, then packed.
package unwrap

import (
	"math"

	"texxer/internal/mesh"
)

type Options struct {
	// Size is the texture size in pixels.
	Size int
	// Padding is the gap between charts in pixels.
	Padding int
	// Hard edges (split normals) bending more than this many degrees become cuts.
	Sharp float64
	// Cone is how far in degrees a chart's faces may turn from its mean normal.
	Cone float64
}

func Defaults(size int) Options {
	return Options{Size: size, Padding: max(2, size/256), Sharp: 35, Cone: 65}
}

type Result struct {
	// UV per face corner, glTF style: v runs down the image.
	UV     [][3][2]float64
	Charts int
	// Density is in texels per world unit.
	Density float64
	// Fill is the share of the texture the charts cover.
	Fill float64
}

func Unwrap(tris []mesh.Tri, o Options) Result {
	g := newGeometry(tris)
	total := 0.0
	for _, a := range g.area {
		total += a
	}
	// As if the whole texture were covered.
	g.texel = math.Sqrt(total) / float64(o.Size)
	all := make([]int, len(tris))
	for i := range all {
		all[i] = i
	}

	var charts []*chart
	for _, faces := range g.charts(all, o.Cone, o.Sharp) {
		charts = append(charts, g.flatten(faces, o.Cone, o.Sharp)...)
	}
	for _, c := range charts {
		g.orient(c)
	}
	density := g.pack(charts, o.Size, o.Padding)

	res := Result{UV: make([][3][2]float64, len(tris)), Charts: len(charts), Density: density}
	s := float64(o.Size)
	for _, c := range charts {
		res.Fill += c.area * density * density / (s * s)
		for _, f := range c.faces {
			for k, v := range g.tri[f] {
				p := c.uv[c.local[v]]
				// The chart's up is image up, so v counts down from its top.
				res.UV[f][k] = [2]float64{
					(float64(c.x) + p[0]*density) / s,
					(float64(c.y) + (c.h-p[1])*density) / s,
				}
			}
		}
	}
	return res
}

// Overlap is the share of covered texels that more than one face covers, drawn at
// size; 1 when the model has next to no UVs at all.
func Overlap(tris []mesh.Tri, size int) float64 {
	count := make([]uint8, size*size)
	s := float64(size)
	for _, t := range tris {
		var p [3][2]float64
		for c := range 3 {
			p[c] = [2]float64{t.UV[c][0] * s, t.UV[c][1] * s}
		}
		for _, xy := range cells(p) {
			if xy[0] >= 0 && xy[1] >= 0 && xy[0] < size && xy[1] < size && count[xy[1]*size+xy[0]] < 255 {
				count[xy[1]*size+xy[0]]++
			}
		}
	}

	covered, twice := 0, 0
	for _, n := range count {
		if n > 0 {
			covered++
		}
		if n > 1 {
			twice++
		}
	}
	if covered < size*size/100 {
		return 1
	}
	return float64(twice) / float64(covered)
}
