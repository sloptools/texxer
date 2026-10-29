package nodes

import (
	"texxer/internal/graph"
	"texxer/internal/img"
	"texxer/internal/noise"
)

var noiseDef = Def{
	Eval:     noiseNode,
	Category: "generator",
	Params: []Param{
		num("scale", 4, 1, 256),
		integer("octaves", 5, 1, 8),
		num("persistence", 0.5, 0, 1),
		seedParam,
	},
}

func noiseNode(e *graph.Env) (*img.Buffer, error) {
	p := e.Params
	px, py := periods(e.Ctx, p.Float("scale", 4))
	octaves := p.Int("octaves", 5)
	persistence := p.Float("persistence", 0.5)
	seed := e.Seed()

	return fill(e.Ctx, func(x, y int) float32 {
		u, v := cell(e.Ctx, x, y, px, py)
		return float32(clamp01(noise.FBM(u, v, px, py, octaves, persistence, seed)*0.5 + 0.5))
	}), nil
}
