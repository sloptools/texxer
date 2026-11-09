import * as THREE from 'three'
import type { FlatMap } from '../types'
import { isMouseWheel } from './wheel'

type GestureEvent = UIEvent & { scale: number; clientX: number; clientY: number }

export type FlatSources = {
  color: THREE.Texture
  mask: THREE.Texture
  // Made when a layer first glows.
  glow: () => THREE.Texture | null
  geo: THREE.Texture
  pos: THREE.Texture
  size: number
}

const modes: Record<FlatMap, [mode: number, channel: number]> = {
  texture: [0, 0],
  mask: [0, 0],
  glow: [0, 0],
  ao: [1, 0],
  edges: [1, 1],
  cavity: [1, 2],
  parts: [2, 3],
}

const black = new THREE.DataTexture(new Uint8Array([0, 0, 0, 255]), 1, 1)
black.needsUpdate = true

const vertex = /* glsl */ `
out vec2 vUv;
void main() {
  vUv = vec2(uv.x, 1.0 - uv.y);
  gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
}
`

const fragment = /* glsl */ `
uniform sampler2D uTex;
uniform sampler2D uGeo;
uniform sampler2D uPos;
uniform sampler2D uOverlap;
uniform int uMode;
uniform int uChannel;
uniform float uSize;
uniform float uHover;
uniform bool uUV;
in vec2 vUv;
out vec4 color;

vec3 partColor(float id) {
  vec3 h = fract(sin(vec3(id + 1.0) * vec3(12.9898, 78.233, 37.719)) * 43758.5453);
  return 0.25 + 0.6 * h;
}

void main() {
  vec4 geo = textureLod(uGeo, vUv, 0.0);
  float part = textureLod(uPos, vUv, 0.0).w;
  vec3 c;
  if (uMode == 0) c = textureLod(uTex, vUv, 0.0).rgb;
  else if (uMode == 1) c = vec3(geo[uChannel]);
  else c = partColor(part);

  if (uUV) {
    if (geo.a < 0.5) c *= 0.3;
    if (texelFetch(uOverlap, ivec2(vUv * uSize), 0).r > 1.5) c = mix(c, vec3(0.95, 0.25, 0.2), 0.55);
  }
  if (uHover >= 0.0 && abs(part - uHover) < 0.5 && geo.a > 0.5) c = mix(c, vec3(0.9, 1.0, 0.95), 0.18);

  // Texel grid once texels are big enough to tell apart.
  vec2 p = vUv * uSize;
  vec2 w = fwidth(p);
  if (w.x < 0.125) {
    vec2 g = step(fract(p), w);
    c = mix(c, vec3(0.5), max(g.x, g.y) * 0.25);
  }
  color = vec4(c, 1.0);
}
`

// The 2D view: the texture or a bake laid flat, with the UV layout over it.
// It shares the 3D view's renderer, so it shows the compositor's targets as they are.
export class Flat {
  active = false
  map: FlatMap = 'texture'
  showUV = true
  onHover: (texel: [number, number] | null, part: number) => void = () => {}
  onClick: (e: PointerEvent, part: number) => void = () => {}

  private scene = new THREE.Scene()
  private camera = new THREE.OrthographicCamera(0, 1, 0, -1, -1, 1)
  private material: THREE.ShaderMaterial
  private wire = new THREE.LineSegments(new THREE.BufferGeometry(), new THREE.LineBasicMaterial({ color: 0xb9b6ad, transparent: true, opacity: 0.3, depthTest: false }))
  private seams = new THREE.LineSegments(new THREE.BufferGeometry(), new THREE.LineBasicMaterial({ color: 0x7fc4af, transparent: true, opacity: 0.9, depthTest: false }))

  // Triangles in UV space, drawn additively to count how often each texel is covered.
  private tris = new THREE.BufferGeometry()
  private overlap: THREE.WebGLRenderTarget | undefined
  private overlapDirty = true

  private sources: FlatSources | undefined
  private posData: Float32Array | undefined
  private center = new THREE.Vector2(0.5, 0.5)
  // Screen pixels per UV unit.
  private scale = 512
  private width = 1
  private height = 1
  private gestureScale = 1
  private drag: { x: number; y: number; moved: boolean } | undefined
  private cleanup: () => void

