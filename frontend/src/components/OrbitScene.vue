<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as THREE from 'three'
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'
import type { LaunchSite, SceneLayers, Selection, Spacecraft } from '../types'
import { EARTH_RADIUS, latLonToVector, sampleOrbit, spacecraftPoint } from '../orbit/coordinates'

const EARTH_AXIAL_TILT_DEGREES = 23.44
const EARTH_TILT = new THREE.Quaternion().setFromAxisAngle(
  new THREE.Vector3(0, 0, 1),
  THREE.MathUtils.degToRad(EARTH_AXIAL_TILT_DEGREES),
)

const props = defineProps<{
  spacecraft: Spacecraft[]
  sites: LaunchSite[]
  layers: SceneLayers
  selection: Selection | null
  focusTarget?: { latitude: number; longitude: number; distance?: number; key: string } | null
  observerTarget?: { latitude: number; longitude: number; label: string } | null
}>()

const emit = defineEmits<{
  select: [selection: Selection]
}>()

const canvasHost = ref<HTMLDivElement | null>(null)
const labels = ref<Array<{ id: string; kind: 'spacecraft' | 'site'; name: string; x: number; y: number; visible: boolean }>>([])
const observerLabel = ref<{ name: string; x: number; y: number; visible: boolean } | null>(null)
const textureState = ref<'loading' | 'ready' | 'fallback'>('loading')
const pointerNearEarth = ref(false)

let renderer: THREE.WebGLRenderer | undefined
let scene: THREE.Scene | undefined
let camera: THREE.PerspectiveCamera | undefined
let controls: OrbitControls | undefined
let frameId = 0
let resizeObserver: ResizeObserver | undefined
let earthSystemGroup: THREE.Group | undefined
let spacecraftGroup: THREE.Group | undefined
let orbitGroup: THREE.Group | undefined
let siteGroup: THREE.Group | undefined
let observerMarker: THREE.Group | undefined
let earth: THREE.Mesh | undefined
let sunLight: THREE.DirectionalLight | undefined
let nightLightsMaterial: THREE.ShaderMaterial | undefined
const markerObjects = new Map<string, THREE.Object3D>()
const raycaster = new THREE.Raycaster()
const earthOcclusionSphere = new THREE.Sphere(new THREE.Vector3(0, 0, 0), EARTH_RADIUS * 1.004)
const earthOcclusionRay = new THREE.Ray()
const earthOcclusionHit = new THREE.Vector3()
const toOcclusionTarget = new THREE.Vector3()
const pointer = new THREE.Vector2()
const pointerStart = new THREE.Vector2()
let lastOrbitUpdate = 0
let lastSunUpdate = 0
let focusAnimation: {
  from: THREE.Vector3
  to: THREE.Vector3
  startedAt: number
  duration: number
} | undefined

const selectionKey = computed(() => props.selection ? `${props.selection.kind}:${props.selection.id}` : '')

function disposeGroup(group?: THREE.Group) {
  if (!group) return
  group.traverse((object) => {
    if (object instanceof THREE.Mesh || object instanceof THREE.Line) {
      object.geometry.dispose()
      const material = object.material
      if (Array.isArray(material)) material.forEach((item) => item.dispose())
      else material.dispose()
    }
  })
  group.clear()
  group.parent?.remove(group)
}

function markerMaterial(color: number, selected: boolean) {
  return new THREE.MeshBasicMaterial({
    color,
    transparent: true,
    opacity: selected ? 1 : 0.82,
    depthTest: true,
  })
}

