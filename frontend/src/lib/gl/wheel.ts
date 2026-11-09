// A mouse wheel clicks in steps (wheelDelta in 120s, or lines); a trackpad scrolls in
// pixels, its wheelDelta three times those. The wheel zooms, as everywhere else; two
// fingers keep orbiting and panning.
export function isMouseWheel(e: WheelEvent): boolean {
  if (e.deltaMode !== 0) return true
  const step = (e as WheelEvent & { wheelDeltaY?: number }).wheelDeltaY
  return e.deltaX === 0 && !!step && step % 120 === 0 && step !== -3 * e.deltaY
}
