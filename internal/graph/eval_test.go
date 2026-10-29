package graph

import (
	"testing"

	"texxer/internal/img"
)

// Test nodes: "const" fills its value param plus the seed, "pair" also outputs value+1, "add" sums a and b.
var testNodes = map[string]EvalFunc{
	"const": func(e *Env) (*img.Buffer, error) {
		return constBuf(e, float32(e.Params.Float("value", 0))+float32(e.Seed())), nil
	},
	"pair": func(e *Env) (*img.Buffer, error) {
		v := float32(e.Params.Float("value", 0))
		e.SetOutput("plus", constBuf(e, v+1))
		return constBuf(e, v), nil
	},
	"add": func(e *Env) (*img.Buffer, error) {
		a, err := e.Required("a")
		if err != nil {
			return nil, err
		}
		b, err := e.Required("b")
		if err != nil {
			return nil, err
		}
		return constBuf(e, a.Pix[0]+b.Pix[0]), nil
	},
}

func constBuf(e *Env, v float32) *img.Buffer {
	b := img.New(e.Ctx.W, e.Ctx.H, 1, e.Ctx.Tiling)
	for i := range b.Pix {
		b.Pix[i] = v
	}
	return b
}

var testCtx = Ctx{W: 2, H: 2, Scale: 1}

func eval(t *testing.T, g *Graph, load Loader, seed int, ref string) float32 {
	t.Helper()

	ev := NewEvaluator(g, testNodes, load)
	ev.SetSeed(seed)

	b, err := ev.Eval(ref, testCtx)
	if err != nil {
		t.Fatal(err)
	}
	return b.Pix[0]
}

func TestOutputs(t *testing.T) {
	g := &Graph{Nodes: map[string]Node{
		"p":   {Type: "pair", Params: Params{"value": 2.0}},
		"sum": {Type: "add", Inputs: map[string]string{"a": "p", "b": "p.plus"}},
	}}

	if v := eval(t, g, nil, 0, "sum"); v != 5 {
		t.Errorf("got %v, want 5", v)
	}
}

func TestSubgraph(t *testing.T) {
	sub := &Graph{
		Outputs: map[string]string{"out": "sum"},
		Exposed: map[string]Exposed{"offset": {Default: 10.0, Targets: []string{"k.value"}}},
		Nodes: map[string]Node{
			"x":   {Type: "input", Params: Params{"name": "x"}},
			"k":   {Type: "const"},
			"sum": {Type: "add", Inputs: map[string]string{"a": "x", "b": "k"}},
		},
	}
	load := func(name string) (*Graph, error) {
		if name == "shift" {
			return sub, nil
		}
		return nil, nil
	}

	g := &Graph{Nodes: map[string]Node{
		"one":     {Type: "const", Params: Params{"value": 1.0}},
		"default": {Type: "shift", Inputs: map[string]string{"x": "one"}},
		"custom":  {Type: "shift", Params: Params{"offset": 100.0}, Inputs: map[string]string{"x": "one"}},
		"seeded":  {Type: "shift", Params: Params{"seed": 1.0}, Inputs: map[string]string{"x": "one"}},
		"typo":    {Type: "shift", Params: Params{"ofset": 1.0}, Inputs: map[string]string{"x": "one"}},
	}}

	if v := eval(t, g, load, 0, "default"); v != 11 {
		t.Errorf("default: got %v, want 11", v)
	}
	if v := eval(t, g, load, 0, "custom"); v != 101 {
		t.Errorf("custom: got %v, want 101", v)
	}
	if v := eval(t, g, load, 0, "seeded"); v != 11+seedStride {
		t.Errorf("seeded: got %v, want %v", v, 11+seedStride)
	}

	if _, err := NewEvaluator(g, testNodes, load).Eval("typo", testCtx); err == nil {
		t.Error("typo: expected unknown param error")
	}
}
