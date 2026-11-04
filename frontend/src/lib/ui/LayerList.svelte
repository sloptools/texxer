<script lang="ts">
  import Icon from '../Icon.svelte'
  import { store } from '../store.svelte'
  import ContextMenu, { type MenuItem } from './ContextMenu.svelte'
  import { layerRows, place, selectedKey, type Drop, type Row, type Where } from '../layerRows'
  import type { Layer } from '../types'

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
  let drag = $state<{ key: string; x: number; y: number; cx: number; cy: number; active: boolean } | null>(null)
  // out: below a folder's bottom layer but left of its indent, so below the folder.
  let drop = $state<Drop | null>(null)
  let list: HTMLDivElement
  let scroll = 0

  const rows = $derived.by(layerRows)

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
    drag = { key: row.key, x: e.clientX, y: e.clientY, cx: e.clientX, cy: e.clientY, active: false }
  }

  const indent = 18

  function move(e: PointerEvent) {
    if (!drag) return
    drag.cx = e.clientX
    drag.cy = e.clientY
    if (!drag.active && Math.hypot(e.clientX - drag.x, e.clientY - drag.y) < 4) return
    drag.active = true
    autoScroll(e.clientY)
    drop = dropAt(e.clientX, e.clientY)
  }

  function dropAt(x: number, y: number): Drop | null {
    const source = rows.find((r) => r.key === drag?.key)
    if (!source) return null
    let el = (document.elementFromPoint(x, y) as HTMLElement | null)?.closest<HTMLElement>('[data-row]')
    let t: number | undefined

    // Below the last row: the bottom of the stack.
    const last = list.querySelector<HTMLElement>('[data-row]:last-of-type')
    if (!el && last && y > last.getBoundingClientRect().bottom && x >= list.getBoundingClientRect().left && x <= list.getBoundingClientRect().right) {
      el = last
      t = 1
    }
    const target = rows.find((r) => r.key === el?.dataset.row)
    if (!el || !target || target.key === source.key) return null
    const r = el.getBoundingClientRect()
    t ??= (y - r.top) / r.height

    let where: Where
    if (target.kind === 'group' && source.kind === 'layer') where = t < 0.3 ? 'above' : !target.g.open && t > 0.7 ? 'below' : 'into'
    else where = t < 0.5 ? 'above' : 'below'

    const folder = target.kind === 'layer' && target.nested ? rows.find((g) => g.kind === 'group' && g.key === target.l.group) : undefined
    const bottom = folder?.kind === 'group' && target.kind === 'layer' && target.i === folder.first
    const out = bottom && where === 'below' && (x < r.left + indent || t === 1)
    return { key: target.key, where, out }
  }

  // Near the list's top or bottom edge, the panel scrolls while dragging.
  function autoScroll(y: number) {
    cancelAnimationFrame(scroll)
    const box = list.closest<HTMLElement>('.scroll')
    if (!box) return
    const r = box.getBoundingClientRect()
    const speed = y < r.top + 30 ? -6 : y > r.bottom - 30 ? 6 : 0
    if (!speed) return
    const step = () => {
      box.scrollTop += speed
      scroll = requestAnimationFrame(step)
    }
    scroll = requestAnimationFrame(step)
  }

  function cancel() {
    cancelAnimationFrame(scroll)
    drag = null
    drop = null
  }

  function up() {
    const d = drag
    const at = drop
    cancel()
    if (d?.active && at) place(rows, d.key, at)
  }

  // Esc drops the drag before the app sees the key; enter renames the selected row.
  function key(e: KeyboardEvent) {
    if (e.key === 'Enter' && !renaming && !(e.target as HTMLElement).closest('input, textarea') && selectedKey()) {
      e.preventDefault()
      renaming = selectedKey()!
      return
    }
    if (e.key !== 'Escape' || !drag?.active) return
    e.stopPropagation()
    e.preventDefault()
    cancel()
  }

  // Dragging over eyes sets them all like the first; ⌥click shows only that layer, again all.
  let painting: boolean | null = null

  function eyeDown(e: PointerEvent, l: Layer) {
    e.stopPropagation()
    const look = store.look(l)
    if (e.altKey) {
      const others = store.project.layers.filter((o) => o.id !== l.id)
      const alone = look.visible && others.every((o) => !store.look(o).visible)
      for (const o of others) store.look(o).visible = alone
      look.visible = true
      return
    }
    painting = !look.visible
    look.visible = painting
  }

  function eyeOver(l: Layer) {
    if (painting !== null) store.look(l).visible = painting
  }

  // The selected row stays in sight when the keys move it.
  $effect(() => {
    const key = selectedKey()
    rows
    if (key) list?.querySelector(`[data-row="${key}"]`)?.scrollIntoView({ block: 'nearest' })
  })

  const dragged = $derived(drag?.active ? rows.find((r) => r.key === drag!.key) : undefined)
