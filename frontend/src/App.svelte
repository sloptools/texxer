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
  import { handleKey } from './lib/keys'
  import Viewport from './lib/Viewport.svelte'
  import VariantBar from './lib/ui/VariantBar.svelte'
  import ModelList from './lib/ui/ModelList.svelte'
  import QuickOpen from './lib/ui/QuickOpen.svelte'
  import MaterialMenu from './lib/ui/MaterialMenu.svelte'
  import { snippets } from './lib/snippets.svelte'
  import { bridge } from './lib/bridge.svelte'
  import PhotoList from './lib/ui/PhotoList.svelte'
  import PhotoView from './lib/ui/PhotoView.svelte'
  import CutPanel from './lib/ui/CutPanel.svelte'
  import SnippetStrip from './lib/ui/SnippetStrip.svelte'
  import { addModels, addTile, createProject, exportModel, openProject, pickProject, projectName, recent, refreshHashes } from './lib/projects.svelte'

  let materials = $state(false)
  type Side = 'models' | 'layers' | 'settings'
  const sides: Side[] = ['models', 'layers', 'settings']

  // Painting the model, or cutting snippets from photos.
  type Mode = 'paint' | 'photos'
  let mode = $state<Mode>(localStorage.getItem('mode') === 'photos' ? 'photos' : 'paint')
  $effect(() => localStorage.setItem('mode', mode))
  let side = $state<Side>(sides.find((s) => s === localStorage.getItem('side')) ?? 'layers')
  $effect(() => localStorage.setItem('side', side))

  onMount(() => {
    store.init()
    bridge.start()
    OnFileDrop((_x, _y, paths) => {
      const project = paths.find((p) => p.toLowerCase().endsWith('.texxer'))
      const glbs = paths.filter((p) => p.toLowerCase().endsWith('.glb'))
      const photos = paths.filter((p) => /\.(png|jpe?g|webp)$/i.test(p))
      if (project) openProject(project)
      else if (glbs.length) addModels(glbs)
      else if (photos.length && store.path) {
        mode = 'photos'
        snippets.addPhotos(photos)
      }
    }, false)
    requestAnimationFrame(() => WindowShow())
  })

  // Each project has its own snippets.
  $effect(() => {
    if (store.path) snippets.load('')
  })

  // A picked stamp belongs to the layer it was picked on.
  $effect(() => {
    store.selected
    store.model
    tools.picked = null
  })

  function onkeydown(e: KeyboardEvent) {
    handleKey(e, () => {
      side = 'layers'
      materials = true
    })
  }
</script>

<svelte:window {onkeydown} onfocus={refreshHashes} />

<div class="app">
  <header class="top">
    <span class="title">texxer{#if store.path}<span class="project">{projectName()}</span>{/if}</span>
    {#if store.workspace}
      <div class="modes" role="tablist">
        <button role="tab" aria-selected={mode === 'paint'} class:on={mode === 'paint'} onclick={() => (mode = 'paint')}>paint</button>
        <button role="tab" aria-selected={mode === 'photos'} class:on={mode === 'photos'} onclick={() => (mode = 'photos')}>photos</button>
      </div>
    {/if}
    <span class="actions">
      <button title="New project (⌘N)" onclick={createProject}><Icon name="plus" size={11} />new</button>
      <button title="Open a project or a model (⌘O)" onclick={pickProject}><Icon name="dir" size={11} />open</button>
      <button title="Export the model into its files, PNGs beside them (⌘E; ⇧⌘E all changed)" disabled={!store.model} onclick={exportModel}><Icon name="export" size={11} />export</button>
    </span>
  </header>

  <aside class="left">
    {#if store.workspace && mode === 'photos'}
      <div class="scroll">
        <PhotoList />
      </div>
    {:else if store.workspace}
      <div class="tabs">
        {#each sides as s}
          <button class="caps" class:on={side === s} onclick={() => (side = s)}>{s}</button>
        {/each}
        {#if side === 'models'}
          <span class="icons">
            <button class="icon" title="Add models (each file its own model)" onclick={() => addModels()}><Icon name="plus" size={11} /></button>
            <button class="text" title="Add a tile for a tileable texture, like walls and floors" onclick={addTile}>tile</button>
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
          <div class="adds">
            <button title="New layer (N)" onclick={() => store.addLayer()}><Icon name="plus" size={11} />layer</button>
            <button title="Put the layer in a new folder (⌘G)" onclick={() => store.groupLayer()}><Icon name="dir" size={11} />folder</button>
            <button title="Add a material or dirt (⇧M)" onclick={() => (materials = !materials)}><Icon name="ball" size={11} />material</button>
          </div>
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
        {#if tools.tool === 'stamp'}
          <SnippetStrip />
        {/if}
      {/if}
    {/if}
    {#if mode === 'photos'}
      {#if snippets.current}
        <PhotoView photo={snippets.current} />
      {:else}
        <div class="nophoto">drop photos here to cut snippets from them</div>
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
    {#if mode === 'photos'}
      <CutPanel />
    {:else if store.model}
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
    position: relative;
    grid-column: 1 / -1;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 10px 0 100px;
    border-bottom: 1px solid var(--line);
    background: var(--panel);
    --wails-draggable: drag;
  }
  .modes {
    position: absolute;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    gap: 1px;
    padding: 2px;
    border: 1px solid var(--line);
    border-radius: 7px;
    background: var(--bg);
    --wails-draggable: no-drag;
  }
  .modes button {
    height: 22px;
    padding: 0 12px;
    border: 0;
    border-radius: 5px;
    background: none;
    font: 11px var(--mono);
    color: var(--dim);
  }
  .modes button.on {
    background: var(--active);
    color: var(--strong);
  }
  .nophoto {
    position: absolute;
    inset: 0;
    z-index: 2;
    display: grid;
    place-items: center;
    background: var(--bg);
    font: 12px var(--mono);
    color: var(--faint);
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
  .adds {
    display: flex;
    gap: 4px;
    padding: 2px 12px 8px;
  }
  .adds button {
    display: flex;
    align-items: center;
    gap: 5px;
    height: 22px;
    padding: 0 7px;
    border: 1px solid var(--line);
    border-radius: 5px;
    background: none;
    font: 11px var(--mono);
    color: var(--dim);
  }
  .adds button:hover {
    color: var(--strong);
    background: var(--hover);
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
  .icon:hover,
  .text:hover {
    background: var(--hover);
    color: var(--strong);
  }
  .text {
    height: 22px;
    padding: 0 6px;
    border: 0;
    border-radius: 4px;
    background: none;
    font: 11px var(--mono);
    color: var(--dim);
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
