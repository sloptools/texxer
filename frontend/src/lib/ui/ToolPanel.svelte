<script lang="ts">
  import { imageURL } from '../assets'
  import { colorDecals, reliefDecals } from '../decals'
  import { newLayer, newMask } from '../project'
  import { fontShape, parseFont, shapes } from '../shapes'
  import { fonts, loadFonts, ready } from '../fonts.svelte'
  import { OpenFontsFolder } from '../../../wailsjs/go/main/App'
  import { store } from '../store.svelte'
  import { tools } from '../tools.svelte'
  import GrungePicker from './GrungePicker.svelte'
  import { snipTiles, tiled } from '../snippets.svelte'
  import Seg from './Seg.svelte'
  import ColorField from './ColorField.svelte'
  import Icon from '../Icon.svelte'
  import ShapeTile from './ShapeTile.svelte'
  import Slider from './Slider.svelte'
  import HelpLink from './HelpLink.svelte'
  import type { GizmoMode } from '../gl/volume'

  const gizmoModes: [GizmoMode, string][] = [
    ['translate', 'move W'],
    ['rotate', 'turn E'],
    ['scale', 'size R'],
  ]

  const brushKinds: [boolean, string][] = [
    [false, 'soft'],
    [true, 'pixel'],
  ]

  // Sliders write here and pass changes on, so a picked stamp follows them.
  let size = $state(tools.stamp.size)
  let rot = $state(tools.stamp.rot)
  let aspect = $state(tools.stamp.aspect)

  $effect(() => {
    size = tools.stamp.size
    rot = tools.stamp.rot
    aspect = tools.stamp.aspect
  })
  $effect(() => {
    if (size !== tools.stamp.size) tools.resize(size / tools.stamp.size)
  })
  $effect(() => {
    if (rot !== tools.stamp.rot) tools.rotate(rot - tools.stamp.rot)
  })
  $effect(() => {
    if (aspect !== tools.stamp.aspect) tools.setAspect(aspect)
  })

  const stampMask = $derived(store.layer?.masks.find((m) => m.kind === 'stamps'))
  const depths: [number, string][] = [
    [0, 'flat'],
    [1, 'sunk'],
    [-1, 'raised'],
  ]

  type Group = 'shapes' | 'relief' | 'decals' | 'images' | 'text'
  const groups: [Group, string][] = [
    ['shapes', 'shapes'],
    ['relief', 'relief'],
    ['decals', 'decals'],
    ['images', 'png'],
    ['text', 'text'],
  ]

  function groupOf(shape: string): Group {
    if (shape.startsWith('relief:')) return 'relief'
    if (colorDecals.includes(shape)) return 'decals'
    if (shape.startsWith('img:')) return 'images'
    if (fontOf(shape)) return 'text'
    return 'shapes'
  }

  let group = $state(groupOf(tools.stamp.shape))

  // Text stamps: the bitmap font, the condensed one they always had, and the fonts.
  type FontRow = { id: string; label: string; css: string }
  const fontRows = $derived<FontRow[]>([
    { id: 'pixel', label: 'pixel 5×7', css: 'var(--mono)' },
    { id: 'condensed', label: 'condensed', css: '700 1em "DIN Condensed", "Arial Narrow", sans-serif' },
    ...[...fonts.builtIn, ...fonts.own, ...fonts.system].map((f) => ({ id: f.name, label: f.name, css: `1em "${f.name}", sans-serif` })),
  ])

  function fontOf(shape: string): string {
    if (shape.startsWith('pixel:')) return 'pixel'
    if (shape.startsWith('text:')) return 'condensed'
    if (shape.startsWith('font:')) return parseFont(shape).name
    return ''
  }

  // ⌥ swaps the picked stamp's text instead.
  async function useFont(id: string, swap = false) {
    const value = tools.text || 'TEXT'
    if (id === 'pixel') tools.setShape('pixel:' + value, undefined, swap)
    else if (id === 'condensed') tools.setShape('text:' + value, undefined, swap)
    else {
      await ready(id)
      tools.setShape(fontShape(id, value), undefined, swap)
    }
  }

  // Fonts put into the folder show when the list opens again; the rows show the
  // text in their own font.
  $effect(() => {
    if (group !== 'text') return
    loadFonts().then(() => {
      for (const f of [...fonts.builtIn, ...fonts.own]) ready(f.name)
    })
  })

  // Relief and color decals need a layer that takes the stamps' colors, or they come
  // out as plain squares; ⌥ swaps the picked stamp's shape instead.
  function useDecal(shape: string, e: MouseEvent) {
    const blend = shape.startsWith('relief:') ? 'relief' : 'normal'
    const l = store.layer
    if (!e.altKey && (!l || !l.stampColors || l.blend !== blend)) {
      store.addLayer(newLayer(blend === 'relief' ? 'relief' : 'decals', '#808080', [newMask('stamps')], { stampColors: true, blend, tone: 0, grain: 0 }))
    }
    tools.setShape(shape, undefined, e.altKey)
  }

  async function useImage(name: string, swap = false) {
    const url = await imageURL(`/stamp?name=${encodeURIComponent(name)}`)
    if (!url) return
    const img = new Image()
    img.onload = () => tools.setShape('img:' + name, img.naturalHeight / img.naturalWidth, swap)
    img.src = url
  }
