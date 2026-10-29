package nodes

import "math"

// Signed distance functions in pixels, centered on the cell.
// hw, hh are half extents, r is the corner radius.
type sdf func(x, y, hw, hh, r float64) float64

var shapes = map[string]sdf{
	"box":     sdBox,
	"diamond": sdDiamond,
	"circle":  sdCircle,
	"groove":  sdGroove,
	"arch":    sdArch,
	"gothic":  sdGothic,
	"lines":   sdLines,
	"chamfer": sdChamfer,
}

// sdLines is a horizontal bar spanning the full cell width.
func sdLines(_, y, _, hh, _ float64) float64 {
	return math.Abs(y) - hh
}

// sdChamfer is a box with 45° cut corners of size c.
func sdChamfer(x, y, hw, hh, c float64) float64 {
	return max(sdBox(x, y, hw, hh, 0), chamferCut(x, y, hw, hh, c))
}

func chamferCut(x, y, hw, hh, c float64) float64 {
	c = min(c, hw, hh)
	return (math.Abs(x) + math.Abs(y) - (hw + hh - c)) / math.Sqrt2
}

// sdTrapezoid has half widths top and bottom (image y points down), after Inigo Quilez.
func sdTrapezoid(x, y, top, bottom, hh float64) float64 {
	x = math.Abs(x)

	r := bottom
	if y < 0 {
		r = top
	}
	ax, ay := x-min(x, r), math.Abs(y)-hh

	// Closest point on the slanted side, in y-up coordinates.
	py := -y
	kx, ky := top-bottom, 2*hh
	t := clamp01(((top-x)*kx + (hh-py)*ky) / (kx*kx + ky*ky))
	bx, by := x-top+kx*t, py-hh+ky*t

	d := math.Sqrt(min(ax*ax+ay*ay, bx*bx+by*by))
	if bx < 0 && ay < 0 {
		return -d
	}
	return d
}

func sdBox(x, y, hw, hh, r float64) float64 {
	r = min(r, hw, hh)
	dx := math.Abs(x) - (hw - r)
	dy := math.Abs(y) - (hh - r)

	outside := math.Hypot(max(dx, 0), max(dy, 0))
	inside := min(max(dx, dy), 0)
	return outside + inside - r
}

func sdDiamond(x, y, hw, hh, _ float64) float64 {
	return (math.Abs(x)*hh + math.Abs(y)*hw - hw*hh) / math.Hypot(hw, hh)
}

func sdCircle(x, y, hw, hh, _ float64) float64 {
	return math.Hypot(x, y) - min(hw, hh)
}

// sdGroove is a vertical bar spanning the full cell height.
func sdGroove(x, _, hw, _, _ float64) float64 {
	return math.Abs(x) - hw
}

// sdArch is a rectangle with a round top.
func sdArch(x, y, hw, hh, _ float64) float64 {
	hw = min(hw, hh)
	cy := -hh + hw

	body := sdBox(x, y-hw/2, hw, hh-hw/2, 0)
	top := math.Hypot(x, y-cy) - hw
	return min(body, top)
}

// sdGothic is a rectangle with a pointed top made from two intersecting circles.
func sdGothic(x, y, hw, hh, _ float64) float64 {
	hw = min(hw, hh/math.Sqrt(3))
	cy := -hh + math.Sqrt(3)*hw

	body := sdBox(x, y-(cy+hh)/2, hw, (hh-cy)/2, 0)
	left := math.Hypot(x+hw, y-cy) - 2*hw
	right := math.Hypot(x-hw, y-cy) - 2*hw
	return min(body, max(left, right))
}
