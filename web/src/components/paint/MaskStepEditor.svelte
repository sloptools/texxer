<script lang="ts">
  import { editor } from '../../lib/editor.svelte';
  import { maskGraphs, meshSources, motifs, noises, type MaskStep } from '../../lib/layers';
  import type { ParamDef } from '../../lib/types';
  import ParamField from '../ParamField.svelte';

  let {
    step,
    first,
    armed,
    onchange,
    onremove,
    onarm,
    onclear,
  }: {
    step: MaskStep;
    first: boolean;
    /** True while clicks in the 3D view go to this step. */
    armed: boolean;
    onchange: (s: MaskStep) => void;
    onremove: () => void;
    onarm: () => void;
    /** Erases a painted mask. */
    onclear?: () => void;
  } = $props();

  const labels: Record<string, string> = {
    parts: 'Teile',
    mesh: 'Mesh-Map',
    noise: 'Noise',
    color: 'Farbe aus Textur',
    decal: 'Decal',
    paint: 'Gemalt',
    graph: 'Masken-Graph',
  };

  const slider = (name: string, def: number, max = 1): ParamDef => ({ name, kind: 'float', default: def, min: 0, max });

  function set(patch: Partial<MaskStep>) {
    onchange({ ...step, ...patch });
  }

  function unset(key: keyof MaskStep) {
    const { [key]: _, ...rest } = step;
    onchange(rest as MaskStep);
  }

  const nodeDef = $derived(step.type ? editor.catalog[step.type] : undefined);
  const field = 'rounded border border-line bg-field px-1 py-0.5 text-[11px]';
</script>

