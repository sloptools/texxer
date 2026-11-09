<script lang="ts">
  import * as api from '../../../wailsjs/go/main/App'
  import type { main } from '../../../wailsjs/go/models'
  import { store } from '../store.svelte'
  import { bridge } from '../bridge.svelte'
  import { uvOptions } from '../uvSource'
  import Seg from './Seg.svelte'
  import Slider from './Slider.svelte'
  import HelpLink from './HelpLink.svelte'

  const p = $derived(store.project)

  let stats = $state<main.UnwrapStats | null>(null)
  $effect(() => {
    const model = store.model
    const layout = p.unwrap
    stats = null
    if (model && layout) api.Unwrap(model.path, layout).then((s) => (stats = s)).catch((e) => store.say(String(e)))
  })

  const sizeName = (s: number) => (s === 1024 ? '1k' : String(s))
</script>

<div class="section">
  <div class="caps">texture <span class="of">{p.name}</span> <HelpLink page="uv" /></div>
  <Seg value={p.size} options={[[128, '128'], [256, '256'], [512, '512'], [1024, '1k']]} onchange={(v) => ((p.size = v), p.blendUV === 'texxer' && bridge.requestSync())} />
  <div class="row">
    <span>uv</span>
    <span class="pair">
      {#each uvOptions() as o}
        <button class:on={o.on} title={o.title} onclick={o.pick}>{o.label}</button>
      {/each}
    </span>
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
</div>

<style>
  .section {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 12px 14px 14px;
    border-top: 1px solid var(--line);
  }

  .row {
    display: flex;
    align-items: center;
    justify-content: space-between;
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
