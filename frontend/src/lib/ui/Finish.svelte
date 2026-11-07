<script lang="ts">
  import * as api from '../../../wailsjs/go/main/App'
  import type { main } from '../../../wailsjs/go/models'
  import { store } from '../store.svelte'
  import { bridge } from '../bridge.svelte'
  import { setTileSize } from '../projects.svelte'
  import type { Finish, GameTone } from '../types'
  import ColorField from './ColorField.svelte'
  import Seg from './Seg.svelte'
  import Slider from './Slider.svelte'
  import HelpLink from './HelpLink.svelte'

  const p = $derived(store.project)
  const tones: GameTone[] = ['standard', 'reinhard', 'filmic', 'aces', 'agx', 'substratum']
  const sh = $derived(store.project.shading)

  let stats = $state<main.UnwrapStats | null>(null)
  $effect(() => {
    const model = store.model
    const layout = p.unwrap
    stats = null
    if (model && layout) api.Unwrap(model.path, layout).then((s) => (stats = s)).catch((e) => store.say(String(e)))
  })

  const sizeName = (s: number) => (s === 1024 ? '1k' : String(s))
  // Powers of two, so the tile takes whole texels and its mipmaps repeat too.
  const meters: [number, string][] = [0.5, 1, 2, 4].map((m) => [m, String(m)])

  const palettes: [Finish['palette'], string][] = [
    ['none', 'none'],
    ['auto', 'auto'],
    ['quake', 'quake'],
    ['file', 'file…'],
  ]
</script>

<div class="section">
  <div class="caps">texture <span class="of">{store.project.name}</span> <HelpLink page="uv" /></div>
  <Seg value={p.size} options={[[128, '128'], [256, '256'], [512, '512'], [1024, '1k']]} onchange={(v) => ((p.size = v), p.blendUV === 'texxer' && bridge.requestSync())} />
  {#if p.tile}
    {@const [w, h] = p.tile}
    <div class="row"><span>width m</span><Seg value={w} options={meters} onchange={(v) => setTileSize(v, h)} /></div>
    <div class="row"><span>height m</span><Seg value={h} options={meters} onchange={(v) => setTileSize(w, v)} /></div>
    <div class="note">{p.size * w / Math.max(w, h)} × {p.size * h / Math.max(w, h)} px · {Math.round(p.size / Math.max(w, h))} px/m · tiles seamlessly</div>
  {:else}
    <div class="row">
      <span>uv</span>
      {#if p.blend}
        <span class="pair">
          <button class:on={!p.blendUV} title="Paint on Blender's own UVs" onclick={() => bridge.setUV(undefined)}>blender</button>
          <button class:on={p.blendUV === 'texxer'} title="texxer lays the UVs out on each sync and writes them into Blender, as the UV map 'texxer'" onclick={() => bridge.setUV('texxer')}>texxer</button>
          <button class:on={p.blendUV === 'locked'} title="Keep texxer's UVs in Blender as they are, so painted masks stay put" onclick={() => bridge.setUV('locked')}>locked</button>
        </span>
      {:else}
        <span class="pair">
          <button class:on={!p.unwrap} title="Paint on the model's own UVs" onclick={() => store.setLayout(0)}>model</button>
          <button class:on={!!p.unwrap} title="Let texxer lay the UVs out: cut at hard edges, upright, one texel density" onclick={() => store.setLayout(p.unwrap || p.size)}>texxer</button>
        </span>
      {/if}
    </div>
    <Slider label="edge width" bind:value={p.bake.edges} min={0.002} max={0.08} step={0.001} digits={3} def={0.03} />
    <Slider label="cavity width" bind:value={p.bake.cavity} min={0.002} max={0.08} step={0.001} digits={3} def={0.03} />
    {#if store.model?.details}
      <Slider label="detail maps" bind:value={p.bake.detail} min={0} max={2} step={0.05} def={1} />
    {/if}
    {#if p.unwrap}
      <div class="note">
        {#if stats}{stats.charts} charts · {Math.round(stats.fill * 100)}% filled · {Math.round(stats.density)} px/m{:else}laying out…{/if}
      </div>
      {#if p.unwrap !== p.size}
        <button class="wide" onclick={() => store.setLayout(p.size)}>lay out for {sizeName(p.size)} <span>now {sizeName(p.unwrap)}</span></button>
      {/if}
    {:else if store.model && store.model.overlap > 0.01}
      <div class="note warn">{store.model.overlap >= 1 ? 'the model has no UVs' : `${Math.round(store.model.overlap * 100)}% of the UVs overlap`}{p.blend ? ' · let texxer lay them out' : ''}</div>
    {/if}
  {/if}
</div>

<div class="section">
  <div class="caps">crunch <span class="of">all models</span> <HelpLink page="look" /></div>
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
  <Slider label="bloom" bind:value={p.light.bloom} max={1.5} def={0.05} />
</div>

<div class="section">
  <div class="caps">game light</div>
  <label class="tone">
    <span>tone</span>
    <select bind:value={p.light.gameTone} title="Tone mapping of the game view (I), as Godot's; substratum adds its grade">
      {#each tones as t}<option value={t}>{t}</option>{/each}
    </select>
  </label>
  <ColorField label="ambient" bind:value={p.light.gameAmbient} />
  <Slider label="energy" bind:value={p.light.gameEnergy} max={4} def={1} />
  <Slider label="top light" bind:value={p.light.gameTop} max={3} def={0.6} />
  <Slider label="torch" bind:value={p.light.torch} max={4} def={1} />
</div>

<style>
  .tone {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 26px;
    font-size: 12px;
    color: var(--dim);
  }
  .tone select {
    height: 24px;
    border: 1px solid var(--line);
    border-radius: 5px;
    background: var(--bg);
    font: 11px var(--mono);
    color: var(--fg);
  }

  .section {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 12px 14px 14px;
    border-top: 1px solid var(--line);
  }
  .section:first-child {
    padding-top: 4px;
    border-top: 0;
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

  .of {
    margin-left: 6px;
    color: var(--faint);
    text-transform: none;
  }

  .pair {
    display: flex;
    gap: 4px;
  }
  .note {
    font: 11px var(--mono);
    color: var(--faint);
  }
  .note.warn {
    color: var(--accent);
  }
  .wide {
    display: flex;
    justify-content: space-between;
    height: 24px;
    color: var(--dim);
  }
  .wide:hover {
    color: var(--strong);
  }
  .wide span {
    color: var(--faint);
  }
</style>
