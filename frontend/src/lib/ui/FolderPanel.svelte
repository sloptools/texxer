<script lang="ts">
  import { saveMaterial } from '../materials.svelte'
  import { store } from '../store.svelte'
  import HelpLink from './HelpLink.svelte'
  import Surfaces from './Surfaces.svelte'

  const group = $derived(store.group(store.selectedGroup))
  let name = $state('')

  $effect(() => {
    name = group?.name ?? ''
  })
</script>

{#if group}
  <div class="section">
    <div class="caps">folder · {group.name} <HelpLink page="layers" /></div>
    <p class="hint">its layers show only within its zone</p>
  </div>

  <div class="section">
    <div class="caps head">
      zone
      <span class="actions">
        <button class="text" class:on={store.pickingZone} title="While on, clicks on the model add or remove surfaces (Esc ends)" onclick={() => (store.pickingZone = !store.pickingZone)}>pick</button>
      </span>
    </div>
    <Surfaces picked={group.zone} />
    {#if !group.zone.parts.length && !group.zone.islands.length}
      <p class="hint">no zone: the folder covers everything</p>
    {/if}
  </div>

  <div class="section">
    <div class="caps">material</div>
    <div class="save">
      <input bind:value={name} placeholder="name" onkeydown={(e) => e.key === 'Enter' && name.trim() && saveMaterial(group, name.trim())} />
      <button disabled={!name.trim()} title="Save the folder's layers to use them on other models" onclick={() => saveMaterial(group, name.trim())}>save</button>
    </div>
    <p class="hint">surfaces, paint and stamps stay with this model</p>
  </div>
{/if}

<style>
  .section {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 12px 14px 14px;
  }
  .section + .section {
    border-top: 1px solid var(--line);
  }
  .head {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .hint {
    margin: 0;
    font-size: 11px;
    color: var(--faint);
  }


  .text {
    height: 20px;
    padding: 0 6px;
    border: 0;
    border-radius: 4px;
    background: none;
    font: 11px var(--mono);
    color: var(--faint);
    text-transform: none;
    letter-spacing: 0;
  }
  .text:hover {
    color: var(--strong);
  }
  .text.on {
    color: var(--accent);
  }
  .actions {
    display: flex;
    gap: 2px;
  }

  .save {
    display: flex;
    gap: 6px;
  }
  .save input {
    flex: 1;
    min-width: 0;
    height: 28px;
    padding: 0 8px;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--bg);
    font: 12px var(--sans);
    color: var(--strong);
    outline: none;
  }
  .save input:focus {
    border-color: var(--dim);
  }
  .save button {
    height: 28px;
    padding: 0 12px;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--raised);
    font: 12px var(--mono);
    color: var(--fg);
  }
  .save button:hover:not(:disabled) {
    border-color: var(--dim);
  }
  .save button:disabled {
    color: var(--faint);
  }
</style>
