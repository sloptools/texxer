<script lang="ts">
  import {
    Background,
    Controls,
    SvelteFlow,
    type Connection,
    type Edge,
    type OnConnectEnd,
    type useSvelteFlow,
  } from '@xyflow/svelte';
  import { onMount, untrack } from 'svelte';
  import * as api from './lib/api';
  import { editor, graphMeta } from './lib/editor.svelte';
  import { edge, fromGraph, inputsOf, toGraph, uniqueId, type TexNode } from './lib/graph';
  import { group } from './lib/group';
  import { History } from './lib/history';
  import * as ops from './lib/ops';
  import { renderer } from './lib/renderer.svelte';
  import { layoutStrips, type Strip } from './lib/trim';
  import AddMenu from './components/AddMenu.svelte';
  import FlowBridge from './components/FlowBridge.svelte';
  import GraphPanel from './components/GraphPanel.svelte';
  import Inspector from './components/Inspector.svelte';
  import Preview from './components/Preview.svelte';
  import TexNodeView from './components/TexNode.svelte';
  import Toolbar from './components/Toolbar.svelte';
  import PaintView from './components/paint/PaintView.svelte';

  const nodeTypes = { tex: TexNodeView };
  const history = new History();

  let nodes = $state.raw<TexNode[]>([]);
  let edges = $state.raw<Edge[]>([]);

  let graphs = $state<string[]>([]);
  let name = $state('');
  let trail = $state<string[]>([]);
  let savedKey = $state('');
  let seed = $state(0);
  let previewSize = $state(512);
  let message = $state('');
  let menu = $state<{ x: number; y: number } | null>(null);
  /** Set when the add menu was opened by dropping a connection on empty canvas. */
  let dangling = $state<{ node: string; handle: string; type: 'source' | 'target' } | null>(null);
  let flow: ReturnType<typeof useSvelteFlow> | undefined;

  let mode = $state<'graph' | 'paint'>('graph');
  /** Selection in the layer list: "l3" for layer 3, "e1" for effect 1. */
  let paintSelected = $state('');
  let showMask = $state(false);

  const picked = $derived(nodes.filter((n) => n.selected));
  const selected = $derived(picked.length === 1 ? picked[0] : undefined);
  const paintNode = $derived(nodes.find((n) => n.data.type === 'layers'));
  const painting = $derived(mode === 'paint' && paintNode !== undefined);
  const previewRef = $derived.by(() => {
    if (!painting) return selected?.id ?? editor.output;

    const id = paintNode!.id;
    const i = paintSelected.startsWith('l') ? Number(paintSelected.slice(1)) : -1;
    const layers = (paintNode!.data.params.layers ?? []) as { mask?: unknown[] }[];
    return showMask && layers[i]?.mask?.length ? `${id}.mask${i}` : id;
  });
  const docKey = $derived(JSON.stringify(currentGraph()));
  const dirty = $derived(name !== '' && docKey !== savedKey);

  const model = $derived.by(() => {
    const node = nodes.find((n) => (n.data.type === 'mesh' || n.data.type === 'layers') && n.data.params.path);
    if (!node) return undefined;
    return { path: String(node.data.params.path), material: String(node.data.params.material ?? '') };
  });

  const strips = $derived.by(() => {
    if (selected?.data.type !== 'trim') return [];
    const def = editor.catalog.trim?.params.find((p) => p.name === 'strips')?.default;
    return layoutStrips((selected.data.params.strips ?? def ?? []) as Strip[], editor.size);
  });

  // Positions don't affect the result, so dragging nodes doesn't re-render.
  const renderGraph = $derived(toGraph(nodes, edges, graphMeta(), false));
  const renderKey = $derived(JSON.stringify(renderGraph) + previewRef + seed + previewSize);

  $effect(() => {
    renderKey;
    const [g, ref, s, ps] = untrack(() => [renderGraph, previewRef, seed, previewSize] as const);
    if (!ref || !g.nodes[ref.split('.')[0]]) return;

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
      editor.openGraph = enter;
      editor.catalog = await api.getCatalog();
      graphs = await api.listGraphs();
      if (graphs.length) await open(graphs[0]);
    } catch (e) {
      message = (e as Error).message;
    }
  });

  function currentGraph() {
    return toGraph(nodes, edges, graphMeta());
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
    editor.outputs = g.outputs ?? {};
    editor.exposed = g.params ?? {};
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
    // Graphs built on a layer stack open for painting.
    mode = loaded.nodes.some((n) => n.data.type === 'layers') ? 'paint' : 'graph';
    paintSelected = '';
    showMask = false;
    editor.output = g.output;
    editor.outputs = g.outputs ?? {};
    editor.exposed = g.params ?? {};
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
      await refreshLibrary();
    } catch (e) {
      message = (e as Error).message;
    }
  }

  /** Saved graphs are node types, so their inputs, outputs and params can change the catalog. */
  async function refreshLibrary() {
    editor.catalog = await api.getCatalog();
    graphs = await api.listGraphs();
  }

  /** Offers to save before the open graph is replaced. Returns false to stay. */
  async function leave(): Promise<boolean> {
    if (!dirty) return true;
    if (window.confirm(`"${name}" has unsaved changes. Save them first?`)) {
      await save();
      return !dirty;
    }
    return window.confirm(`Discard the changes to "${name}"?`);
  }

  async function switchTo(graphName: string) {
    if (!(await leave())) return;
    trail = [];
    await open(graphName);
  }

  /** Opens a subgraph used in the current graph, remembering the way back. */
  async function enter(graphName: string) {
    if (!(await leave())) return;
    trail = [...trail, name];
    await open(graphName);
  }

  async function back(index: number) {
    if (!(await leave())) return;
    const target = trail[index];
    trail = trail.slice(0, index);
    await open(target);
  }

  async function groupSelection() {
    const typeName = window.prompt('Name of the new subgraph (letters, digits, - and _)');
    if (!typeName) return;
    if (!/^[A-Za-z][A-Za-z0-9_-]*$/.test(typeName) || typeName in editor.catalog || graphs.includes(typeName)) {
      message = `can't use "${typeName}" as a graph name`;
      return;
    }

    const result = group(nodes, edges, typeName, editor.size, editor.output);
    if (!result) return;

    try {
      await api.saveGraph(typeName, result.sub);
      await refreshLibrary();
    } catch (e) {
      message = (e as Error).message;
      return;
    }

    nodes = result.nodes;
    edges = result.edges;
    if (result.output) editor.output = result.output;
    message = `grouped into ${typeName}`;
  }

  async function create() {
    const newName = window.prompt('Graph name (letters, digits, - and _)');
    if (!newName) return;

    if (!(await leave())) return;

    nodes = [];
    edges = [];
    editor.output = '';
    editor.outputs = {};
    editor.exposed = {};
    trail = [];
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
    editor.exposed = ops.renameTargets(editor.exposed, from, to);
    renderer.thumbs[to] = renderer.thumbs[from];
  }

  function addNode(type: string) {
    if (!menu || !flow) return;

    const id = uniqueId(type, nodes);
    const position = flow.screenToFlowPosition({ x: menu.x, y: menu.y });
    const data = { type, params: {} };

    // A node fed from the left sits right of the drop point, one feeding an input sits left of it.
    if (dangling?.type === 'target') position.x -= 160;
    position.y -= 20;

    nodes = [...nodes.map((n) => ({ ...n, selected: false })), { id, type: 'tex', position, data, selected: true }];
    if (!editor.output) editor.output = id;

    if (dangling) {
      const def = editor.catalog[type];
      if (dangling.type === 'source') {
        const input = inputsOf(def, data)[0];
        if (input) edges = [...edges, edge(dangling.node, dangling.handle, id, input)];
      } else {
        const output = def?.outputs[0] ?? 'out';
        edges = [...edges.filter((e) => !(e.target === dangling!.node && e.targetHandle === dangling!.handle)), edge(id, output, dangling.node, dangling.handle)];
      }
    }
    closeMenu();
  }

  function closeMenu() {
    menu = null;
    dangling = null;
  }

  /** Dropping a connection on empty canvas opens the add menu to create the other end. */
  const connectEnd: OnConnectEnd = (event, state) => {
    if (!('fromNode' in state) || !state.fromNode || state.toNode || state.isValid) return;

    const handle = state.fromHandle;
    const type = handle.type;
    const id = handle.id ?? (type === 'source' ? 'out' : 'in');

    // Pulling a connected input into the void disconnects it.
    if (type === 'target') {
      edges = edges.filter((e) => !(e.target === state.fromNode!.id && e.targetHandle === id));
    }

    const point = 'changedTouches' in event ? event.changedTouches[0] : event;
    dangling = { node: state.fromNode.id, handle: id, type };
    menu = { x: point.clientX, y: point.clientY };
  };

  function pasteText(text: string) {
    const result = ops.paste(text, nodes, edges);
    if (result) ({ nodes, edges } = result);
  }

  function editingText(e: Event) {
    const el = e.target as HTMLElement;
    return el.closest('input, textarea, select') !== null;
  }

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'Tab' && !editingText(e) && paintNode) {
      e.preventDefault();
      mode = mode === 'paint' ? 'graph' : 'paint';
      return;
    }

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
    } else if (e.key === 'g') {
      e.preventDefault();
      if (picked.length > 0) groupSelection();
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
    {trail}
    {dirty}
    {message}
    bind:seed
    bind:previewSize
    onopen={switchTo}
    oncreate={create}
    onsave={save}
    onadd={(x, y) => (menu = { x, y })}
    onback={back}
    bind:mode
    canPaint={paintNode !== undefined}
  />

  {#if painting}
    <PaintView
      node={paintNode!}
      {model}
      bind:selected={paintSelected}
      bind:showMask
      onparams={(p) => setParams(paintNode!.id, p)}
    />
  {:else}
    <div class="flex min-h-0 flex-1">
      <main class="relative min-w-0 flex-1">
        <SvelteFlow
          bind:nodes
          bind:edges
          {nodeTypes}
          colorMode={editor.theme}
          minZoom={0.1}
          onbeforeconnect={connect}
          onconnectend={connectEnd}
          onedgeclick={({ edge, event }) => {
            if (event.altKey) edges = edges.filter((e) => e.id !== edge.id);
          }}
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
        <Preview label={previewRef || 'no output'} {strips} size={editor.size} {model} />
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
            <div class="flex flex-col gap-2 p-3 text-fg-dim">
              <p>{picked.length} nodes selected. ⌘C / ⌘V to copy, ⌘D to duplicate, ⌫ to delete.</p>
              <button class="self-start rounded border border-line px-2 py-0.5 text-fg hover:border-accent" onclick={groupSelection}>
                group into subgraph <span class="text-fg-faint">⌘G</span>
              </button>
            </div>
          {:else}
            {#if name}<GraphPanel {name} {nodes} />{/if}
            <p class="p-3 leading-relaxed text-fg-dim">
              Select a node to edit it. Right-click the canvas to add one. Shift-drag selects several.
              <br />⌘Z / ⇧⌘Z undo and redo, ⌘C / ⌘V copy and paste, ⌘D duplicates, ⌘G groups into a subgraph.
              Double-click a graph node to open it.
              <br />Drop a connection on empty canvas to add a node there. To disconnect, click a connection and press ⌫,
              ⌥-click it, or drag from the input into empty space.
            </p>
          {/if}
        </div>
      </aside>
    </div>
  {/if}
</div>

{#if menu}
  <AddMenu
    x={menu.x}
    y={menu.y}
    onpick={addNode}
    onclose={closeMenu}
    exclude={name}
    needsInput={dangling?.type === 'source'}
  />
{/if}
