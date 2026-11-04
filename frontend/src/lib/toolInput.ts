import * as THREE from 'three'
import type { Compositor } from './gl/compositor'
import type { Dab } from './gl/paint'
import type { Hit, Stage } from './gl/stage'
import { contains, corners, expand, fitted, handles, rotationTowards } from './stamps'
import { snipTiles, tiled } from './snippets.svelte'
import { store } from './store.svelte'
import { tools, type Picked } from './tools.svelte'

const mark = '#ffb020'
const dim = '#b9b6ad'

// Pointer handling for the brush and stamp tools on the 3D view.
const axes = [new THREE.Vector3(1, 0, 0), new THREE.Vector3(0, 1, 0), new THREE.Vector3(0, 0, 1)]
// As on the view gizmo; -1 is a free line.
const axisColors: Record<number, string> = { [-1]: '#ffffff', 0: '#c77a6e', 1: '#95b36f', 2: '#6f95c4' }

// A straight line from a towards b: along the world axis it runs most along, or free.
function lineTo(a: THREE.Vector3, b: THREE.Vector3, free: boolean): { end: THREE.Vector3; axis: number } {
  if (free) return { end: b.clone(), axis: -1 }
  const d = b.clone().sub(a)
  const axis = [0, 1, 2].reduce((best, k) => (Math.abs(d.getComponent(k)) > Math.abs(d.getComponent(best)) ? k : best), 0)
  return { end: a.clone().addScaledVector(axes[axis], d.getComponent(axis)), axis }
}

export class ToolInput {
  private stroke: { id: string; before: Uint8Array; last: THREE.Vector3 } | undefined
  private lastDab: THREE.Vector3 | undefined
  // A texture texel's size at the stroke, in model diagonals, for pixel brushes.
  private texel = 0
  // The last hover, to show the line again when ⇧ or ⌘ change without a move.
  private lastHit: Hit | undefined
  private dragging = false
  // A grip of the picked stamp being dragged.
  private grip: 'rotate' | number | undefined
  private cleanup: () => void

  constructor(
    private stage: Stage,
    private compositor: () => Compositor | undefined,
    private changed: () => void,
  ) {
    const el = stage.renderer.domElement
    const down = (e: PointerEvent) => this.down(e)
    const move = (e: PointerEvent) => this.move(e)
    const up = () => this.up()
    const leave = () => this.hover(undefined)
    const keys = (e: KeyboardEvent) => (e.key === 'Shift' || e.key === 'Meta') && this.hover(this.lastHit, e)
    el.addEventListener('pointerdown', down)
    el.addEventListener('pointermove', move)
    el.addEventListener('pointerup', up)
    el.addEventListener('pointerleave', leave)
    window.addEventListener('keydown', keys)
    window.addEventListener('keyup', keys)
    this.cleanup = () => {
      window.removeEventListener('keydown', keys)
      window.removeEventListener('keyup', keys)
      el.removeEventListener('pointerdown', down)
      el.removeEventListener('pointermove', move)
      el.removeEventListener('pointerup', up)
      el.removeEventListener('pointerleave', leave)
    }
  }

  private down(e: PointerEvent) {
    if (e.button !== 0 || tools.tool === 'nav') return
    if (tools.tool === 'volume') {
      this.pressVolume(e)
      return
    }
    if (tools.tool === 'stamp' && (this.grip = this.gripAt(e.clientX, e.clientY)) !== undefined) {
      this.stage.renderer.domElement.setPointerCapture(e.pointerId)
      return
    }
    const hit = this.stage.hit(e.clientX, e.clientY)
    if (!hit) return
    this.stage.renderer.domElement.setPointerCapture(e.pointerId)

    if (tools.tool === 'brush') this.startStroke(hit, e)
    else if (e.shiftKey && tools.pickedStamp) tools.row(hit.point, hit.normal)
    else if (e.metaKey) this.fitStamp(hit)
    else this.pressStamp(hit)
  }

