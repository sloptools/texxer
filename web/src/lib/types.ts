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

export interface Graph {
  size: number;
  output: string;
  nodes: Record<string, GraphNode>;
}

export interface TexData extends Record<string, unknown> {
  type: string;
  params: Record<string, unknown>;
}

export interface Stop {
  pos: number;
  color: string;
}
