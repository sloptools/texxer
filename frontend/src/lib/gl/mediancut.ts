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

const far = 24 * 24

// Moves an earlier palette onto the pixels with a few k-means steps, so a small
// edit only shifts the colors near it. A new color far from all of them takes
// over the least used one when that saves more error than it costs, so even a
// few painted pixels of a new color keep it.
// The first fixed colors stay as they are.
export function refine(palette: RGB[], pixels: RGB[], steps = 6, fixed = 0): RGB[] {
  const n = palette.length
  const colors = new Float64Array(palette.flat())
  const flat = new Float64Array(pixels.flat())
  const sums = new Float64Array(n * 4)
  const owner = new Int32Array(pixels.length)
  const error = new Float64Array(pixels.length)

  // The nearest color to pixel i (a flat index), leaving out skip; its distance in dist.
  let dist = 0
  const nearest = (i: number, skip = -1) => {
    const r = flat[i]
    const g = flat[i + 1]
    const b = flat[i + 2]
    let best = 0
    dist = Infinity
    for (let k = 0, j = 0; k < n; k++, j += 3) {
      if (k === skip) continue
      const d = (r - colors[j]) * (r - colors[j]) + (g - colors[j + 1]) * (g - colors[j + 1]) + (b - colors[j + 2]) * (b - colors[j + 2])
      if (d < dist) {
        dist = d
        best = k
      }
    }
    return best
  }

  for (let step = 0; step < steps; step++) {
    sums.fill(0)
    let worst = -1
    let worstDist = far
    let gain = 0

    for (let p = 0; p < pixels.length; p++) {
      const i = p * 3
      const best = nearest(i)
      const bestDist = dist
      owner[p] = best
      error[p] = bestDist
      const s = best * 4
      sums[s] += flat[i]
      sums[s + 1] += flat[i + 1]
      sums[s + 2] += flat[i + 2]
      sums[s + 3]++
      if (bestDist > far) gain += bestDist
      if (bestDist > worstDist) {
        worstDist = bestDist
        worst = i
      }
    }

    let moved = false
    let least = fixed
    for (let k = fixed; k < n; k++) {
      const count = sums[k * 4 + 3]
      if (count < sums[least * 4 + 3]) least = k
      if (!count) continue
      for (let c = 0; c < 3; c++) {
        const v = Math.round(sums[k * 4 + c] / count)
        if (v !== colors[k * 3 + c]) moved = true
        colors[k * 3 + c] = v
      }
    }

    if (worst >= 0 && least < n) {
      // What the least used color's pixels lose by going to their next nearest one.
      let cost = 0
      for (let p = 0; p < pixels.length; p++) {
        if (owner[p] !== least) continue
        nearest(p * 3, least)
        cost += dist - error[p]
      }
      if (gain > cost) {
        colors.set(flat.subarray(worst, worst + 3), least * 3)
        moved = true
      }
    }
    if (!moved) break
  }

  return palette.map((_, k) => [colors[k * 3], colors[k * 3 + 1], colors[k * 3 + 2]])
}
