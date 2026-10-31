<script lang="ts">
  import { store } from '../store.svelte'

  let { page }: { page: string } = $props()
</script>

<button title="Help" onclick={() => store.openDocs(page)}>?</button>

<style>
  button {
    height: 16px;
    padding: 0 4px;
    border: 0;
    border-radius: 4px;
    background: none;
    font: 11px var(--mono);
    color: var(--faint);
  }
  button:hover {
    background: var(--hover);
    color: var(--strong);
  }
</style>
