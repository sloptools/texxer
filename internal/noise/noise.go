package noise

import "math"

// All noise is periodic: px and py are the periods in cells, which keeps textures tileable.

func Hash(x, y, seed int) uint32 {
	h := uint32(x)*0x8da6b343 ^ uint32(y)*0xd8163841 ^ uint32(seed)*0xcb1ab31f
	h ^= h >> 13
	h *= 0x5bd1e995
	h ^= h >> 15
	return h
}

func Rand01(x, y, seed int) float64 {
	return float64(Hash(x, y, seed)) / (1 << 32)
}

// Perlin returns gradient noise in about [-1, 1].
func Perlin(x, y float64, px, py, seed int) float64 {
	x0, y0 := math.Floor(x), math.Floor(y)
	fx, fy := x-x0, y-y0
	ix, iy := int(x0), int(y0)

	grad := func(dx, dy int) float64 {
		a := Rand01(mod(ix+dx, px), mod(iy+dy, py), seed) * 2 * math.Pi
		return math.Cos(a)*(fx-float64(dx)) + math.Sin(a)*(fy-float64(dy))
	}

	u, v := fade(fx), fade(fy)
	top := lerp(grad(0, 0), grad(1, 0), u)
	bottom := lerp(grad(0, 1), grad(1, 1), u)

	return lerp(top, bottom, v) * math.Sqrt2
}

// FBM doubles the period each octave so tiling is kept.
func FBM(x, y float64, px, py, octaves int, persistence float64, seed int) float64 {
	sum, amp, norm := 0.0, 1.0, 0.0

	for o := range octaves {
		f := float64(int(1) << o)
		sum += amp * Perlin(x*f, y*f, px<<o, py<<o, seed+o*101)
		norm += amp
		amp *= persistence
	}
	return sum / norm
}

// Worley returns the distances to the nearest and second nearest feature point in cell units.
func Worley(x, y float64, px, py, seed int) (f1, f2 float64) {
	ix, iy := int(math.Floor(x)), int(math.Floor(y))
	f1, f2 = math.MaxFloat64, math.MaxFloat64

	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			cx, cy := ix+dx, iy+dy
			wx, wy := mod(cx, px), mod(cy, py)

			fx := float64(cx) + Rand01(wx, wy, seed)
			fy := float64(cy) + Rand01(wx, wy, seed+1)
			d := math.Hypot(x-fx, y-fy)

			if d < f1 {
				f1, f2 = d, f1
			} else if d < f2 {
				f2 = d
			}
		}
	}
	return f1, f2
}

func fade(t float64) float64 {
	return t * t * t * (t*(t*6-15) + 10)
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func mod(a, n int) int {
	a %= n
	if a < 0 {
		a += n
	}
	return a
}
