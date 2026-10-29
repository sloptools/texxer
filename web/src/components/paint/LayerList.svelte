<script lang="ts">
  import { editor } from '../../lib/editor.svelte';
  import { effects as effectTypes, move, replace, type Effect, type Layer } from '../../lib/layers';

  let {
    layers,
    effects,
    selected = $bindable(),
    onchange,
  }: {
    layers: Layer[];
    effects: Effect[];
    /** "l3" for layer 3, "e1" for effect 1. */
    selected: string;
    onchange: (layers: Layer[], effects: Effect[]) => void;
  } = $props();

  // Top of the list is the top of the stack.
  const order = $derived(layers.map((_, i) => i).reverse());
  const selLayer = $derived(selected.startsWith('l') ? Number(selected.slice(1)) : -1);

  function setLayers(next: Layer[]) {
    onchange(next, effects);
  }

  function add() {
    const at = selLayer >= 0 ? selLayer + 1 : layers.length;
    const layer: Layer = { name: `Ebene ${layers.length + 1}`, fill: { type: 'mat_paint' } };
    setLayers([...layers.slice(0, at), layer, ...layers.slice(at)]);
    selected = `l${at}`;
  }

  function duplicate() {
    if (selLayer < 0) return;
    const copy = structuredClone($state.snapshot(layers[selLayer])) as Layer;
    copy.name += ' Kopie';
    setLayers([...layers.slice(0, selLayer + 1), copy, ...layers.slice(selLayer + 1)]);
    selected = `l${selLayer + 1}`;
  }

  function remove() {
    if (selLayer < 0) return;
    setLayers(layers.filter((_, i) => i !== selLayer));
    selected = layers.length > 1 ? `l${Math.max(0, selLayer - 1)}` : '';
  }

  function shift(dir: number) {
    if (selLayer < 0) return;
    const to = selLayer + dir;
    if (to < 0 || to >= layers.length) return;
    setLayers(move(layers, selLayer, to));
    selected = `l${to}`;
  }

  function addEffect(type: string) {
    onchange(layers, [...effects, { type }]);
    selected = `e${effects.length}`;
  }

  const button = 'rounded border border-line px-1.5 py-0.5 text-[11px] hover:border-accent disabled:opacity-40';
  const row = 'flex items-center gap-2 rounded px-2 py-1 cursor-pointer';
</script>

<div class="flex h-full flex-col">
  <div class="flex items-center gap-1 border-b border-line-soft p-2">
    <span class="mr-auto font-medium">Ebenen</span>
    <button class={button} onclick={add} title="Neue Ebene">+</button>
    <button class={button} onclick={duplicate} disabled={selLayer < 0} title="Duplizieren">⧉</button>
    <button class={button} onclick={() => shift(1)} disabled={selLayer < 0} title="Nach oben">↑</button>
    <button class={button} onclick={() => shift(-1)} disabled={selLayer < 0} title="Nach unten">↓</button>
    <button class={button} onclick={remove} disabled={selLayer < 0} title="Löschen">✕</button>
  </div>

  <div class="min-h-0 flex-1 overflow-y-auto p-1">
    {#each order as i (i)}
      {@const l = layers[i]}
      <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
      <div class="{row} {selected === `l${i}` ? 'bg-field text-accent' : 'hover:bg-field'}" onclick={() => (selected = `l${i}`)}>
        <button
          class="w-4 text-[11px] {l.hidden ? 'text-fg-faint' : 'text-fg-muted'}"
          title={l.hidden ? 'Einblenden' : 'Ausblenden'}
          onclick={(e) => {
            e.stopPropagation();
            setLayers(replace(layers, i, { ...l, hidden: !l.hidden }));
          }}>{l.hidden ? '○' : '●'}</button
        >
        <span class="truncate {l.hidden ? 'text-fg-dim' : ''}">{l.name}</span>
        <span class="ml-auto truncate text-[10px] text-fg-dim">
          {l.post ? 'post · ' : ''}{l.fill.type?.replace(/^mat_/, '') ?? 'Farbe'}
        </span>
      </div>
    {/each}

    <div class="mt-3 px-2 text-[10px] tracking-wide text-fg-dim uppercase">Effekte</div>
    {#each effects as fx, i}
      <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
      <div class="{row} {selected === `e${i}` ? 'bg-field text-accent' : 'hover:bg-field'}" onclick={() => (selected = `e${i}`)}>
        <button
          class="w-4 text-[11px] {fx.hidden ? 'text-fg-faint' : 'text-fg-muted'}"
          onclick={(e) => {
            e.stopPropagation();
            onchange(layers, replace(effects, i, { ...fx, hidden: !fx.hidden }));
          }}>{fx.hidden ? '○' : '●'}</button
        >
        <span class="truncate">{fx.type.replace(/^fx_/, '')}</span>
        <button
          class="ml-auto text-[11px] text-fg-dim hover:text-danger"
          onclick={(e) => {
            e.stopPropagation();
            onchange(layers, effects.filter((_, k) => k !== i));
            selected = '';
          }}>✕</button
        >
      </div>
    {/each}
    <select
      class="mx-2 mt-1 rounded border border-line bg-field px-1 py-0.5 text-[11px]"
      value=""
      onchange={(e) => {
        if (e.currentTarget.value) addEffect(e.currentTarget.value);
        e.currentTarget.value = '';
      }}
    >
      <option value="">+ Effekt…</option>
      {#each effectTypes(editor.catalog) as t}<option value={t}>{t}</option>{/each}
    </select>
  </div>
</div>