  constructor(
    private renderer: THREE.WebGLRenderer,
    private el: HTMLElement,
  ) {
    this.material = new THREE.ShaderMaterial({
      glslVersion: THREE.GLSL3,
      vertexShader: vertex,
      fragmentShader: fragment,
      uniforms: {
        uTex: { value: null },
        uGeo: { value: null },
        uPos: { value: null },
        uOverlap: { value: null },
        uMode: { value: 0 },
        uChannel: { value: 0 },
        uSize: { value: 1 },
        uHover: { value: -1 },
        uUV: { value: true },
      },
    })
    const quad = new THREE.Mesh(new THREE.PlaneGeometry(1, 1), this.material)
    quad.position.set(0.5, -0.5, 0)
    this.wire.renderOrder = 1
    this.seams.renderOrder = 2
    this.scene.add(quad, this.wire, this.seams)

    const listen = <K extends string>(type: K, fn: (e: any) => void) => {
      const guarded = (e: Event) => this.active && fn(e)
      el.addEventListener(type, guarded, { passive: false })
      return () => el.removeEventListener(type, guarded)
    }
    const offs = [
      listen('wheel', (e: WheelEvent) => this.wheel(e)),
      listen('gesturestart', (e: GestureEvent) => {
        e.preventDefault()
        this.gestureScale = 1
      }),
      listen('gesturechange', (e: GestureEvent) => {
        e.preventDefault()
        this.zoomAt(e.clientX, e.clientY, e.scale / this.gestureScale)
        this.gestureScale = e.scale
      }),
      listen('gestureend', (e: GestureEvent) => e.preventDefault()),
      listen('pointerdown', (e: PointerEvent) => this.down(e)),
      listen('pointermove', (e: PointerEvent) => this.move(e)),
      listen('pointerup', (e: PointerEvent) => this.up(e)),
      listen('pointerleave', () => this.onHover(null, -1)),
    ]
    this.cleanup = () => offs.forEach((off) => off())
  }

  setBackground(color: THREE.Color) {
    this.scene.background = color
  }

  setMeshes(meshes: THREE.Mesh[]) {
    const tris: number[] = []
    for (const mesh of meshes) {
      const uv = mesh.geometry.getAttribute('uv')
      if (!uv) continue
      const index = mesh.geometry.getIndex()
      const count = index ? index.count : uv.count
      for (let i = 0; i < count; i++) {
        const k = index ? index.getX(i) : i
        tris.push(uv.getX(k), uv.getY(k), 0)
      }
    }
    this.tris.setAttribute('position', new THREE.Float32BufferAttribute(tris, 3))
    this.overlapDirty = true

    // Edges used by one triangle only are where the UV layout is cut.
    const edges = new Map<string, { a: number; b: number; n: number }>()
    const key = (i: number) => `${Math.round(tris[i * 3] * 1e5)},${Math.round(tris[i * 3 + 1] * 1e5)}`
    for (let t = 0; t < tris.length / 3; t += 3) {
      for (const [a, b] of [[t, t + 1], [t + 1, t + 2], [t + 2, t]]) {
        const ka = key(a)
        const kb = key(b)
        const k = ka < kb ? ka + '|' + kb : kb + '|' + ka
        const e = edges.get(k)
        if (e) e.n++
        else edges.set(k, { a, b, n: 1 })
      }
    }

    const inner: number[] = []
    const outer: number[] = []
    for (const { a, b, n } of edges.values()) {
      const list = n === 1 ? outer : inner
      list.push(tris[a * 3], -tris[a * 3 + 1], 0, tris[b * 3], -tris[b * 3 + 1], 0)
    }
    this.wire.geometry.setAttribute('position', new THREE.Float32BufferAttribute(inner, 3))
    this.seams.geometry.setAttribute('position', new THREE.Float32BufferAttribute(outer, 3))
  }

  setSources(s: FlatSources, posData: Float32Array) {
    if (this.sources?.size !== s.size) this.overlapDirty = true
    this.sources = s
    this.posData = posData
  }

  setHover(part: number) {
    this.material.uniforms.uHover.value = part
  }

  resize(w: number, h: number) {
    this.width = Math.max(w, 1)
    this.height = Math.max(h, 1)
  }

  fit() {
    // Leaves room for the bar on top.
    this.scale = Math.max(Math.min(this.width - 40, this.height - 110), 64)
    this.center.set(0.5, 0.5 - 22 / this.scale)
  }

  private zoomAt(x: number, y: number, factor: number) {
    const before = this.toUV(x, y)
    this.scale = THREE.MathUtils.clamp(this.scale * factor, 32, 1 << 16)
    const after = this.toUV(x, y)
    this.center.add(before.sub(after))
  }

  private wheel(e: WheelEvent) {
    e.preventDefault()
    const scale = e.deltaMode === 1 ? 16 : 1
    if (e.ctrlKey) this.zoomAt(e.clientX, e.clientY, Math.exp(-e.deltaY * scale * 0.01))
    else if (isMouseWheel(e)) this.zoomAt(e.clientX, e.clientY, Math.exp(-Math.sign(e.deltaY) * 0.15))
    else this.center.add(new THREE.Vector2(e.deltaX, e.deltaY).multiplyScalar(scale / this.scale))
  }

