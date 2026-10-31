package mesh

import (
	"image"
	"image/color"
	"testing"
)

// worldNormal reads a normal map at a UV and turns it into world space for the face.
func worldNormal(t Tri, img image.Image, u, v float64) Vec {
	d := &Detail{Normal: img}
	n := t.P[1].Sub(t.P[0]).Cross(t.P[2].Sub(t.P[0])).Norm()
	ta, bi := tangents(t)
	ta, bi = frame(ta, bi, n)
	tn := d.normalAt(u, v)
	return ta.Scale(tn[0]).Add(bi.Scale(tn[1])).Add(n.Scale(tn[2])).Norm()
}

func TestRemapDetailsTurnsNormals(t *testing.T) {
	// A unit quad facing +z; its normal map leans every normal towards +x.
	p := [4]Vec{{0, 0, 0}, {1, 0, 0}, {1, 1, 0}, {0, 1, 0}}
	src := [4][2]float64{{0, 1}, {1, 1}, {1, 0}, {0, 0}}
	// The same quad turned a quarter in the new layout.
	dst := [4][2]float64{{1, 1}, {1, 0}, {0, 0}, {0, 1}}
	quad := func(uv [4][2]float64) []Tri {
		a := Tri{P: [3]Vec{p[0], p[1], p[2]}, UV: [3][2]float64{uv[0], uv[1], uv[2]}}
		b := Tri{P: [3]Vec{p[0], p[2], p[3]}, UV: [3][2]float64{uv[0], uv[2], uv[3]}}
		for _, tri := range []*Tri{&a, &b} {
			tri.Src = tri.UV
		}
		return []Tri{a, b}
	}
	from, to := quad(src), quad(dst)

	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for i := range 64 {
		img.SetNRGBA(i%8, i/8, color.NRGBA{218, 128, 218, 255})
	}
	details := []*Detail{{Normal: img}}
	normal, _ := RemapDetails(details, from, to, 8)
	if normal == nil {
		t.Fatal("no normal map")
	}

	before := worldNormal(from[0], img, 0.6, 0.6)
	after := worldNormal(to[0], normal, 0.6, 0.6)
	if before.Sub(after).Len() > 0.05 {
		t.Errorf("normal turned: %.2f before, %.2f after", before, after)
	}
	if before[0] < 0.5 {
		t.Errorf("normal %.2f should lean towards +x", before)
	}
}
