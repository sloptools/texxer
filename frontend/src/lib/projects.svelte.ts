import * as api from '../../wailsjs/go/main/App'
import { loading } from './assets'
import { newProject } from './project'
import { load, persist, store, type History } from './store.svelte'
import type { ModelDoc } from './types'
import { baseName, docHash, link, modelKey, modelState, newModel, parseWorkspace, type ModelState } from './workspace'

// Open, switch and export the models of a project. The store holds the state.

const histories = new Map<string, History>()
// Models added this session; they lay their UVs out anew when their own overlap.
const fresh = new Set<string>()

export const recent = $state<{ list: string[] }>({ list: load('recent', []) })

function remember(path: string) {
  recent.list = [path, ...recent.list.filter((p) => p !== path)].slice(0, 8)
  persist('recent', recent.list)
}

function blank(): ModelDoc {
  return { ...newProject(), id: '', name: '', files: [] }
}

// Opens a project, or a model with the project beside it.
export async function openProject(path: string) {
  try {
    const f = await api.LoadProject(path)
    await store.paint?.flush()
    const ws = parseWorkspace(f.path, f.json, f.model)
    if (!f.json) for (const m of ws.models) fresh.add(m.id)
    histories.clear()
    store.path = f.path
    store.workspace = ws
    store.model = null
    store.project = blank()
    remember(f.path)

    const first = ws.models.find((m) => f.model && m.files.includes(f.model)) ?? ws.models[0]
    if (first) await selectModel(first.id)
    refreshHashes()
  } catch (e) {
    store.say(String(e))
  }
}

export async function pickProject() {
  const path = await api.PickProject()
  if (path) await openProject(path)
}

export async function createProject() {
  const path = await api.NewProject()
  if (!path) return
  await store.paint?.flush()
  const base = newProject()
  histories.clear()
  store.workspace = link({ version: 2, finish: base.finish, shading: base.shading, light: base.light, models: [] })
  store.path = path
  store.model = null
  store.project = blank()
  remember(path)
  await addModels()
}

// Adds models to the project, each file a model of its own.
export async function addModels(files?: string[]) {
  const ws = store.workspace
  if (!ws) {
    if (files?.length) await openProject(files[0])
    return
  }
  files ??= (await api.PickModels()) ?? []
  const known = new Set(ws.models.flatMap((m) => m.files))
  const added = files.filter((f) => !known.has(f)).map((f) => newModel([f], newProject()))
  if (!added.length) return
  for (const m of added) fresh.add(m.id)
  ws.models.push(...added)
  link(ws)
  await selectModel(added[0].id)
  refreshHashes()
}

// Adds files to a model, so they share its texture. Their charts join the layout,
// so the model takes texxer's UVs, and its painted masks move along.
export async function addFiles(id: string, picked?: string[]) {
  const m = store.workspace?.models.find((m) => m.id === id)
  if (!m) return
  const files = (picked ?? (await api.PickModels()) ?? []).filter((f) => !m.files.includes(f))
  if (!files.length) return

  await selectModel(id)
  await store.paint?.flush()
  const from = modelKey(m)
  const layout = m.unwrap
  m.files = [...m.files, ...files]
  m.unwrap = m.unwrap || m.size
  try {
    await api.MovePaint(store.path, from, layout, modelKey(m), m.unwrap, store.paintIds())
    store.model = await api.Open(modelKey(m))
  } catch (e) {
    store.say(String(e))
  }
  refreshHashes()
}

// Picks the folder the export puts the model's PNGs into; cancelling puts them beside its files again.
export async function pickTextureFolder(id: string) {
  const m = store.workspace?.models.find((m) => m.id === id)
  if (!m) return
  const dir = await api.PickFolder()
  if (dir) m.textures = dir
  else delete m.textures
}

export function removeModel(id: string) {
  const ws = store.workspace
  if (!ws) return
  ws.models = ws.models.filter((m) => m.id !== id)
  histories.delete(id)
  if (store.project.id !== id) return
  store.model = null
  store.project = blank()
  if (ws.models[0]) selectModel(ws.models[0].id)
}

export async function selectModel(id: string) {
  const m = store.workspace?.models.find((m) => m.id === id)
  if (!m || (store.project.id === id && store.model)) return
  try {
    const info = await api.Open(modelKey(m))
    await store.paint?.flush()
    if (store.project.id) histories.set(store.project.id, store.takeHistory())
    if (fresh.has(m.id) && info.overlap > 0.01 && !m.unwrap) m.unwrap = m.size
    fresh.delete(m.id)

    store.ready = false
    store.project = m
    store.model = info
    store.view = m.shading.on ? 'unlit' : 'lit'
    store.selected = m.layers.length - 1
    store.variant = ''
    store.giveHistory(histories.get(id))
  } catch (e) {
    store.say(String(e))
  }
}

// Steps through the models, for a key.
export function cycleModel(step: number) {
  const list = store.workspace?.models ?? []
  const at = list.findIndex((m) => m.id === store.project.id)
  if (list.length) selectModel(list[(at + step + list.length) % list.length].id)
}

export async function refreshHashes() {
  const ws = store.workspace
  if (!ws) return
  store.fileHashes = await api.FileHashes(ws.models.map(modelKey))
}

export function stateOf(m: ModelDoc): ModelState {
  return modelState(m, store.fileHashes[modelKey(m)])
}

// Writes the model being painted: its textures into its files, PNGs beside them.
export async function exportModel(): Promise<boolean> {
  const m = store.project
  const model = store.model
  if (!model || !store.snapshot) return false
  try {
    const shot = store.snapshot('')
    const variants = m.variants.map((v) => {
      const s = store.snapshot!(v.id)
      return { name: v.name, color: s.color, emissive: s.emissive }
    })
    // A Blender model's file is the add-on's copy; its export goes to a GLB of its own.
    if (m.blend && !m.target) m.target = (await api.PickExportTarget(m.name)) || undefined
    if (m.blend && !m.target) return false
    const files = await api.Export(model.path, m.unwrap, m.name, m.textures ?? '', m.target ?? '', shot.color, shot.emissive, variants, m.finish.glow)
    m.exported = { doc: docHash(m), files }
    store.fileHashes = { ...store.fileHashes, [model.path]: files }
    store.say('exported ' + m.name)
    return true
  } catch (e) {
    store.say(String(e))
    return false
  }
}

async function until(ok: () => boolean, ms = 120000) {
  const end = Date.now() + ms
  while (!ok() && Date.now() < end) await new Promise((r) => setTimeout(r, 50))
  return ok()
}

// Exports every model that was edited or whose files changed since its last export.
export async function exportChanged() {
  const ws = store.workspace
  if (!ws) return
  await refreshHashes()
  const todo = ws.models.filter((m) => ['edited', 'changed', 'new'].includes(stateOf(m)))
  if (!todo.length) {
    store.say('everything is exported')
    return
  }

  const back = store.project.id
  let done = 0
  for (const m of todo) {
    await selectModel(m.id)
    if (!(await until(() => store.ready))) continue
    // The first render asks for grunge maps, stamps and painted masks; wait for them.
    await new Promise((r) => setTimeout(r, 300))
    if ((await until(() => loading.count === 0)) && (await exportModel())) done++
  }
  if (back) await selectModel(back)
  store.say(`exported ${done} of ${todo.length}: ${todo.map((m) => m.name).join(', ')}`)
}

export function projectName() {
  return store.path ? baseName(store.path) : ''
}
