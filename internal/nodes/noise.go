package nodes

import (
	"texxer/internal/graph"
	"texxer/internal/img"
	"texxer/internal/noise"
)

func noiseNode(e *graph.Env) (*img.Buffer, error) {
	p := e.Params
	px, py := periods(e.Ctx, p.Float("scale", 4))
	octaves := p.Int("octaves", 5)
	persistence := p.Float("persistence", 0.5)
	seed := p.Int("seed", 0)

	return fill(e.Ctx, func(x, y int) float32 {
		u, v := cell(e.Ctx, x, y, px, py)
		return float32(clamp01(noise.FBM(u, v, px, py, octaves, persistence, seed)*0.5 + 0.5))
	}), nil
}
