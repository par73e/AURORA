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
import { fetchOrbitOverview } from './api'
import { spacecraftPoint } from './orbit/coordinates'
import { preloadOrbitTextures, preloadSolarTextures } from './preload'
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
const showEventOriginal = ref(false)
type AppSurface = 'cover' | 'solar-system' | 'orbit'

const surface = ref<AppSurface>(surfaceFromHash())
const headerExpanded = ref(true)
const orbitPageActive = ref(true)
const orbitSectionLeaving = ref(false)
/** 是否从 ORBIT 返回太阳系（太阳系场景挂载后从地球近景拉回默认构图） */
const solarEnterFromOrbit = ref(false)
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
    // 全黑停留：遮罩保持不透明
    transitionTimer = window.setTimeout(() => {
      // 过渡期间若用户通过后退/前进改动了 hash，放弃本次过渡，避免 URL 与页面失步
      if (surfaceFromHash() !== nextSurface) {
        cancelPendingTransition()
        return
      }
      // 换页：新页面在遮罩后完成首帧渲染与加载
      void setSurface(nextSurface)
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

const selectedEventSite = computed(() => selectedEvent.value ? nearestSite(selectedEvent.value) : undefined)

watch(() => selectedEvent.value?.externalId, () => {
  showEventOriginal.value = false
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

function nearestSite(event: LaunchEvent): LaunchSite | undefined {
  if (event.latitude == null || event.longitude == null) return undefined
  const candidates = overview.value?.launchSites ?? []
  let best: { site: LaunchSite; distance: number } | undefined
  for (const site of candidates) {
    const latDistance = site.latitude - event.latitude
    const lonDistance = (site.longitude - event.longitude) * Math.cos(event.latitude * Math.PI / 180)
    const distance = Math.hypot(latDistance, lonDistance)
    if (!best || distance < best.distance) best = { site, distance }
  }
  return best && best.distance < 3 ? best.site : undefined
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

function eventDate(value: string) {
  const date = new Date(value)
  return {
    day: new Intl.DateTimeFormat('zh-CN', { day: '2-digit', timeZone: DISPLAY_TIME_ZONE }).format(date),
    month: new Intl.DateTimeFormat('en-US', { month: 'short', timeZone: DISPLAY_TIME_ZONE }).format(date).toUpperCase(),
    weekday: new Intl.DateTimeFormat('zh-CN', { weekday: 'short', timeZone: DISPLAY_TIME_ZONE }).format(date),
    time: new Intl.DateTimeFormat('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false, timeZone: DISPLAY_TIME_ZONE }).format(date),
  }
}

function launchVehicleName(event: LaunchEvent) {
  return event.name.split(' | ')[0]?.trim() || event.providerName || '运载火箭待确认'
}

async function setSurface(nextSurface: AppSurface) {
  surface.value = nextSurface
  document.title = nextSurface === 'cover'
    ? 'AURORA'
    : nextSurface === 'solar-system' ? 'AURORA · 太阳系' : 'AURORA · ORBIT'
  if (nextSurface === 'orbit') {
    orbitPageActive.value = true
    headerExpanded.value = false // 进入 ORBIT 默认收起页头（悬停屏幕顶部可展开）
  } else {
    headerExpanded.value = true
  }
  await nextTick()
  window.scrollTo({ top: 0, behavior: 'instant' })
  updateActivePage()
  if (nextSurface === 'orbit') scheduleHeaderCollapse()
  else clearHeaderIdleTimer()
}

function enterSolarSystem() {
  if (surface.value === 'orbit') {
    enterSolarSystemFromOrbit()
    return
  }
  solarEnterFromOrbit.value = false
  window.history.pushState(null, '', '#solar-system')
  preloadOrbitTextures() // 提前预热地球纹理，为下一步进入 ORBIT 做准备
  // 封面进入太阳系：变暗与全黑停留拉长，形成渐入深空的仪式感
  transitionTo('solar-system', 1.05, '50% 42%', { exitMs: 900, dwellMs: 450, veilSeconds: '0.7s' })
}

/** ORBIT → 太阳系：滚回主地球视图 → 信息淡出只留地球 → 变暗 → 切页，
 *  太阳系场景从地球近景开始拉回（地球缩回轨道位置，遮罩淡出时可见） */
function enterSolarSystemFromOrbit() {
  window.history.pushState(null, '', '#solar-system')
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
    veilDuration.value = reduced ? '0.01s' : '0.4s'
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
        veilDuration.value = reduced ? '0.01s' : '0.5s'
        veilActive.value = false
      })
      transitionTimer = undefined
    }, reduced ? 30 : 420)
  }, reduced ? 20 : 420)
}

