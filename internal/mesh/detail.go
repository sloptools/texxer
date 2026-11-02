package mesh

import (
	"image"
	"image/color"
	"math"
	"sync"
)

// Detail is a material's own maps: its base color, and normal and occlusion maps,
// mostly baked from a high-poly model. They follow the model's own UVs (Tri.Src),
// whatever layout texxer paints on.
type Detail struct {
	// Color is the base color texture, Factor the color it is multiplied by.
	Color  image.Image
	Factor [4]float64
	// Normal is in tangent space, +Y up as glTF has it. Any map may be nil.
	Normal    image.Image
	Occlusion image.Image

	once sync.Once
	// curvature per normal map pixel: positive on ridges, negative in grooves.
	curvature []float32
}

// tangents is a face's tangent and bitangent: the directions of the normal map's
// red and green axes on the surface. Green points up the image, against glTF's v.
func tangents(t Tri) (Vec, Vec) {
	e1, e2 := t.P[1].Sub(t.P[0]), t.P[2].Sub(t.P[0])
	du1, dv1 := t.Src[1][0]-t.Src[0][0], t.Src[1][1]-t.Src[0][1]
	du2, dv2 := t.Src[2][0]-t.Src[0][0], t.Src[2][1]-t.Src[0][1]
	r := du1*dv2 - du2*dv1
	if math.Abs(r) < 1e-12 {
		n := e1.Cross(e2).Norm()
		return basis(n)
	}
	tangent := e1.Scale(dv2).Sub(e2.Scale(dv1)).Scale(1 / r)
	bitangent := e2.Scale(du1).Sub(e1.Scale(du2)).Scale(-1 / r)
	return tangent, bitangent
}

// frame makes the face's tangents orthonormal around the surface normal n.
func frame(t, b, n Vec) (Vec, Vec) {
	t = t.Sub(n.Scale(n.Dot(t))).Norm()
	side := 1.0
	if n.Cross(t).Dot(b) < 0 {
		side = -1
	}
	return t, n.Cross(t).Scale(side)
}

func pixel(img image.Image, u, v float64) (r, g, b float64) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	x := min(w-1, max(0, int(fract(u)*float64(w))))
	y := min(h-1, max(0, int(fract(v)*float64(h))))
	cr, cg, cb, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
	return float64(cr) / 65535, float64(cg) / 65535, float64(cb) / 65535
}

// normalAt is the detail normal at a UV in tangent space.
func (d *Detail) normalAt(u, v float64) Vec {
	r, g, b := pixel(d.Normal, u, v)
	return Vec{r*2 - 1, g*2 - 1, b*2 - 1}
}

// HasDetail tells whether the material has a normal or occlusion map to bake from.
func (d *Detail) HasDetail() bool {
	return d.Normal != nil || d.Occlusion != nil
}

func (d *Detail) colorAt(u, v float64) [3]float64 {
	c := [3]float64{d.Factor[0], d.Factor[1], d.Factor[2]}
	if d.Color != nil {
		r, g, b := pixel(d.Color, u, v)
		c = [3]float64{c[0] * r, c[1] * g, c[2] * b}
	}
	return c
}

func (d *Detail) occlusionAt(u, v float64) float64 {
	r, _, _ := pixel(d.Occlusion, u, v)
	return r
}

// curvatureAt is how much the normal map bends at a UV: the divergence of its
// normals, positive on ridges and bevels, negative in grooves.
func (d *Detail) curvatureAt(u, v float64) float64 {
	d.once.Do(d.measure)
	b := d.Normal.Bounds()
	w, h := b.Dx(), b.Dy()
	x := min(w-1, max(0, int(fract(u)*float64(w))))
	y := min(h-1, max(0, int(fract(v)*float64(h))))
	return float64(d.curvature[y*w+x])
}

func (d *Detail) measure() {
	b := d.Normal.Bounds()
	w, h := b.Dx(), b.Dy()
	nx := make([]float32, w*h)
	ny := make([]float32, w*h)
	for y := range h {
		for x := range w {
			r, g, _, _ := d.Normal.At(b.Min.X+x, b.Min.Y+y).RGBA()
			nx[y*w+x] = float32(r)/65535*2 - 1
			ny[y*w+x] = float32(g)/65535*2 - 1
		}
	}

	d.curvature = make([]float32, w*h)
	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			i := y*w + x
			// Green is up, so it grows towards smaller rows.
			d.curvature[i] = (nx[i+1] - nx[i-1] + ny[i-w] - ny[i+w]) / 2
		}
	}
}

