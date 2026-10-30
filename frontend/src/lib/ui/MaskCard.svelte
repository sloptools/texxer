<script lang="ts">
  import Icon from '../Icon.svelte'
  import { imageURL } from '../assets'
  import { store } from '../store.svelte'
  import type { Axis, Mask } from '../types'
  import Seg from './Seg.svelte'
  import Slider from './Slider.svelte'

  let { mask, parts, onremove }: { mask: Mask; parts: string[]; onremove: () => void } = $props()

  const axes: [Axis, string][] = [
    [0, 'x'],
    [1, 'y'],
    [2, 'z'],
  ]

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
    <span class="caps">{mask.kind}</span>
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
  {:else}
    {#if mask.kind === 'paint'}
      <p class="hint">paint it with the brush (B)</p>
    {:else if mask.kind === 'stamps'}
      <p class="hint">{mask.stamps.length} stamps · place them with the stamp tool (S)</p>
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
    {#if mask.kind === 'gradient' || mask.kind === 'facing'}
      <Seg value={mask.axis} options={axes} onchange={(v) => (mask.axis = v)} />
    {/if}
    <Slider label="amount" bind:value={mask.amount} def={0.5} />
    <Slider label="soft" bind:value={mask.soft} def={0} />
    {#if mask.kind !== 'noise' && mask.kind !== 'grunge'}
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

  .hint {
    margin: 0;
    font-size: 11px;
    color: var(--faint);
  }
</style>