function rebuildDataLayers() {
  if (!earthSystemGroup) return
  markerObjects.clear()
  disposeGroup(spacecraftGroup)
  disposeGroup(orbitGroup)
  disposeGroup(siteGroup)

  spacecraftGroup = new THREE.Group()
  orbitGroup = new THREE.Group()
  siteGroup = new THREE.Group()
  earthSystemGroup.add(spacecraftGroup, orbitGroup, siteGroup)

  const now = new Date()
  for (const craft of props.spacecraft) {
    const point = spacecraftPoint(craft, now)
    if (!point) continue
    const key = `spacecraft:${craft.id}`
    const selected = selectionKey.value === key
    const marker = new THREE.Mesh(
      new THREE.SphereGeometry(selected ? 0.052 : 0.037, 16, 16),
      markerMaterial(0x72d7ff, selected),
    )
    marker.position.copy(point.position)
    marker.userData = { kind: 'spacecraft', id: craft.id }
    spacecraftGroup.add(marker)
    markerObjects.set(key, marker)

    const halo = new THREE.Mesh(
      new THREE.RingGeometry(selected ? 0.075 : 0.058, selected ? 0.084 : 0.064, 28),
      new THREE.MeshBasicMaterial({ color: 0x72d7ff, transparent: true, opacity: selected ? 0.7 : 0.35, side: THREE.DoubleSide }),
    )
    halo.position.copy(point.position)
    halo.lookAt(camera?.position ?? new THREE.Vector3(0, 0, 8))
    spacecraftGroup.add(halo)

    const line = new THREE.Line(
      new THREE.BufferGeometry().setFromPoints(sampleOrbit(craft, now)),
      new THREE.LineBasicMaterial({ color: 0x42b7e8, transparent: true, opacity: selected ? 0.68 : 0.22 }),
    )
    orbitGroup.add(line)
  }

  for (const site of props.sites) {
    const key = `site:${site.id}`
    const selected = selectionKey.value === key
    const position = latLonToVector(site.latitude, site.longitude, EARTH_RADIUS * 1.006)
    const marker = new THREE.Mesh(
      new THREE.ConeGeometry(selected ? 0.047 : 0.035, selected ? 0.16 : 0.12, 8),
      markerMaterial(0xffb866, selected),
    )
    marker.position.copy(position)
    marker.quaternion.setFromUnitVectors(new THREE.Vector3(0, 1, 0), position.clone().normalize())
    marker.userData = { kind: 'site', id: site.id }
    siteGroup.add(marker)
    markerObjects.set(key, marker)
  }

  spacecraftGroup.visible = props.layers.spacecraft
  orbitGroup.visible = props.layers.orbits
  siteGroup.visible = props.layers.sites
}

function updateSpacecraftPositions(now: Date) {
  if (!spacecraftGroup) return
  for (const craft of props.spacecraft) {
    const point = spacecraftPoint(craft, now)
    const marker = markerObjects.get(`spacecraft:${craft.id}`)
    if (!point || !marker) continue
    marker.position.copy(point.position)
  }
}

function rebuildObserverMarker() {
  disposeGroup(observerMarker)
  observerMarker = undefined
  if (!earthSystemGroup || !props.observerTarget) return

  const position = latLonToVector(
    props.observerTarget.latitude,
    props.observerTarget.longitude,
    EARTH_RADIUS * 1.008,
  )
  observerMarker = new THREE.Group()
  observerMarker.position.copy(position)

  const point = new THREE.Mesh(
    new THREE.SphereGeometry(0.031, 18, 18),
    new THREE.MeshBasicMaterial({ color: 0x79e3bd }),
  )
  const ring = new THREE.Mesh(
    new THREE.RingGeometry(0.055, 0.062, 32),
    new THREE.MeshBasicMaterial({ color: 0x79e3bd, transparent: true, opacity: 0.55, side: THREE.DoubleSide }),
  )
  ring.lookAt(camera?.position ?? new THREE.Vector3(0, 0, 8))
  observerMarker.add(point, ring)
  earthSystemGroup.add(observerMarker)
}

function isOccludedByEarth(position: THREE.Vector3) {
  if (!camera) return false
  toOcclusionTarget.copy(position).sub(camera.position)
  const targetDistance = toOcclusionTarget.length()
  if (targetDistance === 0) return false
  earthOcclusionRay.set(camera.position, toOcclusionTarget.normalize())
  const intersection = earthOcclusionRay.intersectSphere(earthOcclusionSphere, earthOcclusionHit)
  if (!intersection) return false
  const intersectionDistance = camera.position.distanceTo(intersection)
  return intersectionDistance < targetDistance - 0.035
}

