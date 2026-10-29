<script lang="ts">
  import { untrack } from 'svelte';
  import * as api from '../../lib/api';
  import { Painter } from '../../lib/brush';
  import { editor } from '../../lib/editor.svelte';
  import type { TexNode } from '../../lib/graph';
  import { replace, stackOf, type Effect, type Layer } from '../../lib/layers';
  import { renderer } from '../../lib/renderer.svelte';
  import type { Graph, ModelRef, Pick } from '../../lib/types';
  import ParamField from '../ParamField.svelte';
  import Preview3D from '../Preview3D.svelte';
  import LayerList from './LayerList.svelte';
  import LayerProps from './LayerProps.svelte';

  let {
    node,
    model,
    graph,
    project,
    selected = $bindable(),
    showMask = $bindable(),
    onparams,
  }: {
    node: TexNode;
    model?: ModelRef;
    /** The current graph, for rendering the maps the brush needs. */
    graph: Graph;
    /** The project's graph name, used to name painted mask files. */
    project: string;
    selected: string;
    showMask: boolean;
    onparams: (params: Record<string, unknown>) => void;
  } = $props();

  let armed = $state(-1);
  let hint = $state('');
  let hoverPart = $state('');
  let lit = $state(false);
  /** Small renders of each layer's mask, by layer index. */
  let thumbs = $state<Record<number, string>>({});

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

  // Mask thumbnails follow each finished preview render, so they show the current stack.
  let thumbGen = 0;
  $effect(() => {
    if (!preview.url || preview.busy) return;
    const gen = ++thumbGen;
    const g = untrack(() => graph);
    const layers = untrack(() => stack.layers);

    (async () => {
      const next: Record<number, string> = {};
      for (const [i, l] of layers.entries()) {
        if (!l.mask?.length || l.hidden) continue;
        try {
          next[i] = await api.render(g, `${node.id}.mask${i}`, 48, 0);
        } catch {
          // A broken mask shows its error in the preview already.
        }
        if (gen !== thumbGen) return;
      }
      for (const url of Object.values(untrack(() => thumbs))) URL.revokeObjectURL(url);
      thumbs = next;
    })();
  });

  // The brush is active while a painted mask step is armed.
  const paintStep = $derived(armed >= 0 && layer?.mask?.[armed]?.kind === 'paint' ? armed : -1);
  let painter = $state.raw<Painter | undefined>();
  let overlayVersion = $state(0);
  let saving = Promise.resolve();

  $effect(() => {
    const i = paintStep;
    if (i < 0) {
      painter = undefined;
      return;
    }

    let cancelled = false;
    const file = untrack(() => layer?.mask?.[i]?.file);
    Painter.load(untrack(() => graph), node.id, editor.size, file)
      .then((p) => {
        if (cancelled) return;
        painter = p;
        overlayVersion++;
      })
      .catch((e) => (hint = `Pinsel: ${(e as Error).message}`));
    return () => (cancelled = true);
  });

  function paint(hit: Pick, _first: boolean, erase: boolean) {
    if (!painter) return;
    painter.stamp(hit.position, hit.normal, { ...editor.brush, erase: erase || editor.brush.erase });
    overlayVersion++;
  }

  /** Saves the mask after a stroke and bumps its revision, so the stack re-renders with it. */
  function paintEnd() {
    const i = paintStep;
    const p = painter;
    if (i < 0 || !p || !layer) return;

    const step = layer.mask![i];
    const file = step.file ?? `paint/${project}_${Math.random().toString(36).slice(2, 8)}.png`;
    saving = saving.then(async () => {
      try {
        await api.putFile(file, await p.png());
        const current = untrack(() => layer);
        if (!current?.mask?.[i]) return;
        setLayer({ ...current, mask: replace(current.mask, i, { ...current.mask[i], file, rev: (current.mask[i].rev ?? 0) + 1 }) });
      } catch (e) {
        hint = `Speichern: ${(e as Error).message}`;
      }
    });
  }

  function clearPaint(i: number) {
    if (paintStep === i && painter) {
      painter.clear();
      overlayVersion++;
      paintEnd();
      return;
    }
    const step = layer?.mask?.[i];
    if (step?.file) setLayer({ ...layer!, mask: replace(layer!.mask!, i, { kind: 'paint' }) });
  }

  function save(layers: Layer[], effects: Effect[]) {
    onparams({ ...node.data.params, layers, effects });
  }

  function setLayer(l: Layer) {
    save(replace(stack.layers, layerIndex, l), stack.effects);
  }

  /** Clicks in the 3D view go to the armed mask step: they toggle parts or set the picked color. */
  function pick(hit: Pick) {
    hint = hit.color ? `${hit.part} · ${hit.color}` : hit.part;
    if (!layer || armed < 0) return;

    const step = layer.mask?.[armed];
    if (!step) return;

    if (step.kind === 'parts') {
      const parts = step.parts ?? [];
      const next = parts.includes(hit.part) ? parts.filter((p) => p !== hit.part) : [...parts, hit.part];
      setLayer({ ...layer, mask: replace(layer.mask!, armed, { ...step, parts: next }) });
    } else if (step.kind === 'decal') {
      setLayer({ ...layer, mask: replace(layer.mask!, armed, { ...step, center: hit.position, direction: hit.normal }) });
      armed = -1;
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
    <LayerList layers={stack.layers} effects={stack.effects} {thumbs} bind:selected onchange={save} />
  </aside>

  <main class="relative min-w-0 flex-1 bg-panel">
    {#if preview.url}
      <Preview3D
        url={preview.url}
        {model}
        onpick={pick}
        autoRotate={false}
        brush={painter ? { radius: editor.brush.radius } : undefined}
        onpaint={paint}
        onpaintend={paintEnd}
        overlay={painter ? { canvas: painter.canvas, version: overlayVersion } : undefined}
        {lit}
        onhover={(p) => (hoverPart = p)}
      />
    {/if}
    <div class="pointer-events-none absolute inset-x-3 top-2 flex items-center gap-2 text-[11px] text-fg-dim">
      <span>{showMask && layer ? `Maske: ${layer.name}` : node.id}</span>
      {#if armed >= 0}<span class="text-accent">Klick wählt im Modell</span>{/if}
      {#if hoverPart}<span class="rounded bg-field px-1.5 text-fg">{hoverPart}</span>{/if}
      <span class="ml-auto">
        {#if preview.busy}rendering…{:else if preview.ms !== undefined}{preview.ms} ms{/if}
        {#if hint} · {hint}{/if}
      </span>
      <div class="pointer-events-auto flex overflow-hidden rounded border border-line" title="Textur unbeleuchtet oder mit Licht wie im Spiel">
        {#each [[false, 'Textur'], [true, 'Szene']] as [on, label]}
          <button class="px-2 py-0.5 {lit === on ? 'bg-line text-fg' : 'text-fg-muted hover:text-fg'}" onclick={() => (lit = on as boolean)}>{label}</button>
        {/each}
      </div>
    </div>
    {#if preview.error}
      <div class="absolute inset-x-3 bottom-3 rounded bg-danger-bg p-2 font-mono text-[11px] text-danger">{preview.error}</div>
    {/if}
  </main>

  <aside class="w-80 shrink-0 overflow-y-auto border-l border-line-soft bg-panel">
    {#if layer}
      {#key layerIndex}
        <LayerProps {layer} {armed} bind:showMask onchange={setLayer} onarm={(i) => (armed = i)} onclear={clearPaint} />
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
