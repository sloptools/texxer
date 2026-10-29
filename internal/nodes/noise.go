package nodes

import (
	"fmt"
	"math"

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
		enum("type", "fbm", "fbm", "ridged", "billow"),
		num("stretch", 1, 0.1, 16),
		seedParam,
	},
}

func noiseNode(e *graph.Env) (*img.Buffer, error) {
	p := e.Params
	px, py := periods(e.Ctx, p.Float("scale", 4))
	octaves := p.Int("octaves", 5)
	persistence := p.Float("persistence", 0.5)
	seed := e.Seed()

	// Stretch makes features longer horizontally, like brushed metal or wood grain.
	if s := p.Float("stretch", 1); s != 1 {
		px = max(1, int(math.Round(float64(px)/s)))
	}

	kind := p.String("type", "fbm")
	if kind != "fbm" && kind != "ridged" && kind != "billow" {
		return nil, fmt.Errorf("unknown noise type %q", kind)
	}

	return fill(e.Ctx, func(x, y int) float32 {
		u, v := cell(e.Ctx, x, y, px, py)
		switch kind {
		case "ridged":
			return float32(clamp01(noise.Ridged(u, v, px, py, octaves, persistence, seed)))
		case "billow":
			return float32(clamp01(noise.Billow(u, v, px, py, octaves, persistence, seed)))
		}
		return float32(clamp01(noise.FBM(u, v, px, py, octaves, persistence, seed)*0.5 + 0.5))
	}), nil
}
