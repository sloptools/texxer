<script lang="ts">
  import Icon from '../Icon.svelte'
  import { store } from '../store.svelte'
  import HelpLink from './HelpLink.svelte'

  let renaming = $state('')

  function rename(id: string, name: string) {
    const v = store.project.variants.find((v) => v.id === id)
    if (v && name.trim()) v.name = name.trim()
    renaming = ''
  }

  function focusInput(el: HTMLInputElement) {
    el.focus()
    el.select()
  }
</script>

<div class="bar" title="Variants: the export writes a texture for each (C cycles)">
  <button class="chip" class:on={!store.activeVariant} onclick={() => (store.variant = '')}>base</button>
  {#each store.project.variants as v (v.id)}
    {#if renaming === v.id}
      <input
        class="chip"
        value={v.name}
        use:focusInput
        onblur={(e) => rename(v.id, e.currentTarget.value)}
        onkeydown={(e) => {
          if (e.key === 'Enter') rename(v.id, e.currentTarget.value)
          if (e.key === 'Escape') renaming = ''
        }}
      />
    {:else}
      <span class="chip" class:on={store.variant === v.id}>
        <button class="name" onclick={() => (store.variant = v.id)} ondblclick={() => (renaming = v.id)}>{v.name}</button>
        {#if store.variant === v.id}
          <button class="x" title="Delete the variant" onclick={() => store.deleteVariant(v.id)}><Icon name="x" size={9} /></button>
        {/if}
      </span>
    {/if}
  {/each}
  <button class="add" title="New variant, like a player color" onclick={() => store.addVariant()}><Icon name="plus" size={11} /></button>
  <HelpLink page="variants" />
</div>

<style>
  .bar {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 4px;
    padding: 0 12px 8px;
  }

  .chip {
    display: flex;
    align-items: center;
    height: 22px;
    padding: 0 8px;
    border: 1px solid var(--line);
    border-radius: 5px;
    background: none;
    font: 11px var(--mono);
    color: var(--dim);
  }
  .chip:hover {
    color: var(--strong);
  }
  .chip.on {
    border-color: var(--accent);
    color: var(--accent);
  }
  span.chip {
    gap: 4px;
    padding: 0 4px 0 0;
  }
  input.chip {
    width: 90px;
    color: var(--strong);
    outline: none;
  }

  .name,
  .x,
  .add {
    padding: 0;
    border: 0;
    background: none;
    font: inherit;
    color: inherit;
  }
  .name {
    height: 100%;
    padding-left: 8px;
  }
  .x {
    display: grid;
    place-items: center;
    width: 14px;
    height: 14px;
    border-radius: 3px;
  }
  .x:hover {
    background: var(--hover);
  }
  .add {
    display: grid;
    place-items: center;
    width: 22px;
    height: 22px;
    border-radius: 4px;
    color: var(--dim);
  }
  .add:hover {
    background: var(--hover);
    color: var(--strong);
  }
</style>
