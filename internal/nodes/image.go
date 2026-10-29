package nodes

import (
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"sync"
	"time"

	"texxer/internal/graph"
	"texxer/internal/img"
)

var imageDef = Def{
	Eval:     imageNode,
	Category: "generator",
	Params: []Param{
		text("path", ""),
		num("repeat", 1, 0.1, 16),
		flag("gray", false),
	},
}

// imageNode loads a PNG or JPEG, such as a scanned grunge mask, and tiles
// it repeat times across the buffer. Paths are relative to the graph file.
func imageNode(e *graph.Env) (*img.Buffer, error) {
	path := e.Params.String("path", "")
	if path == "" {
		return nil, errors.New("no image path set")
	}

	src, err := loadImage(e.Path(path))
	if err != nil {
		return nil, err
	}

	rep := max(e.Params.Float("repeat", 1), 0.01)
	sx := float64(src.W) * rep / float64(e.Ctx.W)
	sy := float64(src.H) * rep / float64(e.Ctx.H)

	// Average the source pixels under each output pixel when shrinking, so fine grunge doesn't alias.
	fx, fy := max(1, int(math.Ceil(sx))), max(1, int(math.Ceil(sy)))

	c := 4
	if e.Params.Bool("gray", false) {
		c = 1
	}
	out := img.New(e.Ctx.W, e.Ctx.H, c, e.Ctx.Tiling)

	img.Rows(e.Ctx.H, func(y int) {
		for x := range e.Ctx.W {
			var px [4]float32
			for dy := range fy {
				for dx := range fx {
					u := (float64(x) + (float64(dx)+0.5)/float64(fx)) * sx
					v := (float64(y) + (float64(dy)+0.5)/float64(fy)) * sy
					for ch := range 4 {
						px[ch] += src.Sample(u, v, ch)
					}
				}
			}

			n := float32(fx * fy)
			i := y*e.Ctx.W + x
			if c == 1 {
				out.Pix[i] = (0.2126*px[0] + 0.7152*px[1] + 0.0722*px[2]) / n
				continue
			}
			for ch := range 4 {
				out.Pix[i*4+ch] = px[ch] / n
			}
		}
	})
	return out, nil
}

type cachedImage struct {
	mod time.Time
	buf *img.Buffer
}

var (
	imagesMu sync.Mutex
	images   = map[string]cachedImage{}
)

// loadImage decodes an image once and again only when the file changes.
func loadImage(path string) (*img.Buffer, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	imagesMu.Lock()
	c, ok := images[path]
	imagesMu.Unlock()
	if ok && c.mod.Equal(info.ModTime()) {
		return c.buf, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoded, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	buf := img.FromImage(decoded)
	imagesMu.Lock()
	images[path] = cachedImage{info.ModTime(), buf}
	imagesMu.Unlock()
	return buf, nil
}
