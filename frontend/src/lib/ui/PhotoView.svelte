<script lang="ts">
  import { ellipse, orderCorners, stretchCorner, stretchEdge, type Corner, type Cut } from '../cut'
  import { isMouseWheel } from '../input.svelte'
  import { libraryURL, snippets, type PhotoEntry } from '../snippets.svelte'
  import Seg from './Seg.svelte'

  let { photo }: { photo: PhotoEntry } = $props()

  let el = $state<HTMLDivElement>()
  let size = $state<[number, number]>([0, 0])
  let zoom = $state(1)
  let offset = $state<[number, number]>([0, 0])
  // Clicked corners of a cut in the making.
  let pending = $state<Corner[]>([])
  let cursor = $state<Corner>()

  const url = $derived(libraryURL('photo', photo.file, snippets.category))

  function fit() {
    if (!el || !size[0]) return
    const r = el.getBoundingClientRect()
    zoom = Math.min((r.width - 80) / size[0], (r.height - 80) / size[1])
    offset = [(r.width - size[0] * zoom) / 2, (r.height - size[1] * zoom) / 2]
  }

  $effect(() => {
    photo.id
    pending = []
  })

  function loaded(e: Event) {
    const img = e.currentTarget as HTMLImageElement
    size = [img.naturalWidth, img.naturalHeight]
    fit()
  }

  const toPhoto = (e: { clientX: number; clientY: number }): Corner => {
    const r = el!.getBoundingClientRect()
    return [(e.clientX - r.left - offset[0]) / zoom, (e.clientY - r.top - offset[1]) / zoom]
  }

  // Two fingers pan, pinch or ⌘/ctrl scroll zooms at the cursor.
  function zoomAt(e: { clientX: number; clientY: number }, factor: number) {
    const r = el!.getBoundingClientRect()
    const x = e.clientX - r.left
    const y = e.clientY - r.top
    const next = Math.min(Math.max(zoom * factor, 0.02), 40)
    offset = [x - ((x - offset[0]) * next) / zoom, y - ((y - offset[1]) * next) / zoom]
    zoom = next
  }

  function wheel(e: WheelEvent) {
    e.preventDefault()
    if (e.ctrlKey || e.metaKey) zoomAt(e, Math.exp(-e.deltaY * 0.01))
    else if (isMouseWheel(e)) zoomAt(e, Math.exp(-Math.sign(e.deltaY) * 0.15))
    else offset = [offset[0] - e.deltaX, offset[1] - e.deltaY]
  }

  // Pinch comes as WebKit gesture events, which the DOM types don't know.
  type GestureEvent = UIEvent & { scale: number; clientX: number; clientY: number }
  let gesture = 1
  $effect(() => {
    const v = el
    if (!v) return
    const start = (e: Event) => {
      e.preventDefault()
      gesture = 1
    }
    const change = (e: Event) => {
      const g = e as GestureEvent
      g.preventDefault()
      zoomAt(g, g.scale / gesture)
      gesture = g.scale
    }
    v.addEventListener('gesturestart', start)
    v.addEventListener('gesturechange', change)
    return () => {
      v.removeEventListener('gesturestart', start)
      v.removeEventListener('gesturechange', change)
    }
  })

  // Dragging: a corner of the picked cut, the whole cut, or a new rectangle.
  type Handle = { kind: 'corner' | 'edge'; cut: Cut; i: number; from: Corner; start: Cut['corners'] }
  type Drag = Handle | { kind: 'move'; cut: Cut; from: Corner; start: Cut['corners'] } | { kind: 'rect'; from: Corner; to: Corner; moved: boolean; square: boolean }
  let drag = $state<Drag>()

  const outline = (cut: Cut) => (cut.round ? ellipse(cut.corners) : cut.corners)

  // Midpoints of the edges, edge i from corner i to i + 1.
  const mids = (q: Corner[]): Corner[] => q.map((c, i) => [(c[0] + q[(i + 1) % 4][0]) / 2, (c[1] + q[(i + 1) % 4][1]) / 2])

  const inside = (p: Corner, q: Corner[]) => {
    let hit = false
    for (let i = 0, j = q.length - 1; i < q.length; j = i++) {
      if (q[i][1] > p[1] !== q[j][1] > p[1] && p[0] < ((q[j][0] - q[i][0]) * (p[1] - q[i][1])) / (q[j][1] - q[i][1]) + q[i][0]) hit = !hit
    }
    return hit
  }

  function down(e: PointerEvent) {
    if (e.button !== 0) return
    el!.setPointerCapture(e.pointerId)
    const p = toPhoto(e)
    const grab = 8 / zoom
    const picked = snippets.selectedCut

    const near = (q: Corner[]) => q.findIndex((c) => Math.hypot(c[0] - p[0], c[1] - p[1]) < grab)
    if (picked) {
      const start = picked.corners.map((c) => [...c]) as Cut['corners']
      const corner = near(picked.corners)
      const edge = near(mids(picked.corners))
      if (corner >= 0) drag = { kind: 'corner', cut: picked, i: corner, from: p, start }
      else if (edge >= 0) drag = { kind: 'edge', cut: picked, i: edge, from: p, start }
      if (drag) return
    }
    const over = pending.length ? undefined : [...photo.cuts].reverse().find((c) => inside(p, outline(c)))
    if (over) {
      snippets.cut = over.id
      drag = { kind: 'move', cut: over, from: p, start: over.corners.map((c) => [...c]) as Cut['corners'] }
      return
    }
    drag = { kind: 'rect', from: p, to: p, moved: false, square: e.shiftKey }
  }

  function move(e: PointerEvent) {
    cursor = toPhoto(e)
    const d = drag
    if (!d) return
    const p = toPhoto(e)
    if (d.kind === 'corner' || d.kind === 'edge') {
      d.cut.corners = handle(d, p).map(clamp) as Cut['corners']
      snippets.changed(photo, d.cut)
    } else if (d.kind === 'move') {
      const dx = p[0] - d.from[0]
      const dy = p[1] - d.from[1]
      d.start.forEach((c, i) => (d.cut.corners[i] = clamp([c[0] + dx, c[1] + dy])))
      snippets.changed(photo, d.cut)
    } else if (d.kind === 'rect') {
      d.square = e.shiftKey
      d.to = p
      if (Math.hypot(p[0] - d.from[0], p[1] - d.from[1]) * zoom > 4) d.moved = true
    }
  }

  function up() {
    const d = drag
    drag = undefined
    if (d?.kind !== 'rect') return

    if (d.moved && !pending.length) {
      const r = box(d)
      if (r.w > 2 && r.h > 2) snippets.addCut(photo, [[r.x, r.y], [r.x + r.w, r.y], [r.x + r.w, r.y + r.h], [r.x, r.y + r.h]])
      return
    }
    // A click without a drag adds a corner; the fourth makes the cut.
    pending = [...pending, clamp(d.from)]
    if (pending.length === 4) {
      snippets.addCut(photo, orderCorners(pending))
      pending = []
    } else snippets.cut = ''
  }

  // Skew moves the corner or edge freely, size stretches the cut along its edges.
  function handle(d: Handle, p: Corner) {
    if (!snippets.skew) return d.kind === 'corner' ? stretchCorner(d.start, d.i, p) : stretchEdge(d.start, d.i, p)
    const out = d.start.map((c) => [...c]) as Cut['corners']
    if (d.kind === 'corner') out[d.i] = p
    else {
      for (const k of [d.i, (d.i + 1) % 4]) out[k] = [out[k][0] + p[0] - d.from[0], out[k][1] + p[1] - d.from[1]]
    }
    return out
  }

  // The dragged box in the photo, square with ⇧.
  function box(d: { from: Corner; to: Corner; square: boolean }) {
    let dx = d.to[0] - d.from[0]
    let dy = d.to[1] - d.from[1]
    if (d.square) {
      const s = Math.max(Math.abs(dx), Math.abs(dy))
      dx = Math.sign(dx) * s
      dy = Math.sign(dy) * s
    }
    const [a, b] = [clamp(d.from), clamp([d.from[0] + dx, d.from[1] + dy])]
    const x = Math.min(a[0], b[0])
    const y = Math.min(a[1], b[1])
    return { x, y, w: Math.max(a[0], b[0]) - x, h: Math.max(a[1], b[1]) - y }
  }

  const clamp = (p: Corner): Corner => [Math.min(Math.max(p[0], 0), size[0]), Math.min(Math.max(p[1], 0), size[1])]

  function onkeydown(e: KeyboardEvent) {
    if ((e.target as HTMLElement).closest('input, textarea')) return
    const cut = snippets.selectedCut
    if (e.key === 'Escape') {
      if (pending.length) pending = []
      else snippets.cut = ''
    } else if ((e.key === 'Backspace' || e.key === 'Delete') && cut) snippets.deleteCut(photo, cut)
    else if (e.key === 'f' || e.key === '0') fit()
    else if (e.metaKey || e.ctrlKey) return
    else if (e.key === 'r') snippets.round = false
    else if (e.key === 'e') snippets.round = true
    else if (e.key === 's') snippets.skew = !snippets.skew
    else return
    e.preventDefault()
    e.stopPropagation()
  }

  const points = (c: Corner[]) => c.map((p) => p.join(',')).join(' ')