</script>

{#snippet mirror()}
  <button class:on={tools.mirror} title="Mirror across x (X)" onclick={() => tools.toggleMirror()}><Icon name="mirror" size={11} />mirror x<kbd>X</kbd></button>
{/snippet}

{#if tools.tool === 'brush'}
  <div class="section">
    <div class="caps">brush · {store.layer?.name} <HelpLink page="brush-stamps" /></div>
    <div class="toggles">
      <button class:on={tools.brush.erase} title="Erase (E, or hold ⌥)" onclick={() => (tools.brush.erase = !tools.brush.erase)}><Icon name="erase" size={11} />erase<kbd>E</kbd></button>
      {@render mirror()}
    </div>
    {#if store.layer?.paintColors}
      <div class="brushcolor">
        <div class="field"><ColorField bind:value={tools.brush.color} /></div>
        <button class="pipette" class:on={store.sampling} title="Take the brush color from the model (K)" onclick={() => (store.sampling = !store.sampling)}><Icon name="pipette" size={11} /></button>
      </div>
      {#if store.layer.brushColors?.length}
        <div class="swatches">
          {#each store.layer.brushColors as c}
            <button class="swatch" class:on={c === tools.brush.color.toLowerCase()} style:background={c} title={c} onclick={() => (tools.brush.color = c)} aria-label={c}></button>
          {/each}
          <button class="clear" title="Forget the painted colors; the palette stops keeping them" onclick={() => store.layer && (store.layer.brushColors = [])}>clear</button>
        </div>
      {/if}
    {/if}
    <Seg value={tools.brush.pixel} options={brushKinds} onchange={(v) => (tools.brush.pixel = v)} />
    {#if tools.brush.pixel}
      <Slider label="pixels" bind:value={tools.brush.pixels} min={1} max={16} step={1} digits={0} def={1} />
    {:else}
      <Slider label="size" bind:value={tools.brush.size} min={0.002} max={0.3} step={0.001} digits={3} def={0.03} />
      <Slider label="hardness" bind:value={tools.brush.hard} def={0.7} />
    {/if}
    <Slider label="strength" bind:value={tools.brush.strength} def={1} />
    <div class="caps sub">grunge · revealed by strength</div>
    <GrungePicker value={tools.brush.map} onpick={(name) => (tools.brush.map = name)} none />
    {#if tools.brush.map}
      <Slider label="scale" bind:value={tools.brush.mapScale} min={1} max={12} step={0.5} digits={1} def={3} />
    {/if}
  </div>
{:else if tools.tool === 'stamp'}
  <div class="section">
    <div class="caps">{tools.picked ? 'stamp' : 'new stamp'} · {store.layer?.name} <HelpLink page="brush-stamps" /></div>
    <div class="toggles">
      <button class:on={tools.snap} title="Snap to a grid on the surface (G)" onclick={() => (tools.snap = !tools.snap)}><Icon name="grid" size={11} />snap<kbd>G</kbd></button>
      {@render mirror()}
    </div>
    {#if stampMask}
      <Seg value={stampMask.depth ?? 0} options={depths} onchange={(v) => stampMask && (stampMask.depth = v)} />
    {/if}
    <Seg value={group} options={groups} onchange={(v) => (group = v)} />
    {#if group === 'shapes'}
      <div class="shapes">
        {#each shapes as shape}
          <ShapeTile {shape} on={tools.stamp.shape === shape} onclick={(e) => tools.setShape(shape, undefined, e.altKey)} />
        {/each}
      </div>
    {:else if group === 'relief'}
      <div class="shapes">
        {#each reliefDecals as shape}
          <ShapeTile {shape} on={tools.stamp.shape === shape} onclick={(e) => useDecal(shape, e)} />
        {/each}
      </div>
      <div class="note">bumps on a relief layer</div>
    {:else if group === 'decals'}
      <div class="shapes">
        {#each colorDecals as shape}
          <ShapeTile {shape} on={tools.stamp.shape === shape} onclick={(e) => useDecal(shape, e)} />
        {/each}
      </div>
      <div class="note">with their own colors, on a decals layer</div>
    {:else if group === 'images'}
      <div class="shapes">
        {#each store.stampImages as name}
          <ShapeTile shape={'img:' + name} on={tools.stamp.shape === 'img:' + name} onclick={(e) => useImage(name, e.altKey)} />
        {/each}
        <button class="import" title="Import PNG stamps" onclick={() => store.importStamps()}>+ png</button>
      </div>
    {:else}
      <div class="text">
        <input bind:value={tools.text} placeholder="text" onkeydown={(e) => e.key === 'Enter' && useFont(fontOf(tools.stamp.shape) || 'pixel')} />
      </div>
      <div class="fonts">
        {#each fontRows as f (f.id)}
          <button class="font" class:on={fontOf(tools.stamp.shape) === f.id} title={f.id === 'pixel' ? 'Bitmap font, crisp at low sizes' : undefined} onclick={(e) => useFont(f.id, e.altKey)}>
            <span class="sample" style:font={f.css}>{tools.text || 'TEXT'}</span>
            <span class="name">{f.label}</span>
          </button>
        {/each}
      </div>
      <div class="row between">
        <span class="note">your own: TTF, OTF, WOFF</span>
        <button class="link" title="Fonts you put there show when this list opens again" onclick={() => OpenFontsFolder()}>fonts folder</button>
      </div>
    {/if}
    <Slider label="size" bind:value={size} min={0.005} max={1} step={0.005} digits={3} def={0.12} />
    <Slider label="rotation" bind:value={rot} min={0} max={345} step={15} digits={0} def={0} />
    <Slider label="aspect" bind:value={aspect} min={0.05} max={4} step={0.01} def={1} />
    {#if tools.stamp.shape.startsWith('snip:')}
      <Slider label="tiles" bind:value={() => snipTiles(tools.stamp.shape).cols, (v) => tools.setShape(tiled(tools.stamp.shape, v, tools.stamp.aspect), undefined, true)} min={1} max={12} step={1} digits={0} def={1} />
    {/if}
    <Slider label="row gap" bind:value={tools.gap} min={0.5} max={4} step={0.05} def={1} />
    {#if tools.picked}
      <div class="row">
        <button onclick={() => tools.duplicate()}>duplicate <kbd>D</kbd></button>
        <button onclick={() => tools.remove()}>delete <kbd>⌫</kbd></button>
      </div>
    {/if}
  </div>
{:else if tools.tool === 'volume' && tools.volume}
  {@const v = tools.volume.volume}
  <div class="section">
    <div class="caps">volume · {store.layer?.name} <HelpLink page="masks" /></div>
    <Seg value={tools.gizmo} options={gizmoModes} onchange={(m) => (tools.gizmo = m)} />
    <div class="toggles">
      <button class:on={v.mirror} title="Mirror across x (X)" onclick={() => (v.mirror = !v.mirror)}><Icon name="mirror" size={11} />mirror x<kbd>X</kbd></button>
    </div>
  </div>
{/if}

<style>
  .swatches {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    align-items: center;
  }
  .swatch {
    width: 18px;
    height: 18px;
    padding: 0;
    border: 1px solid var(--line);
    border-radius: 4px;
  }
  .swatch.on {
    border-color: var(--strong);
  }
  .clear {
    margin-left: 4px;
    padding: 0;
    border: 0;
    background: none;
    font: 10px var(--mono);
    color: var(--faint);
  }
  .clear:hover {
    color: var(--strong);
  }
  .brushcolor {
    display: flex;
    gap: 6px;
    align-items: center;
  }
  .field {
    flex: 1;
    min-width: 0;
  }
  .pipette {
    display: grid;
    place-items: center;
    flex: none;
    width: 28px;
    height: 28px;
    padding: 0;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--bg);
    color: var(--dim);
  }
  .pipette.on {
    border-color: var(--accent);
    color: var(--accent);
  }
  .section {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 12px 14px 14px;
    border-bottom: 1px solid var(--line);
  }

  .sub {
    margin-top: 4px;
    font-size: 10px;
    color: var(--faint);
  }

  .shapes {
    display: grid;
    grid-template-columns: repeat(6, 1fr);
    gap: 4px;
  }

  .import {
    aspect-ratio: 1;
    padding: 0;
    justify-content: center;
    border-style: dashed;
    background: none;
    font-size: 10px;
  }

  .text,
  .row {
    display: flex;
    gap: 6px;
  }
  .toggles {
    display: flex;
    gap: 4px;
  }
  .toggles button {
    display: flex;
    gap: 6px;
    align-items: center;
    height: 24px;
    padding: 0 8px;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--bg);
    font: 12px var(--mono);
    color: var(--dim);
  }
  .toggles button:hover {
    color: var(--strong);
  }
  .toggles button.on {
    border-color: var(--accent);
    color: var(--accent);
  }
  .toggles kbd {
    color: var(--faint);
  }
  .text input {
    flex: 1;
    min-width: 0;
    height: 26px;
    padding: 0 8px;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--bg);
    font: 12px var(--mono);
    color: var(--fg);
    outline: none;
  }
  .text input:focus {
    border-color: var(--dim);
  }
  button {
    display: flex;
    align-items: center;
    gap: 6px;
    height: 26px;
    padding: 0 9px;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--raised);
    font: 12px var(--mono);
    color: var(--dim);
  }
  button:hover {
    color: var(--strong);
  }
  button.on {
    border-color: var(--accent);
    color: var(--strong);
  }

  .note {
    font-size: 11px;
    color: var(--faint);
  }
  .fonts {
    display: flex;
    flex-direction: column;
    max-height: 220px;
    overflow-y: auto;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--bg);
  }
  .fonts .font {
    flex: none;
    justify-content: space-between;
    gap: 10px;
    height: 32px;
    border: 0;
    border-radius: 0;
    background: none;
    color: var(--dim);
  }
  .fonts .font:hover {
    color: var(--strong);
  }
  .fonts .font.on {
    background: var(--active);
    color: var(--accent);
  }
  .sample {
    overflow: hidden;
    font-size: 17px !important;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .name {
    flex: none;
    font: 10px var(--mono);
    color: var(--faint);
  }
  .between {
    justify-content: space-between;
    align-items: center;
  }
  .link {
    height: auto;
    padding: 0;
    border: 0;
    background: none;
    font-size: 11px;
    color: var(--dim);
  }
  .link:hover {
    color: var(--strong);
  }
</style>
