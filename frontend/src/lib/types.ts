export type MaskKind = 'surfaces' | 'paint' | 'stamps' | 'volume' | 'grunge' | 'ao' | 'edges' | 'cavity' | 'gradient' | 'facing' | 'drips'
export type Blend = 'normal' | 'multiply' | 'add' | 'overlay' | 'relief'
// x, y, z as glTF has them (y up, z towards the front), then the same the other way.
export type Axis = 0 | 1 | 2 | 3 | 4 | 5

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
  // Only on this island, for stamps fitted to a surface.
  island?: number
}

export type VolumeShape = 'box' | 'cylinder' | 'sphere'

// A projection volume in model space: p its center, q its rotation as a quaternion,
// size its half extents. A cylinder stands along its own y.
export type Volume = { shape: VolumeShape; p: Vec3; q: [number, number, number, number]; size: Vec3; mirror: boolean }

// Every mask is a field over the model, cut at 1 - amount. Parts masks are on or off.
export type Mask = {
  kind: MaskKind
  invert: boolean
  amount: number
  soft: number
  // Edges and cavities this many texels wide; 0 fades over the baked width.
  crisp: number
  // Noise that roughens the cut, at the given scale; grunge masks use it as repeats.
  rough: number
  scale: number
  axis: Axis
  seed: number
  parts: number[]
  // Surfaces bounded by hard edges, numbered largest first.
  islands: number[]
  // Painted masks keep their pixels in a PNG under this id.
  paint: string
  stamps: Stamp[]
  // Everything inside it, all around, is in the mask, softening towards its walls.
  volume: Volume
  // A built-in grunge map, projected from three sides at `scale` repeats per model size.
  map: string
  // Stamps shaded as sunk in (1) or raised (-1), lit from their top left, with a bevel
  // this part of their width.
  depth?: number
  bevel?: number
}

export type ColorSource = 'layer' | 'stamps' | 'paint' | 'model'

// A folder for layers. Its layers sit next to each other in the stack and
// point to it by id.
export type Group = {
  id: string
  name: string
  open: boolean
  visible: boolean
  // Where the folder's layers may show: these parts and islands. Empty, everywhere.
  zone: { parts: number[]; islands: number[] }
}

// The textures a model gets: its color, and the ones painted besides it when added.
export type Channel = 'color' | 'ao' | 'rough' | 'glow'
export type DataChannel = Exclude<Channel, 'color'>

// A channel painted besides the color. AO goes into the color unless kept separate.
export type ChannelDoc = { kind: DataChannel; separate?: boolean }

export type Layer = {
  id: string
  name: string
  // The channel it paints, the color when unset.
  channel?: DataChannel
  // The folder, empty for none.
  group: string
  color: string
  // Lighter and darker patches over the color, at a world scale; grain varies single texels.
  tone: number
  toneScale: number
  grain: number
  // Baked sheen and glints from the light direction, for glass and gloss.
  shine: number
  // Applied to whatever color the layer takes: hue in degrees, saturation as a factor, brightness added.
  hue: number
  saturation: number
  brightness: number
  // Take the color from image stamps instead of the layer color.
  stampColors: boolean
  // Paint colors, not just where: the first paint mask keeps color and coverage.
  paintColors?: boolean
  // The colors painted so far, kept by the auto palette and offered as swatches.
  brushColors?: string[]
  // Take the color from the model's own texture or material colors, as they were before texxer.
  source?: boolean
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

export type GameTone = 'standard' | 'reinhard' | 'filmic' | 'aces' | 'agx' | 'substratum'

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
  // Tone mapping of the game view, as Godot's; substratum adds its grade.
  gameTone: GameTone
  torch: number
  // How strongly glowing parts bloom in the views.
  bloom: number
}

// How a model's maps are baked: widths of the edge and cavity masks as fractions of
// the model's size, and how much its own normal and occlusion maps count.
export type BakeSettings = { edges: number; cavity: number; detail: number }

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
  bake: BakeSettings
  layers: Layer[]
  groups: Group[]
  variants: Variant[]
  channels: ChannelDoc[]
  finish: Finish
  shading: Shading
  light: Light
}

// A model in a project: its files (several share one texture) and what's painted on it.
// The project's finish, shading and light are shared by all of its models.
export type ModelDoc = Project & {
  id: string
  name: string
  // Absolute in memory, relative to the project file on disk.
  files: string[]
  // Where the export puts the PNGs, beside the first file when unset. Stored like files.
  textures?: string
  // What the last export wrote, to tell whether it needs another.
  exported?: { doc: string; files: string }
  // A model synced from Blender: its .blend, and the GLB the export writes, as
  // its file is the add-on's copy. Stored like files.
  blend?: string
  target?: string
  // Its UVs: Blender's own when unset, texxer's laid out on each sync, or texxer's kept.
  blendUV?: 'texxer' | 'locked'
}

export type Workspace = {
  version: 2
  finish: Finish
  shading: Shading
  light: Light
  models: ModelDoc[]
}

export type View = 'lit' | 'unlit' | 'game' | 'mask'
export type FlatMap = 'texture' | 'mask' | 'glow' | 'ao' | 'edges' | 'cavity' | 'parts'

export type Tool = 'nav' | 'brush' | 'stamp' | 'volume'

export const channels: Channel[] = ['color', 'ao', 'rough', 'glow']

export const maskKinds: MaskKind[] = ['surfaces', 'paint', 'stamps', 'volume', 'grunge', 'ao', 'edges', 'cavity', 'gradient', 'facing', 'drips']
export const blends: Blend[] = ['normal', 'multiply', 'add', 'overlay', 'relief']
