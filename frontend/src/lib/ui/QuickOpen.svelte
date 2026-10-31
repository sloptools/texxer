<script lang="ts">
  import { selectModel, stateOf } from '../projects.svelte'
  import { store } from '../store.svelte'
  import type { ModelDoc } from '../types'

  let query = $state('')
  let cursor = $state(0)

  // Letters in order, in the name or the file's last folders; name hits first.
  function score(m: ModelDoc, q: string): number {
    if (!q) return 1
    const match = (text: string) => {
      let i = 0
      for (const c of text.toLowerCase()) if (c === q[i]) i++
      return i === q.length
    }
    if (match(m.name)) return 2
    return match(m.files.map((f) => f.split('/').slice(-3).join('/')).join(' ')) ? 1 : 0
  }

  const results = $derived.by(() => {
    const q = query.toLowerCase().replace(/\s+/g, '')
    return (store.workspace?.models ?? [])
      .map((m) => ({ m, s: score(m, q) }))
      .filter((r) => r.s > 0)
      .sort((a, b) => b.s - a.s)
      .map((r) => r.m)
  })

  $effect(() => {
    query
    cursor = 0
  })

  function pick(m: ModelDoc | undefined) {
    store.quickOpen = false
    if (m) selectModel(m.id)
  }

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'ArrowDown') cursor = Math.min(cursor + 1, results.length - 1)
    else if (e.key === 'ArrowUp') cursor = Math.max(cursor - 1, 0)
    else if (e.key === 'Enter') pick(results[cursor])
    else if (e.key === 'Escape') store.quickOpen = false
    else return
    e.preventDefault()
    e.stopPropagation()
  }

  function focus(el: HTMLInputElement) {
    el.focus()
  }
</script>

<div class="backdrop" role="presentation" onclick={() => (store.quickOpen = false)}>
  <div class="sheet" role="dialog" aria-label="Switch model" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={() => {}}>
    <input placeholder="switch to model…" bind:value={query} use:focus {onkeydown} />
    <div class="list">
      {#each results as m, i (m.id)}
        <button class:on={i === cursor} class:current={m.id === store.project.id} onmouseenter={() => (cursor = i)} onclick={() => pick(m)}>
          <span class="dot {stateOf(m)}"></span>
          <span class="name">{m.name}</span>
          <span class="where">{m.files[0]?.split('/').slice(-3, -1).join('/')}</span>
        </button>
      {:else}
        <p>no model matches</p>
      {/each}
    </div>
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 20;
    background: rgb(0 0 0 / 0.35);
  }
  .sheet {
    width: 440px;
    margin: 90px auto 0;
    padding: 8px;
    border: 1px solid var(--line);
    border-radius: 10px;
    background: var(--panel);
    box-shadow: 0 12px 40px rgb(0 0 0 / 0.4);
  }

  input {
    width: 100%;
    height: 32px;
    padding: 0 10px;
    border: 1px solid var(--line);
    border-radius: 7px;
    background: var(--bg);
    font: 13px var(--sans);
    color: var(--strong);
    outline: none;
  }
  input:focus {
    border-color: var(--dim);
  }

  .list {
    display: flex;
    flex-direction: column;
    max-height: 360px;
    margin-top: 6px;
    overflow-y: auto;
  }
  button {
    display: flex;
    align-items: center;
    gap: 8px;
    height: 30px;
    padding: 0 10px;
    border: 0;
    border-radius: 6px;
    background: none;
    font: 13px var(--sans);
    color: var(--fg);
    text-align: left;
  }
  button.on {
    background: var(--active);
    color: var(--strong);
  }
  button.current .name {
    color: var(--accent);
  }
  .name {
    flex: 1;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .where {
    font: 11px var(--mono);
    color: var(--faint);
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
  p {
    margin: 8px 10px;
    font-size: 12px;
    color: var(--faint);
  }
</style>
