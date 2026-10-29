package nodes

import (
	"fmt"
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
	"texxer/internal/noise"
)

type point struct {
	i, x, y int
}

var floodFillDef = Def{
	Eval:     floodFillNode,
	Category: "filter",
	Inputs:   []string{"in"},
	Params: []Param{
		enum("mode", "random", "random", "gradient", "facets"),
		num("threshold", 0.5, 0, 1),
		num("angle", 0, -180, 180),
		num("spread", 360, 0, 360),
		num("variation", 0, 0, 1),
		integer("layers", 5, 2, 12),
		seedParam,
	},
}

// floodFillNode finds connected islands in a mask and fills each with a random
// value, a random linear gradient, or facets: several gradients at random
// angles combined into a chunky, flat-sided shape like a cut stone. Islands
// continue across tiled edges.
func floodFillNode(e *graph.Env) (*img.Buffer, error) {
	src, err := e.Required("in")
	if err != nil {
		return nil, err
	}

	p := e.Params
	mode := p.String("mode", "random")
	if mode != "random" && mode != "gradient" && mode != "facets" {
		return nil, fmt.Errorf("unknown flood fill mode %q", mode)
	}
	seed := e.Seed()
	angle := p.Float("angle", 0) * math.Pi / 180
	spread := p.Float("spread", 360) * math.Pi / 180
	variation := p.Float("variation", 0)

	out := img.New(src.W, src.H, 1, src.Tiling)

	for id, isl := range findIslands(src, float32(p.Float("threshold", 0.5))) {
		r := func(k int) float64 { return noise.Rand01(id, k, seed) }
		strength := 1 - variation*r(1)

		if mode == "random" {
			for _, pt := range isl {
				out.Pix[pt.i] = float32(r(0))
			}
			continue
		}

		if mode == "facets" {
			fillFacets(out, isl, p.Int("layers", 5), r, strength)
			continue
		}

		a := angle + (r(0)-0.5)*spread
		fillGradient(out, isl, math.Cos(a), math.Sin(a), strength)
	}
	return out, nil
}

// fillFacets takes the lowest of several ramps that each rise from one side
// of the island. Directions are spread around the circle with some jitter,
// and each ramp gets its own steepness, so every island gets its own facets.
func fillFacets(out *img.Buffer, isl []point, layers int, r func(int) float64, strength float64) {
	type ramp struct{ dx, dy, lo, span, steep float64 }

	ramps := make([]ramp, layers)
	turn := r(2) * 2 * math.Pi
	for k := range ramps {
		a := turn + (float64(k)+(r(10+k)-0.5)*0.6)*2*math.Pi/float64(layers)
		dx, dy := math.Cos(a), math.Sin(a)

		lo, hi := math.Inf(1), math.Inf(-1)
		for _, pt := range isl {
			d := float64(pt.x)*dx + float64(pt.y)*dy
			lo, hi = min(lo, d), max(hi, d)
		}
		ramps[k] = ramp{dx, dy, lo, max(hi-lo, 1), 1.2 + 1.6*r(30+k)}
	}

	vals := make([]float64, len(isl))
	peak := 1e-6
	for j, pt := range isl {
		v := math.Inf(1)
		for _, rp := range ramps {
			d := float64(pt.x)*rp.dx + float64(pt.y)*rp.dy
			v = min(v, (d-rp.lo)/rp.span*rp.steep)
		}
		vals[j] = v
		peak = max(peak, v)
	}

	for j, pt := range isl {
		out.Pix[pt.i] = float32(vals[j] / peak * strength)
	}
}

func fillGradient(out *img.Buffer, isl []point, dx, dy, strength float64) {
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, pt := range isl {
		d := float64(pt.x)*dx + float64(pt.y)*dy
		lo, hi = min(lo, d), max(hi, d)
	}

	for _, pt := range isl {
		d := float64(pt.x)*dx + float64(pt.y)*dy
		out.Pix[pt.i] = float32((d - lo) / max(hi-lo, 1) * strength)
	}
}

// findIslands groups pixels above threshold into 4-connected islands. Point
// coordinates are unwrapped, so an island crossing an edge stays contiguous.
func findIslands(b *img.Buffer, threshold float32) [][]point {
	seen := make([]bool, b.W*b.H)
	var islands [][]point

	for start := range seen {
		if seen[start] || b.Value(start, 0) <= threshold {
			continue
		}

		seen[start] = true
		queue := []point{{start, start % b.W, start / b.W}}

		for k := 0; k < len(queue); k++ {
			pt := queue[k]

			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				x, y := pt.x+d[0], pt.y+d[1]
				i, ok := wrapIndex(b, x, y)
				if !ok || seen[i] || b.Value(i, 0) <= threshold {
					continue
				}

				seen[i] = true
				queue = append(queue, point{i, x, y})
			}
		}
		islands = append(islands, queue)
	}
	return islands
}
