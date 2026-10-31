package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Materials are saved folders of layers, kept as JSON beside the stamps so every
// project can use them.

func materialDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "texxer", "materials"), nil
}

// Materials returns the user's materials as JSON, by file name.
func (a *App) Materials() (map[string]string, error) {
	dir, err := materialDir()
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	for _, e := range entries {
		if !strings.EqualFold(filepath.Ext(e.Name()), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		out[strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))] = string(data)
	}
	return out, err
}

// SaveMaterial stores a material under its name and returns the file name it got.
func (a *App) SaveMaterial(name, json string) (string, error) {
	dir, err := materialDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	file := fileName(name)
	return file, os.WriteFile(filepath.Join(dir, file+".json"), []byte(json), 0o644)
}

func (a *App) DeleteMaterial(file string) error {
	dir, err := materialDir()
	if err != nil {
		return err
	}
	if strings.ContainsAny(file, `/\.`) {
		return errors.New("bad material name")
	}
	return os.Remove(filepath.Join(dir, file+".json"))
}
