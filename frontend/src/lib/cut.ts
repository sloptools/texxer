// Cuts from photos: four corners deskewed into a small image, then calmed towards
// texxer's look. Snippets are small, so this runs on the CPU, fast enough for sliders.

export type Corner = [number, number]

// px is the long side; calm flattens contrast, sat scales saturation, bright lifts.
// Mono keeps only brightness around mid gray, for the relief blend to tint. Width
// widens what the photo foreshortened.
export type CutLook = { px: number; calm: number; sat: number; bright: number; mono: boolean; width?: number }

// Corners in photo pixels: top left, top right, bottom right, bottom left. A round
// cut keeps the ellipse inside them, so a skewed one follows the perspective.
export type Cut = { id: string; name: string; corners: [Corner, Corner, Corner, Corner]; look: CutLook; aspect: number; round?: boolean }

export const defaultLook: CutLook = { px: 256, calm: 0.3, sat: 0.85, bright: 0, mono: false, width: 1 }

// A photo's pixels, halved again and again, so any cut samples a level near its size.
export type Levels = { width: number; height: number; data: ImageData[] }

export function levels(img: ImageBitmap | HTMLImageElement): Levels {
  const width = img.width
  const height = img.height
  const data: ImageData[] = []
  let w = width
  let h = height
  let src: CanvasImageSource = img
  while (true) {
    const canvas = document.createElement('canvas')
    canvas.width = w
    canvas.height = h
    const ctx = canvas.getContext('2d', { willReadFrequently: true })!
    ctx.imageSmoothingQuality = 'high'
    ctx.drawImage(src, 0, 0, w, h)
    data.push(ctx.getImageData(0, 0, w, h))
    if (w <= 64 || h <= 64) break
    src = canvas
    w = Math.max(1, w >> 1)
    h = Math.max(1, h >> 1)
  }
  return { width, height, data }
}

// Maps the unit square onto the quad (Heckbert), u to the right, v down.
function squareToQuad(q: Corner[]) {
  const [[x0, y0], [x1, y1], [x2, y2], [x3, y3]] = q
  const sx = x0 - x1 + x2 - x3
  const sy = y0 - y1 + y2 - y3
  let g = 0
  let h = 0
  if (Math.abs(sx) > 1e-9 || Math.abs(sy) > 1e-9) {
    const dx1 = x1 - x2
    const dx2 = x3 - x2
    const dy1 = y1 - y2
    const dy2 = y3 - y2
    const den = dx1 * dy2 - dx2 * dy1
    g = (sx * dy2 - dx2 * sy) / den
    h = (dx1 * sy - sx * dy1) / den
  }
  const a = x1 - x0 + g * x1
  const b = x3 - x0 + h * x3
  const d = y1 - y0 + g * y1
  const e = y3 - y0 + h * y3
  return (u: number, v: number): Corner => {
    const w = g * u + h * v + 1
    return [(a * u + b * v + x0) / w, (d * u + e * v + y0) / w]
  }
}

// The ellipse inside the quad, as points around it.
export function ellipse(q: Corner[], steps = 64): Corner[] {
  const map = squareToQuad(q)
  return Array.from({ length: steps }, (_, i) => {
    const a = (i / steps) * Math.PI * 2
    return map(0.5 + Math.cos(a) * 0.5, 0.5 + Math.sin(a) * 0.5)
  })
}

const dist = (p: Corner, q: Corner) => Math.hypot(p[0] - q[0], p[1] - q[1])

// Height over width of the cut, from its edges.
export function cutAspect(c: Cut['corners']): number {
  const w = (dist(c[0], c[1]) + dist(c[3], c[2])) / 2
  const h = (dist(c[0], c[3]) + dist(c[1], c[2])) / 2
  return w > 0 ? h / w : 1
}

// The snippet's height over width, as the photo shows it, widened.
export function aspectOf(cut: Cut): number {
  return cutAspect(cut.corners) / (cut.look.width ?? 1)
}

