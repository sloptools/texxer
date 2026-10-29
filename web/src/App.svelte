<script lang="ts">
  import { Background, Controls, SvelteFlow, type Connection, type Edge, type useSvelteFlow } from '@xyflow/svelte';
  import { onMount, untrack } from 'svelte';
  import * as api from './lib/api';
  import { editor } from './lib/editor.svelte';
  import { edge, fromGraph, toGraph, uniqueId, type TexNode } from './lib/graph';
  import { renderer } from './lib/renderer.svelte';
  import AddMenu from './components/AddMenu.svelte';
  import FlowBridge from './components/FlowBridge.svelte';
  import Inspector from './components/Inspector.svelte';
  import Preview from './components/Preview.svelte';
  import TexNodeView from './components/TexNode.svelte';

  const nodeTypes = { tex: TexNodeView };

  let nodes = $state.raw<TexNode[]>([]);
  let edges = $state.raw<Edge[]>([]);
  let size = $state(512);

  let graphs = $state<string[]>([]);
  let name = $state('');
  let savedKey = $state('');
  let seed = $state(0);
  let previewSize = $state(512);
  let message = $state('');
  let menu = $state<{ x: number; y: number } | null>(null);
  let flow: ReturnType<typeof useSvelteFlow> | undefined;

  const selected = $derived(nodes.find((n) => n.selected));
  const previewRef = $derived(selected?.id ?? editor.output);
  const dirty = $derived(name !== '' && JSON.stringify(currentGraph()) !== savedKey);

  // Positions don't affect the result, so dragging nodes doesn't re-render.
  const renderGraph = $derived(toGraph(nodes, edges, size, editor.output, false));
  const renderKey = $derived(JSON.stringify(renderGraph) + previewRef + seed + previewSize);

  $effect(() => {
    renderKey;
    const [g, ref, s, ps] = untrack(() => [renderGraph, previewRef, seed, previewSize] as const);
    if (!ref || !g.nodes[ref]) return;

    const timer = setTimeout(() => renderer.update(g, ref, ps, s), 150);
    return () => clearTimeout(timer);
  });

  onMount(async () => {
    try {
      editor.catalog = await api.getCatalog();
      graphs = await api.listGraphs();
      if (graphs.length) await open(graphs[0]);
    } catch (e) {
      message = (e as Error).message;
    }
  });

  function currentGraph() {
    return toGraph(nodes, edges, size, editor.output);
  }

  async function open(graphName: string) {
    const g = await api.loadGraph(graphName);
    const loaded = fromGraph(g);

    renderer.thumbs = {};
    renderer.errors = {};
    nodes = loaded.nodes;
    edges = loaded.edges;
    size = g.size || 512;
    editor.output = g.output;
    name = graphName;
    savedKey = JSON.stringify(currentGraph());

    setTimeout(() => flow?.fitView({ padding: 0.1 }), 50);
  }

  async function save() {
    if (!name) return;
    try {
      const doc = currentGraph();
      await api.saveGraph(name, doc);
      savedKey = JSON.stringify(doc);
      message = `saved ${name}`;
    } catch (e) {
      message = (e as Error).message;
    }
  }

  async function create() {
    const newName = window.prompt('Graph name (letters, digits, - and _)');
    if (!newName) return;

    nodes = [];
    edges = [];
    editor.output = '';
    name = newName;
    savedKey = '';
    await save();
    graphs = await api.listGraphs();
  }

  function connect(c: Connection): Edge {
    const input = c.targetHandle ?? 'in';
    edges = edges.filter((e) => !(e.target === c.target && e.targetHandle === input));
    return edge(c.source, c.sourceHandle ?? 'out', c.target, input);
  }

  function setParams(id: string, params: Record<string, unknown>) {
    nodes = nodes.map((n) => (n.id === id ? { ...n, data: { ...n.data, params } } : n));
  }

  function addNode(type: string) {
    if (!menu || !flow) return;

    const id = uniqueId(type, nodes);
    const position = flow.screenToFlowPosition({ x: menu.x, y: menu.y });
    nodes = [
      ...nodes.map((n) => ({ ...n, selected: false })),
      { id, type: 'tex', position, data: { type, params: {} }, selected: true },
    ];
    if (!editor.output) editor.output = id;
    menu = null;
  }

  function onkeydown(e: KeyboardEvent) {
    if ((e.metaKey || e.ctrlKey) && e.key === 's') {
      e.preventDefault();
      save();
    }
  }
</script>

<svelte:window {onkeydown} />

<div class="flex h-full flex-col">
  <header class="flex items-center gap-3 border-b border-zinc-800 bg-zinc-900 px-3 py-1.5">
    <span class="font-semibold tracking-tight text-accent">texxer</span>

    <select
      class="rounded border border-zinc-700 bg-zinc-800 px-1 py-0.5"
      value={name}
      onchange={(e) => open(e.currentTarget.value)}
    >
      {#each graphs as g}
        <option value={g}>{g}</option>
      {/each}
    </select>
    <button class="rounded border border-zinc-700 px-2 py-0.5 hover:border-accent" onclick={create}>new</button>
    <button class="rounded border border-zinc-700 px-2 py-0.5 hover:border-accent" onclick={save}>
      save{#if dirty}<span class="text-accent"> •</span>{/if}
    </button>
    <button
      class="rounded border border-zinc-700 px-2 py-0.5 hover:border-accent"
      onclick={(e) => (menu = { x: e.clientX, y: e.clientY + 20 })}>+ node</button
    >

    <span class="truncate text-[11px] text-zinc-500">{message}</span>

    <label class="ml-auto flex items-center gap-1 text-zinc-400">
      seed
      <input type="number" class="w-14 rounded border border-zinc-700 bg-zinc-800 px-1 py-0.5" bind:value={seed} />
    </label>
    <label class="flex items-center gap-1 text-zinc-400">
      preview
      <select class="rounded border border-zinc-700 bg-zinc-800 px-1 py-0.5" bind:value={previewSize}>
        <option value={256}>256</option>
        <option value={512}>512</option>
        <option value={1024}>1024</option>
      </select>
    </label>
  </header>

  <div class="flex min-h-0 flex-1">
    <main class="relative min-w-0 flex-1">
      <SvelteFlow
        bind:nodes
        bind:edges
        {nodeTypes}
        colorMode="dark"
        minZoom={0.1}
        onbeforeconnect={connect}
        onpanecontextmenu={({ event }) => {
          event.preventDefault();
          menu = { x: event.clientX, y: event.clientY };
        }}
      >
        <FlowBridge onready={(a) => (flow = a)} />
        <Background gap={24} />
        <Controls showLock={false} />
      </SvelteFlow>
    </main>

    <aside class="flex w-80 shrink-0 flex-col border-l border-zinc-800 bg-zinc-900/60">
      <Preview label={previewRef || 'no output'} />
      <div class="min-h-0 flex-1 overflow-y-auto">
        {#if selected}
          <Inspector
            node={selected}
            onparams={(p) => setParams(selected.id, p)}
            onoutput={() => (editor.output = selected.id)}
          />
        {:else}
          <p class="p-3 text-zinc-500">Select a node to edit it. Right-click the canvas to add one.</p>
        {/if}
      </div>
    </aside>
  </div>
</div>

{#if menu}
  <AddMenu x={menu.x} y={menu.y} onpick={addNode} onclose={() => (menu = null)} />
{/if}
