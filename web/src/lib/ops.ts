import type { Edge } from '@xyflow/svelte';
import { edge, fromGraph, toGraph, uniqueId, type TexNode } from './graph';

export const validId = /^[A-Za-z][A-Za-z0-9_]*$/;

/** Renames a node and rewires every edge that touches it. */
export function rename(nodes: TexNode[], edges: Edge[], from: string, to: string) {
  return {
    nodes: nodes.map((n) => (n.id === from ? { ...n, id: to } : n)),
    edges: edges.map((e) => {
      if (e.source !== from && e.target !== from) return e;
      const source = e.source === from ? to : e.source;
      const target = e.target === from ? to : e.target;
      return edge(source, e.sourceHandle ?? 'out', target, e.targetHandle ?? 'in');
    }),
  };
}

const marker = 'texxer-nodes';

/** Serializes the selected nodes and the edges between them for the clipboard. */
export function copy(nodes: TexNode[], edges: Edge[]): string | undefined {
  const picked = nodes.filter((n) => n.selected);
  if (picked.length === 0) return;

  const ids = new Set(picked.map((n) => n.id));
  const inner = edges.filter((e) => ids.has(e.source) && ids.has(e.target));
  return JSON.stringify({ [marker]: toGraph(picked, inner, 0, '').nodes });
}

/** Adds clipboard nodes under fresh ids, offset and selected. */
export function paste(text: string, nodes: TexNode[], edges: Edge[]) {
  let fragment;
  try {
    fragment = JSON.parse(text)[marker];
  } catch {
    return;
  }
  if (!fragment) return;

  const pasted = fromGraph({ size: 0, output: '', nodes: fragment });
  const taken = [...nodes];
  const ids: Record<string, string> = {};

  for (const n of pasted.nodes) {
    ids[n.id] = uniqueId(n.data.type, taken);
    taken.push({ ...n, id: ids[n.id] });
  }

  const newNodes = pasted.nodes.map((n) => ({
    ...n,
    id: ids[n.id],
    position: { x: n.position.x + 40, y: n.position.y + 40 },
    selected: true,
  }));
  const newEdges = pasted.edges.map((e) => edge(ids[e.source], e.sourceHandle ?? 'out', ids[e.target], e.targetHandle ?? 'in'));

  return {
    nodes: [...nodes.map((n) => ({ ...n, selected: false })), ...newNodes],
    edges: [...edges, ...newEdges],
  };
}
