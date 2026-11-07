import type { RGB } from './gl/mediancut'

// Hue (0 to 1), saturation and lightness, as the layer shader converts them.
function hsl([r, g, b]: RGB): RGB {
  const hi = Math.max(r, g, b)
  const lo = Math.min(r, g, b)
  const l = (hi + lo) / 2
  const d = hi - lo
  if (d < 1e-5) return [0, 0, l]
  const h = hi === r ? (((g - b) / d) % 6 + 6) % 6 : hi === g ? (b - r) / d + 2 : (r - g) / d + 4
  return [h / 6, d / (1 - Math.abs(2 * l - 1)), l]
}

export function hexRGB(hex: string): RGB {
  const n = parseInt(hex.slice(1), 16)
  return [(n >> 16) / 255, ((n >> 8) & 255) / 255, (n & 255) / 255]
}

// The adjustments that turn one color into another; gray colors keep their hue.
export function matchAdjust(from: RGB, to: RGB) {
  const [h0, s0, l0] = hsl(from)
  const [h1, s1, l1] = hsl(to)
  let hue = s0 > 0.02 && s1 > 0.02 ? Math.round((h1 - h0) * 360) : 0
  if (hue > 180) hue -= 360
  if (hue < -180) hue += 360

  return {
    hue,
    saturation: s0 > 0.02 ? Math.min(2, s1 / s0) : 1,
    brightness: Math.max(-0.5, Math.min(0.5, l1 - l0)),
  }
}
