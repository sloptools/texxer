<script lang="ts">
  import { onMount } from 'svelte'
  import { OnFileDrop, WindowShow } from '../wailsjs/runtime/runtime'
  import Icon from './lib/Icon.svelte'
  import { store } from './lib/store.svelte'
  import Finish from './lib/ui/Finish.svelte'
  import FlatBar from './lib/ui/FlatBar.svelte'
  import Help from './lib/ui/Help.svelte'
  import LayerList from './lib/ui/LayerList.svelte'
  import LayerPanel from './lib/ui/LayerPanel.svelte'
  import StatusBar from './lib/ui/StatusBar.svelte'
  import Toolbar from './lib/ui/Toolbar.svelte'
  import ToolPanel from './lib/ui/ToolPanel.svelte'
  import { tools } from './lib/tools.svelte'
  import Viewport from './lib/Viewport.svelte'

  let headlight = 0.6

  onMount(() => {
    store.init()
    OnFileDrop((_x, _y, paths) => {
      const glb = paths.find((p) => p.toLowerCase().endsWith('.glb'))
      if (glb) store.open(glb)
    }, false)
    requestAnimationFrame(() => WindowShow())
  })

  // A picked stamp belongs to the layer it was picked on.
  $effect(() => {
    store.selected
    store.model
    tools.picked = null
  })

  // Keys for the brush and stamp tools; true when handled.
  function toolKey(e: KeyboardEvent): boolean {
    const picked = !!tools.pickedStamp
    const step = e.shiftKey ? 0.5 : 0.1
    switch (e.key) {
      case 'b':
        tools.set('brush')
        return true
      case 's':
        tools.set('stamp')
        return true
      case 'Escape':
        if (tools.picked) tools.picked = null
        else if (tools.tool !== 'nav') tools.set(tools.tool)
        return true
      case '[':
        tools.resize(1 / 1.2)
        return true
      case ']':
        tools.resize(1.2)
        return true
      case 'x':
        tools.toggleMirror()
        return true
      case 'e':
        tools.brush.erase = !tools.brush.erase
        return true
      case 'g':
        tools.snap = !tools.snap
        return true
      case 'r':
      case 'R':
        tools.rotate(e.shiftKey ? -15 : 15)
        return true
    }
    if (!picked) return false

    switch (e.key) {
      case 'ArrowLeft':
        tools.nudge(-step, 0)
        return true
      case 'ArrowRight':
        tools.nudge(step, 0)
        return true
      case 'ArrowUp':
        tools.nudge(0, step)
        return true
      case 'ArrowDown':
        tools.nudge(0, -step)
        return true
      case 'Backspace':
      case 'Delete':
        tools.remove()
        return true
      case 'd':
        tools.duplicate()
        return true
    }
    return false
  }

  function onkeydown(e: KeyboardEvent) {
    if ((e.target as HTMLElement).closest('input, textarea')) return
    if (store.help) {
      if (e.key === 'Escape') store.help = false
      return
    }
    if (!e.metaKey && !e.ctrlKey && toolKey(e)) {
      e.preventDefault()
      return
    }

    const cmd = e.metaKey || e.ctrlKey
    const l = store.layer
    const light = store.project.light
    let handled = true

    if (cmd) {
      switch (e.key.toLowerCase()) {
        case 'o':
          store.pickModel()
          break
        case 'e':
          store.export(e.shiftKey)
          break
        case 'z':
          if (e.shiftKey) store.redo()
          else store.undo()
          break
        case 'g':
          if (e.shiftKey) store.ungroup()
          else store.groupLayer()
          break
        default:
          handled = false
      }
    } else {
      switch (e.key) {
        case 'n':
          store.addLayer(tools.layerMask)
          break
        case 'd':
          store.duplicateLayer()
          break
        case 'Backspace':
        case 'Delete':
          store.deleteLayer()
          break
        case 'ArrowUp':
          if (e.altKey) store.moveLayer(store.selected, store.selected + 1)
          else store.select(1)
          break
        case 'ArrowDown':
          if (e.altKey) store.moveLayer(store.selected, store.selected - 1)
          else store.select(-1)
          break
        case 'v':
          if (l) l.visible = !l.visible
          break
        case 'l':
          store.view = 'lit'
          break
        case 'u':
          store.view = 'unlit'
          break
        case 'i':
          store.view = 'game'
          break
        case 'm':
          store.view = store.view === 'mask' ? 'lit' : 'mask'
          break
        case 'h':
          if (light.headlight > 0) {
            headlight = light.headlight
            light.headlight = 0
          } else light.headlight = headlight
          break
        case 'B':
          store.bloom = !store.bloom
          break
        case 'G':
          store.grid = !store.grid
          break
        case 'p':
          store.cycleLines()
          break
        case 't':
          store.torch = !store.torch
          break
        case 'Tab':
          store.flat = !store.flat
          break
        case '?':
          store.help = true
          break
        default:
          handled = false
      }
    }
    if (handled) e.preventDefault()
  }
