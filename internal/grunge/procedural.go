package grunge

import (
	"math"
	"math/rand/v2"
)

var order = []string{"clouds", "blotches", "grime", "specks", "streaks", "scratches", "lines"}

// The built-in maps, made from noise, so they tile and may be passed on.
var builtins = map[string]func(size int) Gray{
	"clouds":    clouds,
	"blotches":  blotches,
	"grime":     grime,
	"specks":    specks,
	"streaks":   streaks,
	"scratches": scratches,
	"lines":     lines,
}

func hash(x, y, seed int) float64 {
	h := uint32(x)*374761393 + uint32(y)*668265263 + uint32(seed)*2246822519
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h) / float64(math.MaxUint32)
}

// noise is value noise over u, v in 0..1 that repeats px times across and py down.
func noise(u, v float64, px, py, seed int) float64 {
	x, y := u*float64(px), v*float64(py)
	x0, y0 := math.Floor(x), math.Floor(y)
	fx, fy := x-x0, y-y0
	fx, fy = fx*fx*(3-2*fx), fy*fy*(3-2*fy)
	i, j := int(x0), int(y0)
	at := func(a, b int) float64 { return hash(((a%px)+px)%px, ((b%py)+py)%py, seed) }
	top := at(i, j)*(1-fx) + at(i+1, j)*fx
	bottom := at(i, j+1)*(1-fx) + at(i+1, j+1)*fx
	return top*(1-fy) + bottom*fy
}

// fbm sums octaves of noise, each twice as fine and half as strong.
func fbm(u, v float64, px, py, octaves, seed int) float64 {
	var s, amp, total float64 = 0, 1, 0
	for o := range octaves {
		s += noise(u, v, px<<o, py<<o, seed+o*31) * amp
		total += amp
		amp *= 0.5
	}
	return s / total
}

// field fills a map from a function of u, v in 0..1.
func field(size int, f func(u, v float64) float64) Gray {
	g := newGray(size, size)
	for y := range size {
		for x := range size {
			g.V[y*size+x] = f((float64(x)+0.5)/float64(size), (float64(y)+0.5)/float64(size))
		}
	}
	return g
}

// warped shifts the lookup by noise, which turns round blobs into ragged stains.
func warped(u, v, by float64, base, seed int, f func(u, v float64) float64) float64 {
	du := fbm(u, v, base, base, 3, seed+100) - 0.5
	dv := fbm(u, v, base, base, 3, seed+200) - 0.5
	return f(u+du*by, v+dv*by)
}

func clouds(size int) Gray {
	return field(size, func(u, v float64) float64 { return fbm(u, v, 4, 4, 6, 1) })
}

func blotches(size int) Gray {
	return field(size, func(u, v float64) float64 {
		return warped(u, v, 0.2, 3, 2, func(u, v float64) float64 { return fbm(u, v, 3, 3, 6, 2) })
	})
}

func grime(size int) Gray {
	return field(size, func(u, v float64) float64 {
		coarse := warped(u, v, 0.2, 4, 3, func(u, v float64) float64 { return fbm(u, v, 6, 6, 5, 3) })
		return coarse*0.65 + fbm(u, v, 32, 32, 3, 4)*0.35
	})
}

// streaks run down: fine across, long along.
func streaks(size int) Gray {
	return field(size, func(u, v float64) float64 {
		runs := fbm(u, v, 48, 2, 4, 5)
		return runs*0.75 + fbm(u, v, 4, 4, 3, 6)*0.25
	})
}

// lines are bands across, wavy like wood grain.
func lines(size int) Gray {
	return field(size, func(u, v float64) float64 {
		w := fbm(u, v, 2, 3, 4, 7)
		band := math.Abs(math.Sin((v*14 + w*3) * math.Pi))
		return band*0.8 + fbm(u, v, 64, 8, 2, 8)*0.2
	})
}

// specks are dots of many sizes; the brighter they are, the sooner they show,
// so a higher amount brings more of them.
func specks(size int) Gray {
	g := field(size, func(u, v float64) float64 { return fbm(u, v, 16, 16, 3, 9) * 0.05 })
	r := rand.New(rand.NewPCG(9, 9))
	s := float64(size) / 512
	for range 9000 {
		cx, cy := r.Float64()*float64(size), r.Float64()*float64(size)
		rad := (0.5 + r.ExpFloat64()) * s
		stamp(g, cx, cy, rad, 0.1+r.Float64()*0.9)
	}
	return g
}

// scratches are thin strokes, most of them along one direction.
func scratches(size int) Gray {
	g := field(size, func(u, v float64) float64 { return fbm(u, v, 16, 16, 3, 10) * 0.05 })
	r := rand.New(rand.NewPCG(10, 10))
	s := float64(size) / 512
	for range 1100 {
		angle := 0.5 + r.NormFloat64()*0.35
		if r.Float64() < 0.25 {
			angle = r.Float64() * math.Pi
		}
		length := (10 + r.ExpFloat64()*40) * s
		x, y := r.Float64()*float64(size), r.Float64()*float64(size)
		bright := 0.1 + r.Float64()*0.9
		bend := r.NormFloat64() * 0.004
		for t := 0.0; t < length; t += 0.5 {
			stamp(g, x, y, 0.7*s, bright*(1-0.5*t/length))
			angle += bend
			x += math.Cos(angle) * 0.5
			y += math.Sin(angle) * 0.5
		}
	}
	return g
}

// stamp raises g to a soft dot, wrapping around the edges.
func stamp(g Gray, cx, cy, rad, bright float64) {
	reach := int(math.Ceil(rad)) + 1
	for y := int(cy) - reach; y <= int(cy)+reach; y++ {
		for x := int(cx) - reach; x <= int(cx)+reach; x++ {
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			v := bright * math.Min(1, math.Max(0, rad+0.5-d))
			i := (((y%g.H)+g.H)%g.H)*g.W + ((x%g.W)+g.W)%g.W
			g.V[i] = math.Max(g.V[i], v)
		}
	}
}
