import type { Catalog, Graph } from './types';

export interface LayerFill {
  color?: string;
  type?: string;
  params?: Record<string, unknown>;
}

export type MaskKind = 'parts' | 'mesh' | 'noise' | 'color' | 'decal' | 'paint' | 'graph';

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
  center?: number[];
  direction?: number[];
  size?: number;
  aspect?: number;
  rotation?: number;
  depth?: number;
  file?: string;
  rev?: number;
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

/** Decal motifs are named decal_*. */
export function motifs(catalog: Catalog): string[] {
  return Object.keys(catalog).filter((t) => t.startsWith('decal_')).sort();
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
    case 'decal':
      return { kind, type: 'decal_solid', params: {}, size: 0.15, aspect: 1, rotation: 0, depth: 0.08 };
    case 'paint':
      return { kind };
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

/** A new paint project: the model with a starter stack of base paint, edge wear and dirt, lit and finished. */
export function newProject(path: string, material: string): Graph {
  const layers: Layer[] = [
    { name: 'Basis', fill: { type: 'mat_paint' } },
    {
      name: 'Kantenabnutzung',
      fill: { type: 'mat_bare' },
      mask: [{ kind: 'mesh', source: 'edges', level: 0.55, softness: 0.1, breakup: 1.8 }],
    },
    {
      name: 'Schmutz',
      fill: { type: 'mat_dirt' },
      opacity: 0.55,
      mask: [
        { kind: 'mesh', source: 'cavity', level: 0.3, softness: 0.3, breakup: 0 },
        { kind: 'mesh', source: 'down', level: 0.6, softness: 0.2, breakup: 0, op: 'add' },
        { kind: 'noise', type: 'dirt', params: { scale: 8, coverage: 0.55, softness: 0.3 } },
      ],
    },
  ];
  const effects: Effect[] = [
    { type: 'fx_light' },
    { type: 'finish', params: { factor: 1, sharpen: 0.4, quantize: 1, colors: 128 } },
  ];

  return {
    size: 1024,
    output: 'paint',
    nodes: {
      paint: { type: 'layers', params: { path, material, ao_rays: 48, ao_dist: 0.12, edge_width: 0.0035, cavity_width: 0.012, layers, effects } },
    },
  };
}
