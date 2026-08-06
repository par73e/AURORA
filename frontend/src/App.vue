<!--
THESIS: ORBIT is a scrollable observatory, not a fixed cockpit.
OWN-WORLD: deep spatial navy, orbital blue, launch amber, one shared 12-column frame.
STORY: explore Earth first, then search objects, understand launch sites, and inspect the full launch schedule.
FIRST VIEWPORT: a quiet heading above one dominant globe; controls are compact and details appear only after selection.
FORM: progressive observatory, the assigned seventh Operate structure; dense datasets receive dedicated workspaces below the scene.
-->
<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { CATALOG_PAGE_SIZE } from './catalog'
import { bilingualName, isChineseOrigin } from './bilingual'
import AuroraCover from './components/AuroraCover.vue'
import OrbitScene from './components/OrbitScene.vue'
import SolarSystem from './components/SolarSystem.vue'
import SolarSystemItem from './components/SolarSystemItem.vue'
import MoonScene from './components/MoonScene.vue'
import MarsScene from './components/MarsScene.vue'
import { fetchOrbitOverview } from './api'
import { spacecraftPoint } from './orbit/coordinates'
import { marsHdReady, moonHdReady, orbitTexturesReady, preloadMarsHdTexture, preloadMoonHdTexture, preloadOrbitTextures, preloadSolarTextures } from './preload'
import { solarTexturesReady } from './solar/textures'
import type { LaunchEvent, LaunchSite, OrbitOverview, SceneLayers, Selection } from './types'
import { primaryOperator } from './operators'

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
type AppSurface = 'cover' | 'solar-system' | 'orbit' | 'moon' | 'mars'

// 初始页面：纯 hash 决定（无 hash = 首页；#earth/#moon/#solar-system = 对应页）。
// 不用 sessionStorage 恢复——打开网站应总是首页（上次会话的页面残留会导致"打开就是 #solar-system"）
const surface = ref<AppSurface>(surfaceFromHash())
const solarSystemRef = ref<InstanceType<typeof SolarSystem> | null>(null)
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

/** 页头可收起逻辑当前是否生效（地球主视图 / 月球页 / 火星页） */
function collapsibleHeaderActive() {
  if (surface.value === 'orbit') return orbitPageActive.value
  if (surface.value === 'moon') return moonPageActive.value
  if (surface.value === 'mars') return marsPageActive.value
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
/** 等待组件就绪后再渐亮的回调（onEarthSelect/onMoonSelect/onMarsSelect 注册，组件信号或超时触发） */
let pendingOrbitReveal: (() => void) | null = null
let pendingMoonReveal: (() => void) | null = null
let pendingMarsReveal: (() => void) | null = null
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
  // 离开标志复位：过渡中止时页面不切换，若 leaving 仍为 true 会触发场景元素永久隐藏
  orbitSectionLeaving.value = false
  moonLeaving.value = false
  marsLeaving.value = false
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
  if (['#mars', '#mars-scene', '#mars-objects', '#mars-sites'].includes(window.location.hash)) return 'mars'
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
const dataHealthy = computed(() => overview.value?.freshness.every((item) => item.success) ?? false)
const operators = computed(() => [...new Set((overview.value?.spacecraft ?? []).map((item) => primaryOperator(item.operatorName)))].sort())

/** 目录条目：TLE 航天器（韦布/斯皮策等非地球轨道任务不再在地球页目录/外圈展示，回归太阳系页真实呈现） */
const catalogItems = computed(() => [
  ...(overview.value?.spacecraft ?? []).map((s) => ({ kind: 'spacecraft' as const, ...s })),
])

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

/** 航天器目录分页（每页 12 条；查询/筛选/排序变化时回到第 1 页） */
const catalogPage = ref(1)
const catalogPageCount = computed(() => Math.max(1, Math.ceil(catalogResult.value.items.length / CATALOG_PAGE_SIZE)))
const pagedCatalogItems = computed(() => {
  const start = (catalogPage.value - 1) * CATALOG_PAGE_SIZE
  return catalogResult.value.items.slice(start, start + CATALOG_PAGE_SIZE)
})
watch([objectQuery, operatorFilter, objectSort], () => {
  catalogPage.value = 1
})
function catalogGotoPage(delta: number) {
  catalogPage.value = Math.min(catalogPageCount.value, Math.max(1, catalogPage.value + delta))
}

/** 双语名称（统一规则）：运营方为中国 → 中文主；外国 → 英文主（English（中文）） */
function catalogBilingual(item: { nameZh: string; nameEn: string; operatorName?: string }) {
  return bilingualName(item.nameZh, item.nameEn, item.operatorName)
}

/** 发射事件双语：中文任务名（神舟/天舟/天问…）→ 中文主；其余 → 英文主 */
function eventBilingual(e: { missionName?: string; missionNameZh?: string; name?: string }) {
  const en = (e.missionName || e.name || '').trim()
  const zh = (e.missionNameZh || '').trim()
  const chinese = isChineseOrigin(zh || en)
  if (zh && zh === en) return { primary: en, secondary: '' }
  if (chinese) return { primary: zh || en, secondary: zh ? en : '' }
  return { primary: en, secondary: zh }
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
  if (observerViewActive.value) return {
    latitude: observerLocation.value.latitude,
    longitude: observerLocation.value.longitude,
    distance: 7.6,
    key: `observer:${observerFocusRevision.value}`,
  }
  return null
})

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
    : nextSurface === 'solar-system' ? 'AURORA · 太阳系'
    : nextSurface === 'moon' ? 'AURORA · 月球'
    : nextSurface === 'mars' ? 'AURORA · 火星' : 'AURORA · ORBIT'
  if (nextSurface === 'orbit') {
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
  } else {
    headerExpanded.value = true
    suppressHeaderReveal = false // 切到太阳系/封面：页头恢复正常唤回
  }
  await nextTick()
  window.scrollTo({ top: 0, behavior: 'instant' })
  updateActivePage()
  if (nextSurface === 'orbit' || nextSurface === 'moon' || nextSurface === 'mars') scheduleHeaderCollapse()
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
  solarEnterFromOrbit.value = false
  solarEnterFromMoon.value = false // 封面进入：两个来源标志都清空
  solarEnterFromMars.value = false // 封面进入：火星来源标志同样清空
  window.history.pushState(null, '', '#solar-system')
  preloadOrbitTextures() // 提前预热地球纹理，为下一步进入 ORBIT 做准备
  // 封面进入太阳系：星野页面（星空插图）渐入 → 停留（对应原黑屏时间）→ 渐亮揭示推镜
  cancelPendingTransition()
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
      veilDuration.value = reduced ? '0.01s' : '0.4s' // 渐亮时长：短促地从黑变亮（不拖灰），
      // 全亮时刻 ≈ 推镜路程 70%（剩 1/3 距离）；其后推镜最后 1/3 全是清晰画面
      veilActive.value = false
    })
    transitionTimer = window.setTimeout(() => {
      shellTransitioning.value = false
      transitionTimer = undefined
    }, 400 + 60)
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
  else if (surface.value === 'mars') enterSolarSystemFromMars(skipPush)
}

