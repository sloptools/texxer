<script lang="ts">
  import { editor, toggleTheme } from '../lib/editor.svelte';

  let {
    graphs,
    name,
    trail,
    dirty,
    message,
    seed = $bindable(),
    previewSize = $bindable(),
    onopen,
    oncreate,
    onsave,
    onadd,
    onback,
    mode = $bindable(),
    onexport,
  }: {
    graphs: string[];
    name: string;
    /** Parent graphs when a subgraph was opened from them, outermost first. */
    trail: string[];
    dirty: boolean;
    message: string;
    seed: number;
    previewSize: number;
    onopen: (name: string) => void;
    oncreate: () => void;
    onsave: () => void;
    onadd: (x: number, y: number) => void;
    onback: (index: number) => void;
    mode: 'graph' | 'paint';
    /** Set when the graph textures a model. */
    onexport?: () => void;
  } = $props();

  const button = 'rounded border border-line px-2 py-0.5 hover:border-accent';
</script>

<header class="flex items-center gap-3 border-b border-line-soft bg-panel px-3 py-1.5">
  <span class="font-semibold tracking-tight text-accent">texxer</span>

  {#each trail as parent, i}
    <button class="text-fg-muted hover:text-accent" onclick={() => onback(i)} title="Back to {parent}">{parent}</button>
    <span class="text-fg-faint">›</span>
  {/each}

  <select class="rounded border border-line bg-field px-1 py-0.5" value={name} onchange={(e) => onopen(e.currentTarget.value)}>
    {#each graphs as g}
      <option value={g}>{g}</option>
    {/each}
  </select>
  <button class={button} onclick={oncreate}>new</button>
  <button class={button} onclick={onsave} title="⌘S">
    save{#if dirty}<span class="text-accent"> •</span>{/if}
  </button>
  <button class={button} onclick={(e) => onadd(e.clientX, e.clientY + 20)}>+ node</button>
  {#if onexport}
    <button class={button} onclick={onexport} title="Modell mit Textur als .glb herunterladen">export .glb</button>
  {/if}

  <div class="flex overflow-hidden rounded border border-line text-[11px]" title="Tab wechselt">
    {#each [['graph', 'Graph'], ['paint', 'Paint']] as [m, label]}
      <button
        class="px-2 py-0.5 {mode === m ? 'bg-line text-fg' : 'text-fg-muted hover:text-fg'}"
        onclick={() => (mode = m as 'graph' | 'paint')}>{label}</button
      >
    {/each}
  </div>

  <span class="truncate text-[11px] text-fg-dim">{message}</span>

  <label class="ml-auto flex items-center gap-1 text-fg-muted">
    seed
    <input type="number" class="w-14 rounded border border-line bg-field px-1 py-0.5" bind:value={seed} />
  </label>
  <label class="flex items-center gap-1 text-fg-muted">
    preview
    {#each trail as parent, i}
    <button class="text-fg-muted hover:text-accent" onclick={() => onback(i)} title="Back to {parent}">{parent}</button>
    <span class="text-fg-faint">›</span>
  {/each}

  <select class="rounded border border-line bg-field px-1 py-0.5" bind:value={previewSize}>
      <option value={256}>256</option>
      <option value={512}>512</option>
      <option value={1024}>1024</option>
    </select>
  </label>
  <button class={button} onclick={toggleTheme} title="Toggle theme">
    {editor.theme === 'dark' ? 'light' : 'dark'}
  </button>
</header>
