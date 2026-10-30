// Command grunge turns grunge photos into small tileable masks for texxer:
// square crop, area downsample, large-scale shading removed, white = dirt,
// edges cross-faded so the map tiles, then equalized so that a mask's amount
// is about the share of the surface it covers, whatever the map.
//
// usage: go run ./tools/grunge -o internal/grunge/maps -root ~/Desktop -list tools/grunge/sources.txt
package main

import (
	"cmp"
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func main() {
	out := flag.String("o", ".", "output directory")
	size := flag.Int("size", 512, "output size")
	flatten := flag.Float64("flatten", 0.8, "how much large-scale shading to remove")
	root := flag.String("root", ".", "directory the sources are under")
	list := flag.String("list", "", "file with one \"name source\" per line")
	flag.Parse()

	data, err := os.ReadFile(*list)
	if err != nil {
		log.Fatal(err)
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		name, src := fields[0], filepath.Join(*root, fields[1])
		g, err := load(src)
		if err != nil {
			log.Fatal(err)
		}

		m := process(g, *size, *flatten)
		if err := save(filepath.Join(*out, name+".png"), m, *size); err != nil {
			log.Fatal(err)
		}
		fmt.Println(name)
	}
}

type gray struct {
	w, h int
	v    []float64
}

func (g gray) at(x, y int) float64 {
	x = ((x % g.w) + g.w) % g.w
	y = ((y % g.h) + g.h) % g.h
	return g.v[y*g.w+x]
}

func load(path string) (gray, error) {
	f, err := os.Open(path)
	if err != nil {
		return gray{}, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return gray{}, fmt.Errorf("%s: %w", path, err)
	}

	r := img.Bounds()
	side := min(r.Dx(), r.Dy())
	x0 := r.Min.X + (r.Dx()-side)/2
	y0 := r.Min.Y + (r.Dy()-side)/2
	g := gray{side, side, make([]float64, side*side)}
	for y := range side {
		for x := range side {
			cr, cg, cb, _ := img.At(x0+x, y0+y).RGBA()
			g.v[y*side+x] = (0.299*float64(cr) + 0.587*float64(cg) + 0.114*float64(cb)) / 0xffff
		}
	}
	return g, nil
}

func process(g gray, size int, flatten float64) gray {
	m := downsample(g, size)

	// Vignettes and lighting gradients in the photos would repeat as blotches.
	low := blur(m, size/8)
	for i := range m.v {
		m.v[i] -= low.v[i] * flatten
	}
	stretch(m, 0.01, 0.995)

	// Dirt is the minority of the surface.
	if mean(m) > 0.5 {
		for i := range m.v {
			m.v[i] = 1 - m.v[i]
		}
	}
	return equalize(seamless(equalize(m)))
}

// equalize replaces each value by its rank; equal values share their mean rank.
func equalize(g gray) gray {
	order := make([]int, len(g.v))
	for i := range order {
		order[i] = i
	}
	slices.SortFunc(order, func(a, b int) int { return cmp.Compare(g.v[a], g.v[b]) })

	out := gray{g.w, g.h, make([]float64, len(g.v))}
	n := float64(len(order) - 1)
	for i := 0; i < len(order); {
		j := i
		for j+1 < len(order) && g.v[order[j+1]] == g.v[order[i]] {
			j++
		}
		rank := float64(i+j) / 2 / n
		for k := i; k <= j; k++ {
			out.v[order[k]] = rank
		}
		i = j + 1
	}
	return out
}

func downsample(g gray, size int) gray {
	out := gray{size, size, make([]float64, size*size)}
	scale := float64(g.w) / float64(size)
	for y := range size {
		for x := range size {
			x0, x1 := int(float64(x)*scale), max(int(float64(x+1)*scale), int(float64(x)*scale)+1)
			y0, y1 := int(float64(y)*scale), max(int(float64(y+1)*scale), int(float64(y)*scale)+1)
			var s float64
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					s += g.v[sy*g.w+sx]
				}
			}
			out.v[y*size+x] = s / float64((x1-x0)*(y1-y0))
		}
	}
	return out
}

// blur is three wrapping box passes, close to a gaussian.
func blur(g gray, radius int) gray {
	out := g
	for range 3 {
		out = boxPass(boxPass(out, radius, 1, 0), radius, 0, 1)
	}
	return out
}

func boxPass(g gray, r, dx, dy int) gray {
	out := gray{g.w, g.h, make([]float64, len(g.v))}
	for y := range g.h {
		for x := range g.w {
			var s float64
			for k := -r; k <= r; k++ {
				s += g.at(x+k*dx, y+k*dy)
			}
			out.v[y*g.w+x] = s / float64(2*r+1)
		}
	}
	return out
}

func stretch(g gray, lo, hi float64) {
	sorted := slices.Clone(g.v)
	slices.Sort(sorted)
	a := sorted[int(lo*float64(len(sorted)-1))]
	b := sorted[int(hi*float64(len(sorted)-1))]
	for i, v := range g.v {
		g.v[i] = math.Min(1, math.Max(0, (v-a)/math.Max(b-a, 1e-6)))
	}
}

func mean(g gray) float64 {
	var s float64
	for _, v := range g.v {
		s += v
	}
	return s / float64(len(g.v))
}

// seamless makes g tile, one axis at a time: the borders fade into a copy
// shifted by half along that axis, whose own seam then sits in the middle where
// the original shows. The blend keeps the variance, so the fade doesn't show as
// a washed-out band.
func seamless(g gray) gray {
	return seamAxis(seamAxis(g, 1, 0), 0, 1)
}

func seamAxis(g gray, dx, dy int) gray {
	mu := mean(g)
	out := gray{g.w, g.h, make([]float64, len(g.v))}
	for y := range g.h {
		for x := range g.w {
			t := (float64(x*dx+y*dy) + 0.5) / float64(g.w*dx+g.h*dy)
			d := math.Min(1, math.Min(t, 1-t)/0.22)
			w := d * d * (3 - 2*d)

			shifted := g.at(x+dx*g.w/2, y+dy*g.h/2)
			v := (g.v[y*g.w+x]-mu)*w + (shifted-mu)*(1-w)
			out.v[y*g.w+x] = math.Min(1, math.Max(0, mu+v/math.Sqrt(w*w+(1-w)*(1-w))))
		}
	}
	return out
}

func save(path string, g gray, size int) error {
	img := image.NewGray(image.Rect(0, 0, size, size))
	for i, v := range g.v {
		img.SetGray(i%size, i/size, color.Gray{uint8(math.Round(v * 255))})
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
