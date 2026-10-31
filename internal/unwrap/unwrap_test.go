package unwrap

import (
	"math"
	"testing"
	"time"

	"texxer/internal/mesh"
)

func quad(a, b, c, d mesh.Vec) []mesh.Tri {
	return []mesh.Tri{{P: [3]mesh.Vec{a, b, c}}, {P: [3]mesh.Vec{a, c, d}}}
}

func cube() []mesh.Tri {
	v := func(x, y, z float64) mesh.Vec { return mesh.Vec{x, y, z} }
	var t []mesh.Tri
	t = append(t, quad(v(0, 0, 1), v(1, 0, 1), v(1, 1, 1), v(0, 1, 1))...) // +z
	t = append(t, quad(v(1, 0, 0), v(0, 0, 0), v(0, 1, 0), v(1, 1, 0))...) // -z
	t = append(t, quad(v(1, 0, 1), v(1, 0, 0), v(1, 1, 0), v(1, 1, 1))...) // +x
	t = append(t, quad(v(0, 0, 0), v(0, 0, 1), v(0, 1, 1), v(0, 1, 0))...) // -x
	t = append(t, quad(v(0, 1, 1), v(1, 1, 1), v(1, 1, 0), v(0, 1, 0))...) // +y
	t = append(t, quad(v(0, 0, 0), v(1, 0, 0), v(1, 0, 1), v(0, 0, 1))...) // -y
	return t
}

// cylinder is open-ended unless caps; seen from outside, faces wind counterclockwise.
func cylinder(segments int, caps bool) []mesh.Tri {
	var t []mesh.Tri
	at := func(i int, y float64) mesh.Vec {
		a := 2 * math.Pi * float64(i) / float64(segments)
		return mesh.Vec{math.Cos(a), y, -math.Sin(a)}
	}
	for i := range segments {
		t = append(t, quad(at(i, 0), at(i+1, 0), at(i+1, 3), at(i, 3))...)
		if caps {
			t = append(t, mesh.Tri{P: [3]mesh.Vec{{0, 3, 0}, at(i, 3), at(i+1, 3)}})
			t = append(t, mesh.Tri{P: [3]mesh.Vec{{0, 0, 0}, at(i+1, 0), at(i, 0)}})
		}
	}
	return t
}

func sphere(rings, segments int) []mesh.Tri {
	at := func(r, s int) mesh.Vec {
		th := math.Pi * float64(r) / float64(rings)
		ph := 2 * math.Pi * float64(s) / float64(segments)
		return mesh.Vec{math.Sin(th) * math.Cos(ph), math.Cos(th), -math.Sin(th) * math.Sin(ph)}
	}
	var t []mesh.Tri
	for r := range rings {
		for s := range segments {
			a, b, c, d := at(r, s), at(r+1, s), at(r+1, s+1), at(r, s+1)
			if r > 0 {
				t = append(t, mesh.Tri{P: [3]mesh.Vec{a, b, d}})
			}
			if r < rings-1 {
				t = append(t, mesh.Tri{P: [3]mesh.Vec{b, c, d}})
			}
		}
	}
	return t
}

