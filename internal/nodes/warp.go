package nodes

import (
	"texxer/internal/graph"
	"texxer/internal/img"
	"texxer/internal/noise"
)

var warpDef = Def{
	Eval:     warpNode,
	Category: "filter",
	Inputs:   []string{"in"},
	Params: []Param{
		px("amount", 4, 0, 64),
		num("scale", 16, 1, 128),
		integer("octaves", 3, 1, 8),
		seedParam,
	},
}

// warpNode displaces its input by noise, e.g. for chipped edges or wobbly cracks.
func warpNode(e *graph.Env) (*img.Buffer, error) {
	src, err := e.Required("in")
	if err != nil {
		return nil, err
	}

	p := e.Params
	amount := e.Px("amount", 4)
	px, py := periods(e.Ctx, p.Float("scale", 16))
	octaves := p.Int("octaves", 3)
	seed := e.Seed()

	out := img.New(src.W, src.H, src.C, src.Tiling)
	img.Rows(src.H, func(y int) {
		for x := range src.W {
			u, v := cell(e.Ctx, x, y, px, py)
			ox := noise.FBM(u, v, px, py, octaves, 0.5, seed) * amount
			oy := noise.FBM(u, v, px, py, octaves, 0.5, seed+17) * amount

			for ch := range src.C {
				out.Pix[(y*src.W+x)*src.C+ch] = src.Sample(float64(x)+0.5+ox, float64(y)+0.5+oy, ch)
			}
		}
	})
	return out, nil
}
