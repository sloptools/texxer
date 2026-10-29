<script lang="ts">
  import * as THREE from 'three';
  import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js';
  import { GLTFLoader } from 'three/examples/jsm/loaders/GLTFLoader.js';
  import { EffectComposer } from 'three/examples/jsm/postprocessing/EffectComposer.js';
  import { RenderPass } from 'three/examples/jsm/postprocessing/RenderPass.js';
  import { UnrealBloomPass } from 'three/examples/jsm/postprocessing/UnrealBloomPass.js';
  import { OutputPass } from 'three/examples/jsm/postprocessing/OutputPass.js';
  import { fileURL } from '../lib/api';
  import type { ModelRef } from '../lib/types';

  let {
    url,
    model,
    onpick,
  }: {
    url: string;
    model?: ModelRef;
    /** Called on a click (not a drag) on the model with the part and the original texture color there. */
    onpick?: (hit: { part: string; color?: string }) => void;
  } = $props();

  let container: HTMLDivElement;
  let scene: THREE.Scene | undefined;
  let fit: ((object: THREE.Object3D) => void) | undefined;

  // Materials showing the rendered texture; emissive ones keep their own color.
  let textured: THREE.MeshBasicMaterial[] = [];
  let texture: THREE.Texture | undefined;
  let root: THREE.Object3D | undefined;
  let camera: THREE.PerspectiveCamera | undefined;
  // The model's own texture, for picking colors.
  let original: CanvasRenderingContext2D | undefined;

  // Baked lighting is in the texture, so unlit materials show it as in a Quake-style engine.
  $effect(() => {
    const renderer = new THREE.WebGLRenderer({ antialias: true });
    renderer.setPixelRatio(window.devicePixelRatio);
    container.appendChild(renderer.domElement);

    scene = new THREE.Scene();
    scene.background = new THREE.Color(getComputedStyle(container).getPropertyValue('--color-panel').trim() || '#18181b');

    const cam = new THREE.PerspectiveCamera(40, 1, 0.01, 100);
    cam.position.set(2.2, 1.6, 2.2);
    camera = cam;

    const controls = new OrbitControls(cam, renderer.domElement);
    controls.enableDamping = true;
    controls.autoRotate = true;
    controls.autoRotateSpeed = 1;

    // Only emissive colors above 1 glow, like bloom in the engine.
    const composer = new EffectComposer(renderer);
    composer.addPass(new RenderPass(scene, cam));
    const bloom = new UnrealBloomPass(new THREE.Vector2(256, 256), 0.9, 0.5, 1);
    composer.addPass(bloom);
    composer.addPass(new OutputPass());

    fit = (object) => {
      const box = new THREE.Box3().setFromObject(object);
      const center = box.getCenter(new THREE.Vector3());
      const radius = box.getSize(new THREE.Vector3()).length() / 2;

      controls.target.copy(center);
      cam.position.copy(center).add(new THREE.Vector3(1, 0.7, 1).normalize().multiplyScalar(radius * 3));
      cam.near = radius / 100;
      cam.far = radius * 100;
      cam.updateProjectionMatrix();
    };

    const resize = new ResizeObserver(() => {
      const { clientWidth: w, clientHeight: h } = container;
      renderer.setSize(w, h);
      composer.setSize(w, h);
      bloom.resolution.set(w, h);
      cam.aspect = w / h;
      cam.updateProjectionMatrix();
    });
    resize.observe(container);

    renderer.setAnimationLoop(() => {
      controls.update();
      composer.render();
    });

    // A click picks, a drag orbits.
    let down: { x: number; y: number } | undefined;
    const onDown = (e: PointerEvent) => (down = { x: e.clientX, y: e.clientY });
    const onUp = (e: PointerEvent) => {
      if (down && Math.hypot(e.clientX - down.x, e.clientY - down.y) < 4) pick(e);
      down = undefined;
    };
    renderer.domElement.addEventListener('pointerdown', onDown);
    renderer.domElement.addEventListener('pointerup', onUp);

    return () => {
      renderer.setAnimationLoop(null);
      renderer.domElement.removeEventListener('pointerdown', onDown);
      renderer.domElement.removeEventListener('pointerup', onUp);
      resize.disconnect();
      controls.dispose();
      texture?.dispose();
      composer.dispose();
      renderer.dispose();
      renderer.domElement.remove();
      scene = undefined;
    };
  });

  // The model, or a cube for tiling textures.
  $effect(() => {
    const target = scene;
    const ref = model;
    if (!target) return;

    let cancelled = false;
    const place = (object: THREE.Object3D) => {
      if (cancelled) return;
      root = object;
      target.add(object);
      fit?.(object);
      applyTexture();
    };

    if (ref) {
      new GLTFLoader().load(fileURL(ref.path), (gltf) => place(prepare(gltf.scene, ref.material)));
    } else {
      const material = new THREE.MeshBasicMaterial();
      textured = [material];
      place(new THREE.Mesh(new THREE.BoxGeometry(1.4, 1.4, 1.4), material));
    }

    return () => {
      cancelled = true;
      if (root) target.remove(root);
      root = undefined;
      textured = [];
    };
  });

  $effect(() => {
    const src = url;
    new THREE.TextureLoader().load(src, (loaded) => {
      loaded.magFilter = THREE.NearestFilter;
      loaded.colorSpace = THREE.SRGBColorSpace;
      // glTF UVs start at the top left of the image.
      loaded.flipY = !model;

      texture?.dispose();
      texture = loaded;
      applyTexture();
    });
  });

  function pick(e: PointerEvent) {
    if (!onpick || !root || !camera) return;

    const rect = container.getBoundingClientRect();
    const ndc = new THREE.Vector2(((e.clientX - rect.left) / rect.width) * 2 - 1, -((e.clientY - rect.top) / rect.height) * 2 + 1);
    const ray = new THREE.Raycaster();
    ray.setFromCamera(ndc, camera);

    const hit = ray.intersectObject(root, true)[0];
    if (!hit) return;

    // Parts are the model's top-level nodes, as the bake names them.
    let part: THREE.Object3D = hit.object;
    while (part.parent && part.parent !== root) part = part.parent;

    let color: string | undefined;
    if (original && hit.uv) {
      const { width, height } = original.canvas;
      const x = Math.min(width - 1, Math.max(0, Math.floor(hit.uv.x * width)));
      const y = Math.min(height - 1, Math.max(0, Math.floor(hit.uv.y * height)));
      const [r, g, b] = original.getImageData(x, y, 1, 1).data;
      color = '#' + [r, g, b].map((v) => v.toString(16).padStart(2, '0')).join('');
    }
    onpick({ part: part.name, color });
  }

  function applyTexture() {
    for (const m of textured) {
      m.map = texture ?? null;
      m.needsUpdate = true;
    }
  }

  /** Swaps the model's materials for unlit ones: the texture where it is used, glowing colors where emissive. */
  function prepare(object: THREE.Object3D, material: string): THREE.Object3D {
    textured = [];
    object.traverse((child) => {
      if (!(child instanceof THREE.Mesh)) return;

      const swap = (m: THREE.Material) => {
        const std = m as THREE.MeshStandardMaterial;
        const emissive = std.emissive && std.emissive.getHex() !== 0;

        if (emissive) {
          const color = std.emissive.clone().multiplyScalar(Math.max(std.emissiveIntensity ?? 1, 1) * 1.5);
          return new THREE.MeshBasicMaterial({ color, side: THREE.DoubleSide });
        }
        if (material && m.name !== material) {
          return new THREE.MeshBasicMaterial({ color: std.color ?? 0x808080, side: THREE.DoubleSide });
        }

        const image = std.map?.image as CanvasImageSource & { width: number; height: number } | undefined;
        if (image && !original) {
          const canvas = document.createElement('canvas');
          canvas.width = image.width;
          canvas.height = image.height;
          original = canvas.getContext('2d', { willReadFrequently: true }) ?? undefined;
          original?.drawImage(image, 0, 0);
        }

        const basic = new THREE.MeshBasicMaterial({ side: THREE.DoubleSide });
        textured.push(basic);
        return basic;
      };

      child.material = Array.isArray(child.material) ? child.material.map(swap) : swap(child.material);
    });
    return object;
  }
</script>

<div bind:this={container} class="absolute inset-0"></div>
