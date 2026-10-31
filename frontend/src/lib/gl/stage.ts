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
import { Torch, torchDeclarations } from './torch'

export type Hit = { point: THREE.Vector3; normal: THREE.Vector3; part: number }

// Our textures hold display colors; materials decode them so lighting works in linear.
const decode = /* glsl */ `
uniform sampler2D uPosMap;
uniform float uHover;
uniform sampler2D uEmitMap;
uniform float uEmit;
vec3 srgbDecode(vec3 c) {
  return mix(c / 12.92, pow((c + 0.055) / 1.055, vec3(2.4)), step(0.04045, c));
}
`

const patch = /* glsl */ `
#include <map_fragment>
diffuseColor.rgb = srgbDecode(diffuseColor.rgb);
if (uHover >= 0.0 && abs(texture(uPosMap, vMapUv).w - uHover) < 0.5) diffuseColor.rgb = mix(diffuseColor.rgb, vec3(0.9, 1.0, 0.95), 0.18);
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

  private hemi = new THREE.HemisphereLight(0xfff1e0, 0x2c3038, 1)
  private key = new THREE.DirectionalLight(0xfff6ea, 2)
  private head = new THREE.DirectionalLight(0xffffff, 0.6)

  private root: THREE.Object3D | undefined
  private box = new THREE.Box3()
  private textured: THREE.Mesh[] = []
  private lit: THREE.MeshLambertMaterial
  private unlit: THREE.MeshBasicMaterial
  private gameMat: THREE.MeshBasicMaterial
  private shared = {
    uPosMap: { value: null as THREE.Texture | null },
    uHover: { value: -1 },
    uEmitMap: { value: null as THREE.Texture | null },
    uEmit: { value: 0 },
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

  posData: Float32Array | undefined
  posSize = 0

  constructor(private container: HTMLElement) {
    const renderer = new THREE.WebGLRenderer({ antialias: true })
    renderer.setPixelRatio(window.devicePixelRatio)
    container.appendChild(renderer.domElement)
    this.renderer = renderer

    this.scene.add(this.hemi, this.key, this.key.target, this.camera, this.torch.light, this.torch.light.target)
    // The headlight sits a little above the eye, so faces don't go flat.
    this.head.position.set(0, 0.4, 0)
    this.head.target.position.set(0, 0, -1)
    this.camera.add(this.head, this.head.target)

    this.ring.visible = this.outline.visible = false
    this.ring.renderOrder = this.outline.renderOrder = 10
    this.scene.add(this.ring, this.outline)

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

    this.resize = new ResizeObserver(() => this.applySize())
    this.resize.observe(container)
  }

  private applySize() {
    const { clientWidth: w, clientHeight: h } = this.container
    const ratio = this.lines ? this.lines / Math.max(h, 1) : window.devicePixelRatio
    this.renderer.setPixelRatio(ratio)
    this.renderer.setSize(w, h)
    this.composer.setPixelRatio(ratio)
    this.composer.setSize(w, h)
    this.renderer.domElement.style.imageRendering = this.lines ? 'pixelated' : ''
    this.camera.aspect = w / h
    this.camera.updateProjectionMatrix()
  }

  setResolution(lines: number) {
    this.lines = lines
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

  async load(url: string) {
    const gltf = await new GLTFLoader().loadAsync(url)
    this.clear()

    const meshes: THREE.Mesh[] = []
    gltf.scene.traverse((o) => {
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
      } else if (glow) {
        // Above 1, so only these reach the bloom threshold.
        const color = mat.emissive.clone().multiplyScalar(Math.max(mat.emissiveIntensity, 1) * 2.5)
        mesh.material = new THREE.MeshBasicMaterial({ color })
      } else {
        mesh.material = new THREE.MeshLambertMaterial({ color: mat.color })
      }
    }

    this.root = gltf.scene
    this.scene.add(gltf.scene)
    this.box.setFromObject(gltf.scene)
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

  setBloom(on: boolean) {
    this.bloom.enabled = on
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

  get cameraUp() {
    return new THREE.Vector3(0, 1, 0).applyQuaternion(this.camera.quaternion)
  }

  get cameraPosition() {
    return this.camera.position.clone()
  }

  setEmission(texture: THREE.Texture | null, strength: number) {
    this.shared.uEmitMap.value = texture
    this.shared.uEmit.value = texture ? strength : 0
  }

  setHover(part: number) {
    this.shared.uHover.value = part
  }

  setBackground(css: string) {
    const bg = new THREE.Color(css)
    this.background = bg
    if (!this.game) this.scene.background = bg
    this.gridColor = bg.clone().offsetHSL(0, 0, 0.3)
    if (this.grid) (this.grid.material as THREE.LineBasicMaterial).color.copy(this.gridColor)
  }

  hit(x: number, y: number): Hit | undefined {
    if (!this.root) return
    const rect = this.renderer.domElement.getBoundingClientRect()
    const ndc = new THREE.Vector2(((x - rect.left) / rect.width) * 2 - 1, -((y - rect.top) / rect.height) * 2 + 1)
    this.raycaster.setFromCamera(ndc, this.camera)
    const hit = this.raycaster.intersectObject(this.root, true)[0]
    if (!hit) return

    let part = -1
    if (hit.uv && this.posData && this.textured.includes(hit.object as THREE.Mesh)) {
      const s = this.posSize
      const px = Math.min(s - 1, Math.max(0, Math.floor(hit.uv.x * s)))
      const py = Math.min(s - 1, Math.max(0, Math.floor(hit.uv.y * s)))
      part = this.posData[(py * s + px) * 4 + 3]
    }
    const normal = hit.face ? hit.face.normal.clone().transformDirection(hit.object.matrixWorld) : new THREE.Vector3(0, 1, 0)
    return { point: hit.point, normal, part }
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
    return { point: hit.point, normal, part: -1 }
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
    this.composer.render()
  }

  dispose() {
    this.resize.disconnect()
    this.controls.dispose()
    this.composer.dispose()
    this.renderer.dispose()
    this.renderer.domElement.remove()
  }
}
