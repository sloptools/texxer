<script lang="ts">
  import Icon from '../Icon.svelte'
  import { store } from '../store.svelte'
  import Docs from './Docs.svelte'

  const groups: [string, [string, string][]][] = [
    [
      'trackpad',
      [
        ['two fingers', 'orbit'],
        ['⇧ two fingers', 'pan'],
        ['pinch', 'zoom to cursor'],
        ['⌥ two fingers', 'turn key light'],
        ['drag / ⇧ drag', 'orbit / pan'],
        ['⌘ click', 'add part to layer'],
        ['⇧⌘ click', 'add island (surface up to hard edges)'],
      ],
    ],
    [
      'view',
      [
        ['1 3 7', 'front, side, top (⌃ opposite)'],
        ['0', 'reset view'],
        ['F', 'focus point under cursor'],
        ['L U I M', 'lit, unlit, in game, mask'],
        ['H', 'headlight on/off'],
        ['⇧B ⇧G', 'bloom, grid'],
        ['P', 'game resolution: full, 480, 360, 240'],
        ['T', 'flashlight at the cursor'],
      ],
    ],
    [
      '2d',
      [
        ['tab', 'flat texture and UVs, back to 3D'],
        ['1 – 7', 'texture, mask, glow, ao, edges, cavity, parts'],
        ['W', 'UV layout: cuts, unused space, overlaps in red'],
        ['two fingers / drag', 'pan'],
        ['pinch', 'zoom to cursor'],
        ['0 F', 'fit'],
      ],
    ],
    [
      'tools',
      [
        ['B S esc', 'brush, stamp, view'],
        ['[ ]', 'size'],
        ['⌥ / E', 'erase (brush)'],
        ['⇧ click', 'straight line (brush)'],
        ['X', 'mirror across x'],
        ['G', 'snap stamps to a surface grid'],
        ['⇧ click', 'row of stamps from the picked one'],
        ['R ⇧R', 'rotate stamp 15°'],
        ['arrows', 'nudge picked stamp (⇧ more)'],
        ['D ⌫', 'duplicate, delete picked stamp'],
      ],
    ],
    [
      'layers',
      [
        ['N', 'new layer'],
        ['D', 'duplicate'],
        ['⌫', 'delete'],
        ['↑ ↓', 'select'],
        ['⌥↑ ⌥↓', 'move'],
        ['V', 'show / hide'],
        ['⌘G ⇧⌘G', 'into a new folder, dissolve folder'],
        ['C', 'next variant (color, visibility, opacity per layer)'],
        ['drag', 'reorder, drop on a folder to put it in'],
      ],
    ],
    [
      'file',
      [
        ['⌘O ⌘N', 'open a project or model, new project'],
        ['⌘P', 'switch model'],
        ['PgUp PgDn', 'previous, next model'],
        ['⌘E', 'export the model into its files'],
        ['⇧⌘E', 'export every changed model'],
        ['⌘Z ⇧⌘Z', 'undo, redo'],
        ['? F1', 'shortcuts, docs'],
      ],
    ],
  ]
</script>

<div class="backdrop" onclick={() => (store.help = false)} role="presentation">
  <div class="sheet" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true" aria-label="Help" tabindex="-1" onkeydown={() => {}}>
    <header>
      <span class="tabs">
        <button class:on={store.helpTab === 'docs'} onclick={() => (store.helpTab = 'docs')}>docs</button>
        <button class:on={store.helpTab === 'shortcuts'} onclick={() => (store.helpTab = 'shortcuts')}>shortcuts</button>
      </span>
      <button class="close" onclick={() => (store.help = false)} aria-label="Close"><Icon name="x" size={11} /></button>
    </header>
    {#if store.helpTab === 'docs'}
      <div class="body"><Docs /></div>
    {:else}
      <div class="grid">
        {#each groups as [title, rows]}
          <section>
            <h3>{title}</h3>
            {#each rows as [key, what]}
              <div class="row"><kbd>{key}</kbd><span>{what}</span></div>
            {/each}
          </section>
        {/each}
      </div>
    {/if}
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 10;
    display: grid;
    place-items: center;
    background: rgb(0 0 0 / 0.35);
    animation: fade 0.12s ease-out;
  }

  .sheet {
    display: flex;
    flex-direction: column;
    width: min(880px, calc(100vw - 32px));
    height: min(680px, calc(100vh - 64px));
    border: 1px solid var(--line);
    border-radius: 10px;
    background: var(--panel);
    box-shadow: 0 16px 48px rgb(0 0 0 / 0.3);
    outline: none;
  }

  header {
    flex: none;
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 48px;
    padding: 0 12px 0 20px;
    border-bottom: 1px solid var(--line);
  }
  .tabs {
    display: flex;
    gap: 2px;
  }
  .tabs button {
    height: 24px;
    padding: 0 8px;
    border: 0;
    border-radius: 6px;
    background: none;
    font: 500 11px var(--mono);
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: var(--faint);
  }
  .tabs button:hover {
    color: var(--strong);
  }
  .tabs button.on {
    background: var(--active);
    color: var(--fg);
  }
  .body {
    flex: 1;
    min-height: 0;
  }
  .close {
    display: grid;
    place-items: center;
    width: 24px;
    height: 24px;
    padding: 0;
    border: 0;
    border-radius: 4px;
    background: none;
    color: var(--dim);
  }
  .close:hover {
    background: var(--hover);
    color: var(--strong);
  }

  .grid {
    overflow-y: auto;
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 4px 24px;
    padding: 14px 20px 18px;
  }
  h3 {
    margin: 6px 0 6px;
    font: 500 11px var(--mono);
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: var(--dim);
  }
  .row {
    display: flex;
    align-items: center;
    gap: 10px;
    min-height: 24px;
    color: var(--fg);
  }
  .row kbd {
    min-width: 92px;
    border: 0;
    padding: 0;
    font-size: 11px;
    color: var(--dim);
  }

  @keyframes fade {
    from {
      opacity: 0;
    }
  }
</style>
