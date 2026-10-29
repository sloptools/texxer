<script lang="ts">
  import { Handle, Position, type NodeProps } from '@xyflow/svelte';
  import { editor } from '../lib/editor.svelte';
  import { inputsOf, type TexNode } from '../lib/graph';
  import { renderer } from '../lib/renderer.svelte';

  let { id, data, selected }: NodeProps<TexNode> = $props();

  const def = $derived(editor.catalog[data.type]);
  const inputs = $derived(inputsOf(def, data));
  const outputs = $derived(def?.outputs ?? ['out']);
  const rows = $derived(Math.max(inputs.length, outputs.length));

  const thumb = $derived(renderer.thumbs[id]);
  const error = $derived(renderer.errors[id]);
</script>

<div
  class="w-36 overflow-hidden rounded-md border bg-panel shadow-lg shadow-shadow
    {selected ? 'border-accent' : error ? 'border-danger' : 'border-line'}"
>
  <div class="flex items-baseline justify-between gap-2 px-2 py-1">
    <span class="truncate font-medium">{id}</span>
    {#if id === editor.output}
      <span class="rounded bg-accent px-1 text-[10px] font-semibold text-on-accent">OUT</span>
    {:else}
      <span class="truncate text-[11px] text-fg-dim">{data.type}</span>
    {/if}
  </div>

  <div class="aspect-square bg-field" title={error}>
    {#if thumb}
      <img src={thumb} alt="" class="pixelated size-full {error ? 'opacity-30' : ''}" draggable="false" />
    {/if}
  </div>

  {#if rows > 0}
    <div class="py-1">
      {#each { length: rows } as _, i}
        <div class="relative flex h-5 items-center justify-between px-2 text-[11px] text-fg-muted">
          {#if inputs[i]}
            <Handle type="target" position={Position.Left} id={inputs[i]} />
            <span>{inputs[i]}</span>
          {:else}
            <span></span>
          {/if}

          {#if outputs[i]}
            <span>{outputs.length > 1 ? outputs[i] : ''}</span>
            <Handle type="source" position={Position.Right} id={outputs[i]} />
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>
