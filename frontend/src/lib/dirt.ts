import { newLayer, newMask } from './project'
import { store } from './store.svelte'
import { tools } from './tools.svelte'
import type { Layer, Mask, MaskKind } from './types'

// Dirt to add with a key, above the selected layer: one layer, or a few bottom first.
// A brush preset switches to the brush with that grunge map.
export type Dirt = { name: string; key: string; make: () => Layer | Layer[]; brush?: string }

const m = (kind: MaskKind, o: Partial<Mask> = {}): Mask => ({ ...newMask(kind), ...o })

// Rust under the chips, from the same map and seed a bit wider, rims them.
function chips(): Layer[] {
  const seed = Math.floor(Math.random() * 1000)
  return [
    newLayer('rust rims', '#6e3b22', [m('grunge', { amount: 0.11, map: 'grime', scale: 1.5, seed })], { tone: 0.2, grain: 0.04 }),
    newLayer('chips', '#25221f', [m('grunge', { amount: 0.08, map: 'grime', scale: 1.5, seed })], { tone: 0.1, grain: 0.03 }),
  ]
}

export const dirt: Dirt[] = [
  {
    name: 'dust on top',
    key: '⇧1',
    make: () => newLayer('dust', '#8a8374', [m('facing', { amount: 0.25, rough: 0.3 }), m('grunge', { amount: 0.6, map: 'grime', scale: 3 })], { opacity: 0.6 }),
  },
  {
    name: 'grime in cavities',
    key: '⇧2',
    make: () => newLayer('grime', '#2a2520', [m('cavity', { amount: 0.6, rough: 0.2 }), m('grunge', { amount: 0.7, map: 'grime', scale: 3 })]),
  },
  {
    name: 'dirt from below',
    key: '⇧3',
    make: () => newLayer('dirt', '#3d342b', [m('gradient', { amount: 0.2, rough: 0.3 }), m('grunge', { amount: 0.55, map: 'clouds', scale: 3 })], { opacity: 0.8 }),
  },
  {
    name: 'rust on edges',
    key: '⇧4',
    make: () => newLayer('rust', '#7a3d20', [m('edges', { amount: 1, crisp: 2 }), m('grunge', { amount: 0.45, map: 'specks', scale: 3 })]),
  },
  {
    name: 'drips below edges',
    key: '⇧5',
    make: () => newLayer('drips', '#2e2a24', [m('drips', { amount: 0.3, rough: 0.6, scale: 30 }), m('grunge', { amount: 0.4, map: 'streaks', scale: 2 })], { opacity: 0.5 }),
  },
  {
    name: 'chips with rust',
    key: '⇧6',
    make: chips,
  },
  {
    name: 'grunge brush',
    key: '⇧7',
    brush: 'clouds',
    make: () => newLayer('painted dirt', '#3a3229', [m('paint')]),
  },
]

export function addDirt(d: Dirt) {
  if (!store.model) return
  const made = d.make()
  for (const l of Array.isArray(made) ? made : [made]) store.addLayer(l)
  if (d.brush) {
    tools.brush.map = d.brush
    tools.set('brush')
  }
  store.say('added ' + d.name)
}
