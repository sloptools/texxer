package server

import (
	"errors"
	"net/http"
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
