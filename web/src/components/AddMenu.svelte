<script lang="ts">
  import { editor } from '../lib/editor.svelte';

  let {
    x,
    y,
    onpick,
    onclose,
    exclude,
  }: { x: number; y: number; onpick: (type: string) => void; onclose: () => void; exclude?: string } = $props();

  const order = ['generator', 'grunge', 'mask', 'shape', 'filter', 'color', 'light', 'preset', 'graph', 'io', 'output'];

  let query = $state('');

  const groups = $derived.by(() => {
    const q = query.toLowerCase();
    const byCategory: Record<string, string[]> = {};

    for (const [type, def] of Object.entries(editor.catalog)) {
      if (type === exclude) continue;
      if (q && !type.includes(q) && !def.category.includes(q)) continue;
      (byCategory[def.category] ??= []).push(type);
    }
    return order
      .filter((c) => byCategory[c])
      .map((c) => ({ category: c, types: byCategory[c].sort() }));
  });

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onclose();
    if (e.key === 'Enter' && groups[0]) onpick(groups[0].types[0]);
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
<div class="fixed inset-0 z-40" onclick={onclose}></div>

<div
  class="fixed z-50 flex max-h-96 w-56 flex-col overflow-hidden rounded-md border border-line bg-panel shadow-xl"
  style:left="{x}px"
  style:top="{y}px"
>
  <!-- svelte-ignore a11y_autofocus -->
  <input
    class="border-b border-line bg-panel px-2 py-1.5 outline-none placeholder:text-fg-faint"
    placeholder="Add node…"
    bind:value={query}
    {onkeydown}
    autofocus
  />
  <div class="overflow-y-auto py-1">
    {#each groups as group}
      <div class="px-2 pt-1.5 pb-0.5 text-[10px] tracking-wide text-fg-dim uppercase">{group.category}</div>
      {#each group.types as type}
        <button class="block w-full px-3 py-0.5 text-left hover:bg-field hover:text-accent" onclick={() => onpick(type)}>
          {type}
        </button>
      {/each}
    {/each}
  </div>
</div>
