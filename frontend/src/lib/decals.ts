// Built-in decals, drawn into a 100×100 box.
// Relief decals are gray: 50% leaves the surface as it is under the relief
// blend, darker reads as grooves and shadows, lighter as lit edges. Color
// decals carry their own colors, for layers that take colors from stamps.

type Ctx = CanvasRenderingContext2D

const deep = '#161616'
const dark = '#343434'
const light = '#c8c8c8'
const bright = '#e6e6e6'

function fill(c: Ctx, color: string, draw: () => void) {
  c.fillStyle = color
  c.beginPath()
  draw()
  c.fill()
}

function stroke(c: Ctx, color: string, width: number, draw: () => void) {
  c.strokeStyle = color
  c.lineWidth = width
  c.beginPath()
  draw()
  c.stroke()
}

function rivet(c: Ctx, x: number, y: number, r: number) {
  fill(c, dark, () => c.arc(x + r * 0.35, y + r * 0.35, r, 0, Math.PI * 2))
  fill(c, light, () => c.arc(x, y, r, 0, Math.PI * 2))
  fill(c, bright, () => c.arc(x - r * 0.3, y - r * 0.3, r * 0.4, 0, Math.PI * 2))
}

// A recessed line: dark groove with a lit lip below and right of it.
function groove(c: Ctx, width: number, draw: () => void) {
  c.save()
  c.translate(width * 0.6, width * 0.6)
  stroke(c, light, width * 0.6, draw)
  c.restore()
  stroke(c, dark, width, draw)
}

function poly(c: Ctx, ...pts: number[]) {
  for (let i = 0; i < pts.length; i += 2) c.lineTo(pts[i], pts[i + 1])
  c.closePath()
}

function hexagon(c: Ctx, x: number, y: number, r: number) {
  for (let i = 0; i < 6; i++) c.lineTo(x + Math.cos((i / 6) * Math.PI * 2) * r, y + Math.sin((i / 6) * Math.PI * 2) * r)
  c.closePath()
}

const relief: Record<string, (c: Ctx) => void> = {
  hatch: (c) => {
    groove(c, 6, () => c.roundRect(10, 10, 78, 78, 10))
    fill(c, deep, () => c.rect(36, 72, 26, 6))
    for (const [x, y] of [
      [20, 20],
      [78, 20],
      [20, 78],
      [78, 78],
    ])
      rivet(c, x, y, 4)
  },
  hatch_round: (c) => {
    groove(c, 6, () => c.arc(50, 50, 34, 0, Math.PI * 2))
    for (let i = 0; i < 8; i++) {
      const a = (i / 8) * Math.PI * 2
      rivet(c, 50 + Math.cos(a) * 43, 50 + Math.sin(a) * 43, 4)
    }
    fill(c, dark, () => c.rect(44, 46, 12, 8))
  },
  panel: (c) => {
    groove(c, 5, () => c.rect(5, 5, 88, 88))
    for (let i = 0; i <= 4; i++) {
      const t = 14 + i * 18
      rivet(c, t, 14, 3)
      rivet(c, t, 86, 3)
      if (i > 0 && i < 4) {
        rivet(c, 14, t, 3)
        rivet(c, 86, t, 3)
      }
    }
  },
  seam: (c) => {
    groove(c, 6, () => {
      c.moveTo(0, 50)
      c.lineTo(100, 50)
    })
    for (let x = 10; x < 100; x += 20) rivet(c, x, 34, 4)
  },
  rivets: (c) => {
    for (let x = 10; x < 100; x += 20) rivet(c, x, 50, 5)
  },
  rivet: (c) => rivet(c, 46, 46, 30),
  bolt: (c) => {
    fill(c, dark, () => hexagon(c, 56, 56, 34))
    fill(c, light, () => hexagon(c, 48, 48, 34))
    fill(c, dark, () => c.arc(48, 48, 12, 0, Math.PI * 2))
  },
  vent: (c) => {
    fill(c, dark, () => c.rect(4, 4, 92, 92))
    fill(c, '#808080', () => c.rect(8, 8, 84, 84))
    for (let y = 14; y < 88; y += 15) {
      fill(c, deep, () => c.rect(14, y, 72, 7))
      fill(c, light, () => c.rect(14, y + 7, 72, 3))
    }
  },
  grille: (c) => {
    groove(c, 5, () => c.roundRect(6, 6, 86, 86, 8))
    for (let y = 18; y < 86; y += 13) for (let x = 18; x < 86; x += 13) fill(c, deep, () => c.arc(x, y, 4, 0, Math.PI * 2))
  },
  weld: (c) => {
    for (let x = 4; x < 100; x += 9) {
      fill(c, dark, () => c.ellipse(x + 2, 53, 7, 9, 0.5, 0, Math.PI * 2))
      fill(c, light, () => c.ellipse(x, 50, 6, 8, 0.5, 0, Math.PI * 2))
    }
  },
  patch: (c) => {
    fill(c, dark, () => c.rect(8, 8, 88, 88))
    fill(c, '#9a9a9a', () => c.rect(4, 4, 88, 88))
    for (const [x, y] of [
      [14, 14],
      [82, 14],
      [14, 82],
      [82, 82],
      [48, 14],
      [48, 82],
    ])
      rivet(c, x, y, 4)
  },
  handle: (c) => {
    const u = () => {
      c.moveTo(18, 30)
      c.lineTo(18, 70)
      c.lineTo(82, 70)
      c.lineTo(82, 30)
    }
    c.save()
    c.translate(5, 5)
    stroke(c, deep, 10, u)
    c.restore()
    stroke(c, light, 10, u)
  },
  hinge: (c) => {
    for (const y of [10, 60]) {
      fill(c, dark, () => c.rect(14, y + 4, 76, 30))
      fill(c, light, () => c.rect(10, y, 76, 30))
      rivet(c, 26, y + 15, 4)
      rivet(c, 70, y + 15, 4)
    }
    fill(c, deep, () => c.rect(44, 6, 12, 90))
  },
  dent: (c) => {
    fill(c, dark, () => c.ellipse(46, 46, 40, 28, 0, Math.PI, Math.PI * 2))
    fill(c, light, () => c.ellipse(54, 54, 40, 28, 0, 0, Math.PI))
    fill(c, '#808080', () => c.ellipse(50, 50, 32, 20, 0, 0, Math.PI * 2))
  },
  strap: (c) => {
    fill(c, deep, () => c.rect(34, 0, 36, 100))
    fill(c, dark, () => c.rect(30, 0, 36, 100))
    fill(c, '#4a4a4a', () => c.rect(34, 0, 28, 100))
    fill(c, light, () => c.rect(26, 40, 44, 22))
    fill(c, dark, () => c.rect(34, 46, 28, 10))
  },
}

