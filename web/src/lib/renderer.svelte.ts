import { render } from './api';
import type { Graph } from './types';

const thumbSize = 96;
const parallel = 4;

/** Renders the preview and node thumbnails, dropping results of outdated requests. */
class Renderer {
  thumbs = $state<Record<string, string>>({});
  errors = $state<Record<string, string>>({});
  preview = $state<{ url?: string; error?: string; ms?: number; busy: boolean }>({ busy: false });

  #generation = 0;

  async update(graph: Graph, previewRef: string, size: number, seed: number) {
    const gen = ++this.#generation;
    const current = () => gen === this.#generation;

    this.preview.busy = true;
    const start = performance.now();
    try {
      const url = await render(graph, previewRef, size, seed);
      if (!current()) return URL.revokeObjectURL(url);

      this.#replace(this.preview.url, url);
      this.preview = { url, ms: Math.round(performance.now() - start), busy: false };
    } catch (e) {
      if (current()) this.preview = { ...this.preview, error: String((e as Error).message), busy: false };
    }

    const queue = Object.keys(graph.nodes);
    const worker = async () => {
      for (let id = queue.shift(); id && current(); id = queue.shift()) {
        await this.#thumb(graph, id, seed, current);
      }
    };
    await Promise.all(Array.from({ length: parallel }, worker));
  }

  async #thumb(graph: Graph, id: string, seed: number, current: () => boolean) {
    try {
      const url = await render(graph, id, thumbSize, seed);
      if (!current()) return URL.revokeObjectURL(url);

      this.#replace(this.thumbs[id], url);
      this.thumbs[id] = url;
      delete this.errors[id];
    } catch (e) {
      if (current()) this.errors[id] = String((e as Error).message);
    }
  }

  #replace(old: string | undefined, next: string) {
    if (old && old !== next) URL.revokeObjectURL(old);
  }
}

export const renderer = new Renderer();
