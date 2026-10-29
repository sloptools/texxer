export type ParamKind = 'float' | 'int' | 'bool' | 'enum' | 'color' | 'string' | 'json';

export interface ParamDef {
  name: string;
  kind: ParamKind;
  default: unknown;
  min?: number;
  max?: number;
  options?: string[];
  px?: boolean;
}

export interface NodeDef {
  category: string;
  inputs: string[];
  outputs: string[];
  params: ParamDef[];
  dynamic?: boolean;
}

export type Catalog = Record<string, NodeDef>;

export interface GraphNode {
  type: string;
  params?: Record<string, unknown>;
  inputs?: Record<string, string>;
  ui?: { x: number; y: number };
}

/** A param of a subgraph that sets "node.param" targets inside it. */
export interface Exposed {
  default?: unknown;
  targets: string[];
  min?: number;
  max?: number;
}

export interface Graph {
  size: number;
  output: string;
  nodes: Record<string, GraphNode>;
  outputs?: Record<string, string>;
  params?: Record<string, Exposed>;
}

export type GraphMeta = Omit<Graph, 'nodes'>;

export interface TexData extends Record<string, unknown> {
  type: string;
  params: Record<string, unknown>;
}

export interface Stop {
  pos: number;
  color: string;
}

/** A model the 3D preview shows the texture on, taken from the graph's mesh node. */
export interface ModelRef {
  path: string;
  material: string;
}
