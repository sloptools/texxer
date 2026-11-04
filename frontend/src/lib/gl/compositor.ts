import * as THREE from 'three'
import { newMask } from '../project'
import { expand, frame } from '../stamps'
import type { BakeSettings, Layer, Project, Stamp, Volume } from '../types'
import { StampAtlas, atlasGrid, atlasInset } from './atlas'
import type { PaintStore } from './paint'
import { fetchBlob, imageURL, track } from '../assets'
import { medianCut, refine, type RGB } from './mediancut'
import { blendIndex, finishFragment, kindIndex, shapeIndex, layerFragment, lightFragment, lookupRows, maskRows, maxGroups, maxIslands, maxMasks, maxStamps, quadVertex, reduceFragment } from './shaders'

// Where the model sits: bake positions are (p - min) / diag.
export type ModelFrame = { min: number[]; diag: number; extent: number[] }

export type Bake = {
  size: number
  geo: THREE.DataTexture
  pos: THREE.DataTexture
  nrm: THREE.DataTexture
  // Widths of the edge and cavity fields, as fractions of the model's diagonal.
  widths: [number, number]
  // The model's own colors, when a layer asked for them.
  source?: THREE.DataTexture
  // CPU copies of the position and normal maps, for picking parts and islands.
  posData: Float32Array
  nrmData: Float32Array
}

// source is the model key of the kept originals to take the model's own colors from.
export async function loadBake(path: string, size: number, layout: number, b: BakeSettings, source = ''): Promise<Bake> {
  const q = `path=${encodeURIComponent(path)}&size=${size}&unwrap=${layout}&edges=${b.edges}&cavity=${b.cavity}&detail=${b.detail}&source=${encodeURIComponent(source)}`
  const blob = await fetchBlob(`/bake?${q}`, 2)
  if (!blob) throw new Error('bake failed')
  const buf = await blob.arrayBuffer()

  const n = size * size * 4
  const texture = (i: number) => {
    const t = new THREE.DataTexture(new Float32Array(buf, i * n * 4, n), size, size, THREE.RGBAFormat, THREE.FloatType)
    t.magFilter = t.minFilter = THREE.NearestFilter
    t.needsUpdate = true
    return t
  }
  const bake: Bake = { size, geo: texture(0), pos: texture(1), nrm: texture(2), widths: [b.edges, b.cavity], posData: new Float32Array(buf, n * 4, n), nrmData: new Float32Array(buf, 2 * n * 4, n) }
  if (source) bake.source = texture(3)
  return bake
}

const none = new THREE.DataTexture(new Float32Array(4), 1, 1, THREE.RGBAFormat, THREE.FloatType)
none.needsUpdate = true

// With mipmaps, the finished texture is sampled like Godot's nearest_mipmap filter,
// so the view at a distance or at game resolution matches the game.
function target(size: number, mipmaps = false) {
  return new THREE.WebGLRenderTarget(size, size, {
    minFilter: mipmaps ? THREE.NearestMipmapLinearFilter : THREE.NearestFilter,
    magFilter: THREE.NearestFilter,
    generateMipmaps: mipmaps,
    depthBuffer: false,
  })
}

function lightDir(azimuth: number, elevation: number) {
  const az = THREE.MathUtils.degToRad(azimuth)
  const el = THREE.MathUtils.degToRad(elevation)
  return new THREE.Vector3(Math.cos(el) * Math.cos(az), Math.sin(el), Math.cos(el) * Math.sin(az))
}

function shader(fragmentShader: string, uniforms: Record<string, THREE.IUniform>) {
  return new THREE.ShaderMaterial({ glslVersion: THREE.GLSL3, vertexShader: quadVertex, fragmentShader, uniforms })
}

