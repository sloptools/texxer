package nodes

import (
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
	"texxer/internal/noise"
)

var strokesDef = Def{
	Eval:     strokesNode,
	Category: "grunge",
	Params: []Param{
		px("length", 48, 4, 512),
		px("width", 10, 1, 128),
		num("density", 1, 0.1, 4),
		num("angle", 0, -180, 180),
		num("spread", 20, 0, 180),
		num("taper", 0.6, 0, 1),
		num("bristles", 6, 0, 32),
		num("dryness", 0.3, 0, 1),
		seedParam,
	},
}

var scanDef = Def{
	Eval:     scanNode,
	Category: "filter",
	Inputs:   []string{"in"},
	Params: []Param{
		num("position", 0.5, 0, 1),
		num("contrast", 0.8, 0, 1),
		flag("invert", false),
	},
}

// strokesNode scatters brush strokes: tapered bands with bristle lines along
// them that break up towards the end when dryness is up. Each stroke has
// its own brightness, so the result also works as a height or color map.
func strokesNode(e *graph.Env) (*img.Buffer, error) {
	p := e.Params
	length := e.Px("length", 48)
	width := e.Px("width", 10)
	angle := p.Float("angle", 0) * math.Pi / 180
	spread := p.Float("spread", 20) * math.Pi / 180
	taper := p.Float("taper", 0.6)
	bristles := p.Float("bristles", 6)
	dryness := p.Float("dryness", 0.3)
	seed := e.Seed()

	// Cells about half a stroke wide; density puts more strokes in each.
	cellSize := max(length, width) / 2
	nx := max(1, int(math.Round(float64(e.Ctx.W)/cellSize)))
	ny := max(1, int(math.Round(float64(e.Ctx.H)/cellSize)))
	cw, ch := float64(e.Ctx.W)/float64(nx), float64(e.Ctx.H)/float64(ny)

	perCell := max(1, int(math.Ceil(p.Float("density", 1))))
	keep := p.Float("density", 1) / float64(perCell)
	reach := int(math.Ceil(max(length, width)/2/min(cw, ch))) + 1

	return fill(e.Ctx, func(x, y int) float32 {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		col, row := int(math.Floor(fx/cw)), int(math.Floor(fy/ch))
		best := 0.0

		for dr := -reach; dr <= reach; dr++ {
			for dc := -reach; dc <= reach; dc++ {
				c, r := col+dc, row+dr
				hc, hr := img.Mod(c, nx), img.Mod(r, ny)

				for k := range perCell {
					s := seed + k*7907
					if noise.Rand01(hc, hr, s+1) > keep {
						continue
					}

					cx := (float64(c) + noise.Rand01(hc, hr, s+2)) * cw
					cy := (float64(r) + noise.Rand01(hc, hr, s+3)) * ch
					a := angle + (noise.Rand01(hc, hr, s+4)*2-1)*spread
					l := length * (0.6 + 0.8*noise.Rand01(hc, hr, s+5)) / 2
					w := width * (0.7 + 0.6*noise.Rand01(hc, hr, s+6)) / 2

					// Stroke coordinates: u along, v across, both -1..1.
					dx, dy := fx-cx, fy-cy
					u := (dx*math.Cos(a) - dy*math.Sin(a)) / l
					v := (dx*math.Sin(a) + dy*math.Cos(a)) / w
					if math.Abs(u) >= 1 {
						continue
					}

					// Blunt where the brush lands (u = -1), thinning out towards the end.
					half := 1 - taper*math.Pow(max(u, 0), 2)
					if u < -0.8 {
						half *= math.Sqrt(1 - math.Pow((-u-0.8)/0.2, 2))
					}
					if math.Abs(v) >= half {
						continue
					}

					id := noise.Hash(hc, hr, s+8)
					value := 0.4 + 0.6*noise.Rand01(hc, hr, s+7)
					value *= bristle(v/half, u, bristles, dryness, int(id))
					best = max(best, value)
				}
			}
		}
		return float32(best)
	}), nil
}

// bristle is the paint left by bristle lines across the stroke at v (-1..1);
// dry bristles run out before the stroke's end at u = 1.
func bristle(v, u, count, dryness float64, seed int) float64 {
	if count <= 0 {
		return 1
	}

	t := (v + 1) / 2 * count
	line := int(math.Floor(t))
	f := t - float64(line)

	strength := 0.6 + 0.4*noise.Rand01(line, 0, seed)
	edge := smoothstep(0, 0.2, f) * smoothstep(1, 0.8, f)
	paint := strength * (0.7 + 0.3*edge)

	// Each bristle stops somewhere in the last part of the stroke.
	stop := 1 - dryness*noise.Rand01(line, 1, seed)
	if (u+1)/2 > stop {
		return 0
	}
	return paint
}

// scanNode is a histogram scan: values above position turn white, with
// contrast setting how hard the cut is.
func scanNode(e *graph.Env) (*img.Buffer, error) {
	src, err := e.Required("in")
	if err != nil {
		return nil, err
	}

	pos := e.Params.Float("position", 0.5)
	width := max(1-e.Params.Float("contrast", 0.8), 1e-3) / 2
	invert := e.Params.Bool("invert", false)

	src = gray(src)
	return mapGray(src, func(v float32) float32 {
		m := smoothstep(pos-width, pos+width, float64(v))
		if invert {
			m = 1 - m
		}
		return float32(m)
	}), nil
}
