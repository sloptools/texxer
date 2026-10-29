import type { Catalog } from './types';

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
  theme: initialTheme(),
});

export function toggleTheme() {
  editor.theme = editor.theme === 'dark' ? 'light' : 'dark';
  try {
    localStorage.setItem('texxer-theme', editor.theme);
  } catch {
    // Not persisted, still applied for this session.
  }
}
