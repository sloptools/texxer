package unwrap

import (
	"math"
	"math/bits"
	"slices"
)

// bitmap is a grid of pixels, a row of 64-bit words per line.
type bitmap struct {
	w, h, words int
	rows        []uint64
}

func newBitmap(w, h int) *bitmap {
	words := (w + 63) / 64
	return &bitmap{w, h, words, make([]uint64, words*h)}
}

func (b *bitmap) set(x, y int) {
	b.rows[y*b.words+x/64] |= 1 << (x % 64)
}

func (b *bitmap) get(x, y int) bool {
	return b.rows[y*b.words+x/64]&(1<<(x%64)) != 0
}

// mask is the pixels a chart touches at a density, grown by the padding, with the
// chart's origin at (pad, pad).
func (g *geometry) mask(c *chart, density float64, pad int) *bitmap {
	w, h := pixels(c.w, density), pixels(c.h, density)
	core := newBitmap(w, h)
	for _, f := range c.faces {
		var p [3][2]float64
		for k, v := range g.tri[f] {
			q := c.uv[c.local[v]]
			p[k] = [2]float64{q[0] * density, (c.h - q[1]) * density}
		}
		touch(p, core)
	}

	// Grown square by square: first along rows, then down columns.
	m := newBitmap(w+2*pad, h+2*pad)
	wide := newBitmap(w+2*pad, h)
	for y := range h {
		for x := range w {
			if core.get(x, y) {
				for dx := 0; dx <= 2*pad; dx++ {
					wide.set(x+dx, y)
				}
			}
		}
	}
	for y := range h {
		for x := range wide.w {
			if wide.get(x, y) {
				for dy := 0; dy <= 2*pad; dy++ {
					m.set(x, y+dy)
				}
			}
		}
	}
	return m
}

// touch marks every pixel the triangle overlaps (separating axes on pixel and triangle).
func touch(p [3][2]float64, b *bitmap) {
	x0 := max(0, int(math.Floor(min(p[0][0], p[1][0], p[2][0]))))
	x1 := min(b.w-1, int(math.Ceil(max(p[0][0], p[1][0], p[2][0])))-1)
	y0 := max(0, int(math.Floor(min(p[0][1], p[1][1], p[2][1]))))
	y1 := min(b.h-1, int(math.Ceil(max(p[0][1], p[1][1], p[2][1])))-1)

	for y := y0; y <= max(y0, y1); y++ {
		for x := x0; x <= max(x0, x1); x++ {
			if overlapsPixel(p, float64(x), float64(y)) {
				b.set(min(x, b.w-1), min(y, b.h-1))
			}
		}
	}
}

func overlapsPixel(p [3][2]float64, x, y float64) bool {
	corners := [4][2]float64{{x, y}, {x + 1, y}, {x, y + 1}, {x + 1, y + 1}}
	for i := range 3 {
		a, b, o := p[i], p[(i+1)%3], p[(i+2)%3]
		n := [2]float64{a[1] - b[1], b[0] - a[0]}
		side := n[0]*(o[0]-a[0]) + n[1]*(o[1]-a[1])
		if side == 0 {
			// A sliver: its bounding box is all there is to test.
			return true
		}
		outside := true
		for _, c := range corners {
			if (n[0]*(c[0]-a[0])+n[1]*(c[1]-a[1]))*side >= 0 {
				outside = false
				break
			}
		}
		if outside {
			return false
		}
	}
	return true
}

// fits reports whether the mask at (x, y) only covers free pixels of the bin.
func (b *bitmap) fits(m *bitmap, x, y int) bool {
	shift := x % 64
	for r := range m.h {
		row := b.rows[(y+r)*b.words:]
		at := x / 64
		for k := range m.words {
			word := m.rows[r*m.words+k]
			if word == 0 {
				continue
			}
			if row[at+k]&(word<<shift) != 0 {
				return false
			}
			if shift > 0 && at+k+1 < b.words && row[at+k+1]&(word>>(64-shift)) != 0 {
				return false
			}
		}
	}
	return true
}

func (b *bitmap) stamp(m *bitmap, x, y int) {
	shift := x % 64
	for r := range m.h {
		row := b.rows[(y+r)*b.words:]
		at := x / 64
		for k := range m.words {
			word := m.rows[r*m.words+k]
			row[at+k] |= word << shift
			if shift > 0 && at+k+1 < b.words {
				row[at+k+1] |= word >> (64 - shift)
			}
		}
	}
}

// rasterLayout packs the charts by their pixel masks, first fit line by line, so
// small charts can sit in the gaps of hollow ones. It returns each chart's position.
func (g *geometry) rasterLayout(charts []*chart, size, pad int, density float64) ([][2]int, bool) {
	bin := newBitmap(size, size)
	type item struct {
		i    int
		m    *bitmap
		area int
		// The first pixel of the mask's top line, for a quick test.
		first int
	}
	items := make([]item, len(charts))
	for i, c := range charts {
		m := g.mask(c, density, pad)
		if m.w > size || m.h > size {
			return nil, false
		}
		area := 0
		for _, w := range m.rows {
			area += bits.OnesCount64(w)
		}
		first := 0
		for !m.get(first, 0) {
			first++
		}
		items[i] = item{i, m, area, first}
	}
	slices.SortStableFunc(items, func(a, b item) int { return b.area - a.area })

	at := make([][2]int, len(charts))
	// Lines filled up to here are skipped by later charts.
	top := 0
	for _, it := range items {
		placed := false
		for y := top; y+it.m.h <= size && !placed; y++ {
			for x := 0; x+it.m.w <= size; x++ {
				if bin.rows[y*bin.words+x/64] == math.MaxUint64 {
					x += 63 - x%64
					continue
				}
				if bin.get(x+it.first, y) || !bin.fits(it.m, x, y) {
					continue
				}
				bin.stamp(it.m, x, y)
				at[it.i] = [2]int{x + pad, y + pad}
				placed = true
				break
			}
		}
		if !placed {
			return nil, false
		}
		for top < size && full(bin.rows[top*bin.words:(top+1)*bin.words], size) {
			top++
		}
	}
	return at, true
}

func full(row []uint64, w int) bool {
	for i, word := range row {
		bitsHere := min(64, w-i*64)
		if bits.OnesCount64(word) < bitsHere {
			return false
		}
	}
	return true
}
