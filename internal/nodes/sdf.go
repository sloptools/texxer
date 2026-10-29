package nodes

import (
	"fmt"
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
)

// SDF buffers are gray buffers holding signed distances in pixels, negative inside.

var sdfOps = map[string]func(a, b, amount float64) float64{
	"union":     func(a, b, _ float64) float64 { return min(a, b) },
	"subtract":  func(a, b, _ float64) float64 { return max(a, -b) },
	"intersect": func(a, b, _ float64) float64 { return max(a, b) },
	"offset":    func(a, _, r float64) float64 { return a - r },
	"shell":     func(a, _, w float64) float64 { return math.Abs(a) - w/2 },
}

// sdfNode combines or modifies distance fields. offset grows the shape by
// amount px, shell turns it into an outline amount px wide.
func sdfNode(e *graph.Env) (*img.Buffer, error) {
	op := e.Params.String("op", "union")
	fn, ok := sdfOps[op]
	if !ok {
		return nil, fmt.Errorf("unknown sdf op %q", op)
	}

	a, err := e.Required("a")
	if err != nil {
		return nil, err
	}

	b := a
	if op == "union" || op == "subtract" || op == "intersect" {
		if b, err = e.Required("b"); err != nil {
			return nil, err
		}
	}

	amount := e.Px("amount", 2)
	out := img.New(a.W, a.H, 1, a.Tiling)
	for i := range out.Pix {
		out.Pix[i] = float32(fn(float64(a.Pix[i]), float64(b.Pix[i]), amount))
	}
	return out, nil
}

// profileNode turns a distance field into height: 0 outside, rising to 1
// over bevel px inside the shape.
func profileNode(e *graph.Env) (*img.Buffer, error) {
	src, err := e.Required("in")
	if err != nil {
		return nil, err
	}

	bevel := max(e.Px("bevel", 2), 0.5)
	profile := e.Params.String("profile", "round")

	out := img.New(src.W, src.H, 1, src.Tiling)
	for i, d := range src.Pix {
		out.Pix[i] = float32(bevelProfile(profile, -float64(d)/bevel))
	}
	return out, nil
}
