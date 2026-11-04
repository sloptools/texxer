import * as api from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { openProject, refreshHashes, selectModel } from './projects.svelte'
import { store } from './store.svelte'
import type { ModelDoc } from './types'
import { link, modelKey, newModel } from './workspace'
import { newProject } from './project'

type Sync = { blend: string; glb: string; name: string }

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

    if (store.project.id === m.id && store.model) {
      await store.paint?.flush()
      store.model = await api.Open(modelKey(m))
    } else await selectModel(m.id)
    refreshHashes()
    store.say('synced from blender')
  }

  // Asks Blender for the objects as they are now.
  requestSync() {
    const m = this.model
    if (m?.blend) api.BridgeRequestSync(m.blend)
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
