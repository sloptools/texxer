package nodes

import (
	"errors"
	"fmt"
	stdcolor "image/color"
	"os"
	"strings"
	"sync"

	"texxer/internal/graph"
	"texxer/internal/img"
	"texxer/internal/mesh"
)

var meshDef = Def{
	Eval:     meshNode,
	Category: "mesh",
	Outputs:  []string{"out", "ao", "edges", "cavity", "up", "height", "normal", "part", "selected", "mask"},
	Params: []Param{
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
	},
}

// meshNode bakes a glTF model's geometry into maps in its UV layout:
// ambient occlusion, distance to convex edges (for wear) and concave edges
// (for dirt), upward facing, world height, normals, a random value per part
// and the parts named in select. The main output is the model's own texture.
// Widths and distances are fractions of the model's size.
func meshNode(e *graph.Env) (*img.Buffer, error) {
	p := e.Params
	path := p.String("path", "")
	if path == "" {
		return nil, errors.New("no model path set")
	}
	path = e.Path(path)

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	var sel []string
	for s := range strings.SplitSeq(p.String("select", ""), ",") {
		if s = strings.TrimSpace(s); s != "" {
			sel = append(sel, s)
		}
	}

	opts := mesh.BakeOptions{
		W: e.Ctx.W, H: e.Ctx.H,
		Material:    p.String("material", ""),
		AORays:      p.Int("ao_rays", 32),
		AODist:      p.Float("ao_dist", 0.15),
		EdgeWidth:   p.Float("edge_width", 0.01),
		CavityWidth: p.Float("cavity_width", 0.015),
		EdgeAngle:   p.Float("edge_angle", 30),
		Select:      sel,
		Padding:     p.Int("padding", 8),
	}

	key := fmt.Sprintf("%s|%d|%d|%+v", path, info.ModTime().UnixNano(), info.Size(), opts)
	b, err := bakeCached(key, path, opts)
	if err != nil {
		return nil, err
	}

	t := e.Ctx.Tiling
	gray := func(src []float32) *img.Buffer {
		return &img.Buffer{W: opts.W, H: opts.H, C: 1, Tiling: t, Pix: src}
	}
	for name, src := range map[string][]float32{
		"ao": b.maps.AO, "edges": b.maps.Edges, "cavity": b.maps.Cavity, "up": b.maps.Up,
		"height": b.maps.Height, "part": b.maps.Part, "selected": b.maps.Selected, "mask": b.maps.Mask,
	} {
		e.SetOutput(name, gray(src))
	}

	normal := img.New(opts.W, opts.H, 4, t)
	for i := range opts.W * opts.H {
		copy(normal.Pix[i*4:], b.maps.Normal[i*3:i*3+3])
		normal.Pix[i*4+3] = 1
	}
	e.SetOutput("normal", normal)

	return modelTexture(b.model, opts.W, opts.H, t, p.String("filter", "nearest") == "linear"), nil
}

type baked struct {
	model *mesh.Model
	maps  *mesh.Maps
}

var (
	bakesMu sync.Mutex
	bakes   = map[string]*baked{}
	// Recent keys, oldest first, so only a few bakes stay in memory.
	bakeOrder []string
)

const keepBakes = 8

func bakeCached(key, path string, opts mesh.BakeOptions) (*baked, error) {
	bakesMu.Lock()
	b, ok := bakes[key]
	bakesMu.Unlock()
	if ok {
		return b, nil
	}

	model, err := mesh.LoadGLB(path)
	if err != nil {
		return nil, err
	}
	b = &baked{model, mesh.Bake(model, opts)}

	bakesMu.Lock()
	defer bakesMu.Unlock()
	bakes[key] = b
	bakeOrder = append(bakeOrder, key)
	if len(bakeOrder) > keepBakes {
		delete(bakes, bakeOrder[0])
		bakeOrder = bakeOrder[1:]
	}
	return b, nil
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
