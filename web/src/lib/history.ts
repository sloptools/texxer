const limit = 200;

/** Undo history of serialized documents. */
export class History {
  #past: string[] = [];
  #future: string[] = [];
  #current = '';

  reset(doc: string) {
    this.#past = [];
    this.#future = [];
    this.#current = doc;
  }

  /** Records doc as a new state unless it is the current one. */
  push(doc: string) {
    if (doc === this.#current) return;

    this.#past.push(this.#current);
    if (this.#past.length > limit) this.#past.shift();
    this.#future = [];
    this.#current = doc;
  }

  undo(): string | undefined {
    const doc = this.#past.pop();
    if (doc === undefined) return;

    this.#future.push(this.#current);
    this.#current = doc;
    return doc;
  }

  redo(): string | undefined {
    const doc = this.#future.pop();
    if (doc === undefined) return;

    this.#past.push(this.#current);
    this.#current = doc;
    return doc;
  }
}
