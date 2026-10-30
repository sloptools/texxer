<script lang="ts">
  import { store } from '../store.svelte'
  import { blends, maskKinds } from '../types'
  import ColorField from './ColorField.svelte'
  import MaskCard from './MaskCard.svelte'
  import Seg from './Seg.svelte'
  import Slider from './Slider.svelte'

  const layer = $derived(store.layer)

  // Colors of the other layers, for quick reuse.
  const used = $derived([...new Set(store.project.layers.map((l) => l.color.toLowerCase()))])
</script>

{#if layer}
  <div class="section">
    <div class="caps">color</div>
    <ColorField bind:value={layer.color} />
    {#if layer.masks.some((m) => m.kind === 'stamps')}
      <label class="check"><input type="checkbox" bind:checked={layer.stampColors} /> colors from image stamps</label>
    {/if}
    <div class="swatches">
      {#each used as c}
        <button class="sw" class:on={c === layer.color.toLowerCase()} style:background={c} title={c} aria-label={c} onclick={() => (layer.color = c)}></button>
      {/each}
    </div>
  </div>

  <div class="section">
    <div class="caps">surface</div>
    <Slider label="tone" bind:value={layer.tone} def={0.25} />
    {#if layer.tone > 0}
      <Slider label="tone scale" bind:value={layer.toneScale} min={1} max={64} step={1} digits={0} def={12} />
    {/if}
    <Slider label="grain" bind:value={layer.grain} max={0.3} def={0.06} />
    <Slider label="shine" bind:value={layer.shine} def={0} />
  </div>

  <div class="section">
    <div class="caps">blend</div>
    <Seg value={layer.blend} options={blends.map((b) => [b, b.slice(0, 4)])} onchange={(v) => (layer.blend = v)} />
    <Slider label="opacity" bind:value={layer.opacity} def={1} />
  </div>

  <div class="section">
    <div class="caps">masks</div>
    {#each layer.masks as mask, i (i + mask.kind + layer.id)}
      <MaskCard {mask} parts={store.model?.parts ?? []} onremove={() => layer.masks.splice(i, 1)} />
    {/each}
    {#if layer.masks.length < 4}
      <div class="add">
        {#each maskKinds as kind}
          <button onclick={() => store.addMask(kind)}>+ {kind}</button>
        {/each}
      </div>
    {/if}
    {#if !layer.masks.length}
      <p class="hint">no mask: the layer covers everything</p>
    {/if}
  </div>
{:else}
  <p class="hint pad">no layer selected</p>
{/if}

<style>
  .section {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 12px 14px 14px;
  }
  .section + .section {
    border-top: 1px solid var(--line);
  }

  .check {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    color: var(--dim);
  }
  .check input {
    accent-color: var(--accent);
    margin: 0;
  }

  .swatches {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }
  .sw {
    width: 18px;
    height: 18px;
    padding: 0;
    border: 0;
    border-radius: 4px;
    box-shadow: inset 0 0 0 1px rgb(0 0 0 / 0.25);
  }
  .sw.on {
    box-shadow:
      0 0 0 1px var(--panel),
      0 0 0 2px var(--accent);
  }

  .add {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }
  .add button {
    height: 22px;
    padding: 0 7px;
    border: 1px dashed var(--line);
    border-radius: 5px;
    background: none;
    font: 11px var(--mono);
    color: var(--dim);
  }
  .add button:hover {
    border-style: solid;
    color: var(--strong);
  }

  .hint {
    margin: 0;
    font-size: 11px;
    color: var(--faint);
  }
  .pad {
    padding: 14px;
  }
</style>
