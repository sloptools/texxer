import { bridge } from './bridge.svelte'
import { store } from './store.svelte'

export type UVOption = { label: string; title: string; on: boolean; pick: () => void }

// Where the model's UVs come from: its own or texxer's, for Blender models also kept as they are.
export function uvOptions(): UVOption[] {
  const p = store.project
  if (p.blend)
    return [
      { label: 'blender', title: "Paint on Blender's own UVs", on: !p.blendUV, pick: () => bridge.setUV(undefined) },
      { label: 'texxer', title: "texxer lays the UVs out on each sync and writes them into Blender, as the UV map 'texxer'", on: p.blendUV === 'texxer', pick: () => bridge.setUV('texxer') },
      { label: 'locked', title: "Keep texxer's UVs in Blender as they are, so painted masks stay put", on: p.blendUV === 'locked', pick: () => bridge.setUV('locked') },
    ]
  return [
    { label: 'model', title: "Paint on the model's own UVs", on: !p.unwrap, pick: () => store.setLayout(0) },
    { label: 'texxer', title: 'Let texxer lay the UVs out: cut at hard edges, upright, one texel density', on: !!p.unwrap, pick: () => store.setLayout(p.unwrap || p.size) },
  ]
}
