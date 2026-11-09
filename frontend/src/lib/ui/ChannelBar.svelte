<script lang="ts">
  import { store } from '../store.svelte'
  import { channels, type Channel, type DataChannel } from '../types'
  import ContextMenu, { type MenuItem } from './ContextMenu.svelte'

  const titles: Record<Channel, string> = {
    color: 'The color texture',
    ao: 'Ambient occlusion, gray: multiplies the color, or its own map for Godot',
    rough: 'Roughness, gray: its own map, the GLB takes it',
    glow: 'Emission, over what glowing color layers give',
  }

  let menu = $state<{ x: number; y: number; items: MenuItem[] } | null>(null)
  const ao = $derived(store.project.channels.find((d) => d.kind === 'ao'))

  function context(e: MouseEvent, c: Channel) {
    e.preventDefault()
    if (c === 'color' || !store.hasChannel(c)) return
    menu = { x: e.clientX, y: e.clientY, items: [{ label: `delete ${c} and its layers`, danger: true, run: () => store.deleteChannel(c as DataChannel) }] }
  }
</script>

<div class="bar" class:below={store.flat}>
  <div class="group" title="Channels: each a texture with its own layers (A cycles)">
    {#each channels as c}
      <button class:on={store.channel === c} class:missing={!store.hasChannel(c)} title={store.hasChannel(c) ? titles[c] : `Add ${c}: ${titles[c]}`} onclick={() => store.setChannel(c)} oncontextmenu={(e) => context(e, c)}>
        {store.hasChannel(c) ? c : '+' + c}
      </button>
    {/each}
  </div>
  {#if store.project.channels.length}
    <div class="group">
      <button class:on={!!store.onion} title="Show another channel faintly over this one, not exported (O cycles)" onclick={() => store.cycleOnion()}>
        onion{store.onion ? ' ' + store.onion : ''}<kbd>O</kbd>
      </button>
      {#if store.channel === 'ao' && ao}
        <button class:on={!ao.separate} title="AO multiplies into the color, as retro textures have it; off, it is its own map (for Godot)" onclick={() => (ao.separate = !ao.separate)}>
          into color
        </button>
      {/if}
    </div>
  {/if}
</div>

{#if menu}
  <ContextMenu {...menu} onclose={() => (menu = null)} />
{/if}

<style>
  .bar {
    position: absolute;
    top: 10px;
    left: 10px;
    display: flex;
    gap: 6px;
    pointer-events: none;
  }
  .bar.below {
    top: 44px;
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
  button.missing {
    opacity: 0.5;
  }
  button.missing:hover {
    opacity: 1;
  }
  kbd {
    border: 0;
    padding: 0;
  }
</style>
