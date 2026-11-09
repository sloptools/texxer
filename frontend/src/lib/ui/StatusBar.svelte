<script lang="ts">
  import { store } from '../store.svelte'
  import { tools } from '../tools.svelte'

  const hover = $derived(
    store.onePart && store.hoverIsland >= 0 && !store.flat
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
      <span class="faint">⌘click adds it to the layer, ⇧⌘click its island</span>
    {/if}
  </span>

  <span class="tools">
    <button title="Shortcuts and docs (?)" onclick={() => (store.help = true)}>? help</button>
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
</style>
