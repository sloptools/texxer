package nodes

import (
	"fmt"
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
	"texxer/internal/noise"
)

var tileSamplerDef = Def{
	Eval:     tileSamplerNode,
	Category: "shape",
	Inputs:   []string{"pattern", "mask"},
	Outputs:  []string{"out", "id"},
	Params: []Param{
		integer("count_x", 8, 1, 128),
		integer("count_y", 8, 1, 128),
		enum("shape", "circle", "circle", "box", "chamfer", "diamond", "arch", "gothic"),
		num("size", 0.8, 0.05, 4),
		num("size_var", 0.2, 0, 1),
		num("aspect", 1, 0.1, 10),
		num("rotation", 0, -180, 180),
		num("rotation_var", 0, 0, 180),
		num("jitter", 0, 0, 1),
		num("stagger", 0, 0, 1),
		num("drop", 0, 0, 1),
		num("value_var", 0, 0, 1),
		px("pattern_size", 64, 8, 512),
		px("bevel", 4, 0.5, 64),
		enum("profile", "round", "round", "linear", "flat"),
		enum("blend", "max", "max", "add"),
		seedParam,
	},
}

type instance struct {
	cx, cy    float64
	cos, sin  float64
	hw, hh    float64
	value, id float64
	visible   bool
}

// tileSamplerNode scatters copies of a pattern, or a built-in shape, over a
// grid with random size, rotation, offset and brightness, like stones,
// rivets or leaves. The mask input thins instances out where it is dark.
func tileSamplerNode(e *graph.Env) (*img.Buffer, error) {
	p := e.Params
	nx, ny := max(1, p.Int("count_x", 8)), max(1, p.Int("count_y", 8))
	cw, ch := float64(e.Ctx.W)/float64(nx), float64(e.Ctx.H)/float64(ny)

	size := p.Float("size", 0.8)
	sizeVar := p.Float("size_var", 0.2)
	aspect := max(p.Float("aspect", 1), 0.01)
	rot := p.Float("rotation", 0) * math.Pi / 180
	rotVar := p.Float("rotation_var", 0) * math.Pi / 180
	jitter := p.Float("jitter", 0)
	stagger := p.Float("stagger", 0)
	drop := p.Float("drop", 0)
	valueVar := p.Float("value_var", 0)
	add := p.String("blend", "max") == "add"
	seed := e.Seed()

	base := min(cw, ch) * size / 2
	hwBase, hhBase := base*math.Sqrt(aspect), base/math.Sqrt(aspect)
	reachPx := math.Hypot(hwBase, hhBase)*(1+sizeVar) + max(cw, ch)*(jitter+stagger)
	rx := int(math.Ceil(reachPx/cw)) + 1
	ry := int(math.Ceil(reachPx/ch)) + 1

	mask, err := e.Input("mask")
	if err != nil {
		return nil, err
	}

	shapeAt, err := tileShape(e, hwBase*(1+sizeVar), hhBase*(1+sizeVar))
	if err != nil {
		return nil, err
	}

	// Instances are computed once per cell, then looked up per pixel.
	cells := make([]instance, nx*ny)
	for r := range ny {
		for c := range nx {
			s := func(k int) float64 { return noise.Rand01(c, r, seed+k*7919) }

			cx := (float64(c) + 0.5 + (s(1)-0.5)*jitter) * cw
			if r%2 == 1 {
				cx += stagger * cw / 2
			}
			cy := (float64(r) + 0.5 + (s(2)-0.5)*jitter) * ch

			scale := 1 + (s(3)*2-1)*sizeVar
			a := rot + (s(4)*2-1)*rotVar

			visible := s(5) >= drop
			if visible && mask != nil {
				mx := img.Mod(int(cx), e.Ctx.W)
				my := img.Mod(int(cy), e.Ctx.H)
				visible = float64(mask.Value(my*e.Ctx.W+mx, 0)) > s(6)
			}

			cells[r*nx+c] = instance{
				cx: cx, cy: cy,
				cos: math.Cos(a), sin: math.Sin(a),
				hw: hwBase * scale, hh: hhBase * scale,
				value:   1 - valueVar*s(7),
				id:      s(8),
				visible: visible,
			}
		}
	}

	ids := img.New(e.Ctx.W, e.Ctx.H, 1, e.Ctx.Tiling)
	out := fill(e.Ctx, func(x, y int) float32 {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		col, row := int(math.Floor(fx/cw)), int(math.Floor(fy/ch))

		var v, top, topID float64
		for dr := -ry; dr <= ry; dr++ {
			for dc := -rx; dc <= rx; dc++ {
				c, r := col+dc, row+dr
				in := cells[img.Mod(r, ny)*nx+img.Mod(c, nx)]
				if !in.visible {
					continue
				}

				// Instance centers repeat with the grid, so shift them into this copy.
				cx := in.cx + float64(c-img.Mod(c, nx))*cw
				cy := in.cy + float64(r-img.Mod(r, ny))*ch

				dx, dy := fx-cx, fy-cy
				u := (dx*in.cos - dy*in.sin) / in.hw
				w := (dx*in.sin + dy*in.cos) / in.hh
				if math.Abs(u) >= 1.5 || math.Abs(w) >= 1.5 {
					continue
				}

				h := shapeAt(u, w, in) * in.value
				if h <= 0 {
					continue
				}
				if add {
					v += h
				} else {
					v = max(v, h)
				}
				if h > top {
					top, topID = h, in.id
				}
			}
		}

		ids.Pix[y*e.Ctx.W+x] = float32(topID)
		return float32(v)
	})

	e.SetOutput("id", ids)
	return out, nil
}

// tileShape returns the height of one instance at local coordinates u, w
// (-1..1 across the instance): the pattern input if connected, else a shape.
// The pattern is designed in a pattern_size square and rendered big enough
// for the largest instance.
func tileShape(e *graph.Env, maxHW, maxHH float64) (func(u, w float64, in instance) float64, error) {
	side := max(8, int(math.Ceil(2*max(maxHW, maxHH))))
	design := max(e.Params.Float("pattern_size", 64), 1)
	ctx := graph.Ctx{W: side, H: side, Tiling: img.TileNone, Scale: float64(side) / design}

	pattern, err := e.InputAt("pattern", ctx)
	if err != nil {
		return nil, err
	}
	if pattern != nil {
		return func(u, w float64, _ instance) float64 {
			if math.Abs(u) >= 1 || math.Abs(w) >= 1 {
				return 0
			}
			return float64(pattern.Sample((u+1)/2*float64(side), (w+1)/2*float64(side), 0))
		}, nil
	}

	name := e.Params.String("shape", "circle")
	shape, ok := shapes[name]
	if !ok {
		return nil, fmt.Errorf("unknown shape %q", name)
	}

	bevel := max(e.Px("bevel", 4), 0.5)
	profile := e.Params.String("profile", "round")

	return func(u, w float64, in instance) float64 {
		var d float64
		if name == "circle" {
			// An ellipse, so aspect stretches circles too.
			d = (math.Hypot(u, w) - 1) * min(in.hw, in.hh)
		} else {
			d = shape(u*in.hw, w*in.hh, in.hw, in.hh, min(in.hw, in.hh)*0.3)
		}
		return bevelProfile(profile, -d/bevel)
	}, nil
}
