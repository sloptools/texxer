package nodes

import (
	"fmt"
	"slices"

	"texxer/internal/graph"
	"texxer/internal/img"
)

// Layer stacks, painter style: each layer fills a material through a mask,
// bottom to top, then effects such as baked light run over the result.
// Post layers go on after the effects, for glass or fullbright parts.

type layer struct {
	Name    string     `json:"name"`
	Fill    layerFill  `json:"fill"`
	Mask    []maskPart `json:"mask"`
	Opacity *float64   `json:"opacity,omitempty"`
	Mode    string     `json:"mode,omitempty"`
	Hidden  bool       `json:"hidden,omitempty"`
	Post    bool       `json:"post,omitempty"`
}

// layerFill is a flat color or a node type, usually a material graph.
type layerFill struct {
	Color  string       `json:"color,omitempty"`
	Type   string       `json:"type,omitempty"`
	Params graph.Params `json:"params,omitempty"`
}

// maskPart is one step of a layer mask; steps combine left to right.
type maskPart struct {
	// Kind is parts, mesh, noise, color or graph.
	Kind string `json:"kind"`
	// Op combines with the mask so far: multiply (default), add or subtract.
	Op     string `json:"op,omitempty"`
	Invert bool   `json:"invert,omitempty"`

	Parts []string `json:"parts,omitempty"`
	// Source is a mesh map for mesh masks: edges, cavity, ao, up, down or height.
	Source string `json:"source,omitempty"`
	// Type and Params set the node for noise and graph masks.
	Type   string       `json:"type,omitempty"`
	Params graph.Params `json:"params,omitempty"`
	Color  string       `json:"color,omitempty"`

	Level     *float64 `json:"level,omitempty"`
	Softness  *float64 `json:"softness,omitempty"`
	Tolerance *float64 `json:"tolerance,omitempty"`
	// Breakup roughens mesh masks with a noise, 0 for clean edges.
	Breakup *float64 `json:"breakup,omitempty"`
	// Grow widens color picks by px, e.g. to take the stripes between picked ones.
	Grow *float64 `json:"grow,omitempty"`
}

type effect struct {
	Type   string       `json:"type"`
	Params graph.Params `json:"params,omitempty"`
	Hidden bool         `json:"hidden,omitempty"`
}

var layersDef = Def{
	Eval:     layersNode,
	Category: "mesh",
	Inputs:   []string{"in"},
	Params: append(slices.Clone(meshParams),
		data("layers", []layer{}),
		data("effects", []effect{}),
	),
}

var partMaskDef = Def{
	Eval:     partMaskNode,
	Category: "mesh",
	Inputs:   []string{"id"},
	Params:   []Param{data("ids", []int{})},
}

// layersNode builds its stack into a graph and renders it. Besides "out" it
// outputs "mask<i>" and "layer<i>", the mask of each layer and the result
// after it, for previews while painting.
func layersNode(e *graph.Env) (*img.Buffer, error) {
	var stack []layer
	var effects []effect
	if err := e.Params.Decode("layers", &stack); err != nil {
		return nil, err
	}
	if err := e.Params.Decode("effects", &effects); err != nil {
		return nil, err
	}

	b := layerBuilder{env: e, nodes: map[string]graph.Node{}}
	if err := b.mesh(); err != nil {
		return nil, err
	}
	out, err := b.build(stack, effects)
	if err != nil {
		return nil, err
	}

	refs := []string{out}
	var names []string
	for i := range stack {
		if ref, ok := b.masks[i]; ok {
			refs = append(refs, ref)
			names = append(names, fmt.Sprintf("mask%d", i))
		}
		if ref, ok := b.results[i]; ok {
			refs = append(refs, ref)
			names = append(names, fmt.Sprintf("layer%d", i))
		}
	}

	g := &graph.Graph{Output: out, Nodes: b.nodes}
	bufs, err := e.Generated(g, refs...)
	if err != nil {
		return nil, err
	}
	for i, name := range names {
		e.SetOutput(name, bufs[i+1])
	}
	return bufs[0], nil
}

type layerBuilder struct {
	env     *graph.Env
	nodes   map[string]graph.Node
	n       int
	hasMesh bool
	parts   []string

	masks, results map[int]string
}

