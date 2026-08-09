<!--
THESIS: ORBIT is a scrollable observatory, not a fixed cockpit.
OWN-WORLD: deep spatial navy, orbital blue, launch amber, one shared 12-column frame.
STORY: explore Earth first, then search objects, understand launch sites, and inspect the full launch schedule.
FIRST VIEWPORT: a quiet heading above one dominant globe; controls are compact and details appear only after selection.
FORM: progressive observatory, the assigned seventh Operate structure; dense datasets receive dedicated workspaces below the scene.
-->
<script setup lang="ts">
import { computed, defineAsyncComponent, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { CATALOG_PAGE_SIZE, LAUNCH_SITE_PAGE_SIZE } from './catalog'
import { bilingualName } from './bilingual'
import { JUPITER_PAGE, MERCURY_PAGE, NEPTUNE_PAGE, SATURN_PAGE, SUN_PAGE, URANUS_PAGE, VENUS_PAGE } from './planetPages'
import AuroraCover from './components/AuroraCover.vue'
import SolarSystemItem from './components/SolarSystemItem.vue'
import { fetchObserverPlace, fetchOrbitOverview, fetchSpacecraftCatalog } from './api'
import { spacecraftPoint } from './orbit/coordinates'
import { marsHdReady, moonHdReady, orbitTexturesReady, preloadMarsHdTexture, preloadMoonHdTexture, preloadOrbitTextures, preloadSolarTextures } from './preload'
import { solarTexturesReady } from './solar/textures'
import type { LaunchEvent, LaunchSite, OrbitOverview, SceneLayers, Selection, SpacecraftCatalogPage } from './types'
import { primaryOperator } from './operators'

// 大型 Three.js 场景按路径加载。导入动作总是在原有的黑幕/推镜预热阶段启动，
// 因而不改变用户已经调校过的入场节奏，只减少封面首次下载的负担。
const loadOrbitScene = () => import('./components/OrbitScene.vue')
const loadSolarSystem = () => import('./components/SolarSystem.vue')
const loadMoonScene = () => import('./components/MoonScene.vue')
const loadMarsScene = () => import('./components/MarsScene.vue')
const loadPlanetScene = () => import('./components/PlanetScene.vue')
const loadSkyObservatory = () => import('./components/SkyObservatory.vue')
const OrbitScene = defineAsyncComponent({ loader: loadOrbitScene, suspensible: false })
const SolarSystem = defineAsyncComponent({ loader: loadSolarSystem, suspensible: false })
const MoonScene = defineAsyncComponent({ loader: loadMoonScene, suspensible: false })
const MarsScene = defineAsyncComponent({ loader: loadMarsScene, suspensible: false })
const PlanetScene = defineAsyncComponent({ loader: loadPlanetScene, suspensible: false })
const SkyObservatory = defineAsyncComponent({ loader: loadSkyObservatory, suspensible: false })

type ObserverLocationStatus = 'locating' | 'resolving' | 'located' | 'partial' | 'fallback'

interface ObserverLocation {
  latitude: number
  longitude: number
  label: string
  status: ObserverLocationStatus
}

const fallbackObserver = observerFallback()
const overview = ref<OrbitOverview | null>(null)
const loading = ref(false)
const error = ref('')
const now = ref(new Date())
const selection = ref<Selection | null>(null)
const layers = reactive<SceneLayers>({ spacecraft: true, orbits: true, sites: true })
const objectQuery = ref('')
const operatorFilter = ref('all')
const objectSort = ref<'name' | 'norad' | 'operator'>('name')
const observerLocation = ref<ObserverLocation>({ ...fallbackObserver, status: 'locating' })
const observerFocusRevision = ref(0)
const observerViewActive = ref(false)
let observerLocationRequested = false
let observerLocationRevision = 0
let observerLookupController: AbortController | undefined
const dayNightEnabled = ref(false)
type AppSurface = 'cover' | 'sky' | 'solar-system' | 'orbit' | 'moon' | 'mars' | 'mercury' | 'venus' | 'saturn' | 'jupiter' | 'uranus' | 'neptune' | 'sun'

// 初始页面：纯 hash 决定（无 hash = 首页；#earth/#moon/#solar-system = 对应页）。
// 不用 sessionStorage 恢复——打开网站应总是首页（上次会话的页面残留会导致"打开就是 #solar-system"）
const surface = ref<AppSurface>(surfaceFromHash())
const solarSystemRef = ref<{ resetView?: () => void } | null>(null)
/** 地球界面"进入边界"信号：遮罩开始淡出时递增，OrbitScene 据此播放入场渐亮 */
const orbitRevealTick = ref(0)
const headerExpanded = ref(true)
/** 工具栏/位置按钮：被页头"推下/推回"——rAF 逐帧插值动画。
 *  CSS transition 在该环境被系统减弱动态效果禁用（跳变），JS 动画不受影响 */
const sceneToolbarRef = ref<HTMLElement | null>(null)
const sceneLocationRef = ref<HTMLElement | null>(null)
let toolbarShift = 0
let toolbarAnim: number | undefined
function animateToolbarShift(expanded: boolean) {
  if (toolbarAnim !== undefined) cancelAnimationFrame(toolbarAnim)
  const target = expanded ? 76 : 0
  const from = toolbarShift
  const start = performance.now()
  const tick = (now: number) => {
    const t = Math.min(1, (now - start) / 380)
    const eased = 1 - Math.pow(1 - t, 3)
    toolbarShift = from + (target - from) * eased
    const transform = toolbarShift > 0.5 ? `translateY(${toolbarShift.toFixed(2)}px)` : ''
    if (sceneToolbarRef.value) sceneToolbarRef.value.style.transform = transform
    if (sceneLocationRef.value) sceneLocationRef.value.style.transform = transform
    toolbarAnim = t < 1 ? requestAnimationFrame(tick) : undefined
  }
  toolbarAnim = requestAnimationFrame(tick)
}
watch(headerExpanded, animateToolbarShift, { immediate: true })
const orbitPageActive = ref(true)
const moonPageActive = ref(true)
const marsPageActive = ref(true)
const venusPageActive = ref(true)
const saturnPageActive = ref(true)
const jupiterPageActive = ref(true)
const mercuryPageActive = ref(true)
const uranusPageActive = ref(true)
const neptunePageActive = ref(true)
const sunPageActive = ref(true)

/** 页头可收起逻辑当前是否生效（地球主视图 / 月球页 / 火星页 / 金星土星木星页） */
function collapsibleHeaderActive() {
  if (surface.value === 'orbit') return orbitPageActive.value
  if (surface.value === 'moon') return moonPageActive.value
  if (surface.value === 'mars') return marsPageActive.value
  if (surface.value === 'venus') return venusPageActive.value
  if (surface.value === 'saturn') return saturnPageActive.value
  if (surface.value === 'jupiter') return jupiterPageActive.value
  if (surface.value === 'mercury') return mercuryPageActive.value
  if (surface.value === 'uranus') return uranusPageActive.value
  if (surface.value === 'neptune') return neptunePageActive.value
  if (surface.value === 'sun') return sunPageActive.value
  return false
}
const orbitSectionLeaving = ref(false)
/** 返回太阳系过渡期间抑制页头 hover 唤回（页头随元素一起上滑消失） */
let suppressHeaderReveal = false
/** 首屏 DOM 元素（工具栏/标签/读数）入场状态：由 revealTick 驱动（与 3D/旋转同一时钟）——
 *  旋转完全停住（+900ms）后再缓冲 100ms 置 true，transition 淡入。
 *  不能用 CSS animation-delay：动画从元素挂载起算，与 revealTick 错位（黑幕时长不定） */
const orbitElementsRevealed = ref(!orbitRevealTick.value) // 直接加载（tick=0）默认全显
let orbitElementsTimer: number | undefined
watch(orbitRevealTick, (tick) => {
  if (!tick) return
  orbitElementsRevealed.value = false
  if (orbitElementsTimer !== undefined) clearTimeout(orbitElementsTimer)
  orbitElementsTimer = window.setTimeout(() => {
    orbitElementsRevealed.value = true
  }, 1500) // 与 OrbitScene scheduleRevealLayers 的 3D 弹出延迟一致（旋转 1.45s 停住 + ~50ms 缓冲）
})
/** 是否从 ORBIT 返回太阳系（太阳系场景挂载后从地球近景拉回默认构图） */
const solarEnterFromOrbit = ref(false)
const solarEnterFromMoon = ref(false)
const solarEnterFromMars = ref(false)
const solarEnterFromVenus = ref(false)
const solarEnterFromSaturn = ref(false)
const solarEnterFromJupiter = ref(false)
const solarEnterFromMercury = ref(false)
const solarEnterFromUranus = ref(false)
const solarEnterFromNeptune = ref(false)
const solarEnterFromSun = ref(false)
/** 月球页面"进入边界"信号：遮罩开始淡出时递增，MoonScene 据此渐亮 */
const moonRevealTick = ref(0)
/** 从太阳系进入月球：true 时月球页从"纯月球"开始分阶段揭示 */
const moonEnterFromSolar = ref(false)
/** 返回太阳系：true 时月球页清空月球以外元素（只留球体） */
const moonLeaving = ref(false)
/** 火星页面"进入边界"信号：遮罩开始淡出时递增，MarsScene 据此渐亮 */
const marsRevealTick = ref(0)
/** 从太阳系进入火星：true 时火星页从"纯火星"开始分阶段揭示 */
const marsEnterFromSolar = ref(false)
/** 返回太阳系：true 时火星页清空火星以外元素（只留球体） */
const marsLeaving = ref(false)
/** 金星/土星/木星页（共享 PlanetScene 组件）：状态三件套 ×3 */
const venusRevealTick = ref(0)
const venusEnterFromSolar = ref(false)
const venusLeaving = ref(false)
const saturnRevealTick = ref(0)
const saturnEnterFromSolar = ref(false)
const saturnLeaving = ref(false)
const jupiterRevealTick = ref(0)
const jupiterEnterFromSolar = ref(false)
const jupiterLeaving = ref(false)
const mercuryRevealTick = ref(0)
const mercuryEnterFromSolar = ref(false)
const mercuryLeaving = ref(false)
const uranusRevealTick = ref(0)
const uranusEnterFromSolar = ref(false)
const uranusLeaving = ref(false)
const neptuneRevealTick = ref(0)
const neptuneEnterFromSolar = ref(false)
const neptuneLeaving = ref(false)
const sunRevealTick = ref(0)
const sunEnterFromSolar = ref(false)
const sunLeaving = ref(false)
const siteHeader = ref<HTMLElement | null>(null)
const orbitSection = ref<HTMLElement | null>(null)
const orbitSceneFrame = ref<HTMLElement | null>(null)
let clock: number | undefined
let headerIdleTimer: number | undefined
let pageSurfaceFrame = 0
let lastHeaderActivityAt = 0
const DISPLAY_TIME_ZONE = 'Asia/Shanghai'

// ---- 页面切换过渡（变暗 + 缩放推近/拉远 + 遮罩后换页） ----
const veilActive = ref(false)
/** veil 渐暗/渐亮：rAF 逐帧插值 opacity（系统"减弱动态效果"会禁用 CSS transition——工具栏同款经验） */
const surfaceVeilRef = ref<HTMLElement | null>(null)
let veilOpacity = 0
let veilAnim: number | undefined
function animateVeilOpacity(active: boolean, durationMs: number) {
  if (veilAnim !== undefined) cancelAnimationFrame(veilAnim)
  const target = active ? 1 : 0
  const from = veilOpacity
  const start = performance.now()
  const tick = (now: number) => {
    const t = Math.min(1, (now - start) / Math.max(1, durationMs))
    veilOpacity = from + (target - from) * t
    if (surfaceVeilRef.value) surfaceVeilRef.value.style.opacity = String(veilOpacity)
    veilAnim = t < 1 ? requestAnimationFrame(tick) : undefined
  }
  veilAnim = requestAnimationFrame(tick)
}
/** 根据 veilDuration（'0.4s'）解析毫秒 */
function veilDurationMs(): number {
  const seconds = parseFloat(veilDuration.value)
  return Number.isFinite(seconds) ? seconds * 1000 : 400
}
watch(veilActive, (active) => {
  animateVeilOpacity(active, veilDurationMs())
})
/** 等待 veil 完全变黑（rAF 渐暗动画完成 + 60ms 全黑缓冲）再执行回调。
 *  渐暗时长与 rAF 完成时刻存在竞态（主线程繁忙会推迟 rAF tick）：
 *  若 veil 未到 opacity 1 就切页，新旧场景的首帧会透过遮罩叠影（残影）；
 *  缓冲 60ms 让合成器呈现几帧纯黑，确保旧 canvas 最后一帧已被替换。 */
function waitUntilFullBlack(cb: () => void) {
  const poll = (triesLeft: number) => {
    // 0.999 而非 1：rAF 收尾 `from + (1-from)*t` 浮点可能停在 0.9999...，视觉上已全黑
    if (veilOpacity >= 0.999) {
      transitionTimer = window.setTimeout(cb, 60)
      return
    }
    if (triesLeft <= 0) {
      cb() // 兜底：动画异常（如后台标签页 rAF 暂停）时最多等约 400ms
      return
    }
    transitionTimer = window.setTimeout(() => poll(triesLeft - 1), 16)
  }
  poll(25)
}
/** 封面→太阳系：换页提前到点击瞬间，封面继续覆盖（lingering），黑幕结束才撤下 */
const coverLingering = ref(false)
/** 太阳系入场推镜延迟：封面路径 = 变暗时长（全黑开始时起飞）；直接加载 = 0 */
const solarFlyDelay = ref(0)
/** OrbitScene/MoonScene/MarsScene 首帧贴图上传完成信号（textures-ready） */
const orbitSceneReadyFlag = ref(false)
const moonSceneReadyFlag = ref(false)
const marsSceneReadyFlag = ref(false)
const venusSceneReadyFlag = ref(false)
const saturnSceneReadyFlag = ref(false)
const jupiterSceneReadyFlag = ref(false)
const mercurySceneReadyFlag = ref(false)
const uranusSceneReadyFlag = ref(false)
const neptuneSceneReadyFlag = ref(false)
const sunSceneReadyFlag = ref(false)
/** 等待组件就绪后再渐亮的回调（onEarthSelect/onMoonSelect/onMarsSelect 注册，组件信号或超时触发） */
let pendingOrbitReveal: (() => void) | null = null
let pendingMoonReveal: (() => void) | null = null
let pendingMarsReveal: (() => void) | null = null
let pendingVenusReveal: (() => void) | null = null
let pendingSaturnReveal: (() => void) | null = null
let pendingJupiterReveal: (() => void) | null = null
let pendingMercuryReveal: (() => void) | null = null
let pendingUranusReveal: (() => void) | null = null
let pendingNeptuneReveal: (() => void) | null = null
let pendingSunReveal: (() => void) | null = null
/** 封面路径进入时播放入场推镜；刷新/直接加载不播（静态恢复现场） */
const solarEntryFly = ref(false)
const shellZoom = ref(1)
const shellOrigin = ref('50% 50%')
const shellTransitioning = ref(false)
const veilDuration = ref('0.4s')
let transitionTimer: number | undefined
let transitionFrame: number | undefined
/** 每次导航递增；异步纹理解码完成后先核验代际，旧页面不能把用户拉回去。 */
let navigationGeneration = 0
const deferredNavigationTimers = new Set<number>()

function isCurrentNavigation(generation: number) {
  return generation === navigationGeneration
}

/** 保留原有等待时长，但让它属于当前导航；取消/离开后旧回调不会再写页面状态。 */
function scheduleForNavigation(generation: number, callback: () => void, delay: number) {
  const timer = window.setTimeout(() => {
    deferredNavigationTimers.delete(timer)
    if (isCurrentNavigation(generation)) callback()
  }, delay)
  deferredNavigationTimers.add(timer)
  return timer
}

const shellStyle = computed(() => {
  if (!shellTransitioning.value) return undefined
  return { transform: `scale(${shellZoom.value})`, transformOrigin: shellOrigin.value }
})

interface TransitionTiming {
  /** 退出阶段时长（变暗+缩放），默认 560ms */
  exitMs?: number
  /** 换页前遮罩全黑的停留时长，默认 0 */
  dwellMs?: number
  /** 遮罩淡入时长（CSS），默认 0.4s */
  veilSeconds?: string
}

/** 预取目标场景组件，不等待它完成；原有纹理预热、黑幕与 reveal 时钟仍是唯一节奏来源。 */
function preloadSurfaceComponent(target: AppSurface) {
  if (target === 'sky') void loadSkyObservatory()
  else if (target === 'solar-system') void loadSolarSystem()
  else if (target === 'orbit') void loadOrbitScene()
  else if (target === 'moon') void loadMoonScene()
  else if (target === 'mars') void loadMarsScene()
  else if (target === 'venus' || target === 'saturn' || target === 'jupiter' || target === 'mercury' || target === 'uranus' || target === 'neptune' || target === 'sun') void loadPlanetScene()
}

/** 取消进行中的过渡（含定时器与动画帧），恢复无过渡状态 */
function cancelPendingTransition() {
  navigationGeneration += 1
  for (const timer of deferredNavigationTimers) window.clearTimeout(timer)
  deferredNavigationTimers.clear()
  if (transitionTimer !== undefined) {
    window.clearTimeout(transitionTimer)
    transitionTimer = undefined
  }
  if (transitionFrame !== undefined) {
    window.cancelAnimationFrame(transitionFrame)
    transitionFrame = undefined
  }
  veilActive.value = false
  shellZoom.value = 1
  shellTransitioning.value = false
  coverLingering.value = false
  // 离开标志复位：过渡中止时页面不切换，若 leaving 仍为 true 会触发场景元素永久隐藏
  orbitSectionLeaving.value = false
  moonLeaving.value = false
  marsLeaving.value = false
  venusLeaving.value = false
  saturnLeaving.value = false
  jupiterLeaving.value = false
  mercuryLeaving.value = false
  uranusLeaving.value = false
  neptuneLeaving.value = false
  sunLeaving.value = false
  pendingOrbitReveal = null
  pendingMoonReveal = null
  pendingMarsReveal = null
  pendingVenusReveal = null
  pendingSaturnReveal = null
  pendingJupiterReveal = null
  pendingMercuryReveal = null
  pendingUranusReveal = null
  pendingNeptuneReveal = null
  pendingSunReveal = null
  suppressHeaderReveal = false // 中止返回：页头恢复可 hover 唤回（保持收起态，与正常 orbit 行为一致）
}

/** 过渡切换：当前页变暗并缩放 → 在遮罩后换页（新页利用这段时间加载）→ 新页回弹、遮罩淡出 */
function transitionTo(nextSurface: AppSurface, zoom = 1, origin = '50% 50%', timing: TransitionTiming = {}) {
  if (nextSurface === surface.value) return // 同页切换无意义
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  const exitMs = reduced ? 40 : (timing.exitMs ?? 560)
  const dwellMs = reduced ? 0 : (timing.dwellMs ?? 0)
  const enterMs = reduced ? 40 : 620
  cancelPendingTransition()
  // 过渡动画期间预热目标页资源
  preloadSurfaceComponent(nextSurface)
  if (nextSurface === 'solar-system') preloadSolarTextures()
  if (nextSurface === 'orbit') preloadOrbitTextures()
  // 退出阶段：当前页变暗 + 缩放
  shellOrigin.value = origin
  shellZoom.value = zoom
  shellTransitioning.value = true
  veilDuration.value = reduced ? '0.01s' : (timing.veilSeconds ?? '0.4s')
  veilActive.value = true
  transitionTimer = window.setTimeout(() => {
    // 换页提前到全黑停留开始时：新页面在遮罩后完成挂载、shader 编译与首帧渲染，
    // 等全黑结束淡出时画面已在运动中（消除"黑幕亮起时画面才刚开始/还在编译"的卡顿）
    // abort 检查宽松化：hash 是浏览器中最不稳定的部分（pushState/location.hash 都可能不生效），
    // 严格相等会误伤正常过渡（如返回首页被取消）——只拦"hash 指向其他已定义页面"的情况；
    // hash 未生效时继续过渡（surface 状态才是真实导航，地址栏瑕疵不影响功能）
    const currentFromHash = surfaceFromHash()
    if (currentFromHash !== nextSurface && currentFromHash !== surface.value) {
      cancelPendingTransition()
      return
    }
    void setSurface(nextSurface)
    // 全黑停留：遮罩保持不透明，新页面在幕后完成首帧渲染与加载
    transitionTimer = window.setTimeout(() => {
      // 进入阶段：新页从缩放位置回弹、遮罩淡出
      transitionFrame = requestAnimationFrame(() => {
        transitionFrame = undefined
        shellZoom.value = 1
        veilActive.value = false
      })
      transitionTimer = window.setTimeout(() => {
        // 移除 transform，避免 fixed 定位的页头受影响
        shellTransitioning.value = false
        transitionTimer = undefined
      }, enterMs + 60)
    }, dwellMs)
  }, exitMs)
}

function surfaceFromHash(): AppSurface {
  if (['#sky', '#sky-tonight', '#sky-windows', '#sky-targets', '#sky-events'].includes(window.location.hash)) return 'sky'
  if (window.location.hash === '#solar-system') return 'solar-system'
  if (['#moon', '#moon-scene', '#moon-profile', '#moon-objects', '#moon-sites'].includes(window.location.hash)) return 'moon'
  if (['#mars', '#mars-scene', '#mars-profile', '#mars-objects', '#mars-sites'].includes(window.location.hash)) return 'mars'
  if (['#venus', '#venus-scene', '#venus-profile', '#venus-objects', '#venus-sites'].includes(window.location.hash)) return 'venus'
  if (['#saturn', '#saturn-scene', '#saturn-profile', '#saturn-objects', '#saturn-sites'].includes(window.location.hash)) return 'saturn'
  if (['#jupiter', '#jupiter-scene', '#jupiter-profile', '#jupiter-objects', '#jupiter-sites'].includes(window.location.hash)) return 'jupiter'
  if (['#mercury', '#mercury-scene', '#mercury-profile', '#mercury-objects', '#mercury-sites'].includes(window.location.hash)) return 'mercury'
  if (['#uranus', '#uranus-scene', '#uranus-profile', '#uranus-objects'].includes(window.location.hash)) return 'uranus'
  if (['#neptune', '#neptune-scene', '#neptune-profile', '#neptune-objects'].includes(window.location.hash)) return 'neptune'
  if (['#sun', '#sun-scene', '#sun-profile', '#sun-objects'].includes(window.location.hash)) return 'sun'
  if (['#earth', '#objects', '#sites', '#launches'].includes(window.location.hash)) return 'orbit'
  return 'cover'
}

const selectedSpacecraft = computed(() => selection.value?.kind === 'spacecraft'
  ? overview.value?.spacecraft.find((item) => item.id === selection.value?.id)
  : undefined)
const selectedSite = computed(() => selection.value?.kind === 'site'
  ? overview.value?.launchSites.find((item) => item.id === selection.value?.id)
  : undefined)
const selectedEvent = computed(() => selection.value?.kind === 'event'
  ? overview.value?.events.find((item) => item.externalId === selection.value?.id)
  : undefined)
const upcomingEvents = computed(() => overview.value?.events.filter((item) => new Date(item.net) >= now.value) ?? [])
/** 发射日程折叠：默认只展开前 5 行，可展开全部 */
const launchExpanded = ref(false)
const LAUNCH_PREVIEW_ROWS = 5
const visibleLaunchEvents = computed(() =>
  launchExpanded.value ? upcomingEvents.value : upcomingEvents.value.slice(0, LAUNCH_PREVIEW_ROWS),
)
/** 太阳系头部不把月球/火星/深空的独立同步失败误报成地球轨道不可用；各自页面单独标示其快照状态。 */
const dataHealthy = computed(() => {
  const primary = (overview.value?.freshness ?? []).filter((item) => ['celestrak', 'launch_library_2'].includes(item.sourceCode))
  return primary.length === 2 && primary.every((item) => item.success)
})
/** 地球页页脚数据源：只显示本页实际使用的（CelesTrak 轨道 + Launch Library 发射日程）；
 *  JPL Horizons 服务于太阳系/月球/火星页（深空探测器与月球/火星轨道），不在地球页脚列出 */
const earthSources = computed(() => (overview.value?.freshness ?? []).filter((s) => ['celestrak', 'launch_library_2'].includes(s.sourceCode)))
const operators = computed(() => [...new Set((overview.value?.spacecraft ?? []).map((item) => primaryOperator(item.operatorName)))].sort())

/** 目录条目：TLE 航天器（韦布/斯皮策等非地球轨道任务不再在地球页目录/外圈展示，回归太阳系页真实呈现） */
const catalogItems = computed(() => [
  ...(overview.value?.spacecraft ?? []).map((s) => ({ kind: 'spacecraft' as const, ...s })),
])

function parseCatalogQuery() {
  const raw = objectQuery.value.trim()
  let matcher: ((value: string) => boolean) | undefined
  let queryError = ''
  let query = raw
  let mode: 'keyword' | 'regex' = 'keyword'

  if (raw) {
    if (raw.startsWith('/')) {
      const lastSlash = raw.lastIndexOf('/')
      if (lastSlash <= 0) {
        queryError = '正则表达式需要以 / 结束，例如 /ISS|TIANHE/i'
      } else {
        try {
          const flags = raw.slice(lastSlash + 1)
          if (flags.replaceAll('i', '') !== '') throw new Error('服务端正则仅支持 i（忽略大小写）标记')
          const pattern = raw.slice(1, lastSlash)
          const expression = new RegExp(pattern, flags)
          matcher = (value) => expression.test(value)
          query = pattern
          mode = 'regex'
        } catch (reason) {
          queryError = reason instanceof Error ? `正则表达式无效：${reason.message}` : '正则表达式无效'
        }
      }
    } else {
      const keyword = raw.toLocaleLowerCase()
      matcher = (value) => value.toLocaleLowerCase().includes(keyword)
    }
  }
  return { raw, query, mode, matcher, error: queryError }
}

/** 后端未重启到新版时仍可用当前场景数据完成小目录检索；新版 API 可用后自动切为服务端分页。 */
const parsedCatalogQuery = computed(parseCatalogQuery)
const catalogRemote = ref<SpacecraftCatalogPage | null>(null)
const catalogLoading = ref(false)
const catalogRequestError = ref('')
let catalogRequest: AbortController | undefined
let catalogQueryTimer: number | undefined

const catalogResult = computed(() => {
  const { raw, matcher, error: queryError } = parsedCatalogQuery.value
  const items = catalogItems.value.filter((item) => {
    if (operatorFilter.value !== 'all' && primaryOperator(item.operatorName) !== operatorFilter.value) return false
    if (!matcher) return !raw && !queryError
    return matcher([item.nameZh, item.nameEn, item.noradCatalogId ?? '', item.operatorName, item.category].join(' '))
  })

  items.sort((left, right) => {
    if (objectSort.value === 'norad') return (left.noradCatalogId ?? Number.MAX_SAFE_INTEGER) - (right.noradCatalogId ?? Number.MAX_SAFE_INTEGER)
    if (objectSort.value === 'operator') return primaryOperator(left.operatorName).localeCompare(primaryOperator(right.operatorName), 'zh-CN')
    return left.nameZh.localeCompare(right.nameZh, 'zh-CN')
  })
  return { items, error: queryError }
})

/** 航天器目录分页：接口存在时由 PostgreSQL 分页；旧后端/离线时安全回退到已载入代表性对象。 */
const catalogPage = ref(1)
const catalogTotal = computed(() => catalogRemote.value?.total ?? catalogResult.value.items.length)
const catalogPageCount = computed(() => Math.max(1, Math.ceil(catalogTotal.value / CATALOG_PAGE_SIZE)))
const pagedCatalogItems = computed(() => {
  if (catalogRemote.value) return catalogRemote.value.items.map((item) => ({ kind: 'spacecraft' as const, ...item }))
  const start = (catalogPage.value - 1) * CATALOG_PAGE_SIZE
  return catalogResult.value.items.slice(start, start + CATALOG_PAGE_SIZE)
})
async function loadCatalogPage() {
  if (catalogQueryTimer !== undefined) window.clearTimeout(catalogQueryTimer)
  const { query, mode, error: queryError } = parsedCatalogQuery.value
  if (queryError) {
    catalogRemote.value = null
    catalogRequestError.value = ''
    return
  }
  catalogQueryTimer = window.setTimeout(async () => {
    catalogRequest?.abort()
    const controller = new AbortController()
    catalogRequest = controller
    catalogLoading.value = true
    catalogRequestError.value = ''
    try {
      catalogRemote.value = await fetchSpacecraftCatalog({
        page: catalogPage.value,
        pageSize: CATALOG_PAGE_SIZE,
        query,
        operator: operatorFilter.value,
        sort: objectSort.value,
        mode,
      }, controller.signal)
    } catch (reason) {
      if (controller.signal.aborted) return
      // 过渡部署期间前端可能先于后端更新。此时不让目录空掉，降级为场景内小目录。
      catalogRemote.value = null
      const message = reason instanceof Error ? reason.message : '目录服务暂时不可用'
      if (!message.includes('404')) catalogRequestError.value = message
    } finally {
      if (!controller.signal.aborted) catalogLoading.value = false
    }
  }, 180)
}
watch([objectQuery, operatorFilter, objectSort], () => {
  catalogPage.value = 1
  void loadCatalogPage()
})
watch(catalogPage, () => void loadCatalogPage())
function catalogGotoPage(delta: number) {
  catalogPage.value = Math.min(catalogPageCount.value, Math.max(1, catalogPage.value + delta))
}

/** 发射场分页（每页 4 行；无筛选，纯翻页） */
const launchSitePage = ref(1)
const launchSitePageCount = computed(() => Math.max(1, Math.ceil((overview.value?.launchSites.length ?? 0) / LAUNCH_SITE_PAGE_SIZE)))
const pagedLaunchSites = computed(() =>
  (overview.value?.launchSites ?? []).slice(
    (launchSitePage.value - 1) * LAUNCH_SITE_PAGE_SIZE,
    launchSitePage.value * LAUNCH_SITE_PAGE_SIZE,
  ),
)
const launchSiteGotoPage = (delta: number) => {
  launchSitePage.value = Math.min(launchSitePageCount.value, Math.max(1, launchSitePage.value + delta))
}

/** 双语名称（统一规则）：全部中文主，外国对象附英文注释 */
function catalogBilingual(item: { nameZh: string; nameEn: string }) {
  return bilingualName(item.nameZh, item.nameEn)
}

/** 发射事件双语：中文任务名为主（附英文注释）；无中文名则显示英文任务名 */
function eventBilingual(e: { missionName?: string; missionNameZh?: string; name?: string }) {
  const en = (e.missionName || e.name || '').trim()
  const zh = (e.missionNameZh || '').trim()
  const primary = zh || en
  const secondary = zh && en && zh !== en ? en : ''
  return { primary, secondary, lang: zh ? 'zh-CN' : 'en' }
}



const focusTarget = computed(() => {
  if (selectedEvent.value?.latitude != null && selectedEvent.value.longitude != null) {
    return { latitude: selectedEvent.value.latitude, longitude: selectedEvent.value.longitude, distance: 5.4, key: `event:${selectedEvent.value.externalId}` }
  }
  if (selectedSite.value) {
    return { latitude: selectedSite.value.latitude, longitude: selectedSite.value.longitude, distance: 5.4, key: `site:${selectedSite.value.id}` }
  }
  if (selectedSpacecraft.value) {
    const point = spacecraftPoint(selectedSpacecraft.value, now.value)
    if (point) return { latitude: point.latitude, longitude: point.longitude, distance: 6.0, key: `spacecraft:${selectedSpacecraft.value.id}` }
  }
  if (observerViewActive.value && observerLocation.value.status !== 'locating' && observerLocation.value.status !== 'fallback') return {
    latitude: observerLocation.value.latitude,
    longitude: observerLocation.value.longitude,
    distance: 7.6,
    key: `observer:${observerFocusRevision.value}`,
  }
  return null
})

/** 只有浏览器给出真实 GPS 坐标后才在地球表面展示“我的位置”标记。 */
const observerTarget = computed(() => (
  observerLocation.value.status === 'locating' || observerLocation.value.status === 'fallback'
    ? null
    : observerLocation.value
))

/** 轨道历元/同步时间统一 UTC 显示（与探测器面板一致，避免本地/UTC 混用） */
function formatUTCDateTime(iso: string) {
  if (!iso) return ''
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())} ${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())} UTC`
}

/** UTC 日期（目录表列宽紧凑用） */
function formatUTCDate(iso: string) {
  if (!iso) return ''
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())}`
}

