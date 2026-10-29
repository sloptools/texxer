package mesh

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

func TestWithTexture(t *testing.T) {
	src, err := os.ReadFile("../../examples/models/ship_unique.glb")
	if err != nil {
		t.Skip("example model missing")
	}

	tex := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for i := range tex.Pix {
		tex.Pix[i] = 200
	}
	tex.Set(0, 0, color.NRGBA{255, 0, 0, 255})
	var buf bytes.Buffer
	png.Encode(&buf, tex)

	out, err := WithTexture(src, buf.Bytes(), "albedo")
	if err != nil {
		t.Fatal(err)
	}

	// The result loads, keeps its geometry and carries the new texture.
	path := t.TempDir() + "/out.glb"
	os.WriteFile(path, out, 0o644)
	before, _ := LoadGLB("../../examples/models/ship_unique.glb")
	after, err := LoadGLB(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Tris) != len(before.Tris) || after.Tris[100].P != before.Tris[100].P {
		t.Error("geometry changed")
	}
	if b := after.Texture.Bounds(); b.Dx() != 4 {
		t.Errorf("texture is %v, want the 4x4 replacement", b)
	}
	if r, _, _, _ := after.Texture.At(0, 0).RGBA(); r>>8 != 255 {
		t.Error("texture pixels differ")
	}

	// The old texture's bytes are gone.
	if len(out) > len(src) {
		t.Errorf("export grew from %d to %d bytes with a tiny texture", len(src), len(out))
	}

	if _, err := WithTexture(src, buf.Bytes(), "nope"); err == nil {
		t.Error("expected an error for a missing material")
	}
}
