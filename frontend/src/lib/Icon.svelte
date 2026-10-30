<script lang="ts" module>
  // 11×11 pixel grid, same language as sherry's icons.
  const paths = {
    plus: 'M5 2h1v7H5z M2 5h7v1H2z',
    x: 'M2 2h1v1H2z M3 3h1v1H3z M4 4h1v1H4z M5 5h1v1H5z M6 6h1v1H6z M7 7h1v1H7z M8 8h1v1H8z M8 2h1v1H8z M7 3h1v1H7z M6 4h1v1H6z M4 6h1v1H4z M3 7h1v1H3z M2 8h1v1H2z',
    caret: 'M4 2h1v7H4z M5 3h1v5H5z M6 4h1v3H6z M7 5h1v1H7z',
    dir: 'M1 2h4v1h5v7H1z M2 5h7v4H2z',
    export: 'M5 1h1v1h1v1h1v1H6v4H5V4H3V3h1V2h1z M1 7h1v2h7V7h1v3H1z',
    eye: 'M3 3h5v1H3z M2 4h1v1H2z M8 4h1v1H8z M1 5h1v1H1z M9 5h1v1H9z M2 6h1v1H2z M8 6h1v1H8z M3 7h5v1H3z M4 4h3v3H4z M5 5h1v1H5z',
    sun: 'M5 1h1v2H5z M5 8h1v2H5z M1 5h2v1H1z M8 5h2v1H8z M2 2h1v1H2z M8 2h1v1H8z M2 8h1v1H2z M8 8h1v1H8z M4 4h3v3H4z',
    layers: 'M3 2h5v1H3z M1 3h2v1H1z M8 3h2v1H8z M3 4h5v1H3z M1 6h1v1H1z M9 6h1v1H9z M2 7h1v1H2z M8 7h1v1H8z M3 8h5v1H3z',
  } as const
  export type IconName = keyof typeof paths
</script>

<script lang="ts">
  let { name, size = 16.5 }: { name: IconName; size?: number } = $props()
</script>

<svg width={size} height={size} viewBox="0 0 11 11" shape-rendering="crispEdges" aria-hidden="true">
  <path d={paths[name]} fill="currentColor" fill-rule="evenodd" />
</svg>

<style>
  svg {
    flex: none;
    display: block;
  }
</style>
