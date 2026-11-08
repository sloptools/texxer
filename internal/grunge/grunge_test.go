package grunge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuiltins(t *testing.T) {
	for name, build := range builtins {
		g := Equalize(build(64))
		var mean float64
		for _, v := range g.V {
			mean += v
		}
		mean /= float64(len(g.V))
		if mean < 0.45 || mean > 0.55 {
			t.Errorf("%s: mean %.2f after equalizing", name, mean)
		}
		// Tiles: the left and right columns are as close as neighbors inside.
		var seam, inner float64
		for y := range g.H {
			seam += abs(g.At(0, y) - g.At(-1, y))
			inner += abs(g.At(32, y) - g.At(31, y))
		}
		if seam > inner*2+1 {
			t.Errorf("%s: seam %.1f against %.1f inside", name, seam, inner)
		}
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func TestMapsFromFolder(t *testing.T) {
	dir := t.TempDir()
	png, _ := Encode(newGray(4, 4))
	for _, f := range []string{"specks_1.png", "dirt.PNG", "notes.txt", ".hidden.png"} {
		os.WriteFile(filepath.Join(dir, f), png, 0o644)
	}
	names := Names(dir)
	if n := len(names); n != len(builtins)+2 || names[n-2] != "dirt" || names[n-1] != "specks_1" {
		t.Fatalf("names %v", names)
	}
	if _, err := PNG(dir, "dirt"); err != nil {
		t.Fatal(err)
	}
	if _, err := PNG(dir, "clouds"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"../grunge", "notes", "", ".hidden", "nope"} {
		if _, err := PNG(dir, bad); err == nil {
			t.Fatalf("%q should not be a map", bad)
		}
	}
}
