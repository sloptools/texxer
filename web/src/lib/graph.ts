import type { Edge, Node } from '@xyflow/svelte';
import type { Graph, GraphNode, NodeDef, TexData } from './types';

export type TexNode = Node<TexData>;

const columnWidth = 200;
const rowHeight = 250;

export function fromGraph(g: Graph): { nodes: TexNode[]; edges: Edge[] } {
  const layout = autoLayout(g);

  const nodes: TexNode[] = Object.entries(g.nodes).map(([id, n]) => ({
    id,
    type: 'tex',
    position: n.ui ?? layout[id],
    data: { type: n.type, params: n.params ?? {} },
  }));

  const edges: Edge[] = [];
  for (const [id, n] of Object.entries(g.nodes)) {
    for (const [input, ref] of Object.entries(n.inputs ?? {})) {
      const [source, output = 'out'] = ref.split('.');
      edges.push(edge(source, output, id, input));
    }
  }

  return { nodes, edges };
}

export function toGraph(nodes: TexNode[], edges: Edge[], size: number, output: string, withUI = true): Graph {
  const g: Graph = { size, output, nodes: {} };

  for (const n of nodes) {
    const node: GraphNode = { type: n.data.type };
    if (Object.keys(n.data.params).length > 0) node.params = n.data.params;
    if (withUI) node.ui = { x: Math.round(n.position.x), y: Math.round(n.position.y) };
    g.nodes[n.id] = node;
  }

  for (const e of edges) {
    const target = g.nodes[e.target];
    const source = g.nodes[e.source];
    if (!target || !source) continue;

    const output = e.sourceHandle ?? 'out';
    target.inputs ??= {};
    target.inputs[e.targetHandle ?? 'in'] = output === 'out' ? e.source : `${e.source}.${output}`;
  }

  return g;
}

export function edge(source: string, output: string, target: string, input: string): Edge {
  return {
    id: `${target}:${input}`,
    source,
    sourceHandle: output,
    target,
    targetHandle: input,
  };
}

/** Input names of a node. Dynamic nodes (trim) take theirs from params. */
export function inputsOf(def: NodeDef | undefined, data: TexData): string[] {
  if (!def) return [];
  if (!def.dynamic) return def.inputs;

  const strips = (data.params.strips ?? def.params.find((p) => p.name === 'strips')?.default) as
    | { input: string }[]
    | undefined;
  return (strips ?? []).map((s) => s.input).filter(Boolean);
}

export function uniqueId(type: string, nodes: TexNode[]): string {
  const taken = new Set(nodes.map((n) => n.id));
  for (let i = 1; ; i++) {
    if (!taken.has(`${type}${i}`)) return `${type}${i}`;
  }
}

/** Places nodes in columns by their distance from the sources. */
function autoLayout(g: Graph): Record<string, { x: number; y: number }> {
  const depth: Record<string, number> = {};

  const visit = (id: string, stack: Set<string>): number => {
    if (id in depth) return depth[id];
    if (stack.has(id) || !g.nodes[id]) return 0;

    stack.add(id);
    const ups = Object.values(g.nodes[id].inputs ?? {}).map((ref) => visit(ref.split('.')[0], stack) + 1);
    stack.delete(id);

    depth[id] = Math.max(0, ...ups);
    return depth[id];
  };

  const rows: Record<number, number> = {};
  const out: Record<string, { x: number; y: number }> = {};

  for (const id of Object.keys(g.nodes)) {
    const d = visit(id, new Set());
    const row = rows[d] ?? 0;
    rows[d] = row + 1;
    out[id] = { x: d * columnWidth, y: row * rowHeight };
  }
  return out;
}
