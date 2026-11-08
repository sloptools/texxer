// Command grunge turns grunge photos into small tileable masks for texxer's grunge
// folder: square crop, area downsample, large-scale shading removed, white = dirt,
// edges cross-faded so the map tiles, then equalized so that a mask's amount
// is about the share of the surface it covers, whatever the map.
//
// usage: go run ./tools/grunge -root <dir> -list <file>, the file with one
// "name source" per line, sources under dir.
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"texxer/internal/grunge"
)

func main() {
	dir, err := grunge.Dir()
	if err != nil {
		log.Fatal(err)
	}
	out := flag.String("o", dir, "output directory")
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
		g, err := grunge.Load(src)
		if err != nil {
			log.Fatal(err)
		}

		m := process(g, *size, *flatten)
		png, err := grunge.Encode(m)
		if err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(*out, name+".png"), png, 0o644); err != nil {
			log.Fatal(err)
		}
		fmt.Println(name)
	}
}

func process(g grunge.Gray, size int, flatten float64) grunge.Gray {
	m := grunge.Downsample(g, size)

	// Vignettes and lighting gradients in the photos would repeat as blotches.
	low := blur(m, size/8)
	for i := range m.V {
		m.V[i] -= low.V[i] * flatten
	}
	stretch(m, 0.01, 0.995)

	// Dirt is the minority of the surface.
	if mean(m) > 0.5 {
		for i := range m.V {
			m.V[i] = 1 - m.V[i]
		}
	}
	return grunge.Equalize(seamless(grunge.Equalize(m)))
}

// blur is three wrapping box passes, close to a gaussian.
func blur(g grunge.Gray, radius int) grunge.Gray {
	out := g
	for range 3 {
		out = boxPass(boxPass(out, radius, 1, 0), radius, 0, 1)
	}
	return out
}

func boxPass(g grunge.Gray, r, dx, dy int) grunge.Gray {
	out := grunge.Gray{W: g.W, H: g.H, V: make([]float64, len(g.V))}
	for y := range g.H {
		for x := range g.W {
			var s float64
			for k := -r; k <= r; k++ {
				s += g.At(x+k*dx, y+k*dy)
			}
			out.V[y*g.W+x] = s / float64(2*r+1)
		}
	}
	return out
}

func stretch(g grunge.Gray, lo, hi float64) {
	sorted := slices.Clone(g.V)
	slices.Sort(sorted)
	a := sorted[int(lo*float64(len(sorted)-1))]
	b := sorted[int(hi*float64(len(sorted)-1))]
	for i, v := range g.V {
		g.V[i] = math.Min(1, math.Max(0, (v-a)/math.Max(b-a, 1e-6)))
	}
}

func mean(g grunge.Gray) float64 {
	var s float64
	for _, v := range g.V {
		s += v
	}
	return s / float64(len(g.V))
}

// seamless makes g tile, one axis at a time: the borders fade into a copy
// shifted by half along that axis, whose own seam then sits in the middle where
// the original shows. The blend keeps the variance, so the fade doesn't show as
// a washed-out band.
func seamless(g grunge.Gray) grunge.Gray {
	return seamAxis(seamAxis(g, 1, 0), 0, 1)
}

func seamAxis(g grunge.Gray, dx, dy int) grunge.Gray {
	mu := mean(g)
	out := grunge.Gray{W: g.W, H: g.H, V: make([]float64, len(g.V))}
	for y := range g.H {
		for x := range g.W {
			t := (float64(x*dx+y*dy) + 0.5) / float64(g.W*dx+g.H*dy)
			d := math.Min(1, math.Min(t, 1-t)/0.22)
			w := d * d * (3 - 2*d)

			shifted := g.At(x+dx*g.W/2, y+dy*g.H/2)
			v := (g.V[y*g.W+x]-mu)*w + (shifted-mu)*(1-w)
			out.V[y*g.W+x] = math.Min(1, math.Max(0, mu+v/math.Sqrt(w*w+(1-w)*(1-w))))
		}
	}
	return out
}
