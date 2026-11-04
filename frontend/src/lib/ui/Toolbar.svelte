<script lang="ts">
  import { tools } from '../tools.svelte'
  import type { GizmoMode } from '../gl/volume'
  import type { Tool } from '../types'

  const list: [Tool, string, string][] = [
    ['nav', 'view', 'esc'],
    ['brush', 'brush', 'B'],
    ['stamp', 'stamp', 'S'],
    ['volume', 'volume', 'Q'],
  ]
  const modes: [GizmoMode, string, string][] = [
    ['translate', 'move', 'W'],
    ['rotate', 'turn', 'E'],
    ['scale', 'size', 'R'],
  ]
</script>

<div class="bar">
  <div class="group">
    {#each list as [tool, label, key]}
      <button class:on={tools.tool === tool} onclick={() => (tools.tool === tool ? null : tools.set(tool))}>
        {label}<kbd>{key}</kbd>
      </button>
    {/each}
  </div>

  {#if tools.tool === 'volume' && tools.volume}
    {@const v = tools.volume.volume}
    <div class="group">
      {#each modes as [mode, label, key]}
        <button class:on={tools.gizmo === mode} onclick={() => (tools.gizmo = mode)}>{label}<kbd>{key}</kbd></button>
      {/each}
    </div>
    <div class="group">
      <button class:on={v.mirror} title="Mirror across X (X)" onclick={() => (v.mirror = !v.mirror)}>mirror x<kbd>X</kbd></button>
    </div>
  {:else if tools.tool !== 'nav' && tools.tool !== 'volume'}
    <div class="group">
      <button class:on={tools.mirror} title="Mirror across X (X)" onclick={() => tools.toggleMirror()}>mirror x<kbd>X</kbd></button>
      {#if tools.tool === 'brush'}
        <button class:on={tools.brush.erase} title="Erase (E, or hold ⌥)" onclick={() => (tools.brush.erase = !tools.brush.erase)}>erase<kbd>E</kbd></button>
      {:else}
        <button class:on={tools.snap} title="Snap to a grid on the surface (G)" onclick={() => (tools.snap = !tools.snap)}>snap<kbd>G</kbd></button>
        <button title="Rotate 15° (R, ⇧R back)" onclick={() => tools.rotate(15)}>{tools.stamp.rot}°<kbd>R</kbd></button>
      {/if}
    </div>
  {/if}
</div>

<style>
  .bar {
    position: absolute;
    top: 10px;
    left: 10px;
    display: flex;
    gap: 6px;
    pointer-events: none;
  }

  .group {
    display: flex;
    gap: 1px;
    padding: 2px;
    border: 1px solid var(--line);
    border-radius: 8px;
    background: var(--panel);
    pointer-events: auto;
  }

  button {
    display: flex;
    align-items: center;
    gap: 6px;
    height: 24px;
    padding: 0 8px;
    border: 0;
    border-radius: 6px;
    background: none;
    font: 12px var(--mono);
    color: var(--dim);
  }
  button:hover {
    color: var(--strong);
  }
  button.on {
    background: var(--active);
    color: var(--accent);
  }
  kbd {
    border: 0;
    padding: 0;
  }
</style>
