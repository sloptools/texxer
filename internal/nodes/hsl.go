package nodes

import (
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
)

var hslDef = Def{
	Eval:     hslNode,
	Category: "color",
	Inputs:   []string{"in", "mask"},
	Params: []Param{
		num("hue", 0, -180, 180),
		num("saturation", 0, -1, 1),
		num("lightness", 0, -1, 1),
	},
}

// hslNode shifts hue, scales saturation and offsets lightness, limited to
// the mask when one is connected.
func hslNode(e *graph.Env) (*img.Buffer, error) {
	src, err := e.Required("in")
	if err != nil {
		return nil, err
	}
	mask, err := e.Input("mask")
	if err != nil {
		return nil, err
	}

	hue := e.Params.Float("hue", 0) / 360
	sat := 1 + e.Params.Float("saturation", 0)
	light := e.Params.Float("lightness", 0)

	out := img.New(src.W, src.H, 4, src.Tiling)
	img.Rows(src.H, func(y int) {
		for x := range src.W {
			i := y*src.W + x
			r, g, b := float64(src.Value(i, 0)), float64(src.Value(i, 1)), float64(src.Value(i, 2))

			h, s, l := toHSL(r, g, b)
			h = math.Mod(h+hue+1, 1)
			s = clamp01(s * sat)
			l = clamp01(l + light)
			nr, ng, nb := fromHSL(h, s, l)

			m := 1.0
			if mask != nil {
				m = float64(mask.Value(i, 0))
			}
			out.Pix[i*4] = float32(r + (nr-r)*m)
			out.Pix[i*4+1] = float32(g + (ng-g)*m)
			out.Pix[i*4+2] = float32(b + (nb-b)*m)
			out.Pix[i*4+3] = src.Value(i, 3)
		}
	})
	return out, nil
}

func toHSL(r, g, b float64) (h, s, l float64) {
	hi, lo := max(r, g, b), min(r, g, b)
	l = (hi + lo) / 2
	if hi == lo {
		return 0, 0, l
	}

	d := hi - lo
	if l > 0.5 {
		s = d / (2 - hi - lo)
	} else {
		s = d / (hi + lo)
	}

	switch hi {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	return h / 6, s, l
}

func fromHSL(h, s, l float64) (r, g, b float64) {
	if s == 0 {
		return l, l, l
	}

	q := l * (1 + s)
	if l >= 0.5 {
		q = l + s - l*s
	}
	p := 2*l - q
	return hueToRGB(p, q, h+1.0/3), hueToRGB(p, q, h), hueToRGB(p, q, h-1.0/3)
}

func hueToRGB(p, q, t float64) float64 {
	t = math.Mod(t+1, 1)
	switch {
	case t < 1.0/6:
		return p + (q-p)*6*t
	case t < 0.5:
		return q
	case t < 2.0/3:
		return p + (q-p)*(2.0/3-t)*6
	}
	return p
}
