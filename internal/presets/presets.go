// Package presets holds the built-in subgraphs. Each file is usable as a node type named after it.
package presets

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"texxer/internal/graph"
)

//go:embed *.json
var files embed.FS

func Names() []string {
	entries, _ := files.ReadDir(".")

	var names []string
	for _, e := range entries {
		names = append(names, strings.TrimSuffix(e.Name(), ".json"))
	}
	return names
}

func Load(name string) (*graph.Graph, error) {
	data, err := files.ReadFile(name + ".json")
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	g, err := graph.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("preset %s: %w", name, err)
	}
	return g, nil
}
