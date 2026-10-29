<script lang="ts">
  import * as THREE from 'three';
  import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js';
  import { GLTFLoader } from 'three/examples/jsm/loaders/GLTFLoader.js';
  import { EffectComposer } from 'three/examples/jsm/postprocessing/EffectComposer.js';
  import { RenderPass } from 'three/examples/jsm/postprocessing/RenderPass.js';
  import { UnrealBloomPass } from 'three/examples/jsm/postprocessing/UnrealBloomPass.js';
  import { OutputPass } from 'three/examples/jsm/postprocessing/OutputPass.js';
  import { fileURL } from '../lib/api';
  import { editor } from '../lib/editor.svelte';
  import type { ModelRef, Pick } from '../lib/types';

  let {
    url,
    model,
    onpick,
    autoRotate = true,
    brush,
    onpaint,
    onpaintend,
    overlay,
    lit = false,
    onhover,
  }: {
    url: string;
    model?: ModelRef;
    /** Called on a click (not a drag) on the model with the part and the original texture color there. */
    onpick?: (hit: Pick) => void;
    autoRotate?: boolean;
    /** Set while painting: dragging paints, Space held orbits. Radius is in model diagonals. */
    brush?: { radius: number };
    onpaint?: (hit: Pick, first: boolean, erase: boolean) => void;
    onpaintend?: () => void;
    /** A gray mask drawn tinted over the texture, updated whenever version changes. */
    overlay?: { canvas: HTMLCanvasElement; version: number };
    /** Light the model like the game does instead of showing the texture unlit. */
    lit?: boolean;
    /** Called with the part under the pointer, or '' when there is none. */
    onhover?: (part: string) => void;
  } = $props();

  let container: HTMLDivElement;
  let scene: THREE.Scene | undefined;
  let fit: ((object: THREE.Object3D) => void) | undefined;

  // Materials showing the rendered texture; emissive ones keep their own color.
  let textured: (THREE.MeshBasicMaterial | THREE.MeshLambertMaterial)[] = [];
  let lights: THREE.Group | undefined;
  let rendererRef: THREE.WebGLRenderer | undefined;
  let hovered = '';
  // State, so the paint overlay recomposes when a new render arrives.
  let texture = $state.raw<THREE.Texture | undefined>();
  // State, so switching to the lit look also reaches a model that loads later.
  let root = $state.raw<THREE.Object3D | undefined>();
  let camera: THREE.PerspectiveCamera | undefined;
  // The model's own texture, for picking colors.
  let original: CanvasRenderingContext2D | undefined;
  let controlsRef: OrbitControls | undefined;
  let ring: THREE.Mesh | undefined;
  let spaceDown = false;
  let composite: THREE.CanvasTexture | undefined;

  // Baked lighting is in the texture, so unlit materials show it as in a Quake-style engine.
  $effect(() => {
    const renderer = new THREE.WebGLRenderer({ antialias: true });
    renderer.setPixelRatio(window.devicePixelRatio);
    container.appendChild(renderer.domElement);

    scene = new THREE.Scene();
    rendererRef = renderer;

    // Like Substratum's hangar: teal ambient, a spot from above and a faint headlight from the camera.
    lights = new THREE.Group();
    lights.add(new THREE.AmbientLight(new THREE.Color(0.25, 0.44, 0.51), 1.5));
    const spot = new THREE.SpotLight(0xd8ecff, 40, 0, Math.PI / 5, 0.6, 1.2);
    spot.position.set(0, 3, 0.4);
    lights.add(spot, spot.target);
    lights.visible = false;
    scene.add(lights);

    const cam = new THREE.PerspectiveCamera(40, 1, 0.01, 100);
    cam.position.set(2.2, 1.6, 2.2);
    camera = cam;
    const headlight = new THREE.PointLight(0xfff2dd, 1.5, 0, 1.5);
    cam.add(headlight);
    lights.add(cam);

    const controls = new OrbitControls(cam, renderer.domElement);
    controlsRef = controls;

    ring = new THREE.Mesh(
      new THREE.RingGeometry(0.9, 1, 48),
      new THREE.MeshBasicMaterial({ color: 0xff9a3c, depthTest: false, transparent: true, opacity: 0.9, side: THREE.DoubleSide }),
    );
    ring.renderOrder = 10;
    ring.visible = false;
    scene.add(ring);
    controls.enableDamping = true;
    controls.autoRotate = autoRotate;
    controls.autoRotateSpeed = 1;

    // Only emissive colors above 1 glow, like bloom in the engine.
    const composer = new EffectComposer(renderer);
    composer.addPass(new RenderPass(scene, cam));
    const bloom = new UnrealBloomPass(new THREE.Vector2(256, 256), 0.45, 0.4, 1);
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

    // A click picks, a drag orbits; with a brush a drag paints instead.
    let down: { x: number; y: number } | undefined;
    let painting = false;
    const onDown = (e: PointerEvent) => {
      down = { x: e.clientX, y: e.clientY };
      if (brush && !spaceDown && e.button === 0) {
        const hit = raycast(e);
        if (hit) {
          painting = true;
          renderer.domElement.setPointerCapture(e.pointerId);
          onpaint?.(hit, true, e.altKey);
        }
      }
    };
    const onMove = (e: PointerEvent) => {
      const hit = brush || onhover ? raycast(e) : undefined;
      showRing(brush ? hit : undefined);
      if (painting && hit) onpaint?.(hit, false, e.altKey);
      if (!painting && e.buttons === 0) hover(hit?.part ?? '');
    };
    const onUp = (e: PointerEvent) => {
      if (painting) {
        painting = false;
        onpaintend?.();
      } else if (down && Math.hypot(e.clientX - down.x, e.clientY - down.y) < 4) {
        const hit = raycast(e);
        if (hit) onpick?.(hit);
      }
      down = undefined;
    };
    const onLeave = () => {
      showRing(undefined);
      hover('');
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.code !== 'Space' || (e.target as HTMLElement).closest('input, textarea, select')) return;
      spaceDown = e.type === 'keydown';
      if (brush) e.preventDefault();
      syncControls();
    };

    renderer.domElement.addEventListener('pointerdown', onDown);
    renderer.domElement.addEventListener('pointermove', onMove);
    renderer.domElement.addEventListener('pointerup', onUp);
    renderer.domElement.addEventListener('pointerleave', onLeave);
    window.addEventListener('keydown', onKey);
    window.addEventListener('keyup', onKey);

    return () => {
      renderer.setAnimationLoop(null);
      renderer.domElement.removeEventListener('pointerdown', onDown);
      renderer.domElement.removeEventListener('pointermove', onMove);
      renderer.domElement.removeEventListener('pointerup', onUp);
      renderer.domElement.removeEventListener('pointerleave', onLeave);
      window.removeEventListener('keydown', onKey);
      window.removeEventListener('keyup', onKey);
      composite?.dispose();
      resize.disconnect();
      controls.dispose();
      texture?.dispose();
      composer.dispose();
      renderer.dispose();
      renderer.domElement.remove();
      scene = undefined;
    };
  });

  // The panel color of the current theme, resolved by the browser since theme colors are oklch.
  $effect(() => {
    editor.theme;
    const target = scene;
    if (!target) return;

    // Wait a frame, so the theme's CSS is applied before reading it.
    const frame = requestAnimationFrame(() => {
      const ctx = document.createElement('canvas').getContext('2d')!;
      ctx.fillStyle = getComputedStyle(container).backgroundColor;
      ctx.fillRect(0, 0, 1, 1);
      const [r, g, b] = ctx.getImageData(0, 0, 1, 1).data;
      target.background = new THREE.Color(r / 255, g / 255, b / 255).convertSRGBToLinear();
    });
    return () => cancelAnimationFrame(frame);
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

  $effect(() => {
    brush;
    syncControls();
    if (!brush) showRing(undefined);
  });

  // The painted mask over the texture, tinted, so strokes show before the render catches up.
  $effect(() => {
    const o = overlay;
    o?.version;
    if (!o) {
      composite?.dispose();
      composite = undefined;
      applyTexture();
      return;
    }

    const base = texture?.image as CanvasImageSource | undefined;
    const { width: w, height: h } = o.canvas;
    const out = document.createElement('canvas');
    out.width = w;
    out.height = h;
    const ctx = out.getContext('2d', { willReadFrequently: true })!;
    if (base) ctx.drawImage(base, 0, 0, w, h);

    const img = ctx.getImageData(0, 0, w, h);
    const mask = o.canvas.getContext('2d', { willReadFrequently: true })!.getImageData(0, 0, w, h).data;
    for (let i = 0; i < img.data.length; i += 4) {
      const m = (mask[i] / 255) * 0.6;
      img.data[i] += (255 - img.data[i]) * m;
      img.data[i + 1] += (150 - img.data[i + 1]) * m;
      img.data[i + 2] += (60 - img.data[i + 2]) * m;
    }
    ctx.putImageData(img, 0, 0);

    composite?.dispose();
    composite = new THREE.CanvasTexture(out);
    composite.magFilter = THREE.NearestFilter;
    composite.colorSpace = THREE.SRGBColorSpace;
    composite.flipY = false;
    applyTexture();
  });

  // Switching between the unlit and the game-like look swaps the textured materials.
  $effect(() => {
    const l = lit;
    if (lights) lights.visible = l;
    if (rendererRef) {
      rendererRef.toneMapping = l ? THREE.ACESFilmicToneMapping : THREE.NoToneMapping;
      rendererRef.toneMappingExposure = 1.1;
    }
    if (!root) return;

    const next: typeof textured = [];
    root.traverse((child) => {
      if (!(child instanceof THREE.Mesh)) return;
      const swap = (m: THREE.Material) => {
        if (!textured.includes(m as never)) return m;
        const made = l
          ? new THREE.MeshLambertMaterial({ side: THREE.DoubleSide })
          : new THREE.MeshBasicMaterial({ side: THREE.DoubleSide });
        next.push(made);
        m.dispose();
        return made;
      };
      child.material = Array.isArray(child.material) ? child.material.map(swap) : swap(child.material);
    });
    textured = next;
    applyTexture();
  });

  /** Brightens the part under the pointer and reports it. */
  function hover(part: string) {
    if (part === hovered || !root) return;
    hovered = part;
    onhover?.(part);

    root.traverse((child) => {
      if (!(child instanceof THREE.Mesh)) return;
      let top: THREE.Object3D = child;
      while (top.parent && top.parent !== root) top = top.parent;
      const on = part !== '' && top.name === part;
      for (const m of Array.isArray(child.material) ? child.material : [child.material]) {
        if (textured.includes(m as never)) (m as THREE.MeshBasicMaterial).color.setScalar(on ? 1.45 : 1);
      }
    });
  }

  function syncControls() {
    if (controlsRef) controlsRef.enabled = !brush || spaceDown;
  }

  function showRing(hit: Pick | undefined) {
    if (!ring || !root) return;
    ring.visible = !!hit && !!brush;
    if (!hit || !brush) return;

    const box = new THREE.Box3().setFromObject(root, true);
    const diag = box.getSize(new THREE.Vector3()).length() || 1;
    const p = new THREE.Vector3(...hit.position).multiplyScalar(diag).add(box.min);
    const n = new THREE.Vector3(...hit.normal);

    ring.position.copy(p).addScaledVector(n, diag * 0.002);
    ring.lookAt(p.clone().add(n));
    ring.scale.setScalar(brush.radius * diag);
  }

  /** The model point under the pointer, in the bake's units, or undefined. */
  function raycast(e: PointerEvent): Pick | undefined {
    if (!root || !camera) return;

    const rect = container.getBoundingClientRect();
    const ndc = new THREE.Vector2(((e.clientX - rect.left) / rect.width) * 2 - 1, -((e.clientY - rect.top) / rect.height) * 2 + 1);
    const ray = new THREE.Raycaster();
    ray.setFromCamera(ndc, camera);

    const hit = ray.intersectObject(root, true).find((h) => h.object !== ring);
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
    // Positions in the bake's units: offset from the model's minimum, in model diagonals.
    const box = new THREE.Box3().setFromObject(root, true);
    const diag = box.getSize(new THREE.Vector3()).length() || 1;
    const p = hit.point.clone().sub(box.min).divideScalar(diag);
    const n = hit.face ? hit.face.normal.clone().transformDirection(hit.object.matrixWorld) : new THREE.Vector3(0, 0, 1);
    // Faces may be seen from behind; point the normal at the camera.
    if (n.dot(ray.ray.direction) > 0) n.negate();

    const round = (v: number) => Math.round(v * 1000) / 1000;
    return { part: part.name, color, position: p.toArray().map(round), normal: n.toArray().map(round) };
  }

  function applyTexture() {
    for (const m of textured) {
      m.map = composite ?? texture ?? null;
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

<div bind:this={container} class="absolute inset-0 bg-panel"></div>