function updateLabels() {
  if (!camera || !canvasHost) return
  const width = canvasHost.value?.clientWidth ?? 0
  const height = canvasHost.value?.clientHeight ?? 0
  const next: typeof labels.value = []

  for (const craft of props.spacecraft) {
    const marker = markerObjects.get(`spacecraft:${craft.id}`)
    if (!marker || !props.layers.spacecraft) continue
    const position = marker.getWorldPosition(new THREE.Vector3())
    const projected = position.clone().project(camera)
    next.push({
      id: craft.id,
      kind: 'spacecraft',
      name: craft.nameZh,
      x: (projected.x * 0.5 + 0.5) * width,
      y: (-projected.y * 0.5 + 0.5) * height,
      visible: projected.z > -1 && projected.z < 1 && !isOccludedByEarth(position),
    })
  }

  for (const site of props.sites) {
    const marker = markerObjects.get(`site:${site.id}`)
    if (!marker || !props.layers.sites) continue
    const position = marker.getWorldPosition(new THREE.Vector3())
    const projected = position.clone().project(camera)
    const outward = position.clone().normalize()
    const towardCamera = camera.position.clone().sub(position).normalize()
    next.push({
      id: site.id,
      kind: 'site',
      name: site.nameZh,
      x: (projected.x * 0.5 + 0.5) * width,
      y: (-projected.y * 0.5 + 0.5) * height,
      visible: projected.z > -1 && projected.z < 1 && outward.dot(towardCamera) > -0.05 && !isOccludedByEarth(position),
    })
  }
  if (observerMarker && props.observerTarget) {
    const position = observerMarker.getWorldPosition(new THREE.Vector3())
    const projected = position.clone().project(camera)
    const outward = position.clone().normalize()
    const towardCamera = camera.position.clone().sub(position).normalize()
    observerLabel.value = {
      name: props.observerTarget.label,
      x: (projected.x * 0.5 + 0.5) * width,
      y: (-projected.y * 0.5 + 0.5) * height,
      visible: projected.z > -1 && projected.z < 1 && outward.dot(towardCamera) > -0.05 && !isOccludedByEarth(position),
    }
  } else {
    observerLabel.value = null
  }
  labels.value = next
}

