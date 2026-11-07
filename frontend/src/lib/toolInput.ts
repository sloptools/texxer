import * as THREE from 'three'
import { BrushInput, type Keys } from './brushInput'
import type { Compositor } from './gl/compositor'
import type { Hit, Stage } from './gl/stage'
import { contains, corners, expand, fitted, handles, rotationTowards } from './stamps'
import { snipTiles, tiled } from './snippets.svelte'
import { store } from './store.svelte'
import { tools, type Picked } from './tools.svelte'

const mark = '#ffb020'
const dim = '#b9b6ad'

// Pointer handling for the brush, stamp and volume tools on the 3D view.
export class ToolInput {
  private brush: BrushInput
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
    this.brush = new BrushInput(stage, compositor, changed)
    const el = stage.renderer.domElement
    const down = (e: PointerEvent) => this.down(e)
    const move = (e: PointerEvent) => this.move(e)
    const up = () => this.up()
    const leave = () => this.hover(undefined)
    const keys = (e: KeyboardEvent) => (e.key === 'Shift' || e.key === 'Meta' || e.key === 'Alt') && this.hover(this.lastHit, e)
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

    if (tools.tool === 'brush') this.brush.start(hit, e)
    else if (e.shiftKey && tools.pickedStamp) tools.row(hit.point, hit.normal)
    else if (e.metaKey) this.fitStamp(hit)
    else this.pressStamp(hit, e.altKey)
  }

  private move(e: PointerEvent) {
    if (tools.tool === 'nav') return
    if (this.grip !== undefined) {
      this.dragGrip(e)
      return
    }
    // Over a grip: no preview of a new stamp, a hand to say it can be dragged.
    const grip = tools.tool === 'stamp' && !this.dragging && !this.brush.active ? this.gripAt(e.clientX, e.clientY) : undefined
    this.stage.renderer.domElement.style.cursor = grip !== undefined ? 'grab' : ''
    if (grip !== undefined) {
      if (tools.ghost) {
        tools.ghost = null
        this.changed()
      }
      return
    }
    const hit = this.stage.hit(e.clientX, e.clientY)

    if (this.brush.active) {
      if (hit) this.brush.to(hit, e.altKey)
    } else if (this.dragging && tools.tool === 'volume') {
      if (hit) tools.moveVolume([hit.point.x, hit.point.y, hit.point.z])
    } else if (this.dragging) {
      if (hit) tools.move(hit.point, hit.normal, hit.island)
    }
    this.hover(hit, e)
  }

  private up() {
    this.brush.end()
    this.dragging = false
    this.grip = undefined
  }

  // Cursor feedback: the brush ring, or the frame of the stamp that would be placed or moved.
  hover(hit: Hit | undefined, keys?: Keys) {
    this.lastHit = hit
    this.brush.hover(hit, keys)

    if (tools.tool !== 'stamp') {
      this.stage.setOutline(null)
      if (tools.ghost) {
        tools.ghost = null
        this.changed()
      }
      return
    }

    const over = hit && !this.dragging && !keys?.altKey ? this.stampAt(hit.point) : undefined
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

  // ⌥ places a new one even over others.
  private pressStamp(hit: Hit, alt: boolean) {
    const over = alt ? undefined : this.stampAt(hit.point)
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
    // A mirrored copy lands on another island, so only unmirrored ones keep to theirs.
    if (!tools.mirror) stamp.island = hit.island
    tools.placeStamp(stamp)
    this.changed()
  }

  dispose() {
    this.cleanup()
  }
}
