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
  import { nudgeSelection, stepSelection } from './lib/layerRows'
  import type { GizmoMode } from './lib/gl/volume'
  import Viewport from './lib/Viewport.svelte'
  import VariantBar from './lib/ui/VariantBar.svelte'
  import ModelList from './lib/ui/ModelList.svelte'
  import QuickOpen from './lib/ui/QuickOpen.svelte'
  import MaterialMenu from './lib/ui/MaterialMenu.svelte'
  import { addModels, createProject, cycleModel, exportChanged, exportModel, openProject, pickProject, projectName, recent, refreshHashes } from './lib/projects.svelte'

  let headlight = 0.6
  let materials = $state(false)
  type Side = 'models' | 'layers' | 'settings'
  const sides: Side[] = ['models', 'layers', 'settings']
  let side = $state<Side>(sides.find((s) => s === localStorage.getItem('side')) ?? 'layers')
  $effect(() => localStorage.setItem('side', side))

  onMount(() => {
    store.init()
    OnFileDrop((_x, _y, paths) => {
      const project = paths.find((p) => p.toLowerCase().endsWith('.texxer'))
      const glbs = paths.filter((p) => p.toLowerCase().endsWith('.glb'))
      if (project) openProject(project)
      else if (glbs.length) addModels(glbs)
    }, false)
    requestAnimationFrame(() => WindowShow())
  })

  // A picked stamp belongs to the layer it was picked on.
  $effect(() => {
    store.selected
    store.model
    tools.picked = null
  })

  // W E R pick the gizmo's move, turn and size, as in Godot.
  const gizmoKeys: Record<string, GizmoMode> = { w: 'translate', e: 'rotate', r: 'scale' }

  // Keys for the brush, stamp and volume tools; true when handled.
  function toolKey(e: KeyboardEvent): boolean {
    if (e.key === 'q') {
      tools.set('volume')
      return true
    }
    const volume = tools.tool === 'volume' ? tools.volume : undefined
    if (volume && gizmoKeys[e.key]) {
      tools.gizmo = gizmoKeys[e.key]
      return true
    }
    if (volume && e.key === 'x') {
      volume.volume.mirror = !volume.volume.mirror
      return true
    }
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
        if (store.pickingZone) store.pickingZone = false
        else if (tools.picked) tools.picked = null
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
    if (store.quickOpen) return
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
          pickProject()
          break
        case 'n':
          createProject()
          break
        case 'p':
          if (store.workspace) store.quickOpen = true
          break
        case 'e':
          if (e.shiftKey) exportChanged()
          else exportModel()
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
          store.addLayer()
          break
        case 'd':
          store.duplicateLayer()
          break
        case 'Backspace':
        case 'Delete':
          store.deleteLayer()
          break
        case 'ArrowUp':
          if (e.altKey) nudgeSelection(true)
          else stepSelection(1)
          break
        case 'ArrowDown':
          if (e.altKey) nudgeSelection(false)
          else stepSelection(-1)
          break
        case 'v':
          if (l) {
            const look = store.look(l)
            look.visible = !look.visible
          }
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
        case 'M':
          side = 'layers'
          materials = true
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
        case 'c':
          store.cycleVariant()
          break
        case 'Tab':
          store.flat = !store.flat
          break
        case 'PageUp':
          cycleModel(-1)
          break
        case 'PageDown':
          cycleModel(1)
          break
        case 'F1':
          store.openDocs(store.helpPage)
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

<svelte:window {onkeydown} onfocus={refreshHashes} />

<div class="app">
  <header class="top">
    <span class="title">texxer{#if store.path}<span class="project">{projectName()}</span>{/if}</span>
    <span class="actions">
      <button title="New project (⌘N)" onclick={createProject}><Icon name="plus" size={11} />new</button>
      <button title="Open a project or a model (⌘O)" onclick={pickProject}><Icon name="dir" size={11} />open</button>
      <button title="Export the model into its files, PNGs beside them (⌘E; ⇧⌘E all changed)" disabled={!store.model} onclick={exportModel}><Icon name="export" size={11} />export</button>
    </span>
  </header>

  <aside class="left">
    {#if store.workspace}
      <div class="tabs">
        {#each sides as s}
          <button class="caps" class:on={side === s} onclick={() => (side = s)}>{s}</button>
        {/each}
        {#if side === 'layers' && store.model}
          <span class="icons">
            <button class="icon" title="Add a material: a folder of layers (⇧M)" onclick={() => (materials = !materials)}><Icon name="ball" size={11} /></button>
            <button class="icon" title="Put the layer in a new folder (⌘G)" onclick={() => store.groupLayer()}><Icon name="dir" size={11} /></button>
            <button class="icon" title="New layer (N)" onclick={() => store.addLayer()}><Icon name="plus" size={11} /></button>
          </span>
        {:else if side === 'models'}
          <span class="icons">
            <button class="icon" title="Add models (each file its own model)" onclick={() => addModels()}><Icon name="plus" size={11} /></button>
          </span>
        {/if}
      </div>
      {#if materials && side === 'layers'}
        <MaterialMenu onclose={() => (materials = false)} />
      {/if}
      <div class="scroll">
        {#if side === 'models'}
          <ModelList />
        {:else if !store.model}
          <p class="note">pick a model first</p>
        {:else if side === 'layers'}
          <VariantBar />
          <LayerList />
        {:else}
          <Finish />
        {/if}
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
    {#if !store.workspace}
      <div class="empty">
        <p>Open a project, or a model to paint it right away. Drop files here or</p>
        <div class="buttons">
          <button class="primary" onclick={pickProject}>open <kbd>⌘O</kbd></button>
          <button class="primary" onclick={createProject}>new project <kbd>⌘N</kbd></button>
        </div>
        {#if recent.list.length}
          <div class="recent">
            <div class="caps">recent</div>
            {#each recent.list as path}
              <button onclick={() => openProject(path)} title={path}>{path.split('/').pop()}<span>{path.split('/').slice(-3, -1).join('/')}</span></button>
            {/each}
          </div>
        {/if}
      </div>
    {:else if !store.workspace.models.length}
      <div class="empty">
        <p>Add the models this project paints. Drop GLBs here or</p>
        <button class="primary" onclick={() => addModels()}>add models</button>
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
{#if store.quickOpen}
  <QuickOpen />
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
  .project {
    margin-left: 10px;
    font: 12px var(--mono);
    color: var(--dim);
  }
  .note {
    margin: 4px 16px;
    font-size: 12px;
    color: var(--faint);
  }
  .buttons {
    display: flex;
    gap: 8px;
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
    position: relative;
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
    padding-bottom: 12px;
    overflow-y: auto;
  }

  .tabs {
    display: flex;
    align-items: center;
    gap: 1px;
    height: 38px;
    padding: 0 10px 0 6px;
  }
  .tabs .caps {
    height: 24px;
    padding: 0 6px;
    border: 0;
    border-radius: 6px;
    background: none;
    color: var(--faint);
  }
  .tabs .caps:hover {
    color: var(--strong);
  }
  .tabs .caps.on {
    background: var(--active);
    color: var(--fg);
  }
  .icons {
    display: flex;
    gap: 0;
    margin-left: auto;
  }
  .icon {
    display: grid;
    place-items: center;
    width: 22px;
    height: 22px;
    padding: 0;
    border: 0;
    border-radius: 4px;
    background: none;
    color: var(--dim);
  }
  .icon:hover {
    background: var(--hover);
    color: var(--strong);
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
