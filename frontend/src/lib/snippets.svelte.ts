import * as api from '../../wailsjs/go/main/App'
import { aspectOf, cutAspect, defaultLook, levels, pngBase64, render, type Corner, type Cut, type Levels } from './cut'
import { newLayer, newMask, uid } from './project'
import { store } from './store.svelte'
import { tools } from './tools.svelte'

export type PhotoEntry = { id: string; file: string; cuts: Cut[] }

// The stamp atlas listens, to redraw a snippet that changed.
export const snippetChanged = new EventTarget()

// A library's file: a photo by file name, or a snippet by id.
export function libraryURL(kind: 'photo' | 'snippet', name: string, category: string, version = 0) {
  const q = new URLSearchParams({ project: store.path, category, name, v: String(version) })
  return `/${kind}?${q}`
}

// Snippet stamps can repeat their snippet, `snip:<id>*<cols>x<rows>`.
export function snipTiles(shape: string) {
  const [id, tiles] = shape.slice(5).split('*')
  const [cols, rows] = (tiles ?? '1x1').split('x').map(Number)
  return { id, cols: cols || 1, rows: rows || 1 }
}

// The snippet shape with cols across and as many rows as keep its tiles square to it.
export function tiled(shape: string, cols: number, stampAspect: number): string {
  const { id } = snipTiles(shape)
  const aspect = snippets.all.find((c) => c.id === id)?.aspect ?? 1
  const rows = Math.max(1, Math.round((cols * stampAspect) / aspect))
  return cols <= 1 && rows <= 1 ? 'snip:' + id : `snip:${id}*${cols}x${rows}`
}

// Photos to cut snippets from, in the project's library or a global category.
class Snippets {
  // '' is the project's.
  category = $state('')
  categories = $state<string[]>([])
  photos = $state<PhotoEntry[]>([])
  photo = $state('')
  cut = $state('')
  // Bumped per snippet when its PNG is written, so images load anew.
  versions = $state<Record<string, number>>({})

  private levels = new Map<string, Promise<Levels>>()
  private timers = new Map<string, number>()

  get current() {
    return this.photos.find((p) => p.id === this.photo)
  }

  get selectedCut() {
    return this.current?.cuts.find((c) => c.id === this.cut)
  }

  async load(category = this.category) {
    if (!store.path) return
    try {
      const [categories, photos] = await Promise.all([api.SnippetCategories(), api.Photos(store.path, category)])
      this.categories = categories
      this.category = category
      this.photos = photos.map((p) => ({ id: p.id, file: p.file, cuts: p.cuts ? (JSON.parse(p.cuts) as Cut[]) : [] }))
      if (!this.current) this.photo = this.photos[0]?.id ?? ''
      this.cut = ''
    } catch (e) {
      store.say(String(e))
    }
  }

  async addCategory(name: string) {
    try {
      const made = await api.AddCategory(name)
      await this.load(made)
    } catch (e) {
      store.say(String(e))
    }
  }

  // Without paths, asks for them.
  async addPhotos(paths: string[] = []) {
    if (!store.path) return
    try {
      const ids = (paths.length ? paths : Array.from({ length: 64 })).map(() => uid())
      const added = await api.AddPhotos(store.path, this.category, paths, ids)
      if (!added?.length) return
      await this.load()
      this.photo = added[0]
      store.say(added.length === 1 ? 'added a photo' : `added ${added.length} photos`)
    } catch (e) {
      store.say(String(e))
    }
  }

  async deletePhoto(photo: PhotoEntry) {
    try {
      await api.DeletePhoto(store.path, this.category, photo.id, photo.file, photo.cuts.map((c) => c.id))
      if (this.photo === photo.id) this.photo = ''
      await this.load()
    } catch (e) {
      store.say(String(e))
    }
  }

  // The photo's pixels, read once.
  pixels(photo: PhotoEntry): Promise<Levels> {
    const key = this.category + '/' + photo.file
    let p = this.levels.get(key)
    if (!p) {
      p = new Promise<Levels>((resolve, reject) => {
        const img = new Image()
        img.onload = () => resolve(levels(img))
        img.onerror = () => reject(new Error('could not read ' + photo.file))
        img.src = libraryURL('photo', photo.file, this.category)
      })
      this.levels.set(key, p)
    }
    return p
  }

  addCut(photo: PhotoEntry, corners: Cut['corners']) {
    const cut: Cut = { id: uid(), name: 'snippet ' + (photo.cuts.length + 1), corners, look: { ...defaultLook }, aspect: cutAspect(corners) }
    photo.cuts.push(cut)
    this.cut = cut.id
    this.changed(photo, cut)
  }

  deleteCut(photo: PhotoEntry, cut: Cut) {
    photo.cuts.splice(photo.cuts.indexOf(cut), 1)
    if (this.cut === cut.id) this.cut = ''
    api.DeleteSnippet(store.path, this.category, cut.id).catch(() => {})
    this.saveCuts(photo)
  }

  moveCorner(photo: PhotoEntry, cut: Cut, i: number, to: Corner) {
    cut.corners[i] = to
    this.changed(photo, cut)
  }

  // Writes the snippet and the photo's cuts once edits pause.
  changed(photo: PhotoEntry, cut: Cut) {
    const category = this.category
    const aspect = aspectOf(cut)
    if (cut.aspect !== aspect) cut.aspect = aspect
    clearTimeout(this.timers.get(cut.id))
    this.timers.set(
      cut.id,
      setTimeout(async () => {
        try {
          const img = render(await this.pixels(photo), $state.snapshot(cut) as Cut)
          await api.SaveSnippet(store.path, category, cut.id, pngBase64(img))
          await this.saveCuts(photo, category)
          this.versions[cut.id] = (this.versions[cut.id] ?? 0) + 1
          if (!category) snippetChanged.dispatchEvent(new CustomEvent('change', { detail: cut.id }))
        } catch (e) {
          store.say(String(e))
        }
      }, 250),
    )
  }

  private saveCuts(photo: PhotoEntry, category = this.category) {
    return api.SaveCuts(store.path, category, photo.id, JSON.stringify($state.snapshot(photo.cuts)))
  }

  get all(): Cut[] {
    return this.photos.flatMap((p) => p.cuts)
  }

  // Stamps the snippet: a layer that takes its colors from stamps, relief for mono
  // ones, and the stamp tool with the snippet as its shape.
  async pick(cut: Cut) {
    try {
      await api.UseSnippet(store.path, this.category, cut.id)
    } catch (e) {
      store.say(String(e))
      return
    }
    const blend = cut.look.mono ? 'relief' : 'normal'
    const l = store.layer
    if (!l || !l.stampColors || l.blend !== blend) {
      store.addLayer(newLayer(cut.look.mono ? 'photo relief' : 'photo', '#808080', [newMask('stamps')], { stampColors: true, blend, tone: 0, grain: 0 }))
    }
    tools.setShape('snip:' + cut.id, cut.aspect)
    if (tools.tool !== 'stamp') tools.set('stamp')
  }
}

export const snippets = new Snippets()
