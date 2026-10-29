package noise

import "testing"

func TestPeriodic(t *testing.T) {
	const px, py = 4, 3

	for _, p := range [][2]float64{{0.3, 0.7}, {1.9, 2.2}, {3.5, 0.1}} {
		x, y := p[0], p[1]

		if a, b := FBM(x, y, px, py, 4, 0.5, 7), FBM(x+px, y+py, px, py, 4, 0.5, 7); abs(a-b) > 1e-9 {
			t.Errorf("FBM not periodic at %v: %v vs %v", p, a, b)
		}
		a, _ := Worley(x, y, px, py, 7)
		b, _ := Worley(x+px, y+py, px, py, 7)
		if abs(a-b) > 1e-9 {
			t.Errorf("Worley not periodic at %v: %v vs %v", p, a, b)
		}
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
