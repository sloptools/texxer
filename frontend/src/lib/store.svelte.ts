import * as api from '../../wailsjs/go/main/App'
import type { main } from '../../wailsjs/go/models'
import type { PaintStore } from './gl/paint'
import { newLayer, newMask, newProject, uid } from './project'
import type { Finish, FlatMap, Group, Layer, LayerLook, MaskKind, ModelDoc, Variant, View, Workspace } from './types'
import { serializeWorkspace } from './workspace'

export function load<T>(key: string, fallback: T): T {
  try {
    const v = localStorage.getItem('texxer.' + key)
    return v === null ? fallback : JSON.parse(v)
  } catch {
    return fallback
  }
}

export function persist(key: string, value: unknown) {
  try {
    localStorage.setItem('texxer.' + key, JSON.stringify(value))
  } catch {}
}

// Project edits are JSON snapshots; strokes keep the mask's pixels before and after.
type Entry = { kind: 'project'; before: string; after: string } | { kind: 'paint'; id: string; before: Uint8Array; after: Uint8Array }
export type History = { past: Entry[]; future: Entry[]; last: string }

class Store {
  // The project file and what's in it; project is the model being painted.
  path = $state('')
  workspace = $state<Workspace | null>(null)
  model = $state<main.ModelInfo | null>(null)
  project = $state<ModelDoc>({ ...newProject(), id: '', name: '', files: [] })
  // Set by the viewport once the model's texture renders.
  ready = $state(false)
  // Hashes of each model's files, by model key, to tell which ones changed.
  fileHashes = $state<Record<string, string>>({})
  quickOpen = $state(false)
  selected = $state(0)
  // The variant being looked at and edited, '' for the base.
  variant = $state('')
  view = $state<View>('lit')
  bloom = $state(true)
  grid = $state(false)
  torch = $state(false)
  // Vertical resolution of the view, 0 for full; lets you judge the texture at game size.
  lines = $state(0)
  baking = $state(false)
  hoverPart = $state(-1)
  hoverIsland = $state(-1)
  // The 2D view of the texture and its UV layout, instead of the model.
  flat = $state(false)
  flatMap = $state<FlatMap>('texture')
  flatUV = $state(true)
  hoverTexel = $state<[number, number] | null>(null)
  help = $state(false)
  helpTab = $state<'docs' | 'shortcuts'>('shortcuts')
  helpPage = $state('')
  message = $state('')
  grungeMaps = $state<string[]>([])
  stampImages = $state<string[]>([])

  // Set by the viewport: renders a variant's ('' for the base) finished textures as base64 PNGs; emissive is empty when nothing glows.
  snapshot: ((variant: string) => { color: string; emissive: string }) | null = null
  paint: PaintStore | null = null

  past: Entry[] = []
  future: Entry[] = []
  last = ''
  private commitTimer = 0
  private saveTimer = 0
  private messageTimer = 0

  get layer(): Layer | undefined {
    return this.project.layers[this.selected]
  }

  init() {
    api.GrungeMaps().then((m) => (this.grungeMaps = m))
    api.Stamps().then((s) => (this.stampImages = s))
    $effect.root(() => {
      $effect(() => {
        const json = JSON.stringify(this.project)
        clearTimeout(this.commitTimer)
        this.commitTimer = setTimeout(() => this.commit(json), 250)
      })
      $effect(() => {
        if (!this.workspace || !this.path) return
        const path = this.path
        const json = serializeWorkspace(path, this.workspace)
        clearTimeout(this.saveTimer)
        this.saveTimer = setTimeout(() => api.SaveProject(path, json).catch((e) => this.say(String(e))), 600)
      })
    })
  }

  // Opens the docs at a page, from the ? buttons beside panels.
  openDocs(page: string) {
    this.helpTab = 'docs'
    this.helpPage = page
    this.help = true
  }

  say(text: string) {
    this.message = text
    clearTimeout(this.messageTimer)
    this.messageTimer = setTimeout(() => (this.message = ''), 4000)
  }

  // History

  private push(e: Entry) {
    this.past.push(e)
    if (this.past.length > 200) this.past.shift()
    this.future = []
  }

  private commit(json: string) {
    clearTimeout(this.commitTimer)
    if (json === this.last) return
    this.push({ kind: 'project', before: this.last, after: json })
    this.last = json
  }

  // Each model keeps its own history while another one is open.
  takeHistory(): History {
    this.commit(JSON.stringify(this.project))
    return { past: this.past, future: this.future, last: this.last }
  }

  giveHistory(h: History | undefined) {
    this.past = h?.past ?? []
    this.future = h?.future ?? []
    this.last = h?.last ?? JSON.stringify(this.project)
  }

  pushPaint(id: string, before: Uint8Array, after: Uint8Array) {
    this.commit(JSON.stringify(this.project))
    this.push({ kind: 'paint', id, before, after })
  }

