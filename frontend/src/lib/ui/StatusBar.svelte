<script lang="ts">
  import { store } from '../store.svelte'
  import { tools } from '../tools.svelte'
  import type { View } from '../types'
  import Seg from './Seg.svelte'

  const views: [View, string][] = [
    ['lit', 'lit'],
    ['unlit', 'unlit'],
    ['game', 'game'],
    ['mask', 'mask'],
  ]

  const hover = $derived(
    store.picksIslands && store.hoverIsland >= 0 && !store.flat
      ? 'island ' + store.hoverIsland
      : store.hoverPart >= 0
        ? store.model?.parts[store.hoverPart] || 'part ' + store.hoverPart
        : '',
  )
</script>

<footer>
  <span class="info">
    {#if store.model}
      <span class="strong">{store.project.name}</span>
      <span>{store.project.size}²</span>
      <span>{store.model.parts.length} parts</span>
    {/if}
  </span>

  <span class="status">
    {#if store.baking}
      <span class="accent">baking…</span>
    {:else if store.message}
      <span>{store.message}</span>
    {:else if store.flat && store.hoverTexel}
      <span>{store.hoverTexel[0]}, {store.hoverTexel[1]}</span>
      {#if hover}<span>{hover}</span>{/if}
    {:else if hover && tools.tool === 'nav'}
      <span>{hover}</span>
      <span class="faint">{store.picksIslands ? 'click adds or removes it' : '⌘click adds it to the layer, ⇧⌘click its island'}</span>
    {/if}
  </span>

  <span class="tools">
    <button class:on={store.flat} title="Flat texture and UVs (tab)" onclick={() => (store.flat = !store.flat)}>2d</button>
    <Seg value={store.view} options={views} onchange={(v) => (store.view = v)} />
    <button title="Render at game resolution (P)" class:on={store.lines > 0} onclick={() => store.cycleLines()}>{store.lines ? store.lines + 'p' : 'full res'}</button>
    <button class:on={store.torch} title="Flashlight at the cursor (T)" onclick={() => (store.torch = !store.torch)}>torch</button>
    <button class:on={store.grid} title="Grid (⇧G)" onclick={() => (store.grid = !store.grid)}>grid</button>
    <button class:on={store.bloom} title="Bloom (⇧B)" onclick={() => (store.bloom = !store.bloom)}>bloom</button>
    <button title="Shortcuts (?)" onclick={() => (store.help = true)}>?</button>
  </span>
</footer>

<style>
  footer {
    display: flex;
    align-items: center;
    gap: 16px;
    height: 100%;
    padding: 0 8px 0 14px;
    font: 11px var(--mono);
    color: var(--dim);
  }

  .info,
  .status,
  .tools {
    display: flex;
    align-items: center;
    gap: 12px;
    white-space: nowrap;
  }
  .status {
    flex: 1;
    min-width: 0;
    overflow: hidden;
  }

  .strong {
    color: var(--fg);
  }
  .accent {
    color: var(--accent);
  }
  .faint {
    color: var(--faint);
  }

  .tools {
    gap: 4px;
  }
  .tools :global(.seg) {
    margin-right: 6px;
  }
  .tools :global(.seg button) {
    height: 18px;
    font-size: 11px;
  }
  button {
    height: 22px;
    padding: 0 7px;
    border: 0;
    border-radius: 5px;
    background: none;
    font: 11px var(--mono);
    color: var(--dim);
  }
  button:hover {
    background: var(--hover);
    color: var(--strong);
  }
  button.on {
    color: var(--accent);
  }
</style>
