<!--
THESIS: ORBIT is a scrollable observatory, not a fixed cockpit.
OWN-WORLD: deep spatial navy, orbital blue, launch amber, one shared 12-column frame.
STORY: explore Earth first, then search objects, understand launch sites, and inspect the full launch schedule.
FIRST VIEWPORT: a quiet heading above one dominant globe; controls are compact and details appear only after selection.
FORM: progressive observatory, the assigned seventh Operate structure; dense datasets receive dedicated workspaces below the scene.
-->
<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import AuroraCover from './components/AuroraCover.vue'
import OrbitScene from './components/OrbitScene.vue'
import SolarSystem from './components/SolarSystem.vue'
import SolarSystemItem from './components/SolarSystemItem.vue'
import MoonScene from './components/MoonScene.vue'
import { fetchOrbitOverview } from './api'
import { spacecraftPoint } from './orbit/coordinates'
import { moonHdReady, orbitTexturesReady, preloadMoonHdTexture, preloadOrbitTextures, preloadSolarTextures } from './preload'
import { solarTexturesReady } from './solar/textures'
import type { LaunchEvent, LaunchSite, OrbitOverview, SceneLayers, Selection } from './types'

type ObserverLocationStatus = 'locating' | 'located' | 'fallback'

interface ObserverLocation {
  latitude: number
  longitude: number
  label: string
  status: ObserverLocationStatus
}

const fallbackObserver = observerFallback()
const overview = ref<OrbitOverview | null>(null)
const loading = ref(true)
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
const dayNightEnabled = ref(false)
type AppSurface = 'cover' | 'solar-system' | 'orbit' | 'moon'

// 初始页面：hash 明确指向某页（如分享链接 #orbit）时优先 hash；
// 否则用 sessionStorage 记录（刷新回到上次所在页——hash 设置不可靠环境下的可靠恢复）
const initialSurface = ((): AppSurface => {
  const fromHash = surfaceFromHash()
  if (fromHash !== 'cover') return fromHash
  const stored = sessionStorage.getItem('aurora:surface') as AppSurface | null
  return stored ?? 'cover'
})()
const surface = ref<AppSurface>(initialSurface)
const solarSystemRef = ref<InstanceType<typeof SolarSystem> | null>(null)
/** 地球界面"进入边界"信号：遮罩开始淡出时递增，OrbitScene 据此播放入场渐亮 */
const orbitRevealTick = ref(0)
const headerExpanded = ref(true)
const orbitPageActive = ref(true)
const moonPageActive = ref(true)

/** 页头可收起逻辑当前是否生效（地球主视图 / 月球页） */
function collapsibleHeaderActive() {
  if (surface.value === 'orbit') return orbitPageActive.value
  if (surface.value === 'moon') return moonPageActive.value
  return false
}
const orbitSectionLeaving = ref(false)
/** 是否从 ORBIT 返回太阳系（太阳系场景挂载后从地球近景拉回默认构图） */
const solarEnterFromOrbit = ref(false)
const solarEnterFromMoon = ref(false)
/** 月球页面"进入边界"信号：遮罩开始淡出时递增，MoonScene 据此渐亮 */
const moonRevealTick = ref(0)
/** 从太阳系进入月球：true 时月球页从"纯月球"开始分阶段揭示 */
const moonEnterFromSolar = ref(false)
/** 返回太阳系：true 时月球页清空月球以外元素（只留球体） */
const moonLeaving = ref(false)
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
/** 封面→太阳系：换页提前到点击瞬间，封面继续覆盖（lingering），黑幕结束才撤下 */
const coverLingering = ref(false)
/** 太阳系入场推镜延迟：封面路径 = 变暗时长（全黑开始时起飞）；直接加载 = 0 */
const solarFlyDelay = ref(0)
/** OrbitScene/MoonScene 首帧贴图上传完成信号（textures-ready） */
const orbitSceneReadyFlag = ref(false)
const moonSceneReadyFlag = ref(false)
/** 等待组件就绪后再渐亮的回调（onEarthSelect/onMoonSelect 注册，组件信号或超时触发） */
let pendingOrbitReveal: (() => void) | null = null
let pendingMoonReveal: (() => void) | null = null
/** 封面路径进入时播放入场推镜；刷新/直接加载不播（静态恢复现场） */
const solarEntryFly = ref(false)
const shellZoom = ref(1)
const shellOrigin = ref('50% 50%')
const shellTransitioning = ref(false)
const veilDuration = ref('0.4s')
let transitionTimer: number | undefined
let transitionFrame: number | undefined

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