function observerFallback(): Omit<ObserverLocation, 'status'> {
  return { latitude: 0, longitude: 0, label: '位置未确认' }
}

async function resolveObserverLocationName(latitude: number, longitude: number, revision: number) {
  observerLookupController?.abort()
  const controller = new AbortController()
  observerLookupController = controller
  observerLocation.value = { latitude, longitude, label: '正在确认城市区县', status: 'resolving' }
  try {
    const place = await fetchObserverPlace(latitude, longitude, controller.signal)
    if (revision !== observerLocationRevision) return
    observerLocation.value = { latitude, longitude, label: place.label, status: 'located' }
  } catch {
    if (controller.signal.aborted || revision !== observerLocationRevision) return
    // 逆地理编码失败不应抹掉已经获得的真实坐标，也不能退回假定城市。
    observerLocation.value = { latitude, longitude, label: '当前位置（地名暂不可用）', status: 'partial' }
  } finally {
    if (observerLookupController === controller) observerLookupController = undefined
  }
}

function requestObserverLocation() {
  if (observerLocationRequested) return
  observerLocationRequested = true
  if (!navigator.geolocation) {
    observerLocation.value = { ...fallbackObserver, label: '浏览器不支持定位', status: 'fallback' }
    observerFocusRevision.value += 1
    return
  }

  const revision = ++observerLocationRevision
  observerLookupController?.abort()
  observerLocation.value = { ...fallbackObserver, label: '正在获取位置', status: 'locating' }
  navigator.geolocation.getCurrentPosition(
    ({ coords }) => {
      if (revision !== observerLocationRevision) return
      observerLocation.value = {
        latitude: coords.latitude,
        longitude: coords.longitude,
        label: '正在确认城市区县',
        status: 'resolving',
      }
      // 地球定位不等待地名接口：GPS 一到就立即显示标记并完成运镜。
      observerFocusRevision.value += 1
      void resolveObserverLocationName(coords.latitude, coords.longitude, revision)
    },
    (locationError) => {
      if (revision !== observerLocationRevision) return
      const label = locationError.code === locationError.PERMISSION_DENIED
        ? '定位未授权'
        : locationError.code === locationError.TIMEOUT ? '定位请求超时' : '暂时无法获取位置'
      observerLocation.value = { ...fallbackObserver, label, status: 'fallback' }
      observerFocusRevision.value += 1
    },
    { enableHighAccuracy: false, timeout: 6000, maximumAge: 900_000 },
  )
}

