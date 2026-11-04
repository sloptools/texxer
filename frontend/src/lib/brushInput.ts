import * as THREE from 'three'
import type { Compositor } from './gl/compositor'
import type { Dab } from './gl/paint'
import type { Hit, Stage } from './gl/stage'
import { store } from './store.svelte'
import { tools } from './tools.svelte'

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

export type Keys = { shiftKey: boolean; metaKey: boolean; altKey: boolean }

// Brush strokes on the 3D view: soft or pixel dabs, and straight lines.
export class BrushInput {
  // A fill is a stroke of one click, that moves don't continue.
  private stroke: { id: string; before: Uint8Array; last: THREE.Vector3; fill?: boolean } | undefined
  private lastDab: THREE.Vector3 | undefined
  // A texture texel's size at the stroke, in model diagonals, for pixel brushes.
  private texel = 0

  constructor(
    private stage: Stage,
    private compositor: () => Compositor | undefined,
    private changed: () => void,
  ) {}

  get active() {
    return !!this.stroke
  }

  // The ring where the brush would paint, and with ⇧ the line a click would draw.
  hover(hit: Hit | undefined, keys?: Keys) {
    const brush = tools.tool === 'brush' && hit ? this.at(hit) : undefined
    this.stage.setRing(brush && hit ? { point: brush.point, normal: hit.normal, radius: brush.radius } : null)

    if (brush && keys?.shiftKey && this.lastDab && !this.stroke) {
      const line = lineTo(this.lastDab, brush.point, keys.metaKey)
      this.stage.setLine([this.lastDab, line.end], axisColors[line.axis])
    } else this.stage.setLine(null)
  }

  start(hit: Hit, e: PointerEvent) {
    const paint = store.paint
    const mask = store.ensureMask('paint')
    if (!paint || mask < 0) return
    const l = store.layer!
    const id = l.masks[mask].paint
    paint.get(id)
    const color = tools.brush.color.toLowerCase()
    if (l.paintColors && !(tools.brush.erase !== e.altKey) && !l.brushColors?.includes(color)) l.brushColors = [...(l.brushColors ?? []), color].slice(-16)
    const t = tools.brush.pixel ? this.texelAt(hit) : undefined
    if (t) this.texel = t.texel
    hit = t ? { ...hit, point: t.center } : hit
    this.stroke = { id, before: paint.read(id), last: hit.point.clone() }

    // ⌘click fills the surface under the cursor, bounded by hard edges.
    if (e.metaKey && !e.shiftKey) {
      if (hit.island < 0) return
      this.stroke.fill = true
      paint.fill(id, hit.island, l.paintColors ? new THREE.Color(tools.brush.color).convertLinearToSRGB() : undefined, tools.brush.erase !== e.altKey)
      this.changed()
      return
    }

    // Shift-click draws a straight line from the last stroke, along an axis unless ⌘.
    if (e.shiftKey && this.lastDab) {
      this.stroke.last = this.lastDab.clone()
      const line = lineTo(this.lastDab, this.at(hit).point, e.metaKey)
      this.stage.setLine(null)
      this.to({ ...hit, point: line.end }, e.altKey, line.axis >= 0 ? hit.normal : undefined)
    } else this.dab(hit.point, e.altKey)
  }

  // Dabs from the last one to the hit; a locked line lays each onto the surface, as
  // the axis can leave a curved one.
  to(hit: Hit, alt: boolean, onto?: THREE.Vector3) {
    const s = this.stroke
    if (!s || s.fill) return
    const diag = store.model?.diag ?? 1
    const pixel = tools.brush.pixel && this.texel > 0
    if (pixel && !onto) hit = { ...hit, point: this.at(hit).point }
    const spacing = Math.max(pixel ? this.texel * diag * 0.4 : tools.brush.size * diag * 0.25, 1e-6)
    const dist = s.last.distanceTo(hit.point)
    const steps = Math.floor(dist / spacing)
    for (let i = 1; i <= steps; i++) {
      const p = s.last.clone().lerp(hit.point, (i * spacing) / dist)
      this.dab(onto ? (tools.surface(p, onto, diag * 0.02)?.point ?? p) : p, alt)
    }
    if (steps > 0) s.last.lerp(hit.point, (steps * spacing) / dist)
  }

  end() {
    const s = this.stroke
    this.stroke = undefined
    const paint = store.paint
    if (!s || !paint) return
    store.pushPaint(s.id, s.before, paint.read(s.id))
    paint.changed(s.id)
  }

  // Where the brush paints and how wide: pixel brushes snap to the middle of the
  // texture texel under the cursor and measure whole texels.
  private at(hit: Hit): { point: THREE.Vector3; radius: number } {
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
}
