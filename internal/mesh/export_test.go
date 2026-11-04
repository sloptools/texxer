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

	out, err := WithTexture(src, Textures{Color: buf.Bytes()}, "albedo")
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
	got := baseColor(t, out, "albedo")
	if b := got.Bounds(); b.Dx() != 4 {
		t.Errorf("texture is %v, want the 4x4 replacement", b)
	}
	if r, _, _, _ := got.At(0, 0).RGBA(); r>>8 != 255 {
		t.Error("texture pixels differ")
	}

	var doc map[string]any
	if d, _, err := rawGLB(out); err == nil {
		doc = d
	}
	for _, m := range doc["materials"].([]any) {
		mat := m.(map[string]any)
		if mat["name"] != "albedo" {
			continue
		}
		pbr := mat["pbrMetallicRoughness"].(map[string]any)
		if pbr["roughnessFactor"] != 1.0 || pbr["metallicFactor"] != 0.0 {
			t.Errorf("painted material should be rough and not metallic: %v", pbr)
		}
	}

	// The old texture's bytes are gone.
	if len(out) > len(src) {
		t.Errorf("export grew from %d to %d bytes with a tiny texture", len(src), len(out))
	}

	if _, err := WithTexture(src, Textures{Color: buf.Bytes()}, "nope"); err == nil {
		t.Error("expected an error for a missing material")
	}

	// An emissive texture comes with its strength as an extension.
	glow, err := WithTexture(src, Textures{Color: buf.Bytes(), Emissive: buf.Bytes(), EmissiveStrength: 3}, "albedo")
	if err != nil {
		t.Fatal(err)
	}
	doc, _, err = rawGLB(glow)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range sliceArray(doc, "materials") {
		mat := m.(map[string]any)
		if mat["name"] != "albedo" {
			continue
		}
		ext, _ := mat["extensions"].(map[string]any)
		if mat["emissiveTexture"] == nil || ext["KHR_materials_emissive_strength"] == nil {
			t.Errorf("albedo should glow: %v", mat)
		}
	}
	if len(sliceArray(doc, "images")) < 2 {
		t.Error("emissive image missing")
	}
}

// baseColor decodes the base color texture of a material in a GLB.
func baseColor(t *testing.T, glb []byte, material string) image.Image {
	t.Helper()
	doc, bin, err := rawGLB(glb)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range sliceArray(doc, "materials") {
		mat := m.(map[string]any)
		pbr, _ := mat["pbrMetallicRoughness"].(map[string]any)
		ref, _ := pbr["baseColorTexture"].(map[string]any)
		if mat["name"] != material || ref == nil {
			continue
		}
		tex := sliceArray(doc, "textures")[intField(ref, "index")]
		img := sliceArray(doc, "images")[intField(tex, "source")]
		view := sliceArray(doc, "bufferViews")[intField(img, "bufferView")]
		off := intField(view, "byteOffset")
		decoded, _, err := image.Decode(bytes.NewReader(bin[off : off+intField(view, "byteLength")]))
		if err != nil {
			t.Fatal(err)
		}
		return decoded
	}
	t.Fatalf("no base color texture on %s", material)
	return nil
}

// Textures that extensions point at survive the export, renumbered with the rest.
func TestWithTextureKeepsExtensionTextures(t *testing.T) {
	src, err := os.ReadFile("../../examples/models/ship_unique.glb")
	if err != nil {
		t.Skip("example model missing")
	}
	doc, bin, err := rawGLB(src)
	if err != nil {
		t.Fatal(err)
	}
	// One more texture, used only by an extension of the first textured material.
	images := sliceArray(doc, "images")
	textures := sliceArray(doc, "textures")
	doc["textures"] = append(textures, map[string]any{"source": float64(len(images) - 1)})
	var mat map[string]any
	for _, m := range sliceArray(doc, "materials") {
		if pbr, _ := m.(map[string]any)["pbrMetallicRoughness"].(map[string]any); pbr != nil && pbr["baseColorTexture"] != nil {
			mat = m.(map[string]any)
			break
		}
	}
	mat["extensions"] = map[string]any{"KHR_materials_specular": map[string]any{"specularTexture": map[string]any{"index": float64(len(textures))}}}
	glb, err := writeGLB(doc, bin)
	if err != nil {
		t.Fatal(err)
	}

	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	var buf bytes.Buffer
	png.Encode(&buf, img)
	out, err := WithTexture(glb, Textures{Color: buf.Bytes()}, "")
	if err != nil {
		t.Fatal(err)
	}
	after, _, _ := rawGLB(out)
	count := len(sliceArray(after, "textures"))
	eachTextureRef(sliceArray(after, "materials"), func(ref map[string]any) {
		if i := intField(ref, "index"); i >= count {
			t.Errorf("reference to texture %d, but only %d are left", i, count)
		}
	})
}

// Linked textures stay out of the file and load from beside it.
func TestWithTextureLinked(t *testing.T) {
	src, err := os.ReadFile("../../examples/models/ship_unique.glb")
	if err != nil {
		t.Skip("example model missing")
	}
	tex := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	tex.Set(0, 0, color.NRGBA{255, 0, 0, 255})
	var buf bytes.Buffer
	png.Encode(&buf, tex)

	out, err := WithTexture(src, Textures{Color: buf.Bytes(), Files: &TextureFiles{Color: "../my textures/ship.png"}}, "albedo")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) >= len(src) {
		t.Errorf("linked export is %d bytes, the source %d", len(out), len(src))
	}

	dir := t.TempDir()
	os.MkdirAll(dir+"/model", 0o755)
	os.MkdirAll(dir+"/my textures", 0o755)
	os.WriteFile(dir+"/model/ship.glb", out, 0o644)
	m, err := LoadGLB(dir + "/model/ship.glb")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range m.Details {
		if d.Color != nil {
			t.Fatal("loaded a texture that is not there yet")
		}
	}

	os.WriteFile(dir+"/my textures/ship.png", buf.Bytes(), 0o644)
	if m, err = LoadGLB(dir + "/model/ship.glb"); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range m.Details {
		if d.Color != nil && d.Color.Bounds().Dx() == 4 {
			found = true
		}
	}
	if !found {
		t.Error("linked texture not loaded")
	}
}
