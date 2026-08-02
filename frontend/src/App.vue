<!--
THESIS: ORBIT is a scrollable observatory, not a fixed cockpit.
OWN-WORLD: deep spatial navy, orbital blue, launch amber, one shared 12-column frame.
STORY: explore Earth first, then search objects, inspect the full launch schedule, and understand sites and sources.
FIRST VIEWPORT: a quiet heading above one dominant globe; controls are compact and details appear only after selection.
FORM: progressive observatory, the assigned seventh Operate structure; dense datasets receive dedicated workspaces below the scene.
-->
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import OrbitScene from './components/OrbitScene.vue'
import { fetchOrbitOverview } from './api'
import { spacecraftPoint } from './orbit/coordinates'
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
let clock: number | undefined

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

const focusTarget = computed(() => {
  if (selectedEvent.value?.latitude != null && selectedEvent.value.longitude != null) {
    return { latitude: selectedEvent.value.latitude, longitude: selectedEvent.value.longitude, distance: 6.15, key: `event:${selectedEvent.value.externalId}` }
  }
  if (selectedSite.value) {
    return { latitude: selectedSite.value.latitude, longitude: selectedSite.value.longitude, distance: 6.3, key: `site:${selectedSite.value.id}` }
  }
  if (selectedSpacecraft.value) {
    const point = spacecraftPoint(selectedSpacecraft.value, now.value)
    if (point) return { latitude: point.latitude, longitude: point.longitude, distance: 6.7, key: `spacecraft:${selectedSpacecraft.value.id}` }
  }
  return {
    latitude: observerLocation.value.latitude,
    longitude: observerLocation.value.longitude,
    distance: 7.6,
    key: `observer:${observerLocation.value.status}:${observerFocusRevision.value}`,
  }
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
  observerFocusRevision.value += 1
  if (observerLocation.value.status !== 'located') requestObserverLocation()
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
  selection.value = nextSelection
  window.requestAnimationFrame(() => document.querySelector('#orbit')?.scrollIntoView({ behavior: 'smooth', block: 'start' }))
}

function timeOnly(value: Date | string) {
  return new Intl.DateTimeFormat('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false }).format(new Date(value))
}

function eventDate(value: string) {
  const date = new Date(value)
  return {
    day: new Intl.DateTimeFormat('zh-CN', { day: '2-digit' }).format(date),
    month: new Intl.DateTimeFormat('en-US', { month: 'short' }).format(date).toUpperCase(),
    weekday: new Intl.DateTimeFormat('zh-CN', { weekday: 'short' }).format(date),
    time: new Intl.DateTimeFormat('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false }).format(date),
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

onMounted(() => {
  load()
  requestObserverLocation()
  clock = window.setInterval(() => { now.value = new Date() }, 1000)
})
onBeforeUnmount(() => { if (clock) window.clearInterval(clock) })
</script>

<template>
  <main class="aurora-shell">
    <div class="desktop-only">
      <span>AURORA / ORBIT</span>
      <h1>请使用电脑浏览器查看</h1>
      <p>当前原型专注桌面端三维交互，移动端适配将在后续阶段加入。</p>
    </div>

    <div class="desktop-app">
      <header class="site-header">
        <div class="page-frame header-inner">
          <a class="brand" href="#orbit" aria-label="返回 ORBIT 观察区">
            <span class="brand-mark"><i /><i /><i /></span>
            <span><strong>AURORA</strong><small>ORBITAL OBSERVATORY</small></span>
          </a>
          <nav aria-label="页面导航">
            <a href="#orbit">地球</a>
            <a href="#objects">航天器</a>
            <a href="#launches">发射日程</a>
            <a href="#sites">发射场</a>
          </nav>
          <div class="live-status">
            <span class="status-dot" :class="{ healthy: dataHealthy }" />
            <span>{{ dataHealthy ? '数据正常' : '检查数据' }}</span>
            <strong>{{ timeOnly(now) }} CST</strong>
          </div>
        </div>
      </header>

      <section id="orbit" class="orbit-section">
        <div class="page-frame">
          <div class="scene-frame">
            <OrbitScene
              v-if="overview"
              :spacecraft="overview.spacecraft"
              :sites="overview.launchSites"
              :layers="layers"
              :selection="selection"
              :focus-target="focusTarget"
              :observer-target="observerLocation"
              @select="selection = $event"
            />

            <div class="scene-toolbar" aria-label="场景图层">
              <span>图层</span>
              <label><input v-model="layers.spacecraft" type="checkbox"><i />航天器</label>
              <label><input v-model="layers.orbits" type="checkbox"><i />轨道</label>
              <label><input v-model="layers.sites" type="checkbox"><i class="amber" />发射场</label>
            </div>

            <button v-if="!selection" class="scene-location" type="button" @click="focusObserver">
              <i :class="observerLocation.status" />
              <span><small>默认中心</small><strong>{{ observerLocation.label }}</strong></span>
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
                <h2>{{ selectedEvent.missionName || selectedEvent.name }}</h2>
                <p class="context-subtitle">{{ selectedEvent.name }}</p>
                <div class="event-clock"><strong>{{ eventDate(selectedEvent.net).month }} {{ eventDate(selectedEvent.net).day }}</strong><span>{{ eventDate(selectedEvent.net).time }} CST</span></div>
                <p class="context-description">{{ selectedEvent.missionDescription || '任务详情暂未公开。' }}</p>
                <dl>
                  <div><dt>状态</dt><dd>{{ selectedEvent.statusName }}</dd></div>
                  <div><dt>发射台</dt><dd>{{ selectedEvent.padName }}</dd></div>
                  <div><dt>地点</dt><dd>{{ selectedEvent.locationName }}</dd></div>
                </dl>
                <div class="site-context">
                  <p>发射场</p>
                  <template v-if="selectedEventSite">
                    <strong>{{ selectedEventSite.nameZh }}</strong>
                    <span>{{ selectedEventSite.description }}</span>
                  </template>
                  <template v-else>
                    <strong>{{ selectedEvent.padName || selectedEvent.locationName }}</strong>
                    <span>当前事件源提供了位置和发射台信息，详细场地资料将在后续数据扩充中补充。</span>
                  </template>
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
            <p>当前载入代表性对象。目录按未来数千个对象的使用方式设计，支持关键词、正则表达式、筛选、排序和后端分页扩展。</p>
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

      <section id="launches" class="content-section launches-section">
        <div class="page-frame">
          <div class="section-heading">
            <div><p class="section-kicker launch-kicker">NEXT 30 DAYS</p><h2>发射日程</h2></div>
            <p>完整展示当前数据源中未来 30 天的全部任务。选择任一任务，地球会转向发射位置并显示任务与场地资料。</p>
          </div>
          <div class="launch-list">
            <div class="launch-list-head"><span>日期</span><span>任务</span><span>状态</span><span>发射地点</span><span>时间</span></div>
            <button v-for="event in upcomingEvents" :key="event.externalId" class="launch-row" @click="selectAndFocus({ kind: 'event', id: event.externalId })">
              <time><strong>{{ eventDate(event.net).day }}</strong><span>{{ eventDate(event.net).month }} · {{ eventDate(event.net).weekday }}</span></time>
              <span class="launch-mission"><strong>{{ event.missionName || event.name }}</strong><small>{{ event.name }}</small></span>
              <span><i :class="event.statusAbbrev.toLowerCase()" />{{ event.statusName }}</span>
              <span>{{ event.locationName || event.padName }}</span>
              <span class="launch-time">{{ eventDate(event.net).time }}<small>CST</small></span>
            </button>
            <div v-if="!upcomingEvents.length" class="catalog-empty">未来 30 天内暂无已载入事件。</div>
          </div>
        </div>
      </section>

      <section id="sites" class="content-section sites-section">
        <div class="page-frame">
          <div class="section-heading">
            <div><p class="section-kicker launch-kicker">GROUND NETWORK</p><h2>主要发射场</h2></div>
            <p>选择场地即可回到地球定位。当前先收录少量代表性发射场，后续按国家、轨道能力和任务记录扩展。</p>
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

      <footer class="site-footer">
        <div class="page-frame footer-inner">
          <div><strong>AURORA / ORBIT</strong><p>公开航天数据的三维探索与阅读界面。</p></div>
          <div class="source-list"><span v-for="source in overview?.freshness" :key="source.sourceCode"><i :class="{ healthy: source.success }" />{{ source.sourceName }} · {{ new Date(source.lastFinishedAt).toLocaleString('zh-CN', { hour12: false }) }}</span></div>
        </div>
      </footer>
    </div>
  </main>
</template>
