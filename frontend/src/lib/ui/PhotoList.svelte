<script lang="ts">
  import Icon from '../Icon.svelte'
  import { libraryURL, snippets, type PhotoEntry } from '../snippets.svelte'

  let confirm = $state('')
  let naming = $state(false)

  snippets.load()

  function remove(p: PhotoEntry) {
    if (confirm !== p.id) {
      confirm = p.id
      setTimeout(() => (confirm = ''), 2500)
      return
    }
    snippets.deletePhoto(p)
  }

  function addCategory(e: KeyboardEvent) {
    const input = e.currentTarget as HTMLInputElement
    if (e.key === 'Enter' && input.value.trim()) {
      snippets.addCategory(input.value)
      naming = false
    } else if (e.key === 'Escape') naming = false
  }

  function focus(el: HTMLInputElement) {
    el.focus()
  }
</script>

<div class="libs">
  <button class="lib" class:on={!snippets.category} onclick={() => snippets.load('')}>project</button>
  {#each snippets.categories as c}
    <button class="lib" class:on={snippets.category === c} onclick={() => snippets.load(c)}>{c}</button>
  {/each}
  {#if naming}
    <input class="name" placeholder="category" use:focus onkeydown={addCategory} onblur={() => (naming = false)} />
  {:else}
    <button class="lib add" title="A new global category, for snippets every project can use" onclick={() => (naming = true)}>+</button>
  {/if}
</div>

<div class="list">
  {#each snippets.photos as p (p.id)}
    <div class="row" class:on={p.id === snippets.photo} role="option" aria-selected={p.id === snippets.photo} tabindex="-1" onclick={() => ((snippets.photo = p.id), (snippets.cut = ''))} onkeydown={() => {}}>
      <img src={libraryURL('photo', p.file, snippets.category)} alt="" />
      <span class="text">{p.cuts.length} {p.cuts.length === 1 ? 'snippet' : 'snippets'}</span>
      <button class="x" class:armed={confirm === p.id} title="Delete the photo and its snippets" onclick={(e) => (e.stopPropagation(), remove(p))}>
        {#if confirm === p.id}delete?{:else}<Icon name="x" size={9} />{/if}
      </button>
    </div>
  {:else}
    <p class="note">drop photos here, or add them with +</p>
  {/each}
  <button class="addphotos" onclick={() => snippets.addPhotos()}><Icon name="plus" size={11} />add photos</button>
</div>

<style>
  .libs {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    padding: 8px 12px;
  }
  .lib {
    height: 22px;
    padding: 0 8px;
    border: 1px solid var(--line);
    border-radius: 5px;
    background: none;
    font: 11px var(--mono);
    color: var(--dim);
  }
  .lib.on {
    border-color: var(--accent);
    color: var(--accent);
  }
  .lib.add {
    border-style: dashed;
  }
  .name {
    width: 90px;
    height: 22px;
    padding: 0 6px;
    border: 1px solid var(--accent);
    border-radius: 5px;
    background: var(--bg);
    font: 11px var(--mono);
    color: var(--strong);
  }

  .list {
    display: flex;
    flex-direction: column;
    gap: 1px;
    padding: 0 6px;
  }
  .row {
    display: flex;
    align-items: center;
    gap: 10px;
    height: 52px;
    padding: 0 6px;
    border-radius: 6px;
  }
  .row:hover {
    background: var(--hover);
  }
  .row.on {
    background: var(--active);
  }
  .row img {
    width: 56px;
    height: 42px;
    object-fit: cover;
    border-radius: 4px;
    background: var(--raised);
  }
  .text {
    flex: 1;
    font: 11px var(--mono);
    color: var(--dim);
  }
  .x {
    height: 20px;
    padding: 0 5px;
    border: 0;
    border-radius: 4px;
    background: none;
    font: 10px var(--mono);
    color: var(--faint);
    opacity: 0;
  }
  .row:hover .x,
  .x.armed {
    opacity: 1;
  }
  .x.armed {
    color: var(--error);
  }
  .note {
    margin: 6px 6px 10px;
    font-size: 12px;
    color: var(--faint);
  }
  .addphotos {
    display: flex;
    align-items: center;
    gap: 6px;
    align-self: flex-start;
    height: 24px;
    margin: 8px 6px;
    padding: 0 8px;
    border: 1px dashed var(--line);
    border-radius: 5px;
    background: none;
    font: 11px var(--mono);
    color: var(--dim);
  }
  .addphotos:hover {
    border-style: solid;
    color: var(--strong);
  }
</style>
