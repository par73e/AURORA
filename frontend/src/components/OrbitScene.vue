<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as THREE from 'three'
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'
import type { LaunchSite, SceneLayers, Selection, Spacecraft } from '../types'
import { EARTH_DAY_TEXTURE_URL, EARTH_NIGHT_TEXTURE_URL, EARTH_RADIUS, latLonToVector, sampleOrbit, spacecraftPoint } from '../orbit/coordinates'

const EARTH_AXIAL_TILT_DEGREES = 23.44
/** 地球入场自转（先做地球，月球后续再说）：
 *  挂载即开始绕自转轴匀速转（黑幕期间用户看不到起点，渐亮时已在转），
 *  渐亮结束 + 停前等待后快速停下（400ms 线性匀减速，干脆不拖沓）。
 *  自东向西（从北极俯视顺时针，rotation.y 递减）；真实地球自西向东，方向不符可翻转符号 */
const SPIN_ANGULAR_SPEED = -(Math.PI * 2) / 30 // 30s/圈 ≈ 12°/s，自东向西（快速）
const SPIN_DECEL_DURATION_MS = 400 // 匀减速段：速度从 ω 线性降到 0（全程线性，无突快突慢）
/** 匀减速段的总位移 = |ω|·T/2；角度到达该值时开始减速 → 终点精确落在 0°（南海正中） */
const SPIN_DECEL_SWEEP = (Math.abs(SPIN_ANGULAR_SPEED) * SPIN_DECEL_DURATION_MS) / 2000
/** 挂载时南海的预设偏角：黑幕中先把南海从中心转开 +15°，
 *  随后匀速自东向西转，转到剩 SPIN_DECEL_SWEEP 时线性匀减速，终点恰好 0°（南海正中） */
const SPIN_INITIAL_OFFSET = THREE.MathUtils.degToRad(15)
const EARTH_TILT = new THREE.Quaternion().setFromAxisAngle(
  new THREE.Vector3(0, 0, 1),
  THREE.MathUtils.degToRad(EARTH_AXIAL_TILT_DEGREES),
)

const props = defineProps<{
  spacecraft: Spacecraft[]
  sites: LaunchSite[]
  layers: SceneLayers
  selection: Selection | null
  headerExpanded?: boolean
  focusTarget?: { latitude: number; longitude: number; distance?: number; key: string } | null
  observerTarget?: { latitude: number; longitude: number; label: string } | null
  observerActive?: boolean
  dayNightEnabled?: boolean
  revealTick?: number
  /** 离开信号：所有多余元素（自转轴/轨道/航天器/发射场/观测标记）统一淡出，只留裸地球 */
  leaving?: boolean
}>()

const emit = defineEmits<{
  select: [selection: Selection]
  'view-change': []
  'blank-click': []
  /** 场景首帧贴图渲染完成（解码 + GPU 上传后）——过渡遮罩等待此信号再揭示 */
  'textures-ready': []
  /** 面板关闭（同步 App 的 selection） */
  'clear-selection': []
}>()

let texturesReadySent = false
/** 纹理上传完成信号：双 rAF（等 renderer.render 真正把贴图传到 GPU 之后） */
function emitTexturesReady() {
  if (texturesReadySent) return
  texturesReadySent = true
  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      emit('textures-ready')
    })
  })
}

const canvasHost = ref<HTMLDivElement | null>(null)
const labels = ref<Array<{ id: string; kind: 'spacecraft' | 'site'; name: string; x: number; y: number; visible: boolean }>>([])
const observerLabel = ref<{ name: string; x: number; y: number; visible: boolean } | null>(null)
/** 入场渐亮：进入边界（revealTick 递增）时置 true，0.2s 过渡；直接加载默认已亮 */
const sceneRevealed = ref(!props.revealTick)
/** 分阶段揭示是否已排程（revealTick 递增时才启动——与月球同基准：渐亮开始时计时） */
let revealScheduled = false
/** revealTick 最近一次递增的时刻——所有入场任务（旋转停止/3D 弹出）的统一计时基准。
 *  兜底路径（挂载时 revealTick 已非 0）也用它，避免从挂载时刻起算导致元素提前入场 */
let revealTickAt = 0
watch(
  () => props.revealTick,
  (tick) => {
    if (tick) {
      sceneRevealed.value = true
      revealTickAt = performance.now()
    }
    // axisGuide 等场景对象在 onMounted 构建——watch 可能早于构建触发（首次进入路径），
    // 提前调用 scheduleRevealLayers 会 ReferenceError 并损坏渲染器（信息栏不弹的根因）
    if (tick && !revealScheduled && axisGuide) {
      revealScheduled = true
      scheduleRevealLayers()
    }
  },
)
/** 离开：全部多余元素 250ms 一次性淡出，只留裸地球（随后由 App 遮罩渐暗切页）；
 *  离开被中止（hash 守卫失败）时 leaving 回 false → 恢复到淡出前状态 */
watch(
  () => props.leaving,
  (leaving) => {
    const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    if (leaving) {
      elementsLeavingAt = performance.now()
      scheduleHideAll(reduced ? 1 : 250)
    } else {
      scheduleRestoreAll(reduced ? 1 : 250)
    }
  },
)
/** 本地选中状态：面板渲染只依赖它（与 App 全局 selection 解耦——参照月球组件内面板架构，
 *  避免 App 渲染异常时信息栏不弹） */
