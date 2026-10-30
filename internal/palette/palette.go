package palette

import (
	"cmp"
	_ "embed"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Color [3]float32

// MedianCut reduces pix to at most n representative colors. It reorders pix.
func MedianCut(pix []Color, n int) []Color {
	boxes := [][]Color{pix}

	for len(boxes) < n {
		i, ch := widest(boxes)
		if i < 0 {
			break
		}

		b := boxes[i]
		slices.SortFunc(b, func(p, q Color) int { return cmp.Compare(p[ch], q[ch]) })

		mid := len(b) / 2
		boxes[i] = b[:mid]
		boxes = append(boxes, b[mid:])
	}

	pal := make([]Color, len(boxes))
	for i, b := range boxes {
		pal[i] = mean(b)
	}
	return pal
}

// Load reads a Quake .lmp palette, or all distinct colors of an image reduced to 256 if needed.
func Load(path string) ([]Color, error) {
	if strings.EqualFold(filepath.Ext(path), ".lmp") {
		return loadLmp(path)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	seen := map[Color]bool{}
	r := src.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			cr, cg, cb, _ := src.At(x, y).RGBA()
			seen[Color{float32(cr) / 0xffff, float32(cg) / 0xffff, float32(cb) / 0xffff}] = true
		}
	}

	pal := make([]Color, 0, len(seen))
	for c := range seen {
		pal = append(pal, c)
	}
	if len(pal) > 256 {
		pal = MedianCut(pal, 256)
	}
	return pal, nil
}

// loadLmp reads 256 RGB triplets.
func loadLmp(path string) ([]Color, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 768 {
		return nil, fmt.Errorf("%s: palette too short", path)
	}

	return parseLmp(data), nil
}

func parseLmp(data []byte) []Color {
	pal := make([]Color, 256)
	for i := range pal {
		pal[i] = Color{float32(data[i*3]) / 255, float32(data[i*3+1]) / 255, float32(data[i*3+2]) / 255}
	}
	return pal
}

func Nearest(pal []Color, c Color) Color {
	best, bestDist := pal[0], float32(1e9)

	for _, p := range pal {
		dr, dg, db := p[0]-c[0], p[1]-c[1], p[2]-c[2]
		// Weighted towards green, roughly matching perceived brightness.
		d := 0.3*dr*dr + 0.59*dg*dg + 0.11*db*db
		if d < bestDist {
			best, bestDist = p, d
		}
	}
	return best
}

func widest(boxes [][]Color) (index, channel int) {
	index, channel = -1, 0
	var best float32

	for i, b := range boxes {
		if len(b) < 2 {
			continue
		}

		lo, hi := b[0], b[0]
		for _, c := range b {
			for ch := range 3 {
				lo[ch] = min(lo[ch], c[ch])
				hi[ch] = max(hi[ch], c[ch])
			}
		}

		for ch := range 3 {
			if r := hi[ch] - lo[ch]; r > best {
				best, index, channel = r, i, ch
			}
		}
	}
	return index, channel
}

func mean(b []Color) Color {
	var sum [3]float64
	for _, c := range b {
		for ch := range 3 {
			sum[ch] += float64(c[ch])
		}
	}

	n := float64(len(b))
	return Color{float32(sum[0] / n), float32(sum[1] / n), float32(sum[2] / n)}
}

//go:embed quake.lmp
var quakeLmp []byte

// Quake returns the 256 colors of Quake's palette.
func Quake() []Color {
	return parseLmp(quakeLmp)
}
