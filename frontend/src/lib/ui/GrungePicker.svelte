<script lang="ts">
  import * as api from '../../../wailsjs/go/main/App'
  import Icon from '../Icon.svelte'
  import { imageURL } from '../assets'
  import { store } from '../store.svelte'

  // none adds a first tile for no map.
  let { value, onpick, none = false }: { value: string; onpick: (name: string) => void; none?: boolean } = $props()

  // Folded to the picked map; the grid opens on click and folds on a pick.
  let open = $state(false)

  // Maps put into the grunge folder meanwhile show up on opening.
  async function toggle() {
    open = !open
    if (open) store.grungeMaps = await api.GrungeMaps()
  }

  const missing = $derived(!!value && store.grungeMaps.length > 0 && !store.grungeMaps.includes(value))

  function pick(name: string) {
    onpick(name)
    open = false
  }
</script>

<button class="current" class:open title="Pick a grunge map" onclick={toggle}>
  <span class="thumb">
    {#if value}
      {#await imageURL(`/grunge?name=${encodeURIComponent(value)}`) then url}
        <img src={url} alt={value} />
      {/await}
    {/if}
  </span>
  <span class="name" class:missing title={missing ? 'Not in your grunge folder: the mask shows nothing' : undefined}>{value || 'no map'}{missing ? ' (missing)' : ''}</span>
  <span class="more">{open ? '▴' : '▾'}</span>
</button>

{#if open}
  <div class="maps">
    {#if none}
      <button class="map none" class:on={!value} title="no map" onclick={() => pick('')}>–</button>
    {/if}
    {#each store.grungeMaps as name}
      <button class="map" class:on={value === name} title={name} onclick={() => pick(name)}>
        {#await imageURL(`/grunge?name=${encodeURIComponent(name)}`) then url}
          <img src={url} alt={name} />
        {/await}
      </button>
    {/each}
    <button class="map none" title="Your own maps: PNG or JPG grays, white = dirt, best tileable; the file name is the map's name" onclick={() => api.OpenGrungeFolder()}><Icon name="dir" size={11} /></button>
  </div>
{/if}

<style>
  .current {
    display: flex;
    align-items: center;
    gap: 8px;
    height: 30px;
    padding: 2px 8px 2px 2px;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--bg);
    font: 11px var(--mono);
    color: var(--dim);
  }
  .current:hover,
  .current.open {
    color: var(--strong);
  }
  .thumb {
    flex: none;
    width: 24px;
    height: 24px;
    overflow: hidden;
    border-radius: 4px;
    background: var(--panel);
  }
  .thumb img {
    display: block;
    width: 100%;
    height: 100%;
    image-rendering: pixelated;
  }
  .name {
    flex: 1;
    overflow: hidden;
    text-align: left;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .missing {
    color: var(--error);
  }
  .more {
    color: var(--faint);
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
  .none {
    display: grid;
    place-items: center;
    font: 11px var(--mono);
    color: var(--faint);
  }
  .none:hover {
    color: var(--strong);
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
