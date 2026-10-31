package mesh

import (
	"image"
	"image/color"
	"math"
)

// Remap carries an image laid out by one set of UVs over to another: each texel of the
// result takes the color img had where the same surface point sat before. Faces
// correspond by index. Uncovered texels within padding copy a covered neighbor.
func Remap(img image.Image, from, to []Tri, padding int) *image.NRGBA {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	covered := make([]bool, w*h)

	for i, t := range to {
		var u, v [3]float64
		for c := range 3 {
			u[c], v[c] = t.UV[c][0]*float64(w), t.UV[c][1]*float64(h)
		}
		area := (u[1]-u[0])*(v[2]-v[0]) - (u[2]-u[0])*(v[1]-v[0])
		if math.Abs(area) < 1e-12 {
			continue
		}
		old := from[i].UV

		x0 := max(0, int(math.Floor(min(u[0], u[1], u[2]))))
		x1 := min(w-1, int(math.Ceil(max(u[0], u[1], u[2]))))
		y0 := max(0, int(math.Floor(min(v[0], v[1], v[2]))))
		y1 := min(h-1, int(math.Ceil(max(v[0], v[1], v[2]))))
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				px, py := float64(x)+0.5, float64(y)+0.5
				b0 := ((u[1]-px)*(v[2]-py) - (u[2]-px)*(v[1]-py)) / area
				b1 := ((u[2]-px)*(v[0]-py) - (u[0]-px)*(v[2]-py)) / area
				b2 := 1 - b0 - b1
				if b0 < 0 || b1 < 0 || b2 < 0 {
					continue
				}
				ou := old[0][0]*b0 + old[1][0]*b1 + old[2][0]*b2
				ov := old[0][1]*b0 + old[1][1]*b1 + old[2][1]*b2
				sx := min(w-1, max(0, int(ou*float64(w))))
				sy := min(h-1, max(0, int(ov*float64(h))))
				out.Set(x, y, img.At(b.Min.X+sx, b.Min.Y+sy))
				covered[y*w+x] = true
			}
		}
	}

	for range padding {
		next := append([]bool(nil), covered...)
		for y := range h {
			for x := range w {
				if covered[y*w+x] {
					continue
				}
				for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					nx, ny := x+d[0], y+d[1]
					if nx >= 0 && nx < w && ny >= 0 && ny < h && covered[ny*w+nx] {
						out.Set(x, y, out.At(nx, ny))
						next[y*w+x] = true
						break
					}
				}
			}
		}
		covered = next
	}
	for i, c := range covered {
		if !c {
			out.Set(i%w, i/w, color.NRGBA{A: 255})
		}
	}
	return out
}
