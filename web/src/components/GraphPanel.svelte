<script lang="ts">
  import { editor } from '../lib/editor.svelte';
  import type { TexNode } from '../lib/graph';

  let { name, nodes }: { name: string; nodes: TexNode[] } = $props();

  const names = (type: string) =>
    nodes.filter((n) => n.data.type === type).map((n) => (n.data.params.name as string) || (type === 'output' ? 'out' : '?'));

  const inputs = $derived(names('input'));
  const outputs = $derived(names('output'));
  const params = $derived(Object.entries(editor.exposed));

  let error = $state('');

  function rename(from: string, input: HTMLInputElement) {
    const to = input.value.trim();
    if (to === from) return;

    if (!/^[A-Za-z][A-Za-z0-9_]*$/.test(to) || to === 'seed' || to in editor.exposed) {
      error = `Can't use "${to}".`;
      input.value = from;
      return;
    }

    error = '';
    const { [from]: ex, ...rest } = editor.exposed;
    editor.exposed = { ...rest, [to]: ex };
  }

  function remove(param: string) {
    const { [param]: _, ...rest } = editor.exposed;
    editor.exposed = rest;
  }
</script>

<div class="flex flex-col gap-3 border-b border-line-soft p-3">
  <div>
    <div class="font-semibold">{name}</div>
    <div class="text-[11px] text-fg-dim">Usable as a node in other graphs.</div>
  </div>

  <div class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-[11px]">
    <span class="text-fg-dim">inputs</span>
    <span class="text-fg-muted">{inputs.join(', ') || 'none, add an input node'}</span>
    <span class="text-fg-dim">outputs</span>
    <span class="text-fg-muted">{outputs.join(', ') || 'out (main output)'}</span>
  </div>

  <div class="flex flex-col gap-1">
    <span class="text-[11px] text-fg-dim">params · ◇ next to a node param exposes it</span>
    {#each params as [param, ex] (param)}
      <div class="flex items-center gap-2">
        <input
          class="w-28 rounded border border-line bg-field px-1 py-0.5 font-mono text-xs"
          value={param}
          onchange={(e) => rename(param, e.currentTarget)}
          onkeydown={(e) => e.key === 'Enter' && e.currentTarget.blur()}
        />
        <span class="min-w-0 flex-1 truncate text-[11px] text-fg-dim" title={ex.targets.join(', ')}>
          {ex.targets.join(', ')}
        </span>
        <button class="text-[11px] text-fg-dim hover:text-danger" onclick={() => remove(param)}>remove</button>
      </div>
    {/each}
    {#if error}<span class="text-[11px] text-danger">{error}</span>{/if}
  </div>
</div>
