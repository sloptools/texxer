package mesh

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTile(t *testing.T) {
	glb, err := TileGLB(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "wall.tile.glb")
	if err := os.WriteFile(path, glb, 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadGLB(path)
	if err != nil {
		t.Fatal(err)
	}
	if m.Max != (Vec{2, 1, 0}) || len(m.Tris) != 2 {
		t.Fatalf("tile is %v with %d faces", m.Max, len(m.Tris))
	}

	// The top half of the texture, covered to its border, with no edges there.
	maps := Bake(m, BakeOptions{W: 32, H: 32, AORays: 8, AODist: 0.15, EdgeWidth: 0.05, CavityWidth: 0.05, EdgeAngle: 30, Seamless: true})
	for y := range 16 {
		for x := range 32 {
			i := y*32 + x
			if maps.Mask[i] != 1 || maps.Edges[i] != 0 {
				t.Fatalf("texel %d,%d: coverage %v, edges %v", x, y, maps.Mask[i], maps.Edges[i])
			}
		}
	}
}
