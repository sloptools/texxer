package nodes

import (
	"errors"
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
)

var decalDef = Def{
	Eval:     decalNode,
	Category: "mesh",
	Inputs:   []string{"position", "normal", "motif"},
	Params: []Param{
		data("center", []float64{0.5, 0.5, 0.5}),
		data("direction", []float64{0, 0, 1}),
		num("size", 0.2, 0.001, 2),
		num("aspect", 1, 0.05, 20),
		num("rotation", 0, -180, 180),
		num("depth", 0.1, 0.001, 2),
		num("facing", 0.2, -1, 1),
		integer("resolution", 512, 32, 2048),
	},
}

// decalNode projects a motif onto a baked model along direction, centered
// at center in the position map's units (fractions of the model's size).
// Only surfaces within depth of the center and facing the projection by
// more than facing get it, so the back side stays clean. The motif renders
// at resolution px along its longer side; its first channel is the mask.
func decalNode(e *graph.Env) (*img.Buffer, error) {
	pos, err := e.Required("position")
	if err != nil {
		return nil, err
	}
	nor, err := e.Required("normal")
	if err != nil {
		return nil, err
	}

	p := e.Params
	center, err := vec3(p, "center", [3]float64{0.5, 0.5, 0.5})
	if err != nil {
		return nil, err
	}
	dir, err := vec3(p, "direction", [3]float64{0, 0, 1})
	if err != nil {
		return nil, err
	}
	fwd := norm3(dir)
	if fwd == ([3]float64{}) {
		return nil, errors.New("direction is zero")
	}

	right, up := decalBasis(fwd, p.Float("rotation", 0)*math.Pi/180)
	half := p.Float("size", 0.2) / 2
	aspect := max(p.Float("aspect", 1), 0.01)
	hw, hh := half*math.Sqrt(aspect), half/math.Sqrt(aspect)
	depth := p.Float("depth", 0.1)
	facing := p.Float("facing", 0.2)

	// The motif renders in the decal's proportions, so it isn't stretched; 512 px across is its design width.
	res := p.Int("resolution", 512)
	mw, mh := res, max(8, int(math.Round(float64(res)/aspect)))
	if aspect < 1 {
		mw, mh = max(8, int(math.Round(float64(res)*aspect))), res
	}
	motif, err := e.InputAt("motif", graph.Ctx{W: mw, H: mh, Tiling: img.TileNone, Scale: float64(mw) / 512})
	if err != nil {
		return nil, err
	}

	out := img.New(pos.W, pos.H, 1, pos.Tiling)
	img.Rows(pos.H, func(y int) {
		for x := range pos.W {
			i := y*pos.W + x
			d := [3]float64{
				float64(pos.Value(i, 0)) - center[0],
				float64(pos.Value(i, 1)) - center[1],
				float64(pos.Value(i, 2)) - center[2],
			}

			z := dot3(d, fwd)
			if math.Abs(z) > depth {
				continue
			}
			u := dot3(d, right) / hw
			v := dot3(d, up) / hh
			if math.Abs(u) >= 1 || math.Abs(v) >= 1 {
				continue
			}

			n := [3]float64{float64(nor.Value(i, 0))*2 - 1, float64(nor.Value(i, 1))*2 - 1, float64(nor.Value(i, 2))*2 - 1}
			if dot3(n, fwd) < facing {
				continue
			}

			value := float32(1)
			if motif != nil {
				// Motif y runs down, decal up runs up.
				value = motif.Sample((u+1)/2*float64(motif.W), (1-v)/2*float64(motif.H), 0)
			}
			out.Pix[i] = value
		}
	})
	return out, nil
}

// decalBasis returns the decal's right and up axes for a projection along
// fwd: up leans towards world up, then both turn by rot around fwd.
func decalBasis(fwd [3]float64, rot float64) (right, up [3]float64) {
	worldUp := [3]float64{0, 1, 0}
	if math.Abs(fwd[1]) > 0.95 {
		worldUp = [3]float64{0, 0, -1}
	}
	right = norm3(cross3(worldUp, fwd))
	up = cross3(fwd, right)

	c, s := math.Cos(rot), math.Sin(rot)
	r2, u2 := [3]float64{}, [3]float64{}
	for k := range 3 {
		r2[k] = right[k]*c + up[k]*s
		u2[k] = up[k]*c - right[k]*s
	}
	return r2, u2
}

func vec3(p graph.Params, key string, def [3]float64) ([3]float64, error) {
	var v []float64
	if err := p.Decode(key, &v); err != nil {
		return def, err
	}
	if v == nil {
		return def, nil
	}
	if len(v) != 3 {
		return def, errors.New(key + " needs three numbers")
	}
	return [3]float64{v[0], v[1], v[2]}, nil
}

func dot3(a, b [3]float64) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }

func cross3(a, b [3]float64) [3]float64 {
	return [3]float64{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}

func norm3(a [3]float64) [3]float64 {
	l := math.Sqrt(dot3(a, a))
	if l == 0 {
		return [3]float64{}
	}
	return [3]float64{a[0] / l, a[1] / l, a[2] / l}
}
