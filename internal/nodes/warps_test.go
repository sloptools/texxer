package nodes

import (
	"math"
	"testing"

	"texxer/internal/graph"
	"texxer/internal/img"
)

func TestFlatSlopeKeepsInput(t *testing.T) {
	ctx := graph.Ctx{W: 32, H: 32, Tiling: img.TileBoth, Scale: 1}
	g := &graph.Graph{Nodes: map[string]graph.Node{
		"src":  {Type: "noise", Params: graph.Params{"scale": 4.0}},
		"flat": {Type: "color"},
		"sb":   {Type: "slopeblur", Inputs: map[string]string{"in": "src", "slope": "flat"}},
		"gw":   {Type: "gradwarp", Inputs: map[string]string{"in": "src", "intensity": "flat"}},
	}}

	want, err := graph.NewEvaluator(g, Evals(), nil).Eval("src", ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"sb", "gw"} {
		b, err := graph.NewEvaluator(g, Evals(), nil).Eval(ref, ctx)
		if err != nil {
			t.Fatal(err)
		}
		for i, v := range b.Pix {
			if math.Abs(float64(v-want.Pix[i])) > 1e-5 {
				t.Fatalf("%s changed the input over a flat slope", ref)
			}
		}
	}
}

func TestScan(t *testing.T) {
	ctx := graph.Ctx{W: 2, H: 2, Tiling: img.TileBoth, Scale: 1}
	for _, c := range []struct {
		gray string
		want float32
	}{{"#404040", 0}, {"#c0c0c0", 1}} {
		g := &graph.Graph{Nodes: map[string]graph.Node{
			"c": {Type: "color", Params: graph.Params{"color": c.gray}},
			"s": {Type: "scan", Params: graph.Params{"position": 0.5, "contrast": 0.9}, Inputs: map[string]string{"in": "c"}},
		}}
		b, err := graph.NewEvaluator(g, Evals(), nil).Eval("s", ctx)
		if err != nil {
			t.Fatal(err)
		}
		if b.Pix[0] != c.want {
			t.Errorf("%s: got %v, want %v", c.gray, b.Pix[0], c.want)
		}
	}
}