  private move(e: PointerEvent) {
    if (tools.tool === 'nav') return
    if (this.grip !== undefined) {
      this.dragGrip(e)
      return
    }
    // Over a grip: no preview of a new stamp, a hand to say it can be dragged.
    const grip = tools.tool === 'stamp' && !this.dragging && !this.stroke ? this.gripAt(e.clientX, e.clientY) : undefined
    this.stage.renderer.domElement.style.cursor = grip !== undefined ? 'grab' : ''
    if (grip !== undefined) {
      if (tools.ghost) {
        tools.ghost = null
        this.changed()
      }
      return
    }
    const hit = this.stage.hit(e.clientX, e.clientY)

    if (this.stroke) {
      if (hit) this.strokeTo(hit, e.altKey)
    } else if (this.dragging && tools.tool === 'volume') {
      if (hit) tools.moveVolume([hit.point.x, hit.point.y, hit.point.z])
    } else if (this.dragging) {
      if (hit) tools.move(hit.point, hit.normal)
    }
    this.hover(hit, e)
  }

  private up() {
    if (this.stroke) this.endStroke()
    this.dragging = false
    this.grip = undefined
  }

  // Cursor feedback: the brush ring, or the frame of the stamp that would be placed or moved.
  hover(hit: Hit | undefined, keys?: { shiftKey: boolean; metaKey: boolean }) {
    this.lastHit = hit
    const diag = store.model?.diag ?? 1
    const brush = tools.tool === 'brush' && hit ? this.brushAt(hit) : undefined
    this.stage.setRing(brush && hit ? { point: brush.point, normal: hit.normal, radius: brush.radius } : null)

    // ⇧ shows the straight line a click would paint from the last stroke.
    if (brush && keys?.shiftKey && this.lastDab && !this.stroke) {
      const line = lineTo(this.lastDab, brush.point, keys.metaKey)
      this.stage.setLine([this.lastDab, line.end], axisColors[line.axis])
    } else this.stage.setLine(null)

    if (tools.tool !== 'stamp') {
      this.stage.setOutline(null)
      if (tools.ghost) {
        tools.ghost = null
        this.changed()
      }
      return
    }

    const over = hit && !this.dragging ? this.stampAt(hit.point) : undefined
    if (hit && !over && !this.dragging) {
      tools.ghostAt(hit.point, hit.normal, this.stage.cameraUp)
      this.stage.setOutline(corners(tools.ghost!), mark)
    } else {
      tools.ghost = null
      const s = over ? store.layer?.masks[over.mask].stamps[over.index] : tools.pickedStamp
      this.stage.setOutline(s ? corners(s) : null, over ? dim : mark)
    }
    this.changed()
  }

  // The selected layer's stamp under p, mirrored copies included.
  private stampAt(p: THREE.Vector3): Picked | undefined {
    const c = this.compositor()
    const masks = store.layer?.masks ?? []
    for (let mask = 0; mask < masks.length; mask++) {
      const list = masks[mask].stamps
      for (let index = list.length - 1; index >= 0; index--) {
        if (expand([list[index]], c?.mirrorX ?? 0).some((s) => contains(s, p))) return { mask, index }
      }
    }
  }

  // The rotate grip, or the index of a corner grip, under the cursor.
  private gripAt(x: number, y: number): 'rotate' | number | undefined {
    const s = tools.pickedStamp
    if (!s) return
    const g = handles(s)
    const near = (p: THREE.Vector3) => {
      const at = this.stage.toScreen(p)
      return Math.hypot(at.x - x, at.y - y) < 10
    }
    if (near(g.rotate)) return 'rotate'
    const corner = g.corners.findIndex(near)
    if (corner >= 0) return corner
  }

