import * as THREE from 'three'
import type { GizmoMode } from './gl/volume'
import { shapeAspect } from './shapes'
import { snipTiles, tiled } from './snippets.svelte'
import { cornerTo, moveTo, nudge, place } from './stamps'
import { store } from './store.svelte'
import type { Mask, Stamp, Tool, Vec3 } from './types'

export type Picked = { mask: number; index: number }

class Tools {
  tool = $state<Tool>('nav')
  // Pixel brushes paint whole texels of the texture, pixels across.
  brush = $state({ size: 0.03, hard: 0.7, strength: 1, grit: 0, erase: false, map: '', mapScale: 3, pixel: false, pixels: 1, color: '#c8c2b4' })
  // Settings for the next stamp; a picked stamp shares them while it's picked.
  stamp = $state({ shape: 'circle', size: 0.12, rot: 0, aspect: 1 })
  // Spacing of stamp rows, in stamp widths.
  gap = $state(1)
  text = $state('07')
  mirror = $state(false)
  snap = $state(false)
  picked = $state<Picked | null>(null)
  // The selected layer's volume mask the volume tool moves.
  volumeMask = $state(0)
  gizmo = $state<GizmoMode>('translate')
  ghost = $state<Stamp | null>(null)

  // Set by the viewport: the surface point nearest along -n from p, for keeping stamps on curved hulls.
  surface: (p: THREE.Vector3, n: THREE.Vector3, reach: number) => { point: THREE.Vector3; normal: THREE.Vector3 } | undefined = () => undefined

  set(tool: Tool) {
    this.tool = this.tool === tool ? 'nav' : tool
    this.ghost = null
    if (this.tool !== 'stamp') this.picked = null
    if (this.tool === 'volume' && !this.volume) this.editVolume(store.ensureMask('volume'))
  }

  get volume(): Mask | undefined {
    const masks = store.layer?.masks ?? []
    const m = masks[this.volumeMask]
    return m?.kind === 'volume' ? m : masks.find((m) => m.kind === 'volume')
  }

  editVolume(mask: number) {
    if (mask < 0) return
    this.volumeMask = mask
    this.tool = 'volume'
  }

  moveVolume(p: Vec3) {
    const m = this.volume
    if (m) m.volume.p = p
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

  // A stamp kept to an island goes to the island it is moved onto.
  move(point: THREE.Vector3, normal: THREE.Vector3, island = -1) {
    this.update((s) => {
      const moved = moveTo(s, point, normal, this.snap)
      if (s.island !== undefined) moved.island = island >= 0 ? island : undefined
      return moved
    })
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

  // Corner i of the picked stamp to p, the opposite one staying; tiled snippets keep
  // their tiles square.
  dragCorner(i: number, p: THREE.Vector3, keepAspect: boolean) {
    this.update((s) => {
      const next = cornerTo(s, i, p, keepAspect)
      if (next.shape.startsWith('snip:')) next.shape = tiled(next.shape, snipTiles(next.shape).cols, next.aspect)
      return next
    })
    const s = this.pickedStamp
    if (!s) return
    this.stamp = { ...this.stamp, shape: s.shape, size: s.size / this.diag, aspect: s.aspect }
  }

  // A world size for the picked stamp.
  setSize(size: number) {
    this.stamp.size = Math.min(1, Math.max(0.005, size / this.diag))
    this.update((s) => ({ ...s, size: this.stamp.size * this.diag }))
  }

  resize(factor: number) {
    const v = this.tool === 'volume' ? this.volume : undefined
    if (v) {
      v.volume.size = v.volume.size.map((s) => s * factor) as Vec3
      return
    }
    if (this.tool === 'brush' && this.brush.pixel) {
      this.brush.pixels = Math.min(16, Math.max(1, this.brush.pixels + (factor > 1 ? 1 : -1)))
      return
    }
    if (this.tool === 'brush') {
      this.brush.size = Math.min(0.3, Math.max(0.002, this.brush.size * factor))
      return
    }
    this.stamp.size = Math.min(1, Math.max(0.005, this.stamp.size * factor))
    this.update((s) => ({ ...s, size: this.stamp.size * this.diag }))
    if (this.ghost) this.ghost = { ...this.ghost, size: this.stamp.size * this.diag }
  }

  // Text and images bring their own aspect, so they don't come out squashed. A new
  // shape is for the next stamp, unless it swaps the picked one's.
  setShape(shape: string, aspect = shapeAspect(shape), swap = false) {
    if (!swap) this.picked = null
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
      const copy = { ...$state.snapshot(s), island: undefined }
      list.push(hit ? moveTo(copy, hit.point, hit.normal, false) : { ...copy, p: [at.x, at.y, at.z] })
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
    list.push(this.onSurface(nudge({ ...$state.snapshot(s), island: undefined }, 1, 0)))
    this.picked = { mask: p.mask, index: list.length - 1 }
  }

  remove() {
    const p = this.picked
    if (!p || !this.pickedStamp) return
    store.layer!.masks[p.mask].stamps.splice(p.index, 1)
    this.picked = null
  }

  // A finished stamp, like one fitted to a surface, picked to adjust.
  placeStamp(s: Stamp) {
    const mask = store.ensureMask('stamps')
    if (mask < 0) return
    const list = store.layer!.masks[mask].stamps
    list.push(s)
    this.pick({ mask, index: list.length - 1 })
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
