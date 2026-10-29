package nodes

import (
	"math"
	"testing"

	"texxer/internal/img"
	"texxer/internal/noise"
)

func TestBoxPass(t *testing.T) {
	for _, tiling := range []img.Tiling{img.TileBoth, img.TileX, img.TileNone} {
		src := img.New(13, 7, 1, tiling)
		for i := range src.Pix {
			src.Pix[i] = float32(noise.Rand01(i, 0, 1))
		}

		for _, dir := range [][2]int{{1, 0}, {0, 1}} {
			got := boxPass(src, 3, dir[0], dir[1])

			for y := range src.H {
				for x := range src.W {
					var sum float32
					for k := -3; k <= 3; k++ {
						sum += src.At(x+k*dir[0], y+k*dir[1], 0)
					}
					if d := got.Pix[y*src.W+x] - sum/7; math.Abs(float64(d)) > 1e-5 {
						t.Fatalf("tiling %d dir %v at %d,%d: off by %g", tiling, dir, x, y, d)
					}
				}
			}
		}
	}
}