/** ORBIT → 太阳系（skipPush = 浏览器返回路径，hash 已是目标不重复入栈） */
function enterSolarSystemFromOrbit(skipPush = false) {
  if (!skipPush) window.history.pushState(null, '', '#solar-system')
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
  if (surface.value === 'orbit' || surface.value === 'moon' || surface.value === 'mars') {
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
  // 阶段 1：滚回主视图 + 信息/栏目淡出（页头随之上滑），只留裸星球
  window.scrollTo({ top: 0, behavior: reduced ? 'auto' : 'smooth' })
  if (fromMoon) moonLeaving.value = true
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
      else orbitSectionLeaving.value = false
      void setSurface('cover')
      requestAnimationFrame(() => {
        veilDuration.value = reduced ? '0.01s' : '0.3s'
        veilActive.value = false
      })
      transitionTimer = undefined
    })
  }, reduced ? 20 : (fromMoon ? 450 : 420))
}

// ---- 太阳系 → 地球：镜头在太阳系内放大地球 → 变暗 → 切页 ----

/** 点击地球瞬间：URL 切到 #earth，开始预热 ORBIT 资源 */
function onEarthFlyStart() {
  window.history.pushState(null, '', '#earth')
  preloadOrbitTextures()
  cancelPendingTransition()
}