  private down(e: PointerEvent) {
    if (e.button > 2) return
    this.el.setPointerCapture(e.pointerId)
    this.drag = { x: e.clientX, y: e.clientY, moved: false }
  }

  private move(e: PointerEvent) {
    const d = this.drag
    if (!d) {
      this.hover(e)
      return
    }
    const dx = e.clientX - d.x
    const dy = e.clientY - d.y
    if (!d.moved && Math.hypot(dx, dy) < 3) return
    d.moved = true
    d.x = e.clientX
    d.y = e.clientY
    this.center.sub(new THREE.Vector2(dx, dy).divideScalar(this.scale))
  }

  private up(e: PointerEvent) {
    if (this.drag && !this.drag.moved) this.onClick(e, this.texel(e.clientX, e.clientY)?.part ?? -1)
    this.drag = undefined
  }

  private hover(e: PointerEvent) {
    const t = this.texel(e.clientX, e.clientY)
    this.onHover(t ? [t.x, t.y] : null, t?.part ?? -1)
  }

  private toUV(x: number, y: number) {
    const rect = this.el.getBoundingClientRect()
    return new THREE.Vector2(x - rect.left - this.width / 2, y - rect.top - this.height / 2).divideScalar(this.scale).add(this.center)
  }

  // The texture texel under a screen point, and the part covering it.
  private texel(x: number, y: number) {
    const s = this.sources
    if (!s) return
    const uv = this.toUV(x, y)
    if (uv.x < 0 || uv.y < 0 || uv.x >= 1 || uv.y >= 1) return
    let part = -1
    if (this.posData) {
      const n = Math.round(Math.sqrt(this.posData.length / 4))
      const i = Math.floor(uv.y * n) * n + Math.floor(uv.x * n)
      part = this.posData[i * 4 + 3]
    }
    return { x: Math.floor(uv.x * s.size), y: Math.floor(uv.y * s.size), part }
  }

  private renderOverlap(size: number) {
    this.overlap?.dispose()
    this.overlap = new THREE.WebGLRenderTarget(size, size, { type: THREE.HalfFloatType, depthBuffer: false })
    const mesh = new THREE.Mesh(
      this.tris,
      new THREE.MeshBasicMaterial({ color: 0xffffff, blending: THREE.AdditiveBlending, side: THREE.DoubleSide, depthTest: false }),
    )
    const scene = new THREE.Scene().add(mesh)
    const camera = new THREE.OrthographicCamera(0, 1, 1, 0, -1, 1)

    const r = this.renderer
    const before = r.getRenderTarget()
    const clear = r.getClearColor(new THREE.Color())
    const alpha = r.getClearAlpha()
    r.setRenderTarget(this.overlap)
    r.setClearColor(0x000000, 0)
    r.clear()
    r.render(scene, camera)
    r.setRenderTarget(before)
    r.setClearColor(clear, alpha)
    ;(mesh.material as THREE.Material).dispose()
    this.overlapDirty = false
  }

  render() {
    const s = this.sources
    if (!s) return
    if (this.overlapDirty) this.renderOverlap(s.size)

    const [mode, channel] = modes[this.map]
    const u = this.material.uniforms
    u.uTex.value = this.map === 'mask' ? s.mask : this.map === 'glow' ? (s.glow() ?? black) : s.color
    u.uGeo.value = s.geo
    u.uPos.value = s.pos
    u.uOverlap.value = this.overlap!.texture
    u.uMode.value = mode
    u.uChannel.value = channel
    u.uSize.value = s.size
    u.uUV.value = this.showUV
    this.wire.visible = this.seams.visible = this.showUV

    const hw = this.width / 2 / this.scale
    const hh = this.height / 2 / this.scale
    const c = this.camera
    c.left = this.center.x - hw
    c.right = this.center.x + hw
    c.top = -this.center.y + hh
    c.bottom = -this.center.y - hh
    c.updateProjectionMatrix()

    const r = this.renderer
    const tone = r.toneMapping
    r.toneMapping = THREE.NoToneMapping
    r.setRenderTarget(null)
    r.render(this.scene, c)
    r.toneMapping = tone
  }

  dispose() {
    this.cleanup()
    this.overlap?.dispose()
    this.material.dispose()
    this.tris.dispose()
    this.wire.geometry.dispose()
    this.seams.geometry.dispose()
  }
}
