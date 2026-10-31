package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"net/http"
	"path/filepath"
	"strconv"

	"texxer/internal/grunge"
	"texxer/internal/mesh"
)

// bake returns three float32 RGBA maps of size×size texels, back to back:
// AO, convex edges, concave cavity, coverage;
// position (p - min) / diagonal, part index;
// normal, island.
func (c *modelCache) bake(path string, size, layout int) ([]byte, error) {
	m, err := c.model(path, layout)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("%s|%p|%d", path, m, size)
	c.mu.Lock()
	defer c.mu.Unlock()
	if b, ok := c.bakes[key]; ok {
		return b, nil
	}

	maps := mesh.Bake(m, mesh.BakeOptions{
		W: size, H: size,
		AORays:      48,
		AORes:       min(size, 512),
		AODist:      0.15,
		EdgeWidth:   0.03,
		CavityWidth: 0.03,
		EdgeAngle:   30,
		Padding:     max(4, size/64),
	})

	n := size * size
	out := make([]byte, n*4*4*3)
	put := func(texture, texel, ch int, v float32) {
		binary.LittleEndian.PutUint32(out[((texture*n+texel)*4+ch)*4:], math.Float32bits(v))
	}
	for i := range n {
		put(0, i, 0, maps.AO[i])
		put(0, i, 1, maps.Edges[i])
		put(0, i, 2, maps.Cavity[i])
		put(0, i, 3, maps.Mask[i])
		for k := range 3 {
			put(1, i, k, maps.Position[i*3+k])
			put(2, i, k, maps.Normal[i*3+k]*2-1)
		}
		put(1, i, 3, float32(math.Floor(float64(maps.PartID[i]*256))))
		put(2, i, 3, maps.Island[i])
	}

	c.bakes[key] = out
	return out, nil
}

func withTexture(glb []byte, t mesh.Textures) ([]byte, error) {
	return mesh.WithTexture(glb, t, "")
}

// middleware serves model files and bakes, which are too large for bindings.
func (a *App) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		layout, _ := strconv.Atoi(r.URL.Query().Get("unwrap"))
		switch r.URL.Path {
		case "/model":
			file, _ := strconv.Atoi(r.URL.Query().Get("file"))
			glb, err := a.models.glb(path, layout, file)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "model/gltf-binary")
			w.Write(glb)

		case "/grunge":
			data, err := grunge.PNG(r.URL.Query().Get("name"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			w.Write(data)

		case "/stamp":
			dir, err := stampDir()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			http.ServeFile(w, r, filepath.Join(dir, filepath.Base(r.URL.Query().Get("name"))+".png"))

		case "/paint":
			file, err := paintPath(r.URL.Query().Get("project"), r.URL.Query().Get("id"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Cache-Control", "no-store")
			http.ServeFile(w, r, file)

		case "/bake":
			size, _ := strconv.Atoi(r.URL.Query().Get("size"))
			if size < 16 || size > 4096 {
				http.Error(w, "bad size", http.StatusBadRequest)
				return
			}
			data, err := a.models.bake(path, size, layout)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(data)

		default:
			next.ServeHTTP(w, r)
		}
	})
}
