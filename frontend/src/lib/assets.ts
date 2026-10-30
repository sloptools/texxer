// Loads files from the app's asset server a few at a time and retries, since
// bursts of parallel requests in the webview sometimes fail.
const limit = 3
let running = 0
const waiting: (() => void)[] = []

async function slot<T>(fn: () => Promise<T>): Promise<T> {
  if (running >= limit) await new Promise<void>((resolve) => waiting.push(resolve))
  running++
  try {
    return await fn()
  } finally {
    running--
    waiting.shift()?.()
  }
}

// Resolves to undefined when the file doesn't exist.
export async function fetchBlob(url: string, tries = 4): Promise<Blob | undefined> {
  for (let i = 0; i < tries; i++) {
    try {
      const res = await slot(() => fetch(url))
      if (res.status === 404) return
      if (res.ok) return await res.blob()
    } catch {}
    await new Promise((resolve) => setTimeout(resolve, 150 * 2 ** i))
  }
}

const urls = new Map<string, Promise<string | undefined>>()

// A blob URL for an image, shared by everyone asking for the same file.
export function imageURL(url: string): Promise<string | undefined> {
  let p = urls.get(url)
  if (!p) {
    p = fetchBlob(url).then((b) => b && URL.createObjectURL(b))
    urls.set(url, p)
  }
  return p
}
