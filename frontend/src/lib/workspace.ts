import { normalize, uid } from './project'
import type { ModelDoc, Project, Workspace } from './types'

const shared = ['finish', 'shading', 'light'] as const

export const baseName = (path: string) => slash(path).split('/').pop()!.replace(/\.[^.]+$/, '')
const dirName = (path: string) => path.slice(0, path.lastIndexOf('/'))

// Windows paths too, as C:/dir/file.
export const slash = (path: string) => path.replace(/\\/g, '/')

// Projects saved by an older texxer on Windows hold "../C:\dir\file".
const drive = (path: string) => path.match(/(?:^|\/)([A-Za-z]:\/.*)/)?.[1]

export function relative(fromDir: string, to: string): string {
  const a = slash(fromDir).split('/').filter(Boolean)
  const b = slash(to).split('/').filter(Boolean)
  if (a[0] !== b[0] && (drive(slash(fromDir)) || drive(slash(to)))) return slash(to)
  let i = 0
  while (i < a.length && i < b.length && a[i] === b[i]) i++
  return [...a.slice(i).map(() => '..'), ...b.slice(i)].join('/') || '.'
}

export function resolve(dir: string, path: string): string {
  path = slash(path)
  const abs = drive(path)
  if (abs) return abs
  if (path.startsWith('/')) return path
  const out = slash(dir).split('/').filter(Boolean)
  for (const part of path.split('/')) {
    if (part === '..') out.pop()
    else if (part && part !== '.') out.push(part)
  }
  return (drive(slash(dir)) ? '' : '/') + out.join('/')
}

// The key the backend knows a model by: its files joined by "|".
export const modelKey = (m: ModelDoc) => m.files.join('|')

export function newModel(files: string[], from: Project): ModelDoc {
  const p = normalize(JSON.parse(JSON.stringify(from)))
  return { ...p, id: uid(), name: baseName(files[0]), files }
}

// Reads a project file. An old one belongs to the model beside it and becomes a
// project with that one model.
export function parseWorkspace(path: string, json: string, model: string): Workspace {
  const dir = dirName(path)
  const data = json ? JSON.parse(json) : {}
  if (!data.models) {
    const p = normalize(data)
    const m: ModelDoc = { ...p, id: uid(), name: baseName(model || path), files: model ? [model] : [] }
    return link({ version: 2, finish: p.finish, shading: p.shading, light: p.light, models: model ? [m] : [] })
  }

  const base = normalize({ finish: data.finish, shading: data.shading, light: data.light })
  const models = data.models.map((m: ModelDoc) => ({
    ...normalize({ ...m, finish: base.finish, shading: base.shading, light: base.light }),
    id: m.id,
    name: m.name.includes('\\') ? baseName(m.name) : m.name,
    files: m.files.map((f) => resolve(dir, f)),
    textures: m.textures && resolve(dir, m.textures),
    blend: m.blend && resolve(dir, m.blend),
    target: m.target && resolve(dir, m.target),
    exported: m.exported,
  }))
  return link({ version: 2, finish: base.finish, shading: base.shading, light: base.light, models })
}

// Every model points at the project's shared settings.
export function link(ws: Workspace): Workspace {
  for (const m of ws.models) for (const k of shared) (m as any)[k] = ws[k]
  return ws
}

export function serializeWorkspace(path: string, ws: Workspace): string {
  const dir = dirName(path)
  const models = ws.models.map((m) => {
    const { finish, shading, light, ...rest } = m
    const path = (f?: string) => f && relative(dir, f)
    return { ...rest, files: m.files.map((f) => relative(dir, f)), textures: path(m.textures), blend: path(m.blend), target: path(m.target) }
  })
  return JSON.stringify({ version: 2, finish: ws.finish, shading: ws.shading, light: ws.light, models }, null, 2)
}

// Changes whenever anything that ends up in the export does. Keys are sorted, as
// reading a project back can order them differently.
export function docHash(m: ModelDoc): string {
  const json = JSON.stringify(m, (k, v) => {
    if (k === 'exported' || k === 'id' || k === 'files') return undefined
    if (!v || typeof v !== 'object' || Array.isArray(v)) return v
    return Object.fromEntries(Object.keys(v).sort().map((key) => [key, v[key]]))
  })
  let h = 2166136261
  for (let i = 0; i < json.length; i++) h = Math.imul(h ^ json.charCodeAt(i), 16777619)
  return (h >>> 0).toString(16)
}

export type ModelState = 'clean' | 'edited' | 'changed' | 'missing' | 'new'

// What a model needs: nothing, an export after edits, an export since its files changed.
export function modelState(m: ModelDoc, fileHash: string | undefined): ModelState {
  if (fileHash === '') return 'missing'
  if (!m.exported) return 'new'
  if (fileHash !== undefined && fileHash !== m.exported.files) return 'changed'
  return docHash(m) === m.exported.doc ? 'clean' : 'edited'
}
