<script lang="ts">
  import { store } from '../store.svelte'
  import { tools } from '../tools.svelte'
  import { blends, type MaskKind } from '../types'
  import ContextMenu, { type MenuItem } from './ContextMenu.svelte'
  import ColorField from './ColorField.svelte'
  import MaskCard from './MaskCard.svelte'
  import Seg from './Seg.svelte'
  import Slider from './Slider.svelte'
  import HelpLink from './HelpLink.svelte'
  import FolderPanel from './FolderPanel.svelte'

  // Picked areas, then geometry, then textures.
  const kinds: (MaskKind | 'line')[] = ['parts', 'islands', 'paint', 'stamps', 'volume', 'line', 'ao', 'edges', 'cavity', 'drips', 'gradient', 'facing', 'line', 'grunge', 'noise']

  let menu = $state<{ x: number; y: number }>()

  const items: MenuItem[] = kinds.map((kind) =>
    kind === 'line'
      ? 'line'
      : {
          label: kind,
          icon: kind,
          run: () => {
            const at = store.addMask(kind)
            if (kind === 'volume' && at >= 0) tools.editVolume(at)
          },
        },
  )

  function openMenu(e: MouseEvent) {
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect()
    menu = { x: r.left, y: r.bottom + 4 }
  }

  const layer = $derived(store.layer)
  const look = $derived(layer && store.look(layer))

  // Colors of the other layers and variants, for quick reuse.
  const used = $derived([
    ...new Set(
      [...store.project.layers.map((l) => l.color), ...store.project.variants.flatMap((v) => Object.values(v.layers).map((l) => l.color ?? ''))]
        .filter(Boolean)
        .map((c) => c.toLowerCase()),
    ),
  ])
</script>

{#if store.selectedGroup}
  <FolderPanel />
{:else if layer && look}
  <div class="section">
    <div class="caps">color{#if store.activeVariant}<span class="variant">{store.activeVariant.name}</span>{/if}</div>
    <ColorField bind:value={look.color} />
    <label class="check"><input type="checkbox" bind:checked={layer.glow} /> glows (emissive)</label>
    {#if layer.masks.some((m) => m.kind === 'stamps')}
      <label class="check"><input type="checkbox" bind:checked={layer.stampColors} /> colors from image stamps</label>
    {/if}
    {#if store.model?.texture || layer.source}
      <label class="check" title="The model's texture as it was before texxer; texxer keeps a copy of it beside the project"><input type="checkbox" bind:checked={layer.source} /> colors from the model's texture</label>
    {/if}
    <div class="swatches">
      {#each used as c}
        <button class="sw" class:on={c === look.color.toLowerCase()} style:background={c} title={c} aria-label={c} onclick={() => (look.color = c)}></button>
      {/each}
    </div>
  </div>

  <div class="section">
    <div class="caps">surface</div>
    <Slider label="tone" bind:value={layer.tone} def={0.08} />
    {#if layer.tone > 0}
      <Slider label="tone scale" bind:value={layer.toneScale} min={1} max={64} step={1} digits={0} def={12} />
    {/if}
    <Slider label="grain" bind:value={layer.grain} max={0.3} def={0.02} />
    <Slider label="shine" bind:value={layer.shine} def={0} />
  </div>

  <div class="section">
    <div class="caps">blend <HelpLink page="layers" /></div>
    <Seg value={layer.blend} options={blends.map((b) => [b, b.slice(0, 4)])} onchange={(v) => (layer.blend = v)} />
    <Slider label="opacity" bind:value={look.opacity} def={1} />
  </div>

  <div class="section">
    <div class="caps">masks <HelpLink page="masks" /></div>
    {#each layer.masks as mask, i (i + mask.kind + layer.id)}
      <MaskCard {mask} index={i} parts={store.model?.parts ?? []} onremove={() => layer.masks.splice(i, 1)} />
    {/each}
    {#if layer.masks.length < 4}
      <button class="add" onclick={openMenu}>+ mask</button>
    {/if}
    {#if !layer.masks.length}
      <p class="hint">no mask: the layer covers everything</p>
    {/if}
  </div>
{:else}
  <p class="hint pad">no layer selected</p>
{/if}

{#if menu}
  <ContextMenu x={menu.x} y={menu.y} {items} onclose={() => (menu = undefined)} />
{/if}

<style>
  .section {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 12px 14px 14px;
  }
  .variant {
    margin-left: 8px;
    color: var(--accent);
    text-transform: none;
  }
  .section + .section {
    border-top: 1px solid var(--line);
  }

  .check {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    color: var(--dim);
  }
  .check input {
    accent-color: var(--accent);
    margin: 0;
  }

  .swatches {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }
  .sw {
    width: 18px;
    height: 18px;
    padding: 0;
    border: 0;
    border-radius: 4px;
    box-shadow: inset 0 0 0 1px rgb(0 0 0 / 0.25);
  }
  .sw.on {
    box-shadow:
      0 0 0 1px var(--panel),
      0 0 0 2px var(--accent);
  }

  .add {
    align-self: flex-start;
    height: 22px;
    padding: 0 7px;
    border: 1px dashed var(--line);
    border-radius: 5px;
    background: none;
    font: 11px var(--mono);
    color: var(--dim);
  }
  .add:hover {
    border-style: solid;
    color: var(--strong);
  }

  .hint {
    margin: 0;
    font-size: 11px;
    color: var(--faint);
  }
  .pad {
    padding: 14px;
  }
</style>
