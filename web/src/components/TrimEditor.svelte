<script lang="ts">
  import { layoutStrips, snap, type Strip } from '../lib/trim';

  let { strips, size, onchange }: { strips: Strip[]; size: number; onchange: (s: Strip[]) => void } = $props();

  let grid = $state(8);
  let bar: HTMLDivElement;

  const boxes = $derived(layoutStrips(strips, size));
  const barHeight = 320;
  const scale = $derived(barHeight / size);

  // Dragging the boundary below strip i moves height between strip i and i + 1.
  function startDrag(i: number, e: PointerEvent) {
    const target = e.currentTarget as HTMLElement;
    target.setPointerCapture(e.pointerId);

    const top = bar.getBoundingClientRect().top;
    const start = boxes[i].y;
    const end = i + 1 < boxes.length ? boxes[i + 1].y + boxes[i + 1].height : size;

    const move = (ev: PointerEvent) => {
      const y = snap((ev.clientY - top) / scale, grid);
      const h = Math.min(Math.max(y - start, grid), end - start - grid);

      const next = strips.map((s) => ({ ...s }));
      next[i].height = h;
      // The last strip fills the rest by itself.
      if (i + 1 < next.length - 1) next[i + 1].height = end - start - h;
      onchange(next);
    };
    const up = () => {
      target.removeEventListener('pointermove', move);
      target.removeEventListener('pointerup', up);
    };
    target.addEventListener('pointermove', move);
    target.addEventListener('pointerup', up);
  }

  function nextName(): string {
    const taken = new Set(strips.map((s) => s.input));
    for (let i = 1; ; i++) if (!taken.has(`s${i}`)) return `s${i}`;
  }

  function split(i: number) {
    const half = snap(boxes[i].height / 2, grid);
    if (half < grid) return;

    const next = strips.map((s, j) => (j === i ? { ...s, height: half } : s));
    next.splice(i + 1, 0, { height: i === strips.length - 1 ? 0 : boxes[i].height - half, input: nextName() });
    onchange(next);
  }

  function remove(i: number) {
    if (strips.length > 1) onchange(strips.filter((_, j) => j !== i));
  }

  function renameStrip(i: number, input: string) {
    if (input && !strips.some((s, j) => j !== i && s.input === input)) {
      onchange(strips.map((s, j) => (j === i ? { ...s, input } : s)));
    }
  }
</script>

<div class="flex flex-col gap-2">
  <label class="flex items-center gap-2 text-[11px] text-fg-muted">
    snap
    <select class="rounded border border-line bg-field px-1" bind:value={grid}>
      {#each [4, 8, 16, 32] as g}
        <option value={g}>{g} px</option>
      {/each}
    </select>
  </label>

  <div bind:this={bar} class="relative rounded border border-line bg-field" style:height="{barHeight}px">
    {#each boxes as box, i (box.input)}
      <div
        class="absolute inset-x-0 flex items-center gap-2 overflow-hidden border-b border-line px-2 text-[11px]"
        style:top="{box.y * scale}px"
        style:height="{box.height * scale}px"
      >
        <input
          class="w-14 rounded bg-transparent font-medium outline-none focus:bg-panel"
          value={box.input}
          onchange={(e) => renameStrip(i, e.currentTarget.value.trim())}
        />
        <span class="text-fg-dim">{box.height} px{i === boxes.length - 1 ? ' · fill' : ''}</span>
        <span class="ml-auto flex gap-2">
          <button class="text-fg-dim hover:text-accent" onclick={() => split(i)} title="Split strip">split</button>
          <button class="text-fg-dim hover:text-danger" onclick={() => remove(i)} title="Remove strip">×</button>
        </span>
      </div>

      {#if i < boxes.length - 1}
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div
          class="absolute inset-x-0 z-10 h-2 -translate-y-1 cursor-row-resize hover:bg-accent/40"
          style:top="{(box.y + box.height) * scale}px"
          onpointerdown={(e) => startDrag(i, e)}
        ></div>
      {/if}
    {/each}
  </div>
</div>
