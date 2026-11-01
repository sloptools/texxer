<script lang="ts">
  import { saveMaterial } from '../materials.svelte'
  import { store } from '../store.svelte'
  import HelpLink from './HelpLink.svelte'

  const group = $derived(store.group(store.selectedGroup))
  const parts = $derived(store.model?.parts ?? [])
  let name = $state('')

  $effect(() => {
    name = group?.name ?? ''
  })

  function togglePart(i: number) {
    store.toggleZone('parts', i)
  }
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
        {#if group.zone.parts.length || group.zone.islands.length}
          <button class="text" title="Remove all parts and islands" onclick={() => (group.zone = { parts: [], islands: [] })}>clear</button>
        {/if}
        <button class="text" class:on={store.pickingZone} title="While on, clicks on the model add or remove surfaces (Esc ends)" onclick={() => (store.pickingZone = !store.pickingZone)}>pick</button>
      </span>
    </div>
    <p class="hint">
      {group.zone.islands.length}
      {group.zone.islands.length === 1 ? 'island' : 'islands'} ·
      {store.pickingZone ? 'click surfaces in the view, ⌘click whole parts' : 'pick to add surfaces from the view'}
    </p>
    <div class="parts">
      {#each parts as part, i}
        <button class="part" class:on={group.zone.parts.includes(i)} onclick={() => togglePart(i)}>{part || 'part ' + i}</button>
      {/each}
    </div>
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
    <p class="hint">parts, islands, paint and stamps stay with this model</p>
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

  .parts {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }
  .part {
    height: 22px;
    max-width: 100%;
    padding: 0 8px;
    overflow: hidden;
    border: 1px solid var(--line);
    border-radius: 5px;
    background: var(--bg);
    font: 11px var(--mono);
    color: var(--dim);
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .part:hover {
    color: var(--strong);
  }
  .part.on {
    border-color: var(--accent);
    color: var(--strong);
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
