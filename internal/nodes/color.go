package nodes

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"texxer/internal/graph"
	"texxer/internal/img"
)

type stop struct {
	Pos   float64 `json:"pos"`
	Color string  `json:"color"`
	rgb   [3]float32
}

// gradientNode maps a gray input to colors.
func gradientNode(e *graph.Env) (*img.Buffer, error) {
	src, err := e.Required("in")
	if err != nil {
		return nil, err
	}

	var stops []stop
	if err := e.Params.Decode("stops", &stops); err != nil {
		return nil, err
	}
	if len(stops) == 0 {
		return nil, fmt.Errorf("no gradient stops")
	}

	for i := range stops {
		if stops[i].rgb, err = parseHex(stops[i].Color); err != nil {
			return nil, err
		}
	}
	slices.SortFunc(stops, func(a, b stop) int { return cmp.Compare(a.Pos, b.Pos) })

	out := img.New(src.W, src.H, 4, src.Tiling)
	img.Rows(src.H, func(y int) {
		for x := range src.W {
			i := y*src.W + x
			c := sample(stops, float64(src.Value(i, 0)))
			copy(out.Pix[i*4:], []float32{c[0], c[1], c[2], 1})
		}
	})
	return out, nil
}

func sample(stops []stop, t float64) [3]float32 {
	if t <= stops[0].Pos {
		return stops[0].rgb
	}

	for i := 1; i < len(stops); i++ {
		a, b := stops[i-1], stops[i]
		if t <= b.Pos {
			f := float32((t - a.Pos) / max(b.Pos-a.Pos, 1e-6))
			var c [3]float32
			for ch := range 3 {
				c[ch] = a.rgb[ch] + (b.rgb[ch]-a.rgb[ch])*f
			}
			return c
		}
	}
	return stops[len(stops)-1].rgb
}

func colorNode(e *graph.Env) (*img.Buffer, error) {
	c, err := parseHex(e.Params.String("color", "#808080"))
	if err != nil {
		return nil, err
	}

	out := img.New(e.Ctx.W, e.Ctx.H, 4, e.Ctx.Tiling)
	for i := 0; i < len(out.Pix); i += 4 {
		copy(out.Pix[i:], []float32{c[0], c[1], c[2], 1})
	}
	return out, nil
}

func parseHex(s string) ([3]float32, error) {
	s = strings.TrimPrefix(s, "#")
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil || len(s) != 6 {
		return [3]float32{}, fmt.Errorf("invalid color %q", s)
	}

	return [3]float32{
		float32(v>>16&0xff) / 255,
		float32(v>>8&0xff) / 255,
		float32(v&0xff) / 255,
	}, nil
}

func levelsNode(e *graph.Env) (*img.Buffer, error) {
	src, err := e.Required("in")
	if err != nil {
		return nil, err
	}

	p := e.Params
	inLo, inHi := p.Float("in_low", 0), p.Float("in_high", 1)
	outLo, outHi := p.Float("out_low", 0), p.Float("out_high", 1)
	gamma := p.Float("gamma", 1)
	invert := p.Bool("invert", false)

	out := img.New(src.W, src.H, src.C, src.Tiling)
	img.Rows(src.H, func(y int) {
		for x := range src.W {
			for ch := range src.C {
				i := (y*src.W+x)*src.C + ch
				v := float64(src.Pix[i])

				if ch < 3 {
					v = clamp01((v - inLo) / max(inHi-inLo, 1e-6))
					v = math.Pow(v, 1/gamma)
					if invert {
						v = 1 - v
					}
					v = outLo + v*(outHi-outLo)
				}
				out.Pix[i] = float32(v)
			}
		}
	})
	return out, nil
}
