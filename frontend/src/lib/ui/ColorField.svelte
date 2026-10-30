<script lang="ts">
  let { value = $bindable(), label }: { value: string; label?: string } = $props()
</script>

<div class="field">
  {#if label}<span class="label">{label}</span>{/if}
  <label class="pick" style:background={value}>
    <input type="color" bind:value />
  </label>
  <input class="hex" {value} onchange={(e) => /^#[0-9a-f]{6}$/i.test(e.currentTarget.value) && (value = e.currentTarget.value)} />
</div>

<style>
  .field {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .label {
    flex: none;
    width: 52px;
    font-size: 12px;
    color: var(--dim);
  }
  .pick {
    position: relative;
    flex: none;
    width: 44px;
    height: 26px;
    border-radius: 6px;
    box-shadow: inset 0 0 0 1px rgb(0 0 0 / 0.25);
  }
  .pick input {
    position: absolute;
    inset: 0;
    opacity: 0;
  }
  .hex {
    flex: 1;
    min-width: 0;
    height: 26px;
    padding: 0 8px;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--bg);
    font: 12px var(--mono);
    color: var(--fg);
    outline: none;
  }
  .hex:focus {
    border-color: var(--dim);
  }
</style>
