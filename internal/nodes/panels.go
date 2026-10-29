package nodes

import (
	"fmt"
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
	"texxer/internal/noise"
)

// panelCell is a rectangle in grid units.
type panelCell struct {
	x0, y0, x1, y1 int
	id             int
}

type panelLayout struct {
	min, max, depth int
	stop            float64
	seed            int
	cells           []panelCell
}

func (l *panelLayout) rand(id, k int) float64 {
	return noise.Rand01(id, k, l.seed)
}

// split recursively divides c on the grid until cells are small enough or
// randomly stop. Cells always tile the whole area, so the layout is seamless.
func (l *panelLayout) split(c panelCell, depth int) {
	w, h := c.x1-c.x0, c.y1-c.y0
	canX, canY := w >= 2*l.min, h >= 2*l.min
	big := w > l.max || h > l.max

	if depth >= l.depth || (!canX && !canY) || (!big && l.rand(c.id, 0) < l.stop) {
		l.cells = append(l.cells, c)
		return
	}

	vertical := canX
	if canX && canY {
		vertical = l.rand(c.id, 1) < float64(w)/float64(w+h)
	}

	span := h
	if vertical {
		span = w
	}
	at := l.min + int(l.rand(c.id, 2)*float64(span-2*l.min+1))

	a, b := c, c
	a.id, b.id = c.id*2+1, c.id*2+2
	if vertical {
		a.x1, b.x0 = c.x0+at, c.x0+at
	} else {
		a.y1, b.y0 = c.y0+at, c.y0+at
	}

	l.split(a, depth+1)
	l.split(b, depth+1)
}

// panelsNode subdivides the area into grid-aligned panels. output selects
// "shape" (SDF of a shape fitted into each panel), "cell" (SDF of the panel
// rectangles) or "id" (random value per panel).
func panelsNode(e *graph.Env) (*img.Buffer, error) {
	p := e.Params
	grid := p.Float("grid", 16)
	unitsX := max(1, int(math.Round(float64(e.Ctx.W)/e.Ctx.Scale/grid)))
	unitsY := max(1, int(math.Round(float64(e.Ctx.H)/e.Ctx.Scale/grid)))

	l := panelLayout{
		min:   max(1, int(math.Round(p.Float("min", 32)/grid))),
		max:   max(1, int(math.Round(p.Float("max", 160)/grid))),
		depth: p.Int("depth", 6),
		stop:  p.Float("stop", 0.3),
		seed:  p.Int("seed", 0),
	}
	l.split(panelCell{0, 0, unitsX, unitsY, 0}, 0)

	output := p.String("output", "shape")
	if output != "shape" && output != "cell" && output != "id" {
		return nil, fmt.Errorf("unknown panels output %q", output)
	}

	var shapeNames []string
	if err := p.Decode("shapes", &shapeNames); err != nil {
		return nil, err
	}
	if len(shapeNames) == 0 {
		shapeNames = []string{"chamfer"}
	}

	fit := panelShape{
		inset:   e.Px("inset", 6),
		chamfer: e.Px("chamfer", 6),
		taper:   p.Float("taper", 0.25),
		names:   shapeNames,
	}
	ux := float64(e.Ctx.W) / float64(unitsX)
	uy := float64(e.Ctx.H) / float64(unitsY)

	return fill(e.Ctx, func(x, y int) float32 {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		c := l.cells[l.find(fx/ux, fy/uy)]

		hw := float64(c.x1-c.x0) * ux / 2
		hh := float64(c.y1-c.y0) * uy / 2
		lx := fx - float64(c.x0)*ux - hw
		ly := fy - float64(c.y0)*uy - hh

		switch output {
		case "id":
			return float32(l.rand(c.id, 3))
		case "cell":
			return float32(sdBox(lx, ly, hw, hh, 0))
		default:
			return float32(fit.dist(&l, c, lx, ly, hw, hh))
		}
	}), nil
}

func (l *panelLayout) find(gx, gy float64) int {
	for i, c := range l.cells {
		if gx >= float64(c.x0) && gx < float64(c.x1) && gy >= float64(c.y0) && gy < float64(c.y1) {
			return i
		}
	}
	return 0
}

type panelShape struct {
	inset, chamfer, taper float64
	names                 []string
}

func (s panelShape) dist(l *panelLayout, c panelCell, x, y, hw, hh float64) float64 {
	hw, hh = max(hw-s.inset, 1), max(hh-s.inset, 1)
	name := s.names[int(l.rand(c.id, 4)*float64(len(s.names)))]

	switch name {
	case "trapezoid":
		top, bottom := hw*(1-s.taper), hw
		if l.rand(c.id, 5) < 0.5 {
			top, bottom = bottom, top
		}
		return max(sdTrapezoid(x, y, top, bottom, hh), chamferCut(x, y, hw, hh, s.chamfer))
	case "box":
		return sdBox(x, y, hw, hh, 0)
	default:
		return sdChamfer(x, y, hw, hh, s.chamfer)
	}
}
