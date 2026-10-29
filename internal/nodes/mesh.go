package nodes

import (
	"errors"
	"fmt"
	stdcolor "image/color"
	"math"
	"os"
	"strings"
	"sync"

	"texxer/internal/graph"
	"texxer/internal/img"
	"texxer/internal/mesh"
)

// meshMaps are the baked maps the mesh node outputs, which layers wire into materials and masks.
var meshMaps = []string{"ao", "edges", "cavity", "up", "height", "normal", "part", "partid", "selected", "mask"}

var meshDef = Def{
	Eval:     meshNode,
	Category: "mesh",
	Outputs:  append([]string{"out"}, meshMaps...),
	Params:   meshParams,
}

var meshParams = []Param{
	text("path", ""),
	text("material", ""),
	text("select", ""),
	integer("ao_rays", 32, 0, 256),
	num("ao_dist", 0.15, 0.001, 1),
	num("edge_width", 0.01, 0.0005, 0.2),
	num("cavity_width", 0.015, 0.0005, 0.2),
	num("edge_angle", 30, 1, 90),
	enum("filter", "nearest", "nearest", "linear"),
	integer("padding", 8, 0, 64),
}

// meshNode bakes a glTF model's geometry into maps in its UV layout:
// ambient occlusion, distance to convex edges (for wear) and concave edges
// (for dirt), upward facing, world height, normals, a random value and an
// id per part, and the parts named in select. The main output is the
// model's own texture. Widths and distances are fractions of the model's
// size. Maps are baked once at the graph's size and scaled for previews.
func meshNode(e *graph.Env) (*img.Buffer, error) {
	p := e.Params
	path := p.String("path", "")
	if path == "" {
		return nil, errors.New("no model path set")
	}
	path = e.Path(path)

	var sel []string
	for s := range strings.SplitSeq(p.String("select", ""), ",") {
		if s = strings.TrimSpace(s); s != "" {
			sel = append(sel, s)
		}
	}

	// Bake at the design size, or larger when an upstream node renders bigger.
	bw := max(e.Ctx.W, int(math.Round(float64(e.Ctx.W)/e.Ctx.Scale)))
	bh := max(e.Ctx.H, int(math.Round(float64(e.Ctx.H)/e.Ctx.Scale)))

	opts := mesh.BakeOptions{
		W: bw, H: bh,
		Material:    p.String("material", ""),
		AORays:      p.Int("ao_rays", 32),
		AODist:      p.Float("ao_dist", 0.15),
		EdgeWidth:   p.Float("edge_width", 0.01),
		CavityWidth: p.Float("cavity_width", 0.015),
		EdgeAngle:   p.Float("edge_angle", 30),
		Select:      sel,
		Padding:     p.Int("padding", 8),
	}

	model, err := loadModel(path)
	if err != nil {
		return nil, err
	}
	maps, err := bakeCached(path, model, opts)
	if err != nil {
		return nil, err
	}

	w, h, t := e.Ctx.W, e.Ctx.H, e.Ctx.Tiling
	for name, src := range map[string][]float32{
		"ao": maps.AO, "edges": maps.Edges, "cavity": maps.Cavity, "up": maps.Up,
		"height": maps.Height, "part": maps.Part, "selected": maps.Selected, "mask": maps.Mask,
	} {
		e.SetOutput(name, shrink(src, bw, bh, 1, w, h, t))
	}
	e.SetOutput("partid", nearest(maps.PartID, bw, bh, w, h, t))

	normal := img.New(bw, bh, 4, t)
	for i := range bw * bh {
		copy(normal.Pix[i*4:], maps.Normal[i*3:i*3+3])
		normal.Pix[i*4+3] = 1
	}
	e.SetOutput("normal", shrink(normal.Pix, bw, bh, 4, w, h, t))

	return modelTexture(model.m, w, h, t, p.String("filter", "nearest") == "linear"), nil
}

