<script lang="ts">
  import type { Stop } from '../lib/types';

  let { stops, onchange }: { stops: Stop[]; onchange: (s: Stop[]) => void } = $props();

  const sorted = $derived([...stops].sort((a, b) => a.pos - b.pos));
  const css = $derived(
    `linear-gradient(to right, ${sorted.map((s) => `${s.color} ${s.pos * 100}%`).join(', ')})`,
  );

  function update(i: number, patch: Partial<Stop>) {
    onchange(stops.map((s, j) => (j === i ? { ...s, ...patch } : s)));
  }

  function add() {
    const last = sorted.at(-1) ?? { pos: 0, color: '#808080' };
    onchange([...stops, { pos: Math.min(1, last.pos + 0.1), color: last.color }]);
  }
</script>

<div class="flex flex-col gap-1.5">
  <div class="h-4 rounded border border-line" style:background={css}></div>

  {#each stops as stop, i}
    <div class="flex items-center gap-2">
      <input
        type="color"
        class="h-6 w-8 shrink-0 cursor-pointer rounded border border-line bg-field"
        value={stop.color}
        oninput={(e) => update(i, { color: e.currentTarget.value })}
      />
      <input
        type="range"
        class="min-w-0 flex-1 accent-(--color-accent)"
        min="0"
        max="1"
        step="0.01"
        value={stop.pos}
        oninput={(e) => update(i, { pos: parseFloat(e.currentTarget.value) })}
      />
      <span class="w-8 text-right font-mono text-[11px] text-fg-muted">{stop.pos.toFixed(2)}</span>
      <button
        class="text-fg-dim hover:text-danger disabled:opacity-30"
        disabled={stops.length <= 1}
        onclick={() => onchange(stops.filter((_, j) => j !== i))}
        title="Remove stop">×</button
      >
    </div>
  {/each}

  <button class="self-start text-[11px] text-fg-muted hover:text-accent" onclick={add}>+ stop</button>
</div>
