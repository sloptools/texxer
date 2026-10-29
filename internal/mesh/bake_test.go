package mesh

import "testing"

// cube builds a unit cube whose six faces each fill one cell of a 3x2 UV grid.
func cube() *Model {
	faces := [6][4]Vec{
		{{0, 0, 1}, {1, 0, 1}, {1, 1, 1}, {0, 1, 1}},
		{{1, 0, 0}, {0, 0, 0}, {0, 1, 0}, {1, 1, 0}},
		{{1, 0, 1}, {1, 0, 0}, {1, 1, 0}, {1, 1, 1}},
		{{0, 0, 0}, {0, 0, 1}, {0, 1, 1}, {0, 1, 0}},
		{{0, 1, 1}, {1, 1, 1}, {1, 1, 0}, {0, 1, 0}},
		{{0, 0, 0}, {1, 0, 0}, {1, 0, 1}, {0, 0, 1}},
	}

	m := &Model{Parts: []string{"cube"}, Min: Vec{0, 0, 0}, Max: Vec{1, 1, 1}}
	for f, q := range faces {
		n := q[1].Sub(q[0]).Cross(q[2].Sub(q[0])).Norm()
		u0, v0 := float64(f%3)/3, float64(f/3)/2
		uv := [4][2]float64{{u0, v0}, {u0 + 1.0/3, v0}, {u0 + 1.0/3, v0 + 0.5}, {u0, v0 + 0.5}}

		for _, t := range [2][3]int{{0, 1, 2}, {0, 2, 3}} {
			m.Tris = append(m.Tris, Tri{
				P:  [3]Vec{q[t[0]], q[t[1]], q[t[2]]},
				N:  [3]Vec{n, n, n},
				UV: [3][2]float64{uv[t[0]], uv[t[1]], uv[t[2]]},
			})
		}
	}
	return m
}

func TestBakeCube(t *testing.T) {
	m := cube()
	maps := Bake(m, BakeOptions{W: 48, H: 32, AORays: 16, AODist: 0.5, EdgeWidth: 0.05, CavityWidth: 0.05, EdgeAngle: 30})

	for i, v := range maps.Mask {
		if v != 1 {
			t.Fatalf("texel %d not covered", i)
		}
	}

	// Nothing occludes a convex cube's outside.
	for i, v := range maps.AO {
		if v < 0.99 {
			t.Fatalf("texel %d: ao %v on an open cube", i, v)
		}
	}

	// Face centers are far from edges, face borders are on them.
	center := 8*48 + 8
	border := 0*48 + 8
	if maps.Edges[center] != 0 || maps.Edges[border] < 0.5 {
		t.Errorf("edges: center %v, border %v", maps.Edges[center], maps.Edges[border])
	}
	for i, v := range maps.Cavity {
		if v != 0 {
			t.Fatalf("texel %d: cavity %v on a convex cube", i, v)
		}
	}

	// The top face (index 4) points up.
	if up := maps.Up[(16+8)*48+16+8]; up < 0.99 {
		t.Errorf("top face up = %v", up)
	}
}

func TestOcclusionInsideBox(t *testing.T) {
	b := NewBVH(cube().Tris)
	if !b.Occluded(Vec{0.5, 0.5, 0.5}, Vec{0, 1, 0}, 10) {
		t.Error("ray from inside the cube should hit it")
	}
	if b.Occluded(Vec{0.5, 2, 0.5}, Vec{0, 1, 0}, 10) {
		t.Error("ray away from the cube should miss")
	}
}
