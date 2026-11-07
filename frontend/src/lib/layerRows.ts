import { store } from './store.svelte'
import type { Group, Layer } from './types'

export type Row = { kind: 'layer'; key: string; l: Layer; i: number; nested: boolean } | { kind: 'group'; key: string; g: Group; first: number; last: number; count: number }
export type Where = 'above' | 'below' | 'into'
// out: below a folder's bottom layer, but outside the folder.
export type Drop = { key: string; where: Where; out: boolean }

// The list as shown: top of the stack first, a folder where its top layer is, its
// layers below it while it is open.
export function layerRows(): Row[] {
  const layers = store.project.layers
  const out: Row[] = []
  const seen = new Set<string>()
  for (let i = layers.length - 1; i >= 0; i--) {
    const l = layers[i]
    const g = l.group ? store.group(l.group) : undefined
    if (!g) {
      out.push({ kind: 'layer', key: l.id, l, i, nested: false })
      continue
    }
    if (seen.has(g.id)) continue
    seen.add(g.id)
    const members = layers.map((m, j) => [m, j] as const).filter(([m]) => m.group === g.id)
    out.push({ kind: 'group', key: g.id, g, first: members[0][1], last: members[members.length - 1][1], count: members.length })
    if (g.open) for (const [m, j] of members.reverse()) out.push({ kind: 'layer', key: m.id, l: m, i: j, nested: true })
  }
  return out
}

const folderOf = (rows: Row[], l: Layer) => rows.find((r): r is Extract<Row, { kind: 'group' }> => r.kind === 'group' && r.key === l.group)

// A marked layer takes the other picked ones along.
export function dragged(key: string): string[] {
  const picked = store.pickedLayers().map((l) => l.id)
  return picked.includes(key) ? picked : [key]
}

// Moves a row to a drop place; folders move as a block and never land in another one.
export function place(rows: Row[], sourceKey: string, at: Drop) {
  const target = rows.find((r) => r.key === at.key)
  const source = rows.find((r) => r.key === sourceKey)
  if (!target || !source || target.key === source.key) return

  if (source.kind === 'layer') {
    const ids = dragged(source.key)
    if (target.kind === 'layer' && ids.includes(target.key)) return
    if (target.kind === 'group') {
      if (at.where === 'below') store.moveLayers(ids, target.first, '')
      else store.moveLayers(ids, target.last + 1, at.where === 'into' ? target.g.id : '')
    } else if (at.out) {
      const folder = folderOf(rows, target.l)
      if (folder) store.moveLayers(ids, folder.first, '')
    } else store.moveLayers(ids, at.where === 'above' ? target.i + 1 : target.i, target.l.group)
    return
  }

  if (target.kind === 'layer' && target.l.group === source.key) return
  const ids = store.project.layers.filter((l) => l.group === source.key).map((l) => l.id)
  let span = target.kind === 'group' ? [target.first, target.last] : [target.i, target.i]
  if (target.kind === 'layer' && target.l.group) {
    const other = folderOf(rows, target.l)
    if (other) span = [other.first, other.last]
  }
  store.moveLayers(ids, at.where === 'above' ? span[1] + 1 : span[0], source.key)
}

// The key of the selected row: the folder, or the layer.
export function selectedKey(): string | undefined {
  return store.selectedGroup || store.layer?.id
}

export function selectRow(row: Row) {
  if (row.kind === 'group') store.selectedGroup = row.g.id
  else {
    store.selected = row.i
    store.selectedGroup = ''
  }
  store.marked = []
  store.pickingZone = false
}

// ⌘ toggles a layer's mark, ⇧ marks the layers between the selected one and it.
export function markRow(rows: Row[], row: Extract<Row, { kind: 'layer' }>, range: boolean) {
  const selected = store.layer
  if (store.selectedGroup || !selected) return selectRow(row)
  if (range) {
    const a = rows.findIndex((r) => r.key === selected.id)
    const b = rows.findIndex((r) => r.key === row.key)
    const span = rows.slice(Math.min(a, b), Math.max(a, b) + 1)
    store.marked = span.flatMap((r) => (r.kind === 'layer' && r.key !== selected.id ? [r.key] : []))
    return
  }
  if (row.key === selected.id) {
    // The selection passes to a marked layer, if any.
    const next = store.marked[0]
    if (!next) return
    store.selected = store.project.layers.findIndex((l) => l.id === next)
    store.marked = store.marked.slice(1)
  } else if (store.marked.includes(row.key)) store.marked = store.marked.filter((id) => id !== row.key)
  else store.marked = [...store.marked, row.key]
}

// Steps the selection through the rows as shown, so closed folders are one row.
export function stepSelection(delta: number) {
  const rows = layerRows()
  const at = rows.findIndex((r) => r.key === selectedKey())
  const next = rows[Math.max(0, Math.min(rows.length - 1, at - delta))]
  if (next) selectRow(next)
}

// Moves the selected row one place up or down the list, into and out of folders
// on the way, like dragging it past its neighbor.
export function nudgeSelection(up: boolean) {
  const rows = layerRows()
  const key = selectedKey()
  const at = rows.findIndex((r) => r.key === key)
  const row = rows[at]
  if (!row) return
  // A folder skips its own layers.
  const end = row.kind === 'group' && row.g.open ? at + row.count : at
  const next = up ? rows[at - 1] : rows[end + 1]
  if (!next) return

  let drop: Drop
  if (row.kind === 'group') {
    const other = next.kind === 'layer' ? folderOf(rows, next.l) : undefined
    drop = { key: other?.key ?? next.key, where: up ? 'above' : 'below', out: false }
  } else if (up) {
    // Past the own folder's top, out of it; from below an open folder, into its bottom.
    if (next.kind === 'group' && next.g.id === row.l.group) drop = { key: next.key, where: 'above', out: false }
    else if (next.kind === 'layer' && next.l.group !== row.l.group) drop = { key: next.key, where: 'below', out: false }
    else drop = { key: next.key, where: 'above', out: false }
  } else {
    // The folder's bottom layer leaves it, just below.
    const folder = row.l.group ? folderOf(rows, row.l) : undefined
    if (folder && row.i === folder.first) {
      store.moveLayers([row.key], folder.first, '')
      return
    }
    if (next.kind === 'group') drop = { key: next.key, where: next.g.open ? 'into' : 'below', out: false }
    else drop = { key: next.key, where: 'below', out: false }
  }

  place(rows, row.key, drop)
}
