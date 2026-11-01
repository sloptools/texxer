<script lang="ts">
  import { onMount } from 'svelte'
  import * as api from '../../wailsjs/go/main/App'
  import { Compositor, loadBake } from './gl/compositor'
  import { PaintStore } from './gl/paint'
  import { Stage } from './gl/stage'
  import { withVariant } from './project'
  import { corners } from './stamps'
  import { store } from './store.svelte'
  import { ToolInput } from './toolInput'
  import { tools } from './tools.svelte'
  import type { FlatMap } from './types'

  let container: HTMLDivElement
  let stage = $state<Stage>()
  let compositor = $state<Compositor>()
  let dirty = true
  let hoverPoint: { x: number; y: number } | undefined
  const size = $derived(store.project.size)
  const factor = $derived(store.project.finish.supersample)
  let paletteTimer = 0

  onMount(() => {
    const s = new Stage(container)
    stage = s
    if (import.meta.env.DEV) Object.assign(window, { stage: s })
    const input = new ToolInput(s, () => compositor, () => (dirty = true))
    tools.surface = (p, n, reach) => s.surface(p, n, reach)
    s.onError = (m) => store.say(m, true)

    s.controls.onLight = (d) => (store.project.light.angle = (store.project.light.angle - d * 0.4 + 360) % 360)
    s.controls.onClick = (e) => {
      if (tools.tool !== 'nav') return
      const hit = s.hit(e.clientX, e.clientY)
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

    s.renderer.domElement.addEventListener('pointermove', onMove)
    s.renderer.domElement.addEventListener('pointerleave', onLeave)

    let frame = 0
    const loop = () => {
      frame = requestAnimationFrame(loop)
      if (compositor && dirty) {
        const project = withVariant($state.snapshot(store.project), store.variant)
        compositor.render(project, store.selected, tools.ghost && $state.snapshot(tools.ghost))
        if (project.finish.palette === 'auto') schedulePalette(compositor)
        dirty = false
      }
      if (compositor) {
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
      input.dispose()
      compositor?.dispose()
      store.paint?.dispose()
      store.paint = null
      s.dispose()
      stage = undefined
    }
  })

  // The auto palette follows the texture once edits pause, so dragging stays fast.
  function schedulePalette(c: Compositor) {
    clearTimeout(paletteTimer)
    paletteTimer = setTimeout(() => {
      if (c !== compositor) return
      const project = withVariant($state.snapshot(store.project), store.variant)
      c.pickPalette(project.finish.count)
      c.finish(project)
    }, 150)
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

  // Model and bake, again when the texture size or bake settings change.
  $effect(() => {
    const s = stage
    const model = store.model
    const layout = store.project.unwrap
    const settings = bakeSettings
    if (!s || !model) return

    let cancelled = false
    store.baking = true
    store.ready = false
    const project = store.path
    const key = encodeURIComponent(model.path)
    const files = model.offsets.map((offset, i) => ({ url: `/model?path=${key}&unwrap=${layout}&file=${i}`, offset }))
    // Strokes in flight are saved first, so the new paint store loads them.
    Promise.all([s.load(files), loadBake(model.path, size * factor, layout, settings), store.paint?.flush()])
      .then(([, bake]) => {
        if (cancelled) return
        compositor?.dispose()
        store.paint?.dispose()
        const paint = new PaintStore(
          s.renderer,
          bake,
          (id) => `/paint?project=${encodeURIComponent(project)}&id=${id}`,
          (id, png) => api.SavePaint(project, id, png).catch((e) => store.say(String(e))),
        )
        paint.onChange = () => (dirty = true)
        store.paint = paint
        const c = new Compositor(s.renderer, bake, model, paint, factor)
        c.onLoad = () => (dirty = true)
        s.posData = bake.posData
        s.nrmData = bake.nrmData
        s.posSize = bake.size
        compositor = c
        store.snapshot = (variant) => {
          const project = withVariant($state.snapshot(store.project), variant)
          c.render(project, store.selected)
          if (project.finish.palette === 'auto') {
            c.pickPalette(project.finish.count)
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

  // The picked stamp's frame follows edits from the panel and keys.
  $effect(() => {
    const p = tools.pickedStamp
    if (!stage || tools.ghost) return
    stage.setOutline(p ? corners($state.snapshot(p)) : null, '#7fc4af')
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

<div class="viewport" class:brush={tools.tool === 'brush'} bind:this={container}></div>

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
</style>
