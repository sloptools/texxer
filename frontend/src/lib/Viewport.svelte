<script lang="ts">
  import ViewGizmo from './ui/ViewGizmo.svelte'
  import { onMount, untrack } from 'svelte'
  import * as api from '../../wailsjs/go/main/App'
  import { Compositor, loadBake, tilePeriod } from './gl/compositor'
  import { PaintStore } from './gl/paint'
  import { Stage, type Hit } from './gl/stage'
  import { withVariant } from './project'
  import { stateOf } from './projects.svelte'
  import { corners, handles } from './stamps'
  import { store } from './store.svelte'
  import { ToolInput } from './toolInput'
  import { bridge } from './bridge.svelte'
  import { tools } from './tools.svelte'
  import type { FlatMap, Project } from './types'
  import type { RGB } from './gl/mediancut'

  let container: HTMLDivElement
  let stage = $state<Stage>()
  let compositor = $state<Compositor>()
  let dirty = true
  let hoverPoint: { x: number; y: number } | undefined
  // When the selected layer last changed, to flash where it shows.
  let flashed = -Infinity
  const size = $derived(store.project.size)
  const factor = $derived(store.project.finish.supersample)
  let paletteTimer = 0

  onMount(() => {
    const s = new Stage(container)
    stage = s
    if (import.meta.env.DEV) Object.assign(window, { stage: s })
    const input = new ToolInput(s, () => compositor, () => (dirty = true))
    tools.surface = (p, n, reach) => s.surface(p, n, reach)
    s.volume.onChange = (v) => {
      const m = tools.volume
      if (m) Object.assign(m.volume, v)
    }
    s.onError = (m) => store.say(m, true)

    s.controls.onLight = (d) => (store.project.light.angle = (store.project.light.angle - d * 0.4 + 360) % 360)
    s.controls.onClick = (e) => {
      if (tools.tool !== 'nav') return
      const hit = s.hit(e.clientX, e.clientY)
      if (e.altKey) {
        pickLayer(hit)
        return
      }
      if (!hit || hit.part < 0) return
      if (store.selectedGroup && store.pickingZone) store.toggleZone(e.metaKey ? 'parts' : 'islands', e.metaKey ? hit.part : hit.island)
      else if (store.picksIslands || (e.metaKey && e.shiftKey)) store.toggleIsland(hit.island)
      else if (e.metaKey || store.layer?.masks.some((m) => m.kind === 'parts')) store.togglePart(hit.part)
    }

    const onMove = (e: PointerEvent) => {
      hoverPoint = e.buttons === 0 ? { x: e.clientX, y: e.clientY } : undefined
    }
    const onLeave = () => {
      hoverPoint = undefined
      store.hoverPart = -1
    }
    s.flat.onHover = (texel, part) => {
      store.hoverTexel = texel
      store.hoverPart = part
    }
    s.flat.onClick = (e, part) => {
      if (part >= 0 && tools.tool === 'nav' && (e.metaKey || store.layer?.masks.some((m) => m.kind === 'parts'))) store.togglePart(part)
    }

    // Ahead of the tools and the camera, so sampling works with any tool.
    const onSample = (e: PointerEvent) => {
      if (!store.sampling || e.button !== 0) return
      e.stopImmediatePropagation()
      e.preventDefault()
      sampleColor(s.hit(e.clientX, e.clientY))
    }

    s.renderer.domElement.addEventListener('pointerdown', onSample, { capture: true })
    s.renderer.domElement.addEventListener('pointermove', onMove)
    s.renderer.domElement.addEventListener('pointerleave', onLeave)

    let frame = 0
    const loop = () => {
      frame = requestAnimationFrame(loop)
      if (compositor && dirty) {
        const project = withVariant($state.snapshot(store.project), store.variant)
        compositor.render(project, store.hoverLayer >= 0 ? store.hoverLayer : store.selected, tools.ghost && $state.snapshot(tools.ghost))
        if (project.finish.palette === 'auto') schedulePalette(compositor)
        dirty = false
        bridge.changed()
      }
      if (compositor) {
        const t = (performance.now() - flashed) / 500
        const tint = store.hoverLayer >= 0 ? 0.6 : t < 1 ? 0.6 * (1 - t * t) : 0
        s.setMaskTint(tint > 0 && store.view !== 'mask' ? compositor.mask.texture : null, tint)
        const glow = compositor.hasEmission && store.view !== 'mask' ? compositor.emission!.texture : null
        s.setEmission(glow, store.project.finish.glow)
      }
      if (hoverPoint && !store.flat) {
        const hit = s.hit(hoverPoint.x, hoverPoint.y)
        store.hoverPart = hit?.part ?? -1
        store.hoverIsland = hit?.island ?? -1
        s.torch.aimAt(hit?.point)
      }
      s.render()
    }
    loop()
    s.setBackground(getComputedStyle(container).backgroundColor)

    return () => {
      cancelAnimationFrame(frame)
      s.renderer.domElement.removeEventListener('pointerdown', onSample, { capture: true })
      input.dispose()
      compositor?.dispose()
      store.paint?.dispose()
      store.paint = null
      s.dispose()
      stage = undefined
    }
  })

  // The color the layers show where the model was clicked, into the selected layer.
  function sampleColor(hit: Hit | undefined) {
    store.sampling = false
    const c = compositor
    const l = store.layer
    if (!c || !l || !hit?.uv) return
    const size = c.mask.width
    const color = c.colorAt([Math.min(size - 1, Math.floor(hit.uv.x * size)), Math.min(size - 1, Math.floor(hit.uv.y * size))])
    // Layers that paint colors take it as the brush color.
    if (l.paintColors && tools.tool === 'brush') {
      tools.brush.color = color
      store.say(`brush takes ${color}`)
      return
    }
    store.look(l).color = color
    store.say(`${l.name} takes ${color}`)
  }

  // ⌥click selects the topmost layer that shows where the model was clicked.
  function pickLayer(hit: Hit | undefined) {
    const c = compositor
    if (!c || !hit?.uv) return
    const size = c.mask.width
    const texel: [number, number] = [Math.min(size - 1, Math.floor(hit.uv.x * size)), Math.min(size - 1, Math.floor(hit.uv.y * size))]
    const i = c.layerAt(withVariant($state.snapshot(store.project), store.variant), texel)
    c.renderMask(withVariant($state.snapshot(store.project), store.variant), i >= 0 ? i : store.selected)
    if (i < 0) return
    store.selected = i
    store.selectedGroup = ''
    const l = store.project.layers[i]
    const g = store.group(l.group)
    if (g) g.open = true
    store.say('picked ' + l.name)
  }

  // The auto palette follows the texture once edits pause, so dragging stays fast.
  function schedulePalette(c: Compositor) {
    clearTimeout(paletteTimer)
    paletteTimer = setTimeout(() => {
      if (c !== compositor) return
      const project = withVariant($state.snapshot(store.project), store.variant)
      c.pickPalette(project.finish.count, painted(project))
      c.finish(project)
    }, 150)
  }

  // Colors painted on shown layers, which the auto palette keeps.
  function painted(project: Project): RGB[] {
    const hex = project.layers.filter((l) => l.paintColors && l.visible).flatMap((l) => l.brushColors ?? [])
    return [...new Set(hex)].map((h) => [1, 3, 5].map((i) => parseInt(h.slice(i, i + 2), 16)) as RGB)
  }

  // Bake settings reach the bake once the sliders rest, as each bake takes a moment.
  let bakeSettings = $state(store.project.bake)
  $effect(() => {
    const b = $state.snapshot(store.project.bake)
    const t = setTimeout(() => {
      if (JSON.stringify(b) !== JSON.stringify(bakeSettings)) bakeSettings = b
    }, 400)
    return () => clearTimeout(t)
  })

  // Layers that take the model's own colors need them baked too.
  const wantsSource = $derived(store.project.layers.some((l) => l.source))

  // Model and bake, again when the texture size or bake settings change.
  $effect(() => {
    const s = stage
    const model = store.model
    const layout = store.project.unwrap
    const settings = bakeSettings
    const source = wantsSource
    const texels = size * factor
    const ss = factor
    if (!s || !model) return

    let cancelled = false
    store.baking = true
    store.ready = false
    const project = store.path
    const key = encodeURIComponent(model.path)
    // The hash keeps the webview from serving a file Blender has since rewritten.
    const files = model.offsets.map((offset, i) => ({ url: `/model?path=${key}&unwrap=${layout}&file=${i}&h=${model.hash}`, images: `/linked?path=${key}&file=${i}&uri=`, offset }))
    // The colors come from kept copies of the files, as export overwrites them; a
    // new copy when the files changed outside texxer.
    const doc = store.project
    const refresh = untrack(() => !doc.exported || stateOf(doc) === 'changed')
    const originals = source ? api.KeepOriginals(project, doc.id, model.path, refresh) : Promise.resolve('')
    // Strokes in flight are saved first, so the new paint store loads them.
    Promise.all([s.load(files, model.tile), originals.then((kept) => loadBake(model.path, texels, layout, settings, kept)), store.paint?.flush()])
      .then(([, bake]) => {
        if (cancelled) return
        compositor?.dispose()
        store.paint?.dispose()
        const paint = new PaintStore(
          s.renderer,
          bake,
          (id) => `/paint?project=${encodeURIComponent(project)}&id=${id}`,
          (id, png) => api.SavePaint(project, id, png).catch((e) => store.say(String(e))),
          tilePeriod(model),
        )
        paint.onChange = () => (dirty = true)
        store.paint = paint
        const c = new Compositor(s.renderer, bake, model, paint, ss)
        c.onLoad = () => (dirty = true)
        s.posData = bake.posData
        s.nrmData = bake.nrmData
        s.posSize = bake.size
        compositor = c
        store.flattened = (indices) => c.flatten(withVariant($state.snapshot(store.project), store.variant), indices)
        store.snapshot = (variant) => {
          const project = withVariant($state.snapshot(store.project), variant)
          c.render(project, store.selected)
          if (project.finish.palette === 'auto') {
            c.pickPalette(project.finish.count, painted(project))
            c.finish(project)
          }
          dirty = true
          return { color: c.png(), emissive: c.hasEmission && c.emission ? c.png(c.emission) : '' }
        }
        store.ready = true
        dirty = true
      })
      .catch((e) => store.say(String(e), true))
      .finally(() => {
        if (!cancelled) store.baking = false
      })
    return () => {
      cancelled = true
    }
  })

  $effect(() => {
    JSON.stringify(store.project.layers)
    JSON.stringify(store.project.finish)
    JSON.stringify(store.project.shading)
    JSON.stringify(store.project.groups)
    JSON.stringify(store.project.variants)
    store.variant
    store.selected
    dirty = true
  })

  $effect(() => {
    if (!stage || !compositor) return
    const texture = store.view === 'mask' ? compositor.mask.texture : compositor.out.texture
    stage.setTexture(texture, compositor.bake, store.view)
  })

  $effect(() => {
    if (!stage || !compositor) return
    const c = compositor
    stage.flat.setSources(
      { color: c.out.texture, mask: c.mask.texture, glow: () => (c.hasEmission && c.emission ? c.emission.texture : null), geo: c.bake.geo, pos: c.bake.pos, size: store.project.size },
      c.bake.posData,
    )
  })

  $effect(() => {
    if (!stage) return
    const on = store.flat
    if (on && !stage.flat.active) stage.flat.fit()
    stage.setFlat(on)
    stage.flat.map = store.flatMap
    stage.flat.showUV = store.flatUV
    store.hoverTexel = null
  })

  $effect(() => {
    if (!stage) return
    stage.setLight({ ...store.project.light })
    stage.setBloom(store.bloom, store.project.light.bloom)
    stage.setGrid(store.grid)
    stage.torch.set(store.torch, store.project.light.torch)
  })

  $effect(() => {
    stage?.setResolution(store.lines)
  })

  $effect(() => {
    const nav = tools.tool === 'nav'
    const islands = store.picksIslands
    stage?.setHover(nav && !islands ? store.hoverPart : -1, nav && islands ? store.hoverIsland : -1)
  })

  $effect(() => {
    if (stage) stage.controls.leftDrag = tools.tool === 'nav'
    if (stage && tools.tool !== 'brush') stage.setRing(null)
  })

  // A newly selected layer flashes where it shows, a shown one in the list holds it.
  let selectedBefore = -1
  $effect(() => {
    const c = compositor
    const hover = store.hoverLayer
    const selected = store.selected
    if (!stage || !c) return
    untrack(() => c.renderMask(withVariant($state.snapshot(store.project), store.variant), hover >= 0 ? hover : selected))
    if (selectedBefore >= 0 && selected !== selectedBefore) flashed = performance.now()
    selectedBefore = selected
  })

  // The edited volume and its gizmo, while the volume tool is on.
  $effect(() => {
    const m = tools.tool === 'volume' && !store.flat ? tools.volume : undefined
    const model = store.model
    const v = m && $state.snapshot(m.volume)
    stage?.volume.set(v && model ? v : null, model ? model.min[0] + (model.extent[0] * model.diag) / 2 : 0)
  })

  $effect(() => {
    stage?.volume.setMode(tools.gizmo)
  })

  // The picked stamp's frame follows edits from the panel and keys.
  $effect(() => {
    const p = tools.pickedStamp
    if (!stage || tools.ghost) return
    stage.setOutline(p ? corners($state.snapshot(p)) : null, '#7fc4af')
  })

  // Its grips stay put while the next stamp's preview follows the cursor.
  $effect(() => {
    const p = tools.pickedStamp
    const g = p && tools.tool === 'stamp' ? handles($state.snapshot(p)) : null
    stage?.setGrips(g ? [g.rotate, ...g.corners] : null)
  })

  const flatKeys: Record<string, FlatMap> = { '1': 'texture', '2': 'mask', '3': 'glow', '4': 'ao', '5': 'edges', '6': 'cavity', '7': 'parts' }

  // Camera keys; layer keys live in the app.
  function onkeydown(e: KeyboardEvent) {
    if (!stage || e.metaKey || (e.target as HTMLElement).closest('input, textarea')) return
    if (store.flat) {
      if (flatKeys[e.key]) store.flatMap = flatKeys[e.key]
      else if (e.key === '0' || e.key === 'f') stage.flat.fit()
      else if (e.key === 'w') store.flatUV = !store.flatUV
      else return
      e.preventDefault()
      return
    }
    const c = stage.controls
    const back = e.ctrlKey ? Math.PI : 0
    switch (e.key) {
      case '1':
        c.view(back, 0)
        break
      case '3':
        c.view(Math.PI / 2 + back, 0)
        break
      case '7':
        c.view(Math.round(c.yaw / (Math.PI / 2)) * (Math.PI / 2), e.ctrlKey ? -1.55 : 1.55)
        break
      case '0':
        c.view(0.8, 0.35)
        stage.frame()
        break
      case '5':
        c.setOrtho(!c.ortho)
        break
      case 'f': {
        const hit = hoverPoint && stage.hit(hoverPoint.x, hoverPoint.y)
        if (hit) c.focus(hit.point)
        else stage.frame()
        break
      }
      default:
        return
    }
    e.preventDefault()
  }
</script>

<svelte:window {onkeydown} />

<div class="viewport" class:brush={tools.tool === 'brush'} class:sampling={store.sampling} bind:this={container}></div>
{#if stage && !store.flat && store.model}
  <ViewGizmo {stage} />
{/if}

<style>
  .viewport {
    position: absolute;
    inset: 0;
    background: var(--bg);
    touch-action: none;
  }

  .viewport :global(canvas) {
    display: block;
  }
  .viewport.brush :global(canvas) {
    cursor: none;
  }
  .viewport.sampling :global(canvas) {
    cursor: crosshair;
  }
</style>
