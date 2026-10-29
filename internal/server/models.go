package server

import (
	"io/fs"
	"net/http"
	"path/filepath"
	"slices"
	"strings"

	"texxer/internal/mesh"
)

type modelInfo struct {
	Path      string   `json:"path"`
	Materials []string `json:"materials"`
	// Material is the one with the most triangles, the usual paint target.
	Material string `json:"material"`
}

// handleListModels lists the .glb files under the graph directory.
func (s *Server) handleListModels(w http.ResponseWriter, r *http.Request) {
	out := []modelInfo{}

	err := filepath.WalkDir(s.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != s.dir && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".glb") {
			return nil
		}

		m, err := mesh.LoadGLB(path)
		if err != nil {
			// Unreadable models are left out rather than failing the list.
			return nil
		}
		rel, _ := filepath.Rel(s.dir, path)
		out = append(out, describe(filepath.ToSlash(rel), m))
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	slices.SortFunc(out, func(a, b modelInfo) int { return strings.Compare(a.Path, b.Path) })
	writeJSON(w, out)
}

func describe(path string, m *mesh.Model) modelInfo {
	count := map[string]int{}
	for _, t := range m.Tris {
		count[t.Material]++
	}

	info := modelInfo{Path: path, Materials: []string{}}
	for name, n := range count {
		info.Materials = append(info.Materials, name)
		if n > count[info.Material] || info.Material == "" {
			info.Material = name
		}
	}
	slices.Sort(info.Materials)
	return info
}