function focusObserver() {
  selection.value = null
  observerViewActive.value = true
  observerFocusRevision.value += 1
  if (observerLocation.value.status === 'partial') {
    const revision = ++observerLocationRevision
    void resolveObserverLocationName(observerLocation.value.latitude, observerLocation.value.longitude, revision)
  } else if (observerLocation.value.status === 'fallback') {
    // 用户主动点击时才允许再次尝试定位；封面和无关页面绝不触发权限请求。
    observerLocationRequested = false
    requestObserverLocation()
  }
}

function selectFromScene(nextSelection: Selection) {
  observerViewActive.value = false
  selection.value = nextSelection
}

function leaveObserverView() {
  observerViewActive.value = false
}


function selectAndFocus(nextSelection: Selection) {
  observerViewActive.value = false
  selection.value = nextSelection
  window.requestAnimationFrame(() => document.querySelector('#earth')?.scrollIntoView({ behavior: 'smooth', block: 'start' }))
}

function timeOnly(value: Date | string) {
  return new Intl.DateTimeFormat('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
    timeZone: DISPLAY_TIME_ZONE,
  }).format(new Date(value))
}



async function setSurface(nextSurface: AppSurface) {
  surface.value = nextSurface
  document.title = nextSurface === 'cover'
    ? 'AURORA'
    : nextSurface === 'sky' ? 'AURORA · SKY'
    : nextSurface === 'solar-system' ? 'AURORA · 太阳系'
    : nextSurface === 'moon' ? 'AURORA · MOON'
    : nextSurface === 'mars' ? 'AURORA · MARS'
    : nextSurface === 'venus' ? 'AURORA · VENUS'
    : nextSurface === 'saturn' ? 'AURORA · SATURN'
    : nextSurface === 'jupiter' ? 'AURORA · JUPITER'
    : nextSurface === 'mercury' ? 'AURORA · MERCURY'
    : nextSurface === 'uranus' ? 'AURORA · URANUS'
    : nextSurface === 'neptune' ? 'AURORA · NEPTUNE'
    : nextSurface === 'sun' ? 'AURORA · SUN' : 'AURORA · EARTH'
  if (nextSurface === 'orbit') {
    void ensureOrbitOverview()
    void loadCatalogPage()
    requestObserverLocation()
    orbitPageActive.value = true
    headerExpanded.value = false // 进入 ORBIT 默认收起页头（悬停屏幕顶部可展开）
    // 挂载即隐藏首屏 DOM 元素（黑幕中完成淡出）——若等到 revealTick 才置 false，
    // 元素会经历"从可见淡出 300ms"，在渐亮时呈半透明（"我的位置"等元素残留可见）
    orbitElementsRevealed.value = false
  } else if (nextSurface === 'moon') {
    moonPageActive.value = true
    headerExpanded.value = false // 月球页同样默认收起页头
  } else if (nextSurface === 'mars') {
    marsPageActive.value = true
    headerExpanded.value = false // 火星页同样默认收起页头
  } else if (nextSurface === 'venus' || nextSurface === 'saturn' || nextSurface === 'jupiter' || nextSurface === 'mercury' || nextSurface === 'uranus' || nextSurface === 'neptune' || nextSurface === 'sun') {
    // 金星/土星/木星/水星/天王星/海王星/太阳页：默认收起页头（与月球/火星一致）
    if (nextSurface === 'venus') venusPageActive.value = true
    else if (nextSurface === 'saturn') saturnPageActive.value = true
    else if (nextSurface === 'jupiter') jupiterPageActive.value = true
    else if (nextSurface === 'mercury') mercuryPageActive.value = true
    else if (nextSurface === 'uranus') uranusPageActive.value = true
    else if (nextSurface === 'neptune') neptunePageActive.value = true
    else sunPageActive.value = true
    headerExpanded.value = false
  } else {
    headerExpanded.value = true
    suppressHeaderReveal = false // 切到太阳系/封面：页头恢复正常唤回
  }
  await nextTick()
  window.scrollTo({ top: 0, behavior: 'instant' })
  updateActivePage()
  if (nextSurface === 'orbit' || nextSurface === 'moon' || nextSurface === 'mars' || nextSurface === 'venus' || nextSurface === 'saturn' || nextSurface === 'jupiter' || nextSurface === 'mercury' || nextSurface === 'uranus' || nextSurface === 'neptune' || nextSurface === 'sun') scheduleHeaderCollapse()
  else clearHeaderIdleTimer()
}

