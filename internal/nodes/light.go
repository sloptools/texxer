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
