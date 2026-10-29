<script lang="ts">
  import type { ParamDef, Stop } from '../lib/types';
  import { editor } from '../lib/editor.svelte';
  import type { Strip } from '../lib/trim';
  import GradientEditor from './GradientEditor.svelte';
  import TrimEditor from './TrimEditor.svelte';

  let {
    param,
    value,
    onchange,
    onreset,
  }: {
    param: ParamDef;
    value: unknown;
    onchange: (v: unknown) => void;
    onreset: () => void;
  } = $props();

  const set = $derived(value !== undefined);
  const current = $derived(set ? value : param.default);

  // Presets carry no ranges, so guess one from the default.
  const range = $derived.by(() => {
    if (param.min || param.max) return [param.min ?? 0, param.max ?? 1];
    const d = Number(param.default) || 0;
    return [Math.min(0, d), d <= 1 ? 1 : d * 4];
  });
  const step = $derived(param.kind === 'int' ? 1 : (range[1] - range[0]) / 500);

  let jsonText = $state('');
  let jsonError = $state('');
  $effect(() => {
    jsonText = JSON.stringify(current, null, 1);
    jsonError = '';
  });

  function setNumber(raw: string) {
    const n = param.kind === 'int' ? parseInt(raw) : parseFloat(raw);
    if (!Number.isNaN(n)) onchange(n);
  }

  function setJSON(raw: string) {
    try {
      onchange(JSON.parse(raw));
      jsonError = '';
    } catch (e) {
      jsonError = (e as Error).message;
    }
  }
</script>

<div class="flex flex-col gap-1">
  <div class="flex items-center justify-between">
    <span class={set ? 'text-fg' : 'text-fg-dim'}>
      {param.name}
      {#if param.px}<span class="text-fg-faint">px</span>{/if}
    </span>
    {#if set}
      <button class="text-[11px] text-fg-dim hover:text-accent" onclick={onreset} title="Reset to default">reset</button>
    {/if}
  </div>

  {#if param.kind === 'float' || param.kind === 'int'}
    <div class="flex items-center gap-2">
      <input
        type="range"
        class="min-w-0 flex-1 accent-(--color-accent)"
        min={range[0]}
        max={range[1]}
        {step}
        value={current as number}
        oninput={(e) => setNumber(e.currentTarget.value)}
      />
      <input
        type="number"
        class="w-16 rounded border border-line bg-field px-1 py-0.5 text-right font-mono text-xs"
        step={param.kind === 'int' ? 1 : 'any'}
        value={current as number}
        onchange={(e) => setNumber(e.currentTarget.value)}
      />
    </div>
  {:else if param.kind === 'bool'}
    <label class="flex items-center gap-2">
      <input
        type="checkbox"
        class="accent-(--color-accent)"
        checked={current as boolean}
        onchange={(e) => onchange(e.currentTarget.checked)}
      />
      <span class="text-fg-muted">{current ? 'on' : 'off'}</span>
    </label>
  {:else if param.kind === 'enum'}
    <select
      class="rounded border border-line bg-field px-1 py-1"
      value={current as string}
      onchange={(e) => onchange(e.currentTarget.value)}
    >
      {#each param.options ?? [] as option}
        <option value={option}>{option}</option>
      {/each}
    </select>
  {:else if param.kind === 'color'}
    <div class="flex items-center gap-2">
      <input
        type="color"
        class="h-7 w-10 cursor-pointer rounded border border-line bg-field"
        value={current as string}
        oninput={(e) => onchange(e.currentTarget.value)}
      />
      <span class="font-mono text-xs text-fg-muted">{current}</span>
    </div>
  {:else if param.kind === 'json' && param.name.endsWith('stops')}
    <GradientEditor stops={current as Stop[]} onchange={(s) => onchange(s)} />
  {:else if param.kind === 'json' && param.name === 'strips'}
    <TrimEditor strips={current as Strip[]} size={editor.size} onchange={(s) => onchange(s)} />
  {:else if param.kind === 'json'}
    <textarea
      class="min-h-20 rounded border bg-field p-1 font-mono text-xs {jsonError ? 'border-danger' : 'border-line'}"
      bind:value={jsonText}
      onchange={() => setJSON(jsonText)}
    ></textarea>
    {#if jsonError}<span class="text-[11px] text-danger">{jsonError}</span>{/if}
  {:else}
    <input
      type="text"
      class="rounded border border-line bg-field px-1 py-1 font-mono text-xs"
      value={current as string}
      onchange={(e) => onchange(e.currentTarget.value)}
    />
  {/if}
</div>
