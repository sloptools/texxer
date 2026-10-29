package graph

import (
	"fmt"
	"path/filepath"

	"texxer/internal/img"
)

type Ctx struct {
	W, H   int
	Tiling img.Tiling
	// Scale is render size / design size.
	Scale float64
}

type EvalFunc func(e *Env) (*img.Buffer, error)

type cacheKey struct {
	id  string
	ctx Ctx
}

type Evaluator struct {
	graph *Graph
	nodes map[string]EvalFunc
	cache map[cacheKey]*img.Buffer
	busy  map[cacheKey]bool
}

func NewEvaluator(g *Graph, nodes map[string]EvalFunc) *Evaluator {
	return &Evaluator{
		graph: g,
		nodes: nodes,
		cache: map[cacheKey]*img.Buffer{},
		busy:  map[cacheKey]bool{},
	}
}

func (ev *Evaluator) Eval(id string, ctx Ctx) (*img.Buffer, error) {
	key := cacheKey{id, ctx}
	if b, ok := ev.cache[key]; ok {
		return b, nil
	}
	if ev.busy[key] {
		return nil, fmt.Errorf("cycle at node %q", id)
	}

	node, ok := ev.graph.Nodes[id]
	if !ok {
		return nil, fmt.Errorf("unknown node %q", id)
	}
	fn, ok := ev.nodes[node.Type]
	if !ok {
		return nil, fmt.Errorf("node %q: unknown type %q", id, node.Type)
	}

	ev.busy[key] = true
	defer delete(ev.busy, key)

	b, err := fn(&Env{Ctx: ctx, Params: node.Params, node: node, ev: ev})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", id, err)
	}

	ev.cache[key] = b
	return b, nil
}

type Env struct {
	Ctx    Ctx
	Params Params
	node   Node
	ev     *Evaluator
}

// Input evaluates a connected input at the node's context. Returns nil if unconnected.
func (e *Env) Input(name string) (*img.Buffer, error) {
	return e.InputAt(name, e.Ctx)
}

func (e *Env) InputAt(name string, ctx Ctx) (*img.Buffer, error) {
	id, ok := e.node.Inputs[name]
	if !ok {
		return nil, nil
	}
	return e.ev.Eval(id, ctx)
}

func (e *Env) Required(name string) (*img.Buffer, error) {
	b, err := e.Input(name)
	if err == nil && b == nil {
		err = fmt.Errorf("input %q not connected", name)
	}
	return b, err
}

// Px reads a pixel-sized param and scales it to the render resolution.
func (e *Env) Px(key string, def float64) float64 {
	return e.Params.Float(key, def) * e.Ctx.Scale
}

func (e *Env) Path(rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(e.ev.graph.Dir, rel)
}