// Corner i dragged to p, the opposite one fixed: the quad stretches along its own
// edges, so a skewed one stays skewed.
export function stretchCorner(q: Cut['corners'], i: number, p: Corner): Cut['corners'] {
  const o = q[(i + 2) % 4]
  const a = q[(i + 1) % 4]
  const b = q[(i + 3) % 4]
  const ea = [a[0] - o[0], a[1] - o[1]]
  const eb = [b[0] - o[0], b[1] - o[1]]
  const det = ea[0] * eb[1] - ea[1] * eb[0]
  if (Math.abs(det) < 1e-6) return q

  const local = (c: Corner) => {
    const dx = c[0] - o[0]
    const dy = c[1] - o[1]
    return [(dx * eb[1] - dy * eb[0]) / det, (ea[0] * dy - ea[1] * dx) / det]
  }
  const [u0, v0] = local(q[i])
  const [u1, v1] = local(p)
  const su = Math.max(u1 / u0, 0.02)
  const sv = Math.max(v1 / v0, 0.02)
  return q.map((c) => {
    const [u, v] = local(c)
    return [o[0] + u * su * ea[0] + v * sv * eb[0], o[1] + u * su * ea[1] + v * sv * eb[1]]
  }) as Cut['corners']
}

// The edge from corner i to i + 1 pushed towards p, the opposite edge fixed.
export function stretchEdge(q: Cut['corners'], i: number, p: Corner): Cut['corners'] {
  const [a, b, c, d] = [0, 1, 2, 3].map((k) => q[(i + k) % 4])
  const near = [(a[0] + b[0]) / 2, (a[1] + b[1]) / 2]
  const far = [(c[0] + d[0]) / 2, (c[1] + d[1]) / 2]
  const dir = [near[0] - far[0], near[1] - far[1]]
  const len = dir[0] * dir[0] + dir[1] * dir[1]
  if (len < 1e-6) return q

  const t = Math.max(((p[0] - far[0]) * dir[0] + (p[1] - far[1]) * dir[1]) / len, 0.02)
  const out = [...q] as Cut['corners']
  out[i] = [d[0] + (a[0] - d[0]) * t, d[1] + (a[1] - d[1]) * t]
  out[(i + 1) % 4] = [c[0] + (b[0] - c[0]) * t, c[1] + (b[1] - c[1]) * t]
  return out
}

// Orders four clicked points as top left, top right, bottom right, bottom left.
export function orderCorners(points: Corner[]): Cut['corners'] {
  const cx = points.reduce((s, p) => s + p[0], 0) / 4
  const cy = points.reduce((s, p) => s + p[1], 0) / 4
  const around = [...points].sort((p, q) => Math.atan2(p[1] - cy, p[0] - cx) - Math.atan2(q[1] - cy, q[0] - cx))
  // Starting at the one nearest the top left.
  const first = around.reduce((best, p, i) => (p[0] + p[1] < around[best][0] + around[best][1] ? i : best), 0)
  return [0, 1, 2, 3].map((k) => around[(first + k) % 4]) as Cut['corners']
}

function sample(img: ImageData, x: number, y: number, out: number[]) {
  const w = img.width
  const h = img.height
  x = Math.min(Math.max(x - 0.5, 0), w - 1)
  y = Math.min(Math.max(y - 0.5, 0), h - 1)
  const x0 = Math.floor(x)
  const y0 = Math.floor(y)
  const x1 = Math.min(x0 + 1, w - 1)
  const y1 = Math.min(y0 + 1, h - 1)
  const fx = x - x0
  const fy = y - y0
  const d = img.data
  for (let c = 0; c < 3; c++) {
    const top = d[(y0 * w + x0) * 4 + c] * (1 - fx) + d[(y0 * w + x1) * 4 + c] * fx
    const bottom = d[(y1 * w + x0) * 4 + c] * (1 - fx) + d[(y1 * w + x1) * 4 + c] * fx
    out[c] += top * (1 - fy) + bottom * fy
  }
}

