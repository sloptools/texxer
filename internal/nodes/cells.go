package nodes

import (
	"fmt"

	"texxer/internal/graph"
	"texxer/internal/img"
	"texxer/internal/noise"
)

// cellsNode is Worley noise. "distance" grows from each feature point,
// "edges" is 1 on the borders between cells (cracks, aggregate outlines).
func cellsNode(e *graph.Env) (*img.Buffer, error) {
	p := e.Params
	px, py := periods(e.Ctx, p.Float("scale", 8))
	seed := p.Int("seed", 0)
	width := p.Float("width", 0.05)

	mode := p.String("mode", "distance")
	if mode != "distance" && mode != "edges" {
		return nil, fmt.Errorf("unknown cells mode %q", mode)
	}

	return fill(e.Ctx, func(x, y int) float32 {
		u, v := cell(e.Ctx, x, y, px, py)
		f1, f2 := noise.Worley(u, v, px, py, seed)

		if mode == "edges" {
			return float32(1 - smoothstep(0, width, f2-f1))
		}
		return float32(clamp01(f1))
	}), nil
}
