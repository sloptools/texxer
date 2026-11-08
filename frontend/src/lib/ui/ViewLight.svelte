<script lang="ts">
  import { store } from '../store.svelte'
  import { gameTones } from '../types'
  import ColorField from './ColorField.svelte'
  import Slider from './Slider.svelte'

  let { onclose }: { onclose: () => void } = $props()

  const light = $derived(store.project.light)
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && onclose()} />

<div class="backdrop" role="presentation" onpointerdown={onclose}></div>
<div class="pop">
  {#if store.view === 'game'}
    <div class="caps">game light <span class="of">view only</span></div>
    <label class="tone">
      <span>tone</span>
      <select bind:value={light.gameTone} title="Tone mapping of the game view (I), as Godot's">
        {#each gameTones as t}<option value={t}>{t}</option>{/each}
      </select>
    </label>
    <ColorField label="ambient" bind:value={light.gameAmbient} />
    <Slider label="energy" bind:value={light.gameEnergy} max={4} def={1} />
    <Slider label="top light" bind:value={light.gameTop} max={3} def={0.6} />
    <Slider label="torch" bind:value={light.torch} max={4} def={1} />
  {:else}
    <div class="caps">view light <span class="of">view only</span></div>
    <Slider label="key" bind:value={light.key} max={5} def={2.2} />
    <Slider label="ambient" bind:value={light.ambient} max={3} def={1} />
    <Slider label="headlight" bind:value={light.headlight} max={3} def={0.6} />
    <Slider label="angle" bind:value={light.angle} max={360} step={1} digits={0} def={35} />
    <Slider label="bloom" bind:value={light.bloom} max={1.5} def={0.05} />
    <Slider label="torch" bind:value={light.torch} max={4} def={1} />
  {/if}
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 30;
  }
  .pop {
    position: fixed;
    right: 8px;
    bottom: 36px;
    z-index: 31;
    display: flex;
    flex-direction: column;
    gap: 6px;
    width: 240px;
    padding: 12px 14px 14px;
    border: 1px solid var(--line);
    border-radius: 8px;
    background: var(--panel);
    box-shadow: 0 10px 30px rgb(0 0 0 / 0.35);
  }

  .of {
    margin-left: 6px;
    color: var(--faint);
    text-transform: none;
  }

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
</style>
