package main

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"texxer/internal/grunge"
	"texxer/internal/mesh"
	"texxer/internal/palette"
	"texxer/internal/unwrap"
)

type App struct {
	ctx    context.Context
	models *modelCache
}

func NewApp() *App {
	return &App{models: newModelCache()}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// domReady is a fallback: the frontend shows the window after its first paint.
func (a *App) domReady(ctx context.Context) {
	time.AfterFunc(2*time.Second, func() { runtime.WindowShow(ctx) })
}

type ModelInfo struct {
	// Path is the model key: its files joined by "|".
	Path  string   `json:"path"`
	Parts []string `json:"parts"`
	// Extent is the bounding box size divided by its diagonal. Bake
	// positions are (p - Min) / Diag.
	Extent [3]float64 `json:"extent"`
	Min    [3]float64 `json:"min"`
	Diag   float64    `json:"diag"`
	// Overlap is the share of the model's own UV space used more than once; 1 without UVs.
	Overlap float64 `json:"overlap"`
	// Offsets move each file into place beside the others.
	Offsets [][3]float64 `json:"offsets"`
	// Hash changes whenever one of the files does.
	Hash string `json:"hash"`
}

// Open loads the files of a model key, see keyPaths.
func (a *App) Open(key string) (*ModelInfo, error) {
	l, err := a.models.get(key)
	if err != nil {
		return nil, err
	}
	hash, err := fileHash(key)
	if err != nil {
		return nil, err
	}

	m := l.model
	size := m.Max.Sub(m.Min)
	diag := max(size.Len(), 1e-9)
	info := &ModelInfo{
		Path:    key,
		Parts:   m.Parts,
		Extent:  [3]float64{size[0] / diag, size[1] / diag, size[2] / diag},
		Min:     m.Min,
		Diag:    diag,
		Overlap: unwrap.Overlap(m.Tris, 256),
		Hash:    hash,
	}
	for _, o := range l.offsets {
		info.Offsets = append(info.Offsets, [3]float64(o))
	}
	return info, nil
}

// stampDir holds the user's own stamp images, shared by all projects.
func stampDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "texxer", "stamps"), nil
}

// Stamps lists the user's stamp images by name.
func (a *App) Stamps() ([]string, error) {
	dir, err := stampDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []string{}, nil
	}
	names := []string{}
	for _, e := range entries {
		if strings.EqualFold(filepath.Ext(e.Name()), ".png") {
			names = append(names, strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))
		}
	}
	return names, err
}

// ImportStamps copies PNGs the user picks into the stamp library.
func (a *App) ImportStamps() ([]string, error) {
	paths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Import stamps",
		Filters: []runtime.FileFilter{{DisplayName: "PNG images (*.png)", Pattern: "*.png"}},
	})
	if err != nil || len(paths) == 0 {
		return nil, err
	}
	dir, err := stampDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(dir, filepath.Base(p)), data, 0o644); err != nil {
			return nil, err
		}
	}
	return a.Stamps()
}

func (a *App) GrungeMaps() []string {
	return grunge.Names()
}

// Palette returns colors as 0..255 triplets: Quake's for "quake", otherwise
// from a palette file the user picks. Empty when the dialog is cancelled.
func (a *App) Palette(kind string) ([][3]uint8, error) {
	var pal []palette.Color
	if kind == "quake" {
		pal = palette.Quake()
	} else {
		path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
			Title:   "Load palette",
			Filters: []runtime.FileFilter{{DisplayName: "Palette (*.lmp, *.png)", Pattern: "*.lmp;*.png"}},
		})
		if err != nil || path == "" {
			return nil, err
		}
		if pal, err = palette.Load(path); err != nil {
			return nil, err
		}
	}

	out := make([][3]uint8, len(pal))
	for i, c := range pal {
		for k := range 3 {
			out[i][k] = uint8(c[k]*255 + 0.5)
		}
	}
	return out, nil
}

// VariantTextures are a variant's finished textures as base64 PNGs.
type VariantTextures struct {
	Name     string `json:"name"`
	Color    string `json:"color"`
	Emissive string `json:"emissive"`
}

// Export writes the textures into the model's files in place, and as PNGs beside the
// first one, named after the model; variants get PNGs of their own. An empty emissive
// leaves it out. It returns the files' new hash.
func (a *App) Export(key string, layout int, name, pngBase64, emissiveBase64 string, variants []VariantTextures, emissiveStrength float64) (string, error) {
	t := mesh.Textures{EmissiveStrength: emissiveStrength}
	var err error
	if t.Color, err = base64.StdEncoding.DecodeString(pngBase64); err != nil {
		return "", err
	}
	if emissiveBase64 != "" {
		if t.Emissive, err = base64.StdEncoding.DecodeString(emissiveBase64); err != nil {
			return "", err
		}
	}

	paths := keyPaths(key)
	outs := make([][]byte, len(paths))
	for i := range paths {
		glb, err := a.models.glb(key, layout, i)
		if err != nil {
			return "", err
		}
		if outs[i], err = withTexture(glb, t); err != nil {
			return "", err
		}
	}

	base := filepath.Join(filepath.Dir(paths[0]), fileName(name))
	if err := writePNGs(base, pngBase64, emissiveBase64); err != nil {
		return "", err
	}
	for _, v := range variants {
		if err := writePNGs(base+"_"+fileName(v.Name), v.Color, v.Emissive); err != nil {
			return "", err
		}
	}
	for i, p := range paths {
		if err := os.WriteFile(p, outs[i], 0o644); err != nil {
			return "", err
		}
	}
	a.models.touch(key)
	return fileHash(key)
}

// writePNGs writes base.png and, if there is one, base_emissive.png.
func writePNGs(base, color, emissive string) error {
	for _, f := range []struct{ path, data string }{{base + ".png", color}, {base + "_emissive.png", emissive}} {
		if f.data == "" {
			continue
		}
		png, err := base64.StdEncoding.DecodeString(f.data)
		if err != nil {
			return err
		}
		if err := os.WriteFile(f.path, png, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// fileName makes a name safe for a file: lower case, words joined by underscores.
func fileName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		case b.Len() > 0 && !strings.HasSuffix(b.String(), "_"):
			b.WriteByte('_')
		}
	}
	if s := strings.TrimSuffix(b.String(), "_"); s != "" {
		return s
	}
	return "variant"
}
