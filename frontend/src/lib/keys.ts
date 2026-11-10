import type { GizmoMode } from './gl/volume'
import { nudgeSelection, stepSelection } from './layerRows'
import { createProject, cycleModel, exportChanged, exportModel, pickProject } from './projects.svelte'
import { store } from './store.svelte'
import { tools } from './tools.svelte'

// The headlight's strength before H turned it off.
let headlight = 0.6

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
  switch (e.key) {
    case 'b':
      tools.set('brush')
      return true
    case 's':
      tools.set('stamp')
      return true
    case 'Escape':
      if (store.sampling) store.sampling = false
      else if (store.matching) store.matching = false
      else if (store.pickingZone) store.pickingZone = false
      else if (tools.picked) tools.picked = null
      else if (store.marked.length) store.marked = []
      else if (tools.tool !== 'nav') tools.set(tools.tool)
      return true
  }

  if (tools.tool === 'nav') return false
  switch (e.key) {
    case '[':
      tools.resize(1 / 1.2)
      return true
    case ']':
      tools.resize(1.2)
      return true
  }

  const brush = tools.tool === 'brush'
  const stamp = tools.tool === 'stamp'
  if (!brush && !stamp) return false

  switch (e.key) {
    case 'x':
      tools.toggleMirror()
      return true
  }
  if (brush && e.key === 'e') {
    tools.brush.erase = !tools.brush.erase
    return true
  }
  if (!stamp) return false

  switch (e.key) {
    case 'g':
      tools.snap = !tools.snap
      return true
    case 'r':
    case 'R':
      tools.rotate(e.shiftKey ? -15 : 15)
      return true
  }

  // Over the view these act on the picked stamp, without one they do nothing.
  if (store.focus !== 'view') return false
  const picked = !!tools.pickedStamp
  if (!picked) return ['d', 'Backspace', 'Delete'].includes(e.key)

  const step = e.shiftKey ? 0.5 : 0.1
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

// Keys that change the layers, only while the list has the focus, so they never
// hit a layer while painting; true when handled.
function listKey(e: KeyboardEvent): boolean {
  switch (e.key) {
    case 'd':
      store.duplicateLayer()
      return true
    case 'Backspace':
    case 'Delete':
      store.deleteLayer()
      return true
    case 'ArrowUp':
    case 'ArrowDown':
      if (e.altKey) nudgeSelection(e.key === 'ArrowUp')
      else stepSelection(e.key === 'ArrowUp' ? 1 : -1)
      return true
  }
  return false
}

// The app's keys; openMaterials shows the material menu.
export function handleKey(e: KeyboardEvent, openMaterials: () => void) {
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
        if (e.shiftKey) createProject()
        else store.right = !store.right
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
      case 'b':
        store.left = !store.left
        break
      case 'g':
        if (e.shiftKey) store.ungroup()
        else store.groupLayer()
        break
      default:
        handled = false
    }
  } else if (!(store.focus === 'layers' && listKey(e))) {
    switch (e.key) {
      case 'N':
        store.addLayer()
        break
      case 'v':
        if (l) {
          const visible = !store.look(l).visible
          for (const p of store.pickedLayers()) store.look(p).visible = visible
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
        openMaterials()
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
      case 'k':
        if (store.layer) store.sampling = !store.sampling
        break
      case 't':
        store.torch = !store.torch
        break
      case 'c':
        store.cycleVariant()
        break
      case 'a':
        store.cycleChannel()
        break
      case 'o':
        store.cycleOnion()
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
