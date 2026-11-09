<script lang="ts">
  import * as THREE from 'three'
  import type { Stage } from '../gl/stage'

  let { stage }: { stage: Stage } = $props()

  const size = 72
  const r = 25
  const c = size / 2

  // World axes, where the camera looks from to see along them: yaw, pitch.
  const axes = [
    { name: 'x', dir: new THREE.Vector3(1, 0, 0), view: [Math.PI / 2, 0], color: '#c77a6e' },
    { name: 'y', dir: new THREE.Vector3(0, 1, 0), view: [0, 1.55], color: '#95b36f' },
    { name: 'z', dir: new THREE.Vector3(0, 0, 1), view: [0, 0], color: '#6f95c4' },
  ].flatMap((a) => [
    { ...a, key: a.name, positive: true },
    { ...a, key: '-' + a.name, positive: false, dir: a.dir.clone().negate(), view: a.name === 'y' ? [0, -1.55] : [a.view[0] + Math.PI, 0] },
  ])

  type Dot = { key: string; x: number; y: number; z: number; color: string; positive: boolean; label: string }
  let dots = $state<Dot[]>([])
  let ortho = $state(false)
  let hover = $state('')

  // Follows the camera each frame, back to front.
  $effect(() => {
    let frame = 0
    const inv = new THREE.Quaternion()
    const v = new THREE.Vector3()
    let last = ''
    const loop = () => {
      frame = requestAnimationFrame(loop)
      inv.copy(stage.cameraQuaternion).invert()
      const next = axes
        .map((a) => {
          v.copy(a.dir).applyQuaternion(inv)
          return { key: a.key, x: c + v.x * r, y: c - v.y * r, z: v.z, color: a.color, positive: a.positive, label: a.name.toUpperCase() }
        })
        .sort((p, q) => p.z - q.z)
      const sig = next.map((d) => d.x.toFixed(1) + d.y.toFixed(1)).join()
      if (sig !== last) {
        last = sig
        dots = next
      }
      ortho = stage.controls.ortho
    }
    loop()
    return () => cancelAnimationFrame(frame)
  })

  // Again on the axis already in view looks from the other side, as in Blender.
  function pick(key: string) {
    const ctl = stage.controls
    const facing = dots.at(-1)
    const a = axes.find((a) => a.key === (ctl.onAxis && facing?.key === key ? opposite(key) : key))!
    const yaw = a.name === 'y' ? Math.round(ctl.yaw / (Math.PI / 2)) * (Math.PI / 2) : a.view[0]
    ctl.view(yaw, a.view[1])
  }

  const opposite = (key: string) => (key.startsWith('-') ? key.slice(1) : '-' + key)

  // Dragging the gizmo orbits; a click without moving picks.
  let drag: { x: number; y: number; moved: boolean; key: string } | undefined

  function down(e: PointerEvent, key = '') {
    e.stopPropagation()
    // On the svg, so moves and the release reach it from a dot too.
    ;(e.currentTarget as Element).closest('svg')!.setPointerCapture(e.pointerId)
    drag = { x: e.clientX, y: e.clientY, moved: false, key }
  }

  function move(e: PointerEvent) {
    if (!drag) return
    const dx = e.clientX - drag.x
    const dy = e.clientY - drag.y
    if (!drag.moved && Math.hypot(dx, dy) < 3) return
    drag.moved = true
    drag.x = e.clientX
    drag.y = e.clientY
    stage.controls.orbit(-dx * 0.012, -dy * 0.012)
  }

  function up() {
    if (drag && !drag.moved && drag.key) pick(drag.key)
    drag = undefined
  }
</script>

<div class="gizmo">
  <svg width={size} height={size} viewBox="0 0 {size} {size}" role="toolbar" aria-label="View" tabindex="-1" onpointerdown={(e) => down(e)} onpointermove={move} onpointerup={up}>
    <circle class="disc" cx={c} cy={c} r={size / 2 - 1} />
    {#each dots as d (d.key)}
      {#if d.positive}
        <line x1={c} y1={c} x2={d.x} y2={d.y} stroke={d.color} stroke-width="2" opacity={0.5 + d.z * 0.3} />
      {/if}
    {/each}
    {#each dots as d (d.key)}
      <g
        class="dot"
        role="button"
        tabindex="-1"
        aria-label="View from {d.key}"
        onpointerdown={(e) => down(e, d.key)}
        onpointerenter={() => (hover = d.key)}
        onpointerleave={() => (hover = '')}
      >
        <circle cx={d.x} cy={d.y} r={d.positive ? 7 : 5.5} fill={d.positive || hover === d.key ? d.color : 'var(--bg)'} stroke={d.color} stroke-width="1.5" opacity={0.65 + d.z * 0.35} />
        {#if d.positive || hover === d.key}
          <text x={d.x} y={d.y + 3.5} class="label">{d.positive ? d.label : '-' + d.label}</text>
        {/if}
      </g>
    {/each}
  </svg>
  <button class="lens" class:on={ortho} title="Ortho or perspective (5); axis views turn ortho, orbiting turns it off" onclick={() => stage.controls.setOrtho(!ortho)}>
    {ortho ? 'ortho' : 'persp'}<kbd>5</kbd>
  </button>
  <button class="lens" title="Reset the view (0)" onclick={() => stage.resetView()}>
    reset<kbd>0</kbd>
  </button>
</div>

<style>
  .gizmo {
    position: absolute;
    top: 10px;
    right: 10px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
  }
  svg {
    display: block;
    cursor: grab;
    touch-action: none;
  }
  .disc {
    fill: rgb(235 232 223 / 0);
    transition: fill 0.1s;
  }
  svg:hover .disc {
    fill: rgb(235 232 223 / 0.05);
  }
  .dot {
    cursor: pointer;
  }
  .label {
    font: 600 9px var(--mono);
    fill: var(--bg);
    text-anchor: middle;
    pointer-events: none;
  }
  .lens {
    display: flex;
    gap: 5px;
    align-items: center;
    height: 20px;
    padding: 0 7px;
    border: 1px solid var(--line);
    border-radius: 5px;
    background: var(--panel);
    font: 10px var(--mono);
    color: var(--dim);
  }
  .lens.on {
    color: var(--accent);
  }
  .lens kbd {
    font: 9px var(--mono);
    color: var(--faint);
  }
</style>