  // Turns the picked stamp or drags one of its corners, from where the cursor meets
  // its plane; ⇧ turns in 15° steps, or keeps the aspect.
  private dragGrip(e: PointerEvent) {
    const s = tools.pickedStamp
    if (!s) return
    const p = this.stage.onPlane(e.clientX, e.clientY, new THREE.Vector3(...s.p), new THREE.Vector3(...s.n))
    if (!p) return
    if (typeof this.grip === 'number') tools.dragCorner(this.grip, p, e.shiftKey)
    else {
      const deg = rotationTowards(s, p)
      tools.setRotation(e.shiftKey ? Math.round(deg / 15) * 15 : deg)
    }
    this.changed()
  }

  // A click beside the gizmo puts the volume there; dragging slides it over the surface.
  private pressVolume(e: PointerEvent) {
    if (this.stage.volume.active) return
    const hit = this.stage.hit(e.clientX, e.clientY)
    if (!hit) return
    this.stage.renderer.domElement.setPointerCapture(e.pointerId)
    tools.moveVolume([hit.point.x, hit.point.y, hit.point.z])
    this.dragging = true
  }

  private pressStamp(hit: Hit) {
    const over = this.stampAt(hit.point)
    if (over) tools.pick(over)
    else {
      tools.ghostAt(hit.point, hit.normal, this.stage.cameraUp)
      tools.placeGhost()
    }
    this.dragging = true
  }

  // ⌘click fills the surface under the cursor, bounded by hard edges, with the stamp.
  private fitStamp(hit: Hit) {
    const s = this.stage
    const model = store.model
    if (!s.posData || !s.nrmData || !model || hit.island < 0) return
    const pos = s.posData
    const nrm = s.nrmData
    const points: THREE.Vector3[] = []
    for (let i = 0; i < pos.length; i += 4) {
      if (nrm[i + 3] !== hit.island) continue
      if (nrm[i] * hit.normal.x + nrm[i + 1] * hit.normal.y + nrm[i + 2] * hit.normal.z < 0.95) continue
      points.push(new THREE.Vector3(pos[i], pos[i + 1], pos[i + 2]).multiplyScalar(model.diag).add(new THREE.Vector3(...model.min)))
    }
    if (points.length < 4) return
    const stamp = fitted(points, hit.normal, s.cameraUp, { shape: tools.stamp.shape, mirror: tools.mirror })
    if (stamp.shape.startsWith('snip:')) stamp.shape = tiled(stamp.shape, snipTiles(stamp.shape).cols, stamp.aspect)
    tools.placeStamp(stamp)
    this.changed()
  }

  // Where the brush paints and how wide: pixel brushes snap to the middle of the
  // texture texel under the cursor and measure whole texels.
  private brushAt(hit: Hit): { point: THREE.Vector3; radius: number } {
    const diag = store.model?.diag ?? 1
    const t = tools.brush.pixel ? this.texelAt(hit) : undefined
    if (!t) return { point: hit.point, radius: tools.brush.size * diag }
    return { point: t.center, radius: tools.brush.pixels * 0.5 * t.texel * diag }
  }

  private texelAt(hit: Hit): { center: THREE.Vector3; texel: number } | undefined {
    const s = this.stage
    const c = this.compositor()
    const model = store.model
    if (!hit.uv || !s.posData || !c || !model) return
    const pos = s.posData
    const size = s.posSize
    const f = c.factor
    const at = (x: number, y: number) => {
      const i = (Math.min(size - 1, Math.max(0, y)) * size + Math.min(size - 1, Math.max(0, x))) * 4
      return new THREE.Vector3(pos[i], pos[i + 1], pos[i + 2])
    }
    const x = Math.floor((hit.uv.x * size) / f) * f + (f >> 1)
    const y = Math.floor((hit.uv.y * size) / f) * f + (f >> 1)
    const p = at(x, y)
    if (p.lengthSq() === 0) return

    // The nearest neighbouring texel on the same surface gives the size.
    const texel = Math.min(...[at(x + f, y), at(x - f, y), at(x, y + f), at(x, y - f)].map((q) => q.distanceTo(p)).filter((d) => d > 0))
    if (!isFinite(texel)) return
    return { center: p.multiplyScalar(model.diag).add(new THREE.Vector3(...model.min)), texel }
  }