/** 取消进行中的过渡（含定时器与动画帧），恢复无过渡状态 */
function cancelPendingTransition() {
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
  if (window.location.hash === '#solar-system') return 'solar-system'
  if (['#moon', '#moon-scene', '#moon-objects', '#moon-sites'].includes(window.location.hash)) return 'moon'
  if (['#orbit', '#objects', '#sites', '#launches'].includes(window.location.hash)) return 'orbit'
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
const dataHealthy = computed(() => overview.value?.freshness.every((item) => item.success) ?? false)
const operators = computed(() => [...new Set(overview.value?.spacecraft.map((item) => item.operatorName) ?? [])].sort())

const catalogResult = computed(() => {
  const raw = objectQuery.value.trim()
  let matcher: ((value: string) => boolean) | undefined
  let queryError = ''

  if (raw) {
    if (raw.startsWith('/')) {
      const lastSlash = raw.lastIndexOf('/')
      if (lastSlash <= 0) {
        queryError = '正则表达式需要以 / 结束，例如 /ISS|TIANHE/i'
      } else {
        try {
          const expression = new RegExp(raw.slice(1, lastSlash), raw.slice(lastSlash + 1))
          matcher = (value) => expression.test(value)
        } catch (reason) {
          queryError = reason instanceof Error ? `正则表达式无效：${reason.message}` : '正则表达式无效'
        }
      }
    } else {
      const keyword = raw.toLocaleLowerCase()
      matcher = (value) => value.toLocaleLowerCase().includes(keyword)
    }
  }

  const items = (overview.value?.spacecraft ?? []).filter((item) => {
    if (operatorFilter.value !== 'all' && item.operatorName !== operatorFilter.value) return false
    if (!matcher) return !raw && !queryError
    return matcher([item.nameZh, item.nameEn, item.noradCatalogId, item.operatorName, item.category].join(' '))
  })

  items.sort((left, right) => {
    if (objectSort.value === 'norad') return left.noradCatalogId - right.noradCatalogId
    if (objectSort.value === 'operator') return left.operatorName.localeCompare(right.operatorName, 'zh-CN')
    return left.nameZh.localeCompare(right.nameZh, 'zh-CN')
  })
  return { items, error: queryError }
})



const focusTarget = computed(() => {
  if (selectedEvent.value?.latitude != null && selectedEvent.value.longitude != null) {
    return { latitude: selectedEvent.value.latitude, longitude: selectedEvent.value.longitude, distance: 5.8, key: `event:${selectedEvent.value.externalId}` }
  }
  if (selectedSite.value) {
    return { latitude: selectedSite.value.latitude, longitude: selectedSite.value.longitude, distance: 6.3, key: `site:${selectedSite.value.id}` }
  }
  if (selectedSpacecraft.value) {
    const point = spacecraftPoint(selectedSpacecraft.value, now.value)
    if (point) return { latitude: point.latitude, longitude: point.longitude, distance: 6.7, key: `spacecraft:${selectedSpacecraft.value.id}` }
  }
  if (observerViewActive.value) return {
    latitude: observerLocation.value.latitude,
    longitude: observerLocation.value.longitude,
    distance: 7.6,
    key: `observer:${observerFocusRevision.value}`,
  }
  return null
})

function observerFallback(): Omit<ObserverLocation, 'status'> {
  const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone
  if (timezone === 'Asia/Shanghai' || timezone === 'Asia/Chongqing') {
    return { latitude: 31.2304, longitude: 121.4737, label: '上海（时区回退）' }
  }
  return { latitude: 0, longitude: 0, label: '本初子午线（定位回退）' }
}

function requestObserverLocation() {
  if (!navigator.geolocation) {
    observerLocation.value = { ...fallbackObserver, status: 'fallback' }
    observerFocusRevision.value += 1
    return
  }

  observerLocation.value = { ...fallbackObserver, label: '正在获取位置', status: 'locating' }
  navigator.geolocation.getCurrentPosition(
    ({ coords }) => {
      observerLocation.value = {
        latitude: coords.latitude,
        longitude: coords.longitude,
        label: '当前位置',
        status: 'located',
      }
      observerFocusRevision.value += 1
    },
    () => {
      observerLocation.value = { ...fallbackObserver, status: 'fallback' }
      observerFocusRevision.value += 1
    },
    { enableHighAccuracy: false, timeout: 6000, maximumAge: 900_000 },
  )
}

function focusObserver() {
  selection.value = null
  observerViewActive.value = true
  observerFocusRevision.value += 1
  if (observerLocation.value.status !== 'located') requestObserverLocation()
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
  window.requestAnimationFrame(() => document.querySelector('#orbit')?.scrollIntoView({ behavior: 'smooth', block: 'start' }))
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
  // 记录当前页面到 sessionStorage：hash 在该浏览器环境设置不可靠（刷新会进错页），
  // 刷新恢复优先读这里（回到"刚才所在的地方"）
  try {
    sessionStorage.setItem('aurora:surface', nextSurface)
  } catch {
    /* sessionStorage 不可用时静默降级 */
  }
  surface.value = nextSurface
  document.title = nextSurface === 'cover'
    ? 'AURORA'
    : nextSurface === 'solar-system' ? 'AURORA · 太阳系'
    : nextSurface === 'moon' ? 'AURORA · 月球' : 'AURORA · ORBIT'
  if (nextSurface === 'orbit') {
    orbitPageActive.value = true
    headerExpanded.value = false // 进入 ORBIT 默认收起页头（悬停屏幕顶部可展开）
  } else if (nextSurface === 'moon') {
    moonPageActive.value = true
    headerExpanded.value = false // 月球页同样默认收起页头
  } else {
    headerExpanded.value = true
  }
  await nextTick()
  window.scrollTo({ top: 0, behavior: 'instant' })
  updateActivePage()
  if (nextSurface === 'orbit' || nextSurface === 'moon') scheduleHeaderCollapse()
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
  solarEnterFromOrbit.value = false
  window.history.pushState(null, '', '#solar-system')
  preloadOrbitTextures() // 提前预热地球纹理，为下一步进入 ORBIT 做准备
  // 封面进入太阳系：星野页面（星空插图）渐入 → 停留（对应原黑屏时间）→ 渐亮揭示推镜
  cancelPendingTransition()
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  const exitMs = reduced ? 40 : 200 // 星野渐入 200ms
  const dwellMs = reduced ? 0 : 400 // 星野停留 400ms（对应之前的黑屏时间）
  solarEntryFly.value = true // 封面路径：播放入场推镜（启动由 SolarSystem 侧等纹理就绪）
  shellOrigin.value = '50% 42%'
  shellZoom.value = 1.05
  shellTransitioning.value = true
  veilDuration.value = reduced ? '0.01s' : '0.2s' // 星野渐入
  veilActive.value = true
  coverLingering.value = true
  void setSurface('solar-system')
  // 停留结束 = 纹理解码就绪（或 2.5s 超时兜底）——黑幕时长与加载同步，
  // 渐亮揭示时推镜正在中途（SolarSystem 侧同源就绪信号启动推镜）
  let revealed = false
  const reveal = () => {
    if (revealed) return
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
      veilDuration.value = reduced ? '0.01s' : '0.6s' // 渐亮时长
      veilActive.value = false
    })
    transitionTimer = window.setTimeout(() => {
      shellTransitioning.value = false
      transitionTimer = undefined
    }, 620 + 60)
  }
  transitionTimer = window.setTimeout(() => {
    solarTexturesReady().then(reveal)
    window.setTimeout(reveal, 2500) // 兜底：加载异常时最迟 2.5s 揭示
  }, exitMs + dwellMs)
}


