import * as THREE from 'three'
import { TransformControls } from 'three/examples/jsm/controls/TransformControls.js'
import type { Volume, VolumeShape } from '../types'

export type GizmoMode = 'translate' | 'rotate' | 'scale'

// Line segments of the unit shapes: a box from -1 to 1, a cylinder along y, a sphere.
function outline(shape: VolumeShape): THREE.BufferGeometry {
  if (shape === 'box') return new THREE.EdgesGeometry(new THREE.BoxGeometry(2, 2, 2))

  const points: THREE.Vector3[] = []
  const circle = (at: (a: number) => THREE.Vector3) => {
    const n = 48
    for (let i = 0; i < n; i++) points.push(at((i / n) * Math.PI * 2), at(((i + 1) / n) * Math.PI * 2))
  }
  if (shape === 'cylinder') {
    for (const y of [-1, 1]) circle((a) => new THREE.Vector3(Math.cos(a), y, Math.sin(a)))
    for (let i = 0; i < 4; i++) {
      const a = (i / 4) * Math.PI * 2
      points.push(new THREE.Vector3(Math.cos(a), -1, Math.sin(a)), new THREE.Vector3(Math.cos(a), 1, Math.sin(a)))
    }
  } else {
    circle((a) => new THREE.Vector3(Math.cos(a), Math.sin(a), 0))
    circle((a) => new THREE.Vector3(Math.cos(a), 0, Math.sin(a)))
    circle((a) => new THREE.Vector3(0, Math.cos(a), Math.sin(a)))
  }
  return new THREE.BufferGeometry().setFromPoints(points)
}

// The picked volume mask in the view: its outline, a mirrored one, and a gizmo to
// move, turn and size it.
export class VolumeGizmo {
  private proxy = new THREE.Object3D()
  private mirrored = new THREE.Object3D()
  private controls: TransformControls
  private helper: THREE.Object3D
  private shape: VolumeShape | undefined
  private material = new THREE.LineBasicMaterial({ color: 0x7fc4af, depthTest: false, transparent: true, opacity: 0.8 })
  private mirrorMaterial = new THREE.LineBasicMaterial({ color: 0xb9b6ad, depthTest: false, transparent: true, opacity: 0.4 })

  onChange: (v: Pick<Volume, 'p' | 'q' | 'size'>) => void = () => {}

  constructor(camera: THREE.Camera, el: HTMLElement, scene: THREE.Scene) {
    this.controls = new TransformControls(camera, el)
    this.controls.setSize(0.8)
    this.controls.attach(this.proxy)
    this.helper = this.controls.getHelper()
    this.controls.addEventListener('objectChange', () => {
      const { position: p, quaternion: q, scale: s } = this.proxy
      this.onChange({ p: [p.x, p.y, p.z], q: [q.x, q.y, q.z, q.w], size: [Math.abs(s.x), Math.abs(s.y), Math.abs(s.z)] })
    })
    this.proxy.renderOrder = this.mirrored.renderOrder = 10
    scene.add(this.proxy, this.mirrored, this.helper)
    this.set(null, 0)
  }

  // Over or dragging a handle, so clicks belong to the gizmo.
  get active() {
    return this.controls.axis !== null || this.controls.dragging
  }

  get dragging() {
    return this.controls.dragging
  }

  setMode(mode: GizmoMode) {
    this.controls.setMode(mode)
  }

  get mode(): GizmoMode {
    return this.controls.mode as GizmoMode
  }

  // Shows the volume, or hides it all with null; mirrorX is the model's middle.
  set(v: Volume | null, mirrorX: number) {
    const on = !!v
    this.proxy.visible = this.helper.visible = on
    this.controls.enabled = on
    this.mirrored.visible = on && !!v?.mirror
    if (!v) return

    if (v.shape !== this.shape) {
      this.shape = v.shape
      const geometry = outline(v.shape)
      for (const [o, m] of [[this.proxy, this.material], [this.mirrored, this.mirrorMaterial]] as const) {
        o.clear()
        o.add(new THREE.LineSegments(geometry, m))
      }
    }
    if (!this.controls.dragging) {
      this.proxy.position.set(...v.p)
      this.proxy.quaternion.set(...v.q)
      this.proxy.scale.set(...v.size)
    }

    // Mirrored across the plane x = mirrorX: the same shape with x turned around.
    const flip = new THREE.Matrix4().makeTranslation(2 * mirrorX, 0, 0).multiply(new THREE.Matrix4().makeScale(-1, 1, 1))
    this.proxy.updateMatrix()
    this.mirrored.matrixAutoUpdate = false
    this.mirrored.matrix.copy(flip.multiply(this.proxy.matrix))
  }

  dispose() {
    this.controls.dispose()
  }
}