// apply shapes a bake sample with the detail maps: their occlusion darkens the AO,
// their ridges and grooves add to edges and cavities, and their normals replace the
// surface's. strength 1 takes them as they are.
func (d *Detail) apply(s sample, strength, ao, convex, concave float64) (float64, float64, float64, Vec) {
	u, v := s.src[0], s.src[1]
	w := min(strength, 1)
	n := s.n
	if d.Occlusion != nil {
		ao *= 1 - w + w*d.occlusionAt(u, v)
	}
	if d.Normal != nil {
		c := d.curvatureAt(u, v) * 6 * strength
		convex = max(convex, min(1, c))
		concave = max(concave, min(1, -c))

		t, b := frame(s.tan, s.bit, s.n)
		tn := d.normalAt(u, v)
		detailed := t.Scale(tn[0]).Add(b.Scale(tn[1])).Add(s.n.Scale(tn[2])).Norm()
		n = s.n.Add(detailed.Sub(s.n).Scale(w)).Norm()
	}
	return ao, convex, concave, n
}

// RemapDetails carries the detail maps of the faces over to new UVs, at size: the
// normals turned into each face's new tangent space, so the maps fit the new layout.
// from and to are the same faces with the old and the new UVs. Either result is nil
// when no face has that map.
func RemapDetails(details []*Detail, from, to []Tri, size int) (normal, occlusion *image.NRGBA) {
	for _, t := range from {
		if t.Detail < 0 || t.Detail >= len(details) {
			continue
		}
		if details[t.Detail].Normal != nil && normal == nil {
			normal = image.NewNRGBA(image.Rect(0, 0, size, size))
		}
		if details[t.Detail].Occlusion != nil && occlusion == nil {
			occlusion = image.NewNRGBA(image.Rect(0, 0, size, size))
		}
	}
	if normal == nil && occlusion == nil {
		return nil, nil
	}
	flat := color.NRGBA{128, 128, 255, 255}
	white := color.NRGBA{255, 255, 255, 255}
	for i := range size * size {
		if normal != nil {
			normal.SetNRGBA(i%size, i/size, flat)
		}
		if occlusion != nil {
			occlusion.SetNRGBA(i%size, i/size, white)
		}
	}

	s := float64(size)
	for i, t := range to[:min(len(to), len(from))] {
		src := from[i]
		if src.Detail < 0 || src.Detail >= len(details) {
			continue
		}
		d := details[src.Detail]
		n := src.P[1].Sub(src.P[0]).Cross(src.P[2].Sub(src.P[0])).Norm()
		oldT, oldB := tangents(src)
		oldT, oldB = frame(oldT, oldB, n)
		dst := t
		dst.Src = t.UV
		newT, newB := tangents(dst)
		newT, newB = frame(newT, newB, n)

		var u, v [3]float64
		for c := range 3 {
			u[c], v[c] = t.UV[c][0]*s, t.UV[c][1]*s
		}
		area := (u[1]-u[0])*(v[2]-v[0]) - (u[2]-u[0])*(v[1]-v[0])
		if math.Abs(area) < 1e-12 {
			continue
		}
		x0 := max(0, int(math.Floor(min(u[0], u[1], u[2])))-1)
		x1 := min(size-1, int(math.Ceil(max(u[0], u[1], u[2]))))
		y0 := max(0, int(math.Floor(min(v[0], v[1], v[2])))-1)
		y1 := min(size-1, int(math.Ceil(max(v[0], v[1], v[2]))))
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				px, py := float64(x)+0.5, float64(y)+0.5
				b0 := ((u[1]-px)*(v[2]-py) - (u[2]-px)*(v[1]-py)) / area
				b1 := ((u[2]-px)*(v[0]-py) - (u[0]-px)*(v[2]-py)) / area
				b2 := 1 - b0 - b1
				// A texel's worth of slack, so neighboring faces meet without gaps.
				if b0 < -0.02 || b1 < -0.02 || b2 < -0.02 {
					continue
				}
				su := src.Src[0][0]*b0 + src.Src[1][0]*b1 + src.Src[2][0]*b2
				sv := src.Src[0][1]*b0 + src.Src[1][1]*b1 + src.Src[2][1]*b2

				if d.Normal != nil && normal != nil {
					tn := d.normalAt(su, sv)
					world := oldT.Scale(tn[0]).Add(oldB.Scale(tn[1])).Add(n.Scale(tn[2]))
					nt := Vec{world.Dot(newT), world.Dot(newB), world.Dot(n)}.Norm()
					normal.SetNRGBA(x, y, color.NRGBA{byte255(nt[0]*0.5 + 0.5), byte255(nt[1]*0.5 + 0.5), byte255(nt[2]*0.5 + 0.5), 255})
				}
				if d.Occlusion != nil && occlusion != nil {
					o := byte255(d.occlusionAt(su, sv))
					occlusion.SetNRGBA(x, y, color.NRGBA{o, o, o, 255})
				}
			}
		}
	}
	return normal, occlusion
}

func byte255(v float64) uint8 {
	return uint8(min(255, max(0, math.Round(v*255))))
}
