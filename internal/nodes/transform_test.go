package nodes

import (
	"slices"
	"testing"

	"texxer/internal/graph"
	"texxer/internal/img"
)

func TestTransform(t *testing.T) {
	ctx := graph.Ctx{W: 32, H: 32, Tiling: img.TileBoth, Scale: 1}
	src := map[string]graph.Node{"src": {Type: "noise", Params: graph.Params{"scale": 4.0}}}

	eval := func(nodes map[string]graph.Node, ref string) []float32 {
		t.Helper()
		all := map[string]graph.Node{}
		for k, v := range src {
			all[k] = v
		}
		for k, v := range nodes {
			all[k] = v
		}
		b, err := graph.NewEvaluator(&graph.Graph{Nodes: all}, Evals(), nil).Eval(ref, ctx)
		if err != nil {
			t.Fatal(err)
		}
		return b.Pix
	}

	want := eval(nil, "src")

	turned := map[string]graph.Node{
		"r1": {Type: "transform", Params: graph.Params{"rotate": "90"}, Inputs: map[string]string{"in": "src"}},
		"r2": {Type: "transform", Params: graph.Params{"rotate": "90"}, Inputs: map[string]string{"in": "r1"}},
		"r3": {Type: "transform", Params: graph.Params{"rotate": "180"}, Inputs: map[string]string{"in": "r2"}},
	}
	if !slices.Equal(eval(turned, "r3"), want) {
		t.Error("four quarter turns should give the input back")
	}

	once := map[string]graph.Node{"rep": {Type: "repeat", Params: graph.Params{"count_x": 1.0, "count_y": 1.0}, Inputs: map[string]string{"in": "src"}}}
	if !slices.Equal(eval(once, "rep"), want) {
		t.Error("repeat 1x1 should give the input back")
	}

	twice := eval(map[string]graph.Node{"rep": {Type: "repeat", Inputs: map[string]string{"in": "src"}}}, "rep")
	for y := range 16 {
		for x := range 16 {
			if twice[y*32+x] != twice[(y+16)*32+x+16] {
				t.Fatalf("repeat 2x2 cells differ at %d,%d", x, y)
			}
		}
	}
}

func TestGridSplits(t *testing.T) {
	b, err := splits([]float64{1, 3}, 512)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(b, []float64{0, 128, 512}) {
		t.Errorf("got %v", b)
	}
	if span(b, 127) != 0 || span(b, 128) != 1 || span(b, 600) != 1 {
		t.Error("span picks the wrong interval")
	}
}
