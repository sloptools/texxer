package nodes

import (
	"encoding/json"
	"slices"
	"testing"

	"texxer/internal/graph"
	"texxer/internal/img"
)

// TestDefaults renders every node once without params and once with the
// catalog defaults spelled out. Both must match, so the editor shows what the code does.
func TestDefaults(t *testing.T) {
	ctx := graph.Ctx{W: 64, H: 64, Tiling: img.TileBoth, Scale: 0.25}

	for name, def := range Registry {
		// Nodes that read files have no meaningful defaults.
		if name == "image" || name == "mesh" || name == "layers" {
			continue
		}

		inputs := def.Inputs
		if def.Dynamic {
			inputs = []string{"s1", "s2"}
		}

		implicit := render(t, name, inputs, graph.Params{}, ctx)
		explicit := render(t, name, inputs, defaults(t, def), ctx)

		if !slices.Equal(implicit.Pix, explicit.Pix) {
			t.Errorf("%s: catalog defaults differ from code defaults", name)
		}
	}
}

func render(t *testing.T, name string, inputs []string, params graph.Params, ctx graph.Ctx) *img.Buffer {
	t.Helper()

	g := &graph.Graph{Nodes: map[string]graph.Node{
		"src": {Type: "noise", Params: graph.Params{"scale": 4.0}},
		"n":   {Type: name, Params: params, Inputs: map[string]string{}},
	}}
	for _, in := range inputs {
		g.Nodes["n"].Inputs[in] = "src"
	}

	b, err := graph.NewEvaluator(g, Evals(), nil).Eval("n", ctx)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return b
}

// defaults returns the catalog defaults as they would arrive from JSON.
func defaults(t *testing.T, def Def) graph.Params {
	t.Helper()

	p := graph.Params{}
	for _, param := range def.Params {
		p[param.Name] = param.Default
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}

	var out graph.Params
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}
