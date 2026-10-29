package nodes

import (
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
)

var distanceDef = Def{
	Eval:     distanceNode,
	Category: "shape",
	Inputs:   []string{"in"},
	Params: []Param{
		num("threshold", 0.5, 0, 1),
		px("range", 32, 1, 256),
	},
}

// distanceNode turns a mask into a distance field, so any mask (a scan, an
// image, a flood fill) can be beveled, grown or outlined with the sdf and
// profile nodes. Distances are exact up to range px and clamped beyond.
func distanceNode(e *graph.Env) (*img.Buffer, error) {
	src, err := e.Required("in")
	if err != nil {
		return nil, err
	}
	src = gray(src)

	threshold := float32(e.Params.Float("threshold", 0.5))
	reach := max(1, int(math.Ceil(e.Px("range", 32))))

	inside := make([]bool, src.W*src.H)
	for i, v := range src.Pix {
		inside[i] = v >= threshold
	}

	toInside := edt(inside, src.W, src.H, src.Tiling, reach, true)
	toOutside := edt(inside, src.W, src.H, src.Tiling, reach, false)

	limit := float64(reach)
	out := img.New(src.W, src.H, 1, src.Tiling)
	for i := range out.Pix {
		// Half a pixel puts the boundary between the last inside and first outside pixel.
		d := math.Sqrt(toInside[i]) - math.Sqrt(toOutside[i])
		if inside[i] {
			d += 0.5
		} else {
			d -= 0.5
		}
		out.Pix[i] = float32(min(max(d, -limit), limit))
	}
	return out, nil
}

// edt returns squared distances to the nearest pixel whose inside flag equals
// target, using the separable transform of Felzenszwalb and Huttenlocher.
// Tiled axes are padded by reach px of wrapped pixels.
func edt(inside []bool, w, h int, tiling img.Tiling, reach int, target bool) []float64 {
	padX, padY := 0, 0
	if tiling != img.TileNone {
		padX = min(reach, w)
	}
	if tiling == img.TileBoth {
		padY = min(reach, h)
	}
	pw, ph := w+2*padX, h+2*padY

	const far = 1e20
	grid := make([]float64, pw*ph)
	for y := range ph {
		for x := range pw {
			sx, sy := img.Mod(x-padX, w), img.Mod(y-padY, h)
			if padY == 0 {
				sy = y
			}
			if padX == 0 {
				sx = x
			}
			if inside[sy*w+sx] == target {
				grid[y*pw+x] = 0
			} else {
				grid[y*pw+x] = far
			}
		}
	}

	line := make([]float64, max(pw, ph))
	out := make([]float64, max(pw, ph))
	for x := range pw {
		for y := range ph {
			line[y] = grid[y*pw+x]
		}
		edt1D(line[:ph], out[:ph])
		for y := range ph {
			grid[y*pw+x] = out[y]
		}
	}
	for y := range ph {
		edt1D(grid[y*pw:(y+1)*pw], out[:pw])
		copy(grid[y*pw:], out[:pw])
	}

	res := make([]float64, w*h)
	for y := range h {
		copy(res[y*w:(y+1)*w], grid[(y+padY)*pw+padX:])
	}
	return res
}

// edt1D is the lower envelope of parabolas rooted at f.
func edt1D(f, d []float64) {
	n := len(f)
	v := make([]int, n)
	z := make([]float64, n+1)

	k := 0
	z[0], z[1] = math.Inf(-1), math.Inf(1)
	for q := 1; q < n; q++ {
		for {
			p := v[k]
			s := ((f[q] + float64(q*q)) - (f[p] + float64(p*p))) / float64(2*(q-p))
			if s > z[k] {
				k++
				v[k] = q
				z[k] = s
				z[k+1] = math.Inf(1)
				break
			}
			if k == 0 {
				v[0] = q
				z[1] = math.Inf(1)
				break
			}
			k--
		}
	}

	k = 0
	for q := range n {
		for z[k+1] < float64(q) {
			k++
		}
		dq := float64(q - v[k])
		d[q] = dq*dq + f[v[k]]
	}
}
