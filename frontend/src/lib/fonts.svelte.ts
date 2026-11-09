import { Fonts } from '../../wailsjs/go/main/App'

// Fonts for text stamps: built in, the user's from the font folder, and the
// system's that are there (Chinese and Japanese ones mostly).
export type Font = { name: string; url: string | null }

const builtIn: Font[] = [
  ['Black Ops One', 'BlackOpsOne'],
  ['Big Shoulders Stencil', 'BigShouldersStencil'],
  ['Bebas Neue', 'BebasNeue'],
  ['Anton', 'Anton'],
  ['Archivo Black', 'ArchivoBlack'],
  ['Silkscreen', 'Silkscreen'],
  ['Fusion Pixel', 'FusionPixel12'],
  ['ZCOOL QingKe HuangYou', 'ZCOOLQingKeHuangYou'],
  ['Dela Gothic One', 'DelaGothicOne'],
].map(([name, file]) => ({ name, url: `/fonts/${file}.woff2` }))

const system = ['PingFang HK', 'PingFang SC', 'Hiragino Sans', 'Microsoft JhengHei', 'Microsoft YaHei', 'Yu Gothic', 'Impact', 'Arial Black']

export const fonts = $state({ builtIn, own: [] as Font[], system: [] as Font[] })

// Tells the stamp atlas to draw a font's stamps again once it can.
export const fontLoaded = new EventTarget()

// A system font is there when text in it measures unlike both fallbacks.
function installed(name: string) {
  const ctx = document.createElement('canvas').getContext('2d')!
  const width = (font: string) => {
    ctx.font = `40px ${font}`
    return ctx.measureText('mmWl 永和 あア').width
  }
  return ['monospace', 'serif'].every((fallback) => width(`"${name}", ${fallback}`) !== width(fallback))
}

let listed: Promise<void> | undefined

export function loadFonts(): Promise<void> {
  listed = (async () => {
    fonts.system = system.filter(installed).map((name) => ({ name, url: null }))
    try {
      const files = await Fonts()
      fonts.own = files.map((f) => ({ name: f.replace(/\.\w+$/, ''), url: `/font?file=${encodeURIComponent(f)}` }))
    } catch {
      fonts.own = []
    }
  })()
  return listed
}

const loading = new Map<string, Promise<void>>()

// Resolves once the font can draw; missing fonts fall back to the system's.
export function ready(name: string): Promise<void> {
  let p = loading.get(name)
  if (p) return p
  p = (async () => {
    await (listed ?? loadFonts())
    const font = [...builtIn, ...fonts.own].find((f) => f.name === name)
    if (!font?.url) return
    const face = new FontFace(name, `url(${font.url})`)
    try {
      document.fonts.add(await face.load())
      fontLoaded.dispatchEvent(new CustomEvent('load', { detail: name }))
    } catch {}
  })()
  loading.set(name, p)
  return p
}
