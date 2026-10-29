package graph

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
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
// Without a default the targets keep their own values.
type Exposed struct {
	Default any      `json:"default,omitempty"`
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

// OutputRef resolves a subgraph output from the outputs map, an output node
// with that name, or the graph's main output for "out".
func (g *Graph) OutputRef(name string) (string, bool) {
	if ref, ok := g.Outputs[name]; ok {
		return ref, true
	}

	for id, n := range g.Nodes {
		if n.Type == "output" && n.Params.String("name", "out") == name {
			return id, true
		}
	}

	if name == "out" && g.Output != "" {
		return g.Output, true
	}
	return "", false
}

// OutputNames lists the outputs OutputRef can resolve.
func (g *Graph) OutputNames() []string {
	seen := map[string]bool{}
	for name := range g.Outputs {
		seen[name] = true
	}
	for _, n := range g.Nodes {
		if n.Type == "output" {
			seen[n.Params.String("name", "out")] = true
		}
	}
	if len(seen) == 0 && g.Output != "" {
		seen["out"] = true
	}
	return slices.Sorted(maps.Keys(seen))
}

// InputNames lists the names of the graph's input nodes.
func (g *Graph) InputNames() []string {
	var names []string
	for _, n := range g.Nodes {
		if n.Type == "input" {
			names = append(names, n.Params.String("name", ""))
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}

// types lists the node types used in the graph, sorted.
func (g *Graph) types() []string {
	var out []string
	for _, n := range g.Nodes {
		out = append(out, n.Type)
	}
	slices.Sort(out)
	return slices.Compact(out)
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
		if v == nil {
			continue
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
