package main

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"

	"texxer/internal/mesh"
)

// A layout is the UV set a project paints on: 0 for the model's own UVs, else
// texxer's unwrap for that texture size.

type UnwrapStats struct {
	Charts int `json:"charts"`
	// Fill is the share of the texture the charts cover.
	Fill float64 `json:"fill"`
	// Density is in texels per world unit.
	Density float64 `json:"density"`
}

func (a *App) Unwrap(modelPath string, size int) (UnwrapStats, error) {
	u, err := a.models.unwrap(modelPath, size)
	if err != nil {
		return UnwrapStats{}, err
	}
	return u.stats, nil
}

func (c *modelCache) glb(path string, layout int) ([]byte, error) {
	if layout == 0 {
		return os.ReadFile(path)
	}
	u, err := c.unwrap(path, layout)
	if err != nil {
		return nil, err
	}
	return u.glb, nil
}

// MovePaint carries the project's painted masks from one layout over to another.
func (a *App) MovePaint(modelPath string, from, to int) error {
	if from == to {
		return nil
	}
	src, err := a.models.model(modelPath, from)
	if err != nil {
		return err
	}
	dst, err := a.models.model(modelPath, to)
	if err != nil {
		return err
	}

	files, err := filepath.Glob(filepath.Join(projectPath(modelPath)+".paint", "*.png"))
	if err != nil {
		return err
	}
	for _, file := range files {
		if err := movePNG(file, src, dst); err != nil {
			return err
		}
	}
	return nil
}

func movePNG(file string, src, dst *mesh.Model) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}

	moved := mesh.Remap(img, src.Tris, dst.Tris, max(4, img.Bounds().Dx()/64))
	var out bytes.Buffer
	if err := png.Encode(&out, moved); err != nil {
		return err
	}
	return os.WriteFile(file, out.Bytes(), 0o644)
}
