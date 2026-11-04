import * as THREE from 'three'
import { fetchBlob, track } from '../assets'
import type { Bake } from './compositor'
import { quadVertex } from './shaders'

// A brush dab: a sphere in model space, painted into every texel whose
// position lies inside it and whose surface faces the camera. Hardness sets how far
// in from the rim it fades, strength the most it paints, grit breaks it up with
// noise that sits on the model, so dabs join into one pattern. With a grunge map
// the dab reveals the map, more of it the stronger the dab.
const dabFragment = /* glsl */ `
precision highp float;

uniform sampler2D uPos;
uniform sampler2D uNrm;
uniform vec3 uCenter;
uniform vec3 uView;
uniform float uRadius;
uniform float uHard;
uniform float uStrength;
uniform float uGrit;
uniform int uErase;
uniform sampler2D uMap;
uniform float uMapScale;
uniform int uUseMap;
uniform float uPixels;
// Color dabs lay the brush color over what is there, premultiplied by coverage.
uniform int uColored;
uniform vec3 uBrush;
uniform float uTexel;
uniform int uBlock;

out vec4 outColor;

float hash(vec3 p) {
  return fract(sin(dot(p, vec3(127.1, 311.7, 74.7))) * 43758.5453);
}

float noise(vec3 p) {
  vec3 i = floor(p);
  vec3 f = fract(p);
  f = f * f * (3.0 - 2.0 * f);
  return mix(
    mix(mix(hash(i), hash(i + vec3(1, 0, 0)), f.x), mix(hash(i + vec3(0, 1, 0)), hash(i + vec3(1, 1, 0)), f.x), f.y),
    mix(mix(hash(i + vec3(0, 0, 1)), hash(i + vec3(1, 0, 1)), f.x), mix(hash(i + vec3(0, 1, 1)), hash(i + vec3(1, 1, 1)), f.x), f.y),
    f.z);
}

void main() {
  ivec2 t = ivec2(gl_FragCoord.xy);
  vec3 p = texelFetch(uPos, t, 0).xyz;
  vec3 n = texelFetch(uNrm, t, 0).xyz;

  float v = 1.0 - smoothstep(uHard * uRadius * 0.999, uRadius, distance(p, uCenter));
  v *= step(0.05, dot(n, uView));
  if (uPixels > 0.0) {
    // Whole texels of the texture: every bake texel of a block decides by the block's middle.
    ivec2 b = min((t / uBlock) * uBlock + uBlock / 2, textureSize(uPos, 0) - 1);
    p = texelFetch(uPos, b, 0).xyz;
    n = texelFetch(uNrm, b, 0).xyz;
    v = step(distance(p, uCenter), (uPixels * 0.5 + 0.05) * uTexel) * step(0.05, dot(n, uView));
  }
  if (uGrit > 0.0) {
    float g = noise(p * 8.0 / uRadius) * 0.6 + noise(p * 23.0 / uRadius) * 0.4;
    v *= smoothstep(uGrit - 0.08, uGrit + 0.08, g);
  }
  v *= uStrength;
  if (uUseMap == 1) {
    // As the grunge masks project it, so painted dirt matches them.
    vec3 w = pow(abs(n), vec3(4.0));
    w /= w.x + w.y + w.z;
    vec3 q = p * uMapScale;
    float g = texture(uMap, vec2(q.z, -q.y)).r * w.x + texture(uMap, q.xz).r * w.y + texture(uMap, vec2(q.x, -q.y)).r * w.z;
    v = v > 0.0 ? smoothstep(1.0 - v - 0.04, 1.0 - v + 0.04, g) : 0.0;
  }
  if (uColored == 1) outColor = uErase == 1 ? vec4(v) : vec4(uBrush * v, v);
  else outColor = vec4(uErase == 1 ? 1.0 - v : v);
}
`

const copyFragment = /* glsl */ `
precision highp float;
uniform sampler2D uSrc;
out vec4 outColor;
void main() {
  outColor = texelFetch(uSrc, ivec2(gl_FragCoord.xy), 0);
}
`

export type Dab = {
  center: THREE.Vector3
  view: THREE.Vector3
  radius: number
  hard: number
  strength: number
  grit: number
  erase: boolean
  // A grunge map the dab reveals, at repeats per model size.
  map?: THREE.Texture
  mapScale?: number
  // A brush color makes a color dab, for layers that paint colors.
  color?: THREE.Color
  // Pixel dabs paint whole texture texels, this many across; texel is their size and
  // block how many bake texels make one.
  pixel?: { count: number; texel: number; block: number }
}

