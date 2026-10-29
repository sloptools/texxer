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
		enum("mode", "random", "random", "gradient"),
		num("threshold", 0.5, 0, 1),
		num("angle", 0, -180, 180),
		num("spread", 360, 0, 360),
		num("variation", 0, 0, 1),
		seedParam,
	},
}

// floodFillNode finds connected islands in a mask and fills each with a random
// value or a random linear gradient. Islands continue across tiled edges.
func floodFillNode(e *graph.Env) (*img.Buffer, error) {
	src, err := e.Required("in")
	if err != nil {
		return nil, err
	}

	p := e.Params
	mode := p.String("mode", "random")
	if mode != "random" && mode != "gradient" {
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

		a := angle + (r(0)-0.5)*spread
		fillGradient(out, isl, math.Cos(a), math.Sin(a), strength)
	}
	return out, nil
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
