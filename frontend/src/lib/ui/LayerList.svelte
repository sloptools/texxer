<script lang="ts">
  import Icon from '../Icon.svelte'
  import { store } from '../store.svelte'
  import ContextMenu, { type MenuItem } from './ContextMenu.svelte'
  import type { Group, Layer } from '../types'

  type Row = { kind: 'layer'; key: string; l: Layer; i: number; nested: boolean } | { kind: 'group'; key: string; g: Group; first: number; last: number; count: number }
  type Where = 'above' | 'below' | 'into'

  let renaming = $state('')
  let menu = $state<{ x: number; y: number; items: MenuItem[] } | null>(null)

  // Right click picks the row, then offers what fits it.
  function context(e: MouseEvent, row?: Row) {
    e.preventDefault()
    e.stopPropagation()
    const items: MenuItem[] = []
    if (row?.kind === 'group') {
      const id = row.g.id
      store.selectedGroup = id
      items.push(
        { label: 'new layer in folder', key: 'N', run: () => store.addLayer() },
        { label: 'rename', run: () => (renaming = id) },
        'line',
        { label: 'dissolve folder', key: '⇧⌘G', run: () => store.ungroup(id) },
        { label: 'delete folder and its layers', danger: true, run: () => store.deleteGroup(id) },
      )
    } else if (row?.kind === 'layer') {
      store.selected = row.i
      store.selectedGroup = ''
      items.push(
        { label: 'new layer', key: 'N', run: () => store.addLayer() },
        { label: 'duplicate', key: 'D', run: () => store.duplicateLayer() },
        { label: 'rename', run: () => (renaming = row.key) },
        { label: 'into a new folder', key: '⌘G', run: () => store.groupLayer() },
        'line',
        { label: 'delete', key: '⌫', danger: true, run: () => store.deleteLayer() },
      )
    } else {
      items.push({ label: 'new layer', key: 'N', run: () => store.addLayer() })
    }
    menu = { x: e.clientX, y: e.clientY, items }
  }
  let drag = $state<{ key: string; x: number; y: number; active: boolean } | null>(null)
  let drop = $state<{ key: string; where: Where } | null>(null)

  // Top of the list is the top of the stack; a folder shows where its top layer is.
  const rows = $derived.by(() => {
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
  })

  function rename(target: { name: string }, name: string) {
    if (name.trim()) target.name = name.trim()
    renaming = ''
  }

  function focusInput(el: HTMLInputElement) {
    el.focus()
    el.select()
  }

  function down(e: PointerEvent, row: Row) {
    if (e.button !== 0 || (e.target as HTMLElement).closest('button, input')) return
    drag = { key: row.key, x: e.clientX, y: e.clientY, active: false }
  }

  function move(e: PointerEvent) {
    if (!drag) return
    if (!drag.active && Math.hypot(e.clientX - drag.x, e.clientY - drag.y) < 4) return
    drag.active = true

    const el = (document.elementFromPoint(e.clientX, e.clientY) as HTMLElement | null)?.closest<HTMLElement>('[data-row]')
    const target = rows.find((r) => r.key === el?.dataset.row)
    if (!el || !target || target.key === drag.key) {
      drop = null
      return
    }
    const r = el.getBoundingClientRect()
    const t = (e.clientY - r.top) / r.height
    drop = { key: target.key, where: target.kind === 'group' ? (t < 0.3 ? 'above' : 'into') : t < 0.5 ? 'above' : 'below' }
  }

  function up() {
    const d = drag
    drag = null
    if (!d?.active) return
    const target = rows.find((r) => r.key === drop?.key)
    const source = rows.find((r) => r.key === d.key)
    const where = drop?.where
    drop = null
    if (!target || !source || !where) return

    if (source.kind === 'layer') {
      if (target.kind === 'layer') store.moveLayers([source.key], where === 'above' ? target.i + 1 : target.i, target.l.group)
      else store.moveLayers([source.key], target.last + 1, where === 'into' ? target.g.id : '')
      return
    }

    // A folder moves as a block and never lands inside another one.
    if (target.kind === 'layer' && target.l.group === source.key) return
    const ids = store.project.layers.filter((l) => l.group === source.key).map((l) => l.id)
    let span = target.kind === 'group' ? [target.first, target.last] : [target.i, target.i]
    if (target.kind === 'layer' && target.l.group) {
      const other = rows.find((r) => r.kind === 'group' && r.key === target.l.group)
      if (other?.kind === 'group') span = [other.first, other.last]
    }
    store.moveLayers(ids, where === 'above' ? span[1] + 1 : span[0], source.key)
  }
</script>

<svelte:window onpointermove={move} onpointerup={up} />


