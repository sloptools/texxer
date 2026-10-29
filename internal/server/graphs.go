package server

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"texxer/internal/graph"
)

var validName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func (s *Server) handleListGraphs(w http.ResponseWriter, r *http.Request) {
	files, err := filepath.Glob(filepath.Join(s.dir, "*.json"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	names := []string{}
	for _, f := range files {
		names = append(names, strings.TrimSuffix(filepath.Base(f), ".json"))
	}
	slices.Sort(names)

	writeJSON(w, names)
}

func (s *Server) handleGetGraph(w http.ResponseWriter, r *http.Request) {
	path, ok := s.graphPath(w, r)
	if !ok {
		return
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (s *Server) handlePutGraph(w http.ResponseWriter, r *http.Request) {
	path, ok := s.graphPath(w, r)
	if !ok {
		return
	}

	data, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if _, err := graph.Parse(data); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) graphPath(w http.ResponseWriter, r *http.Request) (string, bool) {
	name := r.PathValue("name")
	if !validName.MatchString(name) {
		writeError(w, http.StatusBadRequest, errors.New("invalid graph name"))
		return "", false
	}
	return filepath.Join(s.dir, name+".json"), true
}