func (b *layerBuilder) add(prefix, typ string, params graph.Params, inputs map[string]string) string {
	b.n++
	id := fmt.Sprintf("%s%d", prefix, b.n)
	b.nodes[id] = graph.Node{Type: typ, Params: params, Inputs: inputs}
	return id
}

// mesh adds the bake, configured by the mesh params this node shares with
// the mesh node. Without a model the stack works on plain textures.
func (b *layerBuilder) mesh() error {
	path := b.env.Params.String("path", "")
	if path == "" {
		return nil
	}
	b.hasMesh = true

	params := graph.Params{}
	for _, p := range meshParams {
		if v, ok := b.env.Params[p.Name]; ok {
			params[p.Name] = v
		}
	}
	b.nodes["mesh"] = graph.Node{Type: "mesh", Params: params}

	parts, err := modelParts(b.env.Path(path))
	if err != nil {
		return err
	}
	b.parts = parts
	return nil
}

// wiring connects every mesh map, plus the color so far, to a material, mask or effect.
// Nodes read the inputs they know and ignore the rest.
func (b *layerBuilder) wiring(color string) map[string]string {
	in := map[string]string{}
	if b.hasMesh {
		in["texture"] = "mesh"
		for _, m := range meshMaps {
			in[m] = "mesh." + m
		}
	}
	if color != "" {
		in["color"], in["in"] = color, color
	}
	return in
}

func (b *layerBuilder) build(stack []layer, effects []effect) (string, error) {
	b.masks, b.results = map[int]string{}, map[int]string{}

	cur := b.add("base", "color", graph.Params{"color": "#808080"}, nil)
	if b.env.HasInput("in") {
		b.nodes["in"] = graph.Node{Type: "input", Params: graph.Params{"name": "in"}}
		cur = "in"
	}

	apply := func(post bool) error {
		for i, l := range stack {
			if l.Hidden || l.Post != post {
				continue
			}
			next, err := b.layer(i, l, cur)
			if err != nil {
				return fmt.Errorf("layer %q: %w", l.Name, err)
			}
			cur = next
			b.results[i] = cur
		}
		return nil
	}

	if err := apply(false); err != nil {
		return "", err
	}
	for _, fx := range effects {
		if fx.Hidden {
			continue
		}
		if !b.env.HasType(fx.Type) {
			return "", fmt.Errorf("effect: unknown type %q", fx.Type)
		}
		cur = b.add("fx", fx.Type, fx.Params, b.wiring(cur))
	}
	if err := apply(true); err != nil {
		return "", err
	}
	return cur, nil
}

func (b *layerBuilder) layer(i int, l layer, below string) (string, error) {
	var src string
	switch {
	case l.Fill.Type != "":
		if !b.env.HasType(l.Fill.Type) {
			return "", fmt.Errorf("unknown material %q", l.Fill.Type)
		}
		src = b.add("fill", l.Fill.Type, l.Fill.Params, b.wiring(below))
	default:
		c := l.Fill.Color
		if c == "" {
			c = "#808080"
		}
		src = b.add("fill", "color", graph.Params{"color": c}, nil)
	}

	mask, err := b.mask(l.Mask)
	if err != nil {
		return "", err
	}
	if mask != "" {
		b.masks[i] = mask
	}

	params := graph.Params{"mode": "normal"}
	if l.Mode != "" {
		params["mode"] = l.Mode
	}
	if l.Opacity != nil {
		params["opacity"] = *l.Opacity
	}

	inputs := map[string]string{"a": below, "b": src}
	if mask != "" {
		inputs["mask"] = mask
	}
	return b.add("layer", "blend", params, inputs), nil
}

// mask combines the parts of a layer mask; "" means the whole surface.
func (b *layerBuilder) mask(steps []maskPart) (string, error) {
	cur := ""
	for k, s := range steps {
		m, err := b.maskStep(s)
		if err != nil {
			return "", fmt.Errorf("mask %d: %w", k+1, err)
		}
		if s.Invert {
			m = b.add("inv", "levels", graph.Params{"invert": true}, map[string]string{"in": m})
		}
		if cur == "" {
			cur = m
			continue
		}

		mode := map[string]string{"": "multiply", "multiply": "multiply", "add": "lighten", "subtract": "subtract"}[s.Op]
		if mode == "" {
			return "", fmt.Errorf("mask %d: unknown op %q", k+1, s.Op)
		}
		cur = b.add("comb", "blend", graph.Params{"mode": mode}, map[string]string{"a": cur, "b": m})
	}
	return cur, nil
}

