<script lang="ts">
  import Icon from '../Icon.svelte'
  import { imageURL } from '../assets'
  import { store } from '../store.svelte'
  import { tools } from '../tools.svelte'
  import type { Axis, Mask, VolumeShape } from '../types'
  import Seg from './Seg.svelte'
  import Slider from './Slider.svelte'

  let { mask, index, parts, onremove }: { mask: Mask; index: number; parts: string[]; onremove: () => void } = $props()

  const shapes: [VolumeShape, string][] = [
    ['box', 'box'],
    ['cylinder', 'cylinder'],
    ['sphere', 'sphere'],
  ]
  const editing = $derived(tools.tool === 'volume' && tools.volume === mask)

  // Directions as seen in a front view, like Blender's. A gradient rises from its
  // side; facing picks surfaces that face that way.
  const from: [Axis, string][] = [
    [1, 'below'],
    [4, 'above'],
    [0, 'left'],
    [3, 'right'],
    [2, 'back'],
    [5, 'front'],
  ]
  const faces: [Axis, string][] = [
    [1, 'up'],
    [4, 'down'],
    [3, 'left'],
    [0, 'right'],
    [5, 'back'],
    [2, 'front'],
  ]

  const widths: [number, string][] = [
    [0, 'fade'],
    [1, '1 px'],
    [2, '2 px'],
    [3, '3 px'],
  ]

  // Crisp lines want the whole width, without a soft cut.
  function setCrisp(v: number) {
    if (v && !mask.crisp) {
      mask.amount = 1
      mask.soft = 0
    }
    mask.crisp = v
  }

  function clearPaint() {
    const paint = store.paint
    if (!paint) return
    const before = paint.read(mask.paint)
    const after = new Uint8Array(before.length)
    paint.write(mask.paint, after)
    store.pushPaint(mask.paint, before, after)
  }

  function toggle(i: number) {
    const at = mask.parts.indexOf(i)
    if (at >= 0) mask.parts.splice(at, 1)
    else mask.parts.push(i)
  }
</script>

