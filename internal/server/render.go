package server

import (
	"encoding/json"
	"fmt"
	"image/png"
	"net/http"

	"texxer/internal/graph"
	"texxer/internal/img"
	"texxer/internal/nodes"
	"texxer/internal/presets"
)

type renderRequest struct {
	Graph *graph.Graph `json:"graph"`
	Node  string       `json:"node"`
	Size  int          `json:"size"`
	Seed  int          `json:"seed"`
}

func (s *Server) handleRender(w http.ResponseWriter, r *http.Request) {
	var req renderRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Graph == nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("missing graph"))
		return
	}

	g := req.Graph
	if g.Size == 0 {
		g.Size = 512
	}
	g.Dir = s.dir

	size := min(max(req.Size, 16), 2048)
	if req.Size == 0 {
		size = 256
	}

	ref := req.Node
	if ref == "" {
		ref = g.Output
	}

	ev := graph.NewEvaluator(g, nodes.Evals(), presets.Load)
	ev.SetSeed(req.Seed)
	ev.UseCache(s.cache)

	ctx := graph.Ctx{W: size, H: size, Tiling: img.TileBoth, Scale: float64(size) / float64(g.Size)}
	b, err := ev.Eval(ref, ctx)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}

	lo, hi := valueRange(b)
	w.Header().Set("X-Texxer-Range", fmt.Sprintf("%g,%g", lo, hi))
	if lo < -0.01 || hi > 1.01 {
		b = normalized(b, lo, hi)
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	png.Encode(w, b.Image())
}

func valueRange(b *img.Buffer) (lo, hi float32) {
	lo, hi = b.Pix[0], b.Pix[0]
	for _, v := range b.Pix {
		lo, hi = min(lo, v), max(hi, v)
	}
	return lo, hi
}

// normalized maps distance fields and other out-of-range data into 0..1 for display.
func normalized(b *img.Buffer, lo, hi float32) *img.Buffer {
	out := img.New(b.W, b.H, b.C, b.Tiling)
	span := max(hi-lo, 1e-6)

	for i, v := range b.Pix {
		out.Pix[i] = (v - lo) / span
	}
	return out
}
