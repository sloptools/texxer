<script lang="ts">
  import { editor } from '../../lib/editor.svelte';
  import { blendModes, materials, newStep, replace, type Layer, type MaskKind, type MaskStep } from '../../lib/layers';
  import ParamField from '../ParamField.svelte';
  import MaskStepEditor from './MaskStepEditor.svelte';

  let {
    layer,
    armed,
    showMask = $bindable(),
    onchange,
    onarm,
  }: {
    layer: Layer;
    /** Index of the mask step clicks in the 3D view go to, or -1. */
    armed: number;
    showMask: boolean;
    onchange: (l: Layer) => void;
    onarm: (step: number) => void;
  } = $props();

  const mats = $derived(materials(editor.catalog));
  const fillDef = $derived(layer.fill.type ? editor.catalog[layer.fill.type] : undefined);
  const steps = $derived(layer.mask ?? []);

  function set(patch: Partial<Layer>) {
    onchange({ ...layer, ...patch });
  }

  function setFill(value: string) {
    set({ fill: value === '' ? { color: layer.fill.color ?? '#808080' } : { type: value, params: {} } });
  }

  function setFillParam(name: string, value: unknown) {
    set({ fill: { ...layer.fill, params: { ...layer.fill.params, [name]: value } } });
  }

  function resetFillParam(name: string) {
    const { [name]: _, ...rest } = layer.fill.params ?? {};
    set({ fill: { ...layer.fill, params: rest } });
  }

  function addStep(kind: MaskKind) {
    set({ mask: [...steps, newStep(kind)] });
    if (kind === 'parts' || kind === 'color') onarm(steps.length);
  }

  const field = 'rounded border border-line bg-field px-1 py-0.5';
  const section = 'text-[10px] tracking-wide text-fg-dim uppercase';
</script>

<div class="flex flex-col gap-3 p-3">
  <input
    class="w-full rounded bg-transparent font-semibold outline-none focus:bg-field"
    value={layer.name}
    onchange={(e) => set({ name: e.currentTarget.value })}
  />

  <div class="flex flex-col gap-2">
    <span class={section}>Füllung</span>
    <select class={field} value={layer.fill.type ?? ''} onchange={(e) => setFill(e.currentTarget.value)}>
      <option value="">Farbe</option>
      {#each mats as m}<option value={m}>{m.replace(/^mat_/, '')}</option>{/each}
    </select>
    {#if !layer.fill.type}
      <input
        type="color"
        class="h-7 w-12 cursor-pointer rounded border border-line bg-field"
        value={layer.fill.color ?? '#808080'}
        oninput={(e) => set({ fill: { color: e.currentTarget.value } })}
      />
    {/if}
    {#each (fillDef?.params ?? []).filter((p) => p.name !== 'seed') as p (p.name)}
      <ParamField param={p} value={layer.fill.params?.[p.name]} onchange={(v) => setFillParam(p.name, v)} onreset={() => resetFillParam(p.name)} />
    {/each}
  </div>

  <div class="flex flex-col gap-2">
    <span class={section}>Ebene</span>
    <ParamField
      param={{ name: 'opacity', kind: 'float', default: 1, min: 0, max: 1 }}
      value={layer.opacity}
      onchange={(v) => set({ opacity: v as number })}
      onreset={() => set({ opacity: undefined })}
    />
    <div class="flex items-center gap-2">
      <select class={field} value={layer.mode ?? 'normal'} onchange={(e) => set({ mode: e.currentTarget.value })}>
        {#each blendModes as m}<option value={m}>{m}</option>{/each}
      </select>
      <label class="flex items-center gap-1 text-[11px] text-fg-muted" title="Nach den Effekten, unbeleuchtet: für Glas oder Leuchtteile">
        <input type="checkbox" class="accent-(--color-accent)" checked={!!layer.post} onchange={(e) => set({ post: e.currentTarget.checked })} />
        nach Effekten
      </label>
    </div>
  </div>

  <div class="flex flex-col gap-2">
    <div class="flex items-center">
      <span class={section}>Maske</span>
      <label class="ml-auto flex items-center gap-1 text-[11px] text-fg-muted">
        <input type="checkbox" class="accent-(--color-accent)" bind:checked={showMask} />
        anzeigen
      </label>
    </div>
    {#if steps.length === 0}
      <p class="text-[11px] text-fg-dim">Ohne Maske deckt die Ebene alles.</p>
    {/if}
    {#each steps as step, i}
      <MaskStepEditor
        {step}
        first={i === 0}
        armed={armed === i}
        onchange={(s: MaskStep) => set({ mask: replace(steps, i, s) })}
        onremove={() => {
          set({ mask: steps.filter((_, k) => k !== i) });
          if (armed === i) onarm(-1);
        }}
        onarm={() => onarm(armed === i ? -1 : i)}
      />
    {/each}
    <div class="flex flex-wrap gap-1">
      {#each [['parts', 'Teile'], ['mesh', 'Mesh-Map'], ['noise', 'Noise'], ['color', 'Farbe'], ['graph', 'Graph']] as [kind, label]}
        <button class="rounded border border-line px-2 py-0.5 text-[11px] hover:border-accent" onclick={() => addStep(kind as MaskKind)}>
          + {label}
        </button>
      {/each}
    </div>
  </div>
</div>
