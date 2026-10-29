import type { Edge } from '@xyflow/svelte';
import { edge, toGraph, type TexNode } from './graph';
import type { Graph } from './types';

const margin = 220;

interface Grouped {
  sub: Graph;
  nodes: TexNode[];
  edges: Edge[];
  /** The parent's new main output, if it was inside the selection. */
  output?: string;
}

/**
 * Moves the selected nodes into a subgraph of the given type. Edges crossing the
 * selection become input and output nodes, and the selection is replaced by one node.
 */
export function group(nodes: TexNode[], edges: Edge[], type: string, size: number, mainOutput: string): Grouped | undefined {
  const picked = nodes.filter((n) => n.selected);
  if (picked.length === 0) return;

  const ids = new Set(picked.map((n) => n.id));
  const taken = new Set(ids);
  const fresh = (base: string) => {
    let id = base;
    for (let i = 2; taken.has(id); i++) id = `${base}${i}`;
    taken.add(id);
    return id;
  };

  const ref = (e: Edge) => `${e.source}.${e.sourceHandle ?? 'out'}`;
  const label = (e: Edge) => (e.sourceHandle && e.sourceHandle !== 'out' ? `${e.source}_${e.sourceHandle}` : e.source);

  const incoming = edges.filter((e) => ids.has(e.target) && !ids.has(e.source));
  const outgoing = edges.filter((e) => ids.has(e.source) && !ids.has(e.target));
  const inner = edges.filter((e) => ids.has(e.source) && ids.has(e.target));

  const xs = picked.map((n) => n.position.x);
  const ys = picked.map((n) => n.position.y);
  const left = Math.min(...xs) - margin;
  const right = Math.max(...xs) + margin;
  const top = Math.min(...ys);

  // One input per distinct upstream output.
  const inputs = new Map<string, { id: string; name: string; from: Edge }>();
  for (const e of incoming) {
    if (inputs.has(ref(e))) continue;
    const name = fresh(label(e));
    inputs.set(ref(e), { id: name, name, from: e });
  }

  // One output per distinct inner output used outside, plus the main output if it moves.
  const [mainId, mainHandle = 'out'] = mainOutput.split('.');
  const sources = outgoing.map((e) => ({ source: e.source, handle: e.sourceHandle ?? 'out' }));
  if (ids.has(mainId)) sources.unshift({ source: mainId, handle: mainHandle });
  if (sources.length === 0) {
    const sink = picked.find((n) => !inner.some((e) => e.source === n.id)) ?? picked[picked.length - 1];
    sources.push({ source: sink.id, handle: 'out' });
  }

  const outputs = new Map<string, { id: string; name: string }>();
  for (const s of sources) {
    const key = `${s.source}.${s.handle}`;
    if (outputs.has(key)) continue;
    const name = outputs.size === 0 ? 'out' : s.handle === 'out' ? s.source : `${s.source}_${s.handle}`;
    outputs.set(key, { id: fresh(name === 'out' ? 'out' : `out_${name}`), name });
  }

  const subNodes: TexNode[] = picked.map((n) => ({ ...n, selected: false }));
  const subEdges: Edge[] = inner.map((e) => ({ ...e }));

  [...inputs.values()].forEach((input, i) => {
    subNodes.push(ioNode(input.id, 'input', input.name, left, top + i * 250));
    for (const e of incoming.filter((e) => ref(e) === ref(input.from))) {
      subEdges.push(edge(input.id, 'out', e.target, e.targetHandle ?? 'in'));
    }
  });

  [...outputs.entries()].forEach(([key, output], i) => {
    subNodes.push(ioNode(output.id, 'output', output.name, right, top + i * 250));
    const [source, handle] = key.split('.');
    subEdges.push(edge(source, handle, output.id, 'in'));
  });

  const sub = toGraph(subNodes, subEdges, { size, output: outputs.values().next().value!.id });

  // The replacement node in the parent.
  const nodeId = freshIn(nodes, ids, type);
  const position = { x: avg(xs), y: avg(ys) };
  const outName = (key: string) => outputs.get(key)!.name;

  const parentEdges = edges.filter((e) => !ids.has(e.source) && !ids.has(e.target));
  for (const input of inputs.values()) {
    parentEdges.push(edge(input.from.source, input.from.sourceHandle ?? 'out', nodeId, input.name));
  }
  for (const e of outgoing) {
    parentEdges.push(edge(nodeId, outName(ref(e)), e.target, e.targetHandle ?? 'in'));
  }

  let output: string | undefined;
  if (ids.has(mainId)) {
    const handle = outName(`${mainId}.${mainHandle}`);
    output = handle === 'out' ? nodeId : `${nodeId}.${handle}`;
  }

  return {
    sub,
    nodes: [
      ...nodes.filter((n) => !ids.has(n.id)).map((n) => ({ ...n, selected: false })),
      { id: nodeId, type: 'tex', position, data: { type, params: {} }, selected: true },
    ],
    edges: parentEdges,
    output,
  };
}

function ioNode(id: string, type: 'input' | 'output', name: string, x: number, y: number): TexNode {
  return { id, type: 'tex', position: { x, y }, data: { type, params: { name } } };
}

function freshIn(nodes: TexNode[], removed: Set<string>, base: string): string {
  const taken = new Set(nodes.filter((n) => !removed.has(n.id)).map((n) => n.id));
  const clean = base.replace(/[^A-Za-z0-9_]/g, '_').replace(/^[^A-Za-z]/, 'g');
  for (let i = 1; ; i++) {
    if (!taken.has(`${clean}${i}`)) return `${clean}${i}`;
  }
}

function avg(v: number[]): number {
  return v.reduce((a, b) => a + b, 0) / v.length;
}
