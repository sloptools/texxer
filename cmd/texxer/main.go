package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"
	"time"

	"texxer/internal/graph"
	"texxer/internal/img"
	"texxer/internal/nodes"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "render" {
		fmt.Fprintln(os.Stderr, "usage: texxer render [flags] graph.json")
		os.Exit(2)
	}

	fs := flag.NewFlagSet("render", flag.ExitOnError)
	out := fs.String("o", "out.png", "output file")
	size := fs.Int("size", 0, "render size in px (default: graph size)")
	node := fs.String("node", "", "render this node instead of the graph output")
	fs.Parse(os.Args[2:])

	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(2)
	}

	if err := render(fs.Arg(0), *out, *node, *size); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func render(path, out, node string, size int) error {
	g, err := graph.Load(path)
	if err != nil {
		return err
	}

	if size == 0 {
		size = g.Size
	}
	if node == "" {
		node = g.Output
	}

	start := time.Now()
	ctx := graph.Ctx{W: size, H: size, Tiling: img.TileBoth, Scale: float64(size) / float64(g.Size)}

	b, err := graph.NewEvaluator(g, nodes.Registry).Eval(node, ctx)
	if err != nil {
		return err
	}

	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := png.Encode(f, b.Image()); err != nil {
		return err
	}

	fmt.Printf("%s: %dx%d in %v\n", out, b.W, b.H, time.Since(start).Round(time.Millisecond))
	return nil
}
