// Package grunge holds the built-in dirt and grunge maps: tileable 512² grays, white = dirt.
package grunge

import (
	"embed"
	"io/fs"
	"path"
	"strings"
)

//go:embed maps/*.png
var maps embed.FS

// Names lists the maps, sorted.
func Names() []string {
	entries, _ := fs.ReadDir(maps, "maps")
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, strings.TrimSuffix(e.Name(), ".png"))
	}
	return names
}

func PNG(name string) ([]byte, error) {
	return maps.ReadFile(path.Join("maps", path.Base(name)+".png"))
}
