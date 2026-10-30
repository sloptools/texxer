import * as THREE from 'three';
import type { Pass } from 'three/examples/jsm/postprocessing/Pass.js';
import { ShaderPass } from 'three/examples/jsm/postprocessing/ShaderPass.js';

/**
 * A preview scene modelled on Substratum's hangar: teal sky light and a dim
 * floor bounce, a spot from above with a visible beam and soft shadows on
 * the floor, fog, drifting particles, and the game's color grade.
 */
export class Stage {
  readonly group = new THREE.Group();
  readonly grade: Pass;

  #spot: THREE.SpotLight;
  #floor: THREE.Mesh;
  #beam: THREE.Mesh;
  #dust: THREE.Points;
  #fog = new THREE.FogExp2(0x0d2a36, 0.06);
  #bg: THREE.Texture;

  constructor() {
    // three.js lights are physical: ambient and directional intensities read about π times
    // weaker than Godot's energies, so these are higher than the hangar's numbers.
    // Sky above, a darker teal bounce from below, so undersides keep some color.
    this.group.add(new THREE.HemisphereLight(0x7fb4c4, 0x5a7470, 3.0));
    this.group.add(new THREE.AmbientLight(0x6a8a90, 1.0));

    // Light bouncing off the seafloor, so the underside reads instead of going black. It stays
    // close to neutral: a pure teal has no red in linear light and turns rust black.
    const bounce = new THREE.DirectionalLight(0xa0c4bc, 2.0);
    bounce.position.set(0.3, -1, 0.4);
    this.group.add(bounce, bounce.target);

    this.#spot = new THREE.SpotLight(0xd8f0ff, 22, 0, Math.PI / 6, 0.7, 1.4);
    this.#spot.castShadow = true;
    this.#spot.shadow.mapSize.set(1024, 1024);
    this.#spot.shadow.radius = 6;
    this.#spot.shadow.bias = -0.0005;
    this.group.add(this.#spot, this.#spot.target);

    // A faint rim from behind, like the hangar's back light.
    const rim = new THREE.DirectionalLight(0x3fc4d8, 1.6);
    rim.position.set(-1, 0.5, -1.5);
    this.group.add(rim);

    this.#floor = new THREE.Mesh(new THREE.CircleGeometry(1, 48), new THREE.ShadowMaterial({ opacity: 0.55, color: 0x020a0e }));
    this.#floor.rotation.x = -Math.PI / 2;
    this.#floor.receiveShadow = true;
    this.group.add(this.#floor);

    // The beam: an open cone, brightest at the top, added on top of the scene.
    this.#beam = new THREE.Mesh(
      new THREE.ConeGeometry(1, 1, 48, 1, true).translate(0, -0.5, 0),
      new THREE.ShaderMaterial({
        transparent: true,
        depthWrite: false,
        blending: THREE.AdditiveBlending,
        side: THREE.DoubleSide,
        uniforms: { color: { value: new THREE.Color(0x6fb8d0) } },
        vertexShader: `varying float vY; varying vec3 vN; varying vec3 vV;
          void main() {
            vY = position.y;
            vec4 mv = modelViewMatrix * vec4(position, 1.0);
            vN = normalize(normalMatrix * normal);
            vV = normalize(-mv.xyz);
            gl_Position = projectionMatrix * mv;
          }`,
        fragmentShader: `uniform vec3 color; varying float vY; varying vec3 vN; varying vec3 vV;
          void main() {
            float along = clamp(1.0 + vY, 0.0, 1.0);
            float edge = pow(abs(dot(normalize(vN), normalize(vV))), 1.5);
            gl_FragColor = vec4(color * along * edge * 0.07, 1.0);
          }`,
      }),
    );
    this.group.add(this.#beam);

    this.#dust = dust(260);
    this.group.add(this.#dust);

    this.#bg = background();
    this.grade = gradePass();
  }

  /** Sizes and places everything around a model's bounds. */
  fit(box: THREE.Box3) {
    const center = box.getCenter(new THREE.Vector3());
    const size = box.getSize(new THREE.Vector3()).length();

    this.#spot.position.set(center.x, box.max.y + size * 1.6, center.z + size * 0.25);
    this.#spot.target.position.copy(center);
    this.#spot.shadow.camera.near = size * 0.2;
    this.#spot.shadow.camera.far = size * 5;

    this.#floor.position.set(center.x, box.min.y - size * 0.04, center.z);
    this.#floor.scale.setScalar(size * 1.4);

    this.#beam.position.copy(this.#spot.position);
    this.#beam.scale.set(size * 0.95, this.#spot.position.y - this.#floor.position.y, size * 0.95);

    this.#dust.position.copy(center);
    this.#dust.scale.setScalar(size * 1.2);
    this.#fog.density = 0.16 / size;
  }

  /** Turns the stage on or off for a scene and its model. */
  apply(scene: THREE.Scene, model: THREE.Object3D | undefined, on: boolean) {
    this.group.visible = on;
    this.grade.enabled = on;
    scene.fog = on ? this.#fog : null;
    scene.background = on ? this.#bg : scene.background;
    model?.traverse((child) => {
      if (child instanceof THREE.Mesh) child.castShadow = on;
    });
  }

  tick(t: number) {
    this.#dust.rotation.y = t * 0.02;
    this.#dust.position.y += Math.sin(t * 0.3) * 0.00005;
  }
}

/** Slow plankton specks in a unit sphere. */
function dust(count: number): THREE.Points {
  const pos = new Float32Array(count * 3);
  for (let i = 0; i < count; i++) {
    const r = Math.cbrt(Math.random());
    const th = Math.random() * Math.PI * 2;
    const ph = Math.acos(2 * Math.random() - 1);
    pos.set([r * Math.sin(ph) * Math.cos(th), r * Math.cos(ph), r * Math.sin(ph) * Math.sin(th)], i * 3);
  }
  const geo = new THREE.BufferGeometry();
  geo.setAttribute('position', new THREE.BufferAttribute(pos, 3));
  return new THREE.Points(
    geo,
    new THREE.PointsMaterial({ color: 0x9fd8e0, size: 2, sizeAttenuation: false, transparent: true, opacity: 0.35, depthWrite: false }),
  );
}

/** Deep teal fading to near black, lighter where the beam comes from. */
function background(): THREE.Texture {
  const c = document.createElement('canvas');
  c.width = 2;
  c.height = 256;
  const ctx = c.getContext('2d')!;
  const g = ctx.createLinearGradient(0, 0, 0, 256);
  g.addColorStop(0, '#14485a');
  g.addColorStop(0.45, '#0a2430');
  g.addColorStop(1, '#03090d');
  ctx.fillStyle = g;
  ctx.fillRect(0, 0, 2, 256);
  const tex = new THREE.CanvasTexture(c);
  tex.colorSpace = THREE.SRGBColorSpace;
  return tex;
}

/** Substratum's adjustment (brightness 0.7, contrast 1.06, saturation 1.38) and a soft vignette. */
function gradePass(): Pass {
  return new ShaderPass({
    uniforms: { tDiffuse: { value: null }, brightness: { value: 0.95 }, contrast: { value: 1.12 }, saturation: { value: 1.3 } },
    vertexShader: `varying vec2 vUv; void main() { vUv = uv; gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0); }`,
    fragmentShader: `uniform sampler2D tDiffuse; uniform float brightness, contrast, saturation; varying vec2 vUv;
      void main() {
        vec3 c = texture2D(tDiffuse, vUv).rgb * brightness;
        c = (c - 0.5) * contrast + 0.5;
        float l = dot(c, vec3(0.2126, 0.7152, 0.0722));
        c = mix(vec3(l), c, saturation);
        float v = smoothstep(1.05, 0.35, length(vUv - 0.5));
        gl_FragColor = vec4(max(c, 0.0) * mix(0.55, 1.0, v), 1.0);
      }`,
  });
}