  undo() {
    this.commit(JSON.stringify(this.project))
    const e = this.past.pop()
    if (!e) return
    this.future.push(e)
    this.apply(e, 'before')
  }

  redo() {
    const e = this.future.pop()
    if (!e) return
    this.past.push(e)
    this.apply(e, 'after')
  }

  private apply(e: Entry, side: 'before' | 'after') {
    if (e.kind === 'paint') {
      this.paint?.write(e.id, e[side])
      return
    }
    // The layout stays: switching it moves the paint, which history doesn't.
    // Shared settings are changed in place, so every model keeps seeing them.
    const p = JSON.parse(e[side])
    const m = this.project
    Object.assign(m.finish, p.finish)
    Object.assign(m.shading, p.shading)
    Object.assign(m.light, p.light)
    Object.assign(m, { size: p.size, layers: p.layers, groups: p.groups, variants: p.variants })
    this.last = JSON.stringify(m)
    this.selected = Math.min(this.selected, this.project.layers.length - 1)
  }

  // Lays the UVs out for `to` (0: the model's own) and carries the painted masks over.
  async setLayout(to: number) {
    const model = this.model
    const from = this.project.unwrap
    if (!model || from === to) return
    try {
      await this.paint?.flush()
      await api.MovePaint(this.path, model.path, from, model.path, to, this.paintIds())
    } catch (e) {
      this.say(String(e))
      return
    }
    // Strokes in history belong to the old layout.
    this.past = this.past.filter((e) => e.kind !== 'paint')
    this.future = this.future.filter((e) => e.kind !== 'paint')
    this.project.unwrap = to
  }

  paintIds(): string[] {
    return this.project.layers.flatMap((l) => l.masks.filter((m) => m.kind === 'paint').map((m) => m.paint))
  }

  // Variants

  get activeVariant(): Variant | undefined {
    return this.project.variants.find((v) => v.id === this.variant)
  }

  addVariant() {
    const v: Variant = { id: uid(), name: 'variant ' + (this.project.variants.length + 1), layers: {} }
    this.project.variants.push(v)
    this.variant = v.id
  }

  deleteVariant(id: string) {
    this.project.variants = this.project.variants.filter((v) => v.id !== id)
    if (this.variant === id) this.variant = ''
  }

  // Base, then each variant, round and round.
  cycleVariant() {
    const ids = ['', ...this.project.variants.map((v) => v.id)]
    this.variant = ids[(ids.indexOf(this.variant) + 1) % ids.length]
  }

  // The layer as the active variant shows it. Color, visibility and opacity read and
  // write the variant's changes; a value back at the base drops the change.
  look(l: Layer) {
    const variant = () => this.activeVariant
    const get = <K extends keyof LayerLook>(k: K) => (variant()?.layers[l.id]?.[k] ?? l[k]) as Layer[K]
    const set = <K extends keyof LayerLook>(k: K, value: Layer[K]) => {
      const v = variant()
      if (!v) {
        l[k] = value
        return
      }
      const look = { ...v.layers[l.id], [k]: value }
      if (value === l[k]) delete look[k]
      if (Object.keys(look).length) v.layers[l.id] = look
      else delete v.layers[l.id]
    }
    return {
      get color() {
        return get('color')
      },
      set color(c: string) {
        set('color', c)
      },
      get visible() {
        return get('visible')
      },
      set visible(on: boolean) {
        set('visible', on)
      },
      get opacity() {
        return get('opacity')
      },
      set opacity(o: number) {
        set('opacity', o)
      },
      // Whether the active variant changes this layer.
      get changed() {
        return !!variant()?.layers[l.id]
      },
    }
  }

  // Layers

  // With a mask, the new layer starts empty, ready for the brush or stamps.
  addLayer(mask?: MaskKind) {
    const at = this.selected + 1
    const group = this.layer?.group ?? ''
    this.project.layers.splice(at, 0, newLayer('layer ' + (this.project.layers.length + 1), '#8a8578', mask ? [newMask(mask)] : [], { group }))
    this.selected = at
  }

  duplicateLayer() {
    const l = this.layer
    if (!l) return
    const copy = { ...$state.snapshot(l), id: uid(), name: l.name + ' copy' }
    for (const m of copy.masks) {
      if (m.kind !== 'paint') continue
      const id = uid()
      this.paint?.clone(m.paint, id)
      m.paint = id
    }
    this.project.layers.splice(this.selected + 1, 0, copy)
    this.selected++
  }

  deleteLayer() {
    if (!this.layer) return
    this.project.layers.splice(this.selected, 1)
    this.selected = Math.max(0, Math.min(this.selected, this.project.layers.length - 1))
    this.dropEmptyGroups()
  }