/** 地球放大到一定程度：遮罩快速变暗（尽量缩短黑屏时间） */
function onEarthFlyZoom() {
  if (surface.value === 'orbit') return // 已切页（幂等保护）；不再依赖 hash——该环境 pushState 不生效
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
  if (surface.value === 'orbit') return // 已切页（幂等保护）；不再依赖 hash——该环境 pushState 不生效
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
  if (surface.value === 'moon') return // 已切页（幂等保护）；不再依赖 hash——该环境 pushState 不生效
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
  if (surface.value === 'moon') return // 已切页（幂等保护）；不再依赖 hash——该环境 pushState 不生效
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

/** 点击火星瞬间：URL 切到 #mars，预热 8k 火星纹理 */
function onMarsFlyStart() {
  window.history.pushState(null, '', '#mars')
  cancelPendingTransition()
  preloadMarsHdTexture() // 预热 8k 火星贴图（本地资源，提前解码避免切换后卡顿）
  marsEnterFromSolar.value = true // 入场路径：火星页分阶段揭示（每次进入都从纯火星开始）
  marsLeaving.value = false // 重置返回清空状态（否则第二次进入残留 true，清空流程失效）
}

/** 火星放大到一定程度：遮罩快速变暗 */
function onMarsFlyZoom() {
  if (surface.value === 'mars') return // 已切页（幂等保护）
  veilDuration.value = '0.22s'
  veilActive.value = true
  window.setTimeout(() => {
    marsHdReady().then(() => onMarsSelect())
    window.setTimeout(() => onMarsSelect(), 3000)
  }, 800)
}

/** 火星放大完成（遮罩已黑）：换页，等首帧贴图 GPU 上传完成再渐亮 */
function onMarsSelect() {
  if (surface.value === 'mars') return // 已切页（幂等保护）
  veilActive.value = true
  solarEnterFromMars.value = false
  void setSurface('mars')
  let revealDone = false
  const reveal = () => {
    if (revealDone) return // 防止兜底超时与信号重复触发
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
    window.setTimeout(reveal, 3000) // 兜底
  }
}

/** MarsScene 首帧贴图上传完成 */
function onMarsSceneReady() {
  marsSceneReadyFlag.value = true
  if (pendingMarsReveal) pendingMarsReveal()
}

/** 火星 → 太阳系（skipPush = 浏览器返回路径） */
function enterSolarSystemFromMars(skipPush = false) {
  if (!skipPush) window.history.pushState(null, '', '#solar-system')
  preloadSolarTextures()
  cancelPendingTransition()
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  solarEnterFromMars.value = true
  solarEnterFromOrbit.value = false // 清空地球来源遗留
  solarEnterFromMoon.value = false // 清空月球来源遗留
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

/** 月球 → 太阳系：渐暗 → 切页（太阳系从月球近景拉回）→ 渐亮 */
/** 月球 → 太阳系（skipPush = 浏览器返回路径） */
function enterSolarSystemFromMoon(skipPush = false) {
  if (!skipPush) window.history.pushState(null, '', '#solar-system')
  preloadSolarTextures()
  cancelPendingTransition()
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  solarEnterFromMoon.value = true
  solarEnterFromOrbit.value = false // 清空地球来源遗留
  solarEnterFromMars.value = false // 清空火星来源遗留
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
  if (surface.value === 'mars') {
    // 主视图 = #mars-objects（第一个板块）顶部仍在视口下半区；滑到第一个板块即展开页头
    const marsObjects = document.getElementById('mars-objects')
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
  } else if (target === 'cover') {
    returnToCover(true)
  } else if (target === 'orbit') {
    cancelPendingTransition()
    void setSurface('orbit')
  } else if (target === 'moon') {
    cancelPendingTransition()
    void setSurface('moon')
  } else if (target === 'mars') {
    cancelPendingTransition()
    void setSurface('mars')
  }
}

/** ESC 键：返回上一级（太阳系）——与点击"太阳系"按钮完全一致 */
function onGlobalKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape') return
  const target = event.target as HTMLElement | null
  if (target && ['INPUT', 'SELECT', 'TEXTAREA'].includes(target.tagName)) return
  if (surface.value === 'orbit' || surface.value === 'moon' || surface.value === 'mars') {
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
    />

    <div v-show="surface !== 'cover' || coverLingering" class="desktop-app" :class="{ 'header-collapsed': !headerExpanded, moon: surface === 'moon', mars: surface === 'mars' }">
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
            <!-- 不 prevent：让浏览器原生执行 href="#home" fragment 导航（该环境禁止 JS 导航 API，
                 但原生同文档 hash 跳转不受限——导航栏链接一直可用即证明）；returnToCover 负责过渡动画 -->
            <a class="brand" href="#home" aria-label="返回 AURORA 封面" @click="() => returnToCover()">
              <span class="brand-mark"><i /><i /><i /></span>
              <span><strong>AURORA</strong><small>ORBITAL OBSERVATORY</small></span>
            </a>
            <SolarSystemItem v-if="surface === 'orbit' || surface === 'moon' || surface === 'mars'" title="太阳系" :icon-size="30" :animated="true" @click="enterSolarSystem" />
          </div>
          <nav v-if="surface === 'orbit'" aria-label="页面导航">
            <a href="#earth"><i class="nav-num">Ⅰ</i>地球</a>
            <a href="#objects"><i class="nav-num">Ⅱ</i>航天器</a>
            <a href="#sites"><i class="nav-num">Ⅲ</i>发射场</a>
            <a href="#launches"><i class="nav-num">Ⅳ</i>发射日程</a>
          </nav>
          <nav v-else-if="surface === 'moon'" aria-label="页面导航">
            <a href="#moon-scene"><i class="nav-num">Ⅰ</i>月球观测</a>
            <a href="#moon-objects"><i class="nav-num">Ⅱ</i>航天器</a>
            <a href="#moon-sites"><i class="nav-num">Ⅲ</i>着陆点</a>
          </nav>
          <nav v-else-if="surface === 'mars'" aria-label="页面导航">
            <a href="#mars-scene"><i class="nav-num">Ⅰ</i>火星观测</a>
            <a href="#mars-objects"><i class="nav-num">Ⅱ</i>航天器</a>
            <a href="#mars-sites"><i class="nav-num">Ⅲ</i>着陆点</a>
          </nav>
          <nav v-else-if="surface === 'solar-system'" aria-label="当前位置">
            <SolarSystemItem title="太阳系" :icon-size="30" :active="true" :animated="true" @click="solarSystemRef?.resetView?.()" />
          </nav>
          <!-- 月球页无中心导航，返回入口在页头左侧（与地球页一致） -->
          <!-- 数据健康灯（只保留一个）：移到太阳系页——深空探测器数据源（CelesTrak/Launch Library/JPL Horizons）
               最近一次同步成败的 3 合 1 聚合；地球/月球页保持身份标签 -->
          <div v-if="surface === 'solar-system'" class="live-status">
            <span class="status-dot" :class="{ healthy: !!overview && dataHealthy, syncing: loading }" />
            <span>{{ loading ? '同步中' : dataHealthy && !!overview ? '数据正常' : '检查数据' }}</span>
            <strong>{{ timeOnly(now) }} UTC+8</strong>
          </div>
          <div v-else class="live-status solar-clock">
            <span>{{ surface === 'moon' ? '月球 · MOON' : surface === 'mars' ? '火星 · MARS' : '地球 · ORBIT' }}</span>
          </div>
        </div>
      </header>

      <MoonScene v-if="surface === 'moon'" :reveal-tick="moonRevealTick" :enter-from-solar="moonEnterFromSolar" :leaving="moonLeaving" :header-expanded="headerExpanded" @blank-click="collapseHeaderFromScene" @textures-ready="onMoonSceneReady" />

      <MarsScene v-if="surface === 'mars'" :reveal-tick="marsRevealTick" :enter-from-solar="marsEnterFromSolar" :leaving="marsLeaving" :header-expanded="headerExpanded" @blank-click="collapseHeaderFromScene" @textures-ready="onMarsSceneReady" />

      <SolarSystem
        ref="solarSystemRef"
        v-if="surface === 'solar-system'"
        :enter-from-orbit="solarEnterFromOrbit"
        :enter-from-moon="solarEnterFromMoon"
        :enter-from-mars="solarEnterFromMars"
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
              :observer-target="observerLocation"
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
              <label><input v-model="layers.spacecraft" type="checkbox"><i />航天器</label>
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
              <button v-for="craft in pagedCatalogItems" :key="craft.id" class="object-row" role="row" @click="selectAndFocus({ kind: craft.kind, id: craft.id })">
                <span>{{ craft.noradCatalogId ?? '—' }}</span>
                <span><strong>{{ catalogBilingual(craft).primary }}</strong><small v-if="catalogBilingual(craft).secondary">（{{ catalogBilingual(craft).secondary }}）</small></span>
                <span>{{ craft.operatorName }}</span>
                <span>{{ craft.category }}</span>
                <span>{{ formatUTCDate(craft.orbitEpoch) }}</span>
              </button>
              <div v-if="!catalogResult.items.length" class="catalog-empty">没有符合当前条件的航天器。请修改搜索词或筛选条件。</div>
            </div>
            <div class="pagination-space"><span>第 {{ catalogPage }} / {{ catalogPageCount }} 页 · 共 {{ catalogResult.items.length }} 个对象</span><div><button :disabled="catalogPage <= 1" @click="catalogGotoPage(-1)">上一页</button><button :disabled="catalogPage >= catalogPageCount" @click="catalogGotoPage(1)">下一页</button></div></div>
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
              <span><strong>{{ bilingualName(site.nameZh, site.nameEn).primary }}</strong><small v-if="bilingualName(site.nameZh, site.nameEn).secondary">（{{ bilingualName(site.nameZh, site.nameEn).secondary }}）</small></span>
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
                <strong :lang="isChineseOrigin(eventBilingual(event).primary) ? 'zh-CN' : 'en'">{{ eventBilingual(event).primary }}</strong>
                <small v-if="eventBilingual(event).secondary">（{{ eventBilingual(event).secondary }}）</small>
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
          <div class="source-list"><span v-for="source in overview?.freshness" :key="source.sourceCode"><i :class="{ healthy: source.success }" />{{ source.sourceName }} · {{ formatUTCDateTime(source.lastFinishedAt) }}</span></div>
        </div>
      </footer>
      </template>
    </div>
  </main>
</template>
