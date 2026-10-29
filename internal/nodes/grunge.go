package nodes

import (
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
	"texxer/internal/noise"
)

// Built-in procedural grunge maps. 1 means "dirty".

var dirtDef = Def{
	Eval:     dirtNode,
	Category: "grunge",
	Params: []Param{
		num("scale", 4, 1, 64),
		num("coverage", 0.5, 0, 1),
		num("softness", 0.15, 0.01, 0.5),
		seedParam,
	},
}

func dirtNode(e *graph.Env) (*img.Buffer, error) {
	p := e.Params
	px, py := periods(e.Ctx, p.Float("scale", 4))
	seed := e.Seed()
	edge := 1 - p.Float("coverage", 0.5)
	soft := p.Float("softness", 0.15)

	return fill(e.Ctx, func(x, y int) float32 {
		u, v := cell(e.Ctx, x, y, px, py)
		n := noise.FBM(u, v, px, py, 6, 0.6, seed)*1.4 + 0.5
		return float32(smoothstep(edge-soft, edge+soft, n))
	}), nil
}

var spotsDef = Def{
	Eval:     spotsNode,
	Category: "grunge",
	Params: []Param{
		num("scale", 8, 1, 128),
		num("size", 0.35, 0.01, 1),
		seedParam,
	},
}

func spotsNode(e *graph.Env) (*img.Buffer, error) {
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
	}), nil
}

var grainDef = Def{
	Eval:     grainNode,
	Category: "grunge",
	Params:   []Param{px("size", 1, 0.25, 8), seedParam},
}

// grainNode is white noise in blocks of size px, the speckle that survives downsampling.
func grainNode(e *graph.Env) (*img.Buffer, error) {
	size := max(1, int(math.Round(e.Px("size", 1))))
	seed := e.Seed()

	return fill(e.Ctx, func(x, y int) float32 {
		return float32(noise.Rand01(x/size, y/size, seed))
	}), nil
}

var streaksDef = Def{
	Eval:     streaksNode,
	Category: "grunge",
	Params: []Param{
		num("scale", 6, 1, 32),
		num("coverage", 0.4, 0, 1),
		seedParam,
	},
}

func streaksNode(e *graph.Env) (*img.Buffer, error) {
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
	}), nil
}