<div class="card">
  <div class="head">
    <span class="caps kind"><Icon name={mask.kind} size={11} />{mask.kind}</span>
    <span class="actions">
      {#if mask.kind === 'paint'}
        <button class="text" title="Clear the painted mask" onclick={clearPaint}>clear</button>
      {/if}
      {#if mask.kind === 'noise' || mask.kind === 'grunge' || mask.rough > 0}
        <button class="text" title="New random seed" onclick={() => (mask.seed = Math.floor(Math.random() * 1000))}>seed</button>
      {/if}
      <button class="text" class:on={mask.invert} title="Invert" onclick={() => (mask.invert = !mask.invert)}>invert</button>
      <button class="icon" title="Remove mask" onclick={onremove}><Icon name="x" size={11} /></button>
    </span>
  </div>

  {#if mask.kind === 'parts'}
    <div class="parts">
      {#each parts as name, i}
        <button class="part" class:on={mask.parts.includes(i)} onclick={() => toggle(i)}>{name || 'part ' + i}</button>
      {/each}
    </div>
    <p class="hint">click parts in the view to add or remove them</p>
  {:else if mask.kind === 'islands'}
    <div class="row">
      <span class="hint">{mask.islands.length} {mask.islands.length === 1 ? 'island' : 'islands'} · click surfaces in the view</span>
      {#if mask.islands.length}
        <button class="text" title="Remove all islands" onclick={() => (mask.islands = [])}>clear</button>
      {/if}
    </div>
  {:else}
    {#if mask.kind === 'paint'}
      <p class="hint">paint it with the brush (B)</p>
    {:else if mask.kind === 'stamps'}
      <p class="hint">{mask.stamps.length} stamps · place them with the stamp tool (S)</p>
    {:else if mask.kind === 'volume'}
      <Seg value={mask.volume.shape} options={shapes} onchange={(v) => (mask.volume.shape = v)} />
      <div class="row">
        <span class="hint">{editing ? 'click the model to put it there · W E R move, turn, size' : 'everything inside it, all around'}</span>
        <button class="text" class:on={editing} title="Move it in the view (Q)" onclick={() => (editing ? tools.set('volume') : tools.editVolume(index))}>edit</button>
        <button class="text" class:on={mask.volume.mirror} title="Mirror across X" onclick={() => (mask.volume.mirror = !mask.volume.mirror)}>mirror</button>
      </div>
    {/if}
    {#if mask.kind === 'grunge'}
      <div class="maps">
        {#each store.grungeMaps as name}
          <button class="map" class:on={mask.map === name} title={name} onclick={() => (mask.map = name)}>
            {#await imageURL(`/grunge?name=${encodeURIComponent(name)}`) then url}
              <img src={url} alt={name} />
            {/await}
          </button>
        {/each}
      </div>
    {/if}
    {#if mask.kind === 'edges' || mask.kind === 'cavity'}
      <Seg value={mask.crisp} options={widths} onchange={setCrisp} />
    {/if}
    {#if mask.kind === 'gradient' || mask.kind === 'facing'}
      <div class="dirs">
        <span>{mask.kind === 'gradient' ? 'from' : 'faces'}</span>
        <Seg value={mask.axis} options={mask.kind === 'gradient' ? from : faces} onchange={(v) => (mask.axis = v)} />
      </div>
    {/if}
    <Slider label="amount" bind:value={mask.amount} def={0.5} />
    {#if mask.kind !== 'paint'}
      <Slider label="soft" bind:value={mask.soft} def={0} />
    {/if}
    {#if mask.kind !== 'noise' && mask.kind !== 'grunge' && mask.kind !== 'paint'}
      <Slider label="rough" bind:value={mask.rough} def={0} />
    {/if}
    {#if mask.kind === 'noise' || mask.kind === 'grunge' || mask.rough > 0}
      <Slider label="scale" bind:value={mask.scale} min={1} max={96} step={1} def={mask.kind === 'grunge' ? 3 : 24} digits={0} />
    {/if}
  {/if}
</div>

<style>
  .card {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 8px 10px 10px;
    border: 1px solid var(--line);
    border-radius: 7px;
    background: var(--raised);
  }

  .head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 20px;
  }
  .kind {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .actions {
    display: flex;
    align-items: center;
    gap: 2px;
  }

  .text {
    height: 20px;
    padding: 0 6px;
    border: 0;
    border-radius: 4px;
    background: none;
    font: 11px var(--mono);
    color: var(--faint);
  }
  .text:hover {
    color: var(--strong);
  }
  .text.on {
    color: var(--accent);
  }

  .icon {
    display: grid;
    place-items: center;
    width: 20px;
    height: 20px;
    padding: 0;
    border: 0;
    border-radius: 4px;
    background: none;
    color: var(--faint);
  }
  .icon:hover {
    background: var(--hover);
    color: var(--error);
  }

  .dirs {
    display: flex;
    align-items: center;
    gap: 8px;
    font: 11px var(--mono);
    color: var(--faint);
  }
  .dirs :global(.seg) {
    flex: 1;
  }
  .dirs :global(.seg button) {
    padding: 0 2px;
    font-size: 11px;
  }

  .parts {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }
  .part {
    height: 22px;
    max-width: 100%;
    padding: 0 8px;
    overflow: hidden;
    border: 1px solid var(--line);
    border-radius: 5px;
    background: var(--bg);
    font: 11px var(--mono);
    color: var(--dim);
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .part:hover {
    color: var(--strong);
  }
  .part.on {
    border-color: var(--accent);
    color: var(--strong);
  }

  .maps {
    display: grid;
    grid-template-columns: repeat(7, 1fr);
    gap: 3px;
    max-height: 132px;
    overflow-y: auto;
  }
  .map {
    aspect-ratio: 1;
    padding: 0;
    overflow: hidden;
    border: 1px solid var(--line);
    border-radius: 4px;
    background: var(--bg);
  }
  .map.on {
    border-color: var(--accent);
    box-shadow: 0 0 0 1px var(--accent);
  }
  .map img {
    display: block;
    width: 100%;
    height: 100%;
    image-rendering: pixelated;
  }

  .row {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .hint {
    margin: 0;
    font-size: 11px;
    color: var(--faint);
  }
</style>
