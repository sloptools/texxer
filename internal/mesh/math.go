package mesh

import "math"

type Vec [3]float64

func (a Vec) Add(b Vec) Vec       { return Vec{a[0] + b[0], a[1] + b[1], a[2] + b[2]} }
func (a Vec) Sub(b Vec) Vec       { return Vec{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }
func (a Vec) Scale(s float64) Vec { return Vec{a[0] * s, a[1] * s, a[2] * s} }
func (a Vec) Dot(b Vec) float64   { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
func (a Vec) Len() float64        { return math.Sqrt(a.Dot(a)) }
func (a Vec) Min(b Vec) Vec       { return Vec{min(a[0], b[0]), min(a[1], b[1]), min(a[2], b[2])} }
func (a Vec) Max(b Vec) Vec       { return Vec{max(a[0], b[0]), max(a[1], b[1]), max(a[2], b[2])} }

func (a Vec) Cross(b Vec) Vec {
	return Vec{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}

func (a Vec) Norm() Vec {
	l := a.Len()
	if l == 0 {
		return Vec{0, 1, 0}
	}
	return a.Scale(1 / l)
}

// mat4 is column-major like glTF.
type mat4 [16]float64

func identity() mat4 {
	return mat4{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}

func (a mat4) mul(b mat4) mat4 {
	var out mat4
	for c := range 4 {
		for r := range 4 {
			var s float64
			for k := range 4 {
				s += a[k*4+r] * b[c*4+k]
			}
			out[c*4+r] = s
		}
	}
	return out
}

func (a mat4) point(p Vec) Vec {
	return Vec{
		a[0]*p[0] + a[4]*p[1] + a[8]*p[2] + a[12],
		a[1]*p[0] + a[5]*p[1] + a[9]*p[2] + a[13],
		a[2]*p[0] + a[6]*p[1] + a[10]*p[2] + a[14],
	}
}

// normalMatrix is the inverse transpose of the upper 3x3, row-major.
func (a mat4) normalMatrix() [9]float64 {
	m := [9]float64{a[0], a[4], a[8], a[1], a[5], a[9], a[2], a[6], a[10]}
	det := m[0]*(m[4]*m[8]-m[5]*m[7]) - m[1]*(m[3]*m[8]-m[5]*m[6]) + m[2]*(m[3]*m[7]-m[4]*m[6])
	if det == 0 {
		return [9]float64{1, 0, 0, 0, 1, 0, 0, 0, 1}
	}

	inv := [9]float64{
		m[4]*m[8] - m[5]*m[7], m[2]*m[7] - m[1]*m[8], m[1]*m[5] - m[2]*m[4],
		m[5]*m[6] - m[3]*m[8], m[0]*m[8] - m[2]*m[6], m[2]*m[3] - m[0]*m[5],
		m[3]*m[7] - m[4]*m[6], m[1]*m[6] - m[0]*m[7], m[0]*m[4] - m[1]*m[3],
	}
	for i := range inv {
		inv[i] /= det
	}
	// Transpose.
	return [9]float64{inv[0], inv[3], inv[6], inv[1], inv[4], inv[7], inv[2], inv[5], inv[8]}
}

func transformNormal(m [9]float64, n Vec) Vec {
	return Vec{
		m[0]*n[0] + m[1]*n[1] + m[2]*n[2],
		m[3]*n[0] + m[4]*n[1] + m[5]*n[2],
		m[6]*n[0] + m[7]*n[1] + m[8]*n[2],
	}.Norm()
}

func localMatrix(matrix, t, r, s []float64) mat4 {
	if len(matrix) == 16 {
		var m mat4
		copy(m[:], matrix)
		return m
	}

	tx, ty, tz := 0.0, 0.0, 0.0
	if len(t) == 3 {
		tx, ty, tz = t[0], t[1], t[2]
	}
	qx, qy, qz, qw := 0.0, 0.0, 0.0, 1.0
	if len(r) == 4 {
		qx, qy, qz, qw = r[0], r[1], r[2], r[3]
	}
	sx, sy, sz := 1.0, 1.0, 1.0
	if len(s) == 3 {
		sx, sy, sz = s[0], s[1], s[2]
	}

	return mat4{
		(1 - 2*(qy*qy+qz*qz)) * sx, 2 * (qx*qy + qz*qw) * sx, 2 * (qx*qz - qy*qw) * sx, 0,
		2 * (qx*qy - qz*qw) * sy, (1 - 2*(qx*qx+qz*qz)) * sy, 2 * (qy*qz + qx*qw) * sy, 0,
		2 * (qx*qz + qy*qw) * sz, 2 * (qy*qz - qx*qw) * sz, (1 - 2*(qx*qx+qy*qy)) * sz, 0,
		tx, ty, tz, 1,
	}
}
