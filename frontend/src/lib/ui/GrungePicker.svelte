<script lang="ts">
  import { imageURL } from '../assets'
  import { store } from '../store.svelte'

  // none adds a first tile for no map.
  let { value, onpick, none = false }: { value: string; onpick: (name: string) => void; none?: boolean } = $props()
</script>

<div class="maps">
  {#if none}
    <button class="map none" class:on={!value} title="no map" onclick={() => onpick('')}>–</button>
  {/if}
  {#each store.grungeMaps as name}
    <button class="map" class:on={value === name} title={name} onclick={() => onpick(name)}>
      {#await imageURL(`/grunge?name=${encodeURIComponent(name)}`) then url}
        <img src={url} alt={name} />
      {/await}
    </button>
  {/each}
</div>

<style>
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
  .none {
    font: 11px var(--mono);
    color: var(--faint);
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
</style>
