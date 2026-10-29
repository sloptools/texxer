package nodes

import (
	"fmt"

	"texxer/internal/graph"
	"texxer/internal/img"
)

var blendModes = map[string]func(a, b float32) float32{
	"normal":   func(a, b float32) float32 { return b },
	"multiply": func(a, b float32) float32 { return a * b },
	"screen":   func(a, b float32) float32 { return 1 - (1-a)*(1-b) },
	"add":      func(a, b float32) float32 { return a + b },
	"subtract": func(a, b float32) float32 { return a - b },
	"darken":   func(a, b float32) float32 { return min(a, b) },
	"lighten":  func(a, b float32) float32 { return max(a, b) },
	"overlay": func(a, b float32) float32 {
		if a < 0.5 {
			return 2 * a * b
		}
		return 1 - 2*(1-a)*(1-b)
	},
}

var blendDef = Def{
	Eval:     blendNode,
	Category: "filter",
	Inputs:   []string{"a", "b", "mask"},
	Params: []Param{
		enum("mode", "normal", "normal", "multiply", "screen", "overlay", "add", "subtract", "darken", "lighten"),
		num("opacity", 1, 0, 1),
	},
}

// blendNode combines b onto a. An optional gray mask scales the opacity.
func blendNode(e *graph.Env) (*img.Buffer, error) {
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

	mode := e.Params.String("mode", "normal")
	fn, ok := blendModes[mode]
	if !ok {
		return nil, fmt.Errorf("unknown blend mode %q", mode)
	}
	opacity := float32(e.Params.Float("opacity", 1))

	c := max(a.C, b.C)
	out := img.New(a.W, a.H, c, a.Tiling)

	img.Rows(a.H, func(y int) {
		for x := range a.W {
			i := y*a.W + x
			m := opacity
			if mask != nil {
				m *= mask.Value(i, 0)
			}

			for ch := range c {
				av := a.Value(i, ch)
				v := av
				if ch < 3 {
					v = img.Clamp01(av + (fn(av, b.Value(i, ch))-av)*m)
				}
				out.Pix[i*c+ch] = v
			}
		}
	})
	return out, nil
}
