<script lang="ts">
  import { render } from '../cut'
  import { snippets } from '../snippets.svelte'
  import Slider from './Slider.svelte'

  const photo = $derived(snippets.current)
  const cut = $derived(snippets.selectedCut)

  let canvas = $state<HTMLCanvasElement>()

  // The styled snippet, again whenever the cut or its look changes.
  $effect(() => {
    const c = cut && $state.snapshot(cut)
    const p = photo
    const target = canvas
    if (!c || !p || !target) return
    let stale = false
    snippets.pixels(p).then((levels) => {
      if (stale) return
      const img = render(levels, c)
      target.width = img.width
      target.height = img.height
      target.getContext('2d')!.putImageData(img, 0, 0)
    })
    return () => (stale = true)
  })

  // Every edit writes the snippet once it rests.
  $effect(() => {
    if (!cut || !photo) return
    JSON.stringify(cut.look) + cut.name
    snippets.changed(photo, cut)
  })
</script>

{#if cut && photo}
  <div class="section">
    <div class="caps">snippet</div>
    <div class="preview"><canvas bind:this={canvas}></canvas></div>
    <input class="name" bind:value={cut.name} />
  </div>
  <div class="section">
    <div class="caps">look</div>
    <Slider label="pixels" bind:value={cut.look.px} min={8} max={256} step={1} digits={0} def={256} />
    <Slider label="width" bind:value={() => cut.look.width ?? 1, (v) => (cut.look.width = v)} min={0.25} max={4} def={1} />
    <Slider label="calm" bind:value={cut.look.calm} def={0.3} />
    <Slider label="saturation" bind:value={cut.look.sat} max={1.5} def={0.85} />
    <Slider label="brightness" bind:value={cut.look.bright} min={-0.5} max={0.5} def={0} />
    <label class="check"><input type="checkbox" bind:checked={cut.look.mono} /> monochrome, tinted by what is below</label>
    <p class="hint">pixels is the long side · calm flattens contrast towards the texture's look</p>
  </div>
  <div class="section">
    <button class="text danger" onclick={() => snippets.deleteCut(photo, cut)}>delete snippet</button>
  </div>
{:else if photo}
  <p class="hint pad">drag a rectangle on the photo, or click four corners for a skewed one</p>
{:else}
  <p class="hint pad">add photos on the left, then cut snippets from them</p>
{/if}

<style>
  .section {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 12px 14px 14px;
  }
  .section + .section {
    border-top: 1px solid var(--line);
  }
  .preview {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 200px;
    border: 1px solid var(--line);
    border-radius: 7px;
    background: var(--bg);
  }
  canvas {
    max-width: 100%;
    max-height: 190px;
    width: auto;
    height: 190px;
    object-fit: contain;
    image-rendering: pixelated;
  }
  .name {
    height: 26px;
    padding: 0 8px;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--bg);
    font: 12px var(--sans);
    color: var(--strong);
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
  .hint {
    margin: 0;
    font-size: 11px;
    color: var(--faint);
  }
  .pad {
    padding: 14px;
  }
  .text {
    align-self: flex-start;
    padding: 0;
    border: 0;
    background: none;
    font: 11px var(--mono);
    color: var(--dim);
  }
  .danger:hover {
    color: var(--error);
  }
</style>