function setupScene() {
  const host = canvasHost.value
  if (!host) return

  scene = new THREE.Scene()
  earthSystemGroup = new THREE.Group()
  earthSystemGroup.name = 'earth-equatorial-frame'
  earthSystemGroup.quaternion.copy(EARTH_TILT)
  scene.add(earthSystemGroup)
  camera = new THREE.PerspectiveCamera(42, host.clientWidth / host.clientHeight, 0.1, 120)
  camera.position.set(0.2, 0.35, 7.6)

  renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true, powerPreference: 'high-performance' })
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
  renderer.setSize(host.clientWidth, host.clientHeight)
  renderer.outputColorSpace = THREE.SRGBColorSpace
  renderer.toneMapping = THREE.ACESFilmicToneMapping
  renderer.toneMappingExposure = 0.96
  host.appendChild(renderer.domElement)

  controls = new OrbitControls(camera, renderer.domElement)
  controls.enableDamping = true
  controls.dampingFactor = 0.055
  controls.enablePan = false
  controls.minDistance = 4.4
  controls.maxDistance = 12
  controls.rotateSpeed = 0.48
  controls.enableZoom = false

  // 夜面保留一层低强度冷色环境光，让海陆轮廓可读但不会像白昼一样明亮。
  scene.add(new THREE.AmbientLight(0x315873, 1.08))
  sunLight = new THREE.DirectionalLight(0xfff3dd, 3.1)
  scene.add(sunLight)

  const loader = new THREE.TextureLoader()
  const material = new THREE.MeshPhongMaterial({ color: 0x244a63, shininess: 7, specular: 0x17364b })
  earth = new THREE.Mesh(new THREE.SphereGeometry(EARTH_RADIUS, 128, 128), material)
  earthSystemGroup.add(earth)
  loader.load(
    'https://unpkg.com/three-globe@2.45.2/example/img/earth-blue-marble.jpg',
    (texture) => {
      texture.colorSpace = THREE.SRGBColorSpace
      material.map = texture
      material.color.set(0xffffff)
      material.needsUpdate = true
      if (nightLightsMaterial) nightLightsMaterial.uniforms.surfaceMap.value = texture
      textureState.value = 'ready'
    },
    undefined,
    () => { textureState.value = 'fallback' },
  )

  nightLightsMaterial = new THREE.ShaderMaterial({
    uniforms: {
      surfaceMap: { value: null },
      nightMap: { value: null },
      sunDirection: { value: new THREE.Vector3(0, 0, 1) },
    },
    transparent: true,
    depthWrite: false,
    vertexShader: `
      varying vec2 vUv;
      varying vec3 vWorldNormal;
      void main() {
        vUv = uv;
        vWorldNormal = normalize(mat3(modelMatrix) * normal);
        gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
      }
    `,
    fragmentShader: `
      uniform sampler2D surfaceMap;
      uniform sampler2D nightMap;
      uniform vec3 sunDirection;
      varying vec2 vUv;
      varying vec3 vWorldNormal;
      void main() {
        vec3 surface = texture2D(surfaceMap, vUv).rgb;
        vec3 lights = texture2D(nightMap, vUv).rgb;
        float solar = dot(normalize(vWorldNormal), normalize(sunDirection));
        float night = 1.0 - smoothstep(-0.22, 0.10, solar);
        float energy = max(lights.r, max(lights.g, lights.b));
        float cityMask = smoothstep(0.07, 0.46, energy);
        vec3 geography = surface * vec3(0.30, 0.39, 0.53);
        vec3 cityLights = lights * cityMask * 1.55;
        gl_FragColor = vec4(geography + cityLights, night * 0.86);
      }
    `,
  })
  const nightLights = new THREE.Mesh(
    new THREE.SphereGeometry(EARTH_RADIUS * 1.0015, 128, 128),
    nightLightsMaterial,
  )
  earthSystemGroup.add(nightLights)
  loader.load(
    'https://unpkg.com/three-globe@2.45.2/example/img/earth-night.jpg',
    (texture) => {
      texture.colorSpace = THREE.SRGBColorSpace
      if (nightLightsMaterial) nightLightsMaterial.uniforms.nightMap.value = texture
    },
  )
  updateSun(new Date())

  const atmosphere = new THREE.Mesh(
    new THREE.SphereGeometry(EARTH_RADIUS * 1.035, 96, 96),
    new THREE.ShaderMaterial({
      transparent: true,
      side: THREE.BackSide,
      blending: THREE.AdditiveBlending,
      vertexShader: `varying vec3 vNormal; void main(){vNormal=normalize(normalMatrix*normal);gl_Position=projectionMatrix*modelViewMatrix*vec4(position,1.0);}`,
      fragmentShader: `varying vec3 vNormal; void main(){float i=pow(0.72-dot(vNormal,vec3(0.0,0.0,1.0)),3.0);gl_FragColor=vec4(0.18,0.65,1.0,1.0)*i;}`,
    }),
  )
  earthSystemGroup.add(atmosphere)

  const axisGuide = new THREE.Line(
    new THREE.BufferGeometry().setFromPoints([
      new THREE.Vector3(0, -EARTH_RADIUS * 1.38, 0),
      new THREE.Vector3(0, EARTH_RADIUS * 1.38, 0),
    ]),
    new THREE.LineBasicMaterial({ color: 0x72d7ff, transparent: true, opacity: 0.58 }),
  )
  axisGuide.name = 'earth-rotation-axis'
  earthSystemGroup.add(axisGuide)
  for (const pole of [-1, 1]) {
    const poleTip = new THREE.Mesh(
      new THREE.SphereGeometry(0.027, 12, 12),
      new THREE.MeshBasicMaterial({ color: 0x72d7ff, transparent: true, opacity: 0.82 }),
    )
    poleTip.position.set(0, pole * EARTH_RADIUS * 1.38, 0)
    earthSystemGroup.add(poleTip)
  }

  const eclipticPoints: THREE.Vector3[] = []
  const eclipticRadius = EARTH_RADIUS * 1.43
  for (let index = 0; index < 180; index += 1) {
    const angle = (index / 180) * Math.PI * 2
    eclipticPoints.push(new THREE.Vector3(Math.cos(angle) * eclipticRadius, 0, Math.sin(angle) * eclipticRadius))
  }
  const eclipticGuide = new THREE.LineLoop(
    new THREE.BufferGeometry().setFromPoints(eclipticPoints),
    new THREE.LineDashedMaterial({ color: 0xffb866, transparent: true, opacity: 0.16, dashSize: 0.11, gapSize: 0.09 }),
  )
  eclipticGuide.name = 'ecliptic-reference-plane'
  eclipticGuide.computeLineDistances()
  scene.add(eclipticGuide)

  const starGeometry = new THREE.BufferGeometry()
  const starData: number[] = []
  for (let index = 0; index < 1000; index += 1) {
    const radius = 32 + Math.random() * 45
    const theta = Math.random() * Math.PI * 2
    const phi = Math.acos(2 * Math.random() - 1)
    starData.push(radius * Math.sin(phi) * Math.cos(theta), radius * Math.cos(phi), radius * Math.sin(phi) * Math.sin(theta))
  }
  starGeometry.setAttribute('position', new THREE.Float32BufferAttribute(starData, 3))
  scene.add(new THREE.Points(starGeometry, new THREE.PointsMaterial({ color: 0xa6c9df, size: 0.035, transparent: true, opacity: 0.58 })))

  renderer.domElement.addEventListener('pointerdown', onPointerDown)
  renderer.domElement.addEventListener('pointerup', onPointerUp)
  renderer.domElement.addEventListener('pointermove', onPointerMove)
  renderer.domElement.addEventListener('pointerleave', onPointerLeave)
  renderer.domElement.addEventListener('wheel', onSceneWheel, { passive: false })
  resizeObserver = new ResizeObserver(resize)
  resizeObserver.observe(host)
  rebuildDataLayers()
  rebuildObserverMarker()
  beginFocus()
  animate()
}

