package nodes

import (
	"errors"

	"texxer/internal/graph"
	"texxer/internal/img"
)

var downsampleDef = Def{
	Eval:     downsampleNode,
	Category: "output",
	Inputs:   []string{"in"},
	Params:   []Param{integer("factor", 4, 1, 8)},
}

var sharpenDef = Def{
	Eval:     sharpenNode,
	Category: "filter",
	Inputs:   []string{"in"},
	Params:   []Param{num("amount", 0.5, 0, 4)},
}

// downsampleNode renders its input at factor times the resolution and
// box-filters it down. Fine detail collapsing into few pixels is a big part of the retro look.
func downsampleNode(e *graph.Env) (*img.Buffer, error) {
	f := max(1, e.Params.Int("factor", 4))

	ctx := e.Ctx
	ctx.W *= f
	ctx.H *= f
	ctx.Scale *= float64(f)

	src, err := e.InputAt("in", ctx)
	if err != nil {
		return nil, err
	}
	if src == nil {
		return nil, errors.New(`input "in" not connected`)
	}

	out := img.New(e.Ctx.W, e.Ctx.H, src.C, src.Tiling)
	n := float32(f * f)

	img.Rows(out.H, func(y int) {
		for x := range out.W {
			for ch := range src.C {
				var sum float32
				for dy := range f {
					for dx := range f {
						sum += src.Pix[((y*f+dy)*src.W+x*f+dx)*src.C+ch]
					}
				}
				out.Pix[(y*out.W+x)*out.C+ch] = sum / n
			}
		}
	})
	return out, nil
}

// sharpenNode is an unsharp mask with a 1px radius.
func sharpenNode(e *graph.Env) (*img.Buffer, error) {
	src, err := e.Required("in")
	if err != nil {
		return nil, err
	}

	amount := float32(e.Params.Float("amount", 0.5))
	soft := boxPass(boxPass(src, 1, 1, 0), 1, 0, 1)

	out := img.New(src.W, src.H, src.C, src.Tiling)
	for i, v := range src.Pix {
		if src.C == 4 && i%4 == 3 {
			out.Pix[i] = v
			continue
		}
		out.Pix[i] = img.Clamp01(v + (v-soft.Pix[i])*amount)
	}
	return out, nil
}
