import { marked } from 'marked'

// The docs are the markdown files in docs/, built into the app with their screenshots.
const files = import.meta.glob('../../../docs/*.md', { query: '?raw', import: 'default', eager: true }) as Record<string, string>
const images = import.meta.glob('../../../docs/img/*.png', { query: '?url', import: 'default', eager: true }) as Record<string, string>

export type Page = { id: string; title: string; html: string }

function render(md: string): string {
  // Decoded right away: WebKit can leave lazily decoded images blank in the scrolling page.
  return (marked.parse(md, { async: false }) as string).replace(
    /src="img\/([^"]+)"/g,
    (_, name) => `loading="eager" decoding="sync" src="${images['../../../docs/img/' + name] ?? ''}"`,
  )
}

export const pages: Page[] = Object.keys(files)
  .sort()
  .map((path) => {
    const md = files[path]
    const id = path.split('/').pop()!.replace(/^\d+-|\.md$/g, '')
    return { id, title: md.match(/^# (.+)$/m)?.[1] ?? id, html: render(md) }
  })