// shrink box-filters a map of c channels from bw x bh down to w x h.
func shrink(src []float32, bw, bh, c, w, h int, t img.Tiling) *img.Buffer {
	if bw == w && bh == h {
		return &img.Buffer{W: w, H: h, C: c, Tiling: t, Pix: src}
	}

	out := img.New(w, h, c, t)
	img.Rows(h, func(y int) {
		y0, y1 := y*bh/h, max(y*bh/h+1, (y+1)*bh/h)
		for x := range w {
			x0, x1 := x*bw/w, max(x*bw/w+1, (x+1)*bw/w)
			n := float32((y1 - y0) * (x1 - x0))

			for ch := range c {
				var sum float32
				for sy := y0; sy < y1; sy++ {
					for sx := x0; sx < x1; sx++ {
						sum += src[(sy*bw+sx)*c+ch]
					}
				}
				out.Pix[(y*w+x)*c+ch] = sum / n
			}
		}
	})
	return out
}

func nearest(src []float32, bw, bh, w, h int, t img.Tiling) *img.Buffer {
	out := img.New(w, h, 1, t)
	for y := range h {
		for x := range w {
			out.Pix[y*w+x] = src[(y*bh/h)*bw+x*bw/w]
		}
	}
	return out
}

type loadedModel struct {
	m   *mesh.Model
	key string
}

var (
	modelsMu sync.Mutex
	models   = map[string]loadedModel{}

	bakesMu sync.Mutex
	bakes   = map[string]*mesh.Maps{}
	// Recent keys, oldest first, so only a few bakes stay in memory.
	bakeOrder []string
)

const keepBakes = 8

// loadModel parses a model once and again only when the file changes.
func loadModel(path string) (loadedModel, error) {
	info, err := os.Stat(path)
	if err != nil {
		return loadedModel{}, err
	}
	key := fmt.Sprintf("%s|%d|%d", path, info.ModTime().UnixNano(), info.Size())

	modelsMu.Lock()
	lm, ok := models[path]
	modelsMu.Unlock()
	if ok && lm.key == key {
		return lm, nil
	}

	m, err := mesh.LoadGLB(path)
	if err != nil {
		return loadedModel{}, err
	}
	lm = loadedModel{m, key}

	modelsMu.Lock()
	models[path] = lm
	modelsMu.Unlock()
	return lm, nil
}

// modelParts lists the part names of a model, for masks that pick parts by name.
func modelParts(path string) ([]string, error) {
	lm, err := loadModel(path)
	if err != nil {
		return nil, err
	}
	return lm.m.Parts, nil
}

func bakeCached(path string, lm loadedModel, opts mesh.BakeOptions) (*mesh.Maps, error) {
	key := fmt.Sprintf("%s|%+v", lm.key, opts)

	bakesMu.Lock()
	maps, ok := bakes[key]
	bakesMu.Unlock()
	if ok {
		return maps, nil
	}

	maps = mesh.Bake(lm.m, opts)

	bakesMu.Lock()
	defer bakesMu.Unlock()
	bakes[key] = maps
	bakeOrder = append(bakeOrder, key)
	if len(bakeOrder) > keepBakes {
		delete(bakes, bakeOrder[0])
		bakeOrder = bakeOrder[1:]
	}
	return maps, nil
}

// modelTexture resamples the model's texture to the bake size, nearest by default since such textures are often pixel art.
func modelTexture(m *mesh.Model, w, h int, t img.Tiling, linear bool) *img.Buffer {
	out := img.New(w, h, 4, t)
	if m.Texture == nil {
		for i := range w * h {
			copy(out.Pix[i*4:], []float32{0.5, 0.5, 0.5, 1})
		}
		return out
	}

	src := m.Texture.Bounds()
	if linear {
		full := img.FromImage(m.Texture)
		img.Rows(h, func(y int) {
			for x := range w {
				u := (float64(x) + 0.5) / float64(w) * float64(full.W)
				v := (float64(y) + 0.5) / float64(h) * float64(full.H)
				for ch := range 4 {
					out.Pix[(y*w+x)*4+ch] = full.Sample(u, v, ch)
				}
			}
		})
		return out
	}

	img.Rows(h, func(y int) {
		for x := range w {
			sx := src.Min.X + x*src.Dx()/w
			sy := src.Min.Y + y*src.Dy()/h
			c := stdcolor.NRGBAModel.Convert(m.Texture.At(sx, sy)).(stdcolor.NRGBA)
			copy(out.Pix[(y*w+x)*4:], []float32{float32(c.R) / 255, float32(c.G) / 255, float32(c.B) / 255, float32(c.A) / 255})
		}
	})
	return out
}