function enterSolarSystem() {
  if (surface.value === 'orbit') {
    enterSolarSystemFromOrbit()
    return
  }
  if (surface.value === 'moon') {
    enterSolarSystemFromMoon()
    return
  }
  if (surface.value === 'mars') {
    enterSolarSystemFromMars()
    return
  }
  if (surface.value === 'venus') {
    enterSolarSystemFromVenus()
    return
  }
  if (surface.value === 'saturn') {
    enterSolarSystemFromSaturn()
    return
  }
  if (surface.value === 'jupiter') {
    enterSolarSystemFromJupiter()
    return
  }
  if (surface.value === 'mercury') {
    enterSolarSystemFromMercury()
    return
  }
  if (surface.value === 'uranus') {
    enterSolarSystemFromUranus()
    return
  }
  if (surface.value === 'neptune') {
    enterSolarSystemFromNeptune()
    return
  }
  if (surface.value === 'sun') {
    enterSolarSystemFromSun()
    return
  }
  solarEnterFromOrbit.value = false
  solarEnterFromMoon.value = false // 封面进入：两个来源标志都清空
  solarEnterFromMars.value = false // 封面进入：火星来源标志同样清空
  solarEnterFromVenus.value = false
  solarEnterFromSaturn.value = false
  solarEnterFromJupiter.value = false
  solarEnterFromMercury.value = false
  solarEnterFromUranus.value = false
  solarEnterFromNeptune.value = false
  solarEnterFromSun.value = false
  window.history.pushState(null, '', '#solar-system')
  preloadSurfaceComponent('solar-system')
  preloadSolarTextures()
  void ensureOrbitOverview()
  preloadOrbitTextures() // 提前预热地球纹理，为下一步进入 ORBIT 做准备
  // 封面进入太阳系：星野页面（星空插图）渐入 → 停留（对应原黑屏时间）→ 渐亮揭示推镜
  cancelPendingTransition()
  const generation = navigationGeneration
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  const exitMs = reduced ? 40 : 20 // 星野渐入 20ms（近乎瞬切——用户指定）
  const dwellMs = reduced ? 0 : 880 // 星野停留 880ms（渐入缩到 20ms 后补回时长：总黑幕保持 900ms 与原来一致，
  // 停留时间未被砍短；太阳系仍在黑幕中提前渲染，纹理解码就绪后 reveal 才触发）
  solarEntryFly.value = true // 封面路径：播放入场推镜（启动由 SolarSystem 侧等纹理就绪）
  shellOrigin.value = '50% 42%'
  shellZoom.value = 1.05
  shellTransitioning.value = true
  veilDuration.value = reduced ? '0.01s' : '0.01s' // 星野瞬切：点击后星点页面直接完整呈现（略过"先黑后星点"的渐入）
  veilActive.value = true
  coverLingering.value = true
  void setSurface('solar-system')
  // 停留结束 = 纹理解码就绪（或 2.5s 超时兜底）——黑幕时长与加载同步，
  // 渐亮揭示时推镜正在中途（SolarSystem 侧同源就绪信号启动推镜）
  let revealed = false
  const reveal = () => {
    if (revealed || !isCurrentNavigation(generation) || surface.value !== 'solar-system') return
    revealed = true
    transitionTimer = undefined
    if (surfaceFromHash() !== 'solar-system') {
      cancelPendingTransition()
      return
    }
    coverLingering.value = false
    transitionFrame = requestAnimationFrame(() => {
      transitionFrame = undefined
      shellZoom.value = 1
      veilDuration.value = reduced ? '0.01s' : '0.4s' // 渐亮时长：短促地从黑变亮（不拖灰），
      // 全亮时刻 ≈ 推镜路程 70%（剩 1/3 距离）；其后推镜最后 1/3 全是清晰画面
      veilActive.value = false
    })
    transitionTimer = scheduleForNavigation(generation, () => {
      shellTransitioning.value = false
      transitionTimer = undefined
    }, 400 + 60)
  }
  transitionTimer = scheduleForNavigation(generation, () => {
    solarTexturesReady().then(() => {
      if (isCurrentNavigation(generation)) reveal()
    })
    scheduleForNavigation(generation, reveal, 2500) // 兜底：加载异常时最迟 2.5s 揭示
  }, exitMs + dwellMs)
}

function enterSky() {
  window.history.pushState(null, '', '#sky-tonight')
  transitionTo('sky', 1.018, '25% 50%', { exitMs: 440, dwellMs: 100, veilSeconds: '0.32s' })
}


/** ORBIT → 太阳系：滚回主地球视图 → 信息淡出只留地球 → 变暗 → 切页，
 *  太阳系场景从地球近景开始拉回（地球缩回轨道位置，遮罩淡出时可见） */
function returnToSolarSystem(skipPush = false) {
  if (surface.value === 'orbit') enterSolarSystemFromOrbit(skipPush)
  else if (surface.value === 'moon') enterSolarSystemFromMoon(skipPush)
  else if (surface.value === 'mars') enterSolarSystemFromMars(skipPush)
  else if (surface.value === 'venus') enterSolarSystemFromVenus(skipPush)
  else if (surface.value === 'saturn') enterSolarSystemFromSaturn(skipPush)
  else if (surface.value === 'jupiter') enterSolarSystemFromJupiter(skipPush)
  else if (surface.value === 'mercury') enterSolarSystemFromMercury(skipPush)
  else if (surface.value === 'uranus') enterSolarSystemFromUranus(skipPush)
  else if (surface.value === 'neptune') enterSolarSystemFromNeptune(skipPush)
  else if (surface.value === 'sun') enterSolarSystemFromSun(skipPush)
}

/** ORBIT → 太阳系（skipPush = 浏览器返回路径，hash 已是目标不重复入栈） */
function enterSolarSystemFromOrbit(skipPush = false) {
  if (!skipPush) window.history.pushState(null, '', '#solar-system')
  preloadSurfaceComponent('solar-system')
  preloadSolarTextures()
  cancelPendingTransition()
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  // 阶段 1：滚回主地球视图，页面信息（标签/工具条/内容区）淡出，只留地球；
  // 页头若展开则随之一同上滑消失（.collapsed 的 translateY(-100%) 过渡），过渡期间 hover 不唤回
  window.scrollTo({ top: 0, behavior: reduced ? 'auto' : 'smooth' })
  orbitSectionLeaving.value = true
  headerExpanded.value = false
  suppressHeaderReveal = true
  solarEnterFromOrbit.value = true
  solarEnterFromMoon.value = false // 关键：清空月球来源遗留——否则 SolarSystem 误执行 flyFromMoon（起点=放大月球）
  solarEnterFromMars.value = false // 清空火星来源遗留（同理）
  solarEnterFromVenus.value = false // 清空金星来源遗留（同理）
  solarEnterFromSaturn.value = false // 清空土星来源遗留（同理）
  solarEnterFromJupiter.value = false // 清空木星来源遗留（同理）
  // 阶段 2：变暗，盖住地球界面
  transitionTimer = window.setTimeout(() => {
    if (surfaceFromHash() !== 'solar-system') {
      cancelPendingTransition()
      return
    }
    veilDuration.value = reduced ? '0.01s' : '0.3s' // 渐暗 300ms（原 400ms——加速，卡顿窗口缩短）
    veilActive.value = true
    // 阶段 3：等 veil 真正全黑（rAF 完成 + 60ms 缓冲）再切页——切页是重操作，
    // 若在渐暗进行中切页：a) 其 JS 卡顿会被感知在渐暗过程；b) veil 未到 opacity 1 时
    // 新旧场景首帧会透过遮罩叠影（残影）；全黑缓冲后再切页则完全不可见
    waitUntilFullBlack(() => {
      if (surfaceFromHash() !== 'solar-system') {
        cancelPendingTransition()
        return
      }
      orbitSectionLeaving.value = false
      void setSurface('solar-system')
      requestAnimationFrame(() => {
        veilDuration.value = reduced ? '0.01s' : '0.3s' // 渐亮 300ms（原 500ms 太长）
        veilActive.value = false
      })
      transitionTimer = undefined
    })
  }, reduced ? 20 : 420)
}

function enterOrbit() {
  window.history.pushState(null, '', '#earth')
  // 朝向地球方向推近（地球大致位于画面 55%/38% 处），形成“放大进入地球”的感觉
  transitionTo('orbit', 1.12, '55% 38%')
}

function returnToCover(skipPush = false) {
  if (surface.value === 'orbit' || surface.value === 'moon' || surface.value === 'mars' || surface.value === 'venus' || surface.value === 'saturn' || surface.value === 'jupiter' || surface.value === 'mercury' || surface.value === 'uranus' || surface.value === 'neptune' || surface.value === 'sun') {
    exitPlanetToCover(skipPush) // 行星界面：完整退出动画（栏目淡出 → 裸星球 → 渐暗 → 封面）
    return
  }
  transitionTo('cover', 0.96) // 太阳系/封面：原有过渡
}

/** 行星界面 → 首页：完全复刻"返回太阳系"的退出动画——
 *  页头/栏目/元素先上滑淡出，只留裸星球，再渐暗切到封面 */
function exitPlanetToCover(skipPush = false) {
  if (!skipPush && window.location.hash !== '#home') window.history.pushState(null, '', '#home')
  cancelPendingTransition()
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  const fromMoon = surface.value === 'moon'
  const fromMars = surface.value === 'mars'
  const fromVenus = surface.value === 'venus'
  const fromSaturn = surface.value === 'saturn'
  const fromJupiter = surface.value === 'jupiter'
  const fromMercury = surface.value === 'mercury'
  const fromUranus = surface.value === 'uranus'
  const fromNeptune = surface.value === 'neptune'
  const fromSun = surface.value === 'sun'
  // 阶段 1：滚回主视图 + 信息/栏目淡出（页头随之上滑），只留裸星球
  window.scrollTo({ top: 0, behavior: reduced ? 'auto' : 'smooth' })
  if (fromMoon) moonLeaving.value = true
  else if (fromMars) marsLeaving.value = true
  else if (fromVenus) venusLeaving.value = true
  else if (fromSaturn) saturnLeaving.value = true
  else if (fromJupiter) jupiterLeaving.value = true
  else if (fromMercury) mercuryLeaving.value = true
  else if (fromUranus) uranusLeaving.value = true
  else if (fromNeptune) neptuneLeaving.value = true
  else if (fromSun) sunLeaving.value = true
  else orbitSectionLeaving.value = true
  headerExpanded.value = false
  suppressHeaderReveal = true
  // 阶段 2：元素淡出完成后 veil 渐暗
  transitionTimer = window.setTimeout(() => {
    if (surfaceFromHash() !== 'cover') {
      cancelPendingTransition()
      return
    }
    veilDuration.value = reduced ? '0.01s' : '0.3s'
    veilActive.value = true
    // 阶段 3：等 veil 真正全黑再切页（与返回太阳系同款，避免新旧画面叠影）
    waitUntilFullBlack(() => {
      if (surfaceFromHash() !== 'cover') {
        cancelPendingTransition()
        return
      }
      if (fromMoon) moonLeaving.value = false
      else if (fromMars) marsLeaving.value = false
      else if (fromVenus) venusLeaving.value = false
      else if (fromSaturn) saturnLeaving.value = false
      else if (fromJupiter) jupiterLeaving.value = false
      else if (fromMercury) mercuryLeaving.value = false
      else if (fromUranus) uranusLeaving.value = false
      else if (fromNeptune) neptuneLeaving.value = false
      else if (fromSun) sunLeaving.value = false
      else orbitSectionLeaving.value = false
      void setSurface('cover')
      requestAnimationFrame(() => {
        veilDuration.value = reduced ? '0.01s' : '0.3s'
        veilActive.value = false
      })
      transitionTimer = undefined
    })
  }, reduced ? 20 : ((fromMoon || fromMars || fromVenus || fromSaturn || fromJupiter || fromMercury || fromUranus || fromNeptune || fromSun) ? 450 : 420))
}

// ---- 太阳系 → 地球：镜头在太阳系内放大地球 → 变暗 → 切页 ----

/** 点击地球瞬间：URL 切到 #earth，开始预热 ORBIT 资源 */
function onEarthFlyStart() {
  window.history.pushState(null, '', '#earth')
  preloadSurfaceComponent('orbit')
  preloadOrbitTextures()
  void ensureOrbitOverview()
  requestObserverLocation()
  cancelPendingTransition()
}

/** 地球放大到一定程度：遮罩快速变暗（尽量缩短黑屏时间） */
function onEarthFlyZoom() {
  if (surface.value === 'orbit') return // 已切页（幂等保护）；不再依赖 hash——该环境 pushState 不生效
  const generation = navigationGeneration
  veilDuration.value = '0.22s'
  veilActive.value = true
  // 遮罩完全变黑后（+800ms，黑屏留足余量）才开始 8k 解码——主线程解码/GPU 上传都发生在黑屏中；
  // 解码完成才换页（3s 超时兜底）
  scheduleForNavigation(generation, () => {
    orbitTexturesReady().then(() => onEarthSelect(generation))
    scheduleForNavigation(generation, () => onEarthSelect(generation), 3000)
  }, 800)
}

/** 地球放大完成（遮罩已黑）：换页，等首帧贴图 GPU 上传完成再渐亮 */
function onEarthSelect(generation = navigationGeneration) {
  if (!isCurrentNavigation(generation) || surface.value !== 'solar-system') return
  veilActive.value = true
  void setSurface('orbit')
  let revealDone = false
  const reveal = () => {
    if (revealDone || !isCurrentNavigation(generation) || surface.value !== 'orbit') return
    revealDone = true
    pendingOrbitReveal = null
    requestAnimationFrame(() => {
      veilDuration.value = '0.3s'
      veilActive.value = false
      // 进入边界：遮罩开始淡出的同一帧递增信号，地球场景据此 0.5s 渐亮（与月球一致）
      orbitRevealTick.value += 1
    })
  }
  if (orbitSceneReadyFlag.value) reveal()
  else {
    pendingOrbitReveal = reveal
    scheduleForNavigation(generation, reveal, 3000) // 兜底：上传异常时最迟 3s 揭示
  }
}

/** OrbitScene 首帧贴图上传完成 */
function onOrbitSceneReady() {
  orbitSceneReadyFlag.value = true
  if (pendingOrbitReveal) pendingOrbitReveal()
}

/** 点击月球瞬间：URL 切到 #moon，预热 8k 月球纹理 */
function onMoonFlyStart() {
  window.history.pushState(null, '', '#moon')
  cancelPendingTransition()
  preloadSurfaceComponent('moon')
  preloadMoonHdTexture() // 预热 8k 月球贴图（本地资源，提前解码避免切换后卡顿）
  moonEnterFromSolar.value = true // 入场路径：月球页分阶段揭示（每次进入都从纯月球开始）
  moonLeaving.value = false // 重置返回清空状态（否则第二次进入残留 true，清空流程失效）
}

