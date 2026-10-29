package nodes

import (
	"fmt"

	"texxer/internal/graph"
	"texxer/internal/img"
	"texxer/internal/palette"
)

var bayer4 = [16]float32{0, 8, 2, 10, 12, 4, 14, 6, 3, 11, 1, 9, 15, 7, 13, 5}

var quantizeDef = Def{
	Eval:     quantizeNode,
	Category: "output",
	Inputs:   []string{"in"},
	Params: []Param{
		text("palette", ""),
		integer("colors", 64, 2, 256),
		enum("dither", "none", "none", "bayer"),
		num("dither_strength", 0.04, 0, 0.5),
	},
}

// quantizeNode reduces colors to a palette loaded from an image, or an
// adaptive palette with the given number of colors.
func quantizeNode(e *graph.Env) (*img.Buffer, error) {
	src, err := e.Required("in")
	if err != nil {
		return nil, err
	}

	pal, err := loadPalette(e, src)
	if err != nil {
		return nil, err
	}

	dither := e.Params.String("dither", "none")
	if dither != "none" && dither != "bayer" {
		return nil, fmt.Errorf("unknown dither %q", dither)
	}
	strength := float32(e.Params.Float("dither_strength", 0.04))

	out := img.New(src.W, src.H, 4, src.Tiling)
	img.Rows(src.H, func(y int) {
		for x := range src.W {
			i := y*src.W + x

			var offset float32
			if dither == "bayer" {
				offset = (bayer4[(y%4)*4+x%4]/16 - 0.5) * strength
			}

			c := palette.Nearest(pal, palette.Color{
				src.Value(i, 0) + offset,
				src.Value(i, 1) + offset,
				src.Value(i, 2) + offset,
			})
			copy(out.Pix[i*4:], []float32{c[0], c[1], c[2], src.Value(i, 3)})
		}
	})
	return out, nil
}

func loadPalette(e *graph.Env, src *img.Buffer) ([]palette.Color, error) {
	if path := e.Params.String("palette", ""); path != "" {
		return palette.Load(e.Path(path))
	}

	pix := make([]palette.Color, src.W*src.H)
	for i := range pix {
		pix[i] = palette.Color{src.Value(i, 0), src.Value(i, 1), src.Value(i, 2)}
	}
	return palette.MedianCut(pix, e.Params.Int("colors", 64)), nil
}