/** ORBIT → 太阳系：滚回主地球视图 → 信息淡出只留地球 → 变暗 → 切页，
 *  太阳系场景从地球近景开始拉回（地球缩回轨道位置，遮罩淡出时可见） */
function returnToSolarSystem(skipPush = false) {
  if (surface.value === 'orbit') enterSolarSystemFromOrbit(skipPush)
  else if (surface.value === 'moon') enterSolarSystemFromMoon(skipPush)
}

/** ORBIT → 太阳系（skipPush = 浏览器返回路径，hash 已是目标不重复入栈） */
function enterSolarSystemFromOrbit(skipPush = false) {
  if (!skipPush) window.history.pushState(null, '', '#solar-system')
  preloadSolarTextures()
  cancelPendingTransition()
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  // 阶段 1：滚回主地球视图，页面信息（标签/工具条/内容区）淡出，只留地球
  window.scrollTo({ top: 0, behavior: reduced ? 'auto' : 'smooth' })
  orbitSectionLeaving.value = true
  solarEnterFromOrbit.value = true
  // 阶段 2：变暗，盖住地球界面
  transitionTimer = window.setTimeout(() => {
    if (surfaceFromHash() !== 'solar-system') {
      cancelPendingTransition()
      return
    }
    veilDuration.value = reduced ? '0.01s' : '0.4s' // 渐暗 400ms
    veilActive.value = true
    // 阶段 3：切页——太阳系挂载并从地球近景拉回；遮罩随即淡出，拉回过程可见
    transitionTimer = window.setTimeout(() => {
      if (surfaceFromHash() !== 'solar-system') {
        cancelPendingTransition()
        return
      }
      orbitSectionLeaving.value = false
      void setSurface('solar-system')
      requestAnimationFrame(() => {
        veilDuration.value = reduced ? '0.01s' : '0.5s' // 渐亮 500ms
        veilActive.value = false
      })
      transitionTimer = undefined
    }, reduced ? 30 : 460) // 等遮罩全黑（400ms）再切页
  }, reduced ? 20 : 420)
}