export function premultiply(data: Uint8Array) {
  for (let i = 0; i < data.length; i += 4) {
    const a = data[i + 3] / 255
    for (let c = 0; c < 3; c++) data[i + c] = Math.round(data[i + c] * a)
  }
}

export function unpremultiply(data: Uint8Array) {
  for (let i = 0; i < data.length; i += 4) {
    const a = data[i + 3]
    for (let c = 0; c < 3; c++) data[i + c] = a ? Math.min(255, Math.round((data[i + c] * 255) / a)) : 0
  }
}

// Painted masks, one render target each, kept at the bake's size.
export class PaintStore {
  private targets = new Map<string, THREE.WebGLRenderTarget>()
  // Masks that hold colors, premultiplied, with coverage in alpha.
  readonly colored = new Set<string>()
  private dirty = new Set<string>()
  private scene = new THREE.Scene()
  private camera = new THREE.OrthographicCamera(-1, 1, 1, -1, 0, 1)
  private quad: THREE.Mesh
  private dabMat: THREE.ShaderMaterial
  private copyMat: THREE.ShaderMaterial
  private saveTimer = 0

  onChange: () => void = () => {}

  constructor(
    private renderer: THREE.WebGLRenderer,
    private bake: Bake,
    private url: (id: string) => string,
    private save: (id: string, png: string) => Promise<unknown>,
  ) {
    this.dabMat = new THREE.ShaderMaterial({
      glslVersion: THREE.GLSL3,
      vertexShader: quadVertex,
      fragmentShader: dabFragment,
      uniforms: {
        uPos: { value: bake.pos },
        uNrm: { value: bake.nrm },
        uCenter: { value: new THREE.Vector3() },
        uView: { value: new THREE.Vector3() },
        uRadius: { value: 0.01 },
        uHard: { value: 0.8 },
        uStrength: { value: 1 },
        uGrit: { value: 0 },
        uErase: { value: 0 },
        uMap: { value: null },
        uMapScale: { value: 3 },
        uUseMap: { value: 0 },
        uPixels: { value: 0 },
        uTexel: { value: 0 },
        uBlock: { value: 1 },
        uColored: { value: 0 },
        uBrush: { value: new THREE.Color() },
      },
      blending: THREE.CustomBlending,
      blendSrc: THREE.OneFactor,
      blendDst: THREE.OneFactor,
      depthTest: false,
      depthWrite: false,
    })
    this.copyMat = new THREE.ShaderMaterial({
      glslVersion: THREE.GLSL3,
      vertexShader: quadVertex,
      fragmentShader: copyFragment,
      uniforms: { uSrc: { value: null } },
      blending: THREE.NoBlending,
      depthTest: false,
      depthWrite: false,
    })
    this.quad = new THREE.Mesh(new THREE.PlaneGeometry(2, 2), this.dabMat)
    this.quad.frustumCulled = false
    this.scene.add(this.quad)
  }

  get size() {
    return this.bake.size
  }

  // The mask's texture; new ones start empty and load their PNG if there is one.
  get(id: string): THREE.Texture {
    let t = this.targets.get(id)
    if (!t) {
      t = this.create(id)
      track(this.load(id))
    }
    return t.texture
  }

  private create(id: string) {
    const size = this.bake.size
    const t = new THREE.WebGLRenderTarget(size, size, {
      minFilter: THREE.NearestFilter,
      magFilter: THREE.NearestFilter,
      generateMipmaps: false,
      depthBuffer: false,
    })
    const prev = this.renderer.getRenderTarget()
    this.renderer.setRenderTarget(t)
    this.renderer.setClearColor(0x000000, 0)
    this.renderer.clear()
    this.renderer.setRenderTarget(prev)
    this.targets.set(id, t)
    return t
  }

  private async load(id: string) {
    const blob = await fetchBlob(this.url(id))
    if (!blob) return
    const img = await createImageBitmap(blob)

    const size = this.bake.size
    const canvas = document.createElement('canvas')
    canvas.width = canvas.height = size
    const ctx = canvas.getContext('2d')!
    ctx.imageSmoothingEnabled = false
    ctx.drawImage(img, 0, 0, size, size)
    const data = new Uint8Array(ctx.getImageData(0, 0, size, size).data.buffer)
    if (this.colored.has(id)) premultiply(data)
    this.write(id, data, false)
  }

