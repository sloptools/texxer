package palette

import "testing"

func TestMedianCut(t *testing.T) {
	var pix []Color
	for i := range 1000 {
		v := float32(i) / 1000
		pix = append(pix, Color{v, v * 0.5, 1 - v})
	}

	pal := MedianCut(pix, 16)
	if len(pal) != 16 {
		t.Fatalf("got %d colors, want 16", len(pal))
	}

	if c := Nearest(pal, Color{0, 0, 1}); c[0] > 0.1 {
		t.Errorf("nearest to blue end is %v", c)
	}
}
