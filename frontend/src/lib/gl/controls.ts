import * as THREE from 'three'

type Pick = (x: number, y: number) => THREE.Vector3 | undefined

// Gesture events are WebKit only, so WKWebView has them but the DOM types don't.
type GestureEvent = UIEvent & { scale: number; clientX: number; clientY: number }

const orbitSpeed = 0.006
const dragSpeed = 0.008
const maxPitch = Math.PI / 2 - 0.01

// Trackpad first: two fingers orbit, shift pans, pinch zooms towards the cursor,
// option turns the key light. Dragging orbits too, for a mouse.
export class TrackpadControls {
  target = new THREE.Vector3()
  yaw = 0.8
  pitch = 0.35
  dist = 3

  private goal = { target: new THREE.Vector3(), yaw: 0.8, pitch: 0.35, dist: 3 }
  private radius = 1
  private gestureScale = 1
  private drag: { x: number; y: number; moved: boolean; mode: 'orbit' | 'pan' | 'light' } | undefined
  private cleanup: () => void

  // Off while a tool owns the left button; the trackpad still orbits.
  leftDrag = true
  onLight: (delta: number) => void = () => {}
  onClick: (e: PointerEvent) => void = () => {}

  constructor(
    private camera: THREE.PerspectiveCamera,
    private el: HTMLElement,
    private pick: Pick,
  ) {
    const opts = { passive: false }
    const listen = <K extends string>(type: K, fn: (e: any) => void) => {
      el.addEventListener(type, fn, opts)
      return () => el.removeEventListener(type, fn)
    }
    const offs = [
      listen('wheel', (e: WheelEvent) => this.wheel(e)),
      listen('gesturestart', (e: GestureEvent) => {
        e.preventDefault()
        this.gestureScale = 1
      }),
      listen('gesturechange', (e: GestureEvent) => {
        e.preventDefault()
        this.zoomAt(e.clientX, e.clientY, this.gestureScale / e.scale)
        this.gestureScale = e.scale
      }),
      listen('gestureend', (e: GestureEvent) => e.preventDefault()),
      listen('pointerdown', (e: PointerEvent) => this.down(e)),
      listen('pointermove', (e: PointerEvent) => this.move(e)),
      listen('pointerup', (e: PointerEvent) => this.up(e)),
    ]
    this.cleanup = () => offs.forEach((off) => off())
  }

  private wheel(e: WheelEvent) {
    e.preventDefault()
    const scale = e.deltaMode === 1 ? 16 : 1
    const dx = e.deltaX * scale
    const dy = e.deltaY * scale

    if (e.ctrlKey) this.zoomAt(e.clientX, e.clientY, Math.exp(dy * 0.01))
    else if (e.altKey) this.onLight(dx + dy)
    else if (e.shiftKey) this.pan(dx, dy)
    else this.orbit(dx * orbitSpeed, dy * orbitSpeed)
  }

  private down(e: PointerEvent) {
    if (e.button > 2 || (e.button === 0 && !this.leftDrag)) return
    this.el.setPointerCapture(e.pointerId)
    const mode = e.altKey ? 'light' : e.shiftKey || e.button === 1 || e.button === 2 ? 'pan' : 'orbit'
    this.drag = { x: e.clientX, y: e.clientY, moved: false, mode }
  }

  private move(e: PointerEvent) {
    const d = this.drag
    if (!d) return
    const dx = e.clientX - d.x
    const dy = e.clientY - d.y
    if (!d.moved && Math.hypot(dx, dy) < 3) return
    d.moved = true
    d.x = e.clientX
    d.y = e.clientY

    if (d.mode === 'light') this.onLight(dx)
    else if (d.mode === 'pan') this.pan(-dx, -dy)
    else this.orbit(-dx * dragSpeed, -dy * dragSpeed)
  }

  private up(e: PointerEvent) {
    if (this.drag && !this.drag.moved) this.onClick(e)
    this.drag = undefined
  }

  orbit(dYaw: number, dPitch: number) {
    this.goal.yaw += dYaw
    this.goal.pitch = THREE.MathUtils.clamp(this.goal.pitch - dPitch, -maxPitch, maxPitch)
  }

  pan(dx: number, dy: number) {
    const perPixel = (2 * this.goal.dist * Math.tan(THREE.MathUtils.degToRad(this.camera.fov / 2))) / this.el.clientHeight
    const right = new THREE.Vector3().setFromMatrixColumn(this.camera.matrixWorld, 0)
    const up = new THREE.Vector3().setFromMatrixColumn(this.camera.matrixWorld, 1)
    this.goal.target.addScaledVector(right, dx * perPixel).addScaledVector(up, -dy * perPixel)
  }

  // Zooms so the point under the cursor stays put.
  zoomAt(x: number, y: number, factor: number) {
    const dist = THREE.MathUtils.clamp(this.goal.dist * factor, this.radius * 0.05, this.radius * 30)
    const hit = this.pick(x, y)
    if (hit) this.goal.target.lerp(hit, 1 - dist / this.goal.dist)
    this.goal.dist = dist
  }

  frame(box: THREE.Box3) {
    const sphere = box.getBoundingSphere(new THREE.Sphere())
    this.radius = sphere.radius
    this.goal.target.copy(sphere.center)
    this.goal.dist = sphere.radius / Math.sin(THREE.MathUtils.degToRad(this.camera.fov / 2)) * 1.1
    this.camera.near = sphere.radius / 200
    this.camera.far = sphere.radius * 200
    this.camera.updateProjectionMatrix()
  }

  focus(point: THREE.Vector3) {
    this.goal.target.copy(point)
    this.goal.dist = Math.min(this.goal.dist, this.radius * 1.2)
  }

  view(yaw: number, pitch: number) {
    // Turn the short way round.
    const turns = Math.round((this.goal.yaw - yaw) / (Math.PI * 2))
    this.goal.yaw = yaw + turns * Math.PI * 2
    this.goal.pitch = pitch
  }

  // Snaps to the goal on the first frame, then eases towards it.
  update(dt: number, snap = false) {
    const k = snap ? 1 : 1 - Math.exp(-dt * 22)
    this.yaw += (this.goal.yaw - this.yaw) * k
    this.pitch += (this.goal.pitch - this.pitch) * k
    this.dist += (this.goal.dist - this.dist) * k
    this.target.lerp(this.goal.target, k)

    const c = Math.cos(this.pitch)
    this.camera.position.set(Math.sin(this.yaw) * c, Math.sin(this.pitch), Math.cos(this.yaw) * c).multiplyScalar(this.dist).add(this.target)
    this.camera.lookAt(this.target)
  }

  dispose() {
    this.cleanup()
  }
}
