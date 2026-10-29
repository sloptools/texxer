import type { Catalog } from './types';

export interface LayerFill {
  color?: string;
  type?: string;
  params?: Record<string, unknown>;
}

export type MaskKind = 'parts' | 'mesh' | 'noise' | 'color' | 'graph';

export interface MaskStep {
  kind: MaskKind;
  op?: 'multiply' | 'add' | 'subtract';
  invert?: boolean;
  parts?: string[];
  source?: string;
  type?: string;
  params?: Record<string, unknown>;
  color?: string;
  level?: number;
  softness?: number;
  tolerance?: number;
  breakup?: number;
  grow?: number;
}

export interface Layer {
  name: string;
  fill: LayerFill;
  mask?: MaskStep[];
  opacity?: number;
  mode?: string;
  hidden?: boolean;
  post?: boolean;
}

export interface Effect {
  type: string;
  params?: Record<string, unknown>;
  hidden?: boolean;
}

export const meshSources = ['edges', 'cavity', 'ao', 'up', 'down', 'height'];
export const blendModes = ['normal', 'multiply', 'screen', 'overlay', 'add', 'subtract', 'darken', 'lighten'];

/** Materials are node types named mat_*, shipped or saved by the user. */
export function materials(catalog: Catalog): string[] {
  return Object.keys(catalog).filter((t) => t.startsWith('mat_')).sort();
}

export function effects(catalog: Catalog): string[] {
  return Object.keys(catalog)
    .filter((t) => t.startsWith('fx_') || t === 'finish' || t === 'bake' || t === 'grime')
    .sort();
}

/** Mask graphs are named mask_*; noise masks use generators and grunge. */
export function maskGraphs(catalog: Catalog): string[] {
  return Object.keys(catalog).filter((t) => t.startsWith('mask_')).sort();
}

export function noises(catalog: Catalog): string[] {
  return Object.entries(catalog)
    .filter(([, d]) => (d.category === 'grunge' || d.category === 'generator') && d.inputs.length === 0)
    .map(([t]) => t)
    .filter((t) => t !== 'mesh' && t !== 'image')
    .sort();
}

export function newStep(kind: MaskKind): MaskStep {
  switch (kind) {
    case 'parts':
      return { kind, parts: [] };
    case 'mesh':
      return { kind, source: 'edges', level: 0.4, softness: 0.15, breakup: 0.8 };
    case 'noise':
      return { kind, type: 'dirt', params: {} };
    case 'color':
      return { kind, color: '#808080', tolerance: 0.05 };
    case 'graph':
      return { kind, type: '', params: {} };
  }
}

export function stackOf(params: Record<string, unknown>): { layers: Layer[]; effects: Effect[] } {
  return {
    layers: (params.layers as Layer[] | undefined) ?? [],
    effects: (params.effects as Effect[] | undefined) ?? [],
  };
}

export function move<T>(list: T[], from: number, to: number): T[] {
  if (to < 0 || to >= list.length) return list;
  const out = [...list];
  const [item] = out.splice(from, 1);
  out.splice(to, 0, item);
  return out;
}

export function replace<T>(list: T[], i: number, item: T): T[] {
  return list.map((x, k) => (k === i ? item : x));
}
