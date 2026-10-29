package nodes

import (
	"fmt"
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
)

var aoDef = Def{
	Eval:     aoNode,
	Category: "light",
	Inputs:   []string{"height"},
	Params: []Param{
		px("radius", 12, 1, 64),
		num("strength", 4, 0, 32),
	},
}

var maskDef = Def{
	Eval:     maskNode,
	Category: "mask",
	Inputs:   []string{"height", "breakup", "ao", "curvature"},
	Params: []Param{
		enum("mode", "edges", "edges", "cavity", "height", "top", "bottom"),
		px("radius", 2, 1, 32),
		num("gain", 8, 0, 64),
		num("level", 0.3, 0, 1),
		num("softness", 0.15, 0, 1),
		num("breakup", 0.5, 0, 2),
		flag("invert", false),
	},
}

// aoNode approximates ambient occlusion from height: how far a pixel sits
// below its surroundings at a few scales up to radius. 1 is open, 0 occluded.
func aoNode(e *graph.Env) (*img.Buffer, error) {
	h, err := e.Required("height")
	if err != nil {
		return nil, err
	}

	radius := e.Px("radius", 12)
	strength := float32(e.Params.Float("strength", 4))
	h = gray(h)

	out := img.New(h.W, h.H, 1, h.Tiling)
	for i := range out.Pix {
		out.Pix[i] = 1
	}

	for _, f := range []float64{0.25, 0.5, 1} {
		r := max(1, int(math.Round(radius*f)))
		avg := boxPass(boxPass(h, r, 1, 0), r, 0, 1)
		for i, v := range h.Pix {
			out.Pix[i] -= max(0, avg.Pix[i]-v) * strength / 3
		}
	}

	for i, v := range out.Pix {
		out.Pix[i] = img.Clamp01(v)
	}
	return out, nil
}

// maskNode is a smart mask: it finds edges, cavities, heights or faces
// pointing up or down, then thresholds them with an optional breakup noise
// so grunge follows the shape. Baked ao and curvature maps replace the
// estimates from height when connected.
func maskNode(e *graph.Env) (*img.Buffer, error) {
	src, err := maskSource(e)
	if err != nil {
		return nil, err
	}

	breakup, err := e.Input("breakup")
	if err != nil {
		return nil, err
	}

	p := e.Params
	level := p.Float("level", 0.3)
	soft := max(p.Float("softness", 0.15), 1e-3)
	amount := p.Float("breakup", 0.5)
	invert := p.Bool("invert", false)

	out := img.New(src.W, src.H, 1, src.Tiling)
	for i, v := range src.Pix {
		t := float64(v)
		// Scaling keeps flat areas empty, the noise only eats into the mask.
		if breakup != nil {
			t *= 1 + (float64(breakup.Pix[i*breakup.C])*2-1)*amount
		}

		m := smoothstep(level-soft, level+soft, t)
		if invert {
			m = 1 - m
		}
		out.Pix[i] = float32(m)
	}
	return out, nil
}

// maskSource is the unthresholded 0..1 measure the mask mode selects.
func maskSource(e *graph.Env) (*img.Buffer, error) {
	mode := e.Params.String("mode", "edges")
	gain := float32(e.Params.Float("gain", 8))
	r := max(1, int(math.Round(e.Px("radius", 2))))

	baked := map[string]string{"edges": "curvature", "cavity": "ao"}
	if name, ok := baked[mode]; ok {
		b, err := e.Input(name)
		if err != nil {
			return nil, err
		}
		if b != nil {
			return mapGray(gray(b), func(v float32) float32 {
				if mode == "cavity" {
					return 1 - v
				}
				return img.Clamp01((v - 0.5) * 2)
			}), nil
		}
	}

	h, err := e.Required("height")
	if err != nil {
		return nil, err
	}
	h = gray(h)

	switch mode {
	case "height":
		return h, nil

	case "edges", "cavity":
		avg := boxPass(boxPass(h, r, 1, 0), r, 0, 1)
		out := img.New(h.W, h.H, 1, h.Tiling)
		for i, v := range h.Pix {
			d := v - avg.Pix[i]
			if mode == "cavity" {
				d = -d
			}
			out.Pix[i] = img.Clamp01(d * gain)
		}
		return out, nil

	case "top", "bottom":
		dir := float32(1)
		if mode == "bottom" {
			dir = -1
		}
		// Image y points down, so a surface facing up rises towards smaller y.
		return fill(e.Ctx, func(x, y int) float32 {
			dy := (h.At(x, y+r, 0) - h.At(x, y-r, 0)) / float32(2*r)
			return img.Clamp01(dy * dir * gain * float32(r))
		}), nil
	}
	return nil, fmt.Errorf("unknown mode %q", mode)
}

// gray returns the first channel of b as a one-channel buffer.
func gray(b *img.Buffer) *img.Buffer {
	if b.C == 1 {
		return b
	}

	out := img.New(b.W, b.H, 1, b.Tiling)
	for i := range out.Pix {
		out.Pix[i] = b.Pix[i*b.C]
	}
	return out
}

func mapGray(b *img.Buffer, fn func(float32) float32) *img.Buffer {
	out := img.New(b.W, b.H, 1, b.Tiling)
	for i, v := range b.Pix {
		out.Pix[i] = fn(v)
	}
	return out
}
