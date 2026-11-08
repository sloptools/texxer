// Package grunge holds the grunge maps: tileable 512² grays, white = dirt. Built-in
// ones are made from noise; the user's own come from a folder and are equalized on
// load like them, so a mask's amount is about the share of the surface it covers.
package grunge

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

const size = 512

var exts = []string{".png", ".jpg", ".jpeg"}

func isMap(file string) bool {
	return slices.Contains(exts, strings.ToLower(filepath.Ext(file))) && !strings.HasPrefix(file, ".")
}

// Dir is the user's grunge folder in the config dir, made if missing.
func Dir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "texxer", "grunge")
	return dir, os.MkdirAll(dir, 0o755)
}

// Names lists the built-in maps, then the ones in dir by file name, sorted. A file
// named like a built-in one takes its place.
func Names(dir string) []string {
	own := []string{}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		if !e.IsDir() && isMap(e.Name()) && builtins[name] == nil {
			own = append(own, name)
		}
	}
	slices.Sort(own)
	return append(slices.Clone(order), own...)
}

// file finds the map by that name in dir.
func file(dir, name string) (string, error) {
	if name == "" || strings.ContainsAny(name, `/\`) || strings.HasPrefix(name, ".") {
		return "", errors.New("bad map name")
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if isMap(e.Name()) && strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())) == name {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", errors.New("no such map")
}

type cached struct {
	mod time.Time
	png []byte
}

var (
	mu    sync.Mutex
	cache = map[string]cached{}
)

// PNG is the map by that name as a PNG: the user's file in dir, else the built-in one.
func PNG(dir, name string) ([]byte, error) {
	path, err := file(dir, name)
	var mod time.Time
	if err == nil {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		mod = info.ModTime()
	} else if builtins[name] == nil {
		return nil, err
	}

	mu.Lock()
	c, ok := cache[name]
	mu.Unlock()
	if ok && c.mod.Equal(mod) {
		return c.png, nil
	}

	var g Gray
	if path != "" {
		if g, err = Load(path); err != nil {
			return nil, err
		}
		g = Downsample(g, size)
	} else {
		g = builtins[name](size)
	}
	png, err := Encode(Equalize(g))
	if err != nil {
		return nil, err
	}
	mu.Lock()
	cache[name] = cached{mod, png}
	mu.Unlock()
	return png, nil
}
