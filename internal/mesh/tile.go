package mesh

import (
	"bytes"
	"encoding/binary"
)

// TileGLB is a wall of w×h meters for a tileable texture: in the xy plane facing +z,
// its UVs over the top left of the texture, the longer side across all of it.
func TileGLB(w, h float64) ([]byte, error) {
	l := max(w, h)
	pos := []float32{0, 0, 0, float32(w), 0, 0, float32(w), float32(h), 0, 0, float32(h), 0}
	nrm := []float32{0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1}
	uv := []float32{0, float32(h / l), float32(w / l), float32(h / l), float32(w / l), 0, 0, 0}
	idx := []uint16{0, 1, 2, 0, 2, 3}

	var bin bytes.Buffer
	views := []any{}
	for _, data := range []any{pos, nrm, uv, idx} {
		start := bin.Len()
		binary.Write(&bin, binary.LittleEndian, data)
		views = append(views, map[string]any{"buffer": 0, "byteOffset": start, "byteLength": bin.Len() - start})
	}
	for bin.Len()%4 != 0 {
		bin.WriteByte(0)
	}

	doc := map[string]any{
		"asset":  map[string]any{"version": "2.0", "generator": "texxer"},
		"scene":  0,
		"scenes": []any{map[string]any{"nodes": []any{0}}},
		"nodes":  []any{map[string]any{"name": "tile", "mesh": 0}},
		"meshes": []any{map[string]any{"primitives": []any{map[string]any{
			"attributes": map[string]any{"POSITION": 0, "NORMAL": 1, "TEXCOORD_0": 2},
			"indices":    3,
			"material":   0,
		}}}},
		"materials": []any{map[string]any{"name": "tile"}},
		"accessors": []any{
			map[string]any{"bufferView": 0, "componentType": 5126, "count": 4, "type": "VEC3", "min": []float64{0, 0, 0}, "max": []float64{w, h, 0}},
			map[string]any{"bufferView": 1, "componentType": 5126, "count": 4, "type": "VEC3"},
			map[string]any{"bufferView": 2, "componentType": 5126, "count": 4, "type": "VEC2"},
			map[string]any{"bufferView": 3, "componentType": 5123, "count": 6, "type": "SCALAR"},
		},
		"bufferViews": views,
		"buffers":     []any{map[string]any{"byteLength": bin.Len()}},
	}
	return writeGLB(doc, bin.Bytes())
}
