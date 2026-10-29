package nodes

import (
	"fmt"
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
	"texxer/internal/noise"
)

var shapeDef = Def{
	Eval:     shapeNode,
	Category: "shape",
	Outputs:  []string{"out", "sdf"},
	Params: []Param{
		enum("shape", "box", "box", "chamfer", "trapezoid", "circle", "diamond", "arch", "gothic"),
		px("width", 64, 1, 512),
		px("height", 64, 1, 512),
		px("x", 0, -256, 256),
		px("y", 0, -256, 256),
		px("radius", 0, 0, 128),
		num("taper", 0.25, -0.9, 0.9),
		integer("count_x", 1, 1, 64),
		integer("count_y", 1, 1, 64),
		px("bevel", 2, 0.5, 64),
		enum("profile", "round", "round", "linear", "flat"),
	},
}

var gridDef = Def{
	Eval:     gridNode,
	Category: "shape",
	Outputs:  []string{"out", "sdf", "id"},
	Params: []Param{
		data("columns", []float64{1, 1}),
		data("rows", []float64{1, 1}),
		px("gap", 4, 0, 64),
		px("radius", 0, 0, 64),
		px("bevel", 2, 0.5, 64),
		enum("profile", "round", "round", "linear", "flat"),
		seedParam,
	},
}

// shapeNode draws one shape per cell of a count_x by count_y grid, centered
// in the cell and moved by x, y. The sdf output is for combining with sdf nodes.
func shapeNode(e *graph.Env) (*img.Buffer, error) {
	p := e.Params
	name := p.String("shape", "box")

	hw := e.Px("width", 64) / 2
	hh := e.Px("height", 64) / 2
	ox, oy := e.Px("x", 0), e.Px("y", 0)
	radius := e.Px("radius", 0)
	taper := p.Float("taper", 0.25)

	var dist func(x, y float64) float64
	switch name {
	case "trapezoid":
		top, bottom := hw*(1-max(taper, 0)), hw*(1+min(taper, 0))
		dist = func(x, y float64) float64 { return sdTrapezoid(x, y, top, bottom, hh) }
	default:
		shape, ok := shapes[name]
		if !ok {
			return nil, fmt.Errorf("unknown shape %q", name)
		}
		dist = func(x, y float64) float64 { return shape(x, y, hw, hh, radius) }
	}

	cw := float64(e.Ctx.W) / float64(max(1, p.Int("count_x", 1)))
	ch := float64(e.Ctx.H) / float64(max(1, p.Int("count_y", 1)))

	sdf := fill(e.Ctx, func(x, y int) float32 {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		col, row := math.Floor(fx/cw), math.Floor(fy/ch)
		d := math.Inf(1)

		// Neighbor cells too, so moved or oversized shapes continue across cells.
		for dr := -1.0; dr <= 1; dr++ {
			for dc := -1.0; dc <= 1; dc++ {
				cx := (col+dc+0.5)*cw + ox
				cy := (row+dr+0.5)*ch + oy
				d = min(d, dist(fx-cx, fy-cy))
			}
		}
		return float32(d)
	})

	e.SetOutput("sdf", sdf)
	return heightFromSDF(e, sdf, 2, "round"), nil
}

// gridNode splits the buffer into columns and rows with relative sizes,
// like the panel grid of a wall. id is random per cell.
func gridNode(e *graph.Env) (*img.Buffer, error) {
	var cols, rows []float64
	if err := e.Params.Decode("columns", &cols); err != nil {
		return nil, err
	}
	if err := e.Params.Decode("rows", &rows); err != nil {
		return nil, err
	}
	if cols == nil {
		cols = []float64{1, 1}
	}
	if rows == nil {
		rows = []float64{1, 1}
	}

	xs, err := splits(cols, float64(e.Ctx.W))
	if err != nil {
		return nil, fmt.Errorf("columns: %w", err)
	}
	ys, err := splits(rows, float64(e.Ctx.H))
	if err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	gap := e.Px("gap", 4) / 2
	radius := e.Px("radius", 0)
	seed := e.Seed()

	ids := fill(e.Ctx, func(x, y int) float32 {
		c, r := span(xs, float64(x)+0.5), span(ys, float64(y)+0.5)
		return float32(noise.Rand01(c, r, seed))
	})

	sdf := fill(e.Ctx, func(x, y int) float32 {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		c, r := span(xs, fx), span(ys, fy)

		hw := (xs[c+1]-xs[c])/2 - gap
		hh := (ys[r+1]-ys[r])/2 - gap
		cx, cy := (xs[c]+xs[c+1])/2, (ys[r]+ys[r+1])/2
		return float32(sdBox(fx-cx, fy-cy, hw, hh, radius))
	})

	e.SetOutput("sdf", sdf)
	e.SetOutput("id", ids)
	return heightFromSDF(e, sdf, 2, "round"), nil
}

// splits turns relative sizes into boundaries from 0 to total.
func splits(sizes []float64, total float64) ([]float64, error) {
	var sum float64
	for _, s := range sizes {
		if s <= 0 {
			return nil, fmt.Errorf("sizes must be positive")
		}
		sum += s
	}
	if sum == 0 {
		return nil, fmt.Errorf("no sizes")
	}

	out := []float64{0}
	var at float64
	for _, s := range sizes {
		at += s / sum * total
		out = append(out, at)
	}
	return out, nil
}

// span finds the interval of bounds that holds v.
func span(bounds []float64, v float64) int {
	for i := 1; i < len(bounds)-1; i++ {
		if v < bounds[i] {
			return i - 1
		}
	}
	return len(bounds) - 2
}

// heightFromSDF applies the node's bevel and profile params to a distance field.
func heightFromSDF(e *graph.Env, sdf *img.Buffer, bevel float64, profile string) *img.Buffer {
	bevel = max(e.Px("bevel", bevel), 0.5)
	profile = e.Params.String("profile", profile)

	out := img.New(sdf.W, sdf.H, 1, sdf.Tiling)
	for i, d := range sdf.Pix {
		out.Pix[i] = float32(bevelProfile(profile, -float64(d)/bevel))
	}
	return out
}
