import * as api from '../../wailsjs/go/main/App'
import type { main } from '../../wailsjs/go/models'
import type { PaintStore } from './gl/paint'
import { newLayer, newMask, newProject, normalize, uid } from './project'
import type { Finish, Group, Layer, MaskKind, Project, View } from './types'

function load<T>(key: string, fallback: T): T {
  try {
    const v = localStorage.getItem('texxer.' + key)
    return v === null ? fallback : JSON.parse(v)
  } catch {
    return fallback
  }
}

function persist(key: string, value: unknown) {
  try {
    localStorage.setItem('texxer.' + key, JSON.stringify(value))
  } catch {}
}

// Project edits are JSON snapshots; strokes keep the mask's pixels before and after.
type Entry = { kind: 'project'; before: string; after: string } | { kind: 'paint'; id: string; before: Uint8Array; after: Uint8Array }

class Store {
  model = $state<main.ModelInfo | null>(null)
  project = $state<Project>(newProject())
  selected = $state(0)
  view = $state<View>('lit')
  bloom = $state(true)
  baking = $state(false)
  hoverPart = $state(-1)
  help = $state(false)
  message = $state('')
  recent = $state<string[]>(load('recent', []))
  grungeMaps = $state<string[]>([])
  stampImages = $state<string[]>([])

  // Set by the viewport: renders the finished texture as a base64 PNG.
  snapshot: (() => string) | null = null
  paint: PaintStore | null = null

  private past: Entry[] = []
  private future: Entry[] = []
  private last = ''
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

        if (!this.model) return
        clearTimeout(this.saveTimer)
        const path = this.model.path
        this.saveTimer = setTimeout(() => api.Save(path, JSON.stringify(this.project, null, 2)).catch((e) => this.say(String(e))), 600)
      })
    })
  }

  say(text: string) {
    this.message = text
    clearTimeout(this.messageTimer)
    this.messageTimer = setTimeout(() => (this.message = ''), 4000)
  }

  async pickModel() {
    const path = await api.PickModel()
    if (path) await this.open(path)
  }

  async open(path: string) {
    try {
      const info = await api.Open(path)
      this.project = info.project ? normalize(JSON.parse(info.project)) : newProject()
      this.model = info
      // Baked light already shades the texture, like in the game.
      this.view = this.project.shading.on ? 'unlit' : 'lit'
      this.selected = this.project.layers.length - 1
      this.past = []
      this.future = []
      this.last = JSON.stringify(this.project)
      this.recent = [path, ...this.recent.filter((p) => p !== path)].slice(0, 8)
      persist('recent', this.recent)
    } catch (e) {
      this.say(String(e))
    }
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
    this.last = e[side]
    this.project = JSON.parse(e[side])
    this.selected = Math.min(this.selected, this.project.layers.length - 1)
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

  async export(ask: boolean) {
    if (!this.model || !this.snapshot) return
    const key = 'export:' + this.model.path
    const target = ask ? '' : load(key, '')
    try {
      const path = await api.Export(this.model.path, this.snapshot(), target)
      if (!path) return
      persist(key, path)
      this.say('exported ' + path.split('/').pop())
    } catch (e) {
      this.say(String(e))
    }
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
