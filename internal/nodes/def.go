package nodes

import "texxer/internal/graph"

// Def describes a node type for the evaluator and the editor.
type Def struct {
	Eval     graph.EvalFunc `json:"-"`
	Category string         `json:"category"`
	Inputs   []string       `json:"inputs"`
	Outputs  []string       `json:"outputs"`
	Params   []Param        `json:"params"`

	// Dynamic nodes take extra inputs named by their params, like trim strips.
	Dynamic bool `json:"dynamic,omitempty"`
}

type Param struct {
	Name    string   `json:"name"`
	Kind    string   `json:"kind"`
	Default any      `json:"default"`
	Min     float64  `json:"min,omitempty"`
	Max     float64  `json:"max,omitempty"`
	Options []string `json:"options,omitempty"`
	// Px params are in design pixels and scale with the render size.
	Px bool `json:"px,omitempty"`
}

func num(name string, def, lo, hi float64) Param {
	return Param{Name: name, Kind: "float", Default: def, Min: lo, Max: hi}
}

func px(name string, def, lo, hi float64) Param {
	p := num(name, def, lo, hi)
	p.Px = true
	return p
}

func integer(name string, def, lo, hi int) Param {
	return Param{Name: name, Kind: "int", Default: def, Min: float64(lo), Max: float64(hi)}
}

func enum(name, def string, options ...string) Param {
	return Param{Name: name, Kind: "enum", Default: def, Options: options}
}

func color(name, def string) Param {
	return Param{Name: name, Kind: "color", Default: def}
}

func flag(name string, def bool) Param {
	return Param{Name: name, Kind: "bool", Default: def}
}

func text(name, def string) Param {
	return Param{Name: name, Kind: "string", Default: def}
}

// data is a structured param edited as JSON or with a dedicated widget.
func data(name string, def any) Param {
	return Param{Name: name, Kind: "json", Default: def}
}

var seedParam = integer("seed", 0, 0, 9999)

// Evals returns the evaluation functions for the graph evaluator.
func Evals() map[string]graph.EvalFunc {
	out := make(map[string]graph.EvalFunc, len(Registry))
	for name, d := range Registry {
		out[name] = d.Eval
	}
	return out
}
