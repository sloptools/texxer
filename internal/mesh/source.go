package mesh

import "math"

// BaseColors lays the faces' own base colors out in new UVs, size×size RGBA: each
// texel of to takes the color its face had there, as from (the same faces with their
// original UVs and materials) has it. Alpha marks covered texels; the rest take the
// nearest covered color, so mipmaps stay on the surface's colors.
func BaseColors(details []*Detail, from, to []Tri, size int) []float32 {
	out := make([]float32, size*size*4)
	s := float64(size)
	for i, t := range to[:min(len(to), len(from))] {
		src := from[i]
		if src.Detail < 0 || src.Detail >= len(details) {
			continue
		}
		d := details[src.Detail]

		var u, v [3]float64
		for c := range 3 {
			u[c], v[c] = t.UV[c][0]*s, t.UV[c][1]*s
		}
		area := (u[1]-u[0])*(v[2]-v[0]) - (u[2]-u[0])*(v[1]-v[0])
		if math.Abs(area) < 1e-12 {
			continue
		}
		x0 := max(0, int(math.Floor(min(u[0], u[1], u[2]))))
		x1 := min(size-1, int(math.Ceil(max(u[0], u[1], u[2]))))
		y0 := max(0, int(math.Floor(min(v[0], v[1], v[2]))))
		y1 := min(size-1, int(math.Ceil(max(v[0], v[1], v[2]))))
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				px, py := float64(x)+0.5, float64(y)+0.5
				b0 := ((u[1]-px)*(v[2]-py) - (u[2]-px)*(v[1]-py)) / area
				b1 := ((u[2]-px)*(v[0]-py) - (u[0]-px)*(v[2]-py)) / area
				b2 := 1 - b0 - b1
				if b0 < 0 || b1 < 0 || b2 < 0 {
					continue
				}
				su := src.Src[0][0]*b0 + src.Src[1][0]*b1 + src.Src[2][0]*b2
				sv := src.Src[0][1]*b0 + src.Src[1][1]*b1 + src.Src[2][1]*b2
				c := d.colorAt(su, sv)
				k := (y*size + x) * 4
				out[k], out[k+1], out[k+2], out[k+3] = float32(c[0]), float32(c[1]), float32(c[2]), 1
			}
		}
	}
	fillRGBA(out, size)
	return out
}

// fillRGBA gives uncovered texels the color of the nearest covered one.
func fillRGBA(px []float32, size int) {
	done := make([]bool, size*size)
	var queue []int
	for i := range done {
		if px[i*4+3] > 0 {
			done[i] = true
			queue = append(queue, i)
		}
	}
	for len(queue) > 0 {
		i := queue[0]
		queue = queue[1:]
		x, y := i%size, i/size
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if nx < 0 || nx >= size || ny < 0 || ny >= size || done[ny*size+nx] {
				continue
			}
			j := ny*size + nx
			copy(px[j*4:j*4+3], px[i*4:i*4+3])
			done[j] = true
			queue = append(queue, j)
		}
	}
}
