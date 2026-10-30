<script lang="ts">
  import { onMount } from 'svelte'
  import * as api from '../../wailsjs/go/main/App'
  import { Compositor, loadBake } from './gl/compositor'
  import { PaintStore } from './gl/paint'
  import { Stage } from './gl/stage'
  import { corners } from './stamps'
  import { store } from './store.svelte'
  import { ToolInput } from './toolInput'
  import { tools } from './tools.svelte'

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

    s.controls.onLight = (d) => (store.project.light.angle = (store.project.light.angle - d * 0.4 + 360) % 360)
    s.controls.onClick = (e) => {
      if (tools.tool !== 'nav') return
      const hit = s.hit(e.clientX, e.clientY)
      if (!hit || hit.part < 0) return
      if (e.metaKey || store.layer?.masks.some((m) => m.kind === 'parts')) store.togglePart(hit.part)
    }

    const onMove = (e: PointerEvent) => {
      hoverPoint = e.buttons === 0 ? { x: e.clientX, y: e.clientY } : undefined
    }
    const onLeave = () => {
      hoverPoint = undefined
      store.hoverPart = -1
    }
    s.renderer.domElement.addEventListener('pointermove', onMove)
    s.renderer.domElement.addEventListener('pointerleave', onLeave)

    let frame = 0
    const loop = () => {
      frame = requestAnimationFrame(loop)
      if (compositor && dirty) {
        const project = $state.snapshot(store.project)
        compositor.render(project, store.selected, tools.ghost && $state.snapshot(tools.ghost))
        if (project.finish.palette === 'auto') schedulePalette(compositor)
        dirty = false
      }
      if (hoverPoint) {
        const hit = s.hit(hoverPoint.x, hoverPoint.y)
        store.hoverPart = hit?.part ?? -1
      }
      s.render()
    }
    loop()

    const media = matchMedia('(prefers-color-scheme: dark)')
    const onScheme = () => background()
    media.addEventListener('change', onScheme)

    return () => {
      cancelAnimationFrame(frame)
      media.removeEventListener('change', onScheme)
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
      const project = $state.snapshot(store.project)
      c.pickPalette(project.finish.count)
      c.finish(project)
    }, 150)
  }

  function background() {
    // Wait a frame, so a theme switch has applied its CSS.
    requestAnimationFrame(() => stage?.setBackground(getComputedStyle(container).backgroundColor))
  }

  // Model and bake, again when the texture size changes.
  $effect(() => {
    const s = stage
    const model = store.model
    if (!s || !model) return

    let cancelled = false
    store.baking = true
    Promise.all([s.load(`/model?path=${encodeURIComponent(model.path)}`), loadBake(model.path, size * factor)])
      .then(([, bake]) => {
        if (cancelled) return
        compositor?.dispose()
        store.paint?.dispose()
        const paint = new PaintStore(
          s.renderer,
          bake,
          (id) => `/paint?path=${encodeURIComponent(model.path)}&id=${id}`,
          (id, png) => api.SavePaint(model.path, id, png).catch((e) => store.say(String(e))),
        )
        paint.onChange = () => (dirty = true)
        store.paint = paint
        const c = new Compositor(s.renderer, bake, model, paint, factor)
        c.onLoad = () => (dirty = true)
        s.posData = bake.posData
        s.posSize = bake.size
        compositor = c
        store.snapshot = () => {
          const project = $state.snapshot(store.project)
          c.render(project, store.selected)
          if (project.finish.palette === 'auto') {
            c.pickPalette(project.finish.count)
            c.finish(project)
          }
          return c.png()
        }
        dirty = true
      })
      .catch((e) => store.say(String(e)))
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
    store.selected
    dirty = true
  })

  $effect(() => {
    if (!stage || !compositor) return
    const texture = store.view === 'mask' ? compositor.mask.texture : compositor.out.texture
    stage.setTexture(texture, compositor.bake, store.view)
  })

  $effect(() => {
    if (!stage) return
    stage.setLight({ ...store.project.light })
    stage.setBloom(store.bloom)
  })

  $effect(() => {
    stage?.setHover(tools.tool === 'nav' ? store.hoverPart : -1)
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

  $effect(() => {
    store.theme
    if (stage) background()
  })

  // Camera keys; layer keys live in the app.
  function onkeydown(e: KeyboardEvent) {
    if (!stage || e.metaKey || (e.target as HTMLElement).closest('input, textarea')) return
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
