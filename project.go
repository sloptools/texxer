package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"texxer/internal/mesh"
)

// A project is a .texxer file anywhere, with the models it paints. Painted masks live
// in a folder beside it. Old projects sat next to their one model with the same name.

type ProjectFile struct {
	Path string `json:"path"`
	// JSON is empty for a new project.
	JSON string `json:"json"`
	// Model is the GLB an old one-model project belongs to, or the GLB that was opened.
	Model string `json:"model"`
}

// LoadProject opens a project, or for a GLB the project beside it, new if there is none.
func (a *App) LoadProject(path string) (*ProjectFile, error) {
	f := &ProjectFile{Path: path}
	if strings.EqualFold(filepath.Ext(path), ".glb") {
		f.Model = path
		f.Path = strings.TrimSuffix(path, filepath.Ext(path)) + ".texxer"
	} else if glb := strings.TrimSuffix(path, filepath.Ext(path)) + ".glb"; exists(glb) {
		f.Model = glb
	}

	data, err := os.ReadFile(f.Path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	f.JSON = string(data)
	return f, nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (a *App) SaveProject(path, json string) error {
	return os.WriteFile(path, []byte(json), 0o644)
}

func (a *App) PickProject() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Open project or model",
		Filters: []runtime.FileFilter{{DisplayName: "texxer project or glTF binary", Pattern: "*.texxer;*.glb"}},
	})
}

func (a *App) NewProject() (string, error) {
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "New project",
		DefaultFilename: "project.texxer",
		Filters:         []runtime.FileFilter{{DisplayName: "texxer project", Pattern: "*.texxer"}},
	})
}

func (a *App) PickModels() ([]string, error) {
	return runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Add models",
		Filters: []runtime.FileFilter{{DisplayName: "glTF binary (*.glb)", Pattern: "*.glb"}},
	})
}

// FileHashes tells for each model key whether its files changed, empty when one is missing.
func (a *App) FileHashes(keys []string) map[string]string {
	out := map[string]string{}
	for _, k := range keys {
		out[k], _ = fileHash(k)
	}
	return out
}

// Painted masks live as PNGs in a folder beside the project.
func paintPath(project, id string) (string, error) {
	if project == "" || id == "" || strings.ContainsAny(id, `/\.`) {
		return "", errors.New("bad paint id")
	}
	return filepath.Join(project+".paint", id+".png"), nil
}

func (a *App) SavePaint(project, id, pngBase64 string) error {
	path, err := paintPath(project, id)
	if err != nil {
		return err
	}
	data, err := base64.StdEncoding.DecodeString(pngBase64)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// MovePaint carries a model's painted masks from one layout over to another. The
// model may have gained files since; their faces come after the old ones.
func (a *App) MovePaint(project, fromKey string, from int, toKey string, to int, ids []string) error {
	if fromKey == toKey && from == to {
		return nil
	}
	src, err := a.models.model(fromKey, from)
	if err != nil {
		return err
	}
	dst, err := a.models.model(toKey, to)
	if err != nil {
		return err
	}

	for _, id := range ids {
		file, err := paintPath(project, id)
		if err != nil {
			return err
		}
		if !exists(file) {
			continue
		}
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

// KeepOriginals keeps copies of a model's files beside the project, for layers that
// take the model's own colors: export writes texxer's texture into the files, and
// the colors must still come from before. It copies them when there are none yet or
// when refresh says the files changed outside texxer, and returns their model key.
func (a *App) KeepOriginals(project, id, key string, refresh bool) (string, error) {
	if project == "" || id == "" || strings.ContainsAny(id, `/\.`) {
		return "", errors.New("bad model id")
	}
	var kept []string
	for i, p := range keyPaths(key) {
		dst := filepath.Join(project+".paint", fmt.Sprintf("%s-original-%d.glb", id, i))
		if refresh || !exists(dst) {
			data, err := os.ReadFile(p)
			if err != nil {
				return "", err
			}
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(dst, data, 0o644); err != nil {
				return "", err
			}
		}
		kept = append(kept, dst)
	}
	return strings.Join(kept, "|"), nil
}