function enterOrbit() {
  window.history.pushState(null, '', '#orbit')
  // 朝向地球方向推近（地球大致位于画面 55%/38% 处），形成“放大进入地球”的感觉
  transitionTo('orbit', 1.12, '55% 38%')
}

function returnToCover(skipPush = false) {
  if (!skipPush) {
    // 该浏览器环境对 hash 的所有 API 设置（pushState/location.hash/replaceState）均不生效——
    // 改用 location.href 强制导航（整页重载，地址栏必定更新为 #home）。
    // 重载前预置 sessionStorage = 'cover'，新页面加载时恢复首页（否则会恢复旧页面）
    try {
      sessionStorage.setItem('aurora:surface', 'cover')
    } catch {
      /* 静默降级 */
    }
    window.location.href = '#home'
    return // 页面即将重载，不再执行过渡动画
  }
  transitionTo('cover', 0.96)
}

// ---- 太阳系 → 地球：镜头在太阳系内放大地球 → 变暗 → 切页 ----

/** 点击地球瞬间：URL 切到 #orbit，开始预热 ORBIT 资源 */
function onEarthFlyStart() {
  window.history.pushState(null, '', '#orbit')
  preloadOrbitTextures()
  cancelPendingTransition()
}

/** 地球放大到一定程度：遮罩快速变暗（尽量缩短黑屏时间） */
function onEarthFlyZoom() {
  if (surface.value === 'orbit' || surfaceFromHash() !== 'orbit') return
  veilDuration.value = '0.22s'
  veilActive.value = true
  // 遮罩完全变黑后（+800ms，黑屏留足余量）才开始 8k 解码——主线程解码/GPU 上传都发生在黑屏中；
  // 解码完成才换页（3s 超时兜底）
  window.setTimeout(() => {
    orbitTexturesReady().then(() => onEarthSelect())
    window.setTimeout(() => onEarthSelect(), 3000)
  }, 800)
}

