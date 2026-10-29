package nodes

import (
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
)

var blurDef = Def{
	Eval:     blurNode,
	Category: "filter",
	Inputs:   []string{"in"},
	Params:   []Param{px("radius", 2, 0, 64)},
}

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

// boxPass blurs along rows (dx=1) or columns (dy=1) with a running sum, so
// the cost doesn't grow with the radius.
func boxPass(src *img.Buffer, r, dx, dy int) *img.Buffer {
	out := img.New(src.W, src.H, src.C, src.Tiling)
	n := float32(2*r + 1)

	lines, length := src.H, src.W
	if dy == 1 {
		lines, length = src.W, src.H
	}

	img.Rows(lines, func(line int) {
		// k runs along the line; x, y follow from the direction.
		pos := func(k int) (int, int) {
			if dx == 1 {
				return k, line
			}
			return line, k
		}

		for ch := range src.C {
			var sum float32
			for k := -r; k <= r; k++ {
				x, y := pos(k)
				sum += src.At(x, y, ch)
			}

			for k := range length {
				x, y := pos(k)
				out.Pix[(y*src.W+x)*src.C+ch] = sum / n

				ax, ay := pos(k + r + 1)
				bx, by := pos(k - r)
				sum += src.At(ax, ay, ch) - src.At(bx, by, ch)
			}
		}
	})
	return out
}