  // Moves layers, in stack order, so they start at index `at` of the current
  // stack, into the given folder.
  moveLayers(ids: string[], at: number, group: string) {
    const layers = this.project.layers
    const selectedId = this.layer?.id
    const moving = layers.filter((l) => ids.includes(l.id))
    const before = layers.slice(0, at).filter((l) => ids.includes(l.id)).length
    const rest = layers.filter((l) => !ids.includes(l.id))
    for (const l of moving) l.group = group
    rest.splice(at - before, 0, ...moving)
    this.project.layers = rest
    this.selected = Math.max(0, rest.findIndex((l) => l.id === selectedId))
    this.dropEmptyGroups()
  }

  group(id: string): Group | undefined {
    return this.project.groups.find((g) => g.id === id)
  }

  groupLayer() {
    const l = this.layer
    if (!l) return
    const g: Group = { id: uid(), name: 'folder ' + (this.project.groups.length + 1), open: true, visible: true }
    this.project.groups.push(g)
    l.group = g.id
    this.dropEmptyGroups()
  }

  // Dissolves the selected layer's folder; its layers stay where they are.
  ungroup() {
    const id = this.layer?.group
    if (!id) return
    for (const l of this.project.layers) if (l.group === id) l.group = ''
    this.dropEmptyGroups()
  }

  private dropEmptyGroups() {
    const used = new Set(this.project.layers.map((l) => l.group))
    this.project.groups = this.project.groups.filter((g) => used.has(g.id))
  }

  // One step up or down, joining the folder it moves into.
  moveLayer(from: number, to: number) {
    const layers = this.project.layers
    if (to < 0 || to >= layers.length || from === to) return
    const [l] = layers.splice(from, 1)
    layers.splice(to, 0, l)
    const below = layers[to - 1]
    const above = layers[to + 1]
    if (below && above && below.group === above.group) l.group = below.group
    else if (below?.group !== l.group && above?.group !== l.group) l.group = ''
    this.selected = to
    this.dropEmptyGroups()
  }

  select(delta: number) {
    this.selected = Math.max(0, Math.min(this.project.layers.length - 1, this.selected + delta))
  }

  addMask(kind: MaskKind) {
    this.layer?.masks.push(newMask(kind))
  }

  // The selected layer's first mask of this kind, added if there is room.
  ensureMask(kind: MaskKind): number {
    const l = this.layer
    if (!l) return -1
    const at = l.masks.findIndex((m) => m.kind === kind)
    if (at >= 0) return at

    // An empty stamps mask would hide the paint, as masks intersect; paint takes its place.
    const empty = l.masks.findIndex((m) => m.kind === 'stamps' && !m.stamps.length)
    if (kind === 'paint' && empty >= 0) {
      l.masks[empty] = newMask(kind)
      return empty
    }
    if (l.masks.length >= 4) {
      this.say('this layer has 4 masks already')
      return -1
    }
    l.masks.push(newMask(kind))
    return l.masks.length - 1
  }

  // Toggles a part in the selected layer's parts mask, adding the mask if needed.
  togglePart(part: number) {
    const l = this.layer
    if (!l || part < 0) return
    let m = l.masks.find((m) => m.kind === 'parts')
    if (!m) {
      m = newMask('parts')
      l.masks.unshift(m)
      m = l.masks[0]
    }
    const i = m.parts.indexOf(part)
    if (i >= 0) m.parts.splice(i, 1)
    else m.parts.push(part)
  }

  // Clicks pick islands while the selected layer has an islands mask.
  get picksIslands() {
    return !!this.layer?.masks.some((m) => m.kind === 'islands')
  }

  // Toggles an island in the selected layer's islands mask, adding the mask if needed.
  toggleIsland(island: number) {
    const l = this.layer
    if (!l || island < 0) return
    let m = l.masks.find((m) => m.kind === 'islands')
    if (!m) {
      l.masks.unshift(newMask('islands'))
      m = l.masks[0]
    }
    const i = m.islands.indexOf(island)
    if (i >= 0) m.islands.splice(i, 1)
    else m.islands.push(island)
  }

  // Finish and export

  async setPalette(kind: Finish['palette']) {
    if (kind === 'none' || kind === 'auto') {
      this.project.finish.palette = kind
      this.project.finish.colors = []
      return
    }
    try {
      const colors = (await api.Palette(kind)) as [number, number, number][]
      if (!colors?.length) return
      this.project.finish.palette = kind
      this.project.finish.colors = colors
    } catch (e) {
      this.say(String(e))
    }
  }

  cycleLines() {
    const steps = [0, 480, 360, 240]
    this.lines = steps[(steps.indexOf(this.lines) + 1) % steps.length]
  }

  async importStamps() {
    try {
      const list = await api.ImportStamps()
      if (list) this.stampImages = list
    } catch (e) {
      this.say(String(e))
    }
  }
}

export const store = new Store()
