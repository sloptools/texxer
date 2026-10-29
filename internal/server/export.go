package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"image/png"
	"net/http"
	"os"
	"path/filepath"

	"texxer/internal/graph"
	"texxer/internal/img"
	"texxer/internal/mesh"
	"texxer/internal/nodes"
	"texxer/internal/presets"
)

// handleExport renders a graph at its own size and returns its model as a
// GLB with the texture in it.
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	var req renderRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Graph == nil {
		writeError(w, http.StatusBadRequest, errors.New("missing graph"))
		return
	}

	g := req.Graph
	if g.Size == 0 {
		g.Size = 512
	}
	g.Dir = s.dir

	model, material, ok := nodes.ModelOf(g)
	if !ok {
		writeError(w, http.StatusBadRequest, errors.New("graph has no layers or mesh node with a model"))
		return
	}

	ev := graph.NewEvaluator(g, nodes.Evals(), presets.Library(s.dir))
	ev.SetSeed(req.Seed)
	ev.UseCache(s.cache)
	b, err := ev.Eval(g.Output, graph.Ctx{W: g.Size, H: g.Size, Tiling: img.TileBoth, Scale: 1})
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}

	var tex bytes.Buffer
	if err := png.Encode(&tex, b.Image()); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if !filepath.IsAbs(model) {
		model = filepath.Join(s.dir, model)
	}
	src, err := os.ReadFile(model)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	out, err := mesh.WithTexture(src, tex.Bytes(), material)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}

	w.Header().Set("Content-Type", "model/gltf-binary")
	w.Write(out)
}