  dab(id: string, d: Dab) {
    const t = this.targets.get(id) ?? this.create(id)
    const u = this.dabMat.uniforms
    u.uCenter.value.copy(d.center)
    u.uView.value.copy(d.view)
    u.uRadius.value = d.radius
    u.uHard.value = d.hard
    u.uStrength.value = d.strength
    u.uGrit.value = d.grit
    u.uErase.value = d.erase ? 1 : 0
    u.uMap.value = d.map ?? null
    u.uMapScale.value = d.mapScale ?? 3
    u.uUseMap.value = d.map ? 1 : 0
    u.uPixels.value = d.pixel?.count ?? 0
    u.uTexel.value = d.pixel?.texel ?? 0
    u.uBlock.value = d.pixel?.block ?? 1
    u.uColored.value = d.color ? 1 : 0
    if (d.color) {
      u.uBrush.value.copy(d.color)
      this.colored.add(id)
    }
    // Coverage only: the strongest dab stays. Colors: over, or scaled down to erase.
    const m = this.dabMat
    m.blendEquation = d.color ? THREE.AddEquation : d.erase ? THREE.MinEquation : THREE.MaxEquation
    m.blendSrc = d.color ? (d.erase ? THREE.ZeroFactor : THREE.OneFactor) : THREE.OneFactor
    m.blendDst = d.color ? THREE.OneMinusSrcAlphaFactor : THREE.OneFactor
    this.draw(m, t)
    this.dirty.add(id)
  }

  read(id: string): Uint8Array {
    const size = this.bake.size
    const data = new Uint8Array(size * size * 4)
    const t = this.targets.get(id)
    if (t) this.renderer.readRenderTargetPixels(t, 0, 0, size, size, data)
    return data
  }

  // Whether nothing was painted into the mask.
  empty(id: string): boolean {
    if (!this.targets.has(id)) return true
    const data = this.read(id)
    for (let i = 0; i < data.length; i += 4) if (data[i]) return false
    return true
  }

  write(id: string, data: Uint8Array, save = true) {
    const size = this.bake.size
    if (data.length !== size * size * 4) return
    const t = this.targets.get(id) ?? this.create(id)
    const src = new THREE.DataTexture(data, size, size, THREE.RGBAFormat)
    src.needsUpdate = true
    this.copyMat.uniforms.uSrc.value = src
    this.draw(this.copyMat, t)
    src.dispose()
    if (save) this.changed(id)
    this.onChange()
  }

  clone(from: string, to: string) {
    if (!this.targets.has(from)) return
    if (this.colored.has(from)) this.colored.add(to)
    this.write(to, this.read(from))
  }

  // Saves strokes a moment after they end.
  changed(id: string) {
    this.dirty.add(id)
    clearTimeout(this.saveTimer)
    this.saveTimer = setTimeout(() => this.flush(), 400)
  }

  async flush() {
    clearTimeout(this.saveTimer)
    const saves = [...this.dirty].map((id) => this.save(id, this.png(id)))
    this.dirty.clear()
    await Promise.all(saves)
  }

  private png(id: string): string {
    const size = this.bake.size
    const canvas = document.createElement('canvas')
    canvas.width = canvas.height = size
    const ctx = canvas.getContext('2d')!
    // Coverage only: opaque, so the canvas doesn't premultiply the values away.
    // Colors: straight, as PNGs keep them.
    const data = this.read(id)
    if (this.colored.has(id)) unpremultiply(data)
    else for (let i = 3; i < data.length; i += 4) data[i] = 255
    ctx.putImageData(new ImageData(new Uint8ClampedArray(data), size, size), 0, 0)
    return canvas.toDataURL('image/png').split(',')[1]
  }

  private draw(material: THREE.ShaderMaterial, to: THREE.WebGLRenderTarget) {
    const r = this.renderer
    const prev = r.getRenderTarget()
    const autoClear = r.autoClear
    this.quad.material = material
    material.uniformsNeedUpdate = true
    // Dabs build on what is there, so no clear before drawing.
    r.autoClear = false
    r.setRenderTarget(to)
    r.render(this.scene, this.camera)
    r.setRenderTarget(prev)
    r.autoClear = autoClear
  }

  dispose() {
    this.flush()
    for (const t of this.targets.values()) t.dispose()
    this.dabMat.dispose()
    this.copyMat.dispose()
    this.quad.geometry.dispose()
  }
}
