package main

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Fonts for text stamps: the built-in ones ship with the frontend, the user's sit in
// the font folder beside the grunge maps.

var fontExts = []string{".ttf", ".otf", ".woff", ".woff2"}

// fontsDir is the user's font folder in the config dir, made if missing.
func fontsDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "texxer", "fonts")
	return dir, os.MkdirAll(dir, 0o755)
}

// Fonts lists the font files in the font folder, served at /font?file=.
func (a *App) Fonts() []string {
	out := []string{}
	dir, err := fontsDir()
	if err != nil {
		return out
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if !e.IsDir() && !strings.HasPrefix(e.Name(), ".") && slices.Contains(fontExts, strings.ToLower(filepath.Ext(e.Name()))) {
			out = append(out, e.Name())
		}
	}
	slices.Sort(out)
	return out
}

// OpenFontsFolder shows the font folder in the file manager.
func (a *App) OpenFontsFolder() error {
	dir, err := fontsDir()
	if err != nil {
		return err
	}
	return openFolder(dir)
}

func serveFont(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("file")
	dir, err := fontsDir()
	if err == nil && (name == "" || strings.ContainsAny(name, `/\`) || strings.HasPrefix(name, ".")) {
		err = errors.New("bad font name")
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.ServeFile(w, r, filepath.Join(dir, name))
}
