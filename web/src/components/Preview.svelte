<script lang="ts">
  import { renderer } from '../lib/renderer.svelte';
  import type { StripBox } from '../lib/trim';
  import type { ModelRef } from '../lib/types';
  import Preview3D from './Preview3D.svelte';

  // strips are drawn as an overlay when a trim node is previewed.
  let {
    label,
    strips = [],
    size,
    model,
  }: { label: string; strips?: StripBox[]; size: number; model?: ModelRef } = $props();

  let mode = $state<'2d' | '3d'>('2d');

  // Graphs built on a model open in 3D.
  $effect(() => {
    if (model) mode = '3d';
  });
  let tiles = $state(2);

  const preview = $derived(renderer.preview);
</script>

<div class="flex min-h-0 flex-col border-b border-line-soft">
  <div class="flex items-center gap-2 border-b border-line-soft px-3 py-1.5">
    <span class="truncate font-medium">{label}</span>
    <span class="ml-auto text-[11px] text-fg-dim">
      {#if preview.busy}rendering…{:else if preview.ms !== undefined}{preview.ms} ms{/if}
    </span>
    <div class="flex overflow-hidden rounded border border-line text-[11px]">
      {#each ['2d', '3d'] as const as m}
        <button
          class="px-2 py-0.5 {mode === m ? 'bg-line text-fg' : 'text-fg-muted hover:text-fg'}"
          onclick={() => (mode = m)}>{m.toUpperCase()}</button
        >
      {/each}
    </div>
    {#if mode === '2d'}
      <select class="rounded border border-line bg-field px-1 text-[11px]" bind:value={tiles}>
        <option value={1}>1×1</option>
        <option value={2}>2×2</option>
        <option value={3}>3×3</option>
      </select>
    {/if}
  </div>

  <div class="relative aspect-square w-full bg-panel">
    {#if preview.url}
      {#if mode === '2d'}
        <div
          class="pixelated absolute inset-0"
          style:background-image="url({preview.url})"
          style:background-size="{100 / tiles}% {100 / tiles}%"
        ></div>
        {#each { length: tiles } as _, row}
          {#each strips as strip}
            <!-- Clipped to the strip height, so labels of thin strips don't overlap. -->
            <div
              class="absolute inset-x-0 overflow-hidden border-t border-accent/60"
              style:top="{((row + strip.y / size) / tiles) * 100}%"
              style:height="{(strip.height / size / tiles) * 100}%"
            >
              {#if row === 0}
                <span class="ml-1 rounded-b bg-accent/80 px-1 align-top text-[9px] leading-tight text-on-accent">
                  {strip.input}
                </span>
              {/if}
            </div>
          {/each}
        {/each}
      {:else}
        {#key model?.path}
          <Preview3D url={preview.url} {model} />
        {/key}
      {/if}
    {/if}

    {#if preview.error}
      <div class="absolute inset-x-2 bottom-2 rounded bg-danger-bg p-2 font-mono text-[11px] text-danger">
        {preview.error}
      </div>
    {/if}
  </div>
</div>
