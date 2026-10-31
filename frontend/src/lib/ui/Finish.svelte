<script lang="ts">
  import { store } from '../store.svelte'
  import type { Finish } from '../types'
  import ColorField from './ColorField.svelte'
  import Seg from './Seg.svelte'
  import Slider from './Slider.svelte'

  const p = $derived(store.project)
  const sh = $derived(store.project.shading)

  const palettes: [Finish['palette'], string][] = [
    ['none', 'none'],
    ['auto', 'auto'],
    ['quake', 'quake'],
    ['file', 'file…'],
  ]
</script>

<div class="section">
  <div class="caps">texture</div>
  <Seg value={p.size} options={[[128, '128'], [256, '256'], [512, '512'], [1024, '1k']]} onchange={(v) => (p.size = v)} />
</div>

<div class="section">
  <div class="caps">crunch</div>
  <Seg value={p.finish.supersample} options={[[1, '1×'], [2, '2×'], [4, '4×']]} onchange={(v) => (p.finish.supersample = v)} />
  <Slider label="sharpen" bind:value={p.finish.sharpen} max={2} def={0.2} />
  <Seg value={p.finish.palette} options={palettes} onchange={(v) => store.setPalette(v)} />
  {#if p.finish.palette === 'auto'}
    <Slider label="colors" bind:value={p.finish.count} min={4} max={256} step={1} digits={0} def={48} />
  {/if}
  <Slider label="dither" bind:value={p.finish.dither} def={0} />
  <Slider label="glow" bind:value={p.finish.glow} min={0} max={8} step={0.1} def={3} digits={1} />
</div>

<div class="section">
  <div class="caps head">
    baked light
    <button class:on={sh.on} onclick={() => ((sh.on = !sh.on), (store.view = sh.on ? 'unlit' : 'lit'))}>{sh.on ? 'on' : 'off'}</button>
  </div>
  {#if sh.on}
    <Slider label="azimuth" bind:value={sh.azimuth} max={360} step={1} digits={0} def={60} />
    <Slider label="elevation" bind:value={sh.elevation} min={-30} max={90} step={1} digits={0} def={50} />
    <Slider label="key" bind:value={sh.key} def={0.45} />
    <ColorField label="key" bind:value={sh.keyColor} />
    <Slider label="form floor" bind:value={sh.formFloor} def={0.72} />
    <Slider label="ao floor" bind:value={sh.aoFloor} def={0.45} />
    <Slider label="under" bind:value={sh.under} def={0.12} />
    <ColorField label="under" bind:value={sh.underColor} />
    <Slider label="rim" bind:value={sh.rim} def={0.3} />
    <div class="row">
      <span>both sides</span>
      <button class:on={sh.bothSides} onclick={() => (sh.bothSides = !sh.bothSides)}>{sh.bothSides ? 'on' : 'off'}</button>
    </div>
  {/if}
</div>

<div class="section">
  <div class="caps">view light</div>
  <Slider label="key" bind:value={p.light.key} max={5} def={2.2} />
  <Slider label="ambient" bind:value={p.light.ambient} max={3} def={1} />
  <Slider label="headlight" bind:value={p.light.headlight} max={3} def={0.6} />
  <Slider label="angle" bind:value={p.light.angle} max={360} step={1} digits={0} def={35} />
</div>

<div class="section">
  <div class="caps">game light</div>
  <ColorField label="ambient" bind:value={p.light.gameAmbient} />
  <Slider label="energy" bind:value={p.light.gameEnergy} max={4} def={1.5} />
  <Slider label="top light" bind:value={p.light.gameTop} max={3} def={0.6} />
  <Slider label="torch" bind:value={p.light.torch} max={4} def={1} />
</div>

<style>
  .section {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 12px 14px 14px;
    border-top: 1px solid var(--line);
  }

  .head,
  .row {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .row {
    font-size: 12px;
    color: var(--dim);
  }

  button {
    height: 20px;
    padding: 0 7px;
    border: 1px solid var(--line);
    border-radius: 5px;
    background: var(--bg);
    font: 11px var(--mono);
    color: var(--faint);
    text-transform: none;
    letter-spacing: 0;
  }
  button.on {
    border-color: var(--accent);
    color: var(--accent);
  }
</style>