function beginFocus() {
  if (!camera || !props.focusTarget) return
  const distance = THREE.MathUtils.clamp(
    props.focusTarget.distance ?? camera.position.length(),
    controls?.minDistance ?? 4.4,
    controls?.maxDistance ?? 12,
  )
  const targetDirection = latLonToVector(
    props.focusTarget.latitude,
    props.focusTarget.longitude,
    1,
  ).normalize().applyQuaternion(earthSystemGroup?.quaternion ?? EARTH_TILT)
  focusAnimation = {
    from: camera.position.clone(),
    to: targetDirection.multiplyScalar(distance),
    startedAt: performance.now(),
    duration: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 1 : 950,
  }
  if (controls) controls.enabled = false
}

function updateSun(date: Date) {
  if (!sunLight) return
  const yearStart = Date.UTC(date.getUTCFullYear(), 0, 0)
  const dayOfYear = Math.floor((date.getTime() - yearStart) / 86_400_000)
  const declination = 23.44 * Math.sin(THREE.MathUtils.degToRad((360 / 365) * (dayOfYear - 81)))
  const utcHours = date.getUTCHours() + date.getUTCMinutes() / 60 + date.getUTCSeconds() / 3600
  const subsolarLongitude = 180 - utcHours * 15
  const localSunDirection = latLonToVector(declination, subsolarLongitude, 12)
  localSunDirection.applyQuaternion(earthSystemGroup?.quaternion ?? EARTH_TILT)
  sunLight.position.copy(localSunDirection)
  nightLightsMaterial?.uniforms.sunDirection.value.copy(sunLight.position).normalize()
}

function resize() {
  const host = canvasHost.value
  if (!host || !renderer || !camera) return
  camera.aspect = host.clientWidth / host.clientHeight
  camera.updateProjectionMatrix()
  renderer.setSize(host.clientWidth, host.clientHeight)
}

function onPointerDown(event: PointerEvent) {
  pointerStart.set(event.clientX, event.clientY)
}

function isNearEarth(clientX: number, clientY: number) {
  if (!renderer || !camera) return false
  const bounds = renderer.domElement.getBoundingClientRect()
  const projectedCenter = new THREE.Vector3(0, 0, 0).project(camera)
  const cameraRight = new THREE.Vector3(1, 0, 0)
    .applyQuaternion(camera.quaternion)
    .multiplyScalar(EARTH_RADIUS * 1.08)
    .project(camera)
  const centerX = bounds.left + (projectedCenter.x * 0.5 + 0.5) * bounds.width
  const centerY = bounds.top + (-projectedCenter.y * 0.5 + 0.5) * bounds.height
  const radius = Math.abs(cameraRight.x - projectedCenter.x) * bounds.width * 0.5
  return Math.hypot(clientX - centerX, clientY - centerY) <= radius * 1.12
}

