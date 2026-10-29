package nodes

import (
	"fmt"
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
)

var repeatDef = Def{
	Eval:     repeatNode,
	Category: "filter",
	Inputs:   []string{"in"},
	Params: []Param{
		integer("count_x", 2, 1, 32),
		integer("count_y", 2, 1, 32),
	},
}

var transformDef = Def{
	Eval:     transformNode,
	Category: "filter",
	Inputs:   []string{"in"},
	Params: []Param{
		px("x", 0, -512, 512),
		px("y", 0, -512, 512),
		enum("rotate", "0", "0", "90", "180", "270"),
		flag("flip_x", false),
		flag("flip_y", false),
	},
}

// repeatNode renders its input as one cell and tiles it count_x by count_y
// times. Pixel params inside keep their size, so a module is designed at cell size.
func repeatNode(e *graph.Env) (*img.Buffer, error) {
	nx := max(1, e.Params.Int("count_x", 2))
	ny := max(1, e.Params.Int("count_y", 2))

	ctx := e.Ctx
	ctx.W = max(1, int(math.Round(float64(e.Ctx.W)/float64(nx))))
	ctx.H = max(1, int(math.Round(float64(e.Ctx.H)/float64(ny))))
	ctx.Tiling = img.TileBoth

	src, err := e.InputAt("in", ctx)
	if err != nil {
		return nil, err
	}
	if src == nil {
		return nil, fmt.Errorf(`input "in" not connected`)
	}

	cw := float64(e.Ctx.W) / float64(nx)
	ch := float64(e.Ctx.H) / float64(ny)

	return remap(e.Ctx, src, func(x, y int) (int, int) {
		sx := int(math.Mod(float64(x), cw) / cw * float64(src.W))
		sy := int(math.Mod(float64(y), ch) / ch * float64(src.H))
		return sx, sy
	}), nil
}

// transformNode moves, rotates and mirrors its input, wrapping around the edges.
func transformNode(e *graph.Env) (*img.Buffer, error) {
	rot := e.Params.String("rotate", "0")
	turned := rot == "90" || rot == "270"

	ctx := e.Ctx
	if turned {
		ctx.W, ctx.H = ctx.H, ctx.W
	}

	src, err := e.InputAt("in", ctx)
	if err != nil {
		return nil, err
	}
	if src == nil {
		return nil, fmt.Errorf(`input "in" not connected`)
	}

	ox := int(math.Round(e.Px("x", 0)))
	oy := int(math.Round(e.Px("y", 0)))
	fx, fy := e.Params.Bool("flip_x", false), e.Params.Bool("flip_y", false)
	w, h := e.Ctx.W, e.Ctx.H

	return remap(e.Ctx, src, func(x, y int) (int, int) {
		x, y = img.Mod(x-ox, w), img.Mod(y-oy, h)
		if fx {
			x = w - 1 - x
		}
		if fy {
			y = h - 1 - y
		}

		switch rot {
		case "90":
			return y, w - 1 - x
		case "180":
			return w - 1 - x, h - 1 - y
		case "270":
			return h - 1 - y, x
		}
		return x, y
	}), nil
}

// remap builds a buffer for ctx whose pixels come from src at fn(x, y).
func remap(ctx graph.Ctx, src *img.Buffer, fn func(x, y int) (int, int)) *img.Buffer {
	out := img.New(ctx.W, ctx.H, src.C, ctx.Tiling)

	img.Rows(ctx.H, func(y int) {
		for x := range ctx.W {
			sx, sy := fn(x, y)
			sx, sy = min(max(sx, 0), src.W-1), min(max(sy, 0), src.H-1)
			copy(out.Pix[(y*ctx.W+x)*src.C:][:src.C], src.Pix[(sy*src.W+sx)*src.C:][:src.C])
		}
	})
	return out
}