function enterOrbit() {
  window.history.pushState(null, '', '#orbit')
  // 朝向地球方向推近（地球大致位于画面 55%/38% 处），形成“放大进入地球”的感觉
  transitionTo('orbit', 1.12, '55% 38%')
}

function returnToCover() {
  window.history.pushState(null, '', '#home')
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
}

/** 地球放大完成（遮罩已黑）：换页，页面内容淡入浮现 */
function onEarthSelect() {
  if (surface.value === 'orbit' || surfaceFromHash() !== 'orbit') return
  veilActive.value = true
  void setSurface('orbit')
  requestAnimationFrame(() => {
    veilDuration.value = '0.3s'
    veilActive.value = false
  })
}

function syncSurfaceFromHash() {
  // 浏览器后退/前进等 hash 变化优先：先取消进行中的过渡，避免遮罩滞留或页面失步
  cancelPendingTransition()
  const nextSurface = surfaceFromHash()
  if (nextSurface === surface.value) return
  void setSurface(nextSurface)
}

function clearHeaderIdleTimer() {
  if (headerIdleTimer !== undefined) window.clearTimeout(headerIdleTimer)
  headerIdleTimer = undefined
}

function scheduleHeaderCollapse() {
  clearHeaderIdleTimer()
  if (surface.value !== 'orbit' || !orbitPageActive.value) {
    headerExpanded.value = true
    return
  }
  headerIdleTimer = window.setTimeout(() => {
    if (!orbitPageActive.value) {
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
  if (orbitPageActive.value) scheduleHeaderCollapse()
  else clearHeaderIdleTimer()
}

function registerHeaderActivity() {
  if (!orbitPageActive.value || !headerExpanded.value) return
  const activityAt = performance.now()
  if (activityAt - lastHeaderActivityAt < 400) return
  lastHeaderActivityAt = activityAt
  scheduleHeaderCollapse()
}

function handleWindowPointerMove(event: PointerEvent) {
  if (!headerExpanded.value && orbitPageActive.value && event.clientY <= 16) {
    revealHeader()
    return
  }
  registerHeaderActivity()
}

function collapseHeaderFromScene() {
  if (!orbitPageActive.value) return
  clearHeaderIdleTimer()
  headerExpanded.value = false
}

function updateActivePage() {
  pageSurfaceFrame = 0
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

onMounted(() => {
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
  window.addEventListener('hashchange', syncSurfaceFromHash)
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
  window.removeEventListener('hashchange', syncSurfaceFromHash)
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

    <AuroraCover v-if="surface === 'cover'" class="desktop-cover" @explore="enterSolarSystem" />

    <div v-else class="desktop-app" :class="{ 'header-collapsed': !headerExpanded }">
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
          <a class="brand" href="#home" aria-label="返回 AURORA 封面" @click.prevent="returnToCover">
            <span class="brand-mark"><i /><i /><i /></span>
            <span><strong>AURORA</strong><small>ORBITAL OBSERVATORY</small></span>
          </a>
          <nav v-if="surface === 'orbit'" aria-label="页面导航">
            <a class="solar-system-return" href="#solar-system" @click.prevent="enterSolarSystem">
              <span class="solar-system-icon" aria-hidden="true"><i /><i /><i /></span>
              <span>太阳系</span>
            </a>
            <span class="nav-divider" aria-hidden="true" />
            <a href="#orbit" @click.prevent="enterOrbit">地球</a>
            <a href="#objects">航天器</a>
            <a href="#sites">发射场</a>
            <a href="#launches">发射日程</a>
          </nav>
          <nav v-else class="solar-system-nav" aria-label="当前位置">
            <span class="solar-system-icon is-current" aria-hidden="true"><i /><i /><i /></span>
            <span>太阳系总览</span>
          </nav>
          <div v-if="surface === 'orbit'" class="live-status">
            <span class="status-dot" :class="{ healthy: dataHealthy }" />
            <span>{{ dataHealthy ? '数据正常' : '检查数据' }}</span>
            <strong>{{ timeOnly(now) }} UTC+8</strong>
          </div>
          <div v-else class="live-status solar-clock">
            <span>SOLAR SYSTEM</span>
          </div>
        </div>
      </header>

      <SolarSystem
        v-if="surface === 'solar-system'"
        :enter-from-orbit="solarEnterFromOrbit"
        @select-earth="onEarthSelect"
        @earth-fly-start="onEarthFlyStart"
        @earth-fly-zoom="onEarthFlyZoom"
      />

      <template v-else>
      <section id="orbit" ref="orbitSection" class="orbit-section" :class="{ leaving: orbitSectionLeaving }">
        <div class="page-frame">
          <div ref="orbitSceneFrame" class="scene-frame">
            <OrbitScene
              v-if="overview"
              :spacecraft="overview.spacecraft"
              :sites="overview.launchSites"
              :layers="layers"
              :selection="selection"
              :focus-target="focusTarget"
              :observer-target="observerLocation"
              :observer-active="observerViewActive"
              :day-night-enabled="dayNightEnabled"
              @select="selectFromScene"
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

            <div v-if="overview" class="scene-counts" aria-label="当前载入数据">
              <span><strong>{{ overview.spacecraft.length }}</strong> 航天器</span>
              <span><strong>{{ overview.launchSites.length }}</strong> 发射场</span>
              <span><strong>{{ upcomingEvents.length }}</strong> 近期任务</span>
            </div>

            <aside v-if="selection" class="context-panel" aria-label="所选对象详情">
              <button class="panel-close" aria-label="关闭详情" @click="selection = null">关闭</button>

              <template v-if="selectedSpacecraft">
                <p class="context-type">NORAD {{ selectedSpacecraft.noradCatalogId }}</p>
                <h2>{{ selectedSpacecraft.nameZh }}</h2>
                <p class="context-subtitle">{{ selectedSpacecraft.nameEn }}</p>
                <p class="context-description">{{ selectedSpacecraft.description }}</p>
                <dl>
                  <div><dt>运营方</dt><dd>{{ selectedSpacecraft.operatorName }}</dd></div>
                  <div><dt>轨道倾角</dt><dd>{{ Number(selectedSpacecraft.omm.INCLINATION).toFixed(2) }}°</dd></div>
                  <div><dt>偏心率</dt><dd>{{ Number(selectedSpacecraft.omm.ECCENTRICITY).toFixed(6) }}</dd></div>
                  <div><dt>每日圈数</dt><dd>{{ Number(selectedSpacecraft.omm.MEAN_MOTION).toFixed(3) }}</dd></div>
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
                    <p class="event-original-description">{{ selectedEvent.missionDescription || 'No mission description is currently available.' }}</p>
                    <a :href="selectedEvent.sourceUrl" target="_blank" rel="noreferrer">打开来源页</a>
                  </div>
                </div>
              </template>
            </aside>

            <section v-if="loading" class="system-message"><strong>正在建立轨道数据链路</strong><small>CONNECTING TO AURORA CORE</small></section>
            <section v-else-if="error" class="system-message error-message"><strong>数据链路未建立</strong><p>{{ error }}</p><button @click="load">重新连接</button></section>
          </div>
        </div>
      </section>

      <section id="objects" class="content-section objects-section">
        <div class="page-frame">
          <div class="section-heading">
            <div><p class="section-kicker">OBJECT CATALOG</p><h2>查找航天器</h2></div>
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
            <div><p class="section-kicker launch-kicker">GROUND NETWORK</p><h2>主要发射场</h2></div>
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
            <div><p class="section-kicker launch-kicker">NEXT 30 DAYS</p><h2>发射日程</h2></div>
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
