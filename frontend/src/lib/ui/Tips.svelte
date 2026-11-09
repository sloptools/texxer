<script lang="ts">
  // Shows title attributes as quick tips in the app's look instead of the system's
  // slow ones; a key in parentheses at the end shows as a key.

  let tip = $state<{ text: string; key: string; x: number; y: number; below: boolean } | null>(null)
  let el = $state<HTMLDivElement>()
  let target: HTMLElement | null = null
  let timer = 0

  function split(title: string): { text: string; key: string } {
    const m = title.match(/^(.*?)\s*\(([^()]{1,12})\)$/)
    return m ? { text: m[1], key: m[2] } : { text: title, key: '' }
  }

  function over(e: PointerEvent) {
    const t = (e.target as HTMLElement).closest?.<HTMLElement>('[title], [data-tip]')
    if (t === target) return
    restore()
    clearTimeout(timer)
    if (!t) {
      timer = window.setTimeout(() => (tip = null), 150)
      return
    }
    // The title moves aside while hovered, so the system tip stays away.
    target = t
    if (t.title) {
      t.dataset.tip = t.title
      t.removeAttribute('title')
    }
    const show = () => {
      if (target !== t || !t.isConnected) return
      const r = t.getBoundingClientRect()
      const below = r.bottom + 40 < innerHeight
      tip = { ...split(t.dataset.tip ?? ''), x: r.left + r.width / 2, y: below ? r.bottom + 6 : r.top - 6, below }
    }
    if (tip) show()
    else timer = window.setTimeout(show, 450)
  }

  function restore() {
    if (target?.dataset.tip) {
      target.title = target.dataset.tip
      delete target.dataset.tip
    }
    target = null
  }

  function hide() {
    clearTimeout(timer)
    restore()
    tip = null
  }

  // Kept inside the window.
  $effect(() => {
    if (!tip || !el) return
    const w = el.offsetWidth
    el.style.left = Math.min(Math.max(tip.x - w / 2, 6), innerWidth - w - 6) + 'px'
  })
</script>

<svelte:window onpointerover={over} onpointerdown={hide} onkeydown={hide} onblur={hide} />

{#if tip?.text}
  <div class="tip" class:above={!tip.below} style:top="{tip.y}px" bind:this={el}>
    {tip.text}{#if tip.key}<kbd>{tip.key}</kbd>{/if}
  </div>
{/if}

<style>
  .tip {
    position: fixed;
    z-index: 100;
    display: flex;
    gap: 8px;
    align-items: baseline;
    max-width: 320px;
    padding: 5px 8px;
    border: 1px solid var(--line);
    border-radius: 6px;
    background: var(--panel);
    font: 11px/1.4 var(--mono);
    color: var(--strong);
    pointer-events: none;
  }
  .tip.above {
    transform: translateY(-100%);
  }
  kbd {
    flex: none;
    color: var(--faint);
  }
</style>
