package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"texxer/internal/grunge"
	"texxer/internal/mesh"
)

type modelCache struct {
	mu     sync.Mutex
	models map[string]cachedModel
	bakes  map[string][]byte
}

type cachedModel struct {
	mtime int64
	model *mesh.Model
}

func newModelCache() *modelCache {
	return &modelCache{models: map[string]cachedModel{}, bakes: map[string][]byte{}}
}

func (c *modelCache) get(path string) (*mesh.Model, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if m, ok := c.models[path]; ok && m.mtime == st.ModTime().UnixNano() {
		return m.model, nil
	}

	m, err := mesh.LoadGLB(path)
	if err != nil {
		return nil, err
	}
	c.models[path] = cachedModel{st.ModTime().UnixNano(), m}
	return m, nil
}

// bake returns three float32 RGBA maps of size×size texels, back to back:
// AO, convex edges, concave cavity, coverage;
// position (p - min) / diagonal, part index;
// normal, unused.
func (c *modelCache) bake(path string, size int) ([]byte, error) {
	m, err := c.get(path)
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
		put(2, i, 3, 1)
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
		switch r.URL.Path {
		case "/model":
			w.Header().Set("Content-Type", "model/gltf-binary")
			http.ServeFile(w, r, path)

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
			file, err := paintPath(path, r.URL.Query().Get("id"))
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
			data, err := a.models.bake(path, size)
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