func (b *layerBuilder) maskStep(s maskPart) (string, error) {
	if !b.hasMesh && (s.Kind == "parts" || s.Kind == "mesh" || s.Kind == "color") {
		return "", fmt.Errorf("%s masks need a model path", s.Kind)
	}
	level := func(def float64) float64 { return deref(s.Level, def) }
	soft := func(def float64) float64 { return deref(s.Softness, def) }

	switch s.Kind {
	case "parts":
		var ids []any
		for _, name := range s.Parts {
			i := slices.Index(b.parts, name)
			if i < 0 {
				return "", fmt.Errorf("model has no part %q", name)
			}
			ids = append(ids, float64(i))
		}
		return b.add("parts", "partmask", graph.Params{"ids": ids}, map[string]string{"id": "mesh.partid"}), nil

	case "mesh":
		src, invert := s.Source, false
		switch src {
		case "edges", "cavity", "up", "height":
		case "ao":
			// Occluded areas, so the mask reads like cavity.
			invert = true
		case "down":
			src, invert = "up", true
		default:
			return "", fmt.Errorf("unknown mesh source %q", s.Source)
		}

		height := "mesh." + src
		if invert {
			height = b.add("src", "levels", graph.Params{"invert": true}, map[string]string{"in": height})
		}
		params := graph.Params{"mode": "height", "level": level(0.4), "softness": soft(0.15), "breakup": deref(s.Breakup, 0.8)}
		inputs := map[string]string{"height": height}
		if deref(s.Breakup, 0.8) > 0 {
			inputs["breakup"] = b.add("brk", "noise", graph.Params{"scale": 48.0, "octaves": 4.0, "persistence": 0.6, "seed": float64(b.n)}, nil)
		}
		return b.add("mesh", "mask", params, inputs), nil

	case "noise":
		typ := s.Type
		if typ == "" {
			typ = "dirt"
		}
		if !b.env.HasType(typ) {
			return "", fmt.Errorf("unknown noise %q", typ)
		}
		src := b.add("noise", typ, s.Params, nil)
		if s.Level == nil {
			return src, nil
		}
		return b.add("cut", "scan", graph.Params{"position": level(0.5), "contrast": 1 - soft(0.2)}, map[string]string{"in": src}), nil

	case "color":
		raw := b.add("pick", "colorselect", graph.Params{"color": s.Color, "mode": "rgb", "tolerance": deref(s.Tolerance, 0.05), "softness": 0.02, "min_saturation": 0.0},
			map[string]string{"in": "mesh"})
		// Smoothed, so picks from low-res textures don't show pixel steps.
		radius, pos := 1.5, 0.5
		if g := deref(s.Grow, 0); g > 0 {
			radius, pos = g, 0.12
		}
		blurred := b.add("pick", "blur", graph.Params{"radius": radius}, map[string]string{"in": raw})
		return b.add("pick", "scan", graph.Params{"position": pos, "contrast": 0.85}, map[string]string{"in": blurred}), nil

	case "graph":
		if !b.env.HasType(s.Type) {
			return "", fmt.Errorf("unknown mask graph %q", s.Type)
		}
		return b.add("mg", s.Type, s.Params, b.wiring("")), nil
	}
	return "", fmt.Errorf("unknown mask kind %q", s.Kind)
}

// partMaskNode is white where the part id map holds one of ids.
func partMaskNode(e *graph.Env) (*img.Buffer, error) {
	src, err := e.Required("id")
	if err != nil {
		return nil, err
	}

	var ids []int
	if err := e.Params.Decode("ids", &ids); err != nil {
		return nil, err
	}
	want := map[int]bool{}
	for _, id := range ids {
		want[id] = true
	}

	out := img.New(src.W, src.H, 1, src.Tiling)
	for i := range out.Pix {
		if want[int(src.Value(i, 0)*256)] {
			out.Pix[i] = 1
		}
	}
	return out, nil
}

func deref(v *float64, def float64) float64 {
	if v == nil {
		return def
	}
	return *v
}
