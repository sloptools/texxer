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

func TestSharedCache(t *testing.T) {
	calls := 0
	nodes := map[string]EvalFunc{
		"count": func(e *Env) (*img.Buffer, error) {
			calls++
			return constBuf(e, 1), nil
		},
		"add": testNodes["add"],
	}

	graphWith := func(v float64) *Graph {
		return &Graph{Nodes: map[string]Node{
			"src": {Type: "count"},
			"k":   {Type: "count", Params: Params{"v": v}},
			"sum": {Type: "add", Inputs: map[string]string{"a": "src", "b": "k"}},
		}}
	}

	cache := NewCache(1 << 20)
	for _, v := range []float64{1, 1, 2} {
		ev := NewEvaluator(graphWith(v), nodes, nil)
		ev.UseCache(cache)
		if _, err := ev.Eval("sum", testCtx); err != nil {
			t.Fatal(err)
		}
	}

	// First run evaluates src and k, the repeat hits the cache, the change re-evaluates only k.
	if calls != 3 {
		t.Errorf("got %d evaluations, want 3", calls)
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

func TestSubgraphNodes(t *testing.T) {
	sub := &Graph{Nodes: map[string]Node{
		"fallback": {Type: "const", Params: Params{"value": 5.0}},
		"x":        {Type: "input", Params: Params{"name": "x"}, Inputs: map[string]string{"default": "fallback"}},
		"k":        {Type: "const", Params: Params{"value": 1.0}},
		"sum":      {Type: "add", Inputs: map[string]string{"a": "x", "b": "k"}},
		"main":     {Type: "output", Inputs: map[string]string{"in": "sum"}},
		"side":     {Type: "output", Params: Params{"name": "side"}, Inputs: map[string]string{"in": "k"}},
	}}
	load := func(name string) (*Graph, error) {
		if name == "sub" {
			return sub, nil
		}
		return nil, nil
	}

	g := &Graph{Nodes: map[string]Node{
		"two":    {Type: "const", Params: Params{"value": 2.0}},
		"wired":  {Type: "sub", Inputs: map[string]string{"x": "two"}},
		"loose":  {Type: "sub"},
		"direct": {Type: "input", Params: Params{"name": "x"}, Inputs: map[string]string{"default": "two"}},
	}}

	cases := map[string]float32{"wired": 3, "loose": 6, "wired.side": 1, "direct": 2}
	for ref, want := range cases {
		if v := eval(t, g, load, 0, ref); v != want {
			t.Errorf("%s: got %v, want %v", ref, v, want)
		}
	}
}

func TestSubgraphEditInvalidatesCache(t *testing.T) {
	sub := &Graph{Nodes: map[string]Node{
		"k":   {Type: "const", Params: Params{"value": 1.0}},
		"out": {Type: "output", Inputs: map[string]string{"in": "k"}},
	}}
	load := func(name string) (*Graph, error) {
		if name == "sub" {
			return sub, nil
		}
		return nil, nil
	}
	g := &Graph{Nodes: map[string]Node{"s": {Type: "sub"}}}

	cache := NewCache(1 << 20)
	run := func() float32 {
		ev := NewEvaluator(g, testNodes, load)
		ev.UseCache(cache)
		b, err := ev.Eval("s", testCtx)
		if err != nil {
			t.Fatal(err)
		}
		return b.Pix[0]
	}

	run()
	sub.Nodes["k"] = Node{Type: "const", Params: Params{"value": 7.0}}
	if v := run(); v != 7 {
		t.Errorf("got %v after editing the subgraph, want 7", v)
	}
}
