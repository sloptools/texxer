package mesh

import (
	"math"
	"slices"
)

type bvhNode struct {
	min, max    Vec
	left, right int
	// Leaves hold tris[start:start+count].
	start, count int
}

// BVH answers whether a ray hits any triangle, for ambient occlusion.
type BVH struct {
	nodes []bvhNode
	tris  []Tri
}

const leafSize = 4

func NewBVH(tris []Tri) *BVH {
	b := &BVH{tris: slices.Clone(tris)}
	b.build(0, len(b.tris))
	return b
}

func (b *BVH) build(start, end int) int {
	lo, hi := Vec{math.Inf(1), math.Inf(1), math.Inf(1)}, Vec{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, t := range b.tris[start:end] {
		for _, p := range t.P {
			lo, hi = lo.Min(p), hi.Max(p)
		}
	}

	index := len(b.nodes)
	b.nodes = append(b.nodes, bvhNode{min: lo, max: hi})
	if end-start <= leafSize {
		b.nodes[index].start, b.nodes[index].count = start, end-start
		return index
	}

	// Split at the median centroid along the longest axis.
	ext := hi.Sub(lo)
	axis := 0
	if ext[1] > ext[axis] {
		axis = 1
	}
	if ext[2] > ext[axis] {
		axis = 2
	}
	slices.SortFunc(b.tris[start:end], func(a, c Tri) int {
		ca := a.P[0][axis] + a.P[1][axis] + a.P[2][axis]
		cc := c.P[0][axis] + c.P[1][axis] + c.P[2][axis]
		switch {
		case ca < cc:
			return -1
		case ca > cc:
			return 1
		}
		return 0
	})

	mid := (start + end) / 2
	left := b.build(start, mid)
	right := b.build(mid, end)
	b.nodes[index].left, b.nodes[index].right = left, right
	return index
}

// Occluded reports whether the ray from o along unit dir d hits a triangle closer than maxDist.
func (b *BVH) Occluded(o, d Vec, maxDist float64) bool {
	inv := Vec{1 / d[0], 1 / d[1], 1 / d[2]}
	stack := [64]int{0}
	n := 1

	for n > 0 {
		n--
		node := &b.nodes[stack[n]]
		if !hitBox(o, inv, node.min, node.max, maxDist) {
			continue
		}

		if node.count > 0 {
			for _, t := range b.tris[node.start : node.start+node.count] {
				if hitTri(o, d, t.P, maxDist) {
					return true
				}
			}
			continue
		}
		if n+2 > len(stack) {
			continue
		}
		stack[n], stack[n+1] = node.left, node.right
		n += 2
	}
	return false
}

func hitBox(o, inv, lo, hi Vec, maxDist float64) bool {
	tmin, tmax := 0.0, maxDist
	for a := range 3 {
		t1 := (lo[a] - o[a]) * inv[a]
		t2 := (hi[a] - o[a]) * inv[a]
		if t1 > t2 {
			t1, t2 = t2, t1
		}
		tmin, tmax = max(tmin, t1), min(tmax, t2)
		if tmin > tmax {
			return false
		}
	}
	return true
}

// hitTri is the Möller-Trumbore test, two-sided.
func hitTri(o, d Vec, p [3]Vec, maxDist float64) bool {
	e1, e2 := p[1].Sub(p[0]), p[2].Sub(p[0])
	h := d.Cross(e2)
	det := e1.Dot(h)
	if math.Abs(det) < 1e-12 {
		return false
	}

	f := 1 / det
	s := o.Sub(p[0])
	u := f * s.Dot(h)
	if u < 0 || u > 1 {
		return false
	}
	q := s.Cross(e1)
	v := f * d.Dot(q)
	if v < 0 || u+v > 1 {
		return false
	}

	t := f * e2.Dot(q)
	return t > 1e-6 && t < maxDist
}
