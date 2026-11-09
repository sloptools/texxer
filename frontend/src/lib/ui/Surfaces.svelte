<script lang="ts">
  import { store } from '../store.svelte'
  import { tools } from '../tools.svelte'

  // A surfaces mask or a folder's zone: whole parts and single islands.
  let { picked }: { picked: { parts: number[]; islands: number[] } } = $props()

  const parts = $derived(store.model?.parts ?? [])

  function toggle(i: number) {
    const at = picked.parts.indexOf(i)
    if (at >= 0) picked.parts.splice(at, 1)
    else picked.parts.push(i)
  }

  function clear() {
    picked.parts = []
    picked.islands = []
  }
</script>

{#if !store.onePart}
  <div class="parts">
    {#each parts as name, i}
      <button class="part" class:on={picked.parts.includes(i)} onclick={() => toggle(i)}>{name || 'part ' + i}</button>
    {/each}
  </div>
{/if}
<div class="row">
  <span class="hint">
    {#if tools.tool !== 'nav'}with the view tool (esc):{/if}
    {#if store.onePart}click islands in the view{:else}click parts in the view, ⇧ single islands{/if}
    {#if picked.islands.length}· {picked.islands.length} {picked.islands.length === 1 ? 'island' : 'islands'}{/if}
  </span>
  {#if picked.parts.length || picked.islands.length}
    <button class="text" title="Remove all parts and islands" onclick={clear}>clear</button>
  {/if}
</div>

<style>
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

  .row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }
  .hint {
    font-size: 11px;
    color: var(--faint);
  }
  .text {
    flex: none;
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
</style>
