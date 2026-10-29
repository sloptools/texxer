package nodes

import (
	"fmt"
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
)

var patternDef = Def{
	Eval:     patternNode,
	Category: "shape",
	Params: []Param{
		enum("shape", "box", "box", "diamond", "circle", "groove", "lines", "chamfer", "arch", "gothic"),
		px("cell_w", 32, 2, 512),
		px("cell_h", 0, 0, 512),
		px("gap", 2, 0, 64),
		px("radius", 0, 0, 64),
		px("bevel", 3, 0.5, 64),
		enum("profile", "round", "round", "linear", "flat"),
		flag("stagger", false),
	},
}

// patternNode repeats a shape as a height map. Cell sizes are rounded so the
// repetition always fits the buffer exactly and stays seamless.
func patternNode(e *graph.Env) (*img.Buffer, error) {
	p := e.Params
	name := p.String("shape", "box")
	shape, ok := shapes[name]
	if !ok {
		return nil, fmt.Errorf("unknown shape %q", name)
	}

	cw := e.Px("cell_w", 32)
	ch := e.Px("cell_h", 0)
	if ch <= 0 {
		ch = cw
	}
	gap := e.Px("gap", 2)
	radius := e.Px("radius", 0)
	bevel := max(e.Px("bevel", 3), 0.5)
	stagger := p.Bool("stagger", false)
	profile := p.String("profile", "round")

	nx := max(1, int(math.Round(float64(e.Ctx.W)/cw)))
	ny := max(1, int(math.Round(float64(e.Ctx.H)/ch)))
	if stagger && ny%2 != 0 && e.Ctx.Tiling == img.TileBoth {
		ny++
	}
	cw = float64(e.Ctx.W) / float64(nx)
	ch = float64(e.Ctx.H) / float64(ny)

	hw, hh := cw/2-gap, ch/2-gap

	return fill(e.Ctx, func(x, y int) float32 {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		row := int(math.Floor(fy / ch))
		d := math.Inf(1)

		// Neighbor cells too, so shapes larger than their cell overlap cleanly.
		for dr := -1; dr <= 1; dr++ {
			r := row + dr
			offset := 0.0
			if stagger && r%2 != 0 {
				offset = cw / 2
			}
			col := int(math.Floor((fx - offset) / cw))

			for dc := -1; dc <= 1; dc++ {
				cx := (float64(col+dc)+0.5)*cw + offset
				cy := (float64(r) + 0.5) * ch
				d = min(d, shape(fx-cx, fy-cy, hw, hh, radius))
			}
		}

		return float32(bevelProfile(profile, -d/bevel))
	}), nil
}

func bevelProfile(name string, t float64) float64 {
	t = clamp01(t)

	switch name {
	case "linear":
		return t
	case "flat":
		return math.Ceil(t)
	default:
		return math.Sqrt(1 - (1-t)*(1-t))
	}
}