const sub = 4

export function render(photo: Levels, cut: Cut): ImageData {
  const aspect = aspectOf(cut)
  const px = Math.max(4, Math.round(cut.look.px))
  const ow = aspect <= 1 ? px : Math.max(1, Math.round(px / aspect))
  const oh = aspect <= 1 ? Math.max(1, Math.round(px * aspect)) : px

  // The level with about sub photo pixels per output pixel.
  const across = Math.max(dist(cut.corners[0], cut.corners[1]), dist(cut.corners[3], cut.corners[2])) / ow
  const level = Math.min(photo.data.length - 1, Math.max(0, Math.floor(Math.log2(across / sub))))
  const img = photo.data[level]
  const scale = img.width / photo.width
  const map = squareToQuad(cut.corners.map(([x, y]) => [x * scale, y * scale]))

  const rgb = new Float32Array(ow * oh * 3)
  const alpha = new Float32Array(ow * oh).fill(1)
  const acc = [0, 0, 0]
  for (let y = 0; y < oh; y++) {
    for (let x = 0; x < ow; x++) {
      acc[0] = acc[1] = acc[2] = 0
      let n = 0
      for (let j = 0; j < sub; j++) {
        for (let i = 0; i < sub; i++) {
          const u = (x + (i + 0.5) / sub) / ow
          const v = (y + (j + 0.5) / sub) / oh
          if (cut.round && (u - 0.5) ** 2 + (v - 0.5) ** 2 > 0.25) continue
          const [sx, sy] = map(u, v)
          sample(img, sx, sy, acc)
          n++
        }
      }
      const p = y * ow + x
      if (cut.round) alpha[p] = n / (sub * sub)
      for (let c = 0; c < 3; c++) rgb[p * 3 + c] = n ? acc[c] / (n * 255) : 0
    }
  }
  return style(rgb, alpha, ow, oh, cut.look)
}

const luma = (r: number, g: number, b: number) => 0.299 * r + 0.587 * g + 0.114 * b

function style(rgb: Float32Array, alpha: Float32Array, w: number, h: number, look: CutLook): ImageData {
  let mean = 0
  let covered = 0
  for (let p = 0; p < w * h; p++) {
    mean += luma(rgb[p * 3], rgb[p * 3 + 1], rgb[p * 3 + 2]) * alpha[p]
    covered += alpha[p]
  }
  mean /= covered || 1

  const out = new ImageData(w, h)
  for (let p = 0; p < w * h; p++) {
    let r = rgb[p * 3]
    let g = rgb[p * 3 + 1]
    let b = rgb[p * 3 + 2]
    const l = luma(r, g, b)
    const calmed = mean + (l - mean) * (1 - look.calm * 0.8) + look.bright * 0.5

    if (look.mono) r = g = b = 0.5 + calmed - mean
    else {
      // Saturation around the pixel's own brightness, then its brightness calmed.
      r = l + (r - l) * look.sat + calmed - l
      g = l + (g - l) * look.sat + calmed - l
      b = l + (b - l) * look.sat + calmed - l
    }
    out.data[p * 4] = Math.round(Math.min(Math.max(r, 0), 1) * 255)
    out.data[p * 4 + 1] = Math.round(Math.min(Math.max(g, 0), 1) * 255)
    out.data[p * 4 + 2] = Math.round(Math.min(Math.max(b, 0), 1) * 255)
    out.data[p * 4 + 3] = Math.round(alpha[p] * 255)
  }
  return out
}

export function pngBase64(img: ImageData): string {
  const canvas = document.createElement('canvas')
  canvas.width = img.width
  canvas.height = img.height
  canvas.getContext('2d')!.putImageData(img, 0, 0)
  return canvas.toDataURL('image/png').split(',')[1]
}