const localSelection = ref<Selection | null>(null)
const showEventOriginal = ref(false)
watch(
  () => props.selection,
  (next) => {
    if (next) localSelection.value = next // App 驱动（下方列表点击）→ 同步本地
  },
  { immediate: true },
)
const selectedSpacecraft = computed(() =>
  localSelection.value?.kind === 'spacecraft' ? props.spacecraft.find((item) => item.id === localSelection.value?.id) : undefined,
)
const selectedSite = computed(() =>
  localSelection.value?.kind === 'site' ? props.sites.find((item) => item.id === localSelection.value?.id) : undefined,
)
const selectedEvent = computed(() =>
  localSelection.value?.kind === 'event' ? props.events.find((item) => item.externalId === localSelection.value?.id) : undefined,
)
const selectedEventSite = computed(() => (selectedEvent.value ? nearestSite(selectedEvent.value) : undefined))
watch(() => selectedEvent.value?.externalId, () => {
  showEventOriginal.value = false
})
function nearestSite(event: LaunchEvent): LaunchSite | undefined {
  if (event.latitude == null || event.longitude == null) return undefined
  let best: LaunchSite | undefined
  let bestDistance = Infinity
  for (const site of props.sites) {
    const distance = (site.latitude - event.latitude) ** 2 + (site.longitude - event.longitude) ** 2
    if (distance < bestDistance) {
      bestDistance = distance
      best = site
    }
  }
  return best
}
function orbitPeriodText(meanMotion: string) {
  const mm = Number(meanMotion)
  if (!Number.isFinite(mm) || mm <= 0) return '—'
  return `${(1440 / mm).toFixed(1)} 分钟`
}
function formatCoordinate(value: number, positive: string, negative: string) {
  return `${Math.abs(value).toFixed(2)}° ${value >= 0 ? positive : negative}`
}
function eventDate(value: string) {
  const date = new Date(value)
  return {
    day: new Intl.DateTimeFormat('zh-CN', { day: '2-digit', timeZone: 'Asia/Shanghai' }).format(date),
    month: new Intl.DateTimeFormat('en-US', { month: 'short', timeZone: 'Asia/Shanghai' }).format(date).toUpperCase(),
    time: new Intl.DateTimeFormat('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false, timeZone: 'Asia/Shanghai' }).format(date),
  }
}
function launchVehicleName(event: LaunchEvent) {
  return event.name.split(' | ')[0]?.trim() || event.providerName || '运载火箭待确认'
}
const textureState = ref<'loading' | 'ready' | 'fallback'>('loading')
const pointerNearEarth = ref(false)

let renderer: THREE.WebGLRenderer | undefined
let scene: THREE.Scene | undefined
let camera: THREE.PerspectiveCamera | undefined
let controls: OrbitControls | undefined
let frameId = 0
let resizeObserver: ResizeObserver | undefined
let earthSystemGroup: THREE.Group | undefined
/** 自转参考系：所有经纬度定位对象（地表/夜面灯光/大气/发射场/航天器/轨道线/观测标记）
 *  整体绕自转轴（earthSystemGroup 局部 Y）旋转——相对关系不变 */
let spinGroup: THREE.Group | undefined
/** 减弱动态效果下不转（与太阳系 timeScale 同策略） */
const spinReduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
/** 入场自转状态机：spin（匀速）→ stop（线性匀减速）→ done（停住） */
let spinPhase: 'spin' | 'stop' | 'done' = spinReduced ? 'done' : 'spin'
let spinStartAt = 0
let spinStopAt = 0 // 匀减速开始时刻（角度到达 SPIN_DECEL_SWEEP 时触发）
let spinStopFrom = 0 // 匀减速起点角度
/** 元素入场揭示延迟（旋转 1.45s 停住 + ~50ms 缓冲） */
const ELEMENTS_REVEAL_DELAY_MS = 1500
/** 元素揭示是否已完成（进入时 false，全部淡入任务完成后 true；直接加载默认 true） */
let elementsShown = true
/** 退出淡出开始时刻（leaving 置 true 时记录，用于每帧元素可见度计算） */
let elementsLeavingAt = 0
/** 元素整体可见度（0..1）：observerMarker 每帧强制应用，任何重建都无法绕过隐藏。
 *  进入：revealTickAt 起延迟 1500ms 后 300ms 淡入到 1；退出：250ms 淡出到 0；
 *  直接加载（revealTickAt=0）/reduced：恒 1 */
function elementsFadeNow(now = performance.now()): number {
  if (spinReduced || revealTickAt === 0) return 1
  if (props.leaving) return THREE.MathUtils.clamp(1 - (now - elementsLeavingAt) / 250, 0, 1)
  return THREE.MathUtils.clamp((now - revealTickAt - ELEMENTS_REVEAL_DELAY_MS) / 300, 0, 1)
}
let axisGuide: THREE.Line | undefined
let poleTips: THREE.Mesh[] = []
let eclipticGuide: THREE.LineLoop | undefined
let spacecraftGroup: THREE.Group | undefined
let orbitGroup: THREE.Group | undefined
let siteGroup: THREE.Group | undefined
let observerMarker: THREE.Group | undefined
let earth: THREE.Mesh | undefined
let earthDayMaterial: THREE.MeshPhongMaterial | undefined
let nightLights: THREE.Mesh | undefined
let ambientLight: THREE.AmbientLight | undefined
let observationLight: THREE.DirectionalLight | undefined
let sunLight: THREE.DirectionalLight | undefined
let nightLightsMaterial: THREE.ShaderMaterial | undefined
const markerObjects = new Map<string, THREE.Object3D>()
const raycaster = new THREE.Raycaster()
/** 标记点距离补偿临时向量（每帧复用，避免分配） */
const markerScaleTmp = new THREE.Vector3()
const earthOcclusionSphere = new THREE.Sphere(new THREE.Vector3(0, 0, 0), EARTH_RADIUS * 1.004)
const earthOcclusionRay = new THREE.Ray()
const earthOcclusionHit = new THREE.Vector3()
const toOcclusionTarget = new THREE.Vector3()
const pointer = new THREE.Vector2()
const pointerStart = new THREE.Vector2()
let pointerViewChangeAnnounced = false
let lastOrbitUpdate = 0
let lastSunUpdate = 0
let focusAnimation: {
  from: THREE.Vector3
  to: THREE.Vector3
  startedAt: number
  duration: number
} | undefined

// ---- 统一入场/退出：裸地球先 0.3s 渐入（scene-host），随后全部元素一次性 0.3s 淡入；
//      退出时全部元素一次性淡出，只留裸地球（由 App 的遮罩完成星球渐暗） ----
interface RevealEntry {
  material: THREE.Material
  from: number
  to: number
  restoreTransparent: boolean
}
interface RevealTask {
  started: number
  delay: number
  duration: number
  entries: RevealEntry[]
}
let revealTasks: RevealTask[] = []

/** 把对象的全部材质透明度归零，并安排 delay 后开始、duration 内淡入到原值。
 *  计时基准 = revealTick 递增时刻（revealTickAt）——与旋转停止/App 的 DOM 弹出同一时钟 */
function scheduleReveal(object: THREE.Object3D, delay: number, duration: number) {
  const entries: RevealEntry[] = []
  object.traverse((child) => {
    const material = (child as THREE.Mesh).material
    if (!material) return
    const list = Array.isArray(material) ? material : [material]
    for (const item of list) {
      const restore = !item.transparent
      const to = item.opacity
      if (restore) item.transparent = true
      item.opacity = 0
      entries.push({ material: item, from: 0, to, restoreTransparent: restore })
    }
  })
  if (entries.length > 0) revealTasks.push({ started: revealTickAt, delay, duration, entries })
}

/** 全部多余元素一次性淡出（leaving 时调用）：只留裸地球。
 *  同时取消未完成的入场任务（防止进入中途退出时元素被拉回）并快照各材质淡出前的状态，供中止恢复 */
let hideSnapshot: { material: THREE.Material; from: number; restoreTransparent: boolean }[] | null = null
function scheduleHideAll(duration: number) {
  // 未完成入场任务的"目标透明度"：进入中途退出后若中止恢复，应回到完整目标而非中途值
  const revealTo = new Map<THREE.Material, number>()
  for (const task of revealTasks) {
    for (const entry of task.entries) revealTo.set(entry.material, entry.to)
  }
  revealTasks = []
  hideSnapshot = []
  const entries: RevealEntry[] = []
  const hide = (object: THREE.Object3D | undefined) => {
    if (!object) return
    object.traverse((child) => {
      const material = (child as THREE.Mesh).material
      if (!material) return
      const list = Array.isArray(material) ? material : [material]
      for (const item of list) {
        const wasTransparent = item.transparent
        if (!wasTransparent) item.transparent = true
        const from = revealTo.get(item) ?? item.opacity
        hideSnapshot.push({ material: item, from, restoreTransparent: !wasTransparent })
        entries.push({ material: item, from, to: 0, restoreTransparent: false })
      }
    })
  }
  hide(axisGuide)
  for (const tip of poleTips) hide(tip)
  hide(eclipticGuide)
  hide(orbitGroup)
  hide(spacecraftGroup)
  hide(siteGroup)
  hide(observerMarker)
  if (entries.length > 0) revealTasks.push({ started: performance.now(), delay: 0, duration, entries })
}

/** 退出被中止（hash 守卫失败等）：把元素恢复到淡出前的状态（从当前透明度平滑过渡） */
function scheduleRestoreAll(duration: number) {
  if (!hideSnapshot || hideSnapshot.length === 0) return
  revealTasks = []
  const now = performance.now()
  const entries: RevealEntry[] = hideSnapshot.map((snap) => ({
    material: snap.material,
    from: snap.material.opacity, // 当前值（可能正处于淡出中途，避免跳变）
    to: snap.from,
    restoreTransparent: snap.restoreTransparent,
  }))
  hideSnapshot = null
  revealTasks.push({ started: now, delay: 0, duration, entries })
}

/** 统一入场（revealTick 递增时调用）：裸地球先 0.3s 渐入（scene-host）并线性旋转至完全停住
 *  （≈1.45s），再缓冲 ~50ms 后自转轴/轨道/航天器/发射场/观测标记一次性 0.3s 淡入——
 *  元素弹出严格发生在旋转静止之后。与月球页基准一致：从"遮罩渐亮开始"计时 */
function scheduleRevealLayers() {
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  const delay = reduced ? 0 : ELEMENTS_REVEAL_DELAY_MS // 旋转 1.45s 停住 + ~50ms 静止缓冲（展示节奏稍快，仍保证停稳后弹出）
  const duration = reduced ? 1 : 300
  elementsShown = false // 进入揭示期：标签隐藏，3D 元素归零待淡入
  // 标签与 3D 淡入同刻出现：在淡入开始（delay）时置 true，而非淡入完成（delay+duration）后——
  // 否则标签会比点晚 300ms 出现
  window.setTimeout(() => {
    elementsShown = true
  }, delay)
  scheduleReveal(axisGuide, delay, duration)
  for (const tip of poleTips) scheduleReveal(tip, delay, duration)
  scheduleReveal(eclipticGuide, delay, duration)
  if (orbitGroup) scheduleReveal(orbitGroup, delay, duration)
  if (spacecraftGroup) scheduleReveal(spacecraftGroup, delay, duration)
  if (siteGroup) scheduleReveal(siteGroup, delay, duration)
  if (observerMarker) scheduleReveal(observerMarker, delay, duration)
}

function updateReveals() {
  if (revealTasks.length === 0) return
  const now = performance.now()
  for (const task of revealTasks) {
    const t = THREE.MathUtils.clamp((now - task.started - task.delay) / task.duration, 0, 1)
    const eased = 1 - Math.pow(1 - t, 3)
    for (const entry of task.entries) {
      entry.material.opacity = entry.from + (entry.to - entry.from) * eased
      if (t >= 1 && entry.restoreTransparent) entry.material.transparent = false
    }
  }
  const before = revealTasks.length
  revealTasks = revealTasks.filter((task) => now < task.started + task.delay + task.duration)
  void before // 标签出现时机由 scheduleRevealLayers 的定时器控制（与 3D 淡入开始同步），不再依赖任务完成时刻
}

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
  if (!earthSystemGroup || !spinGroup) return
  markerObjects.clear()
  disposeGroup(spacecraftGroup)
  disposeGroup(orbitGroup)
  disposeGroup(siteGroup)

  spacecraftGroup = new THREE.Group()
  orbitGroup = new THREE.Group()
  siteGroup = new THREE.Group()
  spinGroup.add(spacecraftGroup, orbitGroup, siteGroup)

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
  const active = props.observerActive !== false
  // 显现时保持真实可辨的颜色（inactive 不再淡到 0.28）；active/inactive 仍略有区分
  const basePoint = active ? 1 : 0.9
  const baseRing = active ? 0.55 : 0.5

  const point = new THREE.Mesh(
    new THREE.SphereGeometry(0.022, 18, 18),
    new THREE.MeshBasicMaterial({ color: 0x79e3bd, transparent: true, opacity: basePoint }),
  )
  point.material.userData.baseOpacity = basePoint
  const ring = new THREE.Mesh(
    new THREE.RingGeometry(0.039, 0.045, 32),
    new THREE.MeshBasicMaterial({ color: 0x79e3bd, transparent: true, opacity: baseRing, side: THREE.DoubleSide }),
  )
  ring.material.userData.baseOpacity = baseRing
  ring.lookAt(camera?.position ?? new THREE.Vector3(0, 0, 8))
  observerMarker.add(point, ring)
  spinGroup.add(observerMarker)
  // 重建瞬间同步当前元素可见度（下一帧起由 animate 每帧强制，任何时序都覆盖）
  const fade = elementsFadeNow()
  observerMarker.visible = fade > 0.001
  point.material.opacity = basePoint * fade
  ring.material.opacity = baseRing * fade
}

