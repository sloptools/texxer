package server

import (
	"path/filepath"
	"slices"
	"strings"

	"texxer/internal/graph"
	"texxer/internal/nodes"
	"texxer/internal/presets"
)

// catalog lists all node types with their inputs, outputs and params:
// built-ins, presets, and the graphs in dir.
func catalog(dir string) (map[string]nodes.Def, error) {
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
	for name, d := range nodes.GraphIO {
		out[name] = d
	}

	for _, name := range presets.Names() {
		g, err := presets.Load(name)
		if err != nil {
			return nil, err
		}
		out[name] = subgraphDef(g, "preset")
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".json")
		if _, taken := out[name]; taken || !validName.MatchString(name) {
			continue
		}

		g, err := graph.Load(f)
		if err != nil {
			// A broken file shouldn't take the whole editor down.
			continue
		}
		out[name] = subgraphDef(g, "graph")
	}
	return out, nil
}

func subgraphDef(g *graph.Graph, category string) nodes.Def {
	d := nodes.Def{
		Category: category,
		Inputs:   g.InputNames(),
		Outputs:  g.OutputNames(),
	}
	if d.Inputs == nil {
		d.Inputs = []string{}
	}

	// Main output first.
	slices.SortStableFunc(d.Outputs, func(a, b string) int {
		if a == "out" {
			return -1
		}
		if b == "out" {
			return 1
		}
		return 0
	})

	for name, ex := range g.Exposed {
		d.Params = append(d.Params, exposedParam(g, name, ex))
	}
	d.Params = append(d.Params, nodes.Param{Name: "seed", Kind: "int", Default: 0, Max: 9999})
	slices.SortFunc(d.Params, func(a, b nodes.Param) int { return strings.Compare(a.Name, b.Name) })

	return d
}

// exposedParam describes an exposed param like the param it targets, so the
// editor shows the same widget, range and options.
func exposedParam(g *graph.Graph, name string, ex graph.Exposed) nodes.Param {
	p := nodes.Param{Name: name, Kind: paramKind(ex.Default), Default: ex.Default}

	if len(ex.Targets) > 0 {
		id, param, _ := strings.Cut(ex.Targets[0], ".")
		node := g.Nodes[id]

		if def, ok := nodes.Registry[node.Type]; ok {
			if i := slices.IndexFunc(def.Params, func(p nodes.Param) bool { return p.Name == param }); i >= 0 {
				p = def.Params[i]
				p.Name = name
				if v, ok := node.Params[param]; ok {
					p.Default = v
				}
				if ex.Default != nil {
					p.Default = ex.Default
				}
			}
		}
	}

	if ex.Min != 0 || ex.Max != 0 {
		p.Min, p.Max = ex.Min, ex.Max
	}
	return p
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
