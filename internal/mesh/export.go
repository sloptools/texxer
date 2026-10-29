package mesh

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
)

// Pixel art texture filtering for engines: nearest when magnified, mipmaps when far away.
const (
	filterNearest           = 9728
	filterNearestMipNearest = 9984
)

// WithTexture returns a copy of a binary glTF whose material uses png as its
// base color texture. An empty material picks every material that already
// has a base color texture. Other materials, like emissive ones, are left
// alone. The binary chunk is rebuilt, so replaced images don't linger in the file.
func WithTexture(glb, png []byte, material string) ([]byte, error) {
	doc, bin, err := rawGLB(glb)
	if err != nil {
		return nil, err
	}

	views := sliceArray(doc, "bufferViews")
	images := sliceArray(doc, "images")
	textures := sliceArray(doc, "textures")
	samplers := sliceArray(doc, "samplers")
	materials := sliceArray(doc, "materials")

	// Every view's bytes, so the buffer can be laid out again.
	data := make([][]byte, len(views))
	for i, v := range views {
		off, length := intField(v, "byteOffset"), intField(v, "byteLength")
		if off+length > len(bin) {
			return nil, fmt.Errorf("buffer view %d is out of range", i)
		}
		data[i] = bin[off : off+length]
	}

	// The new image, texture and pixel art sampler.
	data = append(data, png)
	views = append(views, map[string]any{"buffer": 0.0})
	images = append(images, map[string]any{"bufferView": float64(len(views) - 1), "mimeType": "image/png", "name": "texxer"})
	samplers = append(samplers, map[string]any{"magFilter": float64(filterNearest), "minFilter": float64(filterNearestMipNearest)})
	textures = append(textures, map[string]any{"source": float64(len(images) - 1), "sampler": float64(len(samplers) - 1)})
	tex := float64(len(textures) - 1)

	used := 0
	for _, m := range materials {
		mat, ok := m.(map[string]any)
		if !ok {
			continue
		}
		pbr, _ := mat["pbrMetallicRoughness"].(map[string]any)
		if material == "" {
			if pbr == nil || pbr["baseColorTexture"] == nil {
				continue
			}
		} else if mat["name"] != material {
			continue
		}

		if pbr == nil {
			pbr = map[string]any{}
			mat["pbrMetallicRoughness"] = pbr
		}
		pbr["baseColorTexture"] = map[string]any{"index": tex}
		// The baked texture is the full color; a tint would darken it.
		delete(pbr, "baseColorFactor")
		used++
	}
	if used == 0 {
		if material == "" {
			return nil, errors.New("no material has a base color texture; name the material to texture")
		}
		return nil, fmt.Errorf("model has no material %q", material)
	}

	// Drop what no material uses any more, like the replaced image, and renumber.
	textures, texIndex := compact(textures, usedTextures(materials))
	renumberTextures(materials, texIndex)

	usedImg := map[int]bool{}
	for _, t := range textures {
		usedImg[intField(t, "source")] = true
	}
	images, imgIndex := compact(images, usedImg)
	for _, t := range textures {
		t.(map[string]any)["source"] = float64(imgIndex[intField(t, "source")])
	}

	accessors := sliceArray(doc, "accessors")
	usedView := map[int]bool{}
	for _, list := range [][]any{accessors, images} {
		for _, x := range list {
			if v, ok := x.(map[string]any)["bufferView"].(float64); ok {
				usedView[int(v)] = true
			}
		}
	}
	keptData := [][]byte{}
	for i := range views {
		if usedView[i] {
			keptData = append(keptData, data[i])
		}
	}
	views, viewIndex := compact(views, usedView)
	for _, list := range [][]any{accessors, images} {
		for _, x := range list {
			m := x.(map[string]any)
			if v, ok := m["bufferView"].(float64); ok {
				m["bufferView"] = float64(viewIndex[int(v)])
			}
		}
	}

	var out bytes.Buffer
	for i, v := range views {
		for out.Len()%4 != 0 {
			out.WriteByte(0)
		}
		vm := v.(map[string]any)
		vm["byteOffset"], vm["byteLength"] = float64(out.Len()), float64(len(keptData[i]))
		out.Write(keptData[i])
	}
	for out.Len()%4 != 0 {
		out.WriteByte(0)
	}

	doc["bufferViews"], doc["images"], doc["textures"], doc["samplers"] = views, images, textures, samplers
	doc["buffers"] = []any{map[string]any{"byteLength": float64(out.Len())}}
	return writeGLB(doc, out.Bytes())
}

func rawGLB(data []byte) (map[string]any, []byte, error) {
	if len(data) < 20 || string(data[:4]) != "glTF" {
		return nil, nil, errors.New("not a binary glTF file")
	}

	var doc map[string]any
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
	if doc == nil {
		return nil, nil, errors.New("glTF has no JSON chunk")
	}
	if b, ok := doc["buffers"].([]any); ok && len(b) > 1 {
		return nil, nil, errors.New("models with several buffers are not supported")
	}
	return doc, bin, nil
}

func writeGLB(doc map[string]any, bin []byte) ([]byte, error) {
	js, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	for len(js)%4 != 0 {
		js = append(js, ' ')
	}

	var out bytes.Buffer
	total := 12 + 8 + len(js) + 8 + len(bin)
	binary.Write(&out, binary.LittleEndian, []uint32{0x46546C67, 2, uint32(total)})
	binary.Write(&out, binary.LittleEndian, []uint32{uint32(len(js)), 0x4E4F534A})
	out.Write(js)
	binary.Write(&out, binary.LittleEndian, []uint32{uint32(len(bin)), 0x004E4942})
	out.Write(bin)
	return out.Bytes(), nil
}

var textureRefs = map[string]bool{
	"baseColorTexture": true, "emissiveTexture": true, "normalTexture": true,
	"occlusionTexture": true, "metallicRoughnessTexture": true,
}

// eachTextureRef calls fn with every texture reference in the materials.
func eachTextureRef(v any, fn func(ref map[string]any)) {
	switch v := v.(type) {
	case map[string]any:
		for k, x := range v {
			if ref, ok := x.(map[string]any); ok && textureRefs[k] {
				fn(ref)
			}
			eachTextureRef(x, fn)
		}
	case []any:
		for _, x := range v {
			eachTextureRef(x, fn)
		}
	}
}

func usedTextures(materials []any) map[int]bool {
	used := map[int]bool{}
	eachTextureRef(materials, func(ref map[string]any) { used[intField(ref, "index")] = true })
	return used
}

func renumberTextures(materials []any, index map[int]int) {
	eachTextureRef(materials, func(ref map[string]any) { ref["index"] = float64(index[intField(ref, "index")]) })
}

// compact keeps the used entries of list and returns old to new indices.
func compact(list []any, used map[int]bool) ([]any, map[int]int) {
	out := []any{}
	index := map[int]int{}
	for i, x := range list {
		if used[i] {
			index[i] = len(out)
			out = append(out, x)
		}
	}
	return out, index
}

func sliceArray(doc map[string]any, key string) []any {
	a, _ := doc[key].([]any)
	return a
}

// intField reads a number from a bufferView, which glTF decodes as float64.
func intField(v any, key string) int {
	m, _ := v.(map[string]any)
	f, _ := m[key].(float64)
	return int(f)
}
