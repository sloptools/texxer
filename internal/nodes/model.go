package nodes

import "texxer/internal/graph"

// ModelOf finds the model a graph textures: the path and material of its
// first layers or mesh node with a path set.
func ModelOf(g *graph.Graph) (path, material string, ok bool) {
	for _, typ := range []string{"layers", "mesh"} {
		for _, n := range g.Nodes {
			if n.Type == typ && n.Params.String("path", "") != "" {
				return n.Params.String("path", ""), n.Params.String("material", ""), true
			}
		}
	}
	return "", "", false
}