</script>

<svelte:window onpointermove={move} onpointerup={() => ((painting = null), up())} onkeydowncapture={key} />


<div class="list" class:dragging={drag?.active} bind:this={list} role="tree" tabindex="-1" oncontextmenu={(e) => context(e)}>
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
        class:drop-below={dropWhere === 'below'}
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
        <button class="eye" title="Show or hide the folder" onclick={(e) => (e.stopPropagation(), (row.g.visible = !row.g.visible))} onpointerdown={(e) => e.stopPropagation()}><Icon name="eye" size={11} /></button>
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
        class:on={!store.selectedGroup && row.i === store.selected}
        class:hidden={!look.visible || store.group(row.l.group)?.visible === false}
        class:lifted={drag?.active && drag.key === row.key}
        class:drop-above={dropWhere === 'above'}
        class:drop-below={dropWhere === 'below'}
        class:drop-out={dropWhere === 'below' && drop?.out}
        data-row={row.key}
        role="option"
        aria-selected={!store.selectedGroup && row.i === store.selected}
        tabindex="-1"
        onpointerenter={() => !drag?.active && (store.hoverLayer = row.i)}
        onpointerleave={() => (store.hoverLayer = -1)}
        onpointerdown={(e) => down(e, row)}
        oncontextmenu={(e) => context(e, row)}
        onclick={() => ((store.selected = row.i), (store.selectedGroup = ''), (store.pickingZone = false))}
        ondblclick={() => (renaming = row.key)}
        onkeydown={() => {}}
      >
        <button class="eye" title="Show or hide (V) · drag over others · ⌥click solo" onpointerdown={(e) => eyeDown(e, row.l)} onclick={(e) => e.stopPropagation()} onpointerenter={() => eyeOver(row.l)}><Icon name="eye" size={11} /></button>
        <span class="swatch" class:changed={look.changed} style:background={look.color}></span>
        {#if renaming === row.key}
          {@render nameInput(row.l)}
        {:else}
          <span class="name">{row.l.name}</span>
        {/if}
        <span class="meta" title={[...row.l.masks.map((m) => m.kind), row.l.blend !== 'normal' ? row.l.blend + ' blend' : '', look.opacity < 1 ? 'opacity ' + Math.round(look.opacity * 100) + '%' : ''].filter(Boolean).join(' · ')}>
          {#each row.l.masks as m}<span class="mask" class:inverted={m.invert}><Icon name={m.kind} size={11} /></span>{/each}
          {#if row.l.blend !== 'normal'} {row.l.blend.slice(0, 3)}{/if}
          {#if look.opacity < 1}<span class="opacity">{Math.round(look.opacity * 100)}</span>{/if}
        </span>
      </div>
    {/if}
  {/each}
</div>

{#if dragged && drag}
  <div class="ghost" style:left="{drag.cx + 12}px" style:top="{drag.cy - 12}px">
    {#if dragged.kind === 'group'}
      <Icon name="dir" size={11} />{dragged.g.name}<span class="meta">{dragged.count}</span>
    {:else}
      <span class="swatch" style:background={store.look(dragged.l).color}></span>{dragged.l.name}
    {/if}
  </div>
{/if}

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
  .row.nested.drop-below::after {
    left: 18px;
  }
  .row.nested.drop-out::after {
    left: 4px;
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

  .ghost {
    position: fixed;
    z-index: 50;
    display: flex;
    align-items: center;
    gap: 8px;
    height: 26px;
    padding: 0 10px;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--raised);
    color: var(--strong);
    box-shadow: 0 4px 14px rgb(0 0 0 / 0.35);
    pointer-events: none;
    white-space: nowrap;
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
    box-shadow:
      inset 0 0 0 1px rgb(0 0 0 / 0.25),
      0 0 0 1px rgb(255 255 255 / 0.1);
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

  .opacity {
    margin-left: 4px;
    color: var(--dim);
  }
  .opacity::after {
    content: '%';
  }

  .meta {
    display: flex;
    align-items: center;
    gap: 2px;
    font: 10px var(--mono);
    color: var(--faint);
    white-space: nowrap;
  }
  .mask.inverted {
    opacity: 0.5;
  }
</style>
