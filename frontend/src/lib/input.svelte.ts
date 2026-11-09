// Mouse, trackpad and modifier keys, the same on macOS and Windows.

export const isMac = /Mac/.test(navigator.platform)

// ⌘ on a Mac, ctrl elsewhere.
export const cmd = (e: { metaKey: boolean; ctrlKey: boolean }) => e.metaKey || (!isMac && e.ctrlKey)

export type WheelMode = 'auto' | 'trackpad' | 'mouse'

function stored(): WheelMode {
  try {
    const v = localStorage.getItem('texxer.wheel')
    if (v === 'trackpad' || v === 'mouse') return v
  } catch {}
  return 'auto'
}

export const input = $state({ wheel: stored(), space: false })

export function setWheel(mode: WheelMode) {
  input.wheel = mode
  try {
    localStorage.setItem('texxer.wheel', mode)
  } catch {}
}

// Space held turns any drag on the views into the camera's.
// It never presses a button.
addEventListener('keydown', (e) => {
  if (e.key !== ' ' || (e.target as HTMLElement).closest?.('input, textarea')) return
  e.preventDefault()
  input.space = true
})
addEventListener('keyup', (e) => {
  if (e.key !== ' ') return
  if (!(e.target as HTMLElement).closest?.('input, textarea')) e.preventDefault()
  input.space = false
})
addEventListener('blur', () => (input.space = false))

// A mouse wheel clicks in steps (wheelDelta in 120s, or lines); a trackpad scrolls in
// pixels, its wheelDelta three times those. The setting overrides the guess.
export function isMouseWheel(e: WheelEvent): boolean {
  if (input.wheel !== 'auto') return input.wheel === 'mouse'
  if (e.deltaMode !== 0) return true
  const step = (e as WheelEvent & { wheelDeltaY?: number }).wheelDeltaY
  return e.deltaX === 0 && !!step && step % 120 === 0 && step !== -3 * e.deltaY
}
