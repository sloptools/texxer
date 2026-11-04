import * as THREE from 'three'
import { drawDecal, isDecal } from '../decals'
import { fetchBlob, track } from '../assets'
import { drawShape } from '../shapes'
import { libraryURL, snippetChanged } from '../snippets.svelte'

const grid = 8
const cell = 256
const pad = 12

// Every stamp shape in use gets a cell; cells are handed out as shapes show up.
export class StampAtlas {
  readonly texture: THREE.CanvasTexture
  private ctx: CanvasRenderingContext2D
  private cells = new Map<string, number>()
  onLoad: () => void = () => {}
  private redraw = (e: Event) => this.draw('snip:' + (e as CustomEvent<string>).detail)

  constructor() {
    const canvas = document.createElement('canvas')
    canvas.width = canvas.height = grid * cell
    this.ctx = canvas.getContext('2d')!
    this.texture = new THREE.CanvasTexture(canvas)
    this.texture.flipY = false
    this.texture.generateMipmaps = false
    this.texture.minFilter = this.texture.magFilter = THREE.LinearFilter
    snippetChanged.addEventListener('change', this.redraw)
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
    else if (shape.startsWith('snip:')) track(this.drawImage(libraryURL('snippet', shape.slice(5), '', Date.now()), x, y, size, false))
    else if (isDecal(shape)) drawDecal(this.ctx, shape, x, y, size)
    else drawShape(this.ctx, shape, x, y, size)
    this.texture.needsUpdate = true
  }

  // Images keep their colors. Opaque gray images, like scanned stencils, use
  // their brightness as the shape; snippets stay as they are.
  private async drawImage(url: string, x: number, y: number, size: number, stencil: boolean) {
    const blob = await fetchBlob(url)
    if (!blob) return
    const img = await createImageBitmap(blob)
    this.ctx.drawImage(img, x, y, size, size)
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
    this.texture.dispose()
  }
}

export const atlasGrid = grid
export const atlasInset = pad / cell
