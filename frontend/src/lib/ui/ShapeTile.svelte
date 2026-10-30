<script lang="ts" module>
  import { drawDecal, isDecal } from '../decals'
  import { drawShape } from '../shapes'

  const cache = new Map<string, string>()

  // The shape as a mask image, so it takes the text color.
  function maskURL(shape: string): string {
    let url = cache.get(shape)
    if (url) return url
    const canvas = document.createElement('canvas')
    canvas.width = canvas.height = 64
    const ctx = canvas.getContext('2d')!
    if (isDecal(shape)) {
      // Relief grays sit on mid gray, like on a surface under an overlay.
      if (shape.startsWith('relief:')) {
        ctx.fillStyle = '#808080'
        ctx.fillRect(0, 0, 64, 64)
      }
      drawDecal(ctx, shape, 0, 0, 64)
      url = canvas.toDataURL()
    } else {
      drawShape(ctx, shape, 0, 0, 64)
      url = `url(${canvas.toDataURL()})`
    }
    cache.set(shape, url)
    return url
  }
</script>

<script lang="ts">
  import { imageURL } from '../assets'

  let { shape, on, onclick }: { shape: string; on: boolean; onclick: () => void } = $props()
</script>

<button class:on title={shape.replace(/^(img|relief|decal):/, '')} {onclick}>
  {#if shape.startsWith('img:')}
    {#await imageURL(`/stamp?name=${encodeURIComponent(shape.slice(4))}`) then url}
      <img src={url} alt={shape.slice(4)} />
    {/await}
  {:else if isDecal(shape)}
    <img src={maskURL(shape)} alt={shape} />
  {:else}
    <span style:mask-image={maskURL(shape)} style:-webkit-mask-image={maskURL(shape)}></span>
  {/if}
</button>

<style>
  button {
    display: grid;
    place-items: center;
    aspect-ratio: 1;
    padding: 0;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--bg);
    color: var(--dim);
  }
  button:hover {
    color: var(--strong);
  }
  button.on {
    border-color: var(--accent);
    color: var(--strong);
  }
  img {
    width: 30px;
    height: 30px;
    object-fit: contain;
    image-rendering: pixelated;
  }
  span {
    width: 26px;
    height: 26px;
    background: currentColor;
    mask-size: contain;
    -webkit-mask-size: contain;
  }
</style>
