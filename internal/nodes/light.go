package nodes

import (
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
)

var lightDef = Def{
	Eval:     lightNode,
	Category: "light",
	Inputs:   []string{"height"},
	Params: []Param{
		num("strength", 3, 0, 32),
		num("gain", 1, 0, 4),
		num("azimuth", 135, 0, 360),
		num("elevation", 45, 1, 90),
	},
}

var slopeDef = Def{
	Eval:     slopeNode,
	Category: "light",
	Inputs:   []string{"height"},
	Params: []Param{
		num("strength", 3, 0, 32),
		num("azimuth", 90, 0, 360),
	},
}

var curvatureDef = Def{
	Eval:     curvatureNode,
	Category: "light",
	Inputs:   []string{"height"},
	Params: []Param{
		px("radius", 1, 1, 16),
		num("gain", 4, 0, 32),
	},
}

// lightNode bakes directional lighting from a height map. Flat areas are 0.5,
// so the result works with an overlay blend.
func lightNode(e *graph.Env) (*img.Buffer, error) {
	h, err := e.Required("height")
	if err != nil {
		return nil, err
	}

	p := e.Params
	strength := p.Float("strength", 3) * e.Ctx.Scale
	gain := p.Float("gain", 1)
	az := p.Float("azimuth", 135) * math.Pi / 180
	el := p.Float("elevation", 45) * math.Pi / 180

	// Image y points down, so a positive azimuth lights from above.
	lx := math.Cos(el) * math.Cos(az)
	ly := -math.Cos(el) * math.Sin(az)
	lz := math.Sin(el)

	return fill(e.Ctx, func(x, y int) float32 {
		dx := float64(h.At(x+1, y, 0)-h.At(x-1, y, 0)) / 2 * strength
		dy := float64(h.At(x, y+1, 0)-h.At(x, y-1, 0)) / 2 * strength

		dot := (-dx*lx - dy*ly + lz) / math.Sqrt(dx*dx+dy*dy+1)
		return float32(clamp01(0.5 + (dot-lz)*gain))
	}), nil
}

// slopeNode is how much the surface faces a direction, like the green channel
// of a normal map: 0.5 flat, 1 facing towards azimuth, 0 facing away.
func slopeNode(e *graph.Env) (*img.Buffer, error) {
	h, err := e.Required("height")
	if err != nil {
		return nil, err
	}

	strength := e.Params.Float("strength", 3) * e.Ctx.Scale
	az := e.Params.Float("azimuth", 90) * math.Pi / 180
	lx, ly := math.Cos(az), -math.Sin(az)

	return fill(e.Ctx, func(x, y int) float32 {
		dx := float64(h.At(x+1, y, 0)-h.At(x-1, y, 0)) / 2 * strength
		dy := float64(h.At(x, y+1, 0)-h.At(x, y-1, 0)) / 2 * strength

		s := (-dx*lx - dy*ly) / math.Sqrt(dx*dx+dy*dy+1)
		return float32(clamp01(0.5 + s*0.5))
	}), nil
}

// curvatureNode is positive (above 0.5) on convex edges and negative in creases.
func curvatureNode(e *graph.Env) (*img.Buffer, error) {
	h, err := e.Required("height")
	if err != nil {
		return nil, err
	}

	r := max(1, int(math.Round(e.Px("radius", 1))))
	gain := float32(e.Params.Float("gain", 4))

	return fill(e.Ctx, func(x, y int) float32 {
		avg := (h.At(x-r, y, 0) + h.At(x+r, y, 0) + h.At(x, y-r, 0) + h.At(x, y+r, 0)) / 4
		return img.Clamp01(0.5 + (h.At(x, y, 0)-avg)*gain)
	}), nil
}

var shadowDef = Def{
	Eval:     shadowNode,
	Category: "light",
	Inputs:   []string{"height"},
	Params: []Param{
		px("depth", 8, 0, 64),
		num("azimuth", 100, 0, 360),
		num("elevation", 40, 1, 89),
		px("distance", 24, 1, 128),
		num("softness", 0.5, 0, 4),
	},
}

// shadowNode casts shadows from a height map, where height 1 stands depth px
// above 0. It marches from each pixel towards the light and darkens it when
// the surface rises above the ray. 1 is lit, 0 in shadow.
func shadowNode(e *graph.Env) (*img.Buffer, error) {
	h, err := e.Required("height")
	if err != nil {
		return nil, err
	}
	h = gray(h)

	depth := e.Px("depth", 8)
	az := e.Params.Float("azimuth", 100) * math.Pi / 180
	slope := math.Tan(e.Params.Float("elevation", 40) * math.Pi / 180)
	dist := e.Px("distance", 24)
	soft := max(e.Params.Float("softness", 0.5)*e.Ctx.Scale, 1e-3)

	// Image y points down, so a positive azimuth lights from above.
	dx, dy := math.Cos(az), -math.Sin(az)
	steps := int(math.Ceil(dist))

	return fill(e.Ctx, func(x, y int) float32 {
		z0 := float64(h.At(x, y, 0)) * depth
		lit := 1.0

		for s := 1; s <= steps; s++ {
			t := float64(s)
			sx := int(math.Round(float64(x) + dx*t))
			sy := int(math.Round(float64(y) + dy*t))

			above := float64(h.At(sx, sy, 0))*depth - (z0 + t*slope)
			if above > 0 {
				lit = min(lit, 1-clamp01(above/soft))
				if lit == 0 {
					break
				}
			}
		}
		return float32(lit)
	}), nil
}
