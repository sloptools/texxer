import { newLayer, newMask } from './project'
import { store } from './store.svelte'
import { tools } from './tools.svelte'
import type { Layer, Mask, MaskKind } from './types'

// One-layer dirt to add with a key, above the selected layer.
export type Dirt = { name: string; key: string; make: () => Layer; brush?: boolean }

const m = (kind: MaskKind, o: Partial<Mask> = {}): Mask => ({ ...newMask(kind), ...o })

export const dirt: Dirt[] = [
  {
    name: 'dust on top',
    key: '1',
    make: () => newLayer('dust', '#8a8374', [m('facing', { amount: 0.25, rough: 0.3 }), m('grunge', { amount: 0.6, map: 'grime_1', scale: 3 })], { opacity: 0.6 }),
  },
  {
    name: 'grime in cavities',
    key: '2',
    make: () => newLayer('grime', '#2a2520', [m('cavity', { amount: 0.6, rough: 0.2 }), m('grunge', { amount: 0.7, map: 'grime_2', scale: 3 })]),
  },
  {
    name: 'dirt from below',
    key: '3',
    make: () => newLayer('dirt', '#3d342b', [m('gradient', { amount: 0.2, rough: 0.3 }), m('grunge', { amount: 0.55, map: 'dirt_1', scale: 3 })], { opacity: 0.8 }),
  },
  {
    name: 'rust on edges',
    key: '4',
    make: () => newLayer('rust', '#7a3d20', [m('edges', { amount: 1, crisp: 2 }), m('grunge', { amount: 0.45, map: 'specks_1', scale: 3 })]),
  },
  {
    name: 'streaks on walls',
    key: '5',
    make: () => newLayer('streaks', '#2e2a24', [m('facing', { amount: 0.25, invert: true }), m('facing', { amount: 0.25, axis: 4, invert: true }), m('grunge', { amount: 0.22, map: 'streaks_1', scale: 2 })], { opacity: 0.35 }),
  },
  {
    name: 'grunge brush',
    key: '6',
    brush: true,
    make: () => newLayer('painted dirt', '#3a3229', [m('paint'), m('grunge', { amount: 0.6, map: 'dirt_1', scale: 3 })]),
  },
]

export function addDirt(d: Dirt) {
  if (!store.model) return
  store.addLayer(d.make())
  if (d.brush) tools.set('brush')
  store.say('added ' + d.name)
}
