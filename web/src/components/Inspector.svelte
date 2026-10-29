<script lang="ts">
  import { editor } from '../lib/editor.svelte';
  import type { TexNode } from '../lib/graph';
  import { renderer } from '../lib/renderer.svelte';
  import ParamField from './ParamField.svelte';

  let {
    node,
    onparams,
    onoutput,
  }: {
    node: TexNode;
    onparams: (params: Record<string, unknown>) => void;
    onoutput: () => void;
  } = $props();

  const def = $derived(editor.catalog[node.data.type]);
  const error = $derived(renderer.errors[node.id]);

  function set(name: string, value: unknown) {
    onparams({ ...node.data.params, [name]: value });
  }

  function reset(name: string) {
    const { [name]: _, ...rest } = node.data.params;
    onparams(rest);
  }
</script>

<div class="flex flex-col gap-3 p-3">
  <div class="flex items-baseline justify-between">
    <div>
      <div class="font-semibold">{node.id}</div>
      <div class="text-[11px] text-zinc-500">{node.data.type} · {def?.category ?? 'unknown'}</div>
    </div>
    {#if node.id !== editor.output}
      <button class="rounded border border-zinc-700 px-2 py-0.5 text-[11px] hover:border-accent" onclick={onoutput}>
        set as output
      </button>
    {/if}
  </div>

  {#if error}
    <div class="rounded border border-red-500/50 bg-red-950/40 p-2 font-mono text-[11px] text-red-300">{error}</div>
  {/if}

  {#each def?.params ?? [] as param (param.name)}
    <ParamField
      {param}
      value={node.data.params[param.name]}
      onchange={(v) => set(param.name, v)}
      onreset={() => reset(param.name)}
    />
  {/each}
</div>
