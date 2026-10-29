package nodes

import (
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
)

// blurNode approximates a gaussian with repeated box blurs.
func blurNode(e *graph.Env) (*img.Buffer, error) {
	src, err := e.Required("in")
	if err != nil {
		return nil, err
	}

	r := int(math.Round(e.Px("radius", 2)))
	if r < 1 {
		return src, nil
	}

	b := src
	for range 3 {
		b = boxPass(b, r, 1, 0)
		b = boxPass(b, r, 0, 1)
	}
	return b, nil
}

func boxPass(src *img.Buffer, r, dx, dy int) *img.Buffer {
	out := img.New(src.W, src.H, src.C, src.Tiling)
	n := float32(2*r + 1)

	img.Rows(src.H, func(y int) {
		for x := range src.W {
			for ch := range src.C {
				var sum float32
				for k := -r; k <= r; k++ {
					sum += src.At(x+k*dx, y+k*dy, ch)
				}
				out.Pix[(y*src.W+x)*src.C+ch] = sum / n
			}
		}
	})
	return out
}
