<script lang="ts">
  import { store } from '../store.svelte'
  import type { FlatMap } from '../types'

  const maps: [FlatMap, string][] = [
    ['texture', 'texture'],
    ['mask', 'mask'],
    ['glow', 'glow'],
    ['ao', 'ao'],
    ['edges', 'edges'],
    ['cavity', 'cavity'],
    ['parts', 'parts'],
  ]
</script>

<div class="bar">
  <div class="group">
    {#each maps as [map, label], i}
      <button class:on={store.flatMap === map} onclick={() => (store.flatMap = map)}>{label}<kbd>{i + 1}</kbd></button>
    {/each}
  </div>
  <div class="group">
    <button class:on={store.flatUV} title="UV layout: cuts, unused space, overlaps in red (W)" onclick={() => (store.flatUV = !store.flatUV)}>uv<kbd>W</kbd></button>
  </div>
</div>

<style>
  .bar {
    position: absolute;
    top: 10px;
    left: 10px;
    display: flex;
    gap: 6px;
    pointer-events: none;
  }

  .group {
    display: flex;
    gap: 1px;
    padding: 2px;
    border: 1px solid var(--line);
    border-radius: 8px;
    background: var(--panel);
    pointer-events: auto;
  }

  button {
    display: flex;
    align-items: center;
    gap: 6px;
    height: 24px;
    padding: 0 8px;
    border: 0;
    border-radius: 6px;
    background: none;
    font: 12px var(--mono);
    color: var(--dim);
  }
  button:hover {
    color: var(--strong);
  }
  button.on {
    background: var(--active);
    color: var(--accent);
  }
  kbd {
    border: 0;
    padding: 0;
  }
</style>
