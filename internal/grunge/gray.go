package grunge

import (
	"bytes"
	"cmp"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"math"
	"os"
	"slices"
)

// Gray is a square or rectangular map of values from 0 to 1.
type Gray struct {
	W, H int
	V    []float64
}

func newGray(w, h int) Gray {
	return Gray{w, h, make([]float64, w*h)}
}

// At wraps around the edges, as the maps tile.
func (g Gray) At(x, y int) float64 {
	x = ((x % g.W) + g.W) % g.W
	y = ((y % g.H) + g.H) % g.H
	return g.V[y*g.W+x]
}

// Load reads an image as luminance, cropped to the square in its middle.
func Load(path string) (Gray, error) {
	f, err := os.Open(path)
	if err != nil {
		return Gray{}, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return Gray{}, fmt.Errorf("%s: %w", path, err)
	}

	r := img.Bounds()
	side := min(r.Dx(), r.Dy())
	x0 := r.Min.X + (r.Dx()-side)/2
	y0 := r.Min.Y + (r.Dy()-side)/2
	g := newGray(side, side)
	for y := range side {
		for x := range side {
			cr, cg, cb, _ := img.At(x0+x, y0+y).RGBA()
			g.V[y*side+x] = (0.299*float64(cr) + 0.587*float64(cg) + 0.114*float64(cb)) / 0xffff
		}
	}
	return g, nil
}

// Downsample averages g down to size², never up.
func Downsample(g Gray, size int) Gray {
	if g.W <= size {
		return g
	}
	out := newGray(size, size)
	scale := float64(g.W) / float64(size)
	for y := range size {
		for x := range size {
			x0, x1 := int(float64(x)*scale), max(int(float64(x+1)*scale), int(float64(x)*scale)+1)
			y0, y1 := int(float64(y)*scale), max(int(float64(y+1)*scale), int(float64(y)*scale)+1)
			var s float64
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					s += g.V[sy*g.W+sx]
				}
			}
			out.V[y*size+x] = s / float64((x1-x0)*(y1-y0))
		}
	}
	return out
}

// Equalize replaces each value by its rank, so a mask cut at 1 - amount covers
// about that share of the surface; equal values share their mean rank.
func Equalize(g Gray) Gray {
	order := make([]int, len(g.V))
	for i := range order {
		order[i] = i
	}
	slices.SortFunc(order, func(a, b int) int { return cmp.Compare(g.V[a], g.V[b]) })

	out := newGray(g.W, g.H)
	n := float64(len(order) - 1)
	for i := 0; i < len(order); {
		j := i
		for j+1 < len(order) && g.V[order[j+1]] == g.V[order[i]] {
			j++
		}
		rank := float64(i+j) / 2 / n
		for k := i; k <= j; k++ {
			out.V[order[k]] = rank
		}
		i = j + 1
	}
	return out
}

// Encode writes g as an 8-bit gray PNG.
func Encode(g Gray) ([]byte, error) {
	img := image.NewGray(image.Rect(0, 0, g.W, g.H))
	for i, v := range g.V {
		img.SetGray(i%g.W, i/g.W, color.Gray{uint8(math.Round(math.Min(1, math.Max(0, v)) * 255))})
	}
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	return buf.Bytes(), err
}