const yellow = '#d6a020'
const black = '#16140f'
const white = '#e8e4d8'

const color: Record<string, (c: Ctx) => void> = {
  hazard: (c) => {
    fill(c, yellow, () => c.rect(0, 0, 100, 100))
    c.save()
    c.beginPath()
    c.rect(0, 0, 100, 100)
    c.clip()
    for (let i = -100; i < 200; i += 28) fill(c, black, () => poly(c, i, 100, i + 14, 100, i + 114, 0, i + 100, 0))
    c.restore()
  },
  warning: (c) => {
    fill(c, black, () => poly(c, 50, 2, 98, 94, 2, 94))
    fill(c, yellow, () => poly(c, 50, 16, 86, 86, 14, 86))
    fill(c, black, () => c.rect(45, 38, 10, 26))
    fill(c, black, () => c.rect(45, 70, 10, 9))
  },
  plate_red: (c) => {
    fill(c, white, () => c.rect(0, 0, 100, 100))
    fill(c, '#a82a20', () => c.rect(7, 7, 86, 86))
  },
  plate_white: (c) => {
    fill(c, black, () => c.rect(0, 0, 100, 100))
    fill(c, white, () => c.rect(6, 6, 88, 88))
  },
  logo_wave: (c) => {
    fill(c, white, () => c.arc(50, 50, 48, 0, Math.PI * 2))
    fill(c, '#1d5d8c', () => c.arc(50, 50, 40, 0, Math.PI * 2))
    stroke(c, white, 9, () => {
      c.moveTo(18, 58)
      c.bezierCurveTo(34, 30, 52, 30, 58, 50)
      c.bezierCurveTo(62, 64, 76, 64, 84, 46)
    })
  },
  stripe_white: (c) => fill(c, white, () => c.rect(0, 0, 100, 100)),
  stripe_red: (c) => fill(c, '#a82a20', () => c.rect(0, 0, 100, 100)),
}

export const reliefDecals = Object.keys(relief).map((n) => 'relief:' + n)
export const colorDecals = Object.keys(color).map((n) => 'decal:' + n)

export function isDecal(shape: string) {
  return shape.startsWith('relief:') || shape.startsWith('decal:')
}

// Draws a decal filling the box at x, y with the given size.
export function drawDecal(ctx: Ctx, shape: string, x: number, y: number, size: number) {
  const [kind, name] = shape.split(':')
  const draw = (kind === 'relief' ? relief : color)[name]
  if (!draw) return
  ctx.save()
  ctx.translate(x, y)
  ctx.scale(size / 100, size / 100)
  draw(ctx)
  ctx.restore()
}
