import * as THREE from 'three'
import { shapeAspect } from './shapes'
import { moveTo, nudge, place } from './stamps'
import { store } from './store.svelte'
import type { Stamp, Tool } from './types'

export type Picked = { mask: number; index: number }

class Tools {
  tool = $state<Tool>('nav')
  brush = $state({ size: 0.03, hard: 0.7, strength: 1, grit: 0, erase: false })
  // Settings for the next stamp; a picked stamp shares them while it's picked.
  stamp = $state({ shape: 'circle', size: 0.12, rot: 0, aspect: 1 })
  // Spacing of stamp rows, in stamp widths.
  gap = $state(1)
  text = $state('07')
  mirror = $state(false)
  snap = $state(false)
  picked = $state<Picked | null>(null)
  ghost = $state<Stamp | null>(null)

  // Set by the viewport: the surface point nearest along -n from p, for keeping stamps on curved hulls.
  surface: (p: THREE.Vector3, n: THREE.Vector3, reach: number) => { point: THREE.Vector3; normal: THREE.Vector3 } | undefined = () => undefined

  set(tool: Tool) {
    this.tool = this.tool === tool ? 'nav' : tool
    this.ghost = null
    if (this.tool !== 'stamp') this.picked = null
  }

  // The mask a new layer starts with, so it is ready for the current tool.
  get pickedStamp(): Stamp | undefined {
    const p = this.picked
    return p ? store.layer?.masks[p.mask]?.stamps[p.index] : undefined
  }

  // Stamp sizes are fractions of the model's diagonal, like the brush.
  private get diag() {
    return store.model?.diag ?? 1
  }

  ghostAt(point: THREE.Vector3, normal: THREE.Vector3, hint: THREE.Vector3) {
    const s = this.stamp
    this.ghost = place(point, normal, hint, { shape: s.shape, size: s.size * this.diag, rot: s.rot, aspect: s.aspect, mirror: this.mirror }, this.snap)
  }

  placeGhost() {
    const g = this.ghost
    if (!g) return
    const mask = store.ensureMask('stamps')
    if (mask < 0) return
    const list = store.layer!.masks[mask].stamps
    list.push(g)
    this.picked = { mask, index: list.length - 1 }
    this.ghost = null
  }

  private update(fn: (s: Stamp) => Stamp) {
    const p = this.picked
    const s = this.pickedStamp
    if (!p || !s) return
    store.layer!.masks[p.mask].stamps[p.index] = fn($state.snapshot(s))
  }

  move(point: THREE.Vector3, normal: THREE.Vector3) {
    this.update((s) => moveTo(s, point, normal, this.snap))
  }

  rotate(deg: number) {
    this.stamp.rot = (((this.stamp.rot + deg) % 360) + 360) % 360
    this.update((s) => ({ ...s, rot: this.stamp.rot }))
    if (this.ghost) this.ghost = { ...this.ghost, rot: this.stamp.rot }
  }

  setRotation(deg: number) {
    this.stamp.rot = ((Math.round(deg) % 360) + 360) % 360
    this.update((s) => ({ ...s, rot: this.stamp.rot }))
  }

  // A world size for the picked stamp.
  setSize(size: number) {
    this.stamp.size = Math.min(1, Math.max(0.005, size / this.diag))
    this.update((s) => ({ ...s, size: this.stamp.size * this.diag }))
  }

  resize(factor: number) {
    if (this.tool === 'brush') {
      this.brush.size = Math.min(0.3, Math.max(0.002, this.brush.size * factor))
      return
    }
    this.stamp.size = Math.min(1, Math.max(0.005, this.stamp.size * factor))
    this.update((s) => ({ ...s, size: this.stamp.size * this.diag }))
    if (this.ghost) this.ghost = { ...this.ghost, size: this.stamp.size * this.diag }
  }

  // Text and images bring their own aspect, so they don't come out squashed.
  setShape(shape: string, aspect = shapeAspect(shape)) {
    this.stamp.shape = shape
    if (aspect) this.stamp.aspect = Math.round(aspect * 100) / 100
    this.update((s) => ({ ...s, shape, aspect: this.stamp.aspect }))
  }

  // Copies of the picked stamp from where it is to the point, gap widths apart,
  // each laid onto the surface below it.
  row(point: THREE.Vector3, normal: THREE.Vector3) {
    const p = this.picked
    const s = this.pickedStamp
    if (!p || !s) return
    const from = new THREE.Vector3(...s.p)
    const n0 = new THREE.Vector3(...s.n)
    const step = s.size * this.gap
    const count = Math.floor(from.distanceTo(point) / step)
    const list = store.layer!.masks[p.mask].stamps

    for (let k = 1; k <= count; k++) {
      const t = (k * step) / from.distanceTo(point)
      const at = from.clone().lerp(point, t)
      const n = n0.clone().lerp(normal, t).normalize()
      const hit = this.surface(at, n, s.size)
      list.push(hit ? moveTo($state.snapshot(s), hit.point, hit.normal, false) : { ...$state.snapshot(s), p: [at.x, at.y, at.z] })
    }
    this.picked = { mask: p.mask, index: list.length - 1 }
  }

  setAspect(aspect: number) {
    this.stamp.aspect = aspect
    this.update((s) => ({ ...s, aspect }))
  }

  toggleMirror() {
    this.mirror = !this.mirror
    if (this.tool === 'stamp') this.update((s) => ({ ...s, mirror: this.mirror }))
  }

  nudge(dx: number, dy: number) {
    this.update((s) => this.onSurface(nudge(s, dx, dy)))
  }

  private onSurface(s: Stamp): Stamp {
    const hit = this.surface(new THREE.Vector3(...s.p), new THREE.Vector3(...s.n), s.size)
    return hit ? moveTo(s, hit.point, hit.normal, false) : s
  }

  // A copy one stamp width to the right, for rows.
  duplicate() {
    const p = this.picked
    const s = this.pickedStamp
    if (!p || !s) return
    const list = store.layer!.masks[p.mask].stamps
    list.push(this.onSurface(nudge($state.snapshot(s), 1, 0)))
    this.picked = { mask: p.mask, index: list.length - 1 }
  }

  remove() {
    const p = this.picked
    if (!p || !this.pickedStamp) return
    store.layer!.masks[p.mask].stamps.splice(p.index, 1)
    this.picked = null
  }

  // Takes over a stamp's settings, so the next one matches it.
  pick(p: Picked) {
    this.picked = p
    const s = this.pickedStamp
    if (!s) return
    this.stamp = { shape: s.shape, size: s.size / this.diag, rot: s.rot, aspect: s.aspect }
    this.mirror = s.mirror
  }
}

export const tools = new Tools()
