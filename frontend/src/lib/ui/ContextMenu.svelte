<script lang="ts">
  import Icon, { type IconName } from '../Icon.svelte'

  export type MenuItem = { label: string; icon?: IconName; key?: string; danger?: boolean; run: () => void } | 'line'

  let { x, y, items, onclose }: { x: number; y: number; items: MenuItem[]; onclose: () => void } = $props()

  // Kept inside the window.
  let el = $state<HTMLElement>()
  let left = $state(0)
  let top = $state(0)
  $effect(() => {
    if (!el) return
    left = Math.max(4, Math.min(x, innerWidth - el.offsetWidth - 4))
    top = Math.max(4, Math.min(y, innerHeight - el.offsetHeight - 4))
  })

  function pick(item: Exclude<MenuItem, 'line'>) {
    onclose()
    item.run()
  }
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && onclose()} />

<div class="backdrop" role="presentation" onpointerdown={onclose} oncontextmenu={(e) => (e.preventDefault(), onclose())}></div>
<div class="menu" role="menu" bind:this={el} style:left="{left}px" style:top="{top}px">
  {#each items as item}
    {#if item === 'line'}
      <div class="line"></div>
    {:else}
      <button role="menuitem" class:danger={item.danger} onclick={() => pick(item)}>
        <span class="label">{#if item.icon}<Icon name={item.icon} size={11} />{/if}{item.label}</span>
        {#if item.key}<kbd>{item.key}</kbd>{/if}
      </button>
    {/if}
  {/each}
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 30;
  }
  .menu {
    position: fixed;
    z-index: 31;
    display: flex;
    flex-direction: column;
    min-width: 190px;
    padding: 4px;
    border: 1px solid var(--line);
    border-radius: 8px;
    background: var(--panel);
    box-shadow: 0 10px 30px rgb(0 0 0 / 0.35);
  }
  button {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    height: 26px;
    padding: 0 8px;
    border: 0;
    border-radius: 5px;
    background: none;
    font: 13px var(--sans);
    color: var(--fg);
    text-align: left;
  }
  .label {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  button:hover {
    background: var(--hover);
    color: var(--strong);
  }
  button.danger {
    color: var(--error);
  }
  kbd {
    border: 0;
    padding: 0;
    font-size: 11px;
    color: var(--faint);
  }
  .line {
    height: 1px;
    margin: 4px 6px;
    background: var(--line);
  }
</style>
