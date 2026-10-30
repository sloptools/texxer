<script lang="ts">
  // A scrub field: drag sideways to change, shift for fine steps, double-click resets.
  let {
    label,
    value = $bindable(),
    min = 0,
    max = 1,
    step = 0.01,
    def,
    digits = 2,
  }: { label: string; value: number; min?: number; max?: number; step?: number; def?: number; digits?: number } = $props()

  let el: HTMLDivElement
  let drag: { x: number; v: number } | undefined

  const fill = $derived(((value - min) / (max - min)) * 100)

  function set(v: number) {
    value = Math.min(max, Math.max(min, Math.round(v / step) * step))
  }

  function down(e: PointerEvent) {
    el.setPointerCapture(e.pointerId)
    drag = { x: e.clientX, v: value }
  }

  function move(e: PointerEvent) {
    if (!drag) return
    const range = (max - min) * (e.shiftKey ? 0.1 : 1)
    set(drag.v + ((e.clientX - drag.x) / el.clientWidth) * range)
  }

  function key(e: KeyboardEvent) {
    const d = e.key === 'ArrowRight' ? 1 : e.key === 'ArrowLeft' ? -1 : 0
    if (!d) return
    e.preventDefault()
    e.stopPropagation()
    set(value + d * (max - min) * (e.shiftKey ? 0.01 : 0.05))
  }
</script>

<div
  class="slider"
  bind:this={el}
  role="slider"
  tabindex="0"
  aria-label={label}
  aria-valuenow={value}
  aria-valuemin={min}
  aria-valuemax={max}
  onpointerdown={down}
  onpointermove={move}
  onpointerup={() => (drag = undefined)}
  ondblclick={() => def !== undefined && set(def)}
  onkeydown={key}
>
  <div class="fill" style:width="{fill}%"></div>
  <span class="label">{label}</span>
  <span class="value">{value.toFixed(digits)}</span>
</div>

<style>
  .slider {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 26px;
    padding: 0 9px;
    overflow: hidden;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--bg);
    cursor: ew-resize;
    outline: none;
  }
  .slider:focus-visible {
    border-color: var(--dim);
  }

  .fill {
    position: absolute;
    inset: 0 auto 0 0;
    background: var(--active);
    pointer-events: none;
  }

  .label,
  .value {
    position: relative;
    pointer-events: none;
  }
  .label {
    font-size: 12px;
    color: var(--dim);
  }
  .value {
    font: 12px var(--mono);
    color: var(--fg);
  }
</style>
