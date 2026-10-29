package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"texxer/internal/stats"
)

func statsCmd(args []string) error {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: texxer stats file.png|graph.json ...")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	if fs.NArg() == 0 {
		fs.Usage()
		os.Exit(2)
	}

	fmt.Printf("%-24s %6s %6s %6s %6s %6s %6s\n", "", "mean", "std", "p2", "p98", "detail", "colors")
	for _, path := range fs.Args() {
		im, err := loadImage(path)
		if err != nil {
			return err
		}

		s := stats.Measure(im)
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		fmt.Printf("%-24s %6.3f %6.3f %6.3f %6.3f %6.3f %6d\n", name, s.Mean, s.Std, s.P2, s.P98, s.Detail, s.Colors)
	}
	return nil
}

// loadImage reads a PNG, or renders a graph at its design size.
func loadImage(path string) (image.Image, error) {
	if strings.HasSuffix(path, ".json") {
		b, err := evalGraph(path, "", 0, 0)
		if err != nil {
			return nil, err
		}
		return b.Image(), nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	im, _, err := image.Decode(f)
	return im, err
}
