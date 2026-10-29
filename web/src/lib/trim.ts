export interface Strip {
  height: number;
  input: string;
}

export interface StripBox {
  y: number;
  height: number;
  input: string;
}

/** Lays strips out like the trim node: the last one fills the remaining height. */
export function layoutStrips(strips: Strip[], size: number): StripBox[] {
  const boxes: StripBox[] = [];
  let y = 0;

  strips.forEach((s, i) => {
    let h = s.height;
    if (i === strips.length - 1 || y + h > size) h = size - y;
    if (h <= 0) return;

    boxes.push({ y, height: h, input: s.input });
    y += h;
  });
  return boxes;
}

export function snap(v: number, grid: number): number {
  return Math.round(v / grid) * grid;
}