<div class="flex flex-col gap-2 rounded border border-line-soft p-2 {armed ? 'border-accent' : ''}">
  <div class="flex items-center gap-2">
    {#if !first}
      <select class={field} value={step.op ?? 'multiply'} onchange={(e) => set({ op: e.currentTarget.value as MaskStep['op'] })}>
        <option value="multiply">×</option>
        <option value="add">+</option>
        <option value="subtract">−</option>
      </select>
    {/if}
    <span class="font-medium">{labels[step.kind]}</span>
    <label class="ml-auto flex items-center gap-1 text-[11px] text-fg-muted">
      <input type="checkbox" class="accent-(--color-accent)" checked={!!step.invert} onchange={(e) => set({ invert: e.currentTarget.checked })} />
      invert
    </label>
    <button class="text-[11px] text-fg-dim hover:text-danger" onclick={onremove}>✕</button>
  </div>

  {#if step.kind === 'parts'}
    <div class="flex flex-wrap gap-1">
      {#each step.parts ?? [] as part}
        <span class="flex items-center gap-1 rounded bg-field px-1.5 py-0.5 text-[11px]">
          {part}
          <button class="text-fg-dim hover:text-danger" onclick={() => set({ parts: step.parts!.filter((p) => p !== part) })}>✕</button>
        </span>
      {/each}
    </div>
    <button class="self-start rounded border px-2 py-0.5 text-[11px] {armed ? 'border-accent text-accent' : 'border-line hover:border-accent'}" onclick={onarm}>
      {armed ? 'Teile im Modell anklicken…' : 'im Modell wählen'}
    </button>
  {:else if step.kind === 'mesh'}
    <select class={field} value={step.source} onchange={(e) => set({ source: e.currentTarget.value })}>
      {#each meshSources as s}<option value={s}>{s}</option>{/each}
    </select>
    <ParamField param={slider('level', 0.4)} value={step.level} onchange={(v) => set({ level: v as number })} onreset={() => unset('level')} />
    <ParamField param={slider('softness', 0.15)} value={step.softness} onchange={(v) => set({ softness: v as number })} onreset={() => unset('softness')} />
    <ParamField param={slider('breakup', 0.8, 2)} value={step.breakup} onchange={(v) => set({ breakup: v as number })} onreset={() => unset('breakup')} />
  {:else if step.kind === 'color'}
    <div class="flex items-center gap-2">
      <input type="color" class="h-7 w-10 cursor-pointer rounded border border-line bg-field" value={step.color} oninput={(e) => set({ color: e.currentTarget.value })} />
      <button class="rounded border px-2 py-0.5 text-[11px] {armed ? 'border-accent text-accent' : 'border-line hover:border-accent'}" onclick={onarm}>
        {armed ? 'Farbe im Modell anklicken…' : 'Pipette'}
      </button>
    </div>
    <ParamField param={slider('tolerance', 0.05, 0.5)} value={step.tolerance} onchange={(v) => set({ tolerance: v as number })} onreset={() => unset('tolerance')} />
    <ParamField param={{ ...slider('grow', 0, 32), px: true }} value={step.grow} onchange={(v) => set({ grow: v as number })} onreset={() => unset('grow')} />
  {:else if step.kind === 'paint'}
    <div class="flex items-center gap-2">
      <button class="rounded border px-2 py-0.5 text-[11px] {armed ? 'border-accent text-accent' : 'border-line hover:border-accent'}" onclick={onarm}>
        {armed ? 'malen beenden' : 'malen'}
      </button>
      <label class="flex items-center gap-1 text-[11px] text-fg-muted">
        <input type="checkbox" class="accent-(--color-accent)" bind:checked={editor.brush.erase} />
        radieren <span class="text-fg-faint">(⌥)</span>
      </label>
      {#if onclear}<button class="ml-auto text-[11px] text-fg-dim hover:text-danger" onclick={onclear}>leeren</button>{/if}
    </div>
    {#if armed}
      <p class="text-[11px] text-fg-dim">Ziehen malt, Leertaste halten dreht die Ansicht.</p>
    {/if}
    <ParamField param={{ ...slider('radius', 0.03, 0.2), min: 0.003 }} value={editor.brush.radius} onchange={(v) => (editor.brush.radius = v as number)} onreset={() => (editor.brush.radius = 0.03)} />
    <ParamField param={slider('hardness', 0.5)} value={editor.brush.hardness} onchange={(v) => (editor.brush.hardness = v as number)} onreset={() => (editor.brush.hardness = 0.5)} />
    <ParamField param={slider('strength', 0.6)} value={editor.brush.strength} onchange={(v) => (editor.brush.strength = v as number)} onreset={() => (editor.brush.strength = 0.6)} />
  {:else if step.kind === 'decal'}
    <select class={field} value={step.type} onchange={(e) => set({ type: e.currentTarget.value, params: {} })}>
      {#each motifs(editor.catalog) as t}<option value={t}>{t.replace(/^decal_/, '')}</option>{/each}
    </select>
    <div class="flex items-center gap-2">
      <button class="rounded border px-2 py-0.5 text-[11px] {armed ? 'border-accent text-accent' : 'border-line hover:border-accent'}" onclick={onarm}>
        {armed ? 'Stelle im Modell anklicken…' : step.center ? 'neu platzieren' : 'im Modell platzieren'}
      </button>
      {#if !step.center}<span class="text-[11px] text-fg-dim">noch nicht platziert</span>{/if}
    </div>
    <ParamField param={slider('size', 0.15)} value={step.size} onchange={(v) => set({ size: v as number })} onreset={() => unset('size')} />
    <ParamField param={{ ...slider('aspect', 1, 20), min: 0.1 }} value={step.aspect} onchange={(v) => set({ aspect: v as number })} onreset={() => unset('aspect')} />
    <ParamField param={{ ...slider('rotation', 0, 180), min: -180 }} value={step.rotation} onchange={(v) => set({ rotation: v as number })} onreset={() => unset('rotation')} />
    <ParamField param={slider('depth', 0.08, 0.5)} value={step.depth} onchange={(v) => set({ depth: v as number })} onreset={() => unset('depth')} />
    {#each (nodeDef?.params ?? []).filter((p) => p.name !== 'seed') as p (p.name)}
      <ParamField
        param={p}
        value={step.params?.[p.name]}
        onchange={(v) => set({ params: { ...step.params, [p.name]: v } })}
        onreset={() => {
          const { [p.name]: _, ...rest } = step.params ?? {};
          set({ params: rest });
        }}
      />
    {/each}
  {:else}
    <select class={field} value={step.type} onchange={(e) => set({ type: e.currentTarget.value, params: {} })}>
      {#if step.kind === 'graph'}<option value="">wählen…</option>{/if}
      {#each step.kind === 'noise' ? noises(editor.catalog) : maskGraphs(editor.catalog) as t}<option value={t}>{t}</option>{/each}
    </select>
    {#if step.kind === 'noise'}
      <label class="flex items-center gap-2 text-[11px] text-fg-muted">
        <input
          type="checkbox"
          class="accent-(--color-accent)"
          checked={step.level !== undefined}
          onchange={(e) => (e.currentTarget.checked ? set({ level: 0.5, softness: 0.2 }) : onchange({ ...step, level: undefined, softness: undefined }))}
        />
        hart abschneiden
      </label>
      {#if step.level !== undefined}
        <ParamField param={slider('level', 0.5)} value={step.level} onchange={(v) => set({ level: v as number })} onreset={() => set({ level: 0.5 })} />
        <ParamField param={slider('softness', 0.2)} value={step.softness} onchange={(v) => set({ softness: v as number })} onreset={() => unset('softness')} />
      {/if}
    {/if}
    {#each nodeDef?.params ?? [] as p (p.name)}
      <ParamField
        param={p}
        value={step.params?.[p.name]}
        onchange={(v) => set({ params: { ...step.params, [p.name]: v } })}
        onreset={() => {
          const { [p.name]: _, ...rest } = step.params ?? {};
          set({ params: rest });
        }}
      />
    {/each}
  {/if}
</div>
