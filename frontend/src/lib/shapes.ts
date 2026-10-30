import { drawPixelText, pixelColumns } from './pixelfont'

// Stamp shapes, drawn white on transparent into a square canvas cell.
// Text stamps are "text:…" (condensed font) or "pixel:…" (bitmap font), stretched
// to fill the cell; their stamps get the text's aspect back. Images are "img:…".

export const shapes = [
  'circle',
  'ring',
  'square',
  'frame',
  'triangle',
  'warning',
  'stripes',
  'chevron',
  'arrow',
  'cross',
  'plus',
  'vent',
  'rivets',
  'bar',
  'grate',
  'splat',
] as const

const font = '"DIN Condensed", "Arial Narrow", "Helvetica Neue", sans-serif'

export function drawShape(ctx: CanvasRenderingContext2D, shape: string, x: number, y: number, size: number) {
  ctx.save()
  ctx.translate(x, y)
  ctx.scale(size / 100, size / 100)
  ctx.fillStyle = '#fff'
  ctx.strokeStyle = '#fff'

  if (shape.startsWith('text:')) {
    text(ctx, shape.slice(5))
    ctx.restore()
    return
  }
  if (shape.startsWith('pixel:')) {
    drawPixelText(ctx, shape.slice(6), 0, 0, 100, 100)
    ctx.restore()
    return
  }

  const poly = (...pts: number[]) => {
    ctx.beginPath()
    for (let i = 0; i < pts.length; i += 2) ctx.lineTo(pts[i], pts[i + 1])
    ctx.closePath()
    ctx.fill()
  }

  switch (shape) {
    case 'circle':
      ctx.beginPath()
      ctx.arc(50, 50, 46, 0, Math.PI * 2)
      ctx.fill()
      break
    case 'ring':
      ctx.lineWidth = 14
      ctx.beginPath()
      ctx.arc(50, 50, 39, 0, Math.PI * 2)
      ctx.stroke()
      break
    case 'square':
      ctx.fillRect(4, 4, 92, 92)
      break
    case 'frame':
      ctx.lineWidth = 12
      ctx.strokeRect(10, 10, 80, 80)
      break
    case 'triangle':
      poly(50, 6, 96, 92, 4, 92)
      break
    case 'warning':
      poly(50, 4, 98, 94, 2, 94)
      ctx.globalCompositeOperation = 'destination-out'
      poly(50, 24, 81, 82, 19, 82)
      ctx.globalCompositeOperation = 'source-over'
      ctx.fillRect(45, 40, 10, 24)
      ctx.fillRect(45, 69, 10, 9)
      break
    case 'stripes':
      ctx.beginPath()
      ctx.rect(0, 0, 100, 100)
      ctx.clip()
      for (let i = -100; i < 200; i += 28) poly(i, 100, i + 14, 100, i + 114, 0, i + 100, 0)
      break
    case 'chevron':
      poly(10, 30, 50, 62, 90, 30, 90, 52, 50, 84, 10, 52)
      poly(10, 4, 50, 36, 90, 4, 90, 22, 50, 54, 10, 22)
      break
    case 'arrow':
      poly(50, 4, 94, 50, 66, 50, 66, 96, 34, 96, 34, 50, 6, 50)
      break
    case 'cross':
      ctx.translate(50, 50)
      ctx.rotate(Math.PI / 4)
      ctx.fillRect(-9, -50, 18, 100)
      ctx.fillRect(-50, -9, 100, 18)
      break
    case 'plus':
      ctx.fillRect(38, 6, 24, 88)
      ctx.fillRect(6, 38, 88, 24)
      break
    case 'vent':
      for (let i = 0; i < 5; i++) ctx.fillRect(8, 10 + i * 18, 84, 9)
      break
    case 'rivets':
      for (const [cx, cy] of [
        [16, 16],
        [84, 16],
        [16, 84],
        [84, 84],
      ]) {
        ctx.beginPath()
        ctx.arc(cx, cy, 10, 0, Math.PI * 2)
        ctx.fill()
      }
      break
    case 'bar':
      ctx.fillRect(2, 38, 96, 24)
      break
    case 'grate':
      ctx.lineWidth = 7
      ctx.strokeRect(6, 6, 88, 88)
      for (let i = 1; i < 4; i++) {
        ctx.fillRect(6 + i * 22 - 3, 6, 6, 88)
        ctx.fillRect(6, 6 + i * 22 - 3, 88, 6)
      }
      break
    case 'splat': {
      ctx.beginPath()
      for (let i = 0; i <= 24; i++) {
        const a = (i / 24) * Math.PI * 2
        const r = 30 + 14 * Math.sin(i * 2.7) * Math.cos(i * 1.3)
        ctx.lineTo(50 + Math.cos(a) * r, 50 + Math.sin(a) * r)
      }
      ctx.fill()
      for (const [cx, cy, r] of [
        [86, 30, 6],
        [16, 76, 5],
        [78, 84, 4],
        [20, 18, 4],
      ]) {
        ctx.beginPath()
        ctx.arc(cx, cy, r, 0, Math.PI * 2)
        ctx.fill()
      }
      break
    }
  }
  ctx.restore()
}

function measure(ctx: CanvasRenderingContext2D, value: string) {
  ctx.font = `700 80px ${font}`
  const m = ctx.measureText(value)
  return { w: Math.max(m.width, 1), up: m.actualBoundingBoxAscent, h: Math.max(m.actualBoundingBoxAscent + m.actualBoundingBoxDescent, 1) }
}

// Condensed stencil-ish text, stretched to fill the cell.
function text(ctx: CanvasRenderingContext2D, value: string) {
  const m = measure(ctx, value)
  ctx.scale(100 / m.w, 100 / m.h)
  ctx.textBaseline = 'alphabetic'
  ctx.fillText(value, 0, m.up)
}

// Height over width of a shape's artwork; images report theirs when loaded.
export function shapeAspect(shape: string): number | undefined {
  if (shape.startsWith('pixel:')) return 7 / pixelColumns(shape.slice(6))
  if (shape.startsWith('text:')) {
    const m = measure(document.createElement('canvas').getContext('2d')!, shape.slice(5))
    return m.h / m.w
  }
}