// Composites the layer stack in UV space at the bake's size, one fullscreen pass
// per layer, bakes light into it, then crunches it down to the texture size.
export class Compositor {
  readonly out: THREE.WebGLRenderTarget
  readonly mask: THREE.WebGLRenderTarget
  private ping: [THREE.WebGLRenderTarget, THREE.WebGLRenderTarget]
  private reduced: THREE.WebGLRenderTarget
  // The emissive map and its stack, made when a layer first glows.
  private emitPing: [THREE.WebGLRenderTarget, THREE.WebGLRenderTarget] | undefined
  emission: THREE.WebGLRenderTarget | undefined
  private lightMat: THREE.ShaderMaterial
  private reduceMat: THREE.ShaderMaterial
  // The auto palette, picked from the texture a moment after it changes.
  autoColors: RGB[] = []
  private autoCount = 0
  private autoPinned = 0

  private scene = new THREE.Scene()
  private camera = new THREE.OrthographicCamera(-1, 1, 1, -1, 0, 1)
  private quad: THREE.Mesh
  private layerMat: THREE.ShaderMaterial
  private finishMat: THREE.ShaderMaterial

  private parts = new THREE.DataTexture(new Uint8Array(256 * lookupRows), 256, lookupRows, THREE.RedFormat)
  private islands = new THREE.DataTexture(new Uint8Array(maxIslands * lookupRows), maxIslands, lookupRows, THREE.RedFormat)
  // Folders with a zone, by id, and their row in the lookups.
  private zones = new Map<string, number>()
  private palette = new THREE.DataTexture(new Uint8Array(256 * 4), 256, 1, THREE.RGBAFormat)

  private atlas = new StampAtlas()
  private grunges = new Map<string, THREE.Texture>()
  onLoad: () => void = () => {}

  constructor(
    private renderer: THREE.WebGLRenderer,
    readonly bake: Bake,
    private model: ModelFrame,
    private paint: PaintStore,
    readonly factor: number,
  ) {
    const vec4s = (n: number) => Array.from({ length: n }, () => new THREE.Vector4())
    const size = bake.size
    this.out = target(size / factor, true)
    this.reduced = target(size / factor)
    this.mask = target(size)
    this.ping = [target(size), target(size)]

    this.lightMat = shader(lightFragment, {
      uSrc: { value: null },
      uGeo: { value: bake.geo },
      uNrm: { value: bake.nrm },
      uL: { value: new THREE.Vector3() },
      uUnderL: { value: lightDir(0, -80) },
      uBoth: { value: 1 },
      uAoFloor: { value: 0.45 },
      uFormFloor: { value: 0.72 },
      uKey: { value: 0.45 },
      uKeyColor: { value: new THREE.Color() },
      uUnder: { value: 0.12 },
      uUnderColor: { value: new THREE.Color() },
      uRim: { value: 0.3 },
    })
    this.reduceMat = shader(reduceFragment, { uSrc: { value: null }, uFactor: { value: factor }, uSharpen: { value: 0.2 } })

    this.layerMat = new THREE.ShaderMaterial({
      glslVersion: THREE.GLSL3,
      vertexShader: quadVertex,
      fragmentShader: layerFragment,
      uniforms: {
        uPrev: { value: null },
        uGeo: { value: bake.geo },
        uPos: { value: bake.pos },
        uNrm: { value: bake.nrm },
        uParts: { value: this.parts },
        uIslands: { value: this.islands },
        uZone: { value: -1 },
        uExtent: { value: new THREE.Vector3(...model.extent) },
        uWidths: { value: new THREE.Vector2(...bake.widths) },
        uFactor: { value: factor },
        uColor: { value: new THREE.Color() },
        uTone: { value: 0 },
        uToneScale: { value: 12 },
        uGrain: { value: 0 },
        uShine: { value: 0 },
        uSeed: { value: 0 },
        uStampColors: { value: 0 },
        uPaintColor: { value: -1 },
        uSource: { value: bake.source ?? none },
        uFromSource: { value: 0 },
        uEmitPass: { value: 0 },
        uGlow: { value: 0 },
        uL: { value: new THREE.Vector3() },
        uL2: { value: new THREE.Vector3() },
        uBoth: { value: 1 },
        uOpacity: { value: 1 },
        uBlend: { value: 0 },
        uMaskOnly: { value: 0 },
        uCount: { value: 0 },
        uKind: { value: new Array(maxMasks).fill(0) },
        uA: { value: vec4s(maxMasks) },
        uB: { value: vec4s(maxMasks) },
        uC: { value: vec4s(maxMasks) },
        uV: { value: Array.from({ length: maxMasks }, () => new THREE.Matrix4()) },
        uMirrorX: { value: model.extent[0] / 2 },
        uPaint0: { value: null },
        uPaint1: { value: null },
        uPaint2: { value: null },
        uPaint3: { value: null },
        uGrunge0: { value: null },
        uGrunge1: { value: null },
        uGrunge2: { value: null },
        uGrunge3: { value: null },
        uAtlas: { value: this.atlas.texture },
        uAtlasGrid: { value: atlasGrid },
        uAtlasInset: { value: atlasInset },
        uSC: { value: vec4s(maxStamps) },
        uST: { value: vec4s(maxStamps) },
        uSN: { value: vec4s(maxStamps) },
      },
    })
    this.finishMat = new THREE.ShaderMaterial({
      glslVersion: THREE.GLSL3,
      vertexShader: quadVertex,
      fragmentShader: finishFragment,
      uniforms: {
        uSrc: { value: null },
        uPalette: { value: this.palette },
        uColors: { value: 0 },
        uDither: { value: 0 },
      },
    })

    this.atlas.onLoad = () => this.onLoad()

    this.quad = new THREE.Mesh(new THREE.PlaneGeometry(2, 2), this.layerMat)
    this.quad.frustumCulled = false
    this.scene.add(this.quad)
  }