/** 月球放大到一定程度：遮罩快速变暗 */
function onMoonFlyZoom() {
  if (surface.value === 'moon') return // 已切页（幂等保护）；不再依赖 hash——该环境 pushState 不生效
  const generation = navigationGeneration
  veilDuration.value = '0.22s'
  veilActive.value = true
  // 遮罩完全变黑后（+800ms，黑屏留足余量）才开始 16k 解码——主线程解码/GPU 上传都发生在黑屏中
  scheduleForNavigation(generation, () => {
    moonHdReady().then(() => onMoonSelect(generation))
    scheduleForNavigation(generation, () => onMoonSelect(generation), 3000)
  }, 800)
}

/** 月球放大完成（遮罩已黑）：换页，等首帧贴图 GPU 上传完成再渐亮 */
function onMoonSelect(generation = navigationGeneration) {
  if (!isCurrentNavigation(generation) || surface.value !== 'solar-system') return
  veilActive.value = true
  solarEnterFromMoon.value = false
  void setSurface('moon')
  let revealDone = false
  const reveal = () => {
    if (revealDone || !isCurrentNavigation(generation) || surface.value !== 'moon') return
    revealDone = true
    pendingMoonReveal = null
    requestAnimationFrame(() => {
      veilDuration.value = '0.3s'
      veilActive.value = false
      // 进入边界：遮罩开始淡出的同一帧递增信号，月球场景据此 0.5s 渐亮（与地球一致）
      moonRevealTick.value += 1
    })
  }
  if (moonSceneReadyFlag.value) reveal()
  else {
    pendingMoonReveal = reveal
    scheduleForNavigation(generation, reveal, 3000) // 兜底
  }
}

/** MoonScene 首帧贴图上传完成 */
function onMoonSceneReady() {
  moonSceneReadyFlag.value = true
  if (pendingMoonReveal) pendingMoonReveal()
}

/** 点击火星瞬间：URL 切到 #mars，预热 8k 火星纹理 */
function onMarsFlyStart() {
  window.history.pushState(null, '', '#mars')
  cancelPendingTransition()
  preloadSurfaceComponent('mars')
  preloadMarsHdTexture() // 预热 8k 火星贴图（本地资源，提前解码避免切换后卡顿）
  marsEnterFromSolar.value = true // 入场路径：火星页分阶段揭示（每次进入都从纯火星开始）
  marsLeaving.value = false // 重置返回清空状态（否则第二次进入残留 true，清空流程失效）
}

/** 火星放大到一定程度：遮罩快速变暗 */
function onMarsFlyZoom() {
  if (surface.value === 'mars') return // 已切页（幂等保护）
  const generation = navigationGeneration
  veilDuration.value = '0.22s'
  veilActive.value = true
  scheduleForNavigation(generation, () => {
    marsHdReady().then(() => onMarsSelect(generation))
    scheduleForNavigation(generation, () => onMarsSelect(generation), 3000)
  }, 800)
}

/** 火星放大完成（遮罩已黑）：换页，等首帧贴图 GPU 上传完成再渐亮 */
function onMarsSelect(generation = navigationGeneration) {
  if (!isCurrentNavigation(generation) || surface.value !== 'solar-system') return
  veilActive.value = true
  solarEnterFromMars.value = false
  void setSurface('mars')
  let revealDone = false
  const reveal = () => {
    if (revealDone || !isCurrentNavigation(generation) || surface.value !== 'mars') return
    revealDone = true
    pendingMarsReveal = null
    requestAnimationFrame(() => {
      veilDuration.value = '0.3s'
      veilActive.value = false
      // 进入边界：遮罩开始淡出的同一帧递增信号，火星场景据此 0.5s 渐亮（与地球一致）
      marsRevealTick.value += 1
    })
  }
  if (marsSceneReadyFlag.value) reveal()
  else {
    pendingMarsReveal = reveal
    scheduleForNavigation(generation, reveal, 3000) // 兜底
  }
}

/** MarsScene 首帧贴图上传完成 */
function onMarsSceneReady() {
  marsSceneReadyFlag.value = true
  if (pendingMarsReveal) pendingMarsReveal()
}

// ---- 金星/土星/木星（共享 PlanetScene 组件）：与月球/火星同款 fly-start → fly-zoom → select 链路 ----

/** 外行星（水星/金星/土星/木星/天王星/海王星）共享的太阳系进入处理器（fly 链路 + reveal 信号）。
 *  数据驱动：每颗行星的 ref 状态（enterFromSolar/leaving/revealTick/readyFlag/pendingReveal）由注册表提供，
 *  模板绑定只引用最终的 onXxxFlyStart/onXxxFlyZoom/onXxxSelect/onXxxSceneReady。 */
type OuterPlanetKey2 = 'mercury' | 'venus' | 'saturn' | 'jupiter' | 'uranus' | 'neptune' | 'sun'

/** 读取某颗外行星的 pending reveal 回调（let 变量无法按引用传递，用 key 分发） */
function getPendingReveal(key: OuterPlanetKey2): (() => void) | null {
  if (key === 'mercury') return pendingMercuryReveal
  if (key === 'venus') return pendingVenusReveal
  if (key === 'saturn') return pendingSaturnReveal
  if (key === 'jupiter') return pendingJupiterReveal
  if (key === 'uranus') return pendingUranusReveal
  if (key === 'neptune') return pendingNeptuneReveal
  return pendingSunReveal
}

/** 设置某颗外行星的 pending reveal 回调 */
function setPendingReveal(key: OuterPlanetKey2, fn: (() => void) | null) {
  if (key === 'mercury') pendingMercuryReveal = fn
  else if (key === 'venus') pendingVenusReveal = fn
  else if (key === 'saturn') pendingSaturnReveal = fn
  else if (key === 'jupiter') pendingJupiterReveal = fn
  else if (key === 'uranus') pendingUranusReveal = fn
  else if (key === 'neptune') pendingNeptuneReveal = fn
  else pendingSunReveal = fn
}

/** 太阳系来源标志（SolarSystem 据此恢复初始选中）：置当前行星 true，其余全部清空 */
function setSolarEnterFromOuter(key: OuterPlanetKey2) {
  solarEnterFromMercury.value = key === 'mercury'
  solarEnterFromVenus.value = key === 'venus'
  solarEnterFromSaturn.value = key === 'saturn'
  solarEnterFromJupiter.value = key === 'jupiter'
  solarEnterFromUranus.value = key === 'uranus'
  solarEnterFromNeptune.value = key === 'neptune'
  solarEnterFromSun.value = key === 'sun'
}

function createOuterPlanetHandlers(key: OuterPlanetKey2) {
  const enterFromSolar = { mercury: mercuryEnterFromSolar, venus: venusEnterFromSolar, saturn: saturnEnterFromSolar, jupiter: jupiterEnterFromSolar, uranus: uranusEnterFromSolar, neptune: neptuneEnterFromSolar, sun: sunEnterFromSolar }[key]
  const leaving = { mercury: mercuryLeaving, venus: venusLeaving, saturn: saturnLeaving, jupiter: jupiterLeaving, uranus: uranusLeaving, neptune: neptuneLeaving, sun: sunLeaving }[key]
  const revealTick = { mercury: mercuryRevealTick, venus: venusRevealTick, saturn: saturnRevealTick, jupiter: jupiterRevealTick, uranus: uranusRevealTick, neptune: neptuneRevealTick, sun: sunRevealTick }[key]
  const readyFlag = { mercury: mercurySceneReadyFlag, venus: venusSceneReadyFlag, saturn: saturnSceneReadyFlag, jupiter: jupiterSceneReadyFlag, uranus: uranusSceneReadyFlag, neptune: neptuneSceneReadyFlag, sun: sunSceneReadyFlag }[key]

  const onFlyStart = () => {
    window.history.pushState(null, '', `#${key}`)
    cancelPendingTransition()
    preloadSurfaceComponent(key)
    enterFromSolar.value = true
    leaving.value = false
  }

  const onFlyZoom = () => {
    if (surface.value === key) return
    const generation = navigationGeneration
    veilDuration.value = '0.22s'
    veilActive.value = true
    scheduleForNavigation(generation, () => onSelect(generation), 800)
  }

  const onSelect = (generation = navigationGeneration) => {
    if (!isCurrentNavigation(generation) || surface.value !== 'solar-system') return
    veilActive.value = true
    setSolarEnterFromOuter(key)
    void setSurface(key)
    let revealDone = false
    const reveal = () => {
      if (revealDone || !isCurrentNavigation(generation) || surface.value !== key) return
      revealDone = true
      setPendingReveal(key, null)
      requestAnimationFrame(() => {
        veilDuration.value = '0.3s'
        veilActive.value = false
        revealTick.value += 1
      })
    }
    if (readyFlag.value) reveal()
    else {
      setPendingReveal(key, reveal)
      scheduleForNavigation(generation, reveal, 3000)
    }
  }

  const onSceneReady = () => {
    readyFlag.value = true
    const pending = getPendingReveal(key)
    if (pending) pending()
  }

  return { onFlyStart, onFlyZoom, onSelect, onSceneReady, surface: key }
}

const mercuryHandlers = createOuterPlanetHandlers('mercury')
const venusHandlers = createOuterPlanetHandlers('venus')
const saturnHandlers = createOuterPlanetHandlers('saturn')
const jupiterHandlers = createOuterPlanetHandlers('jupiter')
const uranusHandlers = createOuterPlanetHandlers('uranus')
const neptuneHandlers = createOuterPlanetHandlers('neptune')
const sunHandlers = createOuterPlanetHandlers('sun')

const onVenusFlyStart = venusHandlers.onFlyStart
const onVenusFlyZoom = venusHandlers.onFlyZoom
const onVenusSelect = venusHandlers.onSelect
const onVenusSceneReady = venusHandlers.onSceneReady
const onSaturnFlyStart = saturnHandlers.onFlyStart
const onSaturnFlyZoom = saturnHandlers.onFlyZoom
const onSaturnSelect = saturnHandlers.onSelect
const onSaturnSceneReady = saturnHandlers.onSceneReady
const onJupiterFlyStart = jupiterHandlers.onFlyStart
const onJupiterFlyZoom = jupiterHandlers.onFlyZoom
const onJupiterSelect = jupiterHandlers.onSelect
const onJupiterSceneReady = jupiterHandlers.onSceneReady
const onMercuryFlyStart = mercuryHandlers.onFlyStart
const onMercuryFlyZoom = mercuryHandlers.onFlyZoom
const onMercurySelect = mercuryHandlers.onSelect
const onMercurySceneReady = mercuryHandlers.onSceneReady
const onUranusFlyStart = uranusHandlers.onFlyStart
const onUranusFlyZoom = uranusHandlers.onFlyZoom
const onUranusSelect = uranusHandlers.onSelect
const onUranusSceneReady = uranusHandlers.onSceneReady
const onNeptuneFlyStart = neptuneHandlers.onFlyStart
const onNeptuneFlyZoom = neptuneHandlers.onFlyZoom
const onNeptuneSelect = neptuneHandlers.onSelect
const onNeptuneSceneReady = neptuneHandlers.onSceneReady
const onSunFlyStart = sunHandlers.onFlyStart
const onSunFlyZoom = sunHandlers.onFlyZoom
const onSunSelect = sunHandlers.onSelect
const onSunSceneReady = sunHandlers.onSceneReady

/** 火星 → 太阳系（skipPush = 浏览器返回路径） */
function enterSolarSystemFromMars(skipPush = false) {
  if (!skipPush) window.history.pushState(null, '', '#solar-system')
  preloadSurfaceComponent('solar-system')
  preloadSolarTextures()
  cancelPendingTransition()
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  solarEnterFromMars.value = true
  solarEnterFromOrbit.value = false // 清空地球来源遗留
  solarEnterFromMoon.value = false // 清空月球来源遗留
  solarEnterFromVenus.value = false // 清空金星来源遗留
  solarEnterFromSaturn.value = false // 清空土星来源遗留
  solarEnterFromJupiter.value = false // 清空木星来源遗留
  // 阶段 1：滚回火星主视图 + 清空火星以外的所有元素（标记/飞行器/标签），只留火星球体；
  // 页头若展开则随之上滑消失（与地球/月球返回一致），过渡期间 hover 不唤回
  window.scrollTo({ top: 0, behavior: reduced ? 'auto' : 'smooth' })
  marsLeaving.value = true
  headerExpanded.value = false
  suppressHeaderReveal = true
  // 阶段 2（清空效果可见后才变暗）：变暗 300ms
  transitionTimer = window.setTimeout(() => {
    veilDuration.value = reduced ? '0.01s' : '0.3s'
    veilActive.value = true
    // 阶段 3：等 veil 真正全黑再切页（同地球/月球返回——避免新旧场景首帧透过遮罩叠影）
    waitUntilFullBlack(() => {
      if (surfaceFromHash() !== 'solar-system') {
        cancelPendingTransition()
        return
      }
      void setSurface('solar-system')
      requestAnimationFrame(() => {
        veilDuration.value = reduced ? '0.01s' : '0.3s' // 渐亮 300ms
        veilActive.value = false
      })
      transitionTimer = undefined
    })
  }, reduced ? 20 : 450)
}

