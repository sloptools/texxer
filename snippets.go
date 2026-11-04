package main

import (
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Snippets are cuts from photos, deskewed and styled in the frontend. A library
// is the project's (`<project>.snippets/`) or a global category in the user
// config dir; each holds the photos, their cuts as JSON beside them, and the
// finished snippets as PNGs.

type Photo struct {
	ID   string `json:"id"`
	File string `json:"file"`
	// Cuts as the frontend keeps them, JSON.
	Cuts string `json:"cuts"`
}

func categoryRoot() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "texxer", "snippets"), nil
}

// libDir is the project's library for an empty category.
func libDir(project, category string) (string, error) {
	if category == "" {
		if project == "" {
			return "", errors.New("no project")
		}
		return project + ".snippets", nil
	}
	root, err := categoryRoot()
	if err != nil {
		return "", err
	}
	name := fileName(category)
	if name == "" {
		return "", errors.New("bad category")
	}
	return filepath.Join(root, name), nil
}

func safeID(id string) error {
	if id == "" || strings.ContainsAny(id, `/\.`) {
		return errors.New("bad id")
	}
	return nil
}

func (a *App) SnippetCategories() ([]string, error) {
	root, err := categoryRoot()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return []string{}, nil
	}
	names := []string{}
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, err
}

func (a *App) AddCategory(name string) (string, error) {
	dir, err := libDir("", name)
	if err != nil {
		return "", err
	}
	return filepath.Base(dir), os.MkdirAll(dir, 0o755)
}

// Photos lists a library's photos with their cuts, oldest first.
func (a *App) Photos(project, category string) ([]Photo, error) {
	dir, err := libDir(project, category)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Join(dir, "photos"))
	if errors.Is(err, os.ErrNotExist) {
		return []Photo{}, nil
	}
	photos := []Photo{}
	for _, e := range entries {
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if !imageExt[ext] {
			continue
		}
		id := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		cuts, _ := os.ReadFile(filepath.Join(dir, "photos", id+".json"))
		photos = append(photos, Photo{ID: id, File: e.Name(), Cuts: string(cuts)})
	}
	slices.SortFunc(photos, func(p, q Photo) int { return strings.Compare(p.ID, q.ID) })
	return photos, err
}

// AddPhotos copies photos into the library, asking for them when none are given.
// ids are the names they get, one per path.
func (a *App) AddPhotos(project, category string, paths, ids []string) ([]string, error) {
	if len(paths) == 0 {
		var err error
		paths, err = runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
			Title:   "Add photos",
			Filters: []runtime.FileFilter{{DisplayName: "Images", Pattern: "*.png;*.jpg;*.jpeg;*.webp"}},
		})
		if err != nil {
			return nil, err
		}
	}
	dir, err := libDir(project, category)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(dir, "photos"), 0o755); err != nil {
		return nil, err
	}

	added := []string{}
	for i, p := range paths {
		ext := strings.ToLower(filepath.Ext(p))
		if !imageExt[ext] || i >= len(ids) || safeID(ids[i]) != nil {
			continue
		}
		if err := copyFile(p, filepath.Join(dir, "photos", ids[i]+ext)); err != nil {
			return added, err
		}
		added = append(added, ids[i])
	}
	return added, nil
}

func copyFile(from, to string) error {
	src, err := os.Open(from)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(to)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}
	return dst.Close()
}

func (a *App) SaveCuts(project, category, photo, cuts string) error {
	dir, err := libDir(project, category)
	if err != nil {
		return err
	}
	if err := safeID(photo); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "photos", photo+".json"), []byte(cuts), 0o644)
}

// DeletePhoto removes a photo, its cuts and their snippets.
func (a *App) DeletePhoto(project, category, photo, file string, snippets []string) error {
	dir, err := libDir(project, category)
	if err != nil {
		return err
	}
	if safeID(photo) != nil || filepath.Base(file) != file {
		return errors.New("bad photo")
	}
	for _, id := range snippets {
		if safeID(id) == nil {
			os.Remove(filepath.Join(dir, "snippets", id+".png"))
		}
	}
	os.Remove(filepath.Join(dir, "photos", photo+".json"))
	return os.Remove(filepath.Join(dir, "photos", file))
}

func (a *App) SaveSnippet(project, category, id, pngBase64 string) error {
	dir, err := libDir(project, category)
	if err != nil {
		return err
	}
	if err := safeID(id); err != nil {
		return err
	}
	data, err := base64.StdEncoding.DecodeString(pngBase64)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "snippets"), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "snippets", id+".png"), data, 0o644)
}

func (a *App) DeleteSnippet(project, category, id string) error {
	dir, err := libDir(project, category)
	if err != nil {
		return err
	}
	if err := safeID(id); err != nil {
		return err
	}
	return os.Remove(filepath.Join(dir, "snippets", id+".png"))
}

// UseSnippet copies a category's snippet into the project, which stamps read from.
func (a *App) UseSnippet(project, category, id string) error {
	if category == "" {
		return nil
	}
	from, err := libDir(project, category)
	if err != nil {
		return err
	}
	to, err := libDir(project, "")
	if err != nil {
		return err
	}
	if err := safeID(id); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(to, "snippets"), 0o755); err != nil {
		return err
	}
	return copyFile(filepath.Join(from, "snippets", id+".png"), filepath.Join(to, "snippets", id+".png"))
}

// snippetFile is a photo (kind "photo", by file name) or a snippet (by id) of a library.
func snippetFile(project, category, kind, name string) (string, error) {
	dir, err := libDir(project, category)
	if err != nil {
		return "", err
	}
	if kind == "photo" {
		if name == "" || filepath.Base(name) != name || !imageExt[strings.ToLower(filepath.Ext(name))] {
			return "", errors.New("bad photo")
		}
		return filepath.Join(dir, "photos", name), nil
	}
	if err := safeID(name); err != nil {
		return "", err
	}
	return filepath.Join(dir, "snippets", name+".png"), nil
}
