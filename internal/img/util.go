package img

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// Rows runs fn for every row in parallel.
func Rows(h int, fn func(y int)) {
	var wg sync.WaitGroup
	var next atomic.Int64

	for range runtime.GOMAXPROCS(0) {
		wg.Go(func() {
			for {
				y := int(next.Add(1) - 1)
				if y >= h {
					return
				}
				fn(y)
			}
		})
	}
	wg.Wait()
}

func Mod(a, n int) int {
	a %= n
	if a < 0 {
		a += n
	}
	return a
}

func Clamp01(v float32) float32 {
	return min(max(v, 0), 1)
}

func lerp(a, b, t float32) float32 {
	return a + (b-a)*t
}

func clampInt(v, lo, hi int) int {
	return min(max(v, lo), hi)
}

func toByte(v float32) uint8 {
	return uint8(Clamp01(v)*255 + 0.5)
}
