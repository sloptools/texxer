package nodes

import (
	"fmt"
	"math"
	"slices"

	"texxer/internal/graph"
	"texxer/internal/img"
)

var greebleKinds = []string{"block", "recess", "slots", "pipe", "bolts"}

var greebleDef = Def{
	Eval:     greebleNode,
	Category: "shape",
	Outputs:  []string{"out", "id", "cell"},
	Params: []Param{
		px("grid", 4, 1, 64),
		px("min", 8, 2, 256),
		px("max", 64, 4, 512),
		integer("depth", 8, 1, 16),
		num("stop", 0.25, 0, 1),
		data("kinds", greebleKinds),
		px("gap", 1.5, 0, 16),
		px("bevel", 1.5, 0.5, 16),
		num("height_var", 0.5, 0, 1),
		seedParam,
	},
}

// greebleNode packs random machine parts into the buffer: blocks, recesses,
// slot vents, pipes and bolts on cells of a random subdivision, like the
// busy strips of tech trimsheets. It tiles because cells fill the buffer.
func greebleNode(e *graph.Env) (*img.Buffer, error) {
	p := e.Params
	grid := p.Float("grid", 4)
	unitsX := max(1, int(math.Round(float64(e.Ctx.W)/e.Ctx.Scale/grid)))
	unitsY := max(1, int(math.Round(float64(e.Ctx.H)/e.Ctx.Scale/grid)))

	l := panelLayout{
		min:   max(1, int(math.Round(p.Float("min", 8)/grid))),
		max:   max(1, int(math.Round(p.Float("max", 64)/grid))),
		depth: p.Int("depth", 8),
		stop:  p.Float("stop", 0.25),
		seed:  e.Seed(),
	}
	l.split(panelCell{0, 0, unitsX, unitsY, 0}, 0)

	kinds := greebleKinds
	if err := p.Decode("kinds", &kinds); err != nil {
		return nil, err
	}
	for _, k := range kinds {
		if !slices.Contains(greebleKinds, k) {
			return nil, fmt.Errorf("unknown kind %q", k)
		}
	}
	if len(kinds) == 0 {
		kinds = greebleKinds
	}

	// Lookup from grid unit to cell, so pixels don't search all cells.
	lookup := make([]int, unitsX*unitsY)
	for i, c := range l.cells {
		for gy := c.y0; gy < c.y1; gy++ {
			for gx := c.x0; gx < c.x1; gx++ {
				lookup[gy*unitsX+gx] = i
			}
		}
	}

	gap := e.Px("gap", 1.5)
	bevel := max(e.Px("bevel", 1.5), 0.5)
	heightVar := p.Float("height_var", 0.5)
	ux := float64(e.Ctx.W) / float64(unitsX)
	uy := float64(e.Ctx.H) / float64(unitsY)

	out := img.New(e.Ctx.W, e.Ctx.H, 1, e.Ctx.Tiling)
	ids := img.New(e.Ctx.W, e.Ctx.H, 1, e.Ctx.Tiling)
	cells := img.New(e.Ctx.W, e.Ctx.H, 1, e.Ctx.Tiling)

	img.Rows(e.Ctx.H, func(y int) {
		for x := range e.Ctx.W {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			gx := min(int(fx/ux), unitsX-1)
			gy := min(int(fy/uy), unitsY-1)
			c := l.cells[lookup[gy*unitsX+gx]]

			hw := float64(c.x1-c.x0)*ux/2 - gap
			hh := float64(c.y1-c.y0)*uy/2 - gap
			lx := fx - float64(c.x0+c.x1)*ux/2
			ly := fy - float64(c.y0+c.y1)*uy/2

			kind := kinds[int(l.rand(c.id, 10)*float64(len(kinds)))]
			level := 0.65 - heightVar*0.4*l.rand(c.id, 11)

			d := sdBox(lx, ly, hw, hh, 0)
			h := level * bevelProfile("linear", -d/bevel)
			h += greebleDetail(kind, &l, c.id, lx, ly, hw, hh, bevel) * bevelProfile("flat", -d)

			i := y*e.Ctx.W + x
			out.Pix[i] = float32(clamp01(h))
			ids.Pix[i] = float32(l.rand(c.id, 3))
			cells.Pix[i] = float32(d)
		}
	})

	e.SetOutput("id", ids)
	e.SetOutput("cell", cells)
	return out, nil
}

// greebleDetail is the height a part adds on top of its cell's plate, at
// local coordinates around the cell center.
func greebleDetail(kind string, l *panelLayout, id int, x, y, hw, hh, bevel float64) float64 {
	r := func(k int) float64 { return l.rand(id, k) }
	horizontal := hw >= hh
	short := min(hw, hh)

	switch kind {
	case "block", "recess":
		inset := 1 + r(12)*short*0.5
		d := sdChamfer(x, y, hw-inset, hh-inset, min(hw, hh)*0.3*r(13))
		v := 0.25 * bevelProfile("linear", -d/bevel)
		if kind == "recess" {
			return -v
		}
		return v

	case "slots":
		inset := 1 + short*0.3
		if sdBox(x, y, hw-inset, hh-inset, 0) > 0 {
			return 0
		}
		period := 3 + math.Floor(r(14)*3)
		along := x
		if !horizontal {
			along = y
		}
		f := math.Mod(math.Abs(along), period) / period
		if f < 0.45 {
			return -0.3
		}
		return 0

	case "pipe":
		across := y / hh
		if !horizontal {
			across = x / hw
		}
		return 0.3 * math.Sqrt(max(0, 1-across*across))

	case "bolts":
		radius := max(1, short*0.35)
		along := hw - radius - 1
		cx, cy := along, 0.0
		if !horizontal {
			cx, cy = 0, hh-radius-1
		}
		d := min(math.Hypot(x-cx, y-cy), math.Hypot(x+cx, y+cy)) - radius
		return 0.3 * bevelProfile("round", -d/max(radius*0.6, 0.5))
	}
	return 0
}
