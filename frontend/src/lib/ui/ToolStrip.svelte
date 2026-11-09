<script lang="ts">
  import Icon, { type IconName } from '../Icon.svelte'
  import { store } from '../store.svelte'
  import { tools } from '../tools.svelte'
  import type { Tool } from '../types'

  let { active }: { active: boolean } = $props()

  type Item = { label: string; key: string; icon: IconName; on: () => boolean; run: () => void; off?: () => boolean }

  const use = (tool: Tool) => () => tools.tool !== tool && tools.set(tool)

  const items: (Item | 'line')[] = [
    { label: 'view and pick surfaces', key: 'esc', icon: 'pointer', on: () => tools.tool === 'nav', run: use('nav') },
    { label: 'brush', key: 'B', icon: 'paint', on: () => tools.tool === 'brush', run: use('brush') },
    { label: 'stamp', key: 'S', icon: 'stamps', on: () => tools.tool === 'stamp', run: use('stamp') },
    { label: 'volume', key: 'Q', icon: 'volume', on: () => tools.tool === 'volume', run: use('volume') },
    'line',
    { label: 'take a color from the model', key: 'K', icon: 'pipette', on: () => store.sampling, run: () => (store.sampling = !store.sampling), off: () => !store.layer },
  ]

  // The tip shows after a short rest, then follows from button to button at once.
  let tip = $state<{ item: Item; top: number } | null>(null)
  let timer = 0

  function enter(e: PointerEvent, item: Item) {
    const top = (e.currentTarget as HTMLElement).offsetTop
    clearTimeout(timer)
    if (tip) tip = { item, top }
    else timer = window.setTimeout(() => (tip = { item, top }), 400)
  }

  function leave() {
    clearTimeout(timer)
    timer = window.setTimeout(() => (tip = null), 250)
  }
</script>

<div class="strip">
  {#if active && store.model && !store.flat}
    {#each items as item}
      {#if item === 'line'}
        <span class="line"></span>
      {:else}
        <button
          class:on={item.on()}
          disabled={item.off?.()}
          aria-label={item.label}
          onclick={() => {
            tip = null
            item.run()
          }}
          onpointerenter={(e) => enter(e, item)}
          onpointerleave={leave}
        >
          <Icon name={item.icon} />
        </button>
      {/if}
    {/each}
  {/if}

  {#if tip}
    <div class="tip" style:top="{tip.top}px">{tip.item.label}<kbd>{tip.item.key}</kbd></div>
  {/if}
</div>

<style>
  .strip {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
    padding-top: 8px;
    border-left: 1px solid var(--line);
    background: var(--panel);
  }
  button {
    display: grid;
    place-items: center;
    width: 28px;
    height: 28px;
    padding: 0;
    border: 0;
    border-radius: 6px;
    background: none;
    color: var(--dim);
  }
  button:hover:not(:disabled) {
    color: var(--strong);
  }
  button.on {
    background: var(--active);
    color: var(--accent);
  }
  button:disabled {
    color: var(--faint);
  }
  .line {
    width: 16px;
    height: 1px;
    margin: 4px 0;
    background: var(--line);
  }
  .tip {
    position: absolute;
    right: calc(100% + 6px);
    z-index: 10;
    display: flex;
    gap: 8px;
    align-items: center;
    height: 28px;
    padding: 0 8px;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--panel);
    font: 12px var(--mono);
    color: var(--strong);
    white-space: nowrap;
    pointer-events: none;
  }
  kbd {
    color: var(--faint);
  }
</style>
