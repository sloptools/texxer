import * as THREE from 'three'
import { torchPatch } from './torch'

// Lit from baked maps like a game world with a flat ambient: AO stands in for SSAO,
// plus an optional light from above, and the torch.
export const gamePatch = /* glsl */ `
if (uGame == 1) {
  float ao = mix(1.0, texture(uGeoMap, vMapUv).r, uAoStrength);
  vec3 n = texture(uNrmMap, vMapUv).xyz;
  vec3 light = (uAmbient + uTop * max(n.y, 0.0)) * ao;
  ${torchPatch}
  diffuseColor.rgb *= light;
}
`

export function gameUniforms() {
  return {
    uGame: { value: 0 },
    uGeoMap: { value: null as THREE.Texture | null },
    uNrmMap: { value: null as THREE.Texture | null },
    uAmbient: { value: new THREE.Color() },
    uTop: { value: new THREE.Color() },
    uAoStrength: { value: 0.7 },
  }
}

export const gameDeclarations = `uniform int uGame;
uniform sampler2D uGeoMap;
uniform sampler2D uNrmMap;
uniform vec3 uAmbient;
uniform vec3 uTop;
uniform float uAoStrength;
`
