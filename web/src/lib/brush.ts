import { fileURL, render } from './api';
import type { Graph } from './types';

export interface BrushSettings {
  /** In model diagonals, like the bake's positions. */
  radius: number;
  hardness: number;
  strength: number;
  erase: boolean;
}

/**
 * Paints a mask in the model's UV layout. A stamp at a point on the model
 * reaches every texel whose baked position lies within the radius and that
 * faces the same way, so strokes run across UV seams.
 */
export class Painter {
  readonly canvas: HTMLCanvasElement;
  #ctx: CanvasRenderingContext2D;
  #mask: Float32Array;
  #pos: Uint8ClampedArray;
  #nor: Uint8ClampedArray;
  #w: number;
  #h: number;

  private constructor(w: number, h: number, pos: Uint8ClampedArray, nor: Uint8ClampedArray) {
    this.#w = w;
    this.#h = h;
    this.#pos = pos;
    this.#nor = nor;
    this.#mask = new Float32Array(w * h);
    this.canvas = document.createElement('canvas');
    this.canvas.width = w;
    this.canvas.height = h;
    this.#ctx = this.canvas.getContext('2d', { willReadFrequently: true })!;
  }

  /** Loads the bake maps of a layers node and the mask painted so far. */
  static async load(graph: Graph, node: string, size: number, file?: string): Promise<Painter> {
    const [pos, nor] = await Promise.all([pixels(await render(graph, `${node}.position`, size, 0)), pixels(await render(graph, `${node}.normal`, size, 0))]);
    const p = new Painter(size, size, pos, nor);

    if (file) {
      try {
        const existing = await pixels(fileURL(file) + `&t=${Date.now()}`, size);
        for (let i = 0; i < p.#mask.length; i++) p.#mask[i] = existing[i * 4] / 255;
      } catch {
        // Nothing painted yet.
      }
    }
    p.#draw();
    return p;
  }

  /** One dab at a model position with its normal, both in bake units. */
  stamp(at: number[], normal: number[], b: BrushSettings) {
    const r2 = b.radius * b.radius;
    const inner = Math.min(0.999, b.hardness);
    const target = b.erase ? 0 : 1;

    for (let i = 0; i < this.#mask.length; i++) {
      const k = i * 4;
      const dx = this.#pos[k] / 255 - at[0];
      const dy = this.#pos[k + 1] / 255 - at[1];
      const dz = this.#pos[k + 2] / 255 - at[2];
      const d2 = dx * dx + dy * dy + dz * dz;
      if (d2 > r2) continue;

      const facing = (this.#nor[k] / 127.5 - 1) * normal[0] + (this.#nor[k + 1] / 127.5 - 1) * normal[1] + (this.#nor[k + 2] / 127.5 - 1) * normal[2];
      if (facing < 0.1) continue;

      const t = Math.sqrt(d2) / b.radius;
      const falloff = t <= inner ? 1 : 1 - (t - inner) / (1 - inner);
      const a = falloff * b.strength;
      this.#mask[i] += (target - this.#mask[i]) * a;
    }
    this.#draw();
  }

  clear() {
    this.#mask.fill(0);
    this.#draw();
  }

  async png(): Promise<Blob> {
    return new Promise((resolve, reject) => this.canvas.toBlob((b) => (b ? resolve(b) : reject(new Error('could not encode mask'))), 'image/png'));
  }

  #draw() {
    const img = this.#ctx.createImageData(this.#w, this.#h);
    for (let i = 0; i < this.#mask.length; i++) {
      const v = Math.round(this.#mask[i] * 255);
      img.data[i * 4] = img.data[i * 4 + 1] = img.data[i * 4 + 2] = v;
      img.data[i * 4 + 3] = 255;
    }
    this.#ctx.putImageData(img, 0, 0);
  }
}

/** Decodes an image URL into RGBA pixels, scaled to size if given. */
async function pixels(url: string, size?: number): Promise<Uint8ClampedArray> {
  const res = await fetch(url);
  if (!res.ok) throw new Error(res.statusText);
  const bitmap = await createImageBitmap(await res.blob(), { colorSpaceConversion: 'none', premultiplyAlpha: 'none' });

  const w = size ?? bitmap.width;
  const h = size ?? bitmap.height;
  const canvas = document.createElement('canvas');
  canvas.width = w;
  canvas.height = h;
  const ctx = canvas.getContext('2d', { willReadFrequently: true })!;
  ctx.drawImage(bitmap, 0, 0, w, h);
  return ctx.getImageData(0, 0, w, h).data;
}
