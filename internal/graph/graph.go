package graph

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Node struct {
	Type   string            `json:"type"`
	Params Params            `json:"params"`
	Inputs map[string]string `json:"inputs"`
}

type Graph struct {
	// Size is the design resolution; pixel params are relative to it.
	Size   int             `json:"size"`
	Output string          `json:"output"`
	Nodes  map[string]Node `json:"nodes"`

	Dir string `json:"-"`
}

func Load(path string) (*Graph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var g Graph
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	if g.Size == 0 {
		g.Size = 512
	}
	g.Dir = filepath.Dir(path)

	return &g, nil
}
