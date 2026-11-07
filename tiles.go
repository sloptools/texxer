package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"texxer/internal/mesh"
)

// A tile is a model texxer makes itself: a wall of a few meters for a tileable
// texture, its GLB in a folder beside the project. Its file name tells it apart.

const tileExt = ".tile.glb"

func isTile(key string) bool {
	return strings.HasSuffix(key, tileExt) && !strings.Contains(key, "|")
}

// MakeTile writes the GLB of a tile of w×h meters and returns its path.
func (a *App) MakeTile(project, id string, w, h float64) (string, error) {
	if project == "" || id == "" || strings.ContainsAny(id, `/\.`) {
		return "", errors.New("bad tile id")
	}
	if w <= 0 || h <= 0 {
		return "", errors.New("bad tile size")
	}
	glb, err := mesh.TileGLB(w, h)
	if err != nil {
		return "", err
	}
	path := filepath.Join(project+".tiles", id+tileExt)
	// The same size keeps the file, so its bakes stay.
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, glb) {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	return path, os.WriteFile(path, glb, 0o644)
}

// ExportTile writes a tile's textures as PNGs into dir, cut to the tile's part of
// the texture. It returns the tile's file hash.
func (a *App) ExportTile(key, name, dir, pngBase64, emissiveBase64 string, variants []VariantTextures) (string, error) {
	if !isTile(key) {
		return "", errors.New("not a tile")
	}
	l, err := a.models.get(key)
	if err != nil {
		return "", err
	}
	size := l.model.Max.Sub(l.model.Min)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	base := filepath.Join(dir, fileName(name))
	files := []struct{ path, data string }{{base + ".png", pngBase64}, {base + "_emissive.png", emissiveBase64}}
	for _, v := range variants {
		files = append(files, struct{ path, data string }{base + "_" + fileName(v.Name) + ".png", v.Color})
		files = append(files, struct{ path, data string }{base + "_" + fileName(v.Name) + "_emissive.png", v.Emissive})
	}
	for _, f := range files {
		if f.data == "" {
			continue
		}
		if err := writeCropped(f.path, f.data, size[0], size[1]); err != nil {
			return "", err
		}
	}
	return fileHash(key)
}

// writeCropped keeps the top left w×h of a square PNG, the longer side all of it.
func writeCropped(path, data string, w, h float64) error {
	raw, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return err
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	s := float64(img.Bounds().Dx())
	l := max(w, h)
	rect := image.Rect(0, 0, int(s*w/l+0.5), int(s*h/l+0.5))
	out := image.NewNRGBA(rect)
	draw.Draw(out, rect, img, img.Bounds().Min, draw.Src)

	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
