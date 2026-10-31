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
	Path  string   `json:"path"`
	Name  string   `json:"name"`
	Parts []string `json:"parts"`
	// Extent is the bounding box size divided by its diagonal. Bake
	// positions are (p - Min) / Diag.
	Extent [3]float64 `json:"extent"`
	Min    [3]float64 `json:"min"`
	Diag   float64    `json:"diag"`
	// Project is the saved project JSON next to the model, empty if there is none.
	Project string `json:"project"`
}

func (a *App) PickModel() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Open model",
		Filters: []runtime.FileFilter{{DisplayName: "glTF binary (*.glb)", Pattern: "*.glb"}},
	})
}

func (a *App) Open(path string) (*ModelInfo, error) {
	m, err := a.models.get(path)
	if err != nil {
		return nil, err
	}

	size := m.Max.Sub(m.Min)
	diag := max(size.Len(), 1e-9)
	info := &ModelInfo{
		Path:   path,
		Name:   strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
		Parts:  m.Parts,
		Extent: [3]float64{size[0] / diag, size[1] / diag, size[2] / diag},
		Min:    m.Min,
		Diag:   diag,
	}

	data, err := os.ReadFile(projectPath(path))
	if err == nil {
		info.Project = string(data)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
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

func (a *App) Save(modelPath, project string) error {
	return os.WriteFile(projectPath(modelPath), []byte(project), 0o644)
}

func projectPath(model string) string {
	return strings.TrimSuffix(model, filepath.Ext(model)) + ".texxer"
}

// Painted masks live as PNGs in a folder next to the project.
func paintPath(model, id string) (string, error) {
	if id == "" || strings.ContainsAny(id, `/\.`) {
		return "", errors.New("bad paint id")
	}
	return filepath.Join(projectPath(model)+".paint", id+".png"), nil
}

func (a *App) SavePaint(modelPath, id, pngBase64 string) error {
	path, err := paintPath(modelPath, id)
	if err != nil {
		return err
	}
	png, err := base64.StdEncoding.DecodeString(pngBase64)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, png, 0o644)
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

// Export writes the textures as PNGs and the model with them as a GLB. An
// empty target asks where to put it, an empty emissive leaves it out. It
// returns the GLB path, empty when cancelled.
func (a *App) Export(modelPath, pngBase64, emissiveBase64 string, emissiveStrength float64, target string) (string, error) {
	if target == "" {
		name := strings.TrimSuffix(filepath.Base(modelPath), filepath.Ext(modelPath)) + "_texxer.glb"
		var err error
		target, err = runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:            "Export model",
			DefaultDirectory: filepath.Dir(modelPath),
			DefaultFilename:  name,
			Filters:          []runtime.FileFilter{{DisplayName: "glTF binary (*.glb)", Pattern: "*.glb"}},
		})
		if err != nil || target == "" {
			return "", err
		}
	}

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
	glb, err := os.ReadFile(modelPath)
	if err != nil {
		return "", err
	}
	out, err := withTexture(glb, t)
	if err != nil {
		return "", err
	}

	base := strings.TrimSuffix(target, filepath.Ext(target))
	if err := os.WriteFile(base+".png", t.Color, 0o644); err != nil {
		return "", err
	}
	if t.Emissive != nil {
		if err := os.WriteFile(base+"_emissive.png", t.Emissive, 0o644); err != nil {
			return "", err
		}
	}
	return target, os.WriteFile(target, out, 0o644)
}
