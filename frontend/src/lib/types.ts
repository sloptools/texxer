export type MaskKind = 'parts' | 'paint' | 'stamps' | 'grunge' | 'ao' | 'edges' | 'cavity' | 'gradient' | 'facing' | 'noise'
export type Blend = 'normal' | 'multiply' | 'add' | 'overlay' | 'relief'
export type Axis = 0 | 1 | 2

export type Vec3 = [number, number, number]

// A projected decal shape, in model space. It lies on the plane through p with
// normal n; up is its unrotated up direction, rot turns it around n in degrees.
export type Stamp = {
  shape: string
  p: Vec3
  n: Vec3
  up: Vec3
  rot: number
  size: number
  aspect: number
  mirror: boolean
}

// Every mask is a field over the model, cut at 1 - amount. Parts masks are on or off.
export type Mask = {
  kind: MaskKind
  invert: boolean
  amount: number
  soft: number
  // Noise that roughens the cut, at the given scale; noise masks use scale as their frequency.
  rough: number
  scale: number
  axis: Axis
  seed: number
  parts: number[]
  // Painted masks keep their pixels in a PNG under this id.
  paint: string
  stamps: Stamp[]
  // A built-in grunge map, projected from three sides at `scale` repeats per model size.
  map: string
}

// A folder for layers. Its layers sit next to each other in the stack and
// point to it by id.
export type Group = {
  id: string
  name: string
  open: boolean
  visible: boolean
}

export type Layer = {
  id: string
  name: string
  // The folder, empty for none.
  group: string
  color: string
  // Lighter and darker patches over the color, at a world scale; grain varies single texels.
  tone: number
  toneScale: number
  grain: number
  // Baked sheen and glints from the light direction, for glass and gloss.
  shine: number
  // Take the color from image stamps instead of the layer color.
  stampColors: boolean
  // Glows: the layer's color also goes into the emissive map.
  glow: boolean
  opacity: number
  blend: Blend
  visible: boolean
  masks: Mask[]
}

export type Finish = {
  // Rendered at this many times the size, then box-filtered down and sharpened.
  supersample: number
  sharpen: number
  // auto picks `count` colors from the texture itself.
  palette: 'none' | 'auto' | 'quake' | 'file'
  count: number
  colors: [number, number, number][]
  dither: number
  // Emissive strength in the viewport and the export; above 1 it blooms.
  glow: number
}

// Light baked into the texture, before the colors are crunched.
export type Shading = {
  on: boolean
  azimuth: number
  elevation: number
  key: number
  keyColor: string
  aoFloor: number
  formFloor: number
  under: number
  underColor: string
  rim: number
  bothSides: boolean
}

export type Light = {
  key: number
  ambient: number
  headlight: number
  // Key light angle around the model, in degrees.
  angle: number
  // The game view: ambient color and energy like Godot's, plus a light from above.
  gameAmbient: string
  gameEnergy: number
  gameTop: number
  torch: number
}

// What a variant changes about a layer.
export type LayerLook = { color?: string; visible?: boolean; opacity?: number }

// A variant of the texture, like a player color: the base project with some layers
// looking different. The export writes a texture per variant.
export type Variant = { id: string; name: string; layers: Record<string, LayerLook> }

export type Project = {
  version: 1
  size: number
  // The texture size texxer laid the UVs out for; 0 paints on the model's own UVs.
  unwrap: number
  layers: Layer[]
  groups: Group[]
  variants: Variant[]
  finish: Finish
  shading: Shading
  light: Light
}

export type View = 'lit' | 'unlit' | 'game' | 'mask'
export type FlatMap = 'texture' | 'mask' | 'glow' | 'ao' | 'edges' | 'cavity' | 'parts'

export type Tool = 'nav' | 'brush' | 'stamp'

export const maskKinds: MaskKind[] = ['parts', 'paint', 'stamps', 'grunge', 'ao', 'edges', 'cavity', 'gradient', 'facing', 'noise']
export const blends: Blend[] = ['normal', 'multiply', 'add', 'overlay', 'relief']
