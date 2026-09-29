<template>
  <div id="titan-landing-scene" ref="host" class="titan-scene" role="group" aria-label="土卫六三维着陆点示意，可拖动观察并滚轮缩放">
    <div class="titan-heading"><strong>土卫六 · 惠更斯号着陆点</strong><span>西经 192.3° · 南纬 10.3°</span></div>
    <div class="titan-toolbar scene-toolbar" aria-label="土卫六图层">
      <span>图层</span><label><input v-model="siteVisible" type="checkbox"><i class="sites" />着陆点</label>
    </div>
    <MissionSceneLabel
      v-if="label && siteVisible"
      class="titan-site-label"
      :style="sceneAnnotationStyle(label)"
      kind="surface"
      :name-zh="site.name"
      :name-en="site.nameEn"
      :selected="selected"
      :mode="label.mode"
      :side="label.side"
      :icon-html="siteGlyph('lander')"
      @click.stop="focus"
    />
    <MissionDetailPanel v-if="selected" :detail="detail" @close="selected = false" />
    <p class="titan-caption">Cassini 近红外全球拼图：NASA/JPL-Caltech/Univ. Arizona。落点坐标取自 ESA，位置为示意。拖动旋转，滚轮缩放。</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as THREE from 'three'
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'
import type { PlanetSite } from '../planetPages'
import MissionSceneLabel from './MissionSceneLabel.vue'
import MissionDetailPanel from './MissionDetailPanel.vue'
import type { MissionDetail } from '../missionPresentation'
import { surfaceMissionFields } from '../missionPresentation'
import { sceneAnnotationStyle, layoutSceneAnnotations, projectedSphereRadiusPx, surfaceMarkerRadiusPx, surfaceMarkerWorldRadius } from '../surfaceAnnotations'
import type { SceneAnnotationLayout } from '../surfaceAnnotations'
import { solarTexture } from '../solar/textures'
import titanMapUrl from '../assets/solar/titan-cassini-near-infrared.jpg'

const props = defineProps<{ site: PlanetSite }>()
const host = ref<HTMLDivElement | null>(null)
const selected = ref(false)
const siteVisible = ref(true)
const label = ref<SceneAnnotationLayout | null>(null)
const detail = computed<MissionDetail>(() => ({
  kind: 'surface', typeZh: '土卫六着陆点', typeEn: 'TITAN LANDING SITE',
  nameZh: props.site.name, nameEn: props.site.nameEn, description: props.site.description,
  iconHtml: siteGlyph('lander'),
  fields: surfaceMissionFields({
    mission: props.site.mission, date: props.site.date, operator: props.site.operator,
    category: '软着陆', coordinates: '10.3°S 167.7°E（192.3°W）',
  }),
  source: `${props.site.verifiedAt ?? ''} · ${props.site.source ?? ''}`,
}))

function siteGlyph(_icon: 'lander') {
  return '<svg viewBox="0 0 20 20" fill="none" aria-hidden="true"><path d="M6 12.5h8M8 5h4l2 7H6l2-7Zm-2 7-2 4m10-4 2 4M4 16.5h3m6 0h3" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/></svg>'
}

let renderer: THREE.WebGLRenderer | null = null
let scene: THREE.Scene | null = null
let camera: THREE.PerspectiveCamera | null = null
let controls: OrbitControls | null = null
let globe: THREE.Mesh | null = null
let marker: THREE.Mesh | null = null
let frame = 0
let resizeObserver: ResizeObserver | null = null
const point = new THREE.Vector3()
const origin = new THREE.Vector3()
const RADIUS = 1.6
const FOV = 42
let focusAnimation: { from: THREE.Vector3, to: THREE.Vector3, startedAt: number } | null = null

function focus() {
  selected.value = true
  if (!camera || !controls || !marker || !globe) return
  marker.getWorldPosition(point)
  focusAnimation = { from: camera.position.clone(), to: point.normalize().multiplyScalar(4.15), startedAt: performance.now() }
}
defineExpose({ focus })

watch(siteVisible, (enabled) => {
  if (marker) marker.visible = enabled
  if (!enabled) selected.value = false
})

