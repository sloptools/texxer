<script lang="ts">
  import { Background, Controls, SvelteFlow, type Connection, type Edge, type useSvelteFlow } from '@xyflow/svelte';
  import { onMount, untrack } from 'svelte';
  import * as api from './lib/api';
  import { editor } from './lib/editor.svelte';
  import { edge, fromGraph, inputsOf, toGraph, uniqueId, type TexNode } from './lib/graph';
  import { History } from './lib/history';
  import * as ops from './lib/ops';
  import { renderer } from './lib/renderer.svelte';
  import { layoutStrips, type Strip } from './lib/trim';
  import AddMenu from './components/AddMenu.svelte';
  import FlowBridge from './components/FlowBridge.svelte';
  import Inspector from './components/Inspector.svelte';
  import Preview from './components/Preview.svelte';
  import TexNodeView from './components/TexNode.svelte';
  import Toolbar from './components/Toolbar.svelte';

  const nodeTypes = { tex: TexNodeView };
  const history = new History();

  let nodes = $state.raw<TexNode[]>([]);
  let edges = $state.raw<Edge[]>([]);

  let graphs = $state<string[]>([]);
  let name = $state('');
  let savedKey = $state('');
  let seed = $state(0);
  let previewSize = $state(512);
  let message = $state('');
  let menu = $state<{ x: number; y: number } | null>(null);
  let flow: ReturnType<typeof useSvelteFlow> | undefined;

  const picked = $derived(nodes.filter((n) => n.selected));
  const selected = $derived(picked.length === 1 ? picked[0] : undefined);
  const previewRef = $derived(selected?.id ?? editor.output);
  const docKey = $derived(JSON.stringify(currentGraph()));
  const dirty = $derived(name !== '' && docKey !== savedKey);

  const strips = $derived.by(() => {
    if (selected?.data.type !== 'trim') return [];
    const def = editor.catalog.trim?.params.find((p) => p.name === 'strips')?.default;
    return layoutStrips((selected.data.params.strips ?? def ?? []) as Strip[], editor.size);
  });

  // Positions don't affect the result, so dragging nodes doesn't re-render.
  const renderGraph = $derived(toGraph(nodes, edges, editor.size, editor.output, false));
  const renderKey = $derived(JSON.stringify(renderGraph) + previewRef + seed + previewSize);

  $effect(() => {
    renderKey;
    const [g, ref, s, ps] = untrack(() => [renderGraph, previewRef, seed, previewSize] as const);
    if (!ref || !g.nodes[ref]) return;

    const timer = setTimeout(() => renderer.update(g, ref, ps, s), 150);
    return () => clearTimeout(timer);
  });

  // Edits within a short window, like dragging a slider, become one undo step.
  let pending: string | undefined;
  let pendingTimer: ReturnType<typeof setTimeout> | undefined;

  $effect(() => {
    pending = docKey;
    clearTimeout(pendingTimer);
    pendingTimer = setTimeout(flushHistory, 400);
  });

  $effect(() => {
    document.documentElement.dataset.theme = editor.theme;
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
    return toGraph(nodes, edges, editor.size, editor.output);
  }

  function flushHistory() {
    if (pending !== undefined) history.push(pending);
    pending = undefined;
  }

  function applyDoc(doc: string) {
    const g = JSON.parse(doc);
    const loaded = fromGraph(g);
    const keep = new Set(picked.map((n) => n.id));

    nodes = loaded.nodes.map((n) => ({ ...n, selected: keep.has(n.id) }));
    edges = loaded.edges;
    editor.output = g.output;
  }

  function undo() {
    flushHistory();
    const doc = history.undo();
    if (doc) applyDoc(doc);
  }

  function redo() {
    flushHistory();
    const doc = history.redo();
    if (doc) applyDoc(doc);
  }

  async function open(graphName: string) {
    const g = await api.loadGraph(graphName);
    const loaded = fromGraph(g);

    renderer.thumbs = {};
    renderer.errors = {};
    nodes = loaded.nodes;
    edges = loaded.edges;
    editor.size = g.size || 512;
    editor.output = g.output;
    name = graphName;
    savedKey = JSON.stringify(currentGraph());
    history.reset(savedKey);

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
    history.reset(JSON.stringify(currentGraph()));
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

    // Dynamic nodes can lose inputs, e.g. when a trim strip is removed.
    const node = nodes.find((n) => n.id === id)!;
    const def = editor.catalog[node.data.type];
    if (def?.dynamic) {
      const valid = new Set(inputsOf(def, node.data));
      edges = edges.filter((e) => e.target !== id || valid.has(e.targetHandle ?? ''));
    }
  }

  function rename(from: string, to: string): string | undefined {
    if (!ops.validId.test(to)) return 'Use letters, digits and _, starting with a letter.';
    if (nodes.some((n) => n.id === to)) return `"${to}" already exists.`;

    ({ nodes, edges } = ops.rename(nodes, edges, from, to));
    if (editor.output === from) editor.output = to;
    renderer.thumbs[to] = renderer.thumbs[from];
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

  function pasteText(text: string) {
    const result = ops.paste(text, nodes, edges);
    if (result) ({ nodes, edges } = result);
  }

  function editingText(e: Event) {
    const el = e.target as HTMLElement;
    return el.closest('input, textarea, select') !== null;
  }

  function onkeydown(e: KeyboardEvent) {
    const mod = e.metaKey || e.ctrlKey;
    if (!mod) return;

    if (e.key === 's') {
      e.preventDefault();
      save();
    }
    if (editingText(e)) return;

    if (e.key === 'z' && !e.shiftKey) {
      e.preventDefault();
      undo();
    } else if ((e.key === 'z' && e.shiftKey) || e.key === 'y') {
      e.preventDefault();
      redo();
    } else if (e.key === 'd') {
      e.preventDefault();
      const text = ops.copy(nodes, edges);
      if (text) pasteText(text);
    }
  }

  function oncopy(e: ClipboardEvent) {
    if (editingText(e)) return;
    const text = ops.copy(nodes, edges);
    if (!text) return;

    e.clipboardData?.setData('text/plain', text);
    e.preventDefault();
  }

  function onpaste(e: ClipboardEvent) {
    if (editingText(e)) return;
    const text = e.clipboardData?.getData('text/plain');
    if (text) pasteText(text);
  }
</script>

<svelte:window {onkeydown} {oncopy} {onpaste} />

<div class="flex h-full flex-col">
  <Toolbar
    {graphs}
    {name}
    {dirty}
    {message}
    bind:seed
    bind:previewSize
    onopen={open}
    oncreate={create}
    onsave={save}
    onadd={(x, y) => (menu = { x, y })}
  />

  <div class="flex min-h-0 flex-1">
    <main class="relative min-w-0 flex-1">
      <SvelteFlow
        bind:nodes
        bind:edges
        {nodeTypes}
        colorMode={editor.theme}
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

    <aside class="flex w-80 shrink-0 flex-col border-l border-line-soft bg-panel">
      <Preview label={previewRef || 'no output'} {strips} size={editor.size} />
      <div class="min-h-0 flex-1 overflow-y-auto">
        {#if selected}
          {#key selected.id}
            <Inspector
              node={selected}
              onparams={(p) => setParams(selected.id, p)}
              onoutput={() => (editor.output = selected.id)}
              onrename={(id) => rename(selected.id, id)}
            />
          {/key}
        {:else if picked.length > 1}
          <p class="p-3 text-fg-dim">{picked.length} nodes selected. ⌘C / ⌘V to copy, ⌘D to duplicate, ⌫ to delete.</p>
        {:else}
          <p class="p-3 leading-relaxed text-fg-dim">
            Select a node to edit it. Right-click the canvas to add one. Shift-drag selects several.
            <br />⌘Z / ⇧⌘Z undo and redo, ⌘C / ⌘V copy and paste, ⌘D duplicates.
          </p>
        {/if}
      </div>
    </aside>
  </div>
</div>

{#if menu}
  <AddMenu x={menu.x} y={menu.y} onpick={addNode} onclose={() => (menu = null)} />
{/if}
