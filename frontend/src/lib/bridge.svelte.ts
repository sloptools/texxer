import * as api from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { openProject, refreshHashes, selectModel } from './projects.svelte'
import { store } from './store.svelte'
import type { ModelDoc } from './types'
import { link, modelKey, newModel } from './workspace'
import { newProject } from './project'

type Sync = { blend: string; glb: string; name: string; prev: string }

const dirName = (path: string) => path.slice(0, path.lastIndexOf('/'))

function hash(s: string): string {
  let h = 2166136261
  for (let i = 0; i < s.length; i++) h = Math.imul(h ^ s.charCodeAt(i), 16777619)
  return (h >>> 0).toString(16)
}

// Models from Blender: the add-on sends a .blend's objects as a GLB, texxer sends
// the texture back as PNGs where the export puts them.
class Bridge {
  // The .blend files whose add-on is listening.
  connected = $state<string[]>([])
  private timer = 0
  private sent = ''
  private layout = ''

  start() {
    EventsOn('bridge:sync', (s: Sync) => this.synced(s))
    setInterval(() => this.poll(), 1500)
  }

  get model(): ModelDoc | undefined {
    return store.project.blend ? store.project : undefined
  }

  isConnected(m: ModelDoc) {
    return !!m.blend && this.connected.includes(m.blend)
  }

  private async poll() {
    const before = this.connected
    this.connected = await api.BridgeConnected()
    const m = this.model
    if (m) await this.tellLayout(m)
    // A .blend that just started listening gets the texture as it is.
    if (m && this.isConnected(m) && !before.includes(m.blend!)) this.changed(true)
  }

  // Blender sent new geometry: into the model of that .blend, or a new one.
  private async synced(s: Sync) {
    if (!store.workspace) await openProject(dirName(s.blend) + '/' + s.name + '.texxer')
    const ws = store.workspace
    if (!ws) return

    let m = ws.models.find((m) => m.blend === s.blend || m.files.includes(s.glb))
    if (!m) {
      m = { ...newModel([s.glb], newProject()), name: s.name, blend: s.blend, textures: dirName(s.blend) + '/textures' }
      ws.models.push(m)
      link(ws)
    }

    if (store.project.id !== m.id) await selectModel(m.id)
    await store.paint?.flush()
    // Painted masks follow the surface onto the new geometry and UVs.
    if (s.prev) {
      try {
        await api.MovePaint(store.path, s.prev, 0, modelKey(m), 0, store.paintIds())
      } catch (e) {
        store.say(String(e))
      }
    }
    store.model = await api.Open(modelKey(m))
    refreshHashes()
    store.say('synced from blender')
  }

  // The add-on reads where the UVs come from on its next sync.
  private async tellLayout(m: ModelDoc) {
    const layout = `${m.blend}|${m.blendUV ?? 'blender'}|${m.size}`
    if (layout === this.layout) return
    this.layout = layout
    await api.BridgeLayout(m.blend!, m.blendUV ?? 'blender', m.size)
  }

  // Asks Blender for the objects as they are now.
  async requestSync() {
    const m = this.model
    if (!m?.blend) return
    await this.tellLayout(m)
    api.BridgeRequestSync(m.blend)
  }

  // Where the UVs come from; texxer's need a sync to be laid out.
  async setUV(uv: ModelDoc['blendUV']) {
    const m = this.model
    if (!m) return
    m.blendUV = uv
    if (uv === 'texxer') await this.requestSync()
  }

  // The texture changed: once it rests, Blender gets it.
  changed(now = false) {
    const m = this.model
    if (!m || !this.isConnected(m)) return
    clearTimeout(this.timer)
    this.timer = window.setTimeout(() => this.send(), now ? 0 : 600)
  }

  private async send() {
    const m = this.model
    if (!m?.blend || !store.snapshot || !store.ready) return
    const shot = store.snapshot('')
    // Renders that change nothing don't write the files again.
    const sig = hash(shot.color) + ':' + hash(shot.emissive) + ':' + m.finish.glow
    if (sig === this.sent) return
    this.sent = sig
    try {
      await api.BridgeTexture(m.blend, m.textures ?? dirName(m.blend) + '/textures', m.name, shot.color, shot.emissive, m.finish.glow)
    } catch (e) {
      store.say(String(e))
    }
  }
}

export const bridge = new Bridge()
