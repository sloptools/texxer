<script lang="ts">
  import Icon, { type IconName } from '../Icon.svelte'
  import { store } from '../store.svelte'
  import type { View } from '../types'
  import ViewLight from './ViewLight.svelte'

  // How the view shows the model; none of it goes into the texture.
  const views: [View, string, string][] = [
    ['lit', 'lit', 'L'],
    ['unlit', 'unlit', 'U'],
    ['game', 'game', 'I'],
    ['mask', 'mask', 'M'],
  ]

  let lightOpen = $state(false)

  const overlays: [IconName, string, () => boolean, () => void][] = [
    ['torch', 'Flashlight at the cursor (T)', () => store.torch, () => (store.torch = !store.torch)],
    ['grid', 'Grid (⇧G)', () => store.grid, () => (store.grid = !store.grid)],
    ['bloom', 'Bloom (⇧B)', () => store.bloom, () => (store.bloom = !store.bloom)],
    ['sun', 'Lights of the view, not the texture', () => lightOpen, () => (lightOpen = !lightOpen)],
  ]
</script>

<div class="bar">
  <button class="wide" class:on={store.flat} title="Flat texture and UVs (Tab)" onclick={() => (store.flat = !store.flat)}>2d<kbd>tab</kbd></button>

  {#if !store.flat}
    <div class="views" role="radiogroup">
      {#each views as [v, label, key]}
        <button class="wide" class:on={store.view === v} role="radio" aria-checked={store.view === v} onclick={() => (store.view = v)}>{label}<kbd>{key}</kbd></button>
      {/each}
    </div>
    <button class="wide" class:on={store.lines > 0} title="Render at game resolution (P)" onclick={() => store.cycleLines()}>{store.lines ? store.lines + 'p' : 'full'}<kbd>P</kbd></button>
    <div class="icons">
      {#each overlays as [icon, title, on, toggle]}
        <button class:on={on()} {title} onclick={toggle}><Icon name={icon} size={11} /></button>
      {/each}
    </div>
  {/if}

  {#if lightOpen}
    <ViewLight onclose={() => (lightOpen = false)} />
  {/if}
</div>

<style>
  .bar {
    position: relative;
    display: flex;
    flex-direction: column;
    gap: 4px;
    width: 72px;
  }
  .views {
    display: flex;
    flex-direction: column;
    gap: 1px;
    padding: 2px;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--panel);
  }
  .views button {
    border: 0;
    background: none;
  }
  .icons {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 4px;
  }
  button {
    display: grid;
    place-items: center;
    height: 22px;
    padding: 0 7px;
    border: 1px solid var(--line);
    border-radius: 5px;
    background: var(--panel);
    font: 11px var(--mono);
    color: var(--dim);
  }
  button.wide {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  button:hover {
    color: var(--strong);
  }
  button.on {
    color: var(--accent);
  }
  .views button.on {
    background: var(--active);
  }
  kbd {
    font: 9px var(--mono);
    color: var(--faint);
  }
</style>
