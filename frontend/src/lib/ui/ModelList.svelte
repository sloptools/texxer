<script lang="ts">
  import Icon from '../Icon.svelte'
  import { addFiles, exportChanged, removeModel, selectModel, stateOf } from '../projects.svelte'
  import { store } from '../store.svelte'
  import type { ModelDoc } from '../types'
  import type { ModelState } from '../workspace'

  const labels: Record<ModelState, string> = {
    clean: 'exported',
    edited: 'edited',
    changed: 'file changed',
    new: 'not exported',
    missing: 'file missing',
  }

  let renaming = $state('')
  let confirm = $state('')
  let confirmTimer = 0

  const models = $derived(store.workspace?.models ?? [])
  const pending = $derived(models.filter((m) => ['edited', 'changed', 'new'].includes(stateOf(m))).length)

  function where(m: ModelDoc) {
    const parts = m.files[0]?.split('/') ?? []
    return parts.slice(-3, -1).join('/') + (m.files.length > 1 ? ` · ${m.files.length} files` : '')
  }

  function rename(m: ModelDoc, name: string) {
    if (name.trim()) m.name = name.trim()
    renaming = ''
  }

  function focusInput(el: HTMLInputElement) {
    el.focus()
    el.select()
  }

  function remove(m: ModelDoc) {
    if (confirm !== m.id) {
      confirm = m.id
      clearTimeout(confirmTimer)
      confirmTimer = setTimeout(() => (confirm = ''), 2500)
      return
    }
    confirm = ''
    removeModel(m.id)
  }
</script>

<div class="list">
  {#each models as m (m.id)}
    {@const state = stateOf(m)}
    <div class="row" class:on={m.id === store.project.id} role="option" aria-selected={m.id === store.project.id} tabindex="-1" onclick={() => selectModel(m.id)} ondblclick={() => (renaming = m.id)} onkeydown={() => {}}>
      <span class="dot {state}" title={labels[state]}></span>
      <span class="text">
        {#if renaming === m.id}
          <input
            value={m.name}
            use:focusInput
            onblur={(e) => rename(m, e.currentTarget.value)}
            onkeydown={(e) => {
              if (e.key === 'Enter') rename(m, e.currentTarget.value)
              if (e.key === 'Escape') renaming = ''
            }}
          />
        {:else}
          <span class="name">{m.name}</span>
        {/if}
        <span class="where">{where(m)}</span>
      </span>
      <span class="actions">
        <button class="icon" title="Add files that share this texture" onclick={(e) => (e.stopPropagation(), addFiles(m.id))}><Icon name="plus" size={11} /></button>
        <button class="icon" class:armed={confirm === m.id} title="Remove from the project (the files stay)" onclick={(e) => (e.stopPropagation(), remove(m))}>
          {#if confirm === m.id}remove?{:else}<Icon name="x" size={11} />{/if}
        </button>
      </span>
    </div>
  {/each}
</div>

{#if models.length}
  <div class="foot">
    <button class="wide" disabled={!pending} title="Export every model that was edited or whose files changed (⇧⌘E)" onclick={exportChanged}>
      export changed <span>{pending}</span>
    </button>
  </div>
{:else}
  <p class="hint">no models yet · add GLBs with + or drop them here</p>
{/if}

<style>
  .list {
    display: flex;
    flex-direction: column;
    gap: 1px;
    padding: 0 6px;
  }

  .row {
    display: flex;
    align-items: center;
    gap: 8px;
    height: 40px;
    padding: 0 6px 0 10px;
    border-radius: 6px;
    cursor: default;
  }
  .row:hover {
    background: var(--hover);
  }
  .row.on {
    background: var(--active);
  }

  .dot {
    flex: none;
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--faint);
  }
  .dot.edited,
  .dot.changed,
  .dot.new {
    background: var(--accent);
  }
  .dot.missing {
    background: var(--error);
  }

  .text {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-width: 0;
  }
  .name,
  .where {
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .name {
    color: var(--fg);
  }
  .row.on .name {
    color: var(--strong);
  }
  .where {
    font: 11px var(--mono);
    color: var(--faint);
  }
  input {
    width: 100%;
    padding: 0;
    border: 0;
    background: none;
    font: inherit;
    color: var(--strong);
    outline: none;
  }

  .actions {
    display: none;
    gap: 2px;
  }
  .row:hover .actions {
    display: flex;
  }
  .icon {
    display: grid;
    place-items: center;
    min-width: 22px;
    height: 22px;
    padding: 0 4px;
    border: 0;
    border-radius: 4px;
    background: none;
    font: 11px var(--mono);
    color: var(--dim);
  }
  .icon:hover {
    background: var(--hover);
    color: var(--strong);
  }
  .icon.armed {
    color: var(--error);
  }

  .foot {
    padding: 12px 12px 0;
  }
  .wide {
    display: flex;
    justify-content: space-between;
    width: 100%;
    height: 28px;
    padding: 0 10px;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--raised);
    font: 12px var(--mono);
    color: var(--fg);
  }
  .wide:hover:not(:disabled) {
    border-color: var(--dim);
    color: var(--strong);
  }
  .wide:disabled {
    background: none;
    color: var(--faint);
  }
  .wide span {
    color: var(--accent);
  }
  .wide:disabled span {
    color: var(--faint);
  }

  .hint {
    margin: 4px 16px;
    font-size: 12px;
    color: var(--faint);
  }
</style>
