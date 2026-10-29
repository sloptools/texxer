// Package server serves the editor and its HTTP API.
package server

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"

	"texxer/internal/graph"
)

type Server struct {
	dir   string
	cache *graph.Cache
	mux   *http.ServeMux
}

// New serves graphs from dir and the editor from static.
func New(dir string, static fs.FS) *Server {
	s := &Server{
		dir:   dir,
		cache: graph.NewCache(1 << 30),
		mux:   http.NewServeMux(),
	}

	s.mux.HandleFunc("GET /api/catalog", s.handleCatalog)
	s.mux.HandleFunc("GET /api/graphs", s.handleListGraphs)
	s.mux.HandleFunc("GET /api/graphs/{name}", s.handleGetGraph)
	s.mux.HandleFunc("PUT /api/graphs/{name}", s.handlePutGraph)
	s.mux.HandleFunc("POST /api/render", s.handleRender)
	s.mux.HandleFunc("GET /api/file", s.handleFile)
	s.mux.HandleFunc("GET /api/models", s.handleListModels)
	s.mux.Handle("/", http.FileServerFS(static))

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) handleCatalog(w http.ResponseWriter, r *http.Request) {
	c, err := catalog(s.dir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, c)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Print(err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