  // The x of the mirror plane, in model space.
  get mirrorX() {
    return this.model.min[0] + (this.model.extent[0] * this.model.diag) / 2
  }

  // A ghost stamp previews where the stamp tool would place one, on the selected layer.
  render(project: Project, selected: number, ghost?: Stamp | null) {
    if (ghost) project = withGhost(project, selected, ghost)
    const r = this.renderer
    const prevTarget = r.getRenderTarget()
    const u = this.layerMat.uniforms

    this.fillParts(project)
    const sh = project.shading
    const l = lightDir(sh.azimuth, sh.elevation)
    u.uL.value.copy(l)
    u.uL2.value.copy(lightDir(sh.azimuth + 80, 30))
    u.uBoth.value = sh.bothSides ? 1 : 0

    let src = 0
    r.setRenderTarget(this.ping[0])
    r.setClearColor(0x000000, 1)
    r.clear()

    const hidden = new Set(project.groups.filter((g) => !g.visible).map((g) => g.id))
    const glowing = project.layers.some((l) => l.glow && l.visible && !hidden.has(l.group))
    let esrc = 0
    if (glowing) {
      this.ensureEmission()
      r.setRenderTarget(this.emitPing![0])
      r.clear()
    }

    project.layers.forEach((layer, i) => {
      this.setLayer(layer, i)

      if (i === selected) {
        u.uMaskOnly.value = 1
        this.draw(this.layerMat, this.mask)
        u.uMaskOnly.value = 0
      }
      if (!layer.visible || hidden.has(layer.group)) return

      u.uPrev.value = this.ping[src].texture
      this.draw(this.layerMat, this.ping[1 - src])
      src = 1 - src

      if (glowing && (layer.glow || layer.blend === 'normal')) {
        const e = this.emitPing!
        u.uEmitPass.value = 1
        u.uGlow.value = layer.glow ? 1 : 0
        u.uPrev.value = e[esrc].texture
        this.draw(this.layerMat, e[1 - esrc])
        u.uEmitPass.value = 0
        esrc = 1 - esrc
      }
    })

    this.layered = src
    if (sh.on) {
      const lu = this.lightMat.uniforms
      lu.uSrc.value = this.ping[src].texture
      lu.uL.value.copy(l)
      lu.uBoth.value = sh.bothSides ? 1 : 0
      lu.uAoFloor.value = sh.aoFloor
      lu.uFormFloor.value = sh.formFloor
      lu.uKey.value = sh.key
      lu.uKeyColor.value.set(sh.keyColor).convertLinearToSRGB()
      lu.uUnder.value = sh.under
      lu.uUnderColor.value.set(sh.underColor).convertLinearToSRGB()
      lu.uRim.value = sh.rim
      this.draw(this.lightMat, this.ping[1 - src])
      src = 1 - src
    }

    this.reduceMat.uniforms.uSrc.value = this.ping[src].texture
    this.reduceMat.uniforms.uSharpen.value = project.finish.sharpen
    this.draw(this.reduceMat, this.reduced)

    this.hasEmission = glowing
    if (glowing) {
      this.reduceMat.uniforms.uSrc.value = this.emitPing![esrc].texture
      this.reduceMat.uniforms.uSharpen.value = 0
      this.draw(this.reduceMat, this.emission!)
    }
    this.finish(project)

    r.setRenderTarget(prevTarget)
  }

