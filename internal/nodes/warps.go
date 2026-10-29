package nodes

import (
	"fmt"
	"math"
	"slices"

	"texxer/internal/graph"
	"texxer/internal/img"
)

var dirwarpDef = Def{
	Eval:     dirwarpNode,
	Category: "filter",
	Inputs:   []string{"in", "intensity"},
	Params: []Param{
		px("amount", 8, 0, 128),
		num("angle", 0, -180, 180),
		integer("directions", 1, 1, 8),
		enum("mode", "average", "average", "min", "max"),
	},
}

var gradwarpDef = Def{
	Eval:     gradwarpNode,
	Category: "filter",
	Inputs:   []string{"in", "intensity"},
	Params: []Param{
		px("amount", 8, -128, 128),
		px("smooth", 2, 0, 32),
	},
}

var slopeBlurDef = Def{
	Eval:     slopeBlurNode,
	Category: "filter",
	Inputs:   []string{"in", "slope"},
	Params: []Param{
		px("amount", 8, -128, 128),
		integer("samples", 12, 2, 64),
		px("smooth", 2, 0, 32),
		enum("mode", "blur", "blur", "min", "max"),
	},
}

// dirwarpNode shifts its input by amount px times the intensity mask, along
// angle. With more directions it warps along evenly spaced angles and
// combines the results, which gives organic, smeared shapes.
func dirwarpNode(e *graph.Env) (*img.Buffer, error) {
	src, err := e.Required("in")
	if err != nil {
		return nil, err
	}
	intensity, err := e.Input("intensity")
	if err != nil {
		return nil, err
	}

	amount := e.Px("amount", 8)
	angle := e.Params.Float("angle", 0) * math.Pi / 180
	n := max(1, e.Params.Int("directions", 1))
	combine, err := combiner(e.Params.String("mode", "average"))
	if err != nil {
		return nil, err
	}

	dirs := make([][2]float64, n)
	for k := range dirs {
		a := angle + float64(k)*2*math.Pi/float64(n)
		// Image y points down, so positive angles turn counterclockwise on screen.
		dirs[k] = [2]float64{math.Cos(a), -math.Sin(a)}
	}

	return displace(src, func(x, y int, offsets [][2]float64) [][2]float64 {
		s := amount
		if intensity != nil {
			s *= float64(intensity.Value(y*src.W+x, 0))
		}

		for _, d := range dirs {
			offsets = append(offsets, [2]float64{d[0] * s, d[1] * s})
		}
		return offsets
	}, combine), nil
}

// gradwarpNode is a non-directional warp: it pushes pixels along the slope
// of the intensity map, so shapes flow around its hills and valleys.
func gradwarpNode(e *graph.Env) (*img.Buffer, error) {
	src, err := e.Required("in")
	if err != nil {
		return nil, err
	}
	intensity, err := e.Required("intensity")
	if err != nil {
		return nil, err
	}

	gx, gy := gradient(intensity, e.Px("smooth", 2))
	amount := e.Px("amount", 8)

	return displace(src, func(x, y int, offsets [][2]float64) [][2]float64 {
		i := y*src.W + x
		return append(offsets, [2]float64{float64(gx[i]) * amount, float64(gy[i]) * amount})
	}, average), nil
}

// slopeBlurNode smears its input along the slope of another map, like paint
// dragged downhill. min and max erode or grow shapes instead of blurring,
// which gives chipped, brushed edges.
func slopeBlurNode(e *graph.Env) (*img.Buffer, error) {
	src, err := e.Required("in")
	if err != nil {
		return nil, err
	}
	slope, err := e.Required("slope")
	if err != nil {
		return nil, err
	}

	gx, gy := gradient(slope, e.Px("smooth", 2))
	amount := e.Px("amount", 8)
	n := max(2, e.Params.Int("samples", 12))
	combine, err := combiner(map[string]string{"blur": "average", "min": "min", "max": "max"}[e.Params.String("mode", "blur")])
	if err != nil {
		return nil, err
	}

	// Gradients as buffers, so steps can read them between pixels.
	gxb := &img.Buffer{W: src.W, H: src.H, C: 1, Tiling: src.Tiling, Pix: gx}
	gyb := &img.Buffer{W: src.W, H: src.H, C: 1, Tiling: src.Tiling, Pix: gy}
	step := amount / float64(n-1)

	// Each step follows the slope where the previous one landed, so the
	// smear flows along the slope map instead of running straight.
	return displace(src, func(x, y int, offsets [][2]float64) [][2]float64 {
		px, py := float64(x)+0.5, float64(y)+0.5
		ox, oy := 0.0, 0.0

		offsets = append(offsets, [2]float64{0, 0})
		for range n - 1 {
			ox += float64(gxb.Sample(px+ox, py+oy, 0)) * step
			oy += float64(gyb.Sample(px+ox, py+oy, 0)) * step
			offsets = append(offsets, [2]float64{ox, oy})
		}
		return offsets
	}, combine), nil
}

type combineFunc func(vals []float32) float32

func average(vals []float32) float32 {
	var sum float32
	for _, v := range vals {
		sum += v
	}
	return sum / float32(len(vals))
}

func combiner(mode string) (combineFunc, error) {
	switch mode {
	case "average":
		return average, nil
	case "min":
		return func(v []float32) float32 { return slices.Min(v) }, nil
	case "max":
		return func(v []float32) float32 { return slices.Max(v) }, nil
	}
	return nil, fmt.Errorf("unknown mode %q", mode)
}

// displace samples src at every offset fn appends for a pixel and combines them per channel.
func displace(src *img.Buffer, fn func(x, y int, offsets [][2]float64) [][2]float64, combine combineFunc) *img.Buffer {
	out := img.New(src.W, src.H, src.C, src.Tiling)

	img.Rows(src.H, func(y int) {
		var vals []float32
		var offsets [][2]float64
		for x := range src.W {
			offsets = fn(x, y, offsets[:0])
			for ch := range src.C {
				vals = vals[:0]
				for _, o := range offsets {
					vals = append(vals, src.Sample(float64(x)+0.5+o[0], float64(y)+0.5+o[1], ch))
				}
				out.Pix[(y*src.W+x)*src.C+ch] = combine(vals)
			}
		}
	})
	return out
}

// gradient returns the slope of b's first channel, scaled so the steepest
// slopes (95th percentile) have length 1. That makes amount params mean
// "about this many px" whatever the contrast of the map.
func gradient(b *img.Buffer, smooth float64) (gx, gy []float32) {
	b = gray(b)
	if r := int(math.Round(smooth)); r > 0 {
		b = boxPass(boxPass(b, r, 1, 0), r, 0, 1)
	}

	gx = make([]float32, b.W*b.H)
	gy = make([]float32, b.W*b.H)
	mags := make([]float32, 0, len(gx)/16+1)

	for y := range b.H {
		for x := range b.W {
			i := y*b.W + x
			gx[i] = (b.At(x+1, y, 0) - b.At(x-1, y, 0)) / 2
			gy[i] = (b.At(x, y+1, 0) - b.At(x, y-1, 0)) / 2
			if i%16 == 0 {
				mags = append(mags, float32(math.Hypot(float64(gx[i]), float64(gy[i]))))
			}
		}
	}

	slices.Sort(mags)
	ref := max(mags[len(mags)*95/100], 1e-6)
	for i := range gx {
		gx[i] /= ref
		gy[i] /= ref
	}
	return gx, gy
}