/** 外行星页（金星/土星/木星/水星/天王星/海王星）→ 太阳系（skipPush = 浏览器返回路径） */
function enterSolarSystemFromOuterPlanet(key: 'venus' | 'saturn' | 'jupiter' | 'mercury' | 'uranus' | 'neptune' | 'sun', skipPush = false) {
  if (!skipPush) window.history.pushState(null, '', '#solar-system')
  preloadSurfaceComponent('solar-system')
  preloadSolarTextures()
  cancelPendingTransition()
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  // 来源标志：置当前行星 true，其余全部清空（SolarSystem 据此恢复初始选中）
  setSolarEnterFromOuter(key)
  solarEnterFromOrbit.value = false
  solarEnterFromMoon.value = false
  solarEnterFromMars.value = false
  window.scrollTo({ top: 0, behavior: reduced ? 'auto' : 'smooth' })
  const leavingRefs: Record<string, ReturnType<typeof ref<boolean>>> = {
    venus: venusLeaving, saturn: saturnLeaving, jupiter: jupiterLeaving,
    mercury: mercuryLeaving, uranus: uranusLeaving, neptune: neptuneLeaving, sun: sunLeaving,
  }
  leavingRefs[key].value = true
  headerExpanded.value = false
  suppressHeaderReveal = true
  transitionTimer = window.setTimeout(() => {
    veilDuration.value = reduced ? '0.01s' : '0.3s'
    veilActive.value = true
    waitUntilFullBlack(() => {
      if (surfaceFromHash() !== 'solar-system') {
        cancelPendingTransition()
        return
      }
      void setSurface('solar-system')
      requestAnimationFrame(() => {
        veilDuration.value = reduced ? '0.01s' : '0.3s'
        veilActive.value = false
      })
      transitionTimer = undefined
    })
  }, reduced ? 20 : 450)
}

/** 金星 → 太阳系（skipPush = 浏览器返回路径） */
function enterSolarSystemFromVenus(skipPush = false) {
  enterSolarSystemFromOuterPlanet('venus', skipPush)
}

/** 土星 → 太阳系（skipPush = 浏览器返回路径） */
function enterSolarSystemFromSaturn(skipPush = false) {
  enterSolarSystemFromOuterPlanet('saturn', skipPush)
}

/** 木星 → 太阳系（skipPush = 浏览器返回路径） */
function enterSolarSystemFromJupiter(skipPush = false) {
  enterSolarSystemFromOuterPlanet('jupiter', skipPush)
}

/** 水星 → 太阳系（skipPush = 浏览器返回路径） */
function enterSolarSystemFromMercury(skipPush = false) {
  enterSolarSystemFromOuterPlanet('mercury', skipPush)
}

/** 天王星 → 太阳系（skipPush = 浏览器返回路径） */
function enterSolarSystemFromUranus(skipPush = false) {
  enterSolarSystemFromOuterPlanet('uranus', skipPush)
}

/** 海王星 → 太阳系（skipPush = 浏览器返回路径） */
function enterSolarSystemFromNeptune(skipPush = false) {
  enterSolarSystemFromOuterPlanet('neptune', skipPush)
}

/** 太阳 → 太阳系（skipPush = 浏览器返回路径） */
function enterSolarSystemFromSun(skipPush = false) {
  enterSolarSystemFromOuterPlanet('sun', skipPush)
}

/** 月球 → 太阳系：渐暗 → 切页（太阳系从月球近景拉回）→ 渐亮 */
/** 月球 → 太阳系（skipPush = 浏览器返回路径） */
function enterSolarSystemFromMoon(skipPush = false) {
  if (!skipPush) window.history.pushState(null, '', '#solar-system')
  preloadSurfaceComponent('solar-system')
  preloadSolarTextures()
  cancelPendingTransition()
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  solarEnterFromMoon.value = true
  solarEnterFromOrbit.value = false // 清空地球来源遗留
  solarEnterFromMars.value = false // 清空火星来源遗留
  solarEnterFromVenus.value = false // 清空金星来源遗留
  solarEnterFromSaturn.value = false // 清空土星来源遗留
  solarEnterFromJupiter.value = false // 清空木星来源遗留
  // 阶段 1：滚回月球主视图 + 清空月球以外的所有元素（标记/飞行器/标签），只留月球球体；
  // 页头若展开则随之上滑消失（与地球返回一致），过渡期间 hover 不唤回
  window.scrollTo({ top: 0, behavior: reduced ? 'auto' : 'smooth' })
  moonLeaving.value = true
  headerExpanded.value = false
  suppressHeaderReveal = true
  // 阶段 2（清空效果可见后才变暗——与地球返回"信息淡出只留地球"同节奏）：变暗 300ms（原 400ms——加速，卡顿窗口缩短）
  transitionTimer = window.setTimeout(() => {
    veilDuration.value = reduced ? '0.01s' : '0.3s'
    veilActive.value = true
    // 阶段 3：等 veil 真正全黑再切页（同地球返回——避免新旧场景首帧透过遮罩叠影）
    waitUntilFullBlack(() => {
      if (surfaceFromHash() !== 'solar-system') {
        cancelPendingTransition()
        return
      }
      void setSurface('solar-system')
      requestAnimationFrame(() => {
        veilDuration.value = reduced ? '0.01s' : '0.3s' // 渐亮 300ms（原 500ms 太长）
        veilActive.value = false
      })
      transitionTimer = undefined
    })
  }, reduced ? 20 : 450)
}

// （已移除）syncSurfaceFromHash：hashchange 会在浏览器返回时与 popstate 竞争——
// 直接 setSurface 无动画切页，把 popstate 的渐暗动画整体取消。
// 所有 hash 变化均为 pushState 产生，动画流程自管 surface，popstate 统一处理前进/后退即可。

function clearHeaderIdleTimer() {
  if (headerIdleTimer !== undefined) window.clearTimeout(headerIdleTimer)
  headerIdleTimer = undefined
}

function scheduleHeaderCollapse() {
  clearHeaderIdleTimer()
  if (!collapsibleHeaderActive()) {
    headerExpanded.value = true
    return
  }
  headerIdleTimer = window.setTimeout(() => {
    if (!collapsibleHeaderActive()) {
      headerExpanded.value = true
      headerIdleTimer = undefined
      return
    }
    if (siteHeader.value?.matches(':hover') || siteHeader.value?.contains(document.activeElement)) {
      scheduleHeaderCollapse()
      return
    }
    headerExpanded.value = false
    headerIdleTimer = undefined
  }, 5000)
}

function revealHeader() {
  if (suppressHeaderReveal) return // 返回太阳系过渡期间：页头已上滑消失，hover 不唤回
  headerExpanded.value = true
  if (collapsibleHeaderActive()) scheduleHeaderCollapse()
  else clearHeaderIdleTimer()
}

function registerHeaderActivity() {
  if (!collapsibleHeaderActive() || !headerExpanded.value) return
  const activityAt = performance.now()
  if (activityAt - lastHeaderActivityAt < 400) return
  lastHeaderActivityAt = activityAt
  scheduleHeaderCollapse()
}

function handleWindowPointerMove(event: PointerEvent) {
  if (!headerExpanded.value && collapsibleHeaderActive() && event.clientY <= 16) {
    revealHeader()
    return
  }
  registerHeaderActivity()
}

function collapseHeaderFromScene() {
  if (!collapsibleHeaderActive()) return
  clearHeaderIdleTimer()
  headerExpanded.value = false
}

function updateActivePage() {
  pageSurfaceFrame = 0
  if (surface.value === 'venus' || surface.value === 'saturn' || surface.value === 'jupiter' || surface.value === 'mercury' || surface.value === 'uranus' || surface.value === 'neptune' || surface.value === 'sun') {
    // 主视图 = #<planet>-profile（第一个板块）顶部仍在视口下半区；滑到第一个板块即展开页头
    const planetKey = surface.value
    const profileSection = document.getElementById(`${planetKey}-profile`)
    const profileTop = profileSection?.getBoundingClientRect().top ?? window.innerHeight
    const nextPageActive = profileTop > window.innerHeight / 2
    const activeRef = planetKey === 'venus' ? venusPageActive : planetKey === 'saturn' ? saturnPageActive
      : planetKey === 'jupiter' ? jupiterPageActive : planetKey === 'mercury' ? mercuryPageActive
      : planetKey === 'uranus' ? uranusPageActive : planetKey === 'neptune' ? neptunePageActive : sunPageActive
    if (nextPageActive === activeRef.value) {
      if (activeRef.value && headerExpanded.value) scheduleHeaderCollapse()
      return
    }
    activeRef.value = nextPageActive
    clearHeaderIdleTimer()
    headerExpanded.value = !nextPageActive
    if (activeRef.value && headerExpanded.value) scheduleHeaderCollapse()
    return
  }
  if (surface.value === 'moon') {
    // 主视图 = #moon-profile（第一个资料板块）顶部仍在视口下半区；滑到档案即展开页头
    const moonProfile = document.getElementById('moon-profile')
    const profileTop = moonProfile?.getBoundingClientRect().top ?? window.innerHeight
    const nextMoonPageActive = profileTop > window.innerHeight / 2
    if (nextMoonPageActive === moonPageActive.value) {
      if (moonPageActive.value && headerExpanded.value) scheduleHeaderCollapse()
      return
    }
    moonPageActive.value = nextMoonPageActive
    clearHeaderIdleTimer()
    headerExpanded.value = !nextMoonPageActive
    if (moonPageActive.value && headerExpanded.value) scheduleHeaderCollapse()
    return
  }
  if (surface.value === 'mars') {
    // 主视图 = #mars-profile（第一个板块）顶部仍在视口下半区；滑到第一个板块即展开页头
    const marsObjects = document.getElementById('mars-profile')
    const objectsTop = marsObjects?.getBoundingClientRect().top ?? window.innerHeight
    const nextMarsPageActive = objectsTop > window.innerHeight / 2
    if (nextMarsPageActive === marsPageActive.value) {
      if (marsPageActive.value && headerExpanded.value) scheduleHeaderCollapse()
      return
    }
    marsPageActive.value = nextMarsPageActive
    clearHeaderIdleTimer()
    headerExpanded.value = !nextMarsPageActive
    if (marsPageActive.value && headerExpanded.value) scheduleHeaderCollapse()
    return
  }
  if (surface.value !== 'orbit') {
    orbitPageActive.value = false
    headerExpanded.value = true
    clearHeaderIdleTimer()
    return
  }
  // 是否处于主地球视图：用普通流元素（航天器区块）的视口位置判断——
  // 场景区是 sticky（offsetTop 返回粘住后的位置=scrollY，会退化恒真），不能作为参照；
  // 主视图 = #objects 顶部仍在视口下半区；进入上半区（滚动超过主场景区一半）即进入下方页面
  const objectsSection = document.getElementById('objects')
  const objectsTop = objectsSection?.getBoundingClientRect().top ?? window.innerHeight
  // 主视图 = #objects 顶部仍在视口下半区；进入上半区（滚动超过主场景区一半）即进入下方页面
  const nextOrbitPageActive = objectsTop > window.innerHeight / 2
  if (nextOrbitPageActive === orbitPageActive.value) {
    // 滚动/拖动滚动条期间持续重置闲置收起计时（页头不会中途缩回；停止滚动 2.4s 后才收起）
    if (orbitPageActive.value && headerExpanded.value) scheduleHeaderCollapse()
    return
  }
  orbitPageActive.value = nextOrbitPageActive
  clearHeaderIdleTimer()
  headerExpanded.value = !nextOrbitPageActive
  if (orbitPageActive.value && headerExpanded.value) scheduleHeaderCollapse()
}

function handlePageScroll() {
  if (pageSurfaceFrame) return
  pageSurfaceFrame = window.requestAnimationFrame(updateActivePage)
}

/** 每日圈数 → 轨道周期（分钟） */


function eventDate(value: string) {
  const date = new Date(value)
  return {
    day: new Intl.DateTimeFormat('zh-CN', { day: '2-digit', timeZone: DISPLAY_TIME_ZONE }).format(date),
    month: new Intl.DateTimeFormat('en-US', { month: 'short', timeZone: DISPLAY_TIME_ZONE }).format(date).toUpperCase(),
    weekday: new Intl.DateTimeFormat('zh-CN', { weekday: 'short', timeZone: DISPLAY_TIME_ZONE }).format(date),
    time: new Intl.DateTimeFormat('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false, timeZone: DISPLAY_TIME_ZONE }).format(date),
  }
}
function formatCoordinate(value: number, positive: string, negative: string) {
  return `${Math.abs(value).toFixed(2)}° ${value >= 0 ? positive : negative}`
}

let overviewRequest: Promise<void> | null = null

function ensureOrbitOverview() {
  if (overview.value) return Promise.resolve()
  if (overviewRequest) return overviewRequest
  overviewRequest = load().finally(() => {
    overviewRequest = null
  })
  return overviewRequest
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    overview.value = await fetchOrbitOverview()
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : '无法连接数据服务'
  } finally {
    loading.value = false
  }
}

/** 浏览器前进/后退：hash 变化 → 走与按钮一致的动画（不重复 pushState） */
function onPopState() {
  const target = surfaceFromHash()
  if (target === surface.value) return
  if (target === 'solar-system') {
    if (surface.value === 'orbit') enterSolarSystemFromOrbit(true)
    else if (surface.value === 'moon') enterSolarSystemFromMoon(true)
    else if (surface.value === 'mars') enterSolarSystemFromMars(true)
    else if (surface.value === 'venus') enterSolarSystemFromVenus(true)
    else if (surface.value === 'saturn') enterSolarSystemFromSaturn(true)
    else if (surface.value === 'jupiter') enterSolarSystemFromJupiter(true)
    else if (surface.value === 'mercury') enterSolarSystemFromMercury(true)
    else if (surface.value === 'uranus') enterSolarSystemFromUranus(true)
    else if (surface.value === 'neptune') enterSolarSystemFromNeptune(true)
    else if (surface.value === 'sun') enterSolarSystemFromSun(true)
  } else if (target === 'cover') {
    returnToCover(true)
  } else if (target === 'sky') {
    cancelPendingTransition()
    void setSurface('sky')
  } else if (target === 'orbit') {
    cancelPendingTransition()
    void setSurface('orbit')
  } else if (target === 'moon') {
    cancelPendingTransition()
    void setSurface('moon')
  } else if (target === 'mars') {
    cancelPendingTransition()
    void setSurface('mars')
  } else if (target === 'venus' || target === 'saturn' || target === 'jupiter' || target === 'mercury' || target === 'uranus' || target === 'neptune' || target === 'sun') {
    cancelPendingTransition()
    void setSurface(target)
  }
}