  hasEmission = false
  // The ping target holding the layers before baked light, for sampling colors.
  private layered = 0

  // The layers' color around a texel of the bake, as a hex color.
  colorAt(texel: [number, number]): string {
    const prev = this.renderer.getRenderTarget()
    const size = this.bake.size
    const x = Math.min(Math.max(texel[0] - 1, 0), size - 3)
    const y = Math.min(Math.max(texel[1] - 1, 0), size - 3)
    const px = new Uint8Array(9 * 4)
    this.renderer.readRenderTargetPixels(this.ping[this.layered], x, y, 3, 3, px)
    this.renderer.setRenderTarget(prev)
    const sum = [0, 0, 0]
    for (let i = 0; i < 9; i++) for (let c = 0; c < 3; c++) sum[c] += px[i * 4 + c]
    return '#' + sum.map((v) => Math.round(v / 9).toString(16).padStart(2, '0')).join('')
  }

  private ensureEmission() {
    if (this.emitPing) return
    const size = this.bake.size
    this.emitPing = [target(size), target(size)]
    this.emission = target(size / this.factor, true)
  }

  // Dither and palette only, for when just the palette changed.
  finish(project: Project) {
    const prev = this.renderer.getRenderTarget()
    const f = this.finishMat.uniforms
    f.uSrc.value = this.reduced.texture
    f.uColors.value = this.fillPalette(project.finish.palette === 'auto' ? this.autoColors : project.finish.colors)
    f.uDither.value = project.finish.dither
    this.draw(this.finishMat, this.out)
    this.renderer.setRenderTarget(prev)
  }

  // Picks the auto palette from the texture before quantizing, from a sample of its
  // texels. Later picks refine the last one, so colors stay put while editing.
  // Pinned colors, like painted ones, keep their own entries however few texels use them.
  pickPalette(count: number, pinned: RGB[] = []) {
    pinned = pinned.slice(0, count - 1)
    const size = this.reduced.width
    const data = new Uint8Array(size * size * 4)
    this.renderer.readRenderTargetPixels(this.reduced, 0, 0, size, size, data)

    const step = Math.max(1, Math.floor((size * size) / 32000))
    const pixels: RGB[] = []
    for (let i = 0; i < size * size; i += step) pixels.push([data[i * 4], data[i * 4 + 1], data[i * 4 + 2]])
    const rest = this.autoColors.slice(this.autoPinned)
    const fresh = count !== this.autoCount || rest.length < count - pinned.length
    const start = fresh ? medianCut(pixels, count - pinned.length) : rest.slice(0, count - pinned.length)
    this.autoColors = refine([...pinned, ...start], pixels, fresh ? 10 : 6, pinned.length)
    this.autoCount = count
    this.autoPinned = pinned.length
  }

  private draw(material: THREE.ShaderMaterial, to: THREE.WebGLRenderTarget) {
    this.quad.material = material
    material.uniformsNeedUpdate = true
    this.renderer.setRenderTarget(to)
    this.renderer.render(this.scene, this.camera)
  }

  // Draws one layer's mask into the mask target, for a layer the list is hovering.
  renderMask(project: Project, index: number) {
    const layer = project.layers[index]
    if (!layer) return
    const prev = this.renderer.getRenderTarget()
    this.fillParts(project)
    this.setLayer(layer, index)
    const u = this.layerMat.uniforms
    u.uMaskOnly.value = 1
    this.draw(this.layerMat, this.mask)
    u.uMaskOnly.value = 0
    this.renderer.setRenderTarget(prev)
  }

