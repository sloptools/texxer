// Package stats measures the look of a texture, to compare it against references.
package stats

import (
	"image"
	"math"
	"slices"
)

type Stats struct {
	Mean, Std float64
	// P2 and P98 are luminance percentiles, a contrast range that ignores outliers.
	P2, P98 float64
	// Detail is the mean absolute luminance difference to the right and lower neighbor.
	Detail float64
	Colors int
}

func Measure(im image.Image) Stats {
	b := im.Bounds()
	w, h := b.Dx(), b.Dy()
	lum := make([]float64, w*h)
	colors := map[uint32]bool{}

	for y := range h {
		for x := range w {
			r, g, bl, _ := im.At(b.Min.X+x, b.Min.Y+y).RGBA()
			lum[y*w+x] = (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(bl)) / 0xffff
			colors[r>>8<<16|g>>8<<8|bl>>8] = true
		}
	}

	var s Stats
	s.Colors = len(colors)

	for _, v := range lum {
		s.Mean += v
	}
	s.Mean /= float64(len(lum))

	var detail float64
	for y := range h {
		for x := range w {
			v := lum[y*w+x]
			s.Std += (v - s.Mean) * (v - s.Mean)
			detail += math.Abs(v-lum[y*w+(x+1)%w]) + math.Abs(v-lum[(y+1)%h*w+x])
		}
	}
	s.Std = math.Sqrt(s.Std / float64(len(lum)))
	s.Detail = detail / float64(2*len(lum))

	sorted := slices.Clone(lum)
	slices.Sort(sorted)
	s.P2 = sorted[len(sorted)*2/100]
	s.P98 = sorted[len(sorted)*98/100]

	return s
}
