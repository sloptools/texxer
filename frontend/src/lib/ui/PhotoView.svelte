<script lang="ts">
  import { orderCorners, type Corner, type Cut } from '../cut'
  import { libraryURL, snippets, type PhotoEntry } from '../snippets.svelte'

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
  type Drag = { kind: 'corner'; cut: Cut; i: number } | { kind: 'move'; cut: Cut; from: Corner; start: Cut['corners'] } | { kind: 'rect'; from: Corner; to: Corner; moved: boolean }
  let drag = $state<Drag>()

  const inside = (p: Corner, q: Corner[]) => {
    let hit = false
    for (let i = 0, j = 3; i < 4; j = i++) {
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

    const corner = picked?.corners.findIndex((c) => Math.hypot(c[0] - p[0], c[1] - p[1]) < grab) ?? -1
    if (picked && corner >= 0) {
      drag = { kind: 'corner', cut: picked, i: corner }
      return
    }
    const over = pending.length ? undefined : [...photo.cuts].reverse().find((c) => inside(p, c.corners))
    if (over) {
      snippets.cut = over.id
      drag = { kind: 'move', cut: over, from: p, start: over.corners.map((c) => [...c]) as Cut['corners'] }
      return
    }
    drag = { kind: 'rect', from: p, to: p, moved: false }
  }

  function move(e: PointerEvent) {
    cursor = toPhoto(e)
    const d = drag
    if (!d) return
    const p = toPhoto(e)
    if (d.kind === 'corner') snippets.moveCorner(photo, d.cut, d.i, clamp(p))
    else if (d.kind === 'move') {
      const dx = p[0] - d.from[0]
      const dy = p[1] - d.from[1]
      d.start.forEach((c, i) => (d.cut.corners[i] = clamp([c[0] + dx, c[1] + dy])))
      snippets.changed(photo, d.cut)
    } else {
      d.to = p
      if (Math.hypot(p[0] - d.from[0], p[1] - d.from[1]) * zoom > 4) d.moved = true
    }
  }

  function up() {
    const d = drag
    drag = undefined
    if (d?.kind !== 'rect') return

    if (d.moved && !pending.length) {
      const [a, b] = [clamp(d.from), clamp(d.to)]
      const x0 = Math.min(a[0], b[0])
      const x1 = Math.max(a[0], b[0])
      const y0 = Math.min(a[1], b[1])
      const y1 = Math.max(a[1], b[1])
      if (x1 - x0 > 2 && y1 - y0 > 2) snippets.addCut(photo, [[x0, y0], [x1, y0], [x1, y1], [x0, y1]])
      return
    }
    // A click without a drag adds a corner; the fourth makes the cut.
    pending = [...pending, clamp(d.from)]
    if (pending.length === 4) {
      snippets.addCut(photo, orderCorners(pending))
      pending = []
    } else snippets.cut = ''
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
        <polygon class="halo" points={points(cut.corners)} />
        <polygon class="cut" class:on={cut.id === snippets.cut} points={points(cut.corners)} />
        {#if cut.id === snippets.cut}
          {#each cut.corners as c}
            <circle class="corner" cx={c[0]} cy={c[1]} r={5 / zoom} />
          {/each}
        {:else}
          <text class="name" x={cut.corners[0][0] + 4 / zoom} y={cut.corners[0][1] + 12 / zoom} font-size={10 / zoom}>{cut.name}</text>
        {/if}
      {/each}
      {#if drag?.kind === 'rect' && drag.moved && !pending.length}
        <rect class="halo" x={Math.min(drag.from[0], drag.to[0])} y={Math.min(drag.from[1], drag.to[1])} width={Math.abs(drag.to[0] - drag.from[0])} height={Math.abs(drag.to[1] - drag.from[1])} />
        <rect class="cut on" x={Math.min(drag.from[0], drag.to[0])} y={Math.min(drag.from[1], drag.to[1])} width={Math.abs(drag.to[0] - drag.from[0])} height={Math.abs(drag.to[1] - drag.from[1])} />
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
  <p class="hint">drag a rectangle or click four corners · drag corners to fit · ⌫ deletes · two fingers pan, pinch zooms · F fits</p>
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
