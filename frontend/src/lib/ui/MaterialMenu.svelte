<script lang="ts">
  import Icon from '../Icon.svelte'
  import { addDirt, dirt } from '../dirt'
  import { builtins, deleteMaterial, insertMaterial, library, loadLibrary, type Material } from '../materials.svelte'

  let { onclose }: { onclose: () => void } = $props()
  let confirm = $state('')

  loadLibrary()

  function pick(m: Material) {
    insertMaterial(m)
    onclose()
  }

  function remove(m: Material) {
    if (confirm !== m.file) {
      confirm = m.file ?? ''
      setTimeout(() => (confirm = ''), 2500)
      return
    }
    deleteMaterial(m)
  }
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && onclose()} />

<div class="backdrop" role="presentation" onclick={onclose}></div>
<div class="menu" role="menu">
  <div class="caps">dirt</div>
  {#each dirt as d}
    <button class="item" role="menuitem" onclick={() => (addDirt(d), onclose())}>{d.name}<kbd>{d.key}</kbd></button>
  {/each}
  <div class="caps">materials</div>
  {#each builtins as m}
    <button class="item" role="menuitem" onclick={() => pick(m)}>{m.name}</button>
  {/each}
  {#if library.saved.length}
    <div class="caps">yours</div>
    {#each library.saved as m (m.file)}
      <div class="row">
        <button class="item" role="menuitem" onclick={() => pick(m)}>{m.name}</button>
        <button class="x" class:armed={confirm === m.file} title="Delete the material" onclick={() => remove(m)}>
          {#if confirm === m.file}delete?{:else}<Icon name="x" size={9} />{/if}
        </button>
      </div>
    {/each}
  {/if}
  <p class="hint">dirt is one layer above the selected one · a material a folder; click surfaces to give it its zone</p>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 5;
  }
  .menu {
    position: absolute;
    top: 40px;
    left: 8px;
    right: 8px;
    z-index: 6;
    display: flex;
    flex-direction: column;
    gap: 1px;
    padding: 6px;
    border: 1px solid var(--line);
    border-radius: 8px;
    background: var(--panel);
    box-shadow: 0 10px 30px rgb(0 0 0 / 0.35);
  }
  .caps {
    padding: 6px 8px 4px;
  }
  .row {
    display: flex;
    align-items: center;
  }
  .item {
    flex: 1;
    height: 26px;
    padding: 0 8px;
    border: 0;
    border-radius: 6px;
    background: none;
    font: 13px var(--sans);
    color: var(--fg);
    text-align: left;
  }
  .item:has(kbd) {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  kbd {
    font: 10px var(--mono);
    color: var(--faint);
  }
  .item:hover {
    background: var(--hover);
    color: var(--strong);
  }
  .x {
    display: grid;
    place-items: center;
    min-width: 22px;
    height: 22px;
    padding: 0 4px;
    border: 0;
    border-radius: 4px;
    background: none;
    font: 11px var(--mono);
    color: var(--faint);
  }
  .x:hover {
    background: var(--hover);
    color: var(--strong);
  }
  .x.armed {
    color: var(--error);
  }
  .hint {
    margin: 6px 8px 2px;
    font-size: 11px;
    color: var(--faint);
  }
</style>
