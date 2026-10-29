package nodes

import (
	"math"
	"testing"

	"texxer/internal/graph"
	"texxer/internal/img"
)

func TestLayers(t *testing.T) {
	ctx := graph.Ctx{W: 16, H: 16, Tiling: img.TileBoth, Scale: 1}
	half := 0.5

	stack := []any{
		map[string]any{"name": "red", "fill": map[string]any{"color": "#ff0000"}},
		map[string]any{"name": "blue", "fill": map[string]any{"color": "#0000ff"}, "opacity": half,
			"mask": []any{map[string]any{"kind": "noise", "type": "color", "params": map[string]any{"color": "#ffffff"}}}},
		map[string]any{"name": "off", "fill": map[string]any{"color": "#00ff00"}, "hidden": true},
	}
	g := &graph.Graph{Nodes: map[string]graph.Node{
		"l": {Type: "layers", Params: graph.Params{"layers": stack}},
	}}

	ev := graph.NewEvaluator(g, Evals(), nil)
	b, err := ev.Eval("l", ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Red, then blue at half opacity over all of it; the hidden layer is skipped.
	want := [3]float32{0.5, 0, 0.5}
	for ch, v := range want {
		if math.Abs(float64(b.Value(0, ch)-v)) > 1e-5 {
			t.Errorf("channel %d: got %v, want %v", ch, b.Value(0, ch), v)
		}
	}

	if m, err := ev.Eval("l.mask1", ctx); err != nil || m.Pix[0] != 1 {
		t.Errorf("mask1: %v, %v", m, err)
	}
	if r, err := ev.Eval("l.layer0", ctx); err != nil || r.Value(0, 0) != 1 || r.Value(0, 2) != 0 {
		t.Errorf("layer0 should be pure red: %v", err)
	}
}

func TestLayersNeedModelForPartMasks(t *testing.T) {
	stack := []any{map[string]any{"name": "x", "fill": map[string]any{"color": "#ffffff"},
		"mask": []any{map[string]any{"kind": "parts", "parts": []any{"Hull"}}}}}
	g := &graph.Graph{Nodes: map[string]graph.Node{"l": {Type: "layers", Params: graph.Params{"layers": stack}}}}

	_, err := graph.NewEvaluator(g, Evals(), nil).Eval("l", graph.Ctx{W: 4, H: 4, Scale: 1})
	if err == nil {
		t.Error("expected an error for a parts mask without a model")
	}
}
