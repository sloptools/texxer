package nodes

import (
	"fmt"
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
)

var rampDef = Def{
	Eval:     rampNode,
	Category: "generator",
	Params: []Param{
		enum("mode", "linear", "linear", "mirror", "radial"),
		num("angle", 90, -180, 180),
		num("repeat", 1, 0.1, 64),
		num("offset", 0, -1, 1),
		num("x", 0, -0.5, 0.5),
		num("y", 0, -0.5, 0.5),
	},
}

// rampNode is a gradient from 0 to 1: linear along angle (90 runs top to
// bottom), mirror goes there and back, radial grows from the point x, y
// relative to the center. repeat restarts the gradient, so whole numbers at
// 0 or 90 degrees tile.
func rampNode(e *graph.Env) (*img.Buffer, error) {
	p := e.Params
	mode := p.String("mode", "linear")
	if mode != "linear" && mode != "mirror" && mode != "radial" {
		return nil, fmt.Errorf("unknown mode %q", mode)
	}

	a := p.Float("angle", 90) * math.Pi / 180
	dx, dy := math.Cos(a), math.Sin(a)
	rep := p.Float("repeat", 1)
	offset := p.Float("offset", 0)
	cx, cy := 0.5+p.Float("x", 0), 0.5+p.Float("y", 0)
	w, h := float64(e.Ctx.W), float64(e.Ctx.H)

	return fill(e.Ctx, func(x, y int) float32 {
		u, v := (float64(x)+0.5)/w, (float64(y)+0.5)/h

		var t float64
		if mode == "radial" {
			t = math.Hypot(u-cx, v-cy) * 2
		} else {
			t = u*dx + v*dy
		}
		t = t*rep + offset

		if mode == "radial" && rep <= 1 {
			return float32(clamp01(t))
		}
		f := t - math.Floor(t)
		if mode == "mirror" {
			f = 1 - math.Abs(2*f-1)
		}
		return float32(f)
	}), nil
}
