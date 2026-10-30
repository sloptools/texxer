export type RGB = [number, number, number]

// Picks up to n representative colors: splits the box with the widest channel
// at its median until there are n boxes, then takes each box's mean.
export function medianCut(pixels: RGB[], n: number): RGB[] {
  const boxes: RGB[][] = [pixels]

  while (boxes.length < n) {
    let best = -1
    let channel = 0
    let widest = 0
    boxes.forEach((b, i) => {
      if (b.length < 2) return
      for (let c = 0; c < 3; c++) {
        let lo = 255
        let hi = 0
        for (const p of b) {
          lo = Math.min(lo, p[c])
          hi = Math.max(hi, p[c])
        }
        if (hi - lo > widest) {
          widest = hi - lo
          best = i
          channel = c
        }
      }
    })
    if (best < 0) break

    const b = boxes[best].sort((p, q) => p[channel] - q[channel])
    const mid = b.length >> 1
    boxes[best] = b.slice(0, mid)
    boxes.push(b.slice(mid))
  }

  return boxes.map((b) => {
    const s = [0, 0, 0]
    for (const p of b) for (let c = 0; c < 3; c++) s[c] += p[c]
    return s.map((v) => Math.round(v / b.length)) as RGB
  })
}
