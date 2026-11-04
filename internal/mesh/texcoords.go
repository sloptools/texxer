package mesh

import "fmt"

// SwapTexcoords makes a UV set each object's first, which texxer paints on, by
// swapping it with TEXCOORD_0. Sets are by node name; objects not listed keep theirs.
func SwapTexcoords(glb []byte, sets map[string]int) ([]byte, error) {
	doc, bin, err := rawGLB(glb)
	if err != nil {
		return nil, err
	}
	meshes := sliceArray(doc, "meshes")
	for _, n := range sliceArray(doc, "nodes") {
		node, _ := n.(map[string]any)
		name, _ := node["name"].(string)
		set, ok := sets[name]
		index, hasMesh := node["mesh"].(float64)
		if !ok || set <= 0 || !hasMesh || int(index) >= len(meshes) {
			continue
		}
		mesh, _ := meshes[int(index)].(map[string]any)
		prims, _ := mesh["primitives"].([]any)
		key := fmt.Sprintf("TEXCOORD_%d", set)
		for _, p := range prims {
			prim, _ := p.(map[string]any)
			attrs, _ := prim["attributes"].(map[string]any)
			if v, ok := attrs[key]; ok {
				attrs[key], attrs["TEXCOORD_0"] = attrs["TEXCOORD_0"], v
			}
		}
	}
	return writeGLB(doc, bin)
}
