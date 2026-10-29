package img

import (
	"image"
	"image/color"
	"math"
)

type Tiling int

const (
	TileBoth Tiling = iota
	TileX
	TileNone
)

// Buffer holds float pixels with 1 (gray) or 4 (RGBA) channels.
type Buffer struct {
	W, H   int
	C      int
	Tiling Tiling
	Pix    []float32
}

func New(w, h, c int, t Tiling) *Buffer {
	return &Buffer{W: w, H: h, C: c, Tiling: t, Pix: make([]float32, w*h*c)}
}

// At returns channel ch at (x, y), wrapping or clamping by tiling mode.
func (b *Buffer) At(x, y, ch int) float32 {
	if b.Tiling == TileNone {
		x = clampInt(x, 0, b.W-1)
	} else {
		x = Mod(x, b.W)
	}

	if b.Tiling == TileBoth {
		y = Mod(y, b.H)
	} else {
		y = clampInt(y, 0, b.H-1)
	}

	return b.Pix[(y*b.W+x)*b.C+ch]
}

// Sample reads channel ch bilinearly at a fractional pixel position.
func (b *Buffer) Sample(x, y float64, ch int) float32 {
	x, y = x-0.5, y-0.5
	x0, y0 := math.Floor(x), math.Floor(y)
	fx, fy := float32(x-x0), float32(y-y0)
	ix, iy := int(x0), int(y0)

	top := lerp(b.At(ix, iy, ch), b.At(ix+1, iy, ch), fx)
	bottom := lerp(b.At(ix, iy+1, ch), b.At(ix+1, iy+1, ch), fx)
	return lerp(top, bottom, fy)
}

// Value returns channel ch of pixel i. Gray buffers broadcast to RGB with alpha 1.
func (b *Buffer) Value(i, ch int) float32 {
	if b.C == 4 {
		return b.Pix[i*4+ch]
	}
	if ch == 3 {
		return 1
	}
	return b.Pix[i]
}

func (b *Buffer) Image() image.Image {
	r := image.Rect(0, 0, b.W, b.H)

	if b.C == 1 {
		out := image.NewGray(r)
		for i, v := range b.Pix {
			out.Pix[i] = toByte(v)
		}
		return out
	}

	out := image.NewNRGBA(r)
	for i, v := range b.Pix {
		out.Pix[i] = toByte(v)
	}
	return out
}

func FromImage(src image.Image) *Buffer {
	r := src.Bounds()
	b := New(r.Dx(), r.Dy(), 4, TileBoth)

	for y := range b.H {
		for x := range b.W {
			c := color.NRGBAModel.Convert(src.At(r.Min.X+x, r.Min.Y+y)).(color.NRGBA)
			i := (y*b.W + x) * 4
			b.Pix[i] = float32(c.R) / 255
			b.Pix[i+1] = float32(c.G) / 255
			b.Pix[i+2] = float32(c.B) / 255
			b.Pix[i+3] = float32(c.A) / 255
		}
	}
	return b
}
