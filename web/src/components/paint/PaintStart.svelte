<script lang="ts">
  import { onMount } from 'svelte';
  import * as api from '../../lib/api';
  import type { ModelInfo } from '../../lib/types';

  let { onopen }: { onopen: (model: ModelInfo) => void } = $props();

  let models = $state<ModelInfo[]>([]);
  let error = $state('');

  onMount(async () => {
    try {
      models = await api.listModels();
    } catch (e) {
      error = (e as Error).message;
    }
  });
</script>

<div class="flex min-h-0 flex-1 items-start justify-center overflow-y-auto bg-panel p-10">
  <div class="flex w-full max-w-md flex-col gap-3">
    <div class="font-semibold">Modell öffnen</div>
    <p class="text-fg-dim">
      Ein Modell (.glb) aus dem Graph-Ordner. texxer legt dafür ein Projekt mit Grundebenen an, das du hier mit Ebenen
      texturierst. Unter der Haube ist es ein Graph, Tab zeigt ihn.
    </p>
    {#if error}<p class="text-danger">{error}</p>{/if}
    {#each models as m}
      <button class="flex items-baseline gap-2 rounded border border-line px-3 py-2 text-left hover:border-accent" onclick={() => onopen(m)}>
        <span class="font-medium">{m.path}</span>
        <span class="ml-auto text-[11px] text-fg-dim">{m.materials.join(', ')}</span>
      </button>
    {:else}
      {#if !error}<p class="text-fg-dim">Keine .glb-Dateien gefunden. Lege Modelle in den Graph-Ordner, z. B. nach models/.</p>{/if}
    {/each}
  </div>
</div>
