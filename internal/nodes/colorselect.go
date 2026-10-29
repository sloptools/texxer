package nodes

import (
	"fmt"
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
)

var colorSelectDef = Def{
	Eval:     colorSelectNode,
	Category: "mask",
	Inputs:   []string{"in"},
	Params: []Param{
		color("color", "#c87030"),
		enum("mode", "hue", "hue", "rgb"),
		num("tolerance", 0.08, 0, 1),
		num("softness", 0.05, 0, 1),
		num("min_saturation", 0.25, 0, 1),
	},
}

var shadeDef = Def{
	Eval:     shadeNode,
	Category: "mesh",
	Inputs:   []string{"normal"},
	Params: []Param{
		num("azimuth", 45, -180, 180),
		num("elevation", 50, -90, 90),
		num("power", 1, 0.1, 128),
		flag("wrap", false),
	},
}

// colorSelectNode masks the pixels close to a color, to pick materials out
// of an existing texture. hue mode compares hue only, so light and dark
// shades of a paint are found together; grays are skipped below min_saturation.
func colorSelectNode(e *graph.Env) (*img.Buffer, error) {
	src, err := e.Required("in")
	if err != nil {
		return nil, err
	}

	target, err := parseHex(e.Params.String("color", "#c87030"))
	if err != nil {
		return nil, err
	}
	mode := e.Params.String("mode", "hue")
	if mode != "hue" && mode != "rgb" {
		return nil, fmt.Errorf("unknown mode %q", mode)
	}

	tol := e.Params.Float("tolerance", 0.08)
	soft := max(e.Params.Float("softness", 0.05), 1e-4)
	minSat := e.Params.Float("min_saturation", 0.25)
	th, _, _ := toHSL(float64(target[0]), float64(target[1]), float64(target[2]))

	out := img.New(src.W, src.H, 1, src.Tiling)
	for i := range out.Pix {
		r, g, b := float64(src.Value(i, 0)), float64(src.Value(i, 1)), float64(src.Value(i, 2))

		var d float64
		if mode == "rgb" {
			d = math.Sqrt(((r-float64(target[0]))*(r-float64(target[0])) + (g-float64(target[1]))*(g-float64(target[1])) + (b-float64(target[2]))*(b-float64(target[2]))) / 3)
		} else {
			h, s, _ := toHSL(r, g, b)
			if s < minSat {
				continue
			}
			d = math.Abs(h - th)
			d = min(d, 1-d)
		}
		out.Pix[i] = float32(1 - smoothstep(tol, tol+soft, d))
	}
	return out, nil
}

// shadeNode lights a baked world normal map from a direction: 1 facing the
// light, 0 facing away. A high power narrows it to a glint, which fakes
// reflections on glass; wrap lights half the back too, for softer falloff.
func shadeNode(e *graph.Env) (*img.Buffer, error) {
	n, err := e.Required("normal")
	if err != nil {
		return nil, err
	}

	az := e.Params.Float("azimuth", 45) * math.Pi / 180
	el := e.Params.Float("elevation", 50) * math.Pi / 180
	power := e.Params.Float("power", 1)
	wrap := e.Params.Bool("wrap", false)
	l := [3]float64{math.Cos(el) * math.Cos(az), math.Sin(el), math.Cos(el) * math.Sin(az)}

	out := img.New(n.W, n.H, 1, n.Tiling)
	for i := range out.Pix {
		d := 0.0
		for k := range 3 {
			d += (float64(n.Value(i, k))*2 - 1) * l[k]
		}
		if wrap {
			d = d*0.5 + 0.5
		}
		out.Pix[i] = float32(math.Pow(clamp01(d), power))
	}
	return out, nil
}
