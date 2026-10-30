import * as THREE from 'three'
import { ShaderPass } from 'three/examples/jsm/postprocessing/ShaderPass.js'

// Substratum's world: a flat teal ambient, SSAO, ACES, then brightness,
// contrast and saturation adjustments (scenes/world/world.tscn).
export const game = {
  exposure: 1.1,
  brightness: 0.7,
  contrast: 1.06,
  saturation: 1.38,
  background: '#0b1c21',
}

// Lit from baked maps, so it matches the game's flat ambient: AO stands in for SSAO,
// and an optional light from above for the hangar spot.
export const gamePatch = /* glsl */ `
if (uGame == 1) {
  float ao = mix(1.0, texture(uGeoMap, vMapUv).r, uAoStrength);
  vec3 n = texture(uNrmMap, vMapUv).xyz;
  diffuseColor.rgb *= (uAmbient + uTop * max(n.y, 0.0)) * ao;
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

// Godot's adjustments, applied to display colors after tone mapping.
export function gradePass() {
  return new ShaderPass({
    uniforms: {
      tDiffuse: { value: null },
      uBrightness: { value: game.brightness },
      uContrast: { value: game.contrast },
      uSaturation: { value: game.saturation },
    },
    vertexShader: /* glsl */ `
      varying vec2 vUv;
      void main() {
        vUv = uv;
        gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
      }
    `,
    fragmentShader: /* glsl */ `
      uniform sampler2D tDiffuse;
      uniform float uBrightness;
      uniform float uContrast;
      uniform float uSaturation;
      varying vec2 vUv;
      void main() {
        vec4 c = texture2D(tDiffuse, vUv);
        vec3 col = c.rgb * uBrightness;
        col = mix(vec3(0.5), col, uContrast);
        col = mix(vec3(dot(col, vec3(1.0)) / 3.0), col, uSaturation);
        gl_FragColor = vec4(col, c.a);
      }
    `,
  })
}
