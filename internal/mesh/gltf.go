// Package mesh loads glTF models and bakes their geometry into UV-space maps.
package mesh

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
)

// Tri is a triangle in world space.
type Tri struct {
	P    [3]Vec
	N    [3]Vec
	UV   [3][2]float64
	Part int
	// Material is the glTF material name.
	Material string
	// Island numbers the surface the face belongs to, bounded by hard edges.
	Island int
	// Src keeps the model's own UVs, which its detail maps follow; Detail indexes
	// Model.Details, -1 without.
	Src    [3][2]float64
	Detail int
}

type Model struct {
	Tris []Tri
	// Parts are node names, indexed by Tri.Part.
	Parts []string
	// Details are the normal and occlusion maps of the model's materials.
	Details  []*Detail
	Min, Max Vec
}

type textureRef struct {
	Index int `json:"index"`
}

type gltfDoc struct {
	Scene  int `json:"scene"`
	Scenes []struct {
		Nodes []int `json:"nodes"`
	} `json:"scenes"`
	Nodes []struct {
		Name        string    `json:"name"`
		Mesh        *int      `json:"mesh"`
		Children    []int     `json:"children"`
		Matrix      []float64 `json:"matrix"`
		Translation []float64 `json:"translation"`
		Rotation    []float64 `json:"rotation"`
		Scale       []float64 `json:"scale"`
	} `json:"nodes"`
	Meshes []struct {
		Primitives []struct {
			Attributes map[string]int `json:"attributes"`
			Indices    *int           `json:"indices"`
			Material   *int           `json:"material"`
			Mode       *int           `json:"mode"`
		} `json:"primitives"`
	} `json:"meshes"`
	Materials []struct {
		Name             string      `json:"name"`
		NormalTexture    *textureRef `json:"normalTexture"`
		OcclusionTexture *textureRef `json:"occlusionTexture"`
	} `json:"materials"`
	Textures []struct {
		Source int `json:"source"`
	} `json:"textures"`
	Images []struct {
		BufferView *int   `json:"bufferView"`
		URI        string `json:"uri"`
	} `json:"images"`
	Accessors []struct {
		BufferView    *int   `json:"bufferView"`
		ByteOffset    int    `json:"byteOffset"`
		ComponentType int    `json:"componentType"`
		Count         int    `json:"count"`
		Type          string `json:"type"`
		Normalized    bool   `json:"normalized"`
	} `json:"accessors"`
	BufferViews []struct {
		ByteOffset int `json:"byteOffset"`
		ByteLength int `json:"byteLength"`
		ByteStride int `json:"byteStride"`
	} `json:"bufferViews"`
}