  // The topmost shown layer covering a texel, for picking layers on the model.
  layerAt(project: Project, texel: [number, number]): number {
    const prev = this.renderer.getRenderTarget()
    const hidden = new Set(project.groups.filter((g) => !g.visible).map((g) => g.id))
    const px = new Uint8Array(4)
    const u = this.layerMat.uniforms
    this.fillParts(project)
    let found = -1
    for (let i = project.layers.length - 1; i >= 0 && found < 0; i--) {
      const layer = project.layers[i]
      if (!layer.visible || hidden.has(layer.group) || layer.opacity < 0.2) continue
      this.setLayer(layer, i)
      u.uMaskOnly.value = 1
      this.draw(this.layerMat, this.mask)
      this.renderer.readRenderTargetPixels(this.mask, texel[0], texel[1], 1, 1, px)
      if (px[0] > 128) found = i
    }
    u.uMaskOnly.value = 0
    this.renderer.setRenderTarget(prev)
    return found
  }

  private setLayer(layer: Layer, index: number) {
    const u = this.layerMat.uniforms
    u.uColor.value.set(layer.color).convertLinearToSRGB()
    u.uTone.value = layer.tone
    u.uToneScale.value = layer.toneScale
    u.uGrain.value = layer.grain
    u.uShine.value = layer.shine
    u.uSeed.value = (index * 7.31) % 97
    u.uStampColors.value = layer.stampColors ? 1 : 0
    const colorMask = layer.paintColors ? layer.masks.slice(0, maxMasks).findIndex((m) => m.kind === 'paint') : -1
    u.uPaintColor.value = colorMask
    if (colorMask >= 0) this.paint.colored.add(layer.masks[colorMask].paint)
    u.uFromSource.value = layer.source && this.bake.source ? 1 : 0
    u.uOpacity.value = layer.opacity
    u.uBlend.value = blendIndex[layer.blend]
    u.uZone.value = this.zones.get(layer.group) ?? -1
    const masks = layer.masks.slice(0, maxMasks)
    let stamps = 0
    u.uCount.value = masks.length
    masks.forEach((m, i) => {
      u.uKind.value[i] = kindIndex[m.kind]
      u.uA.value[i].set(m.amount, m.soft, m.rough, m.scale)
      u.uB.value[i].set(m.axis, m.seed, m.invert ? 1 : 0, index * maxMasks + i)
      u['uPaint' + i].value = m.kind === 'paint' ? this.paint.get(m.paint) : null
      u['uGrunge' + i].value = m.kind === 'grunge' ? this.grunge(m.map) : null

      const list = m.kind === 'stamps' ? expand(m.stamps, this.mirrorX).slice(0, maxStamps - stamps) : []
      u.uC.value[i].set(stamps, list.length, m.kind === 'volume' ? shapeIndex[m.volume.shape] : m.crisp, m.volume.mirror ? 1 : 0)
      for (const s of list) this.setStamp(stamps++, s)
      if (m.kind === 'volume') u.uV.value[i].copy(this.volumeMatrix(m.volume))
    })
  }

  grunge(name: string): THREE.Texture {
    let t = this.grunges.get(name)
    if (!t) {
      const texture = new THREE.Texture()
      texture.wrapS = texture.wrapT = THREE.RepeatWrapping
      texture.colorSpace = THREE.NoColorSpace
      track(
        imageURL(`/grunge?name=${encodeURIComponent(name)}`).then(
          (url) =>
            new Promise<void>((resolve) => {
              if (!url) return resolve()
              const img = new Image()
              img.onload = img.onerror = () => {
                texture.image = img
                texture.needsUpdate = true
                this.onLoad()
                resolve()
              }
              img.src = url
            }),
        ),
      )
      this.grunges.set(name, texture)
      t = texture
    }
    return t
  }