/** ESC 键：返回上一级（太阳系）——与点击"太阳系"按钮完全一致 */
function onGlobalKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape') return
  const target = event.target as HTMLElement | null
  if (target && ['INPUT', 'SELECT', 'TEXTAREA'].includes(target.tagName)) return
  if (surface.value === 'orbit' || surface.value === 'moon' || surface.value === 'mars' || surface.value === 'venus' || surface.value === 'saturn' || surface.value === 'jupiter' || surface.value === 'mercury' || surface.value === 'uranus' || surface.value === 'neptune' || surface.value === 'sun') {
    event.preventDefault()
    returnToSolarSystem()
  } else if (surface.value === 'sky') {
    event.preventDefault()
    window.history.pushState(null, '', '#home')
    returnToCover()
  }
}

onMounted(() => {
  window.addEventListener('popstate', onPopState)
  window.addEventListener('keydown', onGlobalKeydown)
  // 首页保持轻量、无权限请求；只有用户进入深空路径或直接打开相应页面时才预热。
  if (surface.value === 'sky') {
    preloadSurfaceComponent('sky')
  } else if (surface.value === 'solar-system') {
    preloadSurfaceComponent('solar-system')
    preloadSolarTextures()
    void ensureOrbitOverview()
  } else if (surface.value === 'orbit') {
    preloadSurfaceComponent('orbit')
    preloadOrbitTextures()
    void ensureOrbitOverview()
    requestObserverLocation()
  } else if (surface.value === 'moon') {
    preloadSurfaceComponent('moon')
    preloadMoonHdTexture()
  } else if (surface.value === 'mars') {
    preloadSurfaceComponent('mars')
    preloadMarsHdTexture()
  } else if (surface.value === 'venus' || surface.value === 'saturn' || surface.value === 'jupiter' || surface.value === 'mercury' || surface.value === 'uranus' || surface.value === 'neptune' || surface.value === 'sun') {
    preloadSurfaceComponent(surface.value)
    preloadSolarTextures() // 纹理与太阳系同源（ALL_TEXTURE_URLS 已含），复用预热
  }
  if (surface.value === 'orbit') void loadCatalogPage()
  document.title = surface.value === 'cover'
    ? 'AURORA'
    : surface.value === 'sky' ? 'AURORA · SKY'
    : surface.value === 'solar-system' ? 'AURORA · 太阳系'
    : surface.value === 'moon' ? 'AURORA · MOON'
    : surface.value === 'mars' ? 'AURORA · MARS'
    : surface.value === 'venus' ? 'AURORA · VENUS'
    : surface.value === 'saturn' ? 'AURORA · SATURN'
    : surface.value === 'jupiter' ? 'AURORA · JUPITER'
    : surface.value === 'mercury' ? 'AURORA · MERCURY'
    : surface.value === 'uranus' ? 'AURORA · URANUS'
    : surface.value === 'neptune' ? 'AURORA · NEPTUNE'
    : surface.value === 'sun' ? 'AURORA · SUN' : 'AURORA · EARTH'
  updateActivePage()
  scheduleHeaderCollapse()
  window.addEventListener('pointermove', handleWindowPointerMove, { passive: true })
  window.addEventListener('pointerdown', registerHeaderActivity, { passive: true })
  window.addEventListener('wheel', registerHeaderActivity, { passive: true })
  window.addEventListener('keydown', registerHeaderActivity)
  window.addEventListener('scroll', handlePageScroll, { passive: true })
  // 不监听 hashchange：与 popstate 竞争会跳过渐暗动画（见 syncSurfaceFromHash 说明）
  clock = window.setInterval(() => { now.value = new Date() }, 1000)
})
onBeforeUnmount(() => {
  if (clock) window.clearInterval(clock)
  if (pageSurfaceFrame) window.cancelAnimationFrame(pageSurfaceFrame)
  clearHeaderIdleTimer()
  cancelPendingTransition()
  observerLocationRevision += 1
  observerLookupController?.abort()
  catalogRequest?.abort()
  if (catalogQueryTimer !== undefined) window.clearTimeout(catalogQueryTimer)
  window.removeEventListener('pointermove', handleWindowPointerMove)
  window.removeEventListener('pointerdown', registerHeaderActivity)
  window.removeEventListener('wheel', registerHeaderActivity)
  window.removeEventListener('keydown', registerHeaderActivity)
  window.removeEventListener('scroll', handlePageScroll)
  // hashchange 监听已移除（popstate 统一处理）
})
</script>