<div class="list" class:dragging={drag?.active} role="tree" tabindex="-1" oncontextmenu={(e) => context(e)}>
  {#each rows as row (row.key)}
    {@const dropWhere = drop?.key === row.key ? drop.where : null}
    {#if row.kind === 'group'}
      <div
        class="row group"
        class:on={store.selectedGroup === row.g.id}
        class:zoned={row.g.zone.parts.length + row.g.zone.islands.length > 0}
        class:hidden={!row.g.visible}
        class:lifted={drag?.active && drag.key === row.key}
        class:drop-above={dropWhere === 'above'}
        class:drop-into={dropWhere === 'into'}
        data-row={row.key}
        role="treeitem"
        aria-expanded={row.g.open}
        aria-selected={store.selectedGroup === row.g.id}
        tabindex="-1"
        onpointerdown={(e) => down(e, row)}
        oncontextmenu={(e) => context(e, row)}
        onclick={() => store.selectedGroup !== row.g.id && ((store.selectedGroup = row.g.id), (store.pickingZone = false))}
        onkeydown={() => {}}
        ondblclick={() => (renaming = row.key)}
      >
        <button class="eye" title="Show or hide the folder" onclick={() => (row.g.visible = !row.g.visible)}><Icon name="eye" size={11} /></button>
        <button class="caret" class:open={row.g.open} title="Open or close" onclick={() => (row.g.open = !row.g.open)}><Icon name="caret" size={11} /></button>
        <span class="folder"><Icon name="dir" size={11} /></span>
        {#if renaming === row.key}
          {@render nameInput(row.g)}
        {:else}
          <span class="name">{row.g.name}</span>
        {/if}
        <span class="meta">{row.count}</span>
      </div>
    {:else}
      {@const look = store.look(row.l)}
      <div
        class="row"
        class:nested={row.nested}
        class:on={row.i === store.selected}
        class:hidden={!look.visible || store.group(row.l.group)?.visible === false}
        class:lifted={drag?.active && drag.key === row.key}
        class:drop-above={dropWhere === 'above'}
        class:drop-below={dropWhere === 'below'}
        data-row={row.key}
        role="option"
        aria-selected={row.i === store.selected}
        tabindex="-1"
        onpointerdown={(e) => down(e, row)}
        oncontextmenu={(e) => context(e, row)}
        onclick={() => ((store.selected = row.i), (store.selectedGroup = ''), (store.pickingZone = false))}
        ondblclick={() => (renaming = row.key)}
        onkeydown={() => {}}
      >
        <button class="eye" title="Show or hide (V)" onclick={() => (look.visible = !look.visible)}><Icon name="eye" size={11} /></button>
        <span class="swatch" class:changed={look.changed} style:background={look.color}></span>
        {#if renaming === row.key}
          {@render nameInput(row.l)}
        {:else}
          <span class="name">{row.l.name}</span>
        {/if}
        <span class="meta">
          {#if row.l.masks.length}{row.l.masks.map((m) => m.kind.slice(0, 2)).join('·')}{/if}
          {#if row.l.blend !== 'normal'} {row.l.blend.slice(0, 3)}{/if}
        </span>
      </div>
    {/if}
  {/each}
</div>

{#snippet nameInput(target: { name: string })}
  <input
    value={target.name}
    use:focusInput
    onblur={(e) => rename(target, e.currentTarget.value)}
    onkeydown={(e) => {
      e.stopPropagation()
      if (e.key === 'Enter') rename(target, e.currentTarget.value)
      if (e.key === 'Escape') renaming = ''
    }}
  />
{/snippet}

{#if menu}
  <ContextMenu x={menu.x} y={menu.y} items={menu.items} onclose={() => (menu = null)} />
{/if}

<style>
  .list {
    display: flex;
    flex-direction: column;
    gap: 1px;
    padding: 0 6px;
  }
  .list.dragging {
    cursor: grabbing;
  }

  .row {
    position: relative;
    display: flex;
    align-items: center;
    gap: 8px;
    height: 30px;
    padding: 0 8px 0 4px;
    border-radius: 6px;
    color: var(--fg);
  }
  .row:hover {
    background: var(--hover);
  }
  .row.nested {
    padding-left: 18px;
  }
  .row.group.zoned .folder {
    color: var(--accent);
  }
  .row.on {
    background: var(--active);
    color: var(--strong);
  }
  .row.on::before {
    content: '';
    position: absolute;
    left: -6px;
    top: 7px;
    bottom: 7px;
    width: 2px;
    border-radius: 1px;
    background: var(--accent);
  }
  .row.lifted {
    opacity: 0.4;
  }
  .row.drop-above::after,
  .row.drop-below::after {
    content: '';
    position: absolute;
    left: 4px;
    right: 4px;
    height: 2px;
    border-radius: 1px;
    background: var(--accent);
  }
  .row.drop-above::after {
    top: -2px;
  }
  .row.drop-below::after {
    bottom: -2px;
  }
  .row.drop-into {
    box-shadow: inset 0 0 0 1px var(--accent);
    background: var(--drop);
  }
  .row.hidden .name,
  .row.hidden .swatch,
  .row.hidden .folder {
    opacity: 0.4;
  }

  .group {
    gap: 4px;
    color: var(--strong);
  }
  .folder {
    display: grid;
    place-items: center;
    margin: 0 4px 0 2px;
    color: var(--dir);
  }

  .eye,
  .caret {
    display: grid;
    place-items: center;
    width: 20px;
    height: 20px;
    padding: 0;
    border: 0;
    border-radius: 4px;
    background: none;
    color: var(--dim);
  }
  .row.hidden .eye {
    color: var(--faint);
  }
  .eye:hover,
  .caret:hover {
    color: var(--strong);
  }
  .caret :global(svg) {
    transition: transform 0.12s;
  }
  .caret.open :global(svg) {
    transform: rotate(90deg);
  }

  .swatch {
    flex: none;
    width: 14px;
    height: 14px;
    border-radius: 3px;
    box-shadow: inset 0 0 0 1px rgb(0 0 0 / 0.25);
  }
  .swatch.changed {
    outline: 1px solid var(--accent);
    outline-offset: 1px;
  }

  .name {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  input {
    flex: 1;
    min-width: 0;
    height: 22px;
    padding: 0 6px;
    border: 1px solid var(--dim);
    border-radius: 5px;
    background: var(--bg);
    font: inherit;
    color: var(--strong);
    outline: none;
  }

  .meta {
    font: 10px var(--mono);
    color: var(--faint);
    white-space: nowrap;
  }
</style>