  // Takes the bake's positions, (p - min) / diagonal, into the volume's unit shape.
  private volumeMatrix(v: Volume): THREE.Matrix4 {
    const { min, diag } = this.model
    const size = new THREE.Vector3(...v.size).max(new THREE.Vector3(1e-6, 1e-6, 1e-6))
    const world = new THREE.Matrix4().compose(new THREE.Vector3(...v.p), new THREE.Quaternion(...v.q), size)
    const fromBake = new THREE.Matrix4().makeTranslation(min[0], min[1], min[2]).multiply(new THREE.Matrix4().makeScale(diag, diag, diag))
    return world.invert().multiply(fromBake)
  }

  private setStamp(j: number, s: Stamp) {
    const u = this.layerMat.uniforms
    const { min, diag } = this.model
    const f = frame(s)
    const c = f.c.sub(new THREE.Vector3(...min)).divideScalar(diag)
    u.uSC.value[j].set(c.x, c.y, c.z, f.hw / diag)
    u.uST.value[j].set(f.t.x, f.t.y, f.t.z, f.hh / diag)
    u.uSN.value[j].set(f.n.x, f.n.y, f.n.z, this.atlas.cell(s.shape))
  }

  // Rows per layer mask, then per folder zone: 1 for every part or island selected.
  private fillParts(project: Project) {
    const parts = this.parts.image.data as Uint8Array
    const islands = this.islands.image.data as Uint8Array
    parts.fill(0)
    islands.fill(0)
    project.layers.slice(0, maskRows / maxMasks).forEach((layer, li) => {
      layer.masks.slice(0, maxMasks).forEach((m, mi) => {
        const row = li * maxMasks + mi
        for (const p of m.parts) if (p >= 0 && p < 256) parts[row * 256 + p] = 255
        for (const p of m.islands) if (p >= 0 && p < maxIslands) islands[row * maxIslands + p] = 255
      })
    })

    this.zones.clear()
    project.groups.slice(0, maxGroups).forEach((g, i) => {
      if (!g.zone.parts.length && !g.zone.islands.length) return
      const row = maskRows + i
      this.zones.set(g.id, row)
      for (const p of g.zone.parts) if (p >= 0 && p < 256) parts[row * 256 + p] = 255
      for (const p of g.zone.islands) if (p >= 0 && p < maxIslands) islands[row * maxIslands + p] = 255
    })
    this.parts.needsUpdate = true
    this.islands.needsUpdate = true
  }

  private fillPalette(list: RGB[]): number {
    const colors = list.slice(0, 256)
    const data = this.palette.image.data as Uint8Array
    colors.forEach(([r, g, b], i) => data.set([r, g, b, 255], i * 4))
    this.palette.needsUpdate = true
    return colors.length
  }

  // The finished texture as a base64 PNG, top row first like glTF UVs.
  png(from: THREE.WebGLRenderTarget = this.out): string {
    const size = from.width
    const pixels = new Uint8Array(size * size * 4)
    this.renderer.readRenderTargetPixels(from, 0, 0, size, size, pixels)

    const canvas = document.createElement('canvas')
    canvas.width = canvas.height = size
    const ctx = canvas.getContext('2d')!
    ctx.putImageData(new ImageData(new Uint8ClampedArray(pixels.buffer), size, size), 0, 0)
    return canvas.toDataURL('image/png').split(',')[1]
  }

  dispose() {
    for (const t of [this.out, this.reduced, this.mask, ...this.ping, ...(this.emitPing ?? []), this.emission]) t?.dispose()
    this.layerMat.dispose()
    this.lightMat.dispose()
    this.reduceMat.dispose()
    this.finishMat.dispose()
    this.parts.dispose()
    this.islands.dispose()
    this.atlas.dispose()
    for (const t of this.grunges.values()) t.dispose()
    this.palette.dispose()
    this.quad.geometry.dispose()
  }
}

function withGhost(project: Project, selected: number, ghost: Stamp): Project {
  const layers = project.layers.slice()
  const layer = layers[selected]
  if (!layer) return project

  const masks = layer.masks.slice()
  const at = masks.findIndex((m) => m.kind === 'stamps')
  if (at >= 0) masks[at] = { ...masks[at], stamps: [...masks[at].stamps, ghost] }
  else if (masks.length < maxMasks) masks.push({ ...newMask('stamps'), stamps: [ghost] })
  layers[selected] = { ...layer, masks, visible: true }
  return { ...project, layers }
}