<template>
  <div ref="surfaceVeilRef" class="surface-veil" :class="{ active: veilActive }" :style="{ '--veil-duration': veilDuration }" aria-hidden="true" />
  <main class="aurora-shell" :style="shellStyle">
    <div class="desktop-only">
      <span>AURORA / ORBIT</span>
      <h1>请使用电脑浏览器查看</h1>
      <p>当前原型专注桌面端三维交互，移动端适配将在后续阶段加入。</p>
    </div>

    <AuroraCover
      v-if="surface === 'cover' || coverLingering"
      class="desktop-cover"
      :class="{ lingering: coverLingering }"
      @explore="enterSolarSystem"
      @astronomy="enterSky"
    />

    <div v-show="surface !== 'cover' || coverLingering" class="desktop-app" :class="{ 'header-collapsed': !headerExpanded, sky: surface === 'sky', moon: surface === 'moon', mars: surface === 'mars', venus: surface === 'venus', saturn: surface === 'saturn', jupiter: surface === 'jupiter', mercury: surface === 'mercury', uranus: surface === 'uranus', neptune: surface === 'neptune', sun: surface === 'sun' }">
      <SkyObservatory v-if="surface === 'sky'" @home="returnToCover" />

      <header
        v-if="surface !== 'sky'"
        ref="siteHeader"
        class="site-header"
        :class="{ collapsed: !headerExpanded }"
        @mouseenter="clearHeaderIdleTimer"
        @mouseleave="scheduleHeaderCollapse"
        @focusin="revealHeader"
        @focusout="scheduleHeaderCollapse"
      >
        <div class="page-frame header-inner">
          <div class="header-left">
            <!-- 不 prevent：让浏览器原生执行 href="#home" fragment 导航（该环境禁止 JS 导航 API，
                 但原生同文档 hash 跳转不受限——导航栏链接一直可用即证明）；returnToCover 负责过渡动画 -->
            <a class="brand" href="#home" aria-label="返回 AURORA 封面" @click="() => returnToCover()">
              <span class="brand-mark"><i /><i /><i /></span>
              <span><strong>AURORA</strong><small>ORBITAL OBSERVATORY</small></span>
            </a>
            <SolarSystemItem v-if="surface === 'orbit' || surface === 'moon' || surface === 'mars' || surface === 'venus' || surface === 'saturn' || surface === 'jupiter' || surface === 'mercury' || surface === 'uranus' || surface === 'neptune' || surface === 'sun'" title="太阳系" :icon-size="30" :animated="true" @click="enterSolarSystem" />
          </div>
          <nav v-if="surface === 'orbit'" aria-label="页面导航">
            <a href="#earth"><i class="nav-num">Ⅰ</i>地球</a>
            <a href="#objects"><i class="nav-num">Ⅱ</i>飞行器</a>
            <a href="#sites"><i class="nav-num">Ⅲ</i>发射场</a>
            <a href="#launches"><i class="nav-num">Ⅳ</i>发射日程</a>
          </nav>
          <nav v-else-if="surface === 'moon'" aria-label="页面导航">
            <a href="#moon-scene"><i class="nav-num">Ⅰ</i>月球</a>
            <a href="#moon-profile"><i class="nav-num">Ⅱ</i>档案</a>
            <a href="#moon-objects"><i class="nav-num">Ⅲ</i>飞行器</a>
            <a href="#moon-sites"><i class="nav-num">Ⅳ</i>着陆点</a>
          </nav>
          <nav v-else-if="surface === 'mars'" aria-label="页面导航">
            <a href="#mars-scene"><i class="nav-num">Ⅰ</i>火星</a>
            <a href="#mars-profile"><i class="nav-num">Ⅱ</i>档案</a>
            <a href="#mars-objects"><i class="nav-num">Ⅲ</i>飞行器</a>
            <a href="#mars-sites"><i class="nav-num">Ⅳ</i>着陆点</a>
          </nav>
          <nav v-else-if="surface === 'venus'" aria-label="页面导航">
            <a href="#venus-scene"><i class="nav-num">Ⅰ</i>金星</a>
            <a href="#venus-profile"><i class="nav-num">Ⅱ</i>档案</a>
            <a href="#venus-objects"><i class="nav-num">Ⅲ</i>飞行器</a>
            <a href="#venus-sites"><i class="nav-num">Ⅳ</i>着陆点</a>
          </nav>
          <nav v-else-if="surface === 'saturn'" aria-label="页面导航">
            <a href="#saturn-scene"><i class="nav-num">Ⅰ</i>土星</a>
            <a href="#saturn-profile"><i class="nav-num">Ⅱ</i>档案</a>
            <a href="#saturn-objects"><i class="nav-num">Ⅲ</i>飞行器</a>
            <a href="#saturn-sites"><i class="nav-num">Ⅳ</i>任务终点</a>
          </nav>
          <nav v-else-if="surface === 'jupiter'" aria-label="页面导航">
            <a href="#jupiter-scene"><i class="nav-num">Ⅰ</i>木星</a>
            <a href="#jupiter-profile"><i class="nav-num">Ⅱ</i>档案</a>
            <a href="#jupiter-objects"><i class="nav-num">Ⅲ</i>飞行器</a>
            <a href="#jupiter-sites"><i class="nav-num">Ⅳ</i>任务终点</a>
          </nav>
          <nav v-else-if="surface === 'mercury'" aria-label="页面导航">
            <a href="#mercury-scene"><i class="nav-num">Ⅰ</i>水星</a>
            <a href="#mercury-profile"><i class="nav-num">Ⅱ</i>档案</a>
            <a href="#mercury-objects"><i class="nav-num">Ⅲ</i>飞行器</a>
            <a href="#mercury-sites"><i class="nav-num">Ⅳ</i>任务终点</a>
          </nav>
          <nav v-else-if="surface === 'uranus'" aria-label="页面导航">
            <a href="#uranus-scene"><i class="nav-num">Ⅰ</i>天王星</a>
            <a href="#uranus-profile"><i class="nav-num">Ⅱ</i>档案</a>
            <a href="#uranus-objects"><i class="nav-num">Ⅲ</i>飞掠器</a>
          </nav>
          <nav v-else-if="surface === 'neptune'" aria-label="页面导航">
            <a href="#neptune-scene"><i class="nav-num">Ⅰ</i>海王星</a>
            <a href="#neptune-profile"><i class="nav-num">Ⅱ</i>档案</a>
            <a href="#neptune-objects"><i class="nav-num">Ⅲ</i>飞掠器</a>
          </nav>
          <nav v-else-if="surface === 'sun'" aria-label="页面导航">
            <a href="#sun-scene"><i class="nav-num">Ⅰ</i>太阳</a>
            <a href="#sun-profile"><i class="nav-num">Ⅱ</i>档案</a>
            <a href="#sun-objects"><i class="nav-num">Ⅲ</i>飞行器</a>
          </nav>
          <nav v-else-if="surface === 'solar-system'" aria-label="当前位置">
            <SolarSystemItem title="太阳系" :icon-size="30" :active="true" :animated="true" @click="solarSystemRef?.resetView?.()" />
          </nav>
          <!-- 数据健康灯（只保留一个）：移到太阳系页——深空探测器数据源（CelesTrak/Launch Library/JPL Horizons）
               最近一次同步成败的 3 合 1 聚合；地球/月球页保持身份标签 -->
          <div v-if="surface === 'solar-system'" class="live-status">
            <span class="status-dot" :class="{ healthy: !!overview && dataHealthy, syncing: loading }" />
            <span>{{ loading ? '同步中' : dataHealthy && !!overview ? '数据正常' : '检查数据' }}</span>
            <strong>{{ timeOnly(now) }} UTC+8</strong>
          </div>
          <div v-else class="live-status solar-clock">
            <span>{{ surface === 'moon' ? '月球 · MOON' : surface === 'mars' ? '火星 · MARS' : surface === 'venus' ? '金星 · VENUS' : surface === 'saturn' ? '土星 · SATURN' : surface === 'jupiter' ? '木星 · JUPITER' : surface === 'mercury' ? '水星 · MERCURY' : surface === 'uranus' ? '天王星 · URANUS' : surface === 'neptune' ? '海王星 · NEPTUNE' : surface === 'sun' ? '太阳 · SUN' : '地球 · EARTH' }}</span>
          </div>
        </div>
      </header>

      <MoonScene v-if="surface === 'moon'" :reveal-tick="moonRevealTick" :enter-from-solar="moonEnterFromSolar" :leaving="moonLeaving" :header-expanded="headerExpanded" @blank-click="collapseHeaderFromScene" @textures-ready="onMoonSceneReady" />

      <MarsScene v-if="surface === 'mars'" :reveal-tick="marsRevealTick" :enter-from-solar="marsEnterFromSolar" :leaving="marsLeaving" :header-expanded="headerExpanded" @blank-click="collapseHeaderFromScene" @textures-ready="onMarsSceneReady" />

      <PlanetScene
        v-if="surface === 'venus'"
        :planet="VENUS_PAGE"
        :reveal-tick="venusRevealTick"
        :enter-from-solar="venusEnterFromSolar"
        :leaving="venusLeaving"
        :header-expanded="headerExpanded"
        @blank-click="collapseHeaderFromScene"
        @textures-ready="onVenusSceneReady"
      />

      <PlanetScene
        v-if="surface === 'saturn'"
        :planet="SATURN_PAGE"
        :reveal-tick="saturnRevealTick"
        :enter-from-solar="saturnEnterFromSolar"
        :leaving="saturnLeaving"
        :header-expanded="headerExpanded"
        @blank-click="collapseHeaderFromScene"
        @textures-ready="onSaturnSceneReady"
      />

      <PlanetScene
        v-if="surface === 'jupiter'"
        :planet="JUPITER_PAGE"
        :reveal-tick="jupiterRevealTick"
        :enter-from-solar="jupiterEnterFromSolar"
        :leaving="jupiterLeaving"
        :header-expanded="headerExpanded"
        @blank-click="collapseHeaderFromScene"
        @textures-ready="onJupiterSceneReady"
      />

      <PlanetScene
        v-if="surface === 'mercury'"
        :planet="MERCURY_PAGE"
        :reveal-tick="mercuryRevealTick"
        :enter-from-solar="mercuryEnterFromSolar"
        :leaving="mercuryLeaving"
        :header-expanded="headerExpanded"
        @blank-click="collapseHeaderFromScene"
        @textures-ready="onMercurySceneReady"
      />

      <PlanetScene
        v-if="surface === 'uranus'"
        :planet="URANUS_PAGE"
        :reveal-tick="uranusRevealTick"
        :enter-from-solar="uranusEnterFromSolar"
        :leaving="uranusLeaving"
        :header-expanded="headerExpanded"
        @blank-click="collapseHeaderFromScene"
        @textures-ready="onUranusSceneReady"
      />

      <PlanetScene
        v-if="surface === 'neptune'"
        :planet="NEPTUNE_PAGE"
        :reveal-tick="neptuneRevealTick"
        :enter-from-solar="neptuneEnterFromSolar"
        :leaving="neptuneLeaving"
        :header-expanded="headerExpanded"
        @blank-click="collapseHeaderFromScene"
        @textures-ready="onNeptuneSceneReady"
      />

      <PlanetScene
        v-if="surface === 'sun'"
        :planet="SUN_PAGE"
        :reveal-tick="sunRevealTick"
        :enter-from-solar="sunEnterFromSolar"
        :leaving="sunLeaving"
        :header-expanded="headerExpanded"
        @blank-click="collapseHeaderFromScene"
        @textures-ready="onSunSceneReady"
      />

      <SolarSystem
        ref="solarSystemRef"
        v-if="surface === 'solar-system'"
        :enter-from-orbit="solarEnterFromOrbit"
        :enter-from-moon="solarEnterFromMoon"
        :enter-from-mars="solarEnterFromMars"
        :enter-from-venus="solarEnterFromVenus"
        :enter-from-saturn="solarEnterFromSaturn"
        :enter-from-jupiter="solarEnterFromJupiter"
        :enter-from-mercury="solarEnterFromMercury"
        :enter-from-uranus="solarEnterFromUranus"
        :enter-from-neptune="solarEnterFromNeptune"
        :enter-from-sun="solarEnterFromSun"
        :fly-delay="solarFlyDelay"
        :play-entry-fly="solarEntryFly"
        @select-earth="onEarthSelect"
        @earth-fly-start="onEarthFlyStart"
        @earth-fly-zoom="onEarthFlyZoom"
        @moon-fly-start="onMoonFlyStart"
        @moon-fly-zoom="onMoonFlyZoom"
        @select-moon="onMoonSelect"
        @mars-fly-start="onMarsFlyStart"
        @mars-fly-zoom="onMarsFlyZoom"
        @select-mars="onMarsSelect"
        @venus-fly-start="onVenusFlyStart"
        @venus-fly-zoom="onVenusFlyZoom"
        @select-venus="onVenusSelect"
        @saturn-fly-start="onSaturnFlyStart"
        @saturn-fly-zoom="onSaturnFlyZoom"
        @select-saturn="onSaturnSelect"
        @jupiter-fly-start="onJupiterFlyStart"
        @jupiter-fly-zoom="onJupiterFlyZoom"
        @select-jupiter="onJupiterSelect"
        @mercury-fly-start="onMercuryFlyStart"
        @mercury-fly-zoom="onMercuryFlyZoom"
        @select-mercury="onMercurySelect"
        @uranus-fly-start="onUranusFlyStart"
        @uranus-fly-zoom="onUranusFlyZoom"
        @select-uranus="onUranusSelect"
        @neptune-fly-start="onNeptuneFlyStart"
        @neptune-fly-zoom="onNeptuneFlyZoom"
        @select-neptune="onNeptuneSelect"
        @sun-fly-start="onSunFlyStart"
        @sun-fly-zoom="onSunFlyZoom"
        @select-sun="onSunSelect"
      />

      <template v-else-if="surface === 'orbit'">
      <section id="earth" ref="orbitSection" class="orbit-section" :class="{ leaving: orbitSectionLeaving, 'elements-revealed': orbitElementsRevealed }">
        <div class="page-frame">
          <div ref="orbitSceneFrame" class="scene-frame">
            <OrbitScene
              v-if="overview"
              :spacecraft="overview.spacecraft"
              :sites="overview.launchSites"
              :events="overview.events"
              :header-expanded="headerExpanded"
              :layers="layers"
              :selection="selection"
              :focus-target="focusTarget"
              :observer-target="observerTarget"
              :observer-active="observerViewActive"
              :day-night-enabled="dayNightEnabled"
              :reveal-tick="orbitRevealTick"
              :leaving="orbitSectionLeaving"
              @textures-ready="onOrbitSceneReady"
              @select="selectFromScene"
              @clear-selection="selection = null"
              @view-change="leaveObserverView"
              @blank-click="collapseHeaderFromScene"
            />

            <div ref="sceneToolbarRef" class="scene-toolbar" aria-label="场景图层">
              <span>图层</span>
              <label><input v-model="layers.spacecraft" type="checkbox"><i />飞行器</label>
              <label><input v-model="layers.orbits" type="checkbox"><i />轨道</label>
              <label><input v-model="layers.sites" type="checkbox"><i class="amber" />发射场</label>
              <label><input v-model="dayNightEnabled" type="checkbox"><i class="terminator" />晨昏线</label>
            </div>

            <button
              ref="sceneLocationRef"
              class="scene-location"
              :class="{ active: observerViewActive }"
              v-show="orbitElementsRevealed"
              type="button"
              :aria-pressed="observerViewActive"
              :aria-label="observerViewActive ? `当前视角位于${observerLocation.label}` : `返回${observerLocation.label}`"
              @click="focusObserver"
            >
              <i :class="observerLocation.status" />
              <span><small>{{ observerViewActive ? '当前中心' : '返回当前位置' }}</small><strong>{{ observerLocation.label }}</strong></span>
            </button>

            <div class="scene-readout" aria-label="当前视角">
              <span>EARTH ORBIT</span>
              <strong>地球</strong>
            </div>
            <div class="scene-credits" aria-hidden="true">NASA Blue Marble / Earth at Night</div>

            

            <section v-if="loading" class="system-message"><strong>正在建立轨道数据链路</strong><small>CONNECTING TO AURORA CORE</small></section>
            <section v-else-if="error" class="system-message error-message"><strong>数据链路未建立</strong><p>{{ error }}</p><button @click="load">重新连接</button></section>
          </div>
        </div>
      </section>

      <section id="objects" class="content-section objects-section">
        <div class="page-frame">
          <div class="section-heading">
            <div><p class="section-kicker">SPACECRAFT CATALOG</p><h2><i class="sec-num">Ⅱ</i>飞行器</h2><p class="section-sub">飞行器 · 轨道与位置用于交互示意；精确状态以数据来源为准。</p></div>
          </div>

          <div class="catalog-workspace">
            <div class="catalog-controls">
              <label class="search-field">
                <span>名称、NORAD 编号、运营方或正则表达式</span>
                <input v-model="objectQuery" type="search" placeholder="输入 ISS，或使用 /ISS|TIANHE/i" spellcheck="false">
              </label>
              <label><span>运营方</span><select v-model="operatorFilter"><option value="all">全部运营方</option><option v-for="operator in operators" :key="operator" :value="operator">{{ operator }}</option></select></label>
              <label><span>排序</span><select v-model="objectSort"><option value="name">名称</option><option value="norad">NORAD 编号</option><option value="operator">运营方</option></select></label>
            </div>
            <p v-if="catalogResult.error || catalogRequestError" class="query-error">{{ catalogResult.error || catalogRequestError }}</p>
            <div class="object-table" role="table" aria-label="飞行器查询结果">
              <div class="object-table-head" role="row"><span>NORAD</span><span>对象</span><span>运营方</span><span>类型</span><span>轨道历元</span></div>
              <button v-for="craft in pagedCatalogItems" :key="craft.id" class="object-row" role="row" @click="selectAndFocus({ kind: craft.kind, id: craft.id })">
                <span>{{ craft.noradCatalogId ?? '—' }}</span>
                <span><strong>{{ catalogBilingual(craft).primary }}</strong><small v-if="catalogBilingual(craft).secondary">（{{ catalogBilingual(craft).secondary }}）</small></span>
                <span>{{ craft.operatorName }}</span>
                <span>{{ craft.category }}</span>
                <span>{{ formatUTCDate(craft.orbitEpoch) }}</span>
              </button>
              <div v-if="catalogLoading" class="catalog-empty">正在查询飞行器目录…</div>
              <div v-else-if="!pagedCatalogItems.length" class="catalog-empty">没有符合当前条件的飞行器。请修改搜索词或筛选条件。</div>
            </div>
            <div class="pagination-space"><span>第 {{ catalogPage }} / {{ catalogPageCount }} 页 · {{ catalogTotal }} 个飞行器</span><div><button :disabled="catalogPage <= 1 || catalogLoading" @click="catalogGotoPage(-1)">上一页</button><button :disabled="catalogPage >= catalogPageCount || catalogLoading" @click="catalogGotoPage(1)">下一页</button></div></div>
          </div>
        </div>
      </section>

      <section id="sites" class="content-section sites-section">
        <div class="page-frame">
          <div class="section-heading">
            <div><p class="section-kicker launch-kicker">GROUND NETWORK</p><h2><i class="sec-num">Ⅲ</i>发射场</h2></div>
          </div>
          <div class="site-directory">
            <button v-for="site in pagedLaunchSites" :key="site.id" @click="selectAndFocus({ kind: 'site', id: site.id })">
              <span class="site-code">{{ site.countryCode }}</span>
              <span><strong>{{ bilingualName(site.nameZh, site.nameEn).primary }}</strong><small v-if="bilingualName(site.nameZh, site.nameEn).secondary">（{{ bilingualName(site.nameZh, site.nameEn).secondary }}）</small></span>
              <p>{{ site.description }}</p>
              <span class="site-coordinate">{{ formatCoordinate(site.latitude, 'N', 'S') }}<br>{{ formatCoordinate(site.longitude, 'E', 'W') }}</span>
            </button>
          </div>
          <div class="pagination-space"><span>第 {{ launchSitePage }} / {{ launchSitePageCount }} 页 · {{ overview?.launchSites.length ?? 0 }} 个发射场</span><div><button :disabled="launchSitePage <= 1" @click="launchSiteGotoPage(-1)">上一页</button><button :disabled="launchSitePage >= launchSitePageCount" @click="launchSiteGotoPage(1)">下一页</button></div></div>
        </div>
      </section>

      <section id="launches" class="content-section launches-section">
        <div class="page-frame">
          <div class="section-heading">
            <div><p class="section-kicker launch-kicker">NEXT 30 DAYS</p><h2><i class="sec-num">Ⅳ</i>发射日程</h2></div>
          </div>
          <div class="launch-list">
            <div class="launch-list-head"><span>日期</span><span>任务</span><span>状态</span><span>发射地点</span><span>时间</span></div>
            <button v-for="event in visibleLaunchEvents" :key="event.externalId" class="launch-row" @click="selectAndFocus({ kind: 'event', id: event.externalId })">
              <time><strong>{{ eventDate(event.net).day }}</strong><span>{{ eventDate(event.net).month }} · {{ eventDate(event.net).weekday }}</span></time>
              <span class="launch-mission">
                <strong :lang="eventBilingual(event).lang">{{ eventBilingual(event).primary }}</strong>
                <small v-if="eventBilingual(event).secondary">（{{ eventBilingual(event).secondary }}）</small>
              </span>
              <span><i :class="event.statusAbbrev.toLowerCase()" />{{ event.statusNameZh }}</span>
              <span>{{ event.locationNameZh || event.padNameZh }}</span>
              <span class="launch-time">{{ eventDate(event.net).time }}<small>UTC+8</small></span>
            </button>
            <div v-if="!upcomingEvents.length" class="catalog-empty">未来 30 天内暂无已载入事件。</div>
          </div>
          <div v-if="upcomingEvents.length" class="pagination-space">
            <span />
            <div><button v-if="upcomingEvents.length > LAUNCH_PREVIEW_ROWS" @click="launchExpanded = !launchExpanded">{{ launchExpanded ? '收起' : `展开全部 ${upcomingEvents.length} 条` }}</button></div>
          </div>
        </div>
      </section>

      <footer class="site-footer">
        <div class="page-frame footer-inner">
          <div><strong>AURORA / EARTH</strong></div>
          <div class="source-list"><span v-for="source in earthSources" :key="source.sourceCode"><i :class="{ healthy: source.success }" />{{ source.sourceName }} · {{ formatUTCDateTime(source.lastFinishedAt) }}</span></div>
        </div>
      </footer>
      </template>
    </div>
  </main>
</template>
