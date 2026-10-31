package mesh

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWithUVs(t *testing.T) {
	path := "../../examples/models/ship.glb"
	glb, err := os.ReadFile(path)
	if err != nil {
		t.Skip(err)
	}
	m, err := LoadGLB(path)
	if err != nil {
		t.Fatal(err)
	}

	uvs := make([][3][2]float64, len(m.Tris))
	for i := range uvs {
		f := float64(i) / float64(len(uvs))
		uvs[i] = [3][2]float64{{f, 0.1}, {f, 0.5}, {f + 0.0001, 0.9}}
	}
	out, err := WithUVs(glb, uvs)
	if err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(t.TempDir(), "out.glb")
	if err := os.WriteFile(file, out, 0o644); err != nil {
		t.Fatal(err)
	}
	back, err := LoadGLB(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Tris) != len(m.Tris) {
		t.Fatalf("%d faces, want %d", len(back.Tris), len(m.Tris))
	}
	for i, tri := range back.Tris {
		for c := range 3 {
			if tri.P[c].Sub(m.Tris[i].P[c]).Len() > 1e-6 {
				t.Fatalf("face %d moved", i)
			}
			if d := tri.UV[c][0] - uvs[i][c][0] + tri.UV[c][1] - uvs[i][c][1]; d > 1e-6 || d < -1e-6 {
				t.Fatalf("face %d corner %d has UV %v, want %v", i, c, tri.UV[c], uvs[i][c])
			}
		}
	}
	t.Logf("%d bytes before, %d after", len(glb), len(out))
}
