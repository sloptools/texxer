import * as THREE from 'three'
import { GLTFLoader } from 'three/examples/jsm/loaders/GLTFLoader.js'
import { EffectComposer } from 'three/examples/jsm/postprocessing/EffectComposer.js'
import { OutputPass } from 'three/examples/jsm/postprocessing/OutputPass.js'
import { RenderPass } from 'three/examples/jsm/postprocessing/RenderPass.js'
import { UnrealBloomPass } from 'three/examples/jsm/postprocessing/UnrealBloomPass.js'
import { ShaderPass } from 'three/examples/jsm/postprocessing/ShaderPass.js'
import type { Light, View } from '../types'
import { TrackpadControls } from './controls'
import { game, gameDeclarations, gamePatch, gameUniforms, gradePass } from './gamelook'
import { Flat } from './flat'
import { Torch, torchDeclarations } from './torch'
import { VolumeGizmo } from './volume'

export type Hit = { point: THREE.Vector3; normal: THREE.Vector3; part: number; island: number; uv?: THREE.Vector2 }

// Our textures hold display colors; materials decode them so lighting works in linear.
const decode = /* glsl */ `
uniform sampler2D uPosMap;
uniform float uHover;
uniform float uHoverIsland;
uniform sampler2D uEmitMap;
uniform float uEmit;
uniform sampler2D uMaskMap;
uniform float uMaskTint;
vec3 srgbDecode(vec3 c) {
  return mix(c / 12.92, pow((c + 0.055) / 1.055, vec3(2.4)), step(0.04045, c));
}
`

const patch = /* glsl */ `
#include <map_fragment>
diffuseColor.rgb = srgbDecode(diffuseColor.rgb);
if (uHover >= 0.0 && abs(texture(uPosMap, vMapUv).w - uHover) < 0.5) diffuseColor.rgb = mix(diffuseColor.rgb, vec3(0.9, 1.0, 0.95), 0.18);
if (uMaskTint > 0.0) diffuseColor.rgb = mix(diffuseColor.rgb, vec3(0.21, 0.55, 0.43), texture(uMaskMap, vMapUv).r * uMaskTint);
if (uHoverIsland >= 0.0 && abs(texture(uNrmMap, vMapUv).w - uHoverIsland) < 0.5) diffuseColor.rgb = mix(diffuseColor.rgb, vec3(0.9, 1.0, 0.95), 0.18);
`

// Glow from the emissive map: the lit material adds it as emission, the
// others, which aren't lit by three, add it to their color after their light.
const emitLambert = /* glsl */ `
#include <emissivemap_fragment>
totalEmissiveRadiance += srgbDecode(texture(uEmitMap, vMapUv).rgb) * uEmit;
`
const emitBasic = /* glsl */ `
#ifndef LAMBERT
diffuseColor.rgb += srgbDecode(texture(uEmitMap, vMapUv).rgb) * uEmit;
#endif
`

// The 3D view: model, lights, bloom for emissive parts, and trackpad camera.
export class Stage {
  readonly renderer: THREE.WebGLRenderer
  readonly controls: TrackpadControls
  private scene = new THREE.Scene()
  private camera = new THREE.PerspectiveCamera(35, 1, 0.01, 100)
  private composer: EffectComposer
  private bloom: UnrealBloomPass
  private grade: ShaderPass
  private game = false
  private background = new THREE.Color()
  private grid: THREE.GridHelper | undefined
  private gridColor = new THREE.Color(0x555555)
  private showGrid = false
  private radius = 1
  // Vertical resolution to render at, like a game would; 0 renders at full size.
  private lines = 0
  readonly torch = new Torch()
  readonly flat: Flat
  readonly volume: VolumeGizmo

  private hemi = new THREE.HemisphereLight(0xfff1e0, 0x2c3038, 1)
  private key = new THREE.DirectionalLight(0xfff6ea, 2)
  private head = new THREE.DirectionalLight(0xffffff, 0.6)

