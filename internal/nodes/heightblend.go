package nodes

import (
	"texxer/internal/graph"
	"texxer/internal/img"
)

var heightBlendDef = Def{
	Eval:     heightBlendNode,
	Category: "filter",
	Inputs:   []string{"a", "b", "mask"},
	Outputs:  []string{"out", "mask"},
	Params: []Param{
		num("offset", 0, -1, 1),
		num("contrast", 0.9, 0, 1),
		num("opacity", 1, 0, 1),
	},
}

// heightBlendNode lays height b over height a where b, raised by offset,
// comes out on top, like moss settling over stones or water filling the
// gaps. The mask output tells where b won, for blending the colors the same way.
func heightBlendNode(e *graph.Env) (*img.Buffer, error) {
	a, err := e.Required("a")
	if err != nil {
		return nil, err
	}
	b, err := e.Required("b")
	if err != nil {
		return nil, err
	}
	mask, err := e.Input("mask")
	if err != nil {
		return nil, err
	}

	offset := float32(e.Params.Float("offset", 0))
	width := max(1-e.Params.Float("contrast", 0.9), 1e-3) / 2
	opacity := float32(e.Params.Float("opacity", 1))

	n := a.W * a.H
	out := img.New(a.W, a.H, 1, a.Tiling)
	won := img.New(a.W, a.H, 1, a.Tiling)

	for i := range n {
		ha, hb := a.Value(i, 0), b.Value(i, 0)+offset
		m := float32(smoothstep(-width, width, float64(hb-ha))) * opacity
		if mask != nil {
			m *= mask.Value(i, 0)
		}

		won.Pix[i] = m
		out.Pix[i] = ha + (hb-ha)*m
	}

	e.SetOutput("mask", won)
	return out, nil
}
