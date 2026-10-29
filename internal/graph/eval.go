package graph

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"texxer/internal/img"
)

type Ctx struct {
	W, H   int
	Tiling img.Tiling
	// Scale is render size / design size.
	Scale float64
}

type EvalFunc func(e *Env) (*img.Buffer, error)

// Loader resolves node types that aren't built in, such as presets.
// It returns nil without error for unknown names.
type Loader func(name string) (*Graph, error)

const (
	maxDepth   = 16
	seedStride = 7919
)

type nodeKey struct {
	id  string
	ctx Ctx
}

type outKey struct {
	nodeKey
	out string
}

type Evaluator struct {
	graph *Graph
	nodes map[string]EvalFunc
	load  Loader

	// Set when evaluating a subgraph.
	parent *Env
	depth  int

	// seed offsets every seed in the graph, for variants.
	seed int

	cache    map[outKey]*img.Buffer
	busy     map[nodeKey]bool
	children map[nodeKey]*Evaluator
}

func NewEvaluator(g *Graph, nodes map[string]EvalFunc, load Loader) *Evaluator {
	return &Evaluator{
		graph:    g,
		nodes:    nodes,
		load:     load,
		cache:    map[outKey]*img.Buffer{},
		busy:     map[nodeKey]bool{},
		children: map[nodeKey]*Evaluator{},
	}
}

// SetSeed offsets every seed in the graph to render a variant.
func (ev *Evaluator) SetSeed(seed int) {
	ev.seed = seed * seedStride
}

// Eval evaluates a reference of the form "node" or "node.output".
func (ev *Evaluator) Eval(ref string, ctx Ctx) (*img.Buffer, error) {
	id, out, _ := strings.Cut(ref, ".")
	if out == "" {
		out = "out"
	}

	nk := nodeKey{id, ctx}
	if b, ok := ev.cache[outKey{nk, out}]; ok {
		return b, nil
	}

	node, ok := ev.graph.Nodes[id]
	if !ok {
		return nil, fmt.Errorf("unknown node %q", id)
	}
	if ev.busy[nk] {
		return nil, fmt.Errorf("cycle at node %q", id)
	}

	ev.busy[nk] = true
	defer delete(ev.busy, nk)

	b, err := ev.evalNode(nk, node, out)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", id, err)
	}

	ev.cache[outKey{nk, out}] = b
	return b, nil
}

func (ev *Evaluator) evalNode(nk nodeKey, node Node, out string) (*img.Buffer, error) {
	env := &Env{Ctx: nk.ctx, Params: node.Params, node: node, ev: ev}

	if node.Type == "input" {
		return ev.parentInput(env)
	}

	if fn, ok := ev.nodes[node.Type]; ok {
		b, err := fn(env)
		if err != nil {
			return nil, err
		}

		ev.cache[outKey{nk, "out"}] = b
		for name, extra := range env.outputs {
			ev.cache[outKey{nk, name}] = extra
		}

		if out == "out" {
			return b, nil
		}
		if extra, ok := env.outputs[out]; ok {
			return extra, nil
		}
		return nil, fmt.Errorf("no output %q", out)
	}

	child, err := ev.child(nk, env)
	if err != nil {
		return nil, err
	}

	ref, ok := child.graph.Outputs[out]
	if !ok {
		return nil, fmt.Errorf("no output %q", out)
	}
	return child.Eval(ref, nk.ctx)
}

func (ev *Evaluator) parentInput(env *Env) (*img.Buffer, error) {
	if ev.parent == nil {
		return nil, errors.New("input node outside a subgraph")
	}

	name := env.Params.String("name", "")
	b, err := ev.parent.InputAt(name, env.Ctx)
	if err == nil && b == nil {
		err = fmt.Errorf("input %q not connected", name)
	}
	return b, err
}

// child returns the evaluator for a subgraph node, created once per node and context.
func (ev *Evaluator) child(nk nodeKey, env *Env) (*Evaluator, error) {
	if c, ok := ev.children[nk]; ok {
		return c, nil
	}

	if ev.load == nil || ev.depth >= maxDepth {
		return nil, fmt.Errorf("unknown type %q", env.node.Type)
	}
	sub, err := ev.load(env.node.Type)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, fmt.Errorf("unknown type %q", env.node.Type)
	}

	g, err := sub.withParams(env.Params)
	if err != nil {
		return nil, err
	}

	c := NewEvaluator(g, ev.nodes, ev.load)
	c.parent = env
	c.depth = ev.depth + 1
	c.seed = ev.seed + env.Params.Int("seed", 0)*seedStride

	ev.children[nk] = c
	return c, nil
}

type Env struct {
	Ctx     Ctx
	Params  Params
	node    Node
	ev      *Evaluator
	outputs map[string]*img.Buffer
}

// Input evaluates a connected input at the node's context. Returns nil if unconnected.
func (e *Env) Input(name string) (*img.Buffer, error) {
	return e.InputAt(name, e.Ctx)
}

func (e *Env) InputAt(name string, ctx Ctx) (*img.Buffer, error) {
	ref, ok := e.node.Inputs[name]
	if !ok {
		return nil, nil
	}
	return e.ev.Eval(ref, ctx)
}

func (e *Env) Required(name string) (*img.Buffer, error) {
	b, err := e.Input(name)
	if err == nil && b == nil {
		err = fmt.Errorf("input %q not connected", name)
	}
	return b, err
}

// SetOutput adds a named output next to the node's main "out" result.
func (e *Env) SetOutput(name string, b *img.Buffer) {
	if e.outputs == nil {
		e.outputs = map[string]*img.Buffer{}
	}
	e.outputs[name] = b
}

// Seed is the node's seed param shifted by the graph variant seed.
func (e *Env) Seed() int {
	return e.Params.Int("seed", 0) + e.ev.seed
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