  private root: THREE.Object3D | undefined
  private box = new THREE.Box3()
  private textured: THREE.Mesh[] = []
  private lit: THREE.MeshLambertMaterial
  private unlit: THREE.MeshBasicMaterial
  private gameMat: THREE.MeshBasicMaterial
  // Until the layers are rendered, instead of the model's own texture.
  private placeholder = new THREE.MeshLambertMaterial({ color: 0x77756e })
  private shared = {
    uPosMap: { value: null as THREE.Texture | null },
    uHover: { value: -1 },
    uHoverIsland: { value: -1 },
    uEmitMap: { value: null as THREE.Texture | null },
    uEmit: { value: 0 },
    uMaskMap: { value: null as THREE.Texture | null },
    uMaskTint: { value: 0 },
  }
  private gameShared = gameUniforms()
  private raycaster = new THREE.Raycaster()
  private timer = new THREE.Timer()
  private resize: ResizeObserver
  private snap = true

  private ring = new THREE.Mesh(
    new THREE.RingGeometry(0.94, 1, 48),
    new THREE.MeshBasicMaterial({ color: 0xffffff, transparent: true, opacity: 0.85, depthTest: false, side: THREE.DoubleSide }),
  )
  private outline = new THREE.LineLoop(
    new THREE.BufferGeometry().setFromPoints([new THREE.Vector3(), new THREE.Vector3(), new THREE.Vector3(), new THREE.Vector3()]),
    new THREE.LineBasicMaterial({ color: 0x7fc4af, depthTest: false, transparent: true }),
  )

  onError: (message: string) => void = () => {}
  // Grips of the picked stamp, square pixels the same size at any distance.
  private grips = new THREE.Points(
    new THREE.BufferGeometry().setFromPoints([new THREE.Vector3(), new THREE.Vector3()]),
    new THREE.PointsMaterial({ color: 0x7fc4af, size: 9, sizeAttenuation: false, depthTest: false, transparent: true }),
  )
  posData: Float32Array | undefined
  nrmData: Float32Array | undefined
  posSize = 0

  constructor(private container: HTMLElement) {
    const renderer = new THREE.WebGLRenderer({ antialias: true })
    renderer.debug.onShaderError = (gl, program, _vs, fs) => {
      const log = gl.getProgramInfoLog(program) || gl.getShaderInfoLog(fs) || 'unknown error'
      console.error('shader error', log)
      this.onError('a shader failed: ' + log.trim().split('\n')[0])
    }
    renderer.setPixelRatio(window.devicePixelRatio)
    container.appendChild(renderer.domElement)
    this.renderer = renderer

    this.scene.add(this.hemi, this.key, this.key.target, this.camera, this.torch.light, this.torch.light.target)
    // The headlight sits a little above the eye, so faces don't go flat.
    this.head.position.set(0, 0.4, 0)
    this.head.target.position.set(0, 0, -1)
    this.camera.add(this.head, this.head.target)

    this.ring.visible = this.outline.visible = this.grips.visible = false
    this.ring.renderOrder = this.outline.renderOrder = 10
    this.grips.renderOrder = 11
    this.scene.add(this.ring, this.outline, this.grips)

    this.lit = this.material(new THREE.MeshLambertMaterial())
    this.unlit = this.material(new THREE.MeshBasicMaterial())
    this.gameMat = this.material(new THREE.MeshBasicMaterial(), true)

    this.composer = new EffectComposer(renderer)
    this.composer.addPass(new RenderPass(this.scene, this.camera))
    this.bloom = new UnrealBloomPass(new THREE.Vector2(256, 256), 0.5, 0.4, 1)
    this.composer.addPass(this.bloom)
    this.composer.addPass(new OutputPass())
    this.grade = gradePass()
    this.composer.addPass(this.grade)

    this.controls = new TrackpadControls(this.camera, renderer.domElement, (x, y) => this.hit(x, y)?.point)
    this.flat = new Flat(renderer, renderer.domElement)
    this.volume = new VolumeGizmo(this.camera, renderer.domElement, this.scene)

    this.resize = new ResizeObserver(() => this.applySize())
    this.resize.observe(container)
  }

