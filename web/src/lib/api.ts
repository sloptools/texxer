import type { Catalog, Graph } from './types';

async function request(path: string, init?: RequestInit): Promise<Response> {
  const res = await fetch(path, init);
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(body.error ?? res.statusText);
  }
  return res;
}

export async function getCatalog(): Promise<Catalog> {
  return (await request('/api/catalog')).json();
}

export async function listGraphs(): Promise<string[]> {
  return (await request('/api/graphs')).json();
}

export async function loadGraph(name: string): Promise<Graph> {
  return (await request(`/api/graphs/${name}`)).json();
}

export async function saveGraph(name: string, graph: Graph): Promise<void> {
  await request(`/api/graphs/${name}`, {
    method: 'PUT',
    body: JSON.stringify(graph, null, 2),
  });
}

/** Renders a node reference and returns an object URL for the PNG. */
export async function render(graph: Graph, node: string, size: number, seed: number): Promise<string> {
  const res = await request('/api/render', {
    method: 'POST',
    body: JSON.stringify({ graph, node, size, seed }),
  });
  return URL.createObjectURL(await res.blob());
}
