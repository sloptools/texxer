package graph

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
)

type Node struct {
	Type   string            `json:"type"`
	Params Params            `json:"params,omitempty"`
	Inputs map[string]string `json:"inputs,omitempty"`

	// UI holds the editor position and is ignored by the evaluator.
	UI *UI `json:"ui,omitempty"`
}

type UI struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Graph struct {
	// Size is the design resolution; pixel params are relative to it.
	Size   int             `json:"size"`
	Output string          `json:"output"`
	Nodes  map[string]Node `json:"nodes"`

	// Used when the graph is a subgraph.
	Outputs map[string]string  `json:"outputs,omitempty"`
	Exposed map[string]Exposed `json:"params,omitempty"`

	Dir string `json:"-"`
}

// Exposed is a subgraph param that sets one or more "node.param" targets.
type Exposed struct {
	Default any      `json:"default"`
	Targets []string `json:"targets"`
	Min     float64  `json:"min,omitempty"`
	Max     float64  `json:"max,omitempty"`
}

func Load(path string) (*Graph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	g, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	g.Dir = filepath.Dir(path)
	return g, nil
}

func Parse(data []byte) (*Graph, error) {
	var g Graph
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, err
	}

	if g.Size == 0 {
		g.Size = 512
	}
	return &g, nil
}

// withParams returns a copy with exposed params set from values or their defaults.
func (g *Graph) withParams(values Params) (*Graph, error) {
	for name := range values {
		if name == "seed" {
			continue
		}
		if _, ok := g.Exposed[name]; !ok {
			return nil, fmt.Errorf("unknown param %q", name)
		}
	}

	out := *g
	out.Nodes = maps.Clone(g.Nodes)

	for name, ex := range g.Exposed {
		v, ok := values[name]
		if !ok {
			v = ex.Default
		}

		for _, t := range ex.Targets {
			id, param, _ := strings.Cut(t, ".")
			node, ok := out.Nodes[id]
			if !ok || param == "" {
				return nil, fmt.Errorf("param %q: bad target %q", name, t)
			}

			node.Params = maps.Clone(node.Params)
			if node.Params == nil {
				node.Params = Params{}
			}
			node.Params[param] = v
			out.Nodes[id] = node
		}
	}
	return &out, nil
}
