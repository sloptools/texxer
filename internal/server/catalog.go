package server

import (
	"slices"
	"strings"

	"texxer/internal/graph"
	"texxer/internal/nodes"
	"texxer/internal/presets"
)

// catalog lists all node types with their inputs, outputs and params.
func catalog() (map[string]nodes.Def, error) {
	out := map[string]nodes.Def{}

	for name, d := range nodes.Registry {
		if len(d.Outputs) == 0 {
			d.Outputs = []string{"out"}
		}
		if d.Inputs == nil {
			d.Inputs = []string{}
		}
		out[name] = d
	}

	for _, name := range presets.Names() {
		g, err := presets.Load(name)
		if err != nil {
			return nil, err
		}
		out[name] = presetDef(g)
	}
	return out, nil
}

func presetDef(g *graph.Graph) nodes.Def {
	d := nodes.Def{Category: "preset", Inputs: []string{}}

	for _, n := range g.Nodes {
		if n.Type == "input" {
			d.Inputs = append(d.Inputs, n.Params.String("name", ""))
		}
	}
	slices.Sort(d.Inputs)

	for name := range g.Outputs {
		d.Outputs = append(d.Outputs, name)
	}
	// Main output first.
	slices.SortFunc(d.Outputs, func(a, b string) int {
		if a == "out" {
			return -1
		}
		if b == "out" {
			return 1
		}
		return strings.Compare(a, b)
	})

	for name, ex := range g.Exposed {
		d.Params = append(d.Params, nodes.Param{
			Name:    name,
			Kind:    paramKind(ex.Default),
			Default: ex.Default,
			Min:     ex.Min,
			Max:     ex.Max,
		})
	}
	d.Params = append(d.Params, nodes.Param{Name: "seed", Kind: "int", Default: 0, Max: 9999})
	slices.SortFunc(d.Params, func(a, b nodes.Param) int { return strings.Compare(a.Name, b.Name) })

	return d
}

func paramKind(v any) string {
	switch v := v.(type) {
	case float64:
		return "float"
	case bool:
		return "bool"
	case string:
		if strings.HasPrefix(v, "#") && len(v) == 7 {
			return "color"
		}
		return "string"
	default:
		return "json"
	}
}
