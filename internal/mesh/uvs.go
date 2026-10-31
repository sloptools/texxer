package mesh

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

const (
	targetVertices = 34962
	targetIndices  = 34963
)

var components = map[string]int{"SCALAR": 1, "VEC2": 2, "VEC3": 3, "VEC4": 4, "MAT2": 4, "MAT3": 9, "MAT4": 16}

// WithUVs returns a copy of a binary glTF with new texture coordinates, given per face
// corner in LoadGLB's face order. Vertices split where their faces got different UVs, and
// a mesh used by several nodes is copied, since each use now has UVs of its own.
func WithUVs(glb []byte, uvs [][3][2]float64) ([]byte, error) {
	doc, bin, err := rawGLB(glb)
	if err != nil {
		return nil, err
	}

	views := sliceArray(doc, "bufferViews")
	data := make([][]byte, len(views))
	for i, v := range views {
		off, length := intField(v, "byteOffset"), intField(v, "byteLength")
		if off+length > len(bin) {
			return nil, fmt.Errorf("buffer view %d is out of range", i)
		}
		data[i] = bin[off : off+length]
	}

	w := &uvWriter{doc: doc, bin: bin, views: views, data: data, accessors: sliceArray(doc, "accessors"), uvs: uvs}
	nodes := sliceArray(doc, "nodes")
	var roots []int
	if scenes := sliceArray(doc, "scenes"); len(scenes) > 0 {
		for _, n := range scenes[intField(doc, "scene")].(map[string]any)["nodes"].([]any) {
			roots = append(roots, int(n.(float64)))
		}
	} else {
		for i := range nodes {
			roots = append(roots, i)
		}
	}
	used := map[int]bool{}
	for _, n := range roots {
		if err := w.node(nodes, n, used); err != nil {
			return nil, err
		}
	}
	if w.next != len(uvs) {
		return nil, fmt.Errorf("model has %d faces, got UVs for %d", w.next, len(uvs))
	}

	doc["accessors"] = w.accessors
	dropUnusedAccessors(doc)
	doc["bufferViews"] = w.views
	return writeGLB(doc, packBuffer(doc, w.data))
}

type uvWriter struct {
	doc       map[string]any
	bin       []byte
	views     []any
	data      [][]byte
	accessors []any
	uvs       [][3][2]float64
	// next is the first face not written yet.
	next int
}

// node walks the scene like LoadGLB does, so faces come in the same order.
func (w *uvWriter) node(nodes []any, index int, used map[int]bool) error {
	node := nodes[index].(map[string]any)
	if m, ok := node["mesh"].(float64); ok {
		meshes := sliceArray(w.doc, "meshes")
		mi := int(m)
		if used[mi] {
			var copied any
			js, _ := json.Marshal(meshes[mi])
			json.Unmarshal(js, &copied)
			meshes = append(meshes, copied)
			w.doc["meshes"] = meshes
			mi = len(meshes) - 1
			node["mesh"] = float64(mi)
		}
		used[mi] = true

		for _, p := range meshes[mi].(map[string]any)["primitives"].([]any) {
			if err := w.primitive(p.(map[string]any)); err != nil {
				return err
			}
		}
	}

	children, _ := node["children"].([]any)
	for _, c := range children {
		if err := w.node(nodes, int(c.(float64)), used); err != nil {
			return err
		}
	}
	return nil
}

func (w *uvWriter) primitive(prim map[string]any) error {
	if mode, ok := prim["mode"].(float64); ok && mode != 4 {
		return nil
	}
	attrs, _ := prim["attributes"].(map[string]any)
	pos, ok := attrs["POSITION"].(float64)
	if !ok {
		return nil
	}

	var idx []int
	if i, ok := prim["indices"].(float64); ok {
		var err error
		if idx, err = w.ints(int(i)); err != nil {
			return err
		}
	} else {
		for i := range intField(w.accessors[int(pos)], "count") {
			idx = append(idx, i)
		}
	}
	faces := len(idx) / 3
	if w.next+faces > len(w.uvs) {
		return errors.New("model has more faces than UVs")
	}

	// A vertex per old vertex and UV.
	type key struct {
		v    int
		u, t float32
	}
	ids := map[key]int{}
	var order []int
	var uv []float32
	newIdx := make([]int, faces*3)
	for f := range faces {
		for c := range 3 {
			t := w.uvs[w.next+f][c]
			k := key{idx[f*3+c], float32(t[0]), float32(t[1])}
			id, ok := ids[k]
			if !ok {
				id = len(order)
				ids[k] = id
				order = append(order, k.v)
				uv = append(uv, k.u, k.t)
			}
			newIdx[f*3+c] = id
		}
	}
	w.next += faces

	for name, a := range attrs {
		if name == "TEXCOORD_0" {
			continue
		}
		i, err := w.gather(int(a.(float64)), order)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		attrs[name] = float64(i)
	}
	if targets, ok := prim["targets"].([]any); ok {
		for _, t := range targets {
			for name, a := range t.(map[string]any) {
				i, err := w.gather(int(a.(float64)), order)
				if err != nil {
					return fmt.Errorf("morph %s: %w", name, err)
				}
				t.(map[string]any)[name] = float64(i)
			}
		}
	}

	raw := make([]byte, len(uv)*4)
	for i, x := range uv {
		binary.LittleEndian.PutUint32(raw[i*4:], math.Float32bits(x))
	}
	attrs["TEXCOORD_0"] = float64(w.add(raw, 8, targetVertices, map[string]any{"componentType": 5126.0, "type": "VEC2", "count": float64(len(order))}))

	indexType, size := 5123.0, 2
	if len(order) > 65535 {
		indexType, size = 5125.0, 4
	}
	rawIdx := make([]byte, len(newIdx)*size)
	for i, v := range newIdx {
		if size == 2 {
			binary.LittleEndian.PutUint16(rawIdx[i*2:], uint16(v))
		} else {
			binary.LittleEndian.PutUint32(rawIdx[i*4:], uint32(v))
		}
	}
	prim["indices"] = float64(w.add(rawIdx, 0, targetIndices, map[string]any{"componentType": indexType, "type": "SCALAR", "count": float64(len(newIdx))}))
	return nil
}