</script>

<svelte:window {onkeydown} />

<div class="app">
  <header class="top">
    <span class="title">texxer</span>
    <span class="actions">
      <button title="Open model (⌘O)" onclick={() => store.pickModel()}><Icon name="dir" size={11} />open</button>
      <button title="Export GLB and PNG (⌘E, ⇧⌘E asks where)" disabled={!store.model} onclick={() => store.export(false)}><Icon name="export" size={11} />export</button>
    </span>
  </header>

  <aside class="left">
    {#if store.model}
      <div class="scroll">
        <LayerList />
        <div class="spacer"></div>
        <Finish />
      </div>
    {/if}
  </aside>

  <main>
    <Viewport />
    {#if store.model}
      {#if store.flat}
        <FlatBar />
      {:else}
        <Toolbar />
      {/if}
    {/if}
    {#if !store.model}
      <div class="empty">
        <p>Open a model to paint it. Drop a .glb here or</p>
        <button class="primary" onclick={() => store.pickModel()}>open model <kbd>⌘O</kbd></button>
        {#if store.recent.length}
          <div class="recent">
            <div class="caps">recent</div>
            {#each store.recent as path}
              <button onclick={() => store.open(path)} title={path}>{path.split('/').pop()}<span>{path.split('/').slice(-3, -1).join('/')}</span></button>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </main>

  <aside class="right">
    {#if store.model}
      <ToolPanel />
      <LayerPanel />
    {/if}
  </aside>

  <div class="bottom">
    <StatusBar />
  </div>
</div>

{#if store.help}
  <Help />
{/if}

<style>
  .app {
    display: grid;
    grid-template-columns: 248px minmax(0, 1fr) 300px;
    grid-template-rows: 44px minmax(0, 1fr) 30px;
    height: 100%;
  }

  .top {
    grid-column: 1 / -1;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 10px 0 100px;
    border-bottom: 1px solid var(--line);
    background: var(--panel);
    --wails-draggable: drag;
  }
  .title {
    font-size: 13px;
    font-weight: var(--title-weight);
    color: var(--strong);
  }
  .actions {
    display: flex;
    gap: 2px;
    --wails-draggable: no-drag;
  }
  .actions button {
    display: flex;
    align-items: center;
    gap: 6px;
    height: 26px;
    padding: 0 9px;
    border: 0;
    border-radius: 6px;
    background: none;
    font: 12px var(--mono);
    color: var(--dim);
  }
  .actions button:hover:not(:disabled) {
    background: var(--hover);
    color: var(--strong);
  }
  .actions button:disabled {
    color: var(--faint);
  }

  aside {
    display: flex;
    flex-direction: column;
    min-height: 0;
    background: var(--panel);
    overflow: hidden;
  }
  .left {
    border-right: 1px solid var(--line);
  }
  .left .scroll {
    flex: 1;
    min-height: 0;
    padding-top: 8px;
    overflow-y: auto;
  }
  .spacer {
    height: 16px;
  }
  .right {
    border-left: 1px solid var(--line);
    overflow-y: auto;
  }

  main {
    position: relative;
    min-width: 0;
  }

  .empty {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 12px;
    color: var(--dim);
  }
  .empty p {
    margin: 0;
  }
  .primary {
    display: flex;
    align-items: center;
    gap: 10px;
    height: 32px;
    padding: 0 14px;
    border: 1px solid var(--line);
    border-radius: 7px;
    background: var(--raised);
    font: 12px var(--mono);
    color: var(--strong);
  }
  .primary:hover {
    border-color: var(--dim);
  }

  .recent {
    display: flex;
    flex-direction: column;
    width: 320px;
    margin-top: 18px;
    gap: 2px;
  }
  .recent .caps {
    padding: 0 10px 4px;
  }
  .recent button {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 12px;
    height: 28px;
    padding: 0 10px;
    border: 0;
    border-radius: 6px;
    background: none;
    font: inherit;
    color: var(--fg);
    text-align: left;
  }
  .recent button:hover {
    background: var(--hover);
    color: var(--strong);
  }
  .recent span {
    overflow: hidden;
    font: 11px var(--mono);
    color: var(--faint);
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  .bottom {
    grid-column: 1 / -1;
    border-top: 1px solid var(--line);
    background: var(--panel);
  }
</style>