</script>

<svelte:window onkeydowncapture={onkeydown} />

<div
  class="view"
  bind:this={el}
  role="application"
  onwheel={wheel}
  onpointerdown={down}
  onpointermove={move}
  onpointerup={up}
  onpointerleave={() => (cursor = undefined)}
>
  <div class="layer" style:transform="translate({offset[0]}px, {offset[1]}px) scale({zoom})" style:width="{size[0]}px" style:height="{size[1]}px">
    <img src={url} alt={photo.file} onload={loaded} draggable="false" />
    <svg viewBox="0 0 {size[0]} {size[1]}" width={size[0]} height={size[1]}>
      {#each photo.cuts as cut (cut.id)}
        <polygon class="halo" points={points(outline(cut))} />
        <polygon class="cut" class:on={cut.id === snippets.cut} points={points(outline(cut))} />
        {#if cut.id === snippets.cut}
          {#if cut.round}
            <polygon class="frame" points={points(cut.corners)} />
          {/if}
          {#each mids(cut.corners) as c}
            <rect class="corner" x={c[0] - 3.5 / zoom} y={c[1] - 3.5 / zoom} width={7 / zoom} height={7 / zoom} />
          {/each}
          {#each cut.corners as c}
            <circle class="corner" cx={c[0]} cy={c[1]} r={5 / zoom} />
          {/each}
        {:else}
          <text class="name" x={cut.corners[0][0] + 4 / zoom} y={cut.corners[0][1] + 12 / zoom} font-size={10 / zoom}>{cut.name}</text>
        {/if}
      {/each}
      {#if drag?.kind === 'rect' && drag.moved && !pending.length}
        {@const r = box(drag)}
        {#if snippets.round}
          <ellipse class="halo" cx={r.x + r.w / 2} cy={r.y + r.h / 2} rx={r.w / 2} ry={r.h / 2} />
          <ellipse class="cut on" cx={r.x + r.w / 2} cy={r.y + r.h / 2} rx={r.w / 2} ry={r.h / 2} />
        {:else}
          <rect class="halo" x={r.x} y={r.y} width={r.w} height={r.h} />
          <rect class="cut on" x={r.x} y={r.y} width={r.w} height={r.h} />
        {/if}
      {/if}
      {#if pending.length}
        <polyline class="halo" points={points(cursor ? [...pending, cursor] : pending)} />
        <polyline class="pending" points={points(cursor ? [...pending, cursor] : pending)} />
        {#each pending as c}
          <circle class="corner" cx={c[0]} cy={c[1]} r={4 / zoom} />
        {/each}
      {/if}
    </svg>
  </div>
  <div class="shape" role="toolbar" tabindex="-1" onpointerdown={(e) => e.stopPropagation()}>
    <Seg value={snippets.round} options={[[false, 'rect R'], [true, 'ellipse E']]} onchange={(v) => (snippets.round = v)} />
    <Seg value={snippets.skew} options={[[false, 'size'], [true, 'skew']]} onchange={(v) => (snippets.skew = v)} />
  </div>
  <p class="hint">drag a box (⇧ square) or click four corners · corners and edges size or skew (S) · ⌫ deletes · two fingers pan, pinch zooms · F fits</p>
</div>

<style>
  .view {
    position: absolute;
    inset: 0;
    z-index: 2;
    overflow: hidden;
    background: var(--bg);
    cursor: crosshair;
    touch-action: none;
  }
  .layer {
    position: absolute;
    top: 0;
    left: 0;
    transform-origin: 0 0;
  }
  img,
  svg {
    position: absolute;
    inset: 0;
    display: block;
    user-select: none;
  }
  svg {
    overflow: visible;
  }
  /* A dark edge under every mark, so it shows on light and dark photos alike. */
  .halo {
    fill: none;
    stroke: rgb(0 0 0 / 0.7);
    stroke-width: 4;
    vector-effect: non-scaling-stroke;
  }
  .cut {
    fill: rgb(255 176 32 / 0.06);
    stroke: rgb(235 232 223 / 0.85);
    stroke-width: 1.5;
    vector-effect: non-scaling-stroke;
  }
  .cut.on {
    fill: rgb(255 176 32 / 0.12);
    stroke: var(--mark);
    stroke-width: 2;
  }
  .pending {
    fill: none;
    stroke: var(--mark);
    stroke-width: 1.5;
    stroke-dasharray: 4 3;
    vector-effect: non-scaling-stroke;
  }
  .frame {
    fill: none;
    stroke: var(--mark);
    stroke-width: 1;
    stroke-dasharray: 3 3;
    opacity: 0.6;
    vector-effect: non-scaling-stroke;
  }
  .shape {
    position: absolute;
    top: 10px;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    gap: 8px;
    cursor: default;
  }
  .corner {
    fill: var(--mark);
    stroke: rgb(0 0 0 / 0.8);
    stroke-width: 1.5;
    vector-effect: non-scaling-stroke;
  }
  .name {
    font-family: var(--mono);
    fill: var(--strong);
    paint-order: stroke;
    stroke: rgb(0 0 0 / 0.6);
    stroke-width: 3;
    vector-effect: non-scaling-stroke;
  }
  .hint {
    position: absolute;
    left: 0;
    right: 0;
    bottom: 10px;
    margin: 0;
    font: 11px var(--mono);
    color: var(--faint);
    text-align: center;
    pointer-events: none;
  }
</style>