// gather makes a new accessor of the old one's elements in the given order.
func (w *uvWriter) gather(index int, order []int) (int, error) {
	elems, err := w.elements(index)
	if err != nil {
		return 0, err
	}
	old := w.accessors[index].(map[string]any)
	size := len(elems[0])
	stride := (size + 3) &^ 3

	raw := make([]byte, len(order)*stride)
	for i, v := range order {
		if v >= len(elems) {
			return 0, fmt.Errorf("index %d is out of range", v)
		}
		copy(raw[i*stride:], elems[v])
	}

	a := map[string]any{"componentType": old["componentType"], "type": old["type"], "count": float64(len(order))}
	for _, k := range []string{"normalized", "min", "max"} {
		if v, ok := old[k]; ok {
			a[k] = v
		}
	}
	viewStride := 0
	if stride != size {
		viewStride = stride
	}
	return w.add(raw, viewStride, targetVertices, a), nil
}

// add stores data in a new buffer view and returns the new accessor's index.
func (w *uvWriter) add(raw []byte, stride, target int, accessor map[string]any) int {
	view := map[string]any{"buffer": 0.0, "target": float64(target)}
	if stride > 0 {
		view["byteStride"] = float64(stride)
	}
	w.views = append(w.views, view)
	w.data = append(w.data, raw)
	accessor["bufferView"] = float64(len(w.views) - 1)
	w.accessors = append(w.accessors, accessor)
	return len(w.accessors) - 1
}

// elements returns each element's bytes of an accessor.
func (w *uvWriter) elements(index int) ([][]byte, error) {
	a := w.accessors[index].(map[string]any)
	if a["sparse"] != nil {
		return nil, fmt.Errorf("accessor %d: sparse accessors are not supported", index)
	}
	view, ok := a["bufferView"].(float64)
	if !ok {
		return nil, fmt.Errorf("accessor %d has no buffer view", index)
	}
	size := componentSize[intField(a, "componentType")] * components[a["type"].(string)]
	if size == 0 {
		return nil, fmt.Errorf("accessor %d has an unknown type", index)
	}

	v := w.views[int(view)]
	stride := intField(v, "byteStride")
	if stride == 0 {
		stride = size
	}
	data := w.data[int(view)]
	start := intField(a, "byteOffset")
	count := intField(a, "count")
	if count == 0 || start+(count-1)*stride+size > len(data) {
		return nil, fmt.Errorf("accessor %d is out of range", index)
	}

	out := make([][]byte, count)
	for i := range out {
		out[i] = data[start+i*stride : start+i*stride+size]
	}
	return out, nil
}

func (w *uvWriter) ints(index int) ([]int, error) {
	elems, err := w.elements(index)
	if err != nil {
		return nil, err
	}
	out := make([]int, len(elems))
	for i, b := range elems {
		switch len(b) {
		case 1:
			out[i] = int(b[0])
		case 2:
			out[i] = int(binary.LittleEndian.Uint16(b))
		case 4:
			out[i] = int(binary.LittleEndian.Uint32(b))
		default:
			return nil, fmt.Errorf("accessor %d: bad index type", index)
		}
	}
	return out, nil
}

// dropUnusedAccessors removes accessors nothing refers to any more and renumbers the rest.
func dropUnusedAccessors(doc map[string]any) {
	var refs []func(func(float64) float64)
	ref := func(m map[string]any, key string) {
		if _, ok := m[key].(float64); ok {
			refs = append(refs, func(fn func(float64) float64) { m[key] = fn(m[key].(float64)) })
		}
	}
	for _, mesh := range sliceArray(doc, "meshes") {
		for _, p := range mesh.(map[string]any)["primitives"].([]any) {
			prim := p.(map[string]any)
			ref(prim, "indices")
			attrs, _ := prim["attributes"].(map[string]any)
			for k := range attrs {
				ref(attrs, k)
			}
			targets, _ := prim["targets"].([]any)
			for _, t := range targets {
				for k := range t.(map[string]any) {
					ref(t.(map[string]any), k)
				}
			}
		}
	}
	for _, s := range sliceArray(doc, "skins") {
		ref(s.(map[string]any), "inverseBindMatrices")
	}
	for _, a := range sliceArray(doc, "animations") {
		samplers, _ := a.(map[string]any)["samplers"].([]any)
		for _, s := range samplers {
			ref(s.(map[string]any), "input")
			ref(s.(map[string]any), "output")
		}
	}

	used := map[int]bool{}
	for _, r := range refs {
		r(func(i float64) float64 { used[int(i)] = true; return i })
	}
	accessors, index := compact(sliceArray(doc, "accessors"), used)
	for _, r := range refs {
		r(func(i float64) float64 { return float64(index[int(i)]) })
	}
	doc["accessors"] = accessors
}
