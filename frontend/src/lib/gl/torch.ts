import * as THREE from 'three'

const angle = THREE.MathUtils.degToRad(11)

// A flashlight from the camera that points where the cursor is, to see how the
// texture holds up under the game's moving lights. The lit view uses a real
// spot light; the game view lights from baked maps and gets the same values
// as uniforms.
export class Torch {
  readonly light = new THREE.SpotLight(0xffe6c4, 0, 0, angle, 0.45, 2)
  readonly uniforms = {
    uTorch: { value: new THREE.Color(0, 0, 0) },
    uTorchPos: { value: new THREE.Vector3() },
    uTorchDir: { value: new THREE.Vector3(0, 0, -1) },
    uTorchCos: { value: new THREE.Vector2(Math.cos(angle), Math.cos(angle * 0.6)) },
    uMin: { value: new THREE.Vector3() },
    uDiag: { value: 1 },
  }
  private aim: THREE.Vector3 | undefined
  private strength = 0

  set(on: boolean, strength: number) {
    this.strength = on ? strength : 0
  }

  aimAt(point: THREE.Vector3 | undefined) {
    this.aim = point?.clone()
  }

  // Sits a little below and right of the eye, like a held lamp.
  update(camera: THREE.PerspectiveCamera, radius: number) {
    const pos = new THREE.Vector3(0.25, -0.2, 0).multiplyScalar(radius).applyQuaternion(camera.quaternion).add(camera.position)
    const forward = new THREE.Vector3(0, 0, -1).applyQuaternion(camera.quaternion)
    const target = this.aim ?? camera.position.clone().addScaledVector(forward, radius * 3)

    this.light.position.copy(pos)
    this.light.target.position.copy(target)
    this.light.intensity = this.strength * radius * radius * 25

    const u = this.uniforms
    u.uTorchPos.value.copy(pos)
    u.uTorchDir.value.copy(target).sub(pos).normalize()
    u.uTorch.value.set(0xffe6c4).multiplyScalar(this.strength * radius * radius * 6)
  }
}

// Added to the game view's light; positions come from the baked position map.
export const torchPatch = /* glsl */ `
  vec3 wp = texture(uPosMap, vMapUv).xyz * uDiag + uMin;
  vec3 tl = uTorchPos - wp;
  float td = length(tl);
  tl /= td;
  float cone = smoothstep(uTorchCos.x, uTorchCos.y, dot(-tl, uTorchDir));
  light += uTorch * max(dot(n, tl), 0.0) * cone / (td * td);
`

export const torchDeclarations = `uniform vec3 uTorch;
uniform vec3 uTorchPos;
uniform vec3 uTorchDir;
uniform vec2 uTorchCos;
uniform vec3 uMin;
uniform float uDiag;
`