// LoadGLB reads a binary glTF with embedded buffers and flattens its scene into world-space triangles.
func LoadGLB(path string) (*Model, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	doc, bin, err := parseGLB(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	m := &Model{Min: Vec{math.Inf(1), math.Inf(1), math.Inf(1)}, Max: Vec{math.Inf(-1), math.Inf(-1), math.Inf(-1)}}
	details, err := m.addDetails(doc, bin)
	if err != nil {
		return nil, err
	}

	roots := []int{}
	if len(doc.Scenes) > 0 {
		roots = doc.Scenes[doc.Scene].Nodes
	} else {
		for i := range doc.Nodes {
			roots = append(roots, i)
		}
	}
	for _, n := range roots {
		if err := m.addNode(doc, bin, details, n, identity()); err != nil {
			return nil, err
		}
	}

	if len(m.Tris) == 0 {
		return nil, errors.New("model has no triangles")
	}
	return m, nil
}

func parseGLB(data []byte) (*gltfDoc, []byte, error) {
	if len(data) < 20 || string(data[:4]) != "glTF" {
		return nil, nil, errors.New("not a binary glTF file")
	}

	var doc gltfDoc
	var bin []byte
	for at := 12; at+8 <= len(data); {
		size := int(binary.LittleEndian.Uint32(data[at:]))
		kind := string(data[at+4 : at+8])
		chunk := data[at+8 : min(at+8+size, len(data))]

		switch kind {
		case "JSON":
			if err := json.Unmarshal(chunk, &doc); err != nil {
				return nil, nil, err
			}
		case "BIN\x00":
			bin = chunk
		}
		at += 8 + size
	}
	return &doc, bin, nil
}

// addDetails reads each material's normal and occlusion maps and returns, per
// material, its index in m.Details or -1.
func (m *Model) addDetails(doc *gltfDoc, bin []byte) ([]int, error) {
	out := make([]int, len(doc.Materials))
	for i, mat := range doc.Materials {
		out[i] = -1
		if mat.NormalTexture == nil && mat.OcclusionTexture == nil {
			continue
		}
		d := &Detail{}
		var err error
		if mat.NormalTexture != nil {
			if d.Normal, err = decodeTexture(doc, bin, mat.NormalTexture.Index); err != nil {
				return nil, fmt.Errorf("normal map: %w", err)
			}
		}
		if mat.OcclusionTexture != nil {
			if d.Occlusion, err = decodeTexture(doc, bin, mat.OcclusionTexture.Index); err != nil {
				return nil, fmt.Errorf("occlusion map: %w", err)
			}
		}
		out[i] = len(m.Details)
		m.Details = append(m.Details, d)
	}
	return out, nil
}

func decodeTexture(doc *gltfDoc, bin []byte, index int) (image.Image, error) {
	if index >= len(doc.Textures) {
		return nil, errors.New("missing texture")
	}
	img := doc.Images[doc.Textures[index].Source]
	if img.BufferView == nil {
		return nil, errors.New("external texture files are not supported yet")
	}
	bv := doc.BufferViews[*img.BufferView]
	decoded, _, err := image.Decode(bytes.NewReader(bin[bv.ByteOffset : bv.ByteOffset+bv.ByteLength]))
	return decoded, err
}

func (m *Model) addNode(doc *gltfDoc, bin []byte, details []int, index int, parent mat4) error {
	node := doc.Nodes[index]
	world := parent.mul(localMatrix(node.Matrix, node.Translation, node.Rotation, node.Scale))

	if node.Mesh != nil {
		part := len(m.Parts)
		m.Parts = append(m.Parts, node.Name)
		normalMat := world.normalMatrix()

		for _, prim := range doc.Meshes[*node.Mesh].Primitives {
			if prim.Mode != nil && *prim.Mode != 4 {
				continue
			}
			if err := m.addPrimitive(doc, bin, details, prim.Attributes, prim.Indices, prim.Material, world, normalMat, part); err != nil {
				return err
			}
		}
	}

	for _, c := range node.Children {
		if err := m.addNode(doc, bin, details, c, world); err != nil {
			return err
		}
	}
	return nil
}

func (m *Model) addPrimitive(doc *gltfDoc, bin []byte, details []int, attrs map[string]int, indices, material *int, world mat4, normalMat [9]float64, part int) error {
	pi, ok := attrs["POSITION"]
	if !ok {
		return nil
	}
	pos, err := readFloats(doc, bin, pi, 3)
	if err != nil {
		return err
	}

	var nor, uv [][]float64
	if ni, ok := attrs["NORMAL"]; ok {
		if nor, err = readFloats(doc, bin, ni, 3); err != nil {
			return err
		}
	}
	if ti, ok := attrs["TEXCOORD_0"]; ok {
		if uv, err = readFloats(doc, bin, ti, 2); err != nil {
			return err
		}
	}

	var idx []int
	if indices != nil {
		if idx, err = readInts(doc, bin, *indices); err != nil {
			return err
		}
	} else {
		for i := range pos {
			idx = append(idx, i)
		}
	}

	matName := ""
	detail := -1
	if material != nil && *material < len(doc.Materials) {
		matName = doc.Materials[*material].Name
		detail = details[*material]
	}

	for k := 0; k+2 < len(idx); k += 3 {
		t := Tri{Part: part, Material: matName, Detail: detail}
		for c := range 3 {
			v := idx[k+c]
			p := world.point(Vec{pos[v][0], pos[v][1], pos[v][2]})
			t.P[c] = p
			m.Min, m.Max = m.Min.Min(p), m.Max.Max(p)

			if nor != nil {
				t.N[c] = transformNormal(normalMat, Vec{nor[v][0], nor[v][1], nor[v][2]})
			}
			if uv != nil {
				t.UV[c] = [2]float64{uv[v][0], uv[v][1]}
			}
		}
		t.Src = t.UV
		if nor == nil {
			n := t.P[1].Sub(t.P[0]).Cross(t.P[2].Sub(t.P[0])).Norm()
			t.N = [3]Vec{n, n, n}
		}
		m.Tris = append(m.Tris, t)
	}
	return nil
}

var componentSize = map[int]int{5120: 1, 5121: 1, 5122: 2, 5123: 2, 5125: 4, 5126: 4}

func readFloats(doc *gltfDoc, bin []byte, index, n int) ([][]float64, error) {
	a := doc.Accessors[index]
	if a.ComponentType != 5126 {
		return nil, fmt.Errorf("accessor %d: only float attributes are supported", index)
	}
	raw, stride, err := accessorBytes(doc, bin, index, n*4)
	if err != nil {
		return nil, err
	}

	out := make([][]float64, a.Count)
	for i := range out {
		out[i] = make([]float64, n)
		for c := range n {
			bits := binary.LittleEndian.Uint32(raw[i*stride+c*4:])
			out[i][c] = float64(math.Float32frombits(bits))
		}
	}
	return out, nil
}

func readInts(doc *gltfDoc, bin []byte, index int) ([]int, error) {
	a := doc.Accessors[index]
	size, ok := componentSize[a.ComponentType]
	if !ok || a.ComponentType == 5126 {
		return nil, fmt.Errorf("accessor %d: bad index type", index)
	}
	raw, stride, err := accessorBytes(doc, bin, index, size)
	if err != nil {
		return nil, err
	}

	out := make([]int, a.Count)
	for i := range out {
		b := raw[i*stride:]
		switch size {
		case 1:
			out[i] = int(b[0])
		case 2:
			out[i] = int(binary.LittleEndian.Uint16(b))
		default:
			out[i] = int(binary.LittleEndian.Uint32(b))
		}
	}
	return out, nil
}

// accessorBytes returns the accessor's data and the byte stride between elements.
func accessorBytes(doc *gltfDoc, bin []byte, index, elemSize int) ([]byte, int, error) {
	a := doc.Accessors[index]
	if a.BufferView == nil {
		return nil, 0, fmt.Errorf("accessor %d has no buffer view", index)
	}

	bv := doc.BufferViews[*a.BufferView]
	stride := bv.ByteStride
	if stride == 0 {
		stride = elemSize
	}

	start := bv.ByteOffset + a.ByteOffset
	end := start + (a.Count-1)*stride + elemSize
	if a.Count == 0 || end > len(bin) {
		return nil, 0, fmt.Errorf("accessor %d is out of range", index)
	}
	return bin[start:end], stride, nil
}
