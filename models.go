package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"texxer/internal/mesh"
	"texxer/internal/unwrap"
)

// A model key names the GLB files that share one texture, joined by "|". Most keys
// name a single file.
func keyPaths(key string) []string {
	return strings.Split(key, "|")
}

// loaded is the files of a key as one model, laid side by side.
type loaded struct {
	model   *mesh.Model
	paths   []string
	offsets []mesh.Vec
	// faces counts each file's faces, in order.
	faces  []int
	mtimes []int64
}

// unwrapped is a model laid out by texxer for a texture size, and each file's GLB with those UVs.
type unwrapped struct {
	model *mesh.Model
	glbs  [][]byte
	stats UnwrapStats
}

type modelCache struct {
	mu        sync.Mutex
	models    map[string]*loaded
	unwrapped map[string]*unwrapped
	bakes     map[string][]byte
}

func newModelCache() *modelCache {
	return &modelCache{models: map[string]*loaded{}, unwrapped: map[string]*unwrapped{}, bakes: map[string][]byte{}}
}

func mtimes(paths []string) ([]int64, error) {
	out := make([]int64, len(paths))
	for i, p := range paths {
		st, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		out[i] = st.ModTime().UnixNano()
	}
	return out, nil
}

func (c *modelCache) get(key string) (*loaded, error) {
	paths := keyPaths(key)
	stamps, err := mtimes(paths)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if l, ok := c.models[key]; ok && slices.Equal(l.mtimes, stamps) {
		return l, nil
	}

	var models []*mesh.Model
	var names []string
	l := &loaded{paths: paths, mtimes: stamps}
	for _, p := range paths {
		m, err := mesh.LoadGLB(p)
		if err != nil {
			return nil, err
		}
		models = append(models, m)
		names = append(names, strings.TrimSuffix(filepath.Base(p), filepath.Ext(p)))
		l.faces = append(l.faces, len(m.Tris))
	}
	l.model, l.offsets = mesh.Merge(models, names)
	for i, id := range unwrap.Islands(l.model.Tris) {
		l.model.Tris[i].Island = id
	}
	c.models[key] = l
	return l, nil
}

// touch keeps a key's cache after the export rewrote its files, which only got new textures.
func (c *modelCache) touch(key string) {
	stamps, err := mtimes(keyPaths(key))
	if err != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if l, ok := c.models[key]; ok {
		l.mtimes = stamps
	}
}

// model is the model with its own UVs for layout 0, else unwrapped for that texture size.
func (c *modelCache) model(key string, layout int) (*mesh.Model, error) {
	if layout == 0 {
		l, err := c.get(key)
		if err != nil {
			return nil, err
		}
		return l.model, nil
	}
	u, err := c.unwrap(key, layout)
	if err != nil {
		return nil, err
	}
	return u.model, nil
}

func (c *modelCache) unwrap(key string, size int) (*unwrapped, error) {
	l, err := c.get(key)
	if err != nil {
		return nil, err
	}

	ukey := fmt.Sprintf("%s|%p|%d", key, l.model, size)
	c.mu.Lock()
	defer c.mu.Unlock()
	if u, ok := c.unwrapped[ukey]; ok {
		return u, nil
	}

	res := unwrap.Unwrap(l.model.Tris, unwrap.Defaults(size))
	u := &unwrapped{stats: UnwrapStats{res.Charts, res.Fill, res.Density}}
	start := 0
	for i, p := range l.paths {
		glb, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		if glb, err = mesh.WithUVs(glb, res.UV[start:start+l.faces[i]]); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		u.glbs = append(u.glbs, glb)
		start += l.faces[i]
	}

	copied := *l.model
	copied.Tris = slices.Clone(l.model.Tris)
	for i := range copied.Tris {
		copied.Tris[i].UV = res.UV[i]
	}
	u.model = &copied
	c.unwrapped[ukey] = u
	return u, nil
}

// glb is file i of a key, with its own UVs for layout 0 or texxer's for that size.
func (c *modelCache) glb(key string, layout, i int) ([]byte, error) {
	paths := keyPaths(key)
	if i < 0 || i >= len(paths) {
		return nil, fmt.Errorf("no file %d in model", i)
	}
	if layout == 0 {
		return os.ReadFile(paths[i])
	}
	u, err := c.unwrap(key, layout)
	if err != nil {
		return nil, err
	}
	return u.glbs[i], nil
}

// fileHash tells when a key's files changed, whoever wrote them.
func fileHash(key string) (string, error) {
	h := sha1.New()
	for _, p := range keyPaths(key) {
		data, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil))[:16], nil
}

type UnwrapStats struct {
	Charts int `json:"charts"`
	// Fill is the share of the texture the charts cover.
	Fill float64 `json:"fill"`
	// Density is in texels per world unit.
	Density float64 `json:"density"`
}

// Unwrap lays a model out for a texture size and tells how well it fits.
func (a *App) Unwrap(key string, size int) (UnwrapStats, error) {
	u, err := a.models.unwrap(key, size)
	if err != nil {
		return UnwrapStats{}, err
	}
	return u.stats, nil
}
