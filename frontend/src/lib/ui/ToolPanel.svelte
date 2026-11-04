<script lang="ts">
  import { imageURL } from '../assets'
  import { colorDecals, reliefDecals } from '../decals'
  import { shapes } from '../shapes'
  import { store } from '../store.svelte'
  import { tools } from '../tools.svelte'
  import GrungePicker from './GrungePicker.svelte'
  import ShapeTile from './ShapeTile.svelte'
  import Slider from './Slider.svelte'
  import HelpLink from './HelpLink.svelte'

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

  const textShape = $derived('text:' + tools.text)
  const pixelShape = $derived('pixel:' + tools.text)

  async function useImage(name: string) {
    const url = await imageURL(`/stamp?name=${encodeURIComponent(name)}`)
    if (!url) return
    const img = new Image()
    img.onload = () => tools.setShape('img:' + name, img.naturalHeight / img.naturalWidth)
    img.src = url
  }
</script>

{#if tools.tool === 'brush'}
  <div class="section">
    <div class="caps">brush · {store.layer?.name} <HelpLink page="brush-stamps" /></div>
    <Slider label="size" bind:value={tools.brush.size} min={0.002} max={0.3} step={0.001} digits={3} def={0.03} />
    <Slider label="hardness" bind:value={tools.brush.hard} def={0.7} />
    <Slider label="strength" bind:value={tools.brush.strength} def={1} />
    <Slider label="grit" bind:value={tools.brush.grit} max={0.9} def={0} />
    <div class="caps sub">grunge · the brush reveals it, more with strength</div>
    <GrungePicker value={tools.brush.map} onpick={(name) => (tools.brush.map = name)} none />
    {#if tools.brush.map}
      <Slider label="scale" bind:value={tools.brush.mapScale} min={1} max={12} step={0.5} digits={1} def={3} />
    {/if}
    <p class="hint">paints into the layer's paint mask · [ ] size · ⌥ erases · ⇧click draws a line · grit breaks the brush up for rust and dirt</p>
  </div>
{:else if tools.tool === 'stamp'}
  <div class="section">
    <div class="caps">{tools.picked ? 'stamp' : 'new stamp'} · {store.layer?.name} <HelpLink page="brush-stamps" /></div>
    <div class="shapes">
      {#each shapes as shape}
        <ShapeTile {shape} on={tools.stamp.shape === shape} onclick={() => tools.setShape(shape)} />
      {/each}
    </div>
    <div class="caps sub">relief · relief blend, colors from stamps</div>
    <div class="shapes">
      {#each reliefDecals as shape}
        <ShapeTile {shape} on={tools.stamp.shape === shape} onclick={() => tools.setShape(shape)} />
      {/each}
    </div>
    <div class="caps sub">decals · colors from stamps</div>
    <div class="shapes">
      {#each colorDecals as shape}
        <ShapeTile {shape} on={tools.stamp.shape === shape} onclick={() => tools.setShape(shape)} />
      {/each}
    </div>
    <div class="caps sub">images</div>
    <div class="shapes">
      {#each store.stampImages as name}
        <ShapeTile shape={'img:' + name} on={tools.stamp.shape === 'img:' + name} onclick={() => useImage(name)} />
      {/each}
      <button class="import" title="Import PNG stamps" onclick={() => store.importStamps()}>+ png</button>
    </div>
    <div class="text">
      <input bind:value={tools.text} placeholder="text" onkeydown={(e) => e.key === 'Enter' && tools.setShape(pixelShape)} />
      <button class:on={tools.stamp.shape === pixelShape} title="Bitmap font, crisp at low sizes" onclick={() => tools.setShape(pixelShape)}>pixel</button>
      <button class:on={tools.stamp.shape === textShape} title="Condensed font" onclick={() => tools.setShape(textShape)}>text</button>
    </div>
    <Slider label="size" bind:value={size} min={0.005} max={1} step={0.005} digits={3} def={0.12} />
    <Slider label="rotation" bind:value={rot} min={0} max={345} step={15} digits={0} def={0} />
    <Slider label="aspect" bind:value={aspect} min={0.05} max={4} step={0.01} def={1} />
    <Slider label="row gap" bind:value={tools.gap} min={0.5} max={4} step={0.05} def={1} />
    {#if tools.picked}
      <div class="row">
        <button onclick={() => tools.duplicate()}>duplicate <kbd>D</kbd></button>
        <button onclick={() => tools.remove()}>delete <kbd>⌫</kbd></button>
      </div>
    {/if}
    <p class="hint">sits flat and upright on the face · ⇧click makes a row from the picked stamp · arrows nudge · R rotates 15° · esc deselects</p>
  </div>
{/if}

<style>
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

  .hint {
    margin: 0;
    font-size: 11px;
    color: var(--faint);
  }
</style>
