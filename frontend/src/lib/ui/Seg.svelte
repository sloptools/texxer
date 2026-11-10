<script lang="ts" generics="T">
  let { value, options, onchange }: { value: T; options: [T, string][]; onchange: (v: T) => void } = $props()
</script>

<div class="seg" role="radiogroup">
  {#each options as [v, label]}
    <button class:on={value === v} role="radio" aria-checked={value === v} onclick={() => onchange(v)}>{label}</button>
  {/each}
</div>

<style>
  .seg {
    display: flex;
    padding: 2px;
    border: 1px solid var(--line);
    border-radius: 7px;
    background: var(--bg);
  }
  button {
    flex: 1;
    height: 22px;
    padding: 0 8px;
    white-space: nowrap;
    border: 0;
    border-radius: 5px;
    background: none;
    font: 12px var(--mono);
    color: var(--dim);
  }
  button:hover {
    color: var(--strong);
  }
  button.on {
    background: var(--active);
    color: var(--strong);
  }
</style>
