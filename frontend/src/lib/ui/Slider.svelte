<script lang="ts">
  import { tick } from 'svelte'

  // A scrub field: drag sideways to change, shift for fine steps, a click types a
  // value, double-click resets.
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
  let input = $state<HTMLInputElement>()
  let drag: { x: number; v: number; moved: boolean } | undefined
  let editing = $state(false)

  const fill = $derived(((value - min) / (max - min)) * 100)

  function set(v: number) {
    value = Math.min(max, Math.max(min, +(Math.round(v / step) * step).toFixed(6)))
  }

  function down(e: PointerEvent) {
    if (editing) return
    el.setPointerCapture(e.pointerId)
    drag = { x: e.clientX, v: value, moved: false }
  }

  function move(e: PointerEvent) {
    if (!drag) return
    if (Math.abs(e.clientX - drag.x) > 2) drag.moved = true
    if (!drag.moved) return
    const range = (max - min) * (e.shiftKey ? 0.1 : 1)
    set(drag.v + ((e.clientX - drag.x) / el.clientWidth) * range)
  }

  async function up() {
    const clicked = drag && !drag.moved
    drag = undefined
    if (!clicked) return
    editing = true
    await tick()
    input?.select()
  }

  function commit() {
    if (!editing) return
    const v = parseFloat(input?.value.replace(',', '.') ?? '')
    if (isFinite(v)) set(v)
    editing = false
  }

  function reset() {
    editing = false
    if (def !== undefined) set(def)
  }

  function key(e: KeyboardEvent) {
    if (editing) return
    if (e.key === 'Enter') {
      editing = true
      tick().then(() => input?.select())
      return
    }
    const d = e.key === 'ArrowRight' ? 1 : e.key === 'ArrowLeft' ? -1 : 0
    if (!d) return
    e.preventDefault()
    e.stopPropagation()
    set(value + d * (max - min) * (e.shiftKey ? 0.01 : 0.05))
  }

  function inputKey(e: KeyboardEvent) {
    e.stopPropagation()
    if (e.key === 'Enter') commit()
    else if (e.key === 'Escape') editing = false
  }
</script>

<div
  class="slider"
  class:editing
  bind:this={el}
  role="slider"
  tabindex="0"
  aria-label={label}
  aria-valuenow={value}
  aria-valuemin={min}
  aria-valuemax={max}
  onpointerdown={down}
  onpointermove={move}
  onpointerup={up}
  ondblclick={reset}
  onkeydown={key}
>
  <div class="fill" style:width="{fill}%"></div>
  <span class="label">{label}</span>
  {#if editing}
    <input bind:this={input} value={value.toFixed(digits)} onkeydown={inputKey} onblur={commit} />
  {:else}
    <span class="value">{value.toFixed(digits)}</span>
  {/if}
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
  .slider:focus-visible,
  .slider.editing {
    border-color: var(--dim);
  }
  .slider.editing {
    cursor: text;
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
  .value,
  input {
    font: 12px var(--mono);
    color: var(--fg);
  }
  input {
    position: relative;
    width: 70px;
    padding: 0;
    border: 0;
    background: none;
    outline: none;
    text-align: right;
  }
</style>
