package nodes

import (
	"math"
	"strings"
	"unicode"

	"texxer/internal/graph"
	"texxer/internal/img"
)

var textDef = Def{
	Eval:     textNode,
	Category: "shape",
	Outputs:  []string{"out", "sdf"},
	Params: []Param{
		text("text", "01"),
		px("x", 16, 0, 512),
		px("y", 16, 0, 512),
		px("size", 3, 0.5, 32),
		px("spacing", 1, 0, 8),
		px("bevel", 1, 0.5, 16),
		enum("profile", "flat", "round", "linear", "flat"),
	},
}

// textNode stamps text in a 5x7 pixel font, for panel labels and numbers.
// size is the length of one font pixel; x, y is the top left corner.
func textNode(e *graph.Env) (*img.Buffer, error) {
	size := max(e.Px("size", 3), 0.5)
	gap := e.Px("spacing", 1)
	ox, oy := e.Px("x", 16), e.Px("y", 16)

	type cell struct{ x, y float64 }
	var cells []cell

	cursor := ox
	for _, r := range strings.ToUpper(e.Params.String("text", "01")) {
		glyph, ok := font5x7[unicode.ToUpper(r)]
		if !ok {
			glyph = font5x7['#']
		}

		for row, line := range glyph {
			for col, c := range line {
				if c == '#' {
					cells = append(cells, cell{cursor + float64(col)*size, oy + float64(row)*size})
				}
			}
		}
		cursor += 5*size + gap*size
	}

	half := size / 2
	reach := 2 * size

	sdf := fill(e.Ctx, func(x, y int) float32 {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		d := reach
		for _, c := range cells {
			if math.Abs(fx-c.x-half) > reach+half || math.Abs(fy-c.y-half) > reach+half {
				continue
			}
			d = min(d, sdBox(fx-c.x-half, fy-c.y-half, half, half, 0))
		}
		return float32(d)
	})

	e.SetOutput("sdf", sdf)
	return heightFromSDF(e, sdf, 1, "flat"), nil
}
