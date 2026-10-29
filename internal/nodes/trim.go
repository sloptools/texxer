package nodes

import (
	"errors"
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
)

type strip struct {
	Height float64 `json:"height"`
	Input  string  `json:"input"`
}

// trimNode stacks horizontal strips into a trimsheet height map. Each strip
// input is rendered at strip size with horizontal-only tiling. Raised rails
// separate the strips; the last strip fills the remaining height.
func trimNode(e *graph.Env) (*img.Buffer, error) {
	var strips []strip
	if err := e.Params.Decode("strips", &strips); err != nil {
		return nil, err
	}
	if len(strips) == 0 {
		return nil, errors.New("no strips defined")
	}

	depth := float32(e.Params.Float("depth", 0.6))
	w, h := e.Ctx.W, e.Ctx.H
	out := img.New(w, h, 1, img.TileX)

	var bounds []int
	y0 := 0

	for i, s := range strips {
		sh := int(math.Round(s.Height * e.Ctx.Scale))
		if i == len(strips)-1 || y0+sh > h {
			sh = h - y0
		}
		if sh <= 0 {
			break
		}

		ctx := graph.Ctx{W: w, H: sh, Tiling: img.TileX, Scale: e.Ctx.Scale}
		content, err := e.InputAt(s.Input, ctx)
		if err != nil {
			return nil, err
		}

		for y := range sh {
			for x := range w {
				v := depth / 2
				if content != nil {
					v = content.Value(y*w+x, 0) * depth
				}
				out.Pix[(y0+y)*w+x] = v
			}
		}

		bounds = append(bounds, y0)
		y0 += sh
	}

	drawRails(out, bounds, e.Px("rail", 6), e.Px("rail_bevel", 2))
	return out, nil
}

// drawRails draws flat-topped bars centered on each strip boundary.
func drawRails(b *img.Buffer, bounds []int, width, bevel float64) {
	half := width / 2

	for y := range b.H {
		fy := float64(y) + 0.5

		var v float64
		for _, by := range bounds {
			d := math.Abs(fy - float64(by))
			d = min(d, float64(b.H)-d)
			v = max(v, clamp01((half-d)/bevel))
		}

		if v == 0 {
			continue
		}
		for x := range b.W {
			i := y*b.W + x
			b.Pix[i] = max(b.Pix[i], float32(v))
		}
	}
}
