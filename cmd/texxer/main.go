package main

import (
	"flag"
	"fmt"
	"image/png"
	"log"
	"net/http"
	"os"
	"runtime/pprof"
	"time"

	"texxer/internal/graph"
	"texxer/internal/img"
	"texxer/internal/nodes"
	"texxer/internal/presets"
	"texxer/internal/server"
	"texxer/web"
)

const usage = `usage:
  texxer render [flags] graph.json
  texxer serve [flags]`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "render":
		err = renderCmd(os.Args[2:])
	case "serve":
		err = serveCmd(os.Args[2:])
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func serveCmd(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", "localhost:7777", "listen address")
	dir := fs.String("dir", ".", "directory with graph files")
	fs.Parse(args)

	log.Printf("texxer on http://%s, graphs in %s", *addr, *dir)
	return http.ListenAndServe(*addr, server.New(*dir, web.Dist()))
}

func renderCmd(args []string) error {
	fs := flag.NewFlagSet("render", flag.ExitOnError)
	out := fs.String("o", "out.png", "output file")
	size := fs.Int("size", 0, "render size in px (default: graph size)")
	node := fs.String("node", "", "render this node instead of the graph output")
	seed := fs.Int("seed", 0, "variant seed, shifts all seeds in the graph")
	profile := fs.String("cpuprofile", "", "write a CPU profile to this file")
	fs.Parse(args)

	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(2)
	}

	if *profile != "" {
		f, err := os.Create(*profile)
		if err != nil {
			return err
		}
		defer f.Close()

		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	return render(fs.Arg(0), *out, *node, *size, *seed)
}

func render(path, out, node string, size, seed int) error {
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

	ev := graph.NewEvaluator(g, nodes.Evals(), presets.Load)
	ev.SetSeed(seed)
	// A cache also dedupes equal work inside presets.
	ev.UseCache(graph.NewCache(2 << 30))

	b, err := ev.Eval(node, ctx)
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
