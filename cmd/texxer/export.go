package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"texxer/internal/graph"
	"texxer/internal/mesh"
	"texxer/internal/nodes"
)

func exportCmd(args []string) error {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	out := fs.String("o", "", "output .glb (default: graph name next to the graph)")
	size := fs.Int("size", 0, "texture size in px (default: graph size)")
	seed := fs.Int("seed", 0, "variant seed")
	fs.Parse(args)

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: texxer export [-o out.glb] [-size N] graph.json")
		os.Exit(2)
	}
	path := fs.Arg(0)
	if *out == "" {
		*out = path[:len(path)-len(filepath.Ext(path))] + ".glb"
	}

	g, err := graph.Load(path)
	if err != nil {
		return err
	}
	data, err := exportGLB(g, path, *size, *seed)
	if err != nil {
		return err
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		return err
	}
	fmt.Printf("%s: %d KB\n", *out, len(data)/1024)
	return nil
}

// exportGLB renders a graph and puts the texture into the model it is built on.
func exportGLB(g *graph.Graph, path string, size, seed int) ([]byte, error) {
	model, material, ok := nodes.ModelOf(g)
	if !ok {
		return nil, errors.New("graph has no layers or mesh node with a model")
	}

	b, err := evalGraph(path, "", size, seed)
	if err != nil {
		return nil, err
	}
	var tex bytes.Buffer
	if err := png.Encode(&tex, b.Image()); err != nil {
		return nil, err
	}

	if !filepath.IsAbs(model) {
		model = filepath.Join(filepath.Dir(path), model)
	}
	src, err := os.ReadFile(model)
	if err != nil {
		return nil, err
	}
	return mesh.WithTexture(src, tex.Bytes(), material)
}
