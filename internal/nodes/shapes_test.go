package nodes

import (
	"math"
	"testing"
)

func TestTrapezoid(t *testing.T) {
	const top, bottom, hh = 10, 20, 10

	cases := []struct {
		x, y, want float64
	}{
		{0, 0, -10},     // center, closest to top/bottom edge
		{0, -15, 5},     // above the top edge
		{0, 15, 5},      // below the bottom edge
		{25, 10, 5},     // right of the bottom corner
		{15, -10, 4.47}, // outside the slanted side near the top
	}

	for _, c := range cases {
		if got := sdTrapezoid(c.x, c.y, top, bottom, hh); math.Abs(got-c.want) > 0.01 {
			t.Errorf("sdTrapezoid(%v, %v) = %.2f, want %.2f", c.x, c.y, got, c.want)
		}
	}
}
