import * as THREE from 'three'
import type { Stamp, Vec3 } from './types'

const v = (a: Vec3) => new THREE.Vector3(...a)
const arr = (a: THREE.Vector3): Vec3 => [a.x, a.y, a.z]
const worldUp = new THREE.Vector3(0, 1, 0)

export type Frame = { c: THREE.Vector3; t: THREE.Vector3; b: THREE.Vector3; n: THREE.Vector3; hw: number; hh: number }

// Upright on walls: world up laid onto the surface. On floors and roofs, the
// world axis closest to the hint, so stamps still line up with the model.
export function uprightOn(n: THREE.Vector3, hint: THREE.Vector3): THREE.Vector3 {
  const up = worldUp.clone().addScaledVector(n, -n.dot(worldUp))
  if (up.length() > 0.35) return up.normalize()

  let best = new THREE.Vector3(0, 0, -1)
  let bestDot = -Infinity
  for (const a of [
    [1, 0, 0],
    [-1, 0, 0],
    [0, 0, 1],
    [0, 0, -1],
  ] as Vec3[]) {
    const axis = v(a)
    if (axis.dot(hint) > bestDot) {
      best = axis
      bestDot = axis.dot(hint)
    }
  }
  return best.addScaledVector(n, -n.dot(best)).normalize()
}

export function frame(s: Stamp): Frame {
  const n = v(s.n).normalize()
  const b0 = v(s.up).addScaledVector(n, -n.dot(v(s.up))).normalize()
  const t0 = b0.clone().cross(n)
  const r = THREE.MathUtils.degToRad(s.rot)
  const t = t0.multiplyScalar(Math.cos(r)).addScaledVector(b0, Math.sin(r))
  const b = n.clone().cross(t)
  return { c: v(s.p), t, b, n, hw: s.size / 2, hh: (s.size * s.aspect) / 2 }
}

// The copy across the plane x = cx, turned so text still reads.
export function mirrored(s: Stamp, cx: number): Stamp {
  return {
    ...s,
    p: [2 * cx - s.p[0], s.p[1], s.p[2]],
    n: [-s.n[0], s.n[1], s.n[2]],
    up: [-s.up[0], s.up[1], s.up[2]],
    rot: -s.rot,
    mirror: false,
  }
}

export function expand(stamps: Stamp[], cx: number): Stamp[] {
  return stamps.flatMap((s) => (s.mirror ? [s, mirrored(s, cx)] : [s]))
}

export function contains(s: Stamp, p: THREE.Vector3): boolean {
  const f = frame(s)
  const d = p.clone().sub(f.c)
  return Math.abs(d.dot(f.t)) <= f.hw && Math.abs(d.dot(f.b)) <= f.hh && Math.abs(d.dot(f.n)) <= f.hw * 0.5
}

export function corners(s: Stamp): THREE.Vector3[] {
  const f = frame(s)
  const lift = f.n.clone().multiplyScalar(f.hw * 0.01)
  return [
    [-1, -1],
    [1, -1],
    [1, 1],
    [-1, 1],
  ].map(([x, y]) => f.c.clone().addScaledVector(f.t, x * f.hw).addScaledVector(f.b, y * f.hh).add(lift))
}

// Snaps p to a grid on the surface plane, half a stamp wide, anchored at the model origin.
export function snapOnPlane(p: THREE.Vector3, n: THREE.Vector3, up: THREE.Vector3, step: number): THREE.Vector3 {
  const t = up.clone().cross(n)
  const a = p.dot(t)
  const b = p.dot(up)
  return p
    .clone()
    .addScaledVector(t, Math.round(a / step) * step - a)
    .addScaledVector(up, Math.round(b / step) * step - b)
}

export function place(point: THREE.Vector3, normal: THREE.Vector3, hint: THREE.Vector3, base: Omit<Stamp, 'p' | 'n' | 'up'>, snap: boolean): Stamp {
  const up = uprightOn(normal, hint)
  const p = snap ? snapOnPlane(point, normal, up, base.size / 2) : point
  return { ...base, p: arr(p), n: arr(normal), up: arr(up) }
}

// Moves a stamp onto a new spot, keeping its up where the surface allows.
export function moveTo(s: Stamp, point: THREE.Vector3, normal: THREE.Vector3, snap: boolean): Stamp {
  const keep = v(s.up).addScaledVector(normal, -normal.dot(v(s.up)))
  const up = keep.length() > 0.35 ? keep.normalize() : uprightOn(normal, v(s.up))
  const p = snap ? snapOnPlane(point, normal, up, s.size / 2) : point
  return { ...s, p: arr(p), n: arr(normal), up: arr(up) }
}

// Shifts a stamp along its own axes, in stamp widths.
export function nudge(s: Stamp, dx: number, dy: number): Stamp {
  const f = frame(s)
  const p = f.c.addScaledVector(f.t, dx * s.size).addScaledVector(f.b, dy * s.size)
  return { ...s, p: arr(p) }
}

// The grips of a picked stamp: one above its top edge turns it, one on its
// top right corner sizes it.
export function handles(s: Stamp): { rotate: THREE.Vector3; scale: THREE.Vector3 } {
  const f = frame(s)
  return {
    rotate: f.c.clone().addScaledVector(f.b, f.hh + Math.max(f.hw, f.hh) * 0.4),
    scale: f.c.clone().addScaledVector(f.t, f.hw).addScaledVector(f.b, f.hh),
  }
}

// The rotation that points a stamp's top at p, in degrees.
export function rotationTowards(s: Stamp, p: THREE.Vector3): number {
  const f = frame({ ...s, rot: 0 })
  const v = p.clone().sub(f.c)
  return THREE.MathUtils.radToDeg(Math.atan2(-v.dot(f.t), v.dot(f.b)))
}

// The size that puts a stamp's corner at p.
export function sizeTowards(s: Stamp, p: THREE.Vector3): number {
  const f = frame(s)
  const v = p.clone().sub(f.c)
  const along = new THREE.Vector3(v.dot(f.t), v.dot(f.b), 0).length()
  return (2 * along) / Math.hypot(1, s.aspect)
}