onMounted(() => {
  const root = host.value
  if (!root) return
  const width = root.clientWidth
  const height = root.clientHeight
  scene = new THREE.Scene()
  camera = new THREE.PerspectiveCamera(FOV, width / height, 0.1, 100)
  camera.position.set(0, 0.15, 5.2)
  renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true })
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
  renderer.setSize(width, height)
  renderer.outputColorSpace = THREE.SRGBColorSpace
  root.prepend(renderer.domElement)
  const texture = solarTexture(titanMapUrl)
  texture.colorSpace = THREE.SRGBColorSpace
  const material = new THREE.MeshStandardMaterial({ map: texture, color: 0xd4ad7d, roughness: 1, metalness: 0 })
  globe = new THREE.Mesh(new THREE.SphereGeometry(RADIUS, 96, 64), material)
  scene.add(globe)
  scene.add(new THREE.AmbientLight(0xf4d7ab, 1.25))
  const sun = new THREE.DirectionalLight(0xffecd0, 2.2)
  sun.position.set(-3, 4, 7)
  scene.add(sun)
  const lat = (props.site.latitude ?? -10.3) * Math.PI / 180
  const lon = (props.site.longitude ?? 167.7) * Math.PI / 180
  marker = new THREE.Mesh(new THREE.SphereGeometry(1, 12, 12), new THREE.MeshBasicMaterial({ color: 0xf5dfaf }))
  marker.position.set(RADIUS * Math.cos(lat) * Math.cos(lon), RADIUS * Math.sin(lat), -RADIUS * Math.cos(lat) * Math.sin(lon))
  globe.add(marker)
  // 首屏让坐标落在可见半球，同时保留可拖动的真实经纬度球面关系。
  globe.quaternion.setFromUnitVectors(marker.position.clone().normalize(), new THREE.Vector3(0.27, -0.1, 0.96).normalize())
  controls = new OrbitControls(camera, renderer.domElement)
  controls.enableDamping = true
  controls.enablePan = false
  controls.minDistance = 2.3
  controls.maxDistance = 8.5
  controls.addEventListener('start', () => { focusAnimation = null })
  resizeObserver = new ResizeObserver(() => {
    if (!host.value || !camera || !renderer) return
    const w = host.value.clientWidth
    const h = host.value.clientHeight
    if (!w || !h) return
    camera.aspect = w / h
    camera.updateProjectionMatrix()
    renderer.setSize(w, h)
  })
  resizeObserver.observe(root)
  const animate = () => {
    frame = requestAnimationFrame(animate)
    if (!camera || !renderer || !scene || !marker || !host.value) return
    if (focusAnimation) {
      const t = Math.min(1, (performance.now() - focusAnimation.startedAt) / 750)
      const eased = 1 - Math.pow(1 - t, 3)
      const direction = focusAnimation.from.clone().normalize().lerp(focusAnimation.to.clone().normalize(), eased).normalize()
      const distance = THREE.MathUtils.lerp(focusAnimation.from.length(), focusAnimation.to.length(), eased)
      camera.position.copy(direction.multiplyScalar(distance))
      camera.lookAt(origin)
      if (t === 1) focusAnimation = null
    }
    controls?.update()
    marker.getWorldPosition(point)
    const height = host.value.clientHeight
    const currentRadius = projectedSphereRadiusPx(RADIUS, camera.position.length(), FOV, height)
    const referenceRadius = projectedSphereRadiusPx(RADIUS, 5.2, FOV, height)
    marker.scale.setScalar(surfaceMarkerWorldRadius(point.distanceTo(camera.position), FOV, height, surfaceMarkerRadiusPx(currentRadius, referenceRadius)))
    const visible = siteVisible.value && point.clone().normalize().dot(camera.position.clone().normalize()) > 0.12
    const projected = point.clone().project(camera)
    label.value = visible ? (layoutSceneAnnotations([{
      id: props.site.id,
      anchorX: (projected.x * 0.5 + 0.5) * host.value.clientWidth,
      anchorY: (-projected.y * 0.5 + 0.5) * height,
      visible: true, selected: selected.value,
    }], { width: host.value.clientWidth, height, currentPlanetRadiusPx: currentRadius, referencePlanetRadiusPx: referenceRadius,
      safeTopPx: 72, preferFullLabels: true }, label.value ? [label.value] : [])[0] ?? null) : null
    renderer.render(scene, camera)
  }
  animate()
})

onBeforeUnmount(() => {
  cancelAnimationFrame(frame)
  resizeObserver?.disconnect()
  controls?.dispose()
  globe?.geometry.dispose()
  ;(globe?.material as THREE.Material | undefined)?.dispose()
  marker?.geometry.dispose()
  ;(marker?.material as THREE.Material | undefined)?.dispose()
  renderer?.dispose()
  renderer?.domElement.remove()
})
</script>

<style scoped>
.titan-scene {
  position: relative;
  height: min(54vw, 430px);
  min-height: 330px;
  margin: 28px 0 34px;
  overflow: hidden;
  border: 1px solid var(--planet-line);
  border-radius: 14px;
  background: radial-gradient(circle at 48% 52%, #1d1b19 0, #0a0d13 58%, #06090f 100%);
  --mission-accent: var(--planet-accent);
  --mission-text: var(--planet-text);
  --mission-quiet: var(--planet-quiet);
  --mission-panel-surface: rgba(14, 16, 20, .96);
}
.titan-scene :deep(canvas) { display: block; width: 100%; height: 100%; }
.titan-heading { position: absolute; z-index: 2; top: 22px; left: 28px; display: grid; gap: 5px; color: var(--planet-text); pointer-events: none; }
.titan-heading strong { font-size: 15px; font-weight: 550; }
.titan-heading span { font: 10px var(--font-mono); color: var(--planet-quiet); }
.titan-toolbar { position: absolute; z-index: 2; top: 18px; left: auto; right: 18px; width: max-content; }
.titan-site-label { position: absolute; z-index: 3; }
.titan-caption { position: absolute; z-index: 2; bottom: 15px; left: 28px; margin: 0; color: var(--planet-quiet); font: 10px/1.5 var(--font-sans); pointer-events: none; }
.titan-scene :deep(.mission-detail-panel) { width: min(410px, calc(100% - 28px)); top: 66px; right: 14px; max-height: calc(100% - 84px); }
</style>