// check tests what every layout must hold: inside the texture, no face mirrored, no
// texel covered twice, and one density everywhere (within spread).
func check(t *testing.T, tris []mesh.Tri, o Options, spread float64) Result {
	t.Helper()
	res := Unwrap(tris, o)
	if res.Density <= 0 {
		t.Fatalf("density %v", res.Density)
	}
	s := float64(o.Size)

	worst := 1.0
	for i, tri := range tris {
		uv := res.UV[i]
		for _, p := range uv {
			if p[0] < 0 || p[1] < 0 || p[0] > 1 || p[1] > 1 {
				t.Fatalf("face %d outside the texture: %v", i, uv)
			}
		}
		area := tri.P[1].Sub(tri.P[0]).Cross(tri.P[2].Sub(tri.P[0])).Len() / 2
		if area < 1e-9 {
			continue
		}
		signed := ((uv[1][0]-uv[0][0])*(uv[2][1]-uv[0][1]) - (uv[2][0]-uv[0][0])*(uv[1][1]-uv[0][1])) / 2
		if signed >= 0 {
			t.Fatalf("face %d is mirrored or flat in UV: %v", i, signed)
		}
		r := -signed * s * s / (area * res.Density * res.Density)
		worst = max(worst, r, 1/r)
	}
	if worst > spread {
		t.Errorf("texel density varies by %.2f×, want at most %.2f×", worst, spread)
	}

	cover := make([]int, o.Size*o.Size)
	for i := range cover {
		cover[i] = -1
	}
	for f, uv := range res.UV {
		for _, texel := range raster(uv, o.Size) {
			if cover[texel] >= 0 {
				t.Fatalf("faces %d and %d share texel %d", cover[texel], f, texel)
			}
			cover[texel] = f
		}
	}
	t.Logf("%d charts, %.1f texels per unit, %.0f%% filled", res.Charts, res.Density, res.Fill*100)
	return res
}

// raster lists the texels whose centers lie strictly inside the face.
func raster(uv [3][2]float64, size int) []int {
	s := float64(size)
	lo, hi := [2]float64{1, 1}, [2]float64{0, 0}
	for _, p := range uv {
		lo = [2]float64{min(lo[0], p[0]), min(lo[1], p[1])}
		hi = [2]float64{max(hi[0], p[0]), max(hi[1], p[1])}
	}
	edge := func(a, b [2]float64, x, y float64) float64 {
		return (b[0]-a[0])*(y-a[1]) - (b[1]-a[1])*(x-a[0])
	}
	var out []int
	for y := int(lo[1] * s); y <= int(hi[1]*s) && y < size; y++ {
		for x := int(lo[0] * s); x <= int(hi[0]*s) && x < size; x++ {
			px, py := (float64(x)+0.5)/s, (float64(y)+0.5)/s
			e0, e1, e2 := edge(uv[0], uv[1], px, py), edge(uv[1], uv[2], px, py), edge(uv[2], uv[0], px, py)
			if e0 < 0 && e1 < 0 && e2 < 0 || e0 > 0 && e1 > 0 && e2 > 0 {
				out = append(out, y*size+x)
			}
		}
	}
	return out
}

func TestCube(t *testing.T) {
	res := check(t, cube(), Defaults(256), 1.05)
	if res.Charts != 6 {
		t.Errorf("%d charts, want 6", res.Charts)
	}

	// Walls stand upright: higher up in the world is higher up in the image.
	front := res.UV[0]
	if !(front[2][1] < front[1][1]) {
		t.Errorf("front face is not upright: %v", front)
	}
	// And they're laid along the pixel grid.
	for _, p := range front {
		for _, q := range front {
			du, dv := math.Abs(p[0]-q[0]), math.Abs(p[1]-q[1])
			if du > 1e-9 && dv > 1e-9 && math.Abs(du-dv) > 1e-9 {
				t.Errorf("front face is turned: %v", front)
			}
		}
	}
}

func TestCylinder(t *testing.T) {
	res := check(t, cylinder(24, true), Defaults(256), 1.05)
	if res.Charts > 8 {
		t.Errorf("%d charts, want at most 8", res.Charts)
	}
}

func TestSphere(t *testing.T) {
	check(t, sphere(12, 24), Defaults(256), 3)
}

func TestShip(t *testing.T) {
	m, err := mesh.LoadGLB("../../examples/models/ship.glb")
	if err != nil {
		t.Skip(err)
	}
	start := time.Now()
	res := check(t, m.Tris, Defaults(512), 3)
	t.Logf("%d faces in %v", len(m.Tris), time.Since(start))
	// Blender's smart project with packing fills 0.24 here.
	if res.Fill < 0.3 {
		t.Errorf("fill %.2f, want at least 0.3", res.Fill)
	}
}
