package nodes

import (
	"fmt"
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
	"texxer/internal/noise"
)

// Built-in procedural grunge maps. 1 means "dirty".

func grungeNode(e *graph.Env) (*img.Buffer, error) {
	switch kind := e.Params.String("kind", "dirt"); kind {
	case "dirt":
		return dirt(e), nil
	case "spots":
		return spots(e), nil
	case "streaks":
		return streaks(e), nil
	case "scratches":
		return scratches(e), nil
	case "grain":
		return grain(e), nil
	default:
		return nil, fmt.Errorf("unknown grunge kind %q", kind)
	}
}

func dirt(e *graph.Env) *img.Buffer {
	p := e.Params
	px, py := periods(e.Ctx, p.Float("scale", 4))
	seed := e.Seed()
	edge := 1 - p.Float("coverage", 0.5)
	soft := p.Float("softness", 0.15)

	return fill(e.Ctx, func(x, y int) float32 {
		u, v := cell(e.Ctx, x, y, px, py)
		n := noise.FBM(u, v, px, py, 6, 0.6, seed)*1.4 + 0.5
		return float32(smoothstep(edge-soft, edge+soft, n))
	})
}

func spots(e *graph.Env) *img.Buffer {
	p := e.Params
	px, py := periods(e.Ctx, p.Float("scale", 8))
	seed := e.Seed()
	size := p.Float("size", 0.35)

	return fill(e.Ctx, func(x, y int) float32 {
		u, v := cell(e.Ctx, x, y, px, py)
		warp := noise.FBM(u, v, px, py, 4, 0.5, seed+7)
		r := size * (1 + warp)
		d, _ := noise.Worley(u+warp*0.4, v-warp*0.4, px, py, seed)
		return float32(1 - smoothstep(r*0.5, r, d))
	})
}

// grain is white noise in blocks of size px, the speckle that survives downsampling.
func grain(e *graph.Env) *img.Buffer {
	size := max(1, int(math.Round(e.Px("size", 1))))
	seed := e.Seed()

	return fill(e.Ctx, func(x, y int) float32 {
		return float32(noise.Rand01(x/size, y/size, seed))
	})
}

func streaks(e *graph.Env) *img.Buffer {
	p := e.Params
	scale := p.Float("scale", 6)
	seed := e.Seed()
	edge := 1 - p.Float("coverage", 0.4)

	// Stretched cells: many columns, few rows.
	px, _ := periods(e.Ctx, scale*6)
	_, py := periods(e.Ctx, scale/2)
	fx, fy := periods(e.Ctx, scale)

	return fill(e.Ctx, func(x, y int) float32 {
		u, v := cell(e.Ctx, x, y, px, py)
		n := noise.FBM(u, v, px, py, 4, 0.5, seed)*1.4 + 0.5

		fu, fv := cell(e.Ctx, x, y, fx, fy)
		fade := noise.FBM(fu, fv, fx, fy, 3, 0.5, seed+3)*1.2 + 0.5
		return float32(smoothstep(edge-0.2, edge+0.2, n) * clamp01(fade))
	})
}
