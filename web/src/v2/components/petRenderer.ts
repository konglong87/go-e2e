import {
  AmbientLight, Box3, DirectionalLight, Group, Mesh, PerspectiveCamera,
  Scene, SRGBColorSpace, Vector3, WebGLRenderer, type Material, type Object3D
} from "three";
import { GLTFLoader } from "three/addons/loaders/GLTFLoader.js";
import type { PetAsset } from "./petAssets";

const FIT_VIEW_SIZE = 1.9;

function disposeModel(model: Object3D): void {
  const materials = new Set<Material>();
  model.traverse((node) => {
    if (!(node instanceof Mesh)) return;
    node.geometry.dispose();
    for (const material of Array.isArray(node.material) ? node.material : [node.material]) materials.add(material);
  });
  materials.forEach((material) => material.dispose());
}

export function createPetRenderer(canvas: HTMLCanvasElement, asset: PetAsset, getAnimation: () => string, onReady: () => void, onError: () => void): () => void {
  const renderer = new WebGLRenderer({ canvas, alpha: true, antialias: true, preserveDrawingBuffer: true });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
  renderer.setSize(112, 132, false);
  renderer.outputColorSpace = SRGBColorSpace;
  const scene = new Scene();
  const camera = new PerspectiveCamera(asset.camera.fov, 112 / 132, 0.1, 20);
  camera.position.set(0.35, 0.3, asset.camera.distance);
  camera.lookAt(0, 0, 0);
  scene.add(new AmbientLight(0xffffff, 2));
  const key = new DirectionalLight(0xffffff, 3.5);
  key.position.set(3, 5, 4);
  scene.add(key);
  const fill = new DirectionalLight(0xffffff, 1.5);
  fill.position.set(-3, 1, -2);
  scene.add(fill);
  const pivot = new Group();
  scene.add(pivot);
  const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)");
  let closed = false;
  let model: Object3D | undefined;
  let frame = 0;
  const start = performance.now();
  const draw = (now: number): void => {
    if (closed) return;
    const time = (now - start) / 1000;
    const animation = getAnimation();
    const moving = !reducedMotion.matches && animation !== "focus";
    pivot.rotation.y = moving ? Math.sin(time * 1.4) * (animation === "celebrate" ? 0.45 : 0.12) : 0;
    pivot.position.y = moving ? Math.sin(time * (animation === "celebrate" ? 5 : 2)) * 0.035 : 0;
    if (!document.hidden && canvas.getClientRects().length > 0) renderer.render(scene, camera);
    frame = window.requestAnimationFrame(draw);
  };
  const contextLost = (event: Event): void => { event.preventDefault(); onError(); };
  canvas.addEventListener("webglcontextlost", contextLost);
  new GLTFLoader().load(asset.url, (gltf) => {
    if (closed) { disposeModel(gltf.scene); return; }
    model = gltf.scene;
    const box = new Box3().setFromObject(model);
    const size = box.getSize(new Vector3());
    const maxDimension = Math.max(size.x, size.y, size.z);
    if (maxDimension > 0) model.scale.setScalar(FIT_VIEW_SIZE / maxDimension);
    const center = new Box3().setFromObject(model).getCenter(new Vector3());
    model.position.sub(center);
    pivot.add(model);
    renderer.render(scene, camera);
    onReady();
    frame = window.requestAnimationFrame(draw);
  }, undefined, () => { if (!closed) onError(); });
  return () => {
    closed = true;
    window.cancelAnimationFrame(frame);
    canvas.removeEventListener("webglcontextlost", contextLost);
    if (model) disposeModel(model);
    renderer.dispose();
  };
}
