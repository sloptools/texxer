package server

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Only asset types the editor previews can be read, so the endpoint doesn't expose other files.
var assetTypes = map[string]bool{".glb": true, ".png": true, ".jpg": true, ".jpeg": true}

// handleFile serves a model or image a graph refers to, relative to the graph directory.
func (s *Server) handleFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if !assetTypes[strings.ToLower(filepath.Ext(path))] {
		writeError(w, http.StatusBadRequest, errors.New("unsupported file type"))
		return
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(s.dir, path)
	}

	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, path)
}

// handlePutFile stores a painted mask. Only PNGs inside the graph directory can be written.
func (s *Server) handlePutFile(w http.ResponseWriter, r *http.Request) {
	rel := filepath.Clean(r.URL.Query().Get("path"))
	if !strings.EqualFold(filepath.Ext(rel), ".png") || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		writeError(w, http.StatusBadRequest, errors.New("only PNG files inside the graph directory can be written"))
		return
	}

	data, err := io.ReadAll(io.LimitReader(r.Body, 64<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if _, err := png.DecodeConfig(bytes.NewReader(data)); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("not a PNG: %w", err))
		return
	}

	path := filepath.Join(s.dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