function applyDayNightMode() {
  const enabled = props.dayNightEnabled !== false
  if (earth && earthDayMaterial) {
    earth.material = earthDayMaterial
    earthDayMaterial.emissive.setHex(0x000000)
    earthDayMaterial.emissiveIntensity = 0
  }
  if (nightLights) nightLights.visible = enabled
  if (ambientLight) {
    ambientLight.color.setHex(0x315873)
    ambientLight.intensity = 1.08
  }
  if (observationLight) observationLight.intensity = enabled ? 0 : 3.1
  if (sunLight) sunLight.intensity = enabled ? 3.1 : 0
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
  // 自转参考系挂在倾斜参考系下：局部 Y = 自转轴（23.44° 倾角由父级承担）
  spinGroup = new THREE.Group()
  spinGroup.name = 'earth-spin-frame'
  earthSystemGroup.add(spinGroup)
  // 挂载即开始入场自转（黑幕期间已在转，渐亮时用户看到转动中段）；
  // 初始把南海从中心偏开 +15°（黑幕中不可见）——匀速转 + 线性匀减速，终点 0° 即南海正中；
  // 直接加载（无 revealTick）同样生效：角度到达减速点时自动匀减速停下。reduced-motion 不转，保持 0°
  spinStartAt = performance.now()
  if (!spinReduced) spinGroup.rotation.y = SPIN_INITIAL_OFFSET
  camera = new THREE.PerspectiveCamera(42, host.clientWidth / host.clientHeight, 0.1, 400) // far 400：容纳 60–150 星空壳层
  // 默认视角：对准东亚大陆，以南海为中心（约 12°N, 115°E）；
  // 自转轴仍保持黄道面参考的 23.44° 倾角（公转平面平行关系不变）
  const defaultDirection = latLonToVector(12, 115, 1)
    .normalize()
    .applyQuaternion(earthSystemGroup?.quaternion ?? EARTH_TILT)
  camera.position.copy(defaultDirection.multiplyScalar(7.6))

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
  controls.minDistance = 3.0 // 最大放大倍率：距地心 3.0（地表约 0.85，仍在大气层外）——原 4.4 放大空间太小
  controls.maxDistance = 12
  controls.rotateSpeed = 0.48
  controls.enableZoom = false

  // 夜面保留一层低强度冷色环境光，让海陆轮廓可读但不会像白昼一样明亮。
  ambientLight = new THREE.AmbientLight(0x315873, 1.08)
  scene.add(ambientLight)
  // 关闭晨昏线时让同色温白昼光跟随相机，明暗边界落在球体轮廓之外。
  observationLight = new THREE.DirectionalLight(0xfff3dd, 0)
  observationLight.position.copy(camera.position)
  scene.add(observationLight)
  sunLight = new THREE.DirectionalLight(0xfff3dd, 3.1)
  scene.add(sunLight)

  const loader = new THREE.TextureLoader()
  earthDayMaterial = new THREE.MeshPhongMaterial({ color: 0x244a63, shininess: 7, specular: 0x17364b })
  earth = new THREE.Mesh(
    new THREE.SphereGeometry(EARTH_RADIUS, 128, 128),
    earthDayMaterial,
  )
  // 纹理就绪前整个地球系统不可见（避免"裸水球"或"亮球"——大气辉光层在球体不可见时仍发光）；
  // 就绪瞬间整个系统（地球+大气+夜间层）一起出现
  earth.visible = false
  if (earthSystemGroup) earthSystemGroup.visible = false
  spinGroup.add(earth)
  loader.load(
    EARTH_DAY_TEXTURE_URL,
    (texture) => {
      texture.colorSpace = THREE.SRGBColorSpace
      if (earthDayMaterial) {
        earthDayMaterial.map = texture
        earthDayMaterial.color.set(0xffffff)
        earthDayMaterial.needsUpdate = true
      }
      if (nightLightsMaterial) nightLightsMaterial.uniforms.surfaceMap.value = texture
      textureState.value = 'ready'
      if (earth) earth.visible = true // 纹理就绪瞬间显示（黑屏后直接是带纹理的地球）
      if (earthSystemGroup) earthSystemGroup.visible = true // 大气/夜间层随地球一起出现
      emitTexturesReady()
    },
    undefined,
    () => {
      textureState.value = 'fallback'
      if (earth) earth.visible = true // 失败降级为纯色地球，不能永久隐藏
      if (earthSystemGroup) earthSystemGroup.visible = true
    },
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
  nightLights = new THREE.Mesh(
    new THREE.SphereGeometry(EARTH_RADIUS * 1.0015, 128, 128),
    nightLightsMaterial,
  )
  nightLights.visible = props.dayNightEnabled !== false
  spinGroup.add(nightLights)
  applyDayNightMode()
  loader.load(
    EARTH_NIGHT_TEXTURE_URL,
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
  spinGroup.add(atmosphere) // 同心壳层，随转无视觉差异

  axisGuide = new THREE.Line(
    new THREE.BufferGeometry().setFromPoints([
      new THREE.Vector3(0, -EARTH_RADIUS * 1.38, 0),
      new THREE.Vector3(0, EARTH_RADIUS * 1.38, 0),
    ]),
    new THREE.LineBasicMaterial({ color: 0x72d7ff, transparent: true, opacity: 0.58 }),
  )
  axisGuide.name = 'earth-rotation-axis'
  earthSystemGroup.add(axisGuide)
  poleTips = []
  for (const pole of [-1, 1]) {
    const poleTip = new THREE.Mesh(
      new THREE.SphereGeometry(0.027, 12, 12),
      new THREE.MeshBasicMaterial({ color: 0x72d7ff, transparent: true, opacity: 0.82 }),
    )
    poleTip.position.set(0, pole * EARTH_RADIUS * 1.38, 0)
    poleTips.push(poleTip)
    earthSystemGroup.add(poleTip)
  }

  const eclipticPoints: THREE.Vector3[] = []
  const eclipticRadius = EARTH_RADIUS * 1.43
  for (let index = 0; index < 180; index += 1) {
    const angle = (index / 180) * Math.PI * 2
    eclipticPoints.push(new THREE.Vector3(Math.cos(angle) * eclipticRadius, 0, Math.sin(angle) * eclipticRadius))
  }
  eclipticGuide = new THREE.LineLoop(
    new THREE.BufferGeometry().setFromPoints(eclipticPoints),
    new THREE.LineDashedMaterial({ color: 0xffb866, transparent: true, opacity: 0.16, dashSize: 0.11, gapSize: 0.09 }),
  )
  eclipticGuide.name = 'ecliptic-reference-plane'
  eclipticGuide.computeLineDistances()
  scene.add(eclipticGuide)

  // 星空粒子球（与月球同参数）：3000 颗、壳层 60–150、浅蓝主题色
  const starGeometry = new THREE.BufferGeometry()
  const starData: number[] = []
  for (let index = 0; index < 3000; index += 1) {
    const radius = 60 + Math.random() * 90
    const theta = Math.random() * Math.PI * 2
    const phi = Math.acos(2 * Math.random() - 1)
    starData.push(radius * Math.sin(phi) * Math.cos(theta), radius * Math.cos(phi), radius * Math.sin(phi) * Math.sin(theta))
  }
  starGeometry.setAttribute('position', new THREE.Float32BufferAttribute(starData, 3))
  // 背景星空挂在相机上：屏幕固定，不随星球/相机旋转（世界固定会有视差，看起来像跟着星球转）
  const starPoints = new THREE.Points(starGeometry, new THREE.PointsMaterial({ color: 0xb4d2e8, size: 0.15, transparent: true, opacity: 0.75 }))
  starPoints.name = 'background-stars'
  scene.add(camera)
  camera.add(starPoints)

  renderer.domElement.addEventListener('pointerdown', onPointerDown)
  renderer.domElement.addEventListener('pointerup', onPointerUp)
  renderer.domElement.addEventListener('pointermove', onPointerMove)
  renderer.domElement.addEventListener('pointerleave', onPointerLeave)
  renderer.domElement.addEventListener('wheel', onSceneWheel, { passive: false })
  resizeObserver = new ResizeObserver(resize)
  resizeObserver.observe(host)
  rebuildDataLayers()
  rebuildObserverMarker()
  // 分阶段入场：由 revealTick 递增触发（scheduleRevealLayers），与月球页同基准
  beginFocus()
  if (!props.focusTarget) {
    // 入场微转：无焦点目标时，从绕地球略微偏转的角度平滑回到默认视角（不硬切到当前位置）
    const defaultPosition = camera.position.clone()
    focusAnimation = {
      from: defaultPosition.clone().applyAxisAngle(new THREE.Vector3(0, 1, 0), -0.5),
      to: defaultPosition,
      startedAt: performance.now(),
      duration: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 1 : 1400,
    }
    if (controls) controls.enabled = false
  }
  animate()
  // 构建完成后的兜底排程：仅当 watch 已触发（revealTickAt > 0）但被 axisGuide 判空跳过时补上——
  // 用 revealTickAt 作计时基准，避免从挂载时刻起算导致元素提前入场；
  // 重新进入（挂载时 tick 非 0）时 watch 尚未触发，等 App 递增 revealTick 自然排程
  if (revealTickAt > 0 && !revealScheduled && axisGuide) {
    revealScheduled = true
    scheduleRevealLayers()
  }
}

function beginFocus() {
  if (!camera || !props.focusTarget) return
  const distance = THREE.MathUtils.clamp(
    props.focusTarget.distance ?? camera.position.length(),
    controls?.minDistance ?? 3.0,
    controls?.maxDistance ?? 12,
  )
  const targetDirection = latLonToVector(
    props.focusTarget.latitude,
    props.focusTarget.longitude,
    1,
  )
    .normalize()
    .applyQuaternion(earthSystemGroup?.quaternion ?? EARTH_TILT)
    // 不叠加自转相位：聚焦目标按自转终态（0° 南海相位）计算——
    // 自转进行中时 marker 与相机相向会合，自转结束即精确对准
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
  pointerViewChangeAnnounced = false
  // 按下瞬间：按在地球上 → 立即收起页头（拖拽中页头不应遮挡操作）
  if (isNearEarth(event.clientX, event.clientY)) emit('blank-click')
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
  if (event.buttons !== 0 && !pointerViewChangeAnnounced && pointerStart.distanceTo(new THREE.Vector2(event.clientX, event.clientY)) > 5) {
    pointerViewChangeAnnounced = true
    emit('view-change')
  }
}

function onPointerLeave() {
  pointerNearEarth.value = false
}

function onSceneWheel(event: WheelEvent) {
  if (!camera || !controls || !isNearEarth(event.clientX, event.clientY)) return
  event.preventDefault()
  emit('view-change')
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
  if (!renderer || !camera) return
  const dragged = pointerStart.distanceTo(new THREE.Vector2(event.clientX, event.clientY)) > 5
  if (dragged) return // 拖拽收起已在按下瞬间处理
  const bounds = renderer.domElement.getBoundingClientRect()
  pointer.set(((event.clientX - bounds.left) / bounds.width) * 2 - 1, -((event.clientY - bounds.top) / bounds.height) * 2 + 1)
  raycaster.setFromCamera(pointer, camera)
  const hits = raycaster.intersectObjects([...markerObjects.values()])
  const target = hits[0]?.object.userData as { kind?: 'spacecraft' | 'site'; id?: string }
  if (target?.kind && target.id) {
    localSelection.value = { kind: target.kind, id: target.id } // 本地立即驱动面板（不依赖 App 渲染）
    emit('select', { kind: target.kind, id: target.id })
    return
  }
  emit('blank-click')
}

function animate(time = 0) {
  frameId = requestAnimationFrame(animate)
  // 入场自转：挂载即从 +15° 匀速转（自东向西，全程线性）→ 角度剩 SPIN_DECEL_SWEEP 时
  // 线性匀减速（速度 ω→0）→ 终点精确 0°（南海正中）完全停住
  if (spinGroup && spinPhase !== 'done') {
    if (spinPhase === 'spin') {
      const t = Math.max(0, (time - spinStartAt) / 1000) // 首帧 time=0 时钳制为 0
      spinGroup.rotation.y = SPIN_INITIAL_OFFSET + SPIN_ANGULAR_SPEED * t
      // 角度锚定减速点：匀减速位移 = |ω|·T/2，此时开始减速，终点恰好落在 0°
      if (spinGroup.rotation.y <= SPIN_DECEL_SWEEP) {
        spinPhase = 'stop'
        spinStopAt = time
        spinStopFrom = spinGroup.rotation.y
      }
    } else {
      const t = Math.min(1, (time - spinStopAt) / SPIN_DECEL_DURATION_MS)
      // 线性匀减速位移积分：θ = θ₀ + ω·T·(t − t²/2)，速度 ω(1−t) 线性降到 0
      spinGroup.rotation.y = spinStopFrom + SPIN_ANGULAR_SPEED * (SPIN_DECEL_DURATION_MS / 1000) * (t - (t * t) / 2)
      if (t >= 1) spinPhase = 'done'
    }
  }
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
  // 动态拖动灵敏度：OrbitControls 每像素固定角度，相机距离近时同样角度在屏幕上的位移更大，
  // 操作显得过于灵敏——按距离反向补偿：最近处 0.2（约原 0.48 的 40%）、默认视角 7.6 处 ~0.46（手感不变）、最远处 0.7
  if (camera && controls && !focusAnimation) {
    const t = THREE.MathUtils.clamp(
      (camera.position.length() - controls.minDistance) / (controls.maxDistance - controls.minDistance),
      0,
      1,
    )
    controls.rotateSpeed = 0.2 + t * 0.5
  }
  if (time - lastOrbitUpdate > 1000) {
    updateSpacecraftPositions(new Date())
    lastOrbitUpdate = time
  }
  if (time - lastSunUpdate > 60_000) {
    updateSun(new Date())
    lastSunUpdate = time
  }
  // 标记点（航天器/发射场/坐标点）固定屏幕大小：世界尺寸 ∝ 到相机距离，
  // 补偿透视——不随地球/相机距离放大，始终像贴在地表上的固定大小物体
  if (camera) {
    const refDistance = 7.6 // 默认视角相机距离（scale = 1 的基准）
    for (const marker of markerObjects.values()) {
      const d = marker.getWorldPosition(markerScaleTmp).distanceTo(camera.position)
      marker.scale.setScalar(d / refDistance)
    }
    if (observerMarker) {
      const d = observerMarker.getWorldPosition(markerScaleTmp).distanceTo(camera.position)
      observerMarker.scale.setScalar(d / refDistance)
    }
  }
  observerMarker?.traverse((item) => {
    if (item instanceof THREE.Mesh && item.geometry.type === 'RingGeometry' && camera) item.lookAt(camera.position)
  })
  // observerMarker 每帧强制随元素整体可见度：进入隐藏期/淡入/稳态/退出淡出全部覆盖，
  // 任何时刻重建（定位回调/active 切换）的新材质都无法绕过隐藏。
  // visible 兜底：隐藏期直接不渲染该对象树（比 opacity 更彻底，任何材质写入都无法绕过）
  if (observerMarker) {
    const fade = elementsFadeNow(time)
    observerMarker.visible = fade > 0.001
    observerMarker.traverse((item) => {
      const material = (item as THREE.Mesh).material
      if (!material) return
      const list = Array.isArray(material) ? material : [material]
      for (const entry of list) {
        const base = (entry.userData.baseOpacity as number | undefined) ?? entry.opacity
        entry.opacity = base * fade
      }
    })
  }
  if (observationLight && camera && props.dayNightEnabled === false) {
    observationLight.position.copy(camera.position).normalize().multiplyScalar(12)
  }
  updateLabels()
  updateReveals()
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
watch(() => [props.observerTarget, props.observerActive], rebuildObserverMarker, { deep: true })
watch(() => props.dayNightEnabled, applyDayNightMode)

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
  <div ref="canvasHost" class="scene-host" :class="{ revealed: sceneRevealed, 'pointer-near-earth': pointerNearEarth }" aria-label="可拖动的三维地球轨道场景">
    <button
      v-for="label in labels"
      v-show="label.visible && elementsShown"
      :key="`${label.kind}:${label.id}`"
      class="scene-label"
      :class="[label.kind, { selected: selectionKey === `${label.kind}:${label.id}` }]"
      :style="{ transform: `translate(${label.x + 14}px, ${label.y - 11}px)` }"
      @click="localSelection = { kind: label.kind, id: label.id }; emit('select', { kind: label.kind, id: label.id })"
    >
      <i />{{ label.name }}
    </button>
    <div
      v-if="observerLabel"
      v-show="observerLabel.visible && elementsShown"
      class="scene-observer-label"
      :class="{ inactive: props.observerActive === false }"
      :style="{ transform: `translate(${observerLabel.x + 14}px, ${observerLabel.y - 11}px)` }"
    >
      <i />{{ observerLabel.name }}
    </div>
    <div v-if="textureState === 'fallback'" class="texture-warning">地表影像未加载，已切换基础材质</div>

    <!-- 信息面板（组件内渲染，本地 selection 驱动——参照月球架构，不依赖 App 全局渲染） -->
    <aside v-if="localSelection" class="context-panel" aria-label="所选对象详情" :style="props.headerExpanded ? { transform: 'translateY(76px)' } : undefined">
      <button class="panel-close" aria-label="关闭详情" @click="localSelection = null; emit('clear-selection')">关闭</button>

      <template v-if="selectedSpacecraft">
        <p class="context-type">NORAD {{ selectedSpacecraft.noradCatalogId }}</p>
        <h2>{{ selectedSpacecraft.nameZh }}</h2>
        <p class="context-subtitle">{{ selectedSpacecraft.nameEn }}</p>
        <p class="context-description">{{ selectedSpacecraft.description }}</p>
        <dl>
          <div><dt>运营方</dt><dd>{{ selectedSpacecraft.operatorName }}</dd></div>
          <div v-if="selectedSpacecraft.launchDate"><dt>发射</dt><dd>{{ selectedSpacecraft.launchDate }} · {{ selectedSpacecraft.launchSite }} · {{ selectedSpacecraft.launchVehicle }}</dd></div>
          <div><dt>轨道倾角</dt><dd>{{ Number(selectedSpacecraft.omm.INCLINATION).toFixed(2) }}°</dd></div>
          <div><dt>偏心率</dt><dd>{{ Number(selectedSpacecraft.omm.ECCENTRICITY).toFixed(6) }}</dd></div>
          <div><dt>轨道周期</dt><dd>{{ orbitPeriodText(selectedSpacecraft.omm.MEAN_MOTION) }}</dd></div>
        </dl>
        <p class="source-caption">轨道历元 {{ new Date(selectedSpacecraft.orbitEpoch).toLocaleString('zh-CN', { hour12: false }) }}<br>{{ selectedSpacecraft.sourceName }}</p>
      </template>

      <template v-else-if="selectedSite">
        <p class="context-type launch-context">LAUNCH SITE · {{ selectedSite.countryCode }}</p>
        <h2>{{ selectedSite.nameZh }}</h2>
        <p class="context-subtitle">{{ selectedSite.nameEn }}</p>
        <p class="context-description">{{ selectedSite.description }}</p>
        <dl>
          <div><dt>国家 / 地区</dt><dd>{{ selectedSite.countryNameZh }}</dd></div>
          <div><dt>纬度</dt><dd>{{ formatCoordinate(selectedSite.latitude, 'N', 'S') }}</dd></div>
          <div><dt>经度</dt><dd>{{ formatCoordinate(selectedSite.longitude, 'E', 'W') }}</dd></div>
        </dl>
      </template>

      <template v-else-if="selectedEvent">
        <p class="context-type launch-context">LAUNCH · {{ selectedEvent.statusAbbrev }}</p>
        <h2 lang="en">{{ selectedEvent.missionName || selectedEvent.name }}</h2>
        <p
          v-if="selectedEvent.missionNameZh && selectedEvent.missionNameZh !== selectedEvent.missionName"
          class="context-translation"
        >
          {{ selectedEvent.missionNameZh }}
        </p>
        <p class="context-subtitle">
          {{ launchVehicleName(selectedEvent) }}<template v-if="selectedEvent.providerName"> · {{ selectedEvent.providerName }}</template>
        </p>
        <div class="event-clock"><strong>{{ eventDate(selectedEvent.net).month }} {{ eventDate(selectedEvent.net).day }}</strong><span>{{ eventDate(selectedEvent.net).time }} UTC+8</span></div>
        <p class="context-description">{{ selectedEvent.missionDescriptionZh || '任务详情暂未公开。' }}</p>
        <dl>
          <div><dt>任务类型</dt><dd>{{ selectedEvent.missionTypeZh || '待确认' }}</dd></div>
          <div><dt>状态</dt><dd>{{ selectedEvent.statusNameZh }}</dd></div>
          <div><dt>发射台</dt><dd>{{ selectedEvent.padNameZh || '待确认' }}</dd></div>
          <div><dt>地点</dt><dd>{{ selectedEvent.locationNameZh || '待确认' }}</dd></div>
        </dl>
        <div class="site-context">
          <p>发射场</p>
          <template v-if="selectedEventSite">
            <strong>{{ selectedEventSite.nameZh }}</strong>
            <span>{{ selectedEventSite.description }}</span>
          </template>
          <template v-else>
            <strong>{{ selectedEvent.padNameZh || selectedEvent.locationNameZh }}</strong>
            <span>当前事件源提供了位置和发射台信息，详细场地资料将在后续数据扩充中补充。</span>
          </template>
        </div>
        <div v-if="selectedEvent.hasOriginal" class="event-original-disclosure">
          <button
            type="button"
            :aria-expanded="showEventOriginal"
            @click="showEventOriginal = !showEventOriginal"
          >
            {{ showEventOriginal ? '收起原文' : '查看原文' }}
          </button>
          <div v-if="showEventOriginal" class="event-original-copy">
            <p>来源原文</p>
            <strong>{{ selectedEvent.missionName || selectedEvent.name }}</strong>
            <span>{{ selectedEvent.name }}</span>
            <dl>
              <div><dt>STATUS</dt><dd>{{ selectedEvent.statusName }}</dd></div>
              <div v-if="selectedEvent.padName"><dt>PAD</dt><dd>{{ selectedEvent.padName }}</dd></div>
              <div v-if="selectedEvent.locationName"><dt>LOCATION</dt><dd>{{ selectedEvent.locationName }}</dd></div>
            </dl>
          </div>
        </div>
      </template>
    </aside>
  </div>
</template>

<style scoped>
.scene-host { position: absolute; inset: 0; overflow: hidden; cursor: default; }
/* 地球场景入场：进入边界触发 0.3s 渐亮（裸星球先出现；默认隐藏，revealed 时过渡显现） */
.scene-host { opacity: 0; transition: opacity 0.3s ease; }
.scene-host.revealed { opacity: 1; }
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
.scene-observer-label.inactive { opacity: .34; }
.scene-observer-label.inactive i { box-shadow: none; }
.texture-warning { position: absolute; z-index: 4; top: 82px; left: 50%; transform: translateX(-50%); color: #e6b985; font: 11px var(--font-mono); }
</style>