/** 地球放大完成（遮罩已黑）：换页，等首帧贴图 GPU 上传完成再渐亮 */
function onEarthSelect() {
  if (surface.value === 'orbit' || surfaceFromHash() !== 'orbit') return
  veilActive.value = true
  void setSurface('orbit')
  let revealDone = false
  const reveal = () => {
    if (revealDone) return // 防止兜底超时与信号重复触发
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
    window.setTimeout(reveal, 3000) // 兜底：上传异常时最迟 3s 揭示
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
  preloadMoonHdTexture() // 预热 8k 月球贴图（本地资源，提前解码避免切换后卡顿）
  moonEnterFromSolar.value = true // 入场路径：月球页分阶段揭示（每次进入都从纯月球开始）
  moonLeaving.value = false // 重置返回清空状态（否则第二次进入残留 true，清空流程失效）
}

/** 月球放大到一定程度：遮罩快速变暗 */
function onMoonFlyZoom() {
  if (surface.value === 'moon' || surfaceFromHash() !== 'moon') return
  veilDuration.value = '0.22s'
  veilActive.value = true
  // 遮罩完全变黑后（+800ms，黑屏留足余量）才开始 16k 解码——主线程解码/GPU 上传都发生在黑屏中
  window.setTimeout(() => {
    moonHdReady().then(() => onMoonSelect())
    window.setTimeout(() => onMoonSelect(), 3000)
  }, 800)
}

/** 月球放大完成（遮罩已黑）：换页，等首帧贴图 GPU 上传完成再渐亮 */
function onMoonSelect() {
  if (surface.value === 'moon' || surfaceFromHash() !== 'moon') return
  veilActive.value = true
  solarEnterFromMoon.value = false
  void setSurface('moon')
  let revealDone = false
  const reveal = () => {
    if (revealDone) return // 防止兜底超时与信号重复触发
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
    window.setTimeout(reveal, 3000) // 兜底
  }
}

/** MoonScene 首帧贴图上传完成 */
function onMoonSceneReady() {
  moonSceneReadyFlag.value = true
  if (pendingMoonReveal) pendingMoonReveal()
}

/** 月球 → 太阳系：渐暗 → 切页（太阳系从月球近景拉回）→ 渐亮 */
/** 月球 → 太阳系（skipPush = 浏览器返回路径） */
function enterSolarSystemFromMoon(skipPush = false) {
  if (!skipPush) window.history.pushState(null, '', '#solar-system')
  preloadSolarTextures()
  cancelPendingTransition()
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  solarEnterFromMoon.value = true
  // 阶段 1：滚回月球主视图 + 清空月球以外的所有元素（标记/飞行器/标签），只留月球球体
  window.scrollTo({ top: 0, behavior: reduced ? 'auto' : 'smooth' })
  moonLeaving.value = true
  // 阶段 2（清空效果可见后才变暗——与地球返回"信息淡出只留地球"同节奏）：变暗 400ms
  transitionTimer = window.setTimeout(() => {
    veilDuration.value = reduced ? '0.01s' : '0.4s'
    veilActive.value = true
    // 阶段 3：等遮罩全黑再切页
    transitionTimer = window.setTimeout(() => {
      if (surfaceFromHash() !== 'solar-system') {
        cancelPendingTransition()
        return
      }
      void setSurface('solar-system')
      requestAnimationFrame(() => {
        veilDuration.value = reduced ? '0.01s' : '0.5s' // 渐亮 500ms
        veilActive.value = false
      })
      transitionTimer = undefined
    }, reduced ? 30 : 460)
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
  if (surface.value === 'moon') {
    // 主视图 = #moon-objects（第一个板块）顶部仍在视口下半区；滑到第一个板块即展开页头
    const moonObjects = document.getElementById('moon-objects')
    const objectsTop = moonObjects?.getBoundingClientRect().top ?? window.innerHeight
    const nextMoonPageActive = objectsTop > window.innerHeight / 2
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
  } else if (target === 'cover') {
    returnToCover(true)
  } else if (target === 'orbit') {
    cancelPendingTransition()
    void setSurface('orbit')
  } else if (target === 'moon') {
    cancelPendingTransition()
    void setSurface('moon')
  }
}

/** ESC 键：返回上一级（太阳系）——与点击"太阳系"按钮完全一致 */
function onGlobalKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape') return
  const target = event.target as HTMLElement | null
  if (target && ['INPUT', 'SELECT', 'TEXTAREA'].includes(target.tagName)) return
  if (surface.value === 'orbit' || surface.value === 'moon') {
    event.preventDefault()
    returnToSolarSystem()
  }
}

onMounted(() => {
  window.addEventListener('popstate', onPopState)
  window.addEventListener('keydown', onGlobalKeydown)
  preloadSolarTextures() // 预热太阳系纹理，让首次进入不出现加载卡顿
  document.title = surface.value === 'cover'
    ? 'AURORA'
    : surface.value === 'solar-system' ? 'AURORA · 太阳系' : 'AURORA · ORBIT'
  load()
  requestObserverLocation()
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
  window.removeEventListener('pointermove', handleWindowPointerMove)
  window.removeEventListener('pointerdown', registerHeaderActivity)
  window.removeEventListener('wheel', registerHeaderActivity)
  window.removeEventListener('keydown', registerHeaderActivity)
  window.removeEventListener('scroll', handlePageScroll)
  // hashchange 监听已移除（popstate 统一处理）
})
</script>

<template>
  <div class="surface-veil" :class="{ active: veilActive }" :style="{ '--veil-duration': veilDuration }" aria-hidden="true" />
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
    />

    <div v-show="surface !== 'cover' || coverLingering" class="desktop-app" :class="{ 'header-collapsed': !headerExpanded, moon: surface === 'moon' }">
      <header
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
            <a class="brand" href="#home" aria-label="返回 AURORA 封面" @click.prevent="returnToCover">
              <span class="brand-mark"><i /><i /><i /></span>
              <span><strong>AURORA</strong><small>ORBITAL OBSERVATORY</small></span>
            </a>
            <SolarSystemItem v-if="surface === 'orbit' || surface === 'moon'" title="太阳系" :icon-size="30" :animated="true" @click="enterSolarSystem" />
          </div>
          <nav v-if="surface === 'orbit'" aria-label="页面导航">
            <a href="#orbit"><i class="nav-num">Ⅰ</i>地球</a>
            <a href="#objects"><i class="nav-num">Ⅱ</i>航天器</a>
            <a href="#sites"><i class="nav-num">Ⅲ</i>发射场</a>
            <a href="#launches"><i class="nav-num">Ⅳ</i>发射日程</a>
          </nav>
          <nav v-else-if="surface === 'moon'" aria-label="页面导航">
            <a href="#moon-scene"><i class="nav-num">Ⅰ</i>月球观测</a>
            <a href="#moon-objects"><i class="nav-num">Ⅱ</i>航天器</a>
            <a href="#moon-sites"><i class="nav-num">Ⅲ</i>着陆点</a>
          </nav>
          <nav v-else-if="surface === 'solar-system'" aria-label="当前位置">
            <SolarSystemItem title="太阳系" :icon-size="30" :active="true" :animated="true" @click="solarSystemRef?.resetView?.()" />
          </nav>
          <!-- 月球页无中心导航，返回入口在页头左侧（与地球页一致） -->
          <div v-if="surface === 'orbit'" class="live-status">
            <span class="status-dot" :class="{ healthy: dataHealthy }" />
            <span>{{ dataHealthy ? '数据正常' : '检查数据' }}</span>
            <strong>{{ timeOnly(now) }} UTC+8</strong>
          </div>
          <div v-else class="live-status solar-clock">
            <span>{{ surface === 'moon' ? '月球 · MOON' : 'SOLAR SYSTEM' }}</span>
          </div>
        </div>
      </header>

      <MoonScene v-if="surface === 'moon'" :reveal-tick="moonRevealTick" :enter-from-solar="moonEnterFromSolar" :leaving="moonLeaving" @blank-click="collapseHeaderFromScene" @textures-ready="onMoonSceneReady" />

      <SolarSystem
        ref="solarSystemRef"
        v-if="surface === 'solar-system'"
        :enter-from-orbit="solarEnterFromOrbit"
        :enter-from-moon="solarEnterFromMoon"
        :fly-delay="solarFlyDelay"
        :play-entry-fly="solarEntryFly"
        @select-earth="onEarthSelect"
        @earth-fly-start="onEarthFlyStart"
        @earth-fly-zoom="onEarthFlyZoom"
        @moon-fly-start="onMoonFlyStart"
        @moon-fly-zoom="onMoonFlyZoom"
        @select-moon="onMoonSelect"
      />

      <template v-else-if="surface === 'orbit'">
      <section id="orbit" ref="orbitSection" class="orbit-section" :class="{ leaving: orbitSectionLeaving }">
        <div class="page-frame">
          <div ref="orbitSceneFrame" class="scene-frame">
            <OrbitScene
              v-if="overview"
              :spacecraft="overview.spacecraft"
              :sites="overview.launchSites"
              :events="overview.events"
              :layers="layers"
              :selection="selection"
              :focus-target="focusTarget"
              :observer-target="observerLocation"
              :observer-active="observerViewActive"
              :day-night-enabled="dayNightEnabled"
              :reveal-tick="orbitRevealTick"
              @textures-ready="onOrbitSceneReady"
              @select="selectFromScene"
              @clear-selection="selection = null"
              @view-change="leaveObserverView"
              @blank-click="collapseHeaderFromScene"
            />

            <div class="scene-toolbar" aria-label="场景图层">
              <span>图层</span>
              <label><input v-model="layers.spacecraft" type="checkbox"><i />航天器</label>
              <label><input v-model="layers.orbits" type="checkbox"><i />轨道</label>
              <label><input v-model="layers.sites" type="checkbox"><i class="amber" />发射场</label>
              <label><input v-model="dayNightEnabled" type="checkbox"><i class="terminator" />晨昏线</label>
            </div>

            <button
              class="scene-location"
              :class="{ active: observerViewActive }"
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
            <div><p class="section-kicker">OBJECT CATALOG</p><h2><i class="sec-num">Ⅱ</i>航天器</h2></div>
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
            <p v-if="catalogResult.error" class="query-error">{{ catalogResult.error }}</p>
            <div class="catalog-meta"><span>找到 {{ catalogResult.items.length }} 个对象</span><span>支持 JavaScript 正则语法</span></div>
            <div class="object-table" role="table" aria-label="航天器查询结果">
              <div class="object-table-head" role="row"><span>NORAD</span><span>对象</span><span>运营方</span><span>类型</span><span>轨道历元</span></div>
              <button v-for="craft in catalogResult.items" :key="craft.id" class="object-row" role="row" @click="selectAndFocus({ kind: 'spacecraft', id: craft.id })">
                <span>{{ craft.noradCatalogId }}</span>
                <span><strong>{{ craft.nameZh }}</strong><small>{{ craft.nameEn }}</small></span>
                <span>{{ craft.operatorName }}</span>
                <span>{{ craft.category }}</span>
                <span>{{ new Date(craft.orbitEpoch).toLocaleDateString('zh-CN') }}</span>
              </button>
              <div v-if="!catalogResult.items.length" class="catalog-empty">没有符合当前条件的航天器。请修改搜索词或筛选条件。</div>
            </div>
            <div class="pagination-space"><span>第 1 页 · 已载入 {{ overview?.spacecraft.length ?? 0 }} 个对象</span><div><button disabled>上一页</button><button disabled>下一页</button></div></div>
          </div>
        </div>
      </section>

      <section id="sites" class="content-section sites-section">
        <div class="page-frame">
          <div class="section-heading">
            <div><p class="section-kicker launch-kicker">GROUND NETWORK</p><h2><i class="sec-num">Ⅲ</i>发射场</h2></div>
          </div>
          <div class="site-directory">
            <button v-for="site in overview?.launchSites" :key="site.id" @click="selectAndFocus({ kind: 'site', id: site.id })">
              <span class="site-code">{{ site.countryCode }}</span>
              <span><strong>{{ site.nameZh }}</strong><small>{{ site.nameEn }}</small></span>
              <p>{{ site.description }}</p>
              <span class="site-coordinate">{{ formatCoordinate(site.latitude, 'N', 'S') }}<br>{{ formatCoordinate(site.longitude, 'E', 'W') }}</span>
            </button>
          </div>
        </div>
      </section>

      <section id="launches" class="content-section launches-section">
        <div class="page-frame">
          <div class="section-heading">
            <div><p class="section-kicker launch-kicker">NEXT 30 DAYS</p><h2><i class="sec-num">Ⅳ</i>发射日程</h2></div>
          </div>
          <div class="launch-list">
            <div class="launch-list-head"><span>日期</span><span>任务</span><span>状态</span><span>发射地点</span><span>时间</span></div>
            <button v-for="event in upcomingEvents" :key="event.externalId" class="launch-row" @click="selectAndFocus({ kind: 'event', id: event.externalId })">
              <time><strong>{{ eventDate(event.net).day }}</strong><span>{{ eventDate(event.net).month }} · {{ eventDate(event.net).weekday }}</span></time>
              <span class="launch-mission">
                <strong lang="en">{{ event.missionName || event.name }}</strong>
                <small v-if="event.missionNameZh && event.missionNameZh !== event.missionName">{{ event.missionNameZh }}</small>
              </span>
              <span><i :class="event.statusAbbrev.toLowerCase()" />{{ event.statusNameZh }}</span>
              <span>{{ event.locationNameZh || event.padNameZh }}</span>
              <span class="launch-time">{{ eventDate(event.net).time }}<small>UTC+8</small></span>
            </button>
            <div v-if="!upcomingEvents.length" class="catalog-empty">未来 30 天内暂无已载入事件。</div>
          </div>
        </div>
      </section>

      <footer class="site-footer">
        <div class="page-frame footer-inner">
          <div><strong>AURORA / ORBIT</strong></div>
          <div class="source-list"><span v-for="source in overview?.freshness" :key="source.sourceCode"><i :class="{ healthy: source.success }" />{{ source.sourceName }} · {{ new Date(source.lastFinishedAt).toLocaleString('zh-CN', { hour12: false }) }}</span></div>
        </div>
      </footer>
      </template>
    </div>
  </main>
</template>
