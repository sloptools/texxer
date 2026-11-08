import * as THREE from 'three'
import * as api from '../../wailsjs/go/main/App'
import type { main } from '../../wailsjs/go/models'
import type { PaintStore } from './gl/paint'
import { channelOf, newLayer, newMask, newProject, uid } from './project'
import { channels, type Channel, type ColorSource, type DataChannel, type Finish, type FlatMap, type Group, type Layer, type LayerLook, type Mask, type MaskKind, type ModelDoc, type Variant, type Vec3, type View, type Workspace } from './types'
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
  // The layer the list is hovering, -1 for none.
  hoverLayer = $state(-1)
  // The next click on the model takes its color into the selected layer.
  sampling = $state(false)
  // The next click on the model sets the selected layer's adjustments to match its color.
  matching = $state(false)
  model = $state<main.ModelInfo | null>(null)
  project = $state<ModelDoc>({ ...newProject(), id: '', name: '', files: [] })
  // Set by the viewport once the model's texture renders.
  ready = $state(false)
  // Hashes of each model's files, by model key, to tell which ones changed.
  fileHashes = $state<Record<string, string>>({})
  quickOpen = $state(false)
  selected = $state(0)
  // Layers marked besides the selected one (⌘ and ⇧ click), by id, to delete or move together.
  marked = $state<string[]>([])
  // A folder picked in the list shows its panel instead of the layer's.
  selectedGroup = $state('')
  // Clicks on the model add to the picked folder's zone only while this is on.
  pickingZone = $state(false)
  // The channel being painted, and another one shown faintly over it for reference.
  channel = $state<Channel>('color')
  onion = $state<Channel | ''>('')
  onionAmount = 0.3
  private picks: Partial<Record<Channel, string>> = {}
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

  // Set by the viewport: renders a variant's ('' for the base) finished textures as base64 PNGs;
  // emissive, rough and ao are empty when nothing glows or the channel goes nowhere of its own.
  snapshot: ((variant: string) => Snapshot) | null = null
  // Set by the viewport: layers rendered alone, premultiplied, at the bake's size.
  flattened: ((indices: number[]) => Uint8Array) | null = null
  paint: PaintStore | null = null

  past: Entry[] = []
  future: Entry[] = []
  last = ''
  private commitTimer = 0
  private saveTimer = 0
  private messageTimer = 0

  // Only a layer of the channel being painted.
  get layer(): Layer | undefined {
    const l = this.project.layers[this.selected]
    return l && channelOf(l) === this.channel ? l : undefined
  }

  init() {
    api.GrungeMaps().then((m) => (this.grungeMaps = m))
    api.Stamps().then((s) => (this.stampImages = s))
    $effect.root(() => {
      // Another model may not have the channel being painted.
      $effect(() => {
        if (!this.hasChannel(this.channel)) this.setChannel('color')
      })
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

  // Sticky messages stay until the next one, for errors that leave the view broken.
  say(text: string, sticky = false) {
    this.message = text
    clearTimeout(this.messageTimer)
    if (!sticky) this.messageTimer = setTimeout(() => (this.message = ''), 4000)
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
    Object.assign(m, { size: p.size, layers: p.layers, groups: p.groups, variants: p.variants, channels: p.channels })
    if (!this.hasChannel(this.channel)) this.channel = 'color'
    this.last = JSON.stringify(m)
    this.selected = Math.min(this.selected, this.project.layers.length - 1)
    this.marked = []
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

  // Channels

  hasChannel(c: Channel) {
    return c === 'color' || this.project.channels.some((d) => d.kind === c)
  }

  // Switches to a channel, adding it the first time; the selection goes back to the
  // layer picked there last, else to its top layer.
  setChannel(c: Channel) {
    if (!this.hasChannel(c)) this.project.channels.push({ kind: c as DataChannel })
    if (this.layer) this.picks[this.channel] = this.layer.id
    this.channel = c
    if (this.onion === c) this.onion = ''
    const layers = this.project.layers
    let top = layers.findIndex((l) => l.id === this.picks[c] && channelOf(l) === c)
    if (top < 0) {
      top = layers.length - 1
      while (top >= 0 && channelOf(layers[top]) !== c) top--
    }
    this.selected = top
    this.selectedGroup = ''
    this.marked = []
    this.pickingZone = false
  }

  // The channels there are, round and round.
  cycleChannel() {
    const list = channels.filter((c) => this.hasChannel(c))
    this.setChannel(list[(list.indexOf(this.channel) + 1) % list.length])
  }

  // Off, then each other channel as the onion.
  cycleOnion() {
    const list: (Channel | '')[] = ['', ...channels.filter((c) => c !== this.channel && this.hasChannel(c))]
    this.onion = list[(list.indexOf(this.onion) + 1) % list.length]
  }

  // Removes a channel and its layers.
  deleteChannel(c: DataChannel) {
    this.project.layers = this.project.layers.filter((l) => l.channel !== c)
    this.project.channels = this.project.channels.filter((d) => d.kind !== c)
    if (this.onion === c) this.onion = ''
    this.dropEmptyGroups()
    if (this.channel === c) this.setChannel('color')
  }

  // Copies the picked layers, or the picked folder with its layers, to the top of
  // another channel, painted masks and all.
  copyToChannel(c: Channel) {
    const group = this.selectedGroup ? this.group(this.selectedGroup) : undefined
    const from = group ? this.project.layers.filter((l) => l.group === group.id) : this.pickedLayers()
    if (!from.length) return
    if (!this.hasChannel(c)) this.project.channels.push({ kind: c as DataChannel })
    const copyGroup = group && { ...$state.snapshot(group), id: uid() }
    if (copyGroup) this.project.groups.push(copyGroup)
    const copies = from.map((l) => {
      const copy = { ...this.copy(l), group: copyGroup?.id ?? '' }
      if (c === 'color') delete copy.channel
      else copy.channel = c as DataChannel
      return copy
    })
    this.project.layers.push(...copies)
    this.say(`copied ${copies.length === 1 ? copies[0].name : copies.length + ' layers'} to ${c}`)
  }

  // A layer as a new one, with its own copies of its painted masks.
  private copy(l: Layer): Layer {
    const copy = { ...$state.snapshot(l), id: uid() }
    for (const m of copy.masks) {
      if (m.kind !== 'paint') continue
      const id = uid()
      this.paint?.clone(m.paint, id)
      m.paint = id
    }
    return copy
  }

  // Layers

  // Above the selected layer, or at the top of the picked folder. A new one starts
  // with an empty paint mask, so it shows nothing until painted or given another mask.
  addLayer(layer?: Layer) {
    const layers = this.project.layers
    let at = this.selected + 1
    let group = this.layer?.group ?? ''
    if (this.selectedGroup) {
      group = this.selectedGroup
      at = Math.max(...layers.map((l, i) => (l.group === group ? i : -1))) + 1
    }
    if (!this.layer && !this.selectedGroup) at = layers.length
    at = Math.min(at, layers.length)
    const l = layer ?? newLayer('layer ' + (layers.length + 1), layerColors[this.channel], [newMask('paint')])
    const channel = this.channel === 'color' ? undefined : this.channel
    layers.splice(at, 0, { ...l, group, channel })
    this.selected = at
    this.selectedGroup = ''
    this.marked = []
    this.pickingZone = false
  }

  // The selected layer, or the picked folder's shown layers, baked into one layer
  // that paints colors, above them; they stay, hidden, to go back to.
  flatten() {
    const layers = this.project.layers
    const group = this.selectedGroup
    const indices = group ? layers.flatMap((l, i) => (l.group === group && l.visible ? [i] : [])) : this.layer ? [this.selected] : []
    if (!indices.length || !this.paint || !this.flattened) return
    const data = this.flattened(indices)

    const top = indices[indices.length - 1]
    const name = group ? (this.group(group)?.name ?? 'folder') : layers[top].name
    const mask = newMask('paint')
    const flat = newLayer(name + ' flat', '#808080', [mask], { paintColors: true, tone: 0, grain: 0, group: group ? '' : layers[top].group, channel: layers[top].channel })
    for (const i of indices) layers[i].visible = false
    // Out of the folder, which would hide it again by its zone, right above it.
    const at = group ? Math.max(...layers.map((l, i) => (l.group === group ? i : -1))) + 1 : top + 1
    layers.splice(at, 0, flat)
    this.selected = at
    this.selectedGroup = ''

    this.paint.colored.add(mask.paint)
    const before = new Uint8Array(data.length)
    this.paint.write(mask.paint, data)
    this.pushPaint(mask.paint, before, data)
    this.say('flattened into ' + flat.name)
  }

  // Where the selected layer's colors come from: its own color, its image stamps,
  // its paint, or the model's texture.
  colorSource(l: Layer): ColorSource {
    return l.source ? 'model' : l.paintColors ? 'paint' : l.stampColors ? 'stamps' : 'layer'
  }

  setColorSource(kind: ColorSource) {
    const l = this.layer
    if (!l) return
    if (l.paintColors && kind !== 'paint') this.setPaintColors(false)
    l.stampColors = kind === 'stamps'
    l.source = kind === 'model'
    if (kind === 'paint' && !l.paintColors) this.setPaintColors(true)
  }

  // Painted colors on or off for the selected layer; what is painted carries over,
  // in the layer's color when colors come on.
  setPaintColors(on: boolean) {
    const l = this.layer
    const paint = this.paint
    if (!l || !paint) return
    l.paintColors = on
    if (on) this.ensureMask('paint')
    const id = l.masks.find((m) => m.kind === 'paint')?.paint
    if (!id) return

    paint.get(id)
    const before = paint.read(id)
    const after = new Uint8Array(before)
    const rgb = new THREE.Color(l.color).convertLinearToSRGB()
    const color = [rgb.r, rgb.g, rgb.b].map((v) => v * 255)
    for (let i = 0; i < after.length; i += 4) {
      const c = on ? before[i] : before[i + 3]
      for (let k = 0; k < 3; k++) after[i + k] = on ? Math.round((color[k] * c) / 255) : c
      after[i + 3] = c
    }
    if (on) paint.colored.add(id)
    else paint.colored.delete(id)
    paint.write(id, after)
    this.pushPaint(id, before, after)
  }

  duplicateLayer() {
    const l = this.layer
    if (!l) return
    const copy = { ...this.copy(l), name: l.name + ' copy' }
    this.project.layers.splice(this.selected + 1, 0, copy)
    this.selected++
  }

  // The selected layer and the marked ones, in stack order.
  pickedLayers(): Layer[] {
    const ids = new Set(this.marked)
    if (this.layer) ids.add(this.layer.id)
    return this.project.layers.filter((l) => ids.has(l.id))
  }

  deleteLayer() {
    const ids = this.pickedLayers().map((l) => l.id)
    if (!ids.length) return
    const first = this.project.layers.findIndex((l) => ids.includes(l.id))
    this.project.layers = this.project.layers.filter((l) => !ids.includes(l.id))
    this.marked = []
    // The next layer down in the same channel, else the next up.
    const layers = this.project.layers
    let at = Math.min(first, layers.length) - 1
    while (at >= 0 && channelOf(layers[at]) !== this.channel) at--
    if (at < 0) at = layers.findIndex((l) => channelOf(l) === this.channel)
    this.selected = at
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
    const g: Group = { id: uid(), name: 'folder ' + (this.project.groups.length + 1), open: true, visible: true, zone: { parts: [], islands: [] } }
    this.project.groups.push(g)
    l.group = g.id
    this.dropEmptyGroups()
  }

  // Dissolves the selected layer's folder; its layers stay where they are.
  ungroup(id = this.layer?.group) {
    if (!id) return
    if (this.selectedGroup === id) this.selectedGroup = ''
    for (const l of this.project.layers) if (l.group === id) l.group = ''
    this.dropEmptyGroups()
  }

  // Removes a folder and its layers.
  deleteGroup(id: string) {
    const keep = this.project.layers.filter((l) => l.group !== id)
    const selectedId = this.layer?.id
    this.project.layers = keep
    this.selected = Math.max(0, keep.findIndex((l) => l.id === selectedId))
    this.marked = this.marked.filter((id) => keep.some((l) => l.id === id))
    if (this.selectedGroup === id) this.selectedGroup = ''
    this.dropEmptyGroups()
  }

  private dropEmptyGroups() {
    const used = new Set(this.project.layers.map((l) => l.group))
    this.project.groups = this.project.groups.filter((g) => used.has(g.id))
  }

  // Added to the selected layer, in place of an empty paint or stamps mask, which
  // would hide it as masks intersect.
  addMask(kind: MaskKind): number {
    const l = this.layer
    if (!l) return -1
    const at = l.masks.findIndex((m) => (m.kind === 'stamps' && !m.stamps.length) || (m.kind === 'paint' && this.paint?.empty(m.paint)))
    if (at >= 0) {
      l.masks[at] = this.newMask(kind)
      return at
    }
    if (l.masks.length >= 4) {
      this.say('this layer has 4 masks already')
      return -1
    }
    l.masks.push(this.newMask(kind))
    return l.masks.length - 1
  }

  // A volume starts in the middle of the model, a fifth of its size.
  private newMask(kind: MaskKind): Mask {
    const m = newMask(kind)
    const model = this.model
    if (kind === 'volume' && model) {
      m.volume.p = [0, 1, 2].map((k) => model.min[k] + (model.extent[k] * model.diag) / 2) as Vec3
      m.volume.size = [1, 1, 1].map(() => model.diag * 0.1) as Vec3
    }
    return m
  }

  // The selected layer's first mask of this kind, added if there is room.
  ensureMask(kind: MaskKind): number {
    const l = this.layer
    if (!l) return -1
    const at = l.masks.findIndex((m) => m.kind === kind)
    return at >= 0 ? at : this.addMask(kind)
  }

  // Clicks take whole parts, islands with ⇧; a model of one part has only islands.
  get onePart() {
    return (this.model?.parts.length ?? 0) <= 1
  }

  // Toggles a part or an island in the picked folder's zone while picking, else in the
  // selected layer's surfaces mask, adding the mask if needed.
  toggleSurface(kind: 'parts' | 'islands', id: number) {
    if (id < 0) return
    let target = this.pickingZone ? this.group(this.selectedGroup)?.zone : undefined
    if (!target) {
      const l = this.layer
      if (!l) return
      target = l.masks.find((m) => m.kind === 'surfaces')
      if (!target) {
        l.masks.unshift(newMask('surfaces'))
        target = l.masks[0]
      }
    }
    const list = target[kind]
    const i = list.indexOf(id)
    if (i >= 0) list.splice(i, 1)
    else list.push(id)
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

export type Snapshot = { color: string; emissive: string; rough: string; ao: string }

// What a new layer paints with in each channel.
const layerColors: Record<Channel, string> = { color: '#8a8578', ao: '#404040', rough: '#808080', glow: '#ffa040' }

export const store = new Store()
