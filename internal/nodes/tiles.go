package nodes

import (
	"math"
	"sort"

	"texxer/internal/graph"
	"texxer/internal/img"
	"texxer/internal/noise"
)

var tilesDef = Def{
	Eval:     tilesNode,
	Category: "shape",
	Outputs:  []string{"out", "id", "sdf"},
	Params: []Param{
		px("width", 64, 2, 512),
		num("width_var", 0.5, 0, 1),
		px("row_h", 32, 2, 512),
		num("offset", 0.5, 0, 1),
		num("offset_var", 0, 0, 1),
		px("gap", 2, 0, 32),
		px("radius", 0, 0, 32),
		px("bevel", 2, 0.5, 32),
		enum("profile", "round", "round", "linear", "flat"),
		num("height_var", 0, 0, 1),
		num("slope", 0, 0, 1),
		seedParam,
	},
}

// tileRow holds the left edges of the tiles in one row; widths always add up
// to the buffer width so rows tile seamlessly.
type tileRow struct {
	edges  []float64
	offset float64
}

// tilesNode lays out rows of tiles with random widths and offsets, like
// shingles, planks or irregular bricks. Main output is height, "id" a random
// value per tile and "sdf" the tile distance field.
func tilesNode(e *graph.Env) (*img.Buffer, error) {
	p := e.Params
	seed := e.Seed()
	w, h := float64(e.Ctx.W), float64(e.Ctx.H)

	rows := max(1, int(math.Round(h/e.Px("row_h", 32))))
	rowH := h / float64(rows)
	perRow := max(1, int(math.Round(w/e.Px("width", 64))))
	meanW := w / float64(perRow)

	layout := make([]tileRow, rows)
	for r := range layout {
		layout[r] = makeRow(r, perRow, w, meanW, p, seed)
	}

	gap := e.Px("gap", 2)
	radius := e.Px("radius", 0)
	bevel := max(e.Px("bevel", 2), 0.5)
	profile := p.String("profile", "round")
	heightVar := p.Float("height_var", 0)
	slope := p.Float("slope", 0)

	height := img.New(e.Ctx.W, e.Ctx.H, 1, e.Ctx.Tiling)
	id := img.New(e.Ctx.W, e.Ctx.H, 1, e.Ctx.Tiling)
	sdf := img.New(e.Ctx.W, e.Ctx.H, 1, e.Ctx.Tiling)

	img.Rows(e.Ctx.H, func(y int) {
		fy := float64(y) + 0.5
		r := min(int(fy/rowH), rows-1)
		row := layout[r]
		ly := fy - (float64(r)+0.5)*rowH

		for x := range e.Ctx.W {
			fx := math.Mod(float64(x)+0.5-row.offset+w, w)
			t := sort.SearchFloat64s(row.edges, fx+1e-9) - 1
			t = max(t, 0)

			x0 := row.edges[t]
			x1 := w
			if t+1 < len(row.edges) {
				x1 = row.edges[t+1]
			}
			lx := fx - (x0+x1)/2

			d := sdBox(lx, ly, (x1-x0)/2-gap, rowH/2-gap, radius)
			rnd := noise.Rand01(r, t, seed)

			v := bevelProfile(profile, -d/bevel) * (1 - heightVar*noise.Rand01(r, t, seed+1))
			// Slope rises towards the bottom edge, so tiles look like they overlap the row below.
			v *= 1 - slope*(0.5-ly/rowH)

			i := y*e.Ctx.W + x
			height.Pix[i] = float32(v)
			id.Pix[i] = float32(rnd)
			sdf.Pix[i] = float32(d)
		}
	})

	e.SetOutput("id", id)
	e.SetOutput("sdf", sdf)
	return height, nil
}

func makeRow(r, n int, w, meanW float64, p graph.Params, seed int) tileRow {
	widthVar := p.Float("width_var", 0.5)

	weights := make([]float64, n)
	var sum float64
	for t := range weights {
		weights[t] = 1 + widthVar*(noise.Rand01(r, t, seed+2)-0.5)*1.6
		sum += weights[t]
	}

	row := tileRow{edges: make([]float64, n)}
	var x float64
	for t, wt := range weights {
		row.edges[t] = x
		x += wt / sum * w
	}

	shift := float64(r)*p.Float("offset", 0.5) + p.Float("offset_var", 0)*noise.Rand01(r, 0, seed+3)
	row.offset = math.Mod(shift, 1) * meanW
	return row
}
