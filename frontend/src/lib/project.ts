import type { Layer, Mask, MaskKind, Project } from './types'

let counter = 0
export const uid = () => Date.now().toString(36) + (counter++).toString(36)

const maskDefaults: Record<MaskKind, Partial<Mask>> = {
  parts: { amount: 1 },
  islands: { amount: 1 },
  paint: { amount: 0.5 },
  stamps: { amount: 0.5 },
  volume: { amount: 1, soft: 0.3 },
  grunge: { amount: 0.4, scale: 3, map: 'grunge_1' },
  ao: { amount: 0.45, soft: 0.4 },
  edges: { amount: 0.3, rough: 0.25 },
  cavity: { amount: 0.5, rough: 0.3 },
  gradient: { amount: 0.3, rough: 0.4 },
  facing: { amount: 0.3, rough: 0.2 },
  noise: { amount: 0.4, scale: 8 },
}

export function newMask(kind: MaskKind): Mask {
  return {
    kind,
    invert: false,
    amount: 0.5,
    soft: 0,
    crisp: 0,
    rough: 0,
    scale: 24,
    axis: 1,
    seed: Math.floor(Math.random() * 1000),
    parts: [],
    islands: [],
    paint: kind === 'paint' ? uid() : '',
    stamps: [],
    volume: { shape: 'box', p: [0, 0, 0], q: [0, 0, 0, 1], size: [0, 0, 0], mirror: false },
    map: '',
    ...maskDefaults[kind],
  }
}

export function newLayer(name: string, color: string, masks: Mask[] = [], extra: Partial<Layer> = {}): Layer {
  return { id: uid(), name, group: '', color, tone: 0.08, toneScale: 12, grain: 0.02, shine: 0, stampColors: false, glow: false, opacity: 1, blend: 'normal', visible: true, masks, ...extra }
}

// A starter stack that already reads as worn metal, so a fresh model looks like something.
// Shading comes from the baked light.
export function newProject(): Project {
  return {
    version: 1,
    size: 512,
    unwrap: 0,
    bake: { edges: 0.03, cavity: 0.03, detail: 1 },
    layers: [
      newLayer('base', '#6b6a62'),
      newLayer('dirt', '#4a4234', [newMask('gradient')], { opacity: 0.8 }),
      newLayer('wear', '#a8a59a', [newMask('edges')]),
    ],
    groups: [],
    variants: [],
    finish: { supersample: 2, sharpen: 0.2, palette: 'auto', count: 48, colors: [], dither: 0, glow: 3 },
    shading: {
      on: true,
      azimuth: 60,
      elevation: 50,
      key: 0.45,
      keyColor: '#ffd8a8',
      aoFloor: 0.45,
      formFloor: 0.72,
      under: 0.12,
      underColor: '#0e1c22',
      rim: 0.3,
      bothSides: true,
    },
    light: { key: 2.2, ambient: 1, headlight: 0.6, angle: 35, gameAmbient: '#ffffff', gameEnergy: 1, gameTop: 0.6, torch: 1, bloom: 0.05 },
  }
}

// Fills fields that older project files don't have yet.
export function normalize(p: Partial<Project>): Project {
  const base = newProject()
  return {
    ...base,
    ...p,
    bake: { ...base.bake, ...p.bake },
    finish: { ...base.finish, ...p.finish },
    shading: { ...base.shading, ...p.shading },
    light: { ...base.light, ...p.light },
    groups: (p.groups ?? []).map((g) => ({ ...g, zone: g.zone ?? { parts: [], islands: [] } })),
    layers: (p.layers ?? base.layers).map((l) => ({
      ...newLayer(l.name, l.color),
      ...l,
      masks: (l.masks ?? []).map((m) => ({ ...newMask(m.kind), ...m })),
    })),
  }
}

// The project as a variant shows it; an unknown id is the base.
export function withVariant(p: Project, id: string): Project {
  const v = p.variants.find((v) => v.id === id)
  if (!v) return p
  return { ...p, layers: p.layers.map((l) => ({ ...l, ...v.layers[l.id] })) }
}
