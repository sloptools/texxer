import type { Catalog, Exposed, GraphMeta } from './types';

export type Theme = 'dark' | 'light';

function initialTheme(): Theme {
  try {
    const saved = localStorage.getItem('texxer-theme');
    if (saved === 'dark' || saved === 'light') return saved;
  } catch {
    // Storage can be unavailable; fall back to the system preference.
  }
  return matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
}

/** State the node components read without prop drilling. */
export const editor = $state({
  catalog: {} as Catalog,
  output: '',
  size: 512,
  outputs: {} as Record<string, string>,
  exposed: {} as Record<string, Exposed>,
  theme: initialTheme(),
  openGraph: undefined as ((name: string) => void) | undefined,
  brush: { radius: 0.03, hardness: 0.5, strength: 0.6, erase: false },
});

/** Everything of the open graph that isn't nodes and edges. */
export function graphMeta(): GraphMeta {
  return { size: editor.size, output: editor.output, outputs: editor.outputs, params: editor.exposed };
}

/** The exposed param that sets node.param, if any. */
export function exposedName(node: string, param: string): string | undefined {
  const target = `${node}.${param}`;
  return Object.keys(editor.exposed).find((name) => editor.exposed[name].targets.includes(target));
}

export function toggleExposed(node: string, param: string) {
  const target = `${node}.${param}`;
  const current = exposedName(node, param);

  if (current) {
    const targets = editor.exposed[current].targets.filter((t) => t !== target);
    const { [current]: _, ...rest } = editor.exposed;
    editor.exposed = targets.length > 0 ? { ...rest, [current]: { ...editor.exposed[current], targets } } : rest;
    return;
  }

  const name = param in editor.exposed || param === 'seed' ? `${node}_${param}` : param;
  editor.exposed = { ...editor.exposed, [name]: { targets: [target] } };
}

export function toggleTheme() {
  editor.theme = editor.theme === 'dark' ? 'light' : 'dark';
  try {
    localStorage.setItem('texxer-theme', editor.theme);
  } catch {
    // Not persisted, still applied for this session.
  }
}
