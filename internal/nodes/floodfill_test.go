package nodes

import (
	"testing"

	"texxer/internal/img"
)

func TestIslandsWrap(t *testing.T) {
	b := img.New(8, 4, 1, img.TileBoth)
	set := func(x, y int) { b.Pix[y*8+x] = 1 }

	// One island crossing the left/right edge, one in the middle.
	set(0, 1)
	set(7, 1)
	set(4, 2)

	if n := len(findIslands(b, 0.5)); n != 2 {
		t.Errorf("tiled: got %d islands, want 2", n)
	}

	b.Tiling = img.TileNone
	if n := len(findIslands(b, 0.5)); n != 3 {
		t.Errorf("untiled: got %d islands, want 3", n)
	}
}
