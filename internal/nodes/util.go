package nodes

import (
	"math"

	"texxer/internal/graph"
	"texxer/internal/img"
)

// fill creates a gray buffer for ctx with fn evaluated per pixel.
func fill(ctx graph.Ctx, fn func(x, y int) float32) *img.Buffer {
	b := img.New(ctx.W, ctx.H, 1, ctx.Tiling)

	img.Rows(ctx.H, func(y int) {
		for x := range ctx.W {
			b.Pix[y*ctx.W+x] = fn(x, y)
		}
	})
	return b
}

// periods turns "cells across the width" into integer noise periods that fit ctx.
func periods(ctx graph.Ctx, scale float64) (px, py int) {
	px = max(1, int(math.Round(scale)))
	py = max(1, int(math.Round(scale*float64(ctx.H)/float64(ctx.W))))
	return px, py
}

// cell maps a pixel to noise cell coordinates.
func cell(ctx graph.Ctx, x, y, px, py int) (float64, float64) {
	u := (float64(x) + 0.5) / float64(ctx.W) * float64(px)
	v := (float64(y) + 0.5) / float64(ctx.H) * float64(py)
	return u, v
}

// wrapIndex maps a pixel to its buffer index by tiling mode. ok is false outside non-tiled edges.
func wrapIndex(b *img.Buffer, x, y int) (int, bool) {
	if b.Tiling != img.TileNone {
		x = img.Mod(x, b.W)
	}
	if b.Tiling == img.TileBoth {
		y = img.Mod(y, b.H)
	}

	if x < 0 || x >= b.W || y < 0 || y >= b.H {
		return 0, false
	}
	return y*b.W + x, true
}

func smoothstep(lo, hi, v float64) float64 {
	t := min(max((v-lo)/(hi-lo), 0), 1)
	return t * t * (3 - 2*t)
}

func clamp01(v float64) float64 {
	return min(max(v, 0), 1)
}