  private startStroke(hit: Hit, e: PointerEvent) {
    const paint = store.paint
    const mask = store.ensureMask('paint')
    if (!paint || mask < 0) return
    const id = store.layer!.masks[mask].paint
    paint.get(id)
    const l = store.layer!
    const color = tools.brush.color.toLowerCase()
    if (l.paintColors && !(tools.brush.erase !== e.altKey) && !l.brushColors?.includes(color)) l.brushColors = [...(l.brushColors ?? []), color].slice(-16)
    const t = tools.brush.pixel ? this.texelAt(hit) : undefined
    if (t) this.texel = t.texel
    hit = t ? { ...hit, point: t.center } : hit
    this.stroke = { id, before: paint.read(id), last: hit.point.clone() }

    // Shift-click draws a straight line from the last stroke, along an axis unless ⌘.
    if (e.shiftKey && this.lastDab) {
      this.stroke.last = this.lastDab.clone()
      const line = lineTo(this.lastDab, this.brushAt(hit).point, e.metaKey)
      this.stage.setLine(null)
      this.strokeTo({ ...hit, point: line.end }, e.altKey, line.axis >= 0 ? hit.normal : undefined)
    } else this.dab(hit.point, e.altKey)
  }

  // Dabs from the last one to the hit; a locked line lays each onto the surface, as
  // the axis can leave a curved one.
  private strokeTo(hit: Hit, alt: boolean, onto?: THREE.Vector3) {
    const s = this.stroke!
    const diag = store.model?.diag ?? 1
    const pixel = tools.brush.pixel && this.texel > 0
    if (pixel && !onto) hit = { ...hit, point: this.brushAt(hit).point }
    const spacing = Math.max(pixel ? this.texel * diag * 0.4 : tools.brush.size * diag * 0.25, 1e-6)
    const dist = s.last.distanceTo(hit.point)
    const steps = Math.floor(dist / spacing)
    for (let i = 1; i <= steps; i++) {
      const p = s.last.clone().lerp(hit.point, (i * spacing) / dist)
      this.dab(onto ? (tools.surface(p, onto, diag * 0.02)?.point ?? p) : p, alt)
    }
    if (steps > 0) s.last.lerp(hit.point, (steps * spacing) / dist)
  }

  private dab(p: THREE.Vector3, alt: boolean) {
    const paint = store.paint
    const c = this.compositor()
    const model = store.model
    if (!paint || !c || !model || !this.stroke) return

    const min = new THREE.Vector3(...model.min)
    const view = this.stage.cameraPosition.sub(p).normalize()
    const d: Dab = {
      center: p.clone().sub(min).divideScalar(model.diag),
      view,
      radius: tools.brush.size,
      hard: tools.brush.hard,
      strength: tools.brush.strength,
      grit: tools.brush.grit,
      erase: tools.brush.erase !== alt,
      map: tools.brush.map ? c.grunge(tools.brush.map) : undefined,
      mapScale: tools.brush.mapScale,
      pixel: tools.brush.pixel && this.texel > 0 ? { count: tools.brush.pixels, texel: this.texel, block: c.factor } : undefined,
      color: store.layer?.paintColors ? new THREE.Color(tools.brush.color).convertLinearToSRGB() : undefined,
    }
    paint.dab(this.stroke.id, d)

    if (tools.mirror) {
      const m = p.clone()
      m.x = 2 * c.mirrorX - m.x
      paint.dab(this.stroke.id, { ...d, center: m.sub(min).divideScalar(model.diag), view: view.clone().setX(-view.x) })
    }
    this.lastDab = p.clone()
    this.changed()
  }

  private endStroke() {
    const s = this.stroke!
    this.stroke = undefined
    const paint = store.paint
    if (!paint) return
    store.pushPaint(s.id, s.before, paint.read(s.id))
    paint.changed(s.id)
  }

  dispose() {
    this.cleanup()
  }
}