  private applySize() {
    const { clientWidth: w, clientHeight: h } = this.container
    const lines = this.flat.active ? 0 : this.lines
    const ratio = lines ? lines / Math.max(h, 1) : window.devicePixelRatio
    this.renderer.setPixelRatio(ratio)
    this.renderer.setSize(w, h)
    this.composer.setPixelRatio(ratio)
    this.composer.setSize(w, h)
    this.renderer.domElement.style.imageRendering = lines ? 'pixelated' : ''
    this.flat.resize(w, h)
    this.camera.aspect = w / h
    this.camera.updateProjectionMatrix()
  }

  setResolution(lines: number) {
    this.lines = lines
    this.applySize()
  }

  // Switches between the model and the flat texture; tools and the camera rest in 2D.
  setFlat(on: boolean) {
    if (this.flat.active === on) return
    this.flat.active = on
    this.controls.enabled = !on
    if (on) this.setRing(null)
    this.outline.visible &&= !on
    this.applySize()
  }

  setGrid(on: boolean) {
    this.showGrid = on
    if (this.grid) this.grid.visible = on
  }

  private material<T extends THREE.MeshLambertMaterial | THREE.MeshBasicMaterial>(m: T, game = false): T {
    m.onBeforeCompile = (s) => {
      Object.assign(s.uniforms, this.shared, this.gameShared, this.torch.uniforms)
      s.fragmentShader = (decode + gameDeclarations + torchDeclarations + s.fragmentShader)
        .replace('#include <map_fragment>', patch + gamePatch + emitBasic)
        .replace('#include <emissivemap_fragment>', emitLambert)
    }
    // A fixed define keeps the game material's program apart from the unlit one.
    if (game) m.defines = { GAME_VIEW: '' }
    return m
  }

  // Loads a model's files, each moved to its place beside the others.
  async load(files: { url: string; images: string; offset: number[] }[]) {
    const scenes = await Promise.all(files.map((f) => new GLTFLoader().setResourcePath(f.images).loadAsync(f.url)))
    this.clear()

    const root = new THREE.Group()
    scenes.forEach((gltf, i) => {
      gltf.scene.position.fromArray(files[i].offset)
      root.add(gltf.scene)
    })
    root.updateMatrixWorld(true)

    const meshes: THREE.Mesh[] = []
    root.traverse((o) => {
      if ((o as THREE.Mesh).isMesh) meshes.push(o as THREE.Mesh)
    })

    // Same rule as the export: textured materials get ours, or every non-emissive one if none is textured.
    const std = (m: THREE.Mesh) => m.material as THREE.MeshStandardMaterial
    const anyTextured = meshes.some((m) => std(m).map)
    for (const mesh of meshes) {
      const mat = std(mesh)
      const glow = mat.emissive && mat.emissive.getHex() !== 0 && !mat.emissiveMap
      if (anyTextured ? mat.map : !glow) {
        this.textured.push(mesh)
        mesh.material = this.placeholder
        mat.map?.dispose()
      } else if (glow) {
        // Above 1, so only these reach the bloom threshold.
        const color = mat.emissive.clone().multiplyScalar(Math.max(mat.emissiveIntensity, 1) * 2.5)
        mesh.material = new THREE.MeshBasicMaterial({ color })
      } else {
        mesh.material = new THREE.MeshLambertMaterial({ color: mat.color })
      }
    }

    this.flat.setMeshes(this.textured)
    this.root = root
    this.scene.add(root)
    this.box.setFromObject(root)
    this.radius = this.box.getBoundingSphere(new THREE.Sphere()).radius
    this.torch.uniforms.uMin.value.copy(this.box.min)
    this.torch.uniforms.uDiag.value = this.box.getSize(new THREE.Vector3()).length()
    this.controls.frame(this.box)
    this.snap = true

    const size = this.box.getSize(new THREE.Vector3())
    const span = Math.max(size.x, size.z) * 1.6
    const center = this.box.getCenter(new THREE.Vector3())
    this.grid = new THREE.GridHelper(span, 16, 0xffffff, 0xffffff)
    this.grid.position.set(center.x, this.box.min.y, center.z)
    const gm = this.grid.material as THREE.LineBasicMaterial
    gm.color.copy(this.gridColor)
    gm.transparent = true
    gm.opacity = 0.35
    this.grid.visible = this.showGrid
    this.scene.add(this.grid)
  }

