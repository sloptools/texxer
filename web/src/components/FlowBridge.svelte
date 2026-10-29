<script lang="ts">
  import { useSvelteFlow } from '@xyflow/svelte';
  import { onMount } from 'svelte';

  // Hands the flow API to the parent; it is only available inside <SvelteFlow>.
  const props: { onready: (api: ReturnType<typeof useSvelteFlow>) => void } = $props();

  const api = useSvelteFlow();
  onMount(() => props.onready(api));
</script>
