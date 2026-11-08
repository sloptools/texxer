import * as api from '../../wailsjs/go/main/App'
import { newLayer, newMask, uid } from './project'
import { store } from './store.svelte'
import type { Group, Layer, Mask } from './types'

// A material is a folder of layers to put on any model: the folder gets an empty
// zone, and clicking surfaces fills it. Layers bottom first.
export type Material = { name: string; layers: MaterialLayer[]; file?: string }
type MaterialLayer = Partial<Omit<Layer, 'masks'>> & { name: string; color: string; masks?: Partial<Mask>[] }

const m = (kind: Mask['kind'], o: Partial<Mask> = {}): Partial<Mask> => ({ kind, ...o })

export const builtins: Material[] = [
  {
    name: 'worn steel',
    layers: [
      { name: 'steel', color: '#b4b1aa', tone: 0.32, toneScale: 40, grain: 0.2 },
      { name: 'dark specks', color: '#6a665f', grain: 0.12, masks: [m('grunge', { amount: 0.3, map: 'specks', scale: 4 })] },
      { name: 'grime', color: '#2b2420', grain: 0.1, masks: [m('cavity', { amount: 0.55, rough: 0.3 })] },
      { name: 'rust', color: '#8a4522', tone: 0.3, grain: 0.15, masks: [m('cavity', { amount: 0.45, rough: 0.6 }), m('grunge', { amount: 0.45, map: 'specks', scale: 3 })] },
      { name: 'bright edges', color: '#f1efe8', grain: 0.08, masks: [m('edges', { amount: 0.16, rough: 0.3, scale: 60 })] },
    ],
  },
  {
    name: 'painted metal',
    layers: [
      { name: 'paint', color: '#5b6b4a', tone: 0.15, toneScale: 10, grain: 0.06 },
      { name: 'grime', color: '#2a2722', opacity: 0.6, masks: [m('cavity', { amount: 0.4, rough: 0.3 })] },
      { name: 'chipped', color: '#8c8a85', tone: 0.2, masks: [m('edges', { amount: 0.1, rough: 0.4, scale: 40 })] },
      { name: 'rust', color: '#6b3a22', tone: 0.3, masks: [m('edges', { amount: 0.14, rough: 0.6, scale: 30 }), m('grunge', { amount: 0.35, map: 'blotches', scale: 3 })] },
      { name: 'dirt low', color: '#3d342b', opacity: 0.5, masks: [m('gradient', { amount: 0.16, rough: 0.4 }), m('grunge', { amount: 0.5, map: 'clouds', scale: 3 })] },
    ],
  },
  {
    name: 'rust',
    layers: [
      { name: 'rust', color: '#6e3a22', tone: 0.3, toneScale: 20, grain: 0.12 },
      { name: 'dark rust', color: '#4a2616', masks: [m('grunge', { amount: 0.4, map: 'blotches', scale: 2 })] },
      { name: 'flakes', color: '#a4552a', grain: 0.1, masks: [m('grunge', { amount: 0.25, map: 'specks', scale: 3 })] },
      { name: 'grime', color: '#241a14', masks: [m('cavity', { amount: 0.5, rough: 0.3 })] },
    ],
  },
  {
    name: 'wood',
    layers: [
      { name: 'wood', color: '#b8621c', tone: 0.28, toneScale: 24, grain: 0.12 },
      { name: 'grain', color: '#7c3a10', grain: 0.1, masks: [m('grunge', { amount: 0.35, map: 'lines', scale: 3 })] },
      { name: 'worn edges', color: '#d08a4a', masks: [m('edges', { amount: 0.1, rough: 0.4, scale: 40 })] },
      { name: 'grime', color: '#3a2414', masks: [m('cavity', { amount: 0.4, rough: 0.3 })] },
    ],
  },
  {
    name: 'plastic',
    layers: [
      { name: 'plastic', color: '#2e2f31', tone: 0.12, toneScale: 10, grain: 0.04, shine: 0.15 },
      { name: 'scuffs', color: '#55575a', masks: [m('edges', { amount: 0.08, rough: 0.5, scale: 40 })] },
      { name: 'dust', color: '#6b685f', opacity: 0.3, masks: [m('facing', { amount: 0.2, axis: 1, rough: 0.3 })] },
    ],
  },
  {
    name: 'rubber',
    layers: [
      { name: 'rubber', color: '#232322', tone: 0.1, grain: 0.05 },
      { name: 'worn', color: '#3a3936', masks: [m('edges', { amount: 0.1, rough: 0.4 })] },
      { name: 'dust', color: '#5a564e', opacity: 0.4, masks: [m('facing', { amount: 0.25, axis: 1, rough: 0.3 })] },
    ],
  },
  {
    name: 'glass',
    layers: [
      { name: 'glass', color: '#1d3a40', tone: 0.05, grain: 0.02, shine: 0.6 },
      { name: 'grime', color: '#2a2a24', opacity: 0.5, masks: [m('grunge', { amount: 0.25, map: 'grime', scale: 2 })] },
    ],
  },
]

export const library = $state<{ saved: Material[] }>({ saved: [] })

export async function loadLibrary() {
  try {
    const files = await api.Materials()
    library.saved = Object.entries(files)
      .map(([file, json]) => ({ ...(JSON.parse(json) as Material), file }))
      .sort((a, b) => a.name.localeCompare(b.name))
  } catch (e) {
    store.say(String(e))
  }
}

// Masks that point at one model's surfaces don't travel; the folder's zone takes their place.
const local = new Set<Mask['kind']>(['surfaces', 'paint', 'stamps'])

export async function saveMaterial(group: Group, name: string) {
  const layers: MaterialLayer[] = store.project.layers
    .filter((l) => l.group === group.id)
    .map(({ id, group: _, channel: __, masks, ...rest }) => ({ ...rest, masks: masks.filter((k) => !local.has(k.kind)) }))
  try {
    await api.SaveMaterial(name, JSON.stringify({ name, layers }, null, 2))
    await loadLibrary()
    store.say('saved material ' + name)
  } catch (e) {
    store.say(String(e))
  }
}

export async function deleteMaterial(mat: Material) {
  if (!mat.file) return
  try {
    await api.DeleteMaterial(mat.file)
    await loadLibrary()
  } catch (e) {
    store.say(String(e))
  }
}

// Puts a material above the selected layer as a new folder, in the channel being
// painted, and picks the folder, so clicks on the model fill its zone.
export function insertMaterial(mat: Material) {
  const p = store.project
  const g: Group = { id: uid(), name: mat.name, open: false, visible: true, zone: { parts: [], islands: [] } }
  const layers = mat.layers.map(({ masks, ...rest }) =>
    newLayer(rest.name, rest.color, (masks ?? []).map((k) => ({ ...newMask(k.kind!), ...k })), { ...rest, group: g.id, channel: store.channel === 'color' ? undefined : store.channel }),
  )

  // Above the selected layer's folder, so it doesn't land inside one.
  const sel = store.layer
  let at = sel ? store.selected + 1 : p.layers.length
  if (sel?.group) at = Math.max(...p.layers.map((l, i) => (l.group === sel.group ? i : -1))) + 1
  p.groups.push(g)
  p.layers.splice(Math.min(at, p.layers.length), 0, ...layers)
  store.selected = at + layers.length - 1
  store.selectedGroup = g.id
  store.pickingZone = true
}