  private clear() {
    if (this.root) this.scene.remove(this.root)
    if (this.grid) this.scene.remove(this.grid)
    this.root = undefined
    this.textured = []
  }

  setTexture(texture: THREE.Texture, maps: { pos: THREE.Texture; geo: THREE.Texture; nrm: THREE.Texture }, view: View) {
    this.shared.uPosMap.value = maps.pos
    this.gameShared.uGeoMap.value = maps.geo
    this.gameShared.uNrmMap.value = maps.nrm
    for (const m of [this.lit, this.unlit, this.gameMat]) {
      m.map = texture
      m.needsUpdate = true
    }
    const material = view === 'lit' ? this.lit : view === 'game' ? this.gameMat : this.unlit
    for (const mesh of this.textured) mesh.material = material
    this.setGame(view === 'game')
  }

  // The game view also takes Substratum's tone mapping, grade and dark water background.
  private setGame(on: boolean) {
    this.game = on
    this.gameShared.uGame.value = on ? 1 : 0
    this.renderer.toneMapping = on ? THREE.ACESFilmicToneMapping : THREE.NoToneMapping
    this.renderer.toneMappingExposure = on ? game.exposure : 1
    this.grade.enabled = on
    this.scene.background = on ? new THREE.Color(game.background) : this.background
  }

  setLight(l: Light) {
    // Godot takes the ambient color as sRGB and lights in linear.
    this.gameShared.uAmbient.value.set(l.gameAmbient).multiplyScalar(l.gameEnergy)
    this.gameShared.uTop.value.setRGB(1, 0.95, 0.88).multiplyScalar(l.gameTop)

    this.key.intensity = l.key
    this.hemi.intensity = l.ambient
    this.head.intensity = l.headlight

    const center = this.box.isEmpty() ? new THREE.Vector3() : this.box.getCenter(new THREE.Vector3())
    const a = THREE.MathUtils.degToRad(l.angle)
    this.key.position.set(Math.sin(a), 1.4, Math.cos(a)).add(center)
    this.key.target.position.copy(center)
  }

  setBloom(on: boolean, strength: number) {
    this.bloom.enabled = on && strength > 0
    this.bloom.strength = strength
  }

  // The brush cursor: a ring lying on the surface.
  setRing(at: { point: THREE.Vector3; normal: THREE.Vector3; radius: number } | null) {
    this.ring.visible = !!at
    if (!at) return
    this.ring.position.copy(at.point)
    this.ring.lookAt(at.point.clone().add(at.normal))
    this.ring.scale.setScalar(at.radius)
  }

  // A stamp's frame, for lining it up.
  setOutline(points: THREE.Vector3[] | null, color?: string) {
    this.outline.visible = !!points
    if (!points) return
    this.outline.geometry.setFromPoints(points)
    if (color) (this.outline.material as THREE.LineBasicMaterial).color.set(color)
  }

  setGrips(points: THREE.Vector3[] | null) {
    this.grips.visible = !!points
    if (points) this.grips.geometry.setFromPoints(points)
  }

  // Where a point in the scene lands on screen, in client pixels.
  toScreen(p: THREE.Vector3): { x: number; y: number } {
    const rect = this.renderer.domElement.getBoundingClientRect()
    const v = p.clone().project(this.camera)
    return { x: rect.left + ((v.x + 1) / 2) * rect.width, y: rect.top + ((1 - v.y) / 2) * rect.height }
  }

  // Where the ray under a screen point meets the plane through p facing n.
  onPlane(x: number, y: number, p: THREE.Vector3, n: THREE.Vector3): THREE.Vector3 | undefined {
    const rect = this.renderer.domElement.getBoundingClientRect()
    const ndc = new THREE.Vector2(((x - rect.left) / rect.width) * 2 - 1, -((y - rect.top) / rect.height) * 2 + 1)
    this.raycaster.setFromCamera(ndc, this.camera)
    const plane = new THREE.Plane().setFromNormalAndCoplanarPoint(n, p)
    return this.raycaster.ray.intersectPlane(plane, new THREE.Vector3()) ?? undefined
  }

