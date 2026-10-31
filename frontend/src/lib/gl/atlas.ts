import * as THREE from 'three'
import { drawDecal, isDecal } from '../decals'
import { fetchBlob, track } from '../assets'
import { drawShape } from '../shapes'

const grid = 8
const cell = 256
const pad = 12

// Every stamp shape in use gets a cell; cells are handed out as shapes show up.
export class StampAtlas {
  readonly texture: THREE.CanvasTexture
  private ctx: CanvasRenderingContext2D
  private cells = new Map<string, number>()
  onLoad: () => void = () => {}

  constructor() {
    const canvas = document.createElement('canvas')
    canvas.width = canvas.height = grid * cell
    this.ctx = canvas.getContext('2d')!
    this.texture = new THREE.CanvasTexture(canvas)
    this.texture.flipY = false
    this.texture.generateMipmaps = false
    this.texture.minFilter = this.texture.magFilter = THREE.LinearFilter
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
    const x = (i % grid) * cell
    const y = Math.floor(i / grid) * cell
    this.ctx.clearRect(x, y, cell, cell)
    if (shape.startsWith('img:')) track(this.drawImage(shape.slice(4), x + pad, y + pad, cell - pad * 2))
    else if (isDecal(shape)) drawDecal(this.ctx, shape, x + pad, y + pad, cell - pad * 2)
    else drawShape(this.ctx, shape, x + pad, y + pad, cell - pad * 2)
    this.texture.needsUpdate = true
    return i
  }

  // Images keep their colors. Opaque gray images, like scanned stencils, use
  // their brightness as the shape.
  private async drawImage(name: string, x: number, y: number, size: number) {
    const blob = await fetchBlob(`/stamp?name=${encodeURIComponent(name)}`)
    if (!blob) return
    const img = await createImageBitmap(blob)
    this.ctx.drawImage(img, x, y, size, size)

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
    this.texture.dispose()
  }
}

export const atlasGrid = grid
export const atlasInset = pad / cell
