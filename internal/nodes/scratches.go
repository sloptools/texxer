package nodes

import (
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
	"texxer/internal/noise"
)

func scratches(e *graph.Env) *img.Buffer {
	p := e.Params
	count := p.Int("count", 150)
	length := e.Px("length", 30)
	width := e.Px("width", 1)
	angle := p.Float("angle", 30) * math.Pi / 180
	spread := p.Float("spread", 40) * math.Pi / 180
	seed := e.Seed()

	b := img.New(e.Ctx.W, e.Ctx.H, 1, e.Ctx.Tiling)

	for i := range count {
		r := func(k int) float64 { return noise.Rand01(i, k, seed) }

		x0 := r(0) * float64(b.W)
		y0 := r(1) * float64(b.H)
		a := angle + (r(2)-0.5)*spread
		l := length * (0.3 + r(3))
		x1, y1 := x0+math.Cos(a)*l, y0+math.Sin(a)*l

		drawLine(b, x0, y0, x1, y1, width, float32(0.4+0.6*r(4)))
	}
	return b
}

// drawLine draws an anti-aliased segment, wrapping around tiled edges.
func drawLine(b *img.Buffer, x0, y0, x1, y1, width float64, v float32) {
	pad := width + 1
	minX, maxX := int(math.Floor(min(x0, x1)-pad)), int(math.Ceil(max(x0, x1)+pad))
	minY, maxY := int(math.Floor(min(y0, y1)-pad)), int(math.Ceil(max(y0, y1)+pad))

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			d := segmentDist(float64(x)+0.5, float64(y)+0.5, x0, y0, x1, y1)
			a := float32(clamp01(width/2 + 0.5 - d))
			if a <= 0 {
				continue
			}

			i, ok := wrapIndex(b, x, y)
			if ok {
				b.Pix[i] = max(b.Pix[i], a*v)
			}
		}
	}
}

func segmentDist(px, py, x0, y0, x1, y1 float64) float64 {
	dx, dy := x1-x0, y1-y0
	t := clamp01(((px-x0)*dx + (py-y0)*dy) / (dx*dx + dy*dy))
	return math.Hypot(px-(x0+t*dx), py-(y0+t*dy))
}
