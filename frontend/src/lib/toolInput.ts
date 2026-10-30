import * as THREE from 'three'
import type { Compositor } from './gl/compositor'
import type { Dab } from './gl/paint'
import type { Hit, Stage } from './gl/stage'
import { contains, corners, expand } from './stamps'
import { store } from './store.svelte'
import { tools, type Picked } from './tools.svelte'

const accent = '#7fc4af'
const dim = '#b9b6ad'

// Pointer handling for the brush and stamp tools on the 3D view.
export class ToolInput {
  private stroke: { id: string; before: Uint8Array; last: THREE.Vector3 } | undefined
  private lastDab: THREE.Vector3 | undefined
  private dragging = false
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
    el.addEventListener('pointerdown', down)
    el.addEventListener('pointermove', move)
    el.addEventListener('pointerup', up)
    el.addEventListener('pointerleave', leave)
    this.cleanup = () => {
      el.removeEventListener('pointerdown', down)
      el.removeEventListener('pointermove', move)
      el.removeEventListener('pointerup', up)
      el.removeEventListener('pointerleave', leave)
    }
  }

  private down(e: PointerEvent) {
    if (e.button !== 0 || tools.tool === 'nav') return
    const hit = this.stage.hit(e.clientX, e.clientY)
    if (!hit) return
    this.stage.renderer.domElement.setPointerCapture(e.pointerId)

    if (tools.tool === 'brush') this.startStroke(hit, e)
    else if (e.shiftKey && tools.pickedStamp) tools.row(hit.point, hit.normal)
    else this.pressStamp(hit)
  }

  private move(e: PointerEvent) {
    if (tools.tool === 'nav') return
    const hit = this.stage.hit(e.clientX, e.clientY)

    if (this.stroke) {
      if (hit) this.strokeTo(hit, e.altKey)
    } else if (this.dragging) {
      if (hit) tools.move(hit.point, hit.normal)
    }
    this.hover(hit)
  }

  private up() {
    if (this.stroke) this.endStroke()
    this.dragging = false
  }

  // Cursor feedback: the brush ring, or the frame of the stamp that would be placed or moved.
  hover(hit: Hit | undefined) {
    const diag = store.model?.diag ?? 1
    this.stage.setRing(tools.tool === 'brush' && hit ? { point: hit.point, normal: hit.normal, radius: tools.brush.size * diag } : null)

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
      this.stage.setOutline(corners(tools.ghost!), accent)
    } else {
      tools.ghost = null
      const s = over ? store.layer?.masks[over.mask].stamps[over.index] : tools.pickedStamp
      this.stage.setOutline(s ? corners(s) : null, over ? dim : accent)
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

  private pressStamp(hit: Hit) {
    const over = this.stampAt(hit.point)
    if (over) tools.pick(over)
    else {
      tools.ghostAt(hit.point, hit.normal, this.stage.cameraUp)
      tools.placeGhost()
    }
    this.dragging = true
  }

  private startStroke(hit: Hit, e: PointerEvent) {
    const paint = store.paint
    const mask = store.ensureMask('paint')
    if (!paint || mask < 0) return
    const id = store.layer!.masks[mask].paint
    paint.get(id)
    this.stroke = { id, before: paint.read(id), last: hit.point.clone() }

    // Shift-click draws a straight line from the last stroke.
    if (e.shiftKey && this.lastDab) {
      this.stroke.last = this.lastDab.clone()
      this.strokeTo(hit, e.altKey)
    } else this.dab(hit.point, e.altKey)
  }

  private strokeTo(hit: Hit, alt: boolean) {
    const s = this.stroke!
    const spacing = Math.max(tools.brush.size * (store.model?.diag ?? 1) * 0.25, 1e-6)
    const dist = s.last.distanceTo(hit.point)
    const steps = Math.floor(dist / spacing)
    for (let i = 1; i <= steps; i++) this.dab(s.last.clone().lerp(hit.point, (i * spacing) / dist), alt)
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
      erase: tools.brush.erase !== alt,
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
