<script lang="ts">
  import { libraryURL, snippets } from '../snippets.svelte'
  import { tools } from '../tools.svelte'

  snippets.load()

  const cuts = $derived(snippets.all)
</script>

<div class="strip">
  <select value={snippets.category} onchange={(e) => snippets.load((e.currentTarget as HTMLSelectElement).value)} title="Snippets of the project or a global category">
    <option value="">project</option>
    {#each snippets.categories as c}
      <option value={c}>{c}</option>
    {/each}
  </select>
  <div class="cuts">
    {#each cuts as cut (cut.id)}
      <button class="cut" class:on={tools.stamp.shape === 'snip:' + cut.id} title={cut.name} onclick={() => snippets.pick(cut)}>
        <img src={libraryURL('snippet', cut.id, snippets.category, snippets.versions[cut.id])} alt={cut.name} />
      </button>
    {:else}
      <span class="none">no snippets yet · cut them in photos, at the top</span>
    {/each}
  </div>
</div>

<style>
  .strip {
    position: absolute;
    left: 50%;
    bottom: 12px;
    transform: translateX(-50%);
    display: flex;
    align-items: center;
    gap: 8px;
    max-width: calc(100% - 40px);
    padding: 6px;
    border: 1px solid var(--line);
    border-radius: 9px;
    background: var(--panel);
    box-shadow: 0 8px 24px rgb(0 0 0 / 0.3);
  }
  select {
    height: 24px;
    border: 1px solid var(--line);
    border-radius: 5px;
    background: var(--bg);
    font: 11px var(--mono);
    color: var(--dim);
  }
  .cuts {
    display: flex;
    gap: 4px;
    overflow-x: auto;
  }
  .cut {
    flex: none;
    width: 48px;
    height: 48px;
    padding: 2px;
    border: 1px solid var(--line);
    border-radius: 5px;
    background: var(--bg);
  }
  .cut.on {
    border-color: var(--accent);
    box-shadow: 0 0 0 1px var(--accent);
  }
  .cut img {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: contain;
    image-rendering: pixelated;
  }
  .none {
    padding: 0 8px;
    font: 11px var(--mono);
    color: var(--faint);
    white-space: nowrap;
  }
</style>
