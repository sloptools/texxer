<script lang="ts">
  import { editor } from '../../lib/editor.svelte';
  import type { TexNode } from '../../lib/graph';
  import { replace, stackOf, type Effect, type Layer } from '../../lib/layers';
  import { renderer } from '../../lib/renderer.svelte';
  import type { ModelRef } from '../../lib/types';
  import ParamField from '../ParamField.svelte';
  import Preview3D from '../Preview3D.svelte';
  import LayerList from './LayerList.svelte';
  import LayerProps from './LayerProps.svelte';

  let {
    node,
    model,
    selected = $bindable(),
    showMask = $bindable(),
    onparams,
  }: {
    node: TexNode;
    model?: ModelRef;
    selected: string;
    showMask: boolean;
    onparams: (params: Record<string, unknown>) => void;
  } = $props();

  let armed = $state(-1);
  let hint = $state('');

  const stack = $derived(stackOf(node.data.params));
  const layerIndex = $derived(selected.startsWith('l') ? Number(selected.slice(1)) : -1);
  const effectIndex = $derived(selected.startsWith('e') ? Number(selected.slice(1)) : -1);
  const layer = $derived(stack.layers[layerIndex]);
  const fx = $derived(stack.effects[effectIndex]);
  const preview = $derived(renderer.preview);

  $effect(() => {
    selected;
    armed = -1;
  });

  function save(layers: Layer[], effects: Effect[]) {
    onparams({ ...node.data.params, layers, effects });
  }

  function setLayer(l: Layer) {
    save(replace(stack.layers, layerIndex, l), stack.effects);
  }

  /** Clicks in the 3D view go to the armed mask step: they toggle parts or set the picked color. */
  function pick(hit: { part: string; color?: string }) {
    hint = hit.color ? `${hit.part} · ${hit.color}` : hit.part;
    if (!layer || armed < 0) return;

    const step = layer.mask?.[armed];
    if (!step) return;

    if (step.kind === 'parts') {
      const parts = step.parts ?? [];
      const next = parts.includes(hit.part) ? parts.filter((p) => p !== hit.part) : [...parts, hit.part];
      setLayer({ ...layer, mask: replace(layer.mask!, armed, { ...step, parts: next }) });
    } else if (step.kind === 'color' && hit.color) {
      setLayer({ ...layer, mask: replace(layer.mask!, armed, { ...step, color: hit.color }) });
      armed = -1;
    }
  }

  function setEffectParam(name: string, value: unknown) {
    save(stack.layers, replace(stack.effects, effectIndex, { ...fx, params: { ...fx.params, [name]: value } }));
  }

  function resetEffectParam(name: string) {
    const { [name]: _, ...rest } = fx.params ?? {};
    save(stack.layers, replace(stack.effects, effectIndex, { ...fx, params: rest }));
  }
</script>

<div class="flex min-h-0 flex-1">
  <aside class="w-60 shrink-0 border-r border-line-soft bg-panel">
    <LayerList layers={stack.layers} effects={stack.effects} bind:selected onchange={save} />
  </aside>

  <main class="relative min-w-0 flex-1 bg-panel">
    {#if preview.url}
      <Preview3D url={preview.url} {model} onpick={pick} autoRotate={false} />
    {/if}
    <div class="pointer-events-none absolute inset-x-3 top-2 flex items-center gap-2 text-[11px] text-fg-dim">
      <span>{showMask && layer ? `Maske: ${layer.name}` : node.id}</span>
      {#if armed >= 0}<span class="text-accent">Klick wählt im Modell</span>{/if}
      <span class="ml-auto">
        {#if preview.busy}rendering…{:else if preview.ms !== undefined}{preview.ms} ms{/if}
        {#if hint} · {hint}{/if}
      </span>
    </div>
    {#if preview.error}
      <div class="absolute inset-x-3 bottom-3 rounded bg-danger-bg p-2 font-mono text-[11px] text-danger">{preview.error}</div>
    {/if}
  </main>

  <aside class="w-80 shrink-0 overflow-y-auto border-l border-line-soft bg-panel">
    {#if layer}
      {#key layerIndex}
        <LayerProps {layer} {armed} bind:showMask onchange={setLayer} onarm={(i) => (armed = i)} />
      {/key}
    {:else if fx}
      <div class="flex flex-col gap-3 p-3">
        <div class="font-semibold">{fx.type}</div>
        {#each (editor.catalog[fx.type]?.params ?? []).filter((p) => p.name !== 'seed') as p (p.name)}
          <ParamField param={p} value={fx.params?.[p.name]} onchange={(v) => setEffectParam(p.name, v)} onreset={() => resetEffectParam(p.name)} />
        {/each}
      </div>
    {:else}
      <p class="p-3 leading-relaxed text-fg-dim">
        Wähle links eine Ebene. Jede Ebene füllt ein Material durch eine Maske; oben liegt über unten. Effekte wie das
        gebackene Licht laufen über alle Ebenen, Ebenen mit „nach Effekten“ kommen danach.
        <br />Teile und Farben wählst du per Klick ins Modell, nachdem du in der Maske „im Modell wählen“ oder
        „Pipette“ gedrückt hast. Ziehen dreht die Ansicht.
      </p>
    {/if}
  </aside>
</div>
