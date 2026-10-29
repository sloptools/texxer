<script lang="ts">
  import { renderer } from '../lib/renderer.svelte';
  import Preview3D from './Preview3D.svelte';

  let { label }: { label: string } = $props();

  let mode = $state<'2d' | '3d'>('2d');
  let tiles = $state(2);

  const preview = $derived(renderer.preview);
</script>

<div class="flex min-h-0 flex-col border-b border-zinc-800">
  <div class="flex items-center gap-2 border-b border-zinc-800 px-3 py-1.5">
    <span class="truncate font-medium">{label}</span>
    <span class="ml-auto text-[11px] text-zinc-500">
      {#if preview.busy}rendering…{:else if preview.ms !== undefined}{preview.ms} ms{/if}
    </span>
    <div class="flex overflow-hidden rounded border border-zinc-700 text-[11px]">
      {#each ['2d', '3d'] as const as m}
        <button
          class="px-2 py-0.5 {mode === m ? 'bg-zinc-700 text-zinc-100' : 'text-zinc-400 hover:text-zinc-200'}"
          onclick={() => (mode = m)}>{m.toUpperCase()}</button
        >
      {/each}
    </div>
    {#if mode === '2d'}
      <select class="rounded border border-zinc-700 bg-zinc-800 px-1 text-[11px]" bind:value={tiles}>
        <option value={1}>1×1</option>
        <option value={2}>2×2</option>
        <option value={3}>3×3</option>
      </select>
    {/if}
  </div>

  <div class="relative aspect-square w-full bg-zinc-900">
    {#if preview.url}
      {#if mode === '2d'}
        <div
          class="pixelated absolute inset-0"
          style:background-image="url({preview.url})"
          style:background-size="{100 / tiles}% {100 / tiles}%"
        ></div>
      {:else}
        <Preview3D url={preview.url} />
      {/if}
    {/if}

    {#if preview.error}
      <div class="absolute inset-x-2 bottom-2 rounded bg-red-950/90 p-2 font-mono text-[11px] text-red-300">
        {preview.error}
      </div>
    {/if}
  </div>
</div>
