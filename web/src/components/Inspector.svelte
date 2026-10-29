<script lang="ts">
  import { editor } from '../lib/editor.svelte';
  import type { TexNode } from '../lib/graph';
  import { renderer } from '../lib/renderer.svelte';
  import ParamField from './ParamField.svelte';

  let {
    node,
    onparams,
    onoutput,
    onrename,
  }: {
    node: TexNode;
    onparams: (params: Record<string, unknown>) => void;
    onoutput: () => void;
    /** Returns an error message if the name can't be used. */
    onrename: (id: string) => string | undefined;
  } = $props();

  let renameError = $state('');

  function rename(input: HTMLInputElement) {
    const id = input.value.trim();
    renameError = id === node.id ? '' : (onrename(id) ?? '');
    if (renameError) input.value = node.id;
  }

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
    <div class="min-w-0">
      <input
        class="w-full rounded bg-transparent font-semibold outline-none focus:bg-field"
        value={node.id}
        title="Rename"
        onchange={(e) => rename(e.currentTarget)}
        onkeydown={(e) => e.key === 'Enter' && e.currentTarget.blur()}
      />
      <div class="text-[11px] text-fg-dim">{node.data.type} · {def?.category ?? 'unknown'}</div>
      {#if renameError}<div class="text-[11px] text-danger">{renameError}</div>{/if}
    </div>
    {#if node.id !== editor.output}
      <button class="rounded border border-line px-2 py-0.5 text-[11px] hover:border-accent" onclick={onoutput}>
        set as output
      </button>
    {/if}
  </div>

  {#if error}
    <div class="rounded border border-danger bg-danger-bg p-2 font-mono text-[11px] text-danger">{error}</div>
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