  get cameraUp() {
    return new THREE.Vector3(0, 1, 0).applyQuaternion(this.camera.quaternion)
  }

  get cameraQuaternion() {
    return this.camera.quaternion
  }

  get cameraPosition() {
    return this.camera.position.clone()
  }

  setEmission(texture: THREE.Texture | null, strength: number) {
    this.shared.uEmitMap.value = texture
    this.shared.uEmit.value = texture ? strength : 0
  }

  // Tints where a layer shows, for the layer the list is hovering; null clears it.
  setMaskTint(mask: THREE.Texture | null) {
    this.shared.uMaskMap.value = mask
    this.shared.uMaskTint.value = mask ? 0.6 : 0
  }

  setHover(part: number, island = -1) {
    this.shared.uHover.value = part
    this.shared.uHoverIsland.value = island
    this.flat.setHover(part)
  }

  setBackground(css: string) {
    const bg = new THREE.Color(css)
    this.background = bg
    if (!this.game) this.scene.background = bg
    this.flat.setBackground(bg)
    this.gridColor = bg.clone().offsetHSL(0, 0, 0.3)
    if (this.grid) (this.grid.material as THREE.LineBasicMaterial).color.copy(this.gridColor)
  }

  hit(x: number, y: number): Hit | undefined {
    if (!this.root || this.flat.active) return
    const rect = this.renderer.domElement.getBoundingClientRect()
    const ndc = new THREE.Vector2(((x - rect.left) / rect.width) * 2 - 1, -((y - rect.top) / rect.height) * 2 + 1)
    this.raycaster.setFromCamera(ndc, this.camera)
    const hit = this.raycaster.intersectObject(this.root, true)[0]
    if (!hit) return

    let part = -1
    let island = -1
    if (hit.uv && this.posData && this.textured.includes(hit.object as THREE.Mesh)) {
      const s = this.posSize
      const px = Math.min(s - 1, Math.max(0, Math.floor(hit.uv.x * s)))
      const py = Math.min(s - 1, Math.max(0, Math.floor(hit.uv.y * s)))
      part = this.posData[(py * s + px) * 4 + 3]
      island = this.nrmData?.[(py * s + px) * 4 + 3] ?? -1
    }
    const normal = hit.face ? hit.face.normal.clone().transformDirection(hit.object.matrixWorld) : new THREE.Vector3(0, 1, 0)
    return { point: hit.point, normal, part, island, uv: hit.uv?.clone() }
  }

  // Casts from above p back down along -n, so points off a curved surface land on it.
  surface(p: THREE.Vector3, n: THREE.Vector3, reach: number): Hit | undefined {
    if (!this.root) return
    const origin = p.clone().addScaledVector(n, reach)
    this.raycaster.set(origin, n.clone().negate())
    this.raycaster.far = reach * 2
    const hit = this.raycaster.intersectObject(this.root, true)[0]
    this.raycaster.far = Infinity
    if (!hit) return
    const normal = hit.face ? hit.face.normal.clone().transformDirection(hit.object.matrixWorld) : n.clone()
    return { point: hit.point, normal, part: -1, island: -1 }
  }

  frame() {
    if (!this.box.isEmpty()) this.controls.frame(this.box)
  }

  render() {
    this.timer.update()
    const dt = Math.min(this.timer.getDelta(), 0.1)
    this.controls.update(dt, this.snap)
    this.snap = false
    this.torch.update(this.camera, this.radius)
    if (this.flat.active) this.flat.render()
    else this.composer.render()
  }

  dispose() {
    this.resize.disconnect()
    this.controls.dispose()
    this.volume.dispose()
    this.flat.dispose()
    this.composer.dispose()
    this.renderer.dispose()
    this.renderer.domElement.remove()
  }
}