function onPointerMove(event: PointerEvent) {
  pointerNearEarth.value = isNearEarth(event.clientX, event.clientY)
}

function onPointerLeave() {
  pointerNearEarth.value = false
}

function onSceneWheel(event: WheelEvent) {
  if (!camera || !controls || !isNearEarth(event.clientX, event.clientY)) return
  event.preventDefault()
  focusAnimation = undefined
  controls.enabled = true
  const normalizedDelta = event.deltaMode === WheelEvent.DOM_DELTA_LINE ? event.deltaY * 16 : event.deltaY
  const nextDistance = THREE.MathUtils.clamp(
    camera.position.length() * Math.exp(normalizedDelta * 0.0012),
    controls.minDistance,
    controls.maxDistance,
  )
  camera.position.setLength(nextDistance)
  controls.update()
}

function onPointerUp(event: PointerEvent) {
  if (!renderer || !camera || pointerStart.distanceTo(new THREE.Vector2(event.clientX, event.clientY)) > 5) return
  const bounds = renderer.domElement.getBoundingClientRect()
  pointer.set(((event.clientX - bounds.left) / bounds.width) * 2 - 1, -((event.clientY - bounds.top) / bounds.height) * 2 + 1)
  raycaster.setFromCamera(pointer, camera)
  const hits = raycaster.intersectObjects([...markerObjects.values()])
  const target = hits[0]?.object.userData as { kind?: 'spacecraft' | 'site'; id?: string }
  if (target?.kind && target.id) emit('select', { kind: target.kind, id: target.id })
}

function animate(time = 0) {
  frameId = requestAnimationFrame(animate)
  if (focusAnimation && camera) {
    const progress = Math.min(1, (time - focusAnimation.startedAt) / focusAnimation.duration)
    const eased = 1 - Math.pow(1 - progress, 3)
    const distance = THREE.MathUtils.lerp(focusAnimation.from.length(), focusAnimation.to.length(), eased)
    camera.position
      .copy(focusAnimation.from)
      .normalize()
      .lerp(focusAnimation.to.clone().normalize(), eased)
      .normalize()
      .multiplyScalar(distance)
    camera.lookAt(0, 0, 0)
    if (progress >= 1) {
      focusAnimation = undefined
      if (controls) {
        controls.enabled = true
        controls.update()
      }
    }
  } else {
    controls?.update()
  }
  if (time - lastOrbitUpdate > 1000) {
    updateSpacecraftPositions(new Date())
    lastOrbitUpdate = time
  }
  if (time - lastSunUpdate > 60_000) {
    updateSun(new Date())
    lastSunUpdate = time
  }
  spacecraftGroup?.traverse((item) => {
    if (item instanceof THREE.Mesh && item.geometry.type === 'RingGeometry' && camera) item.lookAt(camera.position)
  })
  observerMarker?.traverse((item) => {
    if (item instanceof THREE.Mesh && item.geometry.type === 'RingGeometry' && camera) item.lookAt(camera.position)
  })
  updateLabels()
  if (scene && camera && renderer) renderer.render(scene, camera)
}

watch(() => [props.spacecraft, props.sites], async () => {
  await nextTick()
  rebuildDataLayers()
}, { deep: true })

watch(() => props.layers, () => {
  if (spacecraftGroup) spacecraftGroup.visible = props.layers.spacecraft
  if (orbitGroup) orbitGroup.visible = props.layers.orbits
  if (siteGroup) siteGroup.visible = props.layers.sites
}, { deep: true })

watch(selectionKey, rebuildDataLayers)
watch(() => props.focusTarget?.key, beginFocus)
watch(() => props.observerTarget, rebuildObserverMarker, { deep: true })

onMounted(setupScene)
onBeforeUnmount(() => {
  cancelAnimationFrame(frameId)
  resizeObserver?.disconnect()
  if (renderer) {
    renderer.domElement.removeEventListener('pointerdown', onPointerDown)
    renderer.domElement.removeEventListener('pointerup', onPointerUp)
    renderer.domElement.removeEventListener('pointermove', onPointerMove)
    renderer.domElement.removeEventListener('pointerleave', onPointerLeave)
    renderer.domElement.removeEventListener('wheel', onSceneWheel)
    renderer.dispose()
    renderer.domElement.remove()
  }
  controls?.dispose()
})
</script>

