import * as THREE from 'three'
import { drawDecal, isDecal } from '../decals'
import { fetchBlob, track } from '../assets'
import { fontLoaded, ready } from '../fonts.svelte'
import { drawShape, parseFont } from '../shapes'
import { libraryURL, snipTiles, snippetChanged } from '../snippets.svelte'

const grid = 8
const cell = 256
const pad = 12

// Every stamp shape in use gets a cell; cells are handed out as shapes show up.
export class StampAtlas {
  readonly texture: THREE.CanvasTexture
  private ctx: CanvasRenderingContext2D
  private cells = new Map<string, number>()
  onLoad: () => void = () => {}
  private redraw = (e: Event) => {
    const id = (e as CustomEvent<string>).detail
    for (const shape of this.cells.keys()) if (shape.startsWith('snip:') && snipTiles(shape).id === id) this.draw(shape)
  }
  private redrawFont = (e: Event) => {
    const name = (e as CustomEvent<string>).detail
    for (const shape of this.cells.keys()) if (shape.startsWith('font:') && parseFont(shape).name === name) this.draw(shape)
    this.onLoad()
  }

  constructor() {
    const canvas = document.createElement('canvas')
    canvas.width = canvas.height = grid * cell
    this.ctx = canvas.getContext('2d')!
    this.texture = new THREE.CanvasTexture(canvas)
    this.texture.flipY = false
    this.texture.generateMipmaps = false
    this.texture.minFilter = this.texture.magFilter = THREE.LinearFilter
    snippetChanged.addEventListener('change', this.redraw)
    fontLoaded.addEventListener('load', this.redrawFont)
  }

  cell(shape: string): number {
    let i = this.cells.get(shape)
    if (i !== undefined) return i

    // Full: start over, the next composite fills it again.
    if (this.cells.size >= grid * grid) {
      this.cells.clear()
      this.ctx.clearRect(0, 0, grid * cell, grid * cell)
    }
    i = this.cells.size
    this.cells.set(shape, i)
    this.draw(shape)
    return i
  }

  // A shape that has a cell, again, as when a snippet changed.
  private draw(shape: string) {
    const i = this.cells.get(shape)
    if (i === undefined) return
    const x = (i % grid) * cell + pad
    const y = Math.floor(i / grid) * cell + pad
    const size = cell - pad * 2
    this.ctx.clearRect(x - pad, y - pad, cell, cell)
    if (shape.startsWith('img:')) track(this.drawImage(`/stamp?name=${encodeURIComponent(shape.slice(4))}`, x, y, size, true))
    else if (shape.startsWith('snip:')) {
      const t = snipTiles(shape)
      track(this.drawImage(libraryURL('snippet', t.id, '', Date.now()), x, y, size, false, t.cols, t.rows))
    }
    else if (isDecal(shape)) drawDecal(this.ctx, shape, x, y, size)
    else {
      // Drawn now in what is there; a font still loading draws it again.
      if (shape.startsWith('font:')) track(ready(parseFont(shape).name))
      drawShape(this.ctx, shape, x, y, size)
    }
    this.texture.needsUpdate = true
  }

  // Images keep their colors. Opaque gray images, like scanned stencils, use
  // their brightness as the shape; snippets stay as they are.
  private async drawImage(url: string, x: number, y: number, size: number, stencil: boolean, cols = 1, rows = 1) {
    const blob = await fetchBlob(url)
    if (!blob) return
    const img = await createImageBitmap(blob)
    const w = size / cols
    const h = size / rows
    for (let j = 0; j < rows; j++) for (let i = 0; i < cols; i++) this.ctx.drawImage(img, x + i * w, y + j * h, w, h)
    if (!stencil) {
      this.texture.needsUpdate = true
      this.onLoad()
      return
    }

    const data = this.ctx.getImageData(x, y, size, size)
    const px = data.data
    let opaque = true
    let gray = true
    for (let i = 0; i < px.length; i += 4) {
      if (px[i + 3] < 250) opaque = false
      if (Math.max(px[i], px[i + 1], px[i + 2]) - Math.min(px[i], px[i + 1], px[i + 2]) > 12) gray = false
    }
    if (opaque && gray) {
      for (let i = 0; i < px.length; i += 4) px[i + 3] = 0.299 * px[i] + 0.587 * px[i + 1] + 0.114 * px[i + 2]
      this.ctx.putImageData(data, x, y)
    }
    this.texture.needsUpdate = true
    this.onLoad()
  }

  dispose() {
    snippetChanged.removeEventListener('change', this.redraw)
    fontLoaded.removeEventListener('load', this.redrawFont)
    this.texture.dispose()
  }
}

export const atlasGrid = grid
export const atlasInset = pad / cell
