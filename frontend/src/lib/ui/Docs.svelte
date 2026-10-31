<script lang="ts">
  import { pages } from '../docs'
  import { store } from '../store.svelte'

  const page = $derived(pages.find((p) => p.id === store.helpPage) ?? pages[0])
  let body: HTMLElement

  $effect(() => {
    page
    body?.scrollTo(0, 0)
  })
</script>

<div class="docs">
  <nav>
    {#each pages as p}
      <button class:on={p === page} onclick={() => (store.helpPage = p.id)}>{p.title}</button>
    {/each}
  </nav>
  <article bind:this={body}>
    {@html page?.html ?? ''}
  </article>
</div>

<style>
  .docs {
    display: grid;
    grid-template-columns: 170px minmax(0, 1fr);
    height: 100%;
    min-height: 0;
  }

  nav {
    display: flex;
    flex-direction: column;
    gap: 1px;
    padding: 12px 8px;
    border-right: 1px solid var(--line);
  }
  nav button {
    height: 28px;
    padding: 0 10px;
    border: 0;
    border-radius: 6px;
    background: none;
    font: 13px var(--sans);
    color: var(--dim);
    text-align: left;
  }
  nav button:hover {
    color: var(--strong);
  }
  nav button.on {
    background: var(--active);
    color: var(--strong);
  }

  article {
    padding: 4px 28px 28px;
    overflow-y: auto;
    line-height: 1.5;
    color: var(--fg);
  }
  article :global(h1) {
    margin: 18px 0 12px;
    font-size: 15px;
    font-weight: var(--title-weight);
    color: var(--strong);
  }
  article :global(h2) {
    margin: 22px 0 6px;
    font: 500 11px var(--mono);
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: var(--dim);
  }
  article :global(p),
  article :global(ul),
  article :global(ol) {
    margin: 0 0 10px;
  }
  article :global(ul),
  article :global(ol) {
    padding-left: 18px;
  }
  article :global(li) {
    margin: 2px 0;
  }
  article :global(strong) {
    font-weight: 500;
    color: var(--strong);
  }
  article :global(code) {
    font: 12px var(--mono);
    color: var(--strong);
  }
  article :global(img) {
    display: block;
    max-width: 100%;
    margin: 4px 0 14px;
    border: 1px solid var(--line);
    border-radius: 7px;
  }
  article :global(table) {
    width: 100%;
    margin: 4px 0 12px;
    border-collapse: collapse;
  }
  article :global(th) {
    padding: 4px 8px 4px 0;
    font: 500 11px var(--mono);
    text-align: left;
    text-transform: uppercase;
    color: var(--dim);
  }
  article :global(td) {
    padding: 4px 8px 4px 0;
    border-top: 1px solid var(--line);
    vertical-align: top;
  }
  article :global(td:first-child) {
    font: 12px var(--mono);
    color: var(--strong);
    white-space: nowrap;
  }
</style>