<template>
  <div ref="canvasHost" class="scene-host" :class="{ 'pointer-near-earth': pointerNearEarth }" aria-label="可拖动的三维地球轨道场景">
    <button
      v-for="label in labels"
      v-show="label.visible"
      :key="`${label.kind}:${label.id}`"
      class="scene-label"
      :class="[label.kind, { selected: selectionKey === `${label.kind}:${label.id}` }]"
      :style="{ transform: `translate(${label.x}px, ${label.y}px)` }"
      @click="emit('select', { kind: label.kind, id: label.id })"
    >
      <i />{{ label.name }}
    </button>
    <div class="scene-guide">
      <span>拖动旋转</span><span>地球附近滚轮缩放</span><span>轨道高度 ×3.2</span>
    </div>
    <div
      v-if="observerLabel"
      v-show="observerLabel.visible"
      class="scene-observer-label"
      :style="{ transform: `translate(${observerLabel.x}px, ${observerLabel.y}px)` }"
    >
      <i />{{ observerLabel.name }}
    </div>
    <div v-if="textureState === 'fallback'" class="texture-warning">地表影像未加载，已切换基础材质</div>
  </div>
</template>

<style scoped>
.scene-host { position: absolute; inset: 0; overflow: hidden; cursor: default; }
.scene-host.pointer-near-earth { cursor: grab; }
.scene-host.pointer-near-earth:active { cursor: grabbing; }
.scene-host::after { content: ''; position: absolute; inset: 0; pointer-events: none; background: radial-gradient(circle at 50% 48%, transparent 26%, rgba(3, 7, 12, .13) 58%, rgba(3, 7, 12, .68) 100%); }
.scene-label { position: absolute; left: 0; top: 0; z-index: 3; display: flex; gap: 7px; align-items: center; padding: 5px 8px; border: 1px solid rgba(124, 184, 216, .22); background: rgba(3, 10, 17, .74); color: #bfd1dc; font: 500 10px/1.2 var(--font-sans); letter-spacing: .04em; white-space: nowrap; backdrop-filter: blur(8px); cursor: pointer; transition: border-color .2s, color .2s; }
.scene-label::before { content: ''; position: absolute; right: 100%; top: 50%; width: 14px; height: 1px; background: rgba(120, 188, 222, .35); }
.scene-label i { width: 4px; height: 4px; border-radius: 50%; background: #72d7ff; box-shadow: 0 0 8px #72d7ff; }
.scene-label.site i { background: #ffb866; box-shadow: 0 0 8px #ffb866; }
.scene-label.selected { color: #fff; border-color: rgba(114, 215, 255, .72); }
.scene-observer-label { position: absolute; left: 0; top: 0; z-index: 3; display: flex; align-items: center; gap: 7px; padding: 5px 8px; border: 1px solid rgba(121, 227, 189, .34); background: rgba(3, 10, 17, .78); color: #c7eee1; font: 500 10px/1.2 var(--font-sans); white-space: nowrap; pointer-events: none; backdrop-filter: blur(8px); }
.scene-observer-label::before { content: ''; position: absolute; right: 100%; top: 50%; width: 14px; height: 1px; background: rgba(121, 227, 189, .4); }
.scene-observer-label i { width: 5px; height: 5px; border-radius: 50%; background: #79e3bd; box-shadow: 0 0 8px rgba(121, 227, 189, .65); }
.scene-guide { position: absolute; z-index: 3; bottom: 22px; left: 50%; transform: translateX(-50%); display: flex; gap: 18px; color: rgba(169, 192, 206, .62); font: 500 9px/1 var(--font-mono); letter-spacing: .1em; text-transform: uppercase; pointer-events: none; }
.scene-guide span + span::before { content: '·'; margin-right: 18px; }
.texture-warning { position: absolute; z-index: 4; top: 82px; left: 50%; transform: translateX(-50%); color: #e6b985; font: 11px var(--font-mono); }
</style>
