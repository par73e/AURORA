<!--
THESIS: SKY is a local observing decision room, not a weather dashboard or a generic star-map clone.
OWN-WORLD: midnight blue-black, lunar white, horizon amber, flat instrument rails, and one continuous sky field.
STORY: establish place and time, judge the observing window, choose a target, then understand the next event.
FIRST VIEWPORT: a permanent left observatory rail frames a wide horizon simulation with one plain-language verdict.
FORM: desktop field observatory; four focused workspaces share one clock, one location, and one evidence boundary.
-->
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

type SkyPage = 'tonight' | 'windows' | 'targets' | 'events'

interface Target {
  id: string
  name: string
  nameEn: string
  symbol: string
  color: string
  window: string
  direction: string
  altitude: string
  magnitude: string
  equipment: string
  reason: string
}

interface EventItem {
  id: number
  day: string
  month: string
  type: '太阳系' | '流星雨' | '月相'
  title: string
  description: string
  window: string
  visibility: string
}

const emit = defineEmits<{ home: [] }>()

const menu = [
  { id: 'tonight' as const, numeral: 'Ⅰ', label: '今夜', en: 'TONIGHT' },
  { id: 'windows' as const, numeral: 'Ⅱ', label: '窗口', en: 'WINDOWS' },
  { id: 'targets' as const, numeral: 'Ⅲ', label: '目标', en: 'TARGETS' },
  { id: 'events' as const, numeral: 'Ⅳ', label: '天象', en: 'EVENTS' },
]

const titles: Record<SkyPage, { kicker: string; title: string; summary: string }> = {
  tonight: { kicker: 'LOCAL SKY / 观测判定', title: '今晚的天空', summary: '先判断能不能看，再决定看什么。' },
  windows: { kicker: 'DARKNESS / 时间条件', title: '观测窗口', summary: '把黑夜、月光、云量和天体可见时间放在同一条时间轴上。' },
  targets: { kicker: 'OBJECTS / 目标建议', title: '今晚看什么', summary: '一个首选目标，两项备选，以及清楚的现场指引。' },
  events: { kicker: 'NEXT 30 DAYS / 近期事件', title: '天象日历', summary: '只保留真正值得提前安排时间的重要天象。' },
}

function pageFromHash(): SkyPage {
  const hash = window.location.hash
  if (hash === '#sky-windows') return 'windows'
  if (hash === '#sky-targets') return 'targets'
  if (hash === '#sky-events') return 'events'
  return 'tonight'
}

const activePage = ref<SkyPage>(pageFromHash())
const minuteOfDay = ref(20 * 60 + 12)
const now = ref(new Date())
const locationLabel = ref('上海 · 黄浦区')
const latitude = ref(31.23)
const longitude = ref(121.47)
const locationStatus = ref<'fallback' | 'locating' | 'located'>('fallback')
const selectedTargetId = ref('mars')
const targetQuery = ref('')
const eventFilter = ref<'全部' | EventItem['type']>('全部')
let clock: number | undefined

const targets: Target[] = [
  {
    id: 'mars', name: '火星', nameEn: 'MARS', symbol: '●', color: '#d98b72', window: '20:18—22:46',
    direction: '西南偏西', altitude: '31°', magnitude: '1.3', equipment: '裸眼可见',
    reason: '当前与月亮距离较近，适合作为今晚第一个定位目标。',
  },
  {
    id: 'saturn', name: '土星', nameEn: 'SATURN', symbol: '◉', color: '#d8c58c', window: '21:28—次日 04:36',
    direction: '东南', altitude: '42°', magnitude: '0.5', equipment: '双筒镜更佳',
    reason: '升起较晚，但拥有更长的深夜观测窗口和更稳定的高度。',
  },
  {
    id: 'moon', name: '月亮', nameEn: 'MOON', symbol: '◐', color: '#dbe5ee', window: '20:12—01:56',
    direction: '西南', altitude: '34°', magnitude: '17% 亮面', equipment: '裸眼可见',
    reason: '低照度残月对深空观测干扰较小，也适合观察明暗交界附近的月貌。',
  },
]

const events: EventItem[] = [
  { id: 1, day: '09', month: 'AUG', type: '太阳系', title: '火星与月亮接近', description: '月亮可作为定位参照，适合裸眼观察两者在暮色中的相对位置。', window: '日落后 40 分钟', visibility: '西南低空' },
  { id: 2, day: '12', month: 'AUG', type: '流星雨', title: '夏季流星雨观测窗口', description: '选择背离城市光源的开阔天空，并为眼睛预留二十分钟暗适应。', window: '23:30—次日 04:10', visibility: '东北至天顶' },
  { id: 3, day: '16', month: 'AUG', type: '月相', title: '上弦月附近', description: '明暗交界附近阴影更明显，是观察环形山和月海边缘的舒适时段。', window: '日落至午夜', visibility: '南方天空' },
  { id: 4, day: '23', month: 'AUG', type: '太阳系', title: '土星整夜窗口', description: '目标高度在午夜前后达到较佳区间，适合安排双筒镜或小型望远镜。', window: '21:40—次日 04:50', visibility: '东南至西南' },
]

const selectedTarget = computed(() => targets.find((target) => target.id === selectedTargetId.value) ?? targets[0])
const filteredTargets = computed(() => {
  const query = targetQuery.value.trim().toLocaleLowerCase()
  if (!query) return targets
  return targets.filter((target) => `${target.name} ${target.nameEn}`.toLocaleLowerCase().includes(query))
})
const filteredEvents = computed(() => eventFilter.value === '全部' ? events : events.filter((event) => event.type === eventFilter.value))
const currentTitle = computed(() => titles[activePage.value])
const timeLabel = computed(() => `${String(Math.floor(minuteOfDay.value / 60)).padStart(2, '0')}:${String(minuteOfDay.value % 60).padStart(2, '0')}`)
const timeProgress = computed(() => minuteOfDay.value / 1440)
const isNight = computed(() => minuteOfDay.value >= 19 * 60 || minuteOfDay.value < 5 * 60 + 30)
const skyPhase = computed(() => isNight.value ? 'night' : minuteOfDay.value < 7 * 60 || minuteOfDay.value > 17 * 60 + 30 ? 'twilight' : 'day')
const coordinateLabel = computed(() => `${Math.abs(latitude.value).toFixed(2)}°${latitude.value >= 0 ? 'N' : 'S'} · ${Math.abs(longitude.value).toFixed(2)}°${longitude.value >= 0 ? 'E' : 'W'}`)

function objectStyle(offset: number, maxHeight: number) {
  const cycle = (timeProgress.value + offset + 1) % 1
  const altitude = Math.sin(Math.PI * cycle)
  return {
    left: `${5 + cycle * 90}%`,
    bottom: `${9 + Math.max(0, altitude) * maxHeight}%`,
    opacity: altitude > .08 ? 1 : .18,
  }
}

function selectPage(page: SkyPage) {
  activePage.value = page
  const hash = page === 'tonight' ? '#sky-tonight' : `#sky-${page}`
  window.history.pushState(null, '', hash)
  document.querySelector('.sky-content-scroll')?.scrollTo({ top: 0, behavior: 'smooth' })
}

function requestLocation() {
  if (!navigator.geolocation || locationStatus.value === 'locating') return
  locationStatus.value = 'locating'
  navigator.geolocation.getCurrentPosition(
    ({ coords }) => {
      latitude.value = coords.latitude
      longitude.value = coords.longitude
      locationLabel.value = '当前观测地'
      locationStatus.value = 'located'
    },
    () => {
      locationStatus.value = 'fallback'
      locationLabel.value = '上海 · 黄浦区'
    },
    { enableHighAccuracy: false, timeout: 6000, maximumAge: 900_000 },
  )
}

function formatClock(value: Date) {
  return new Intl.DateTimeFormat('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false }).format(value)
}

function onPopState() {
  if (window.location.hash.startsWith('#sky')) activePage.value = pageFromHash()
}

onMounted(() => {
  window.addEventListener('popstate', onPopState)
  clock = window.setInterval(() => { now.value = new Date() }, 1000)
})

onBeforeUnmount(() => {
  window.removeEventListener('popstate', onPopState)
  if (clock) window.clearInterval(clock)
})
</script>

<template>
  <section class="sky-shell" aria-label="AURORA 天文观测">
    <aside class="sky-sidebar">
      <a class="sky-brand" href="#home" aria-label="返回 AURORA 封面" @click="emit('home')">
        <span class="sky-brand-orbit" aria-hidden="true"><i /><i /></span>
        <span><strong>AURORA</strong><small>LOCAL SKY OBSERVATORY</small></span>
      </a>

      <button class="sky-location" type="button" :aria-busy="locationStatus === 'locating'" @click="requestLocation">
        <span class="location-mark" aria-hidden="true" />
        <span><strong>{{ locationStatus === 'locating' ? '正在获取位置' : locationLabel }}</strong><small>{{ coordinateLabel }}</small></span>
        <i>{{ locationStatus === 'located' ? 'GPS' : '设置' }}</i>
      </button>

      <nav class="sky-menu" aria-label="天文观测页面">
        <button
          v-for="item in menu"
          :key="item.id"
          type="button"
          :class="{ active: activePage === item.id }"
          :aria-current="activePage === item.id ? 'page' : undefined"
          @click="selectPage(item.id)"
        >
          <i>{{ item.numeral }}</i>
          <span><strong>{{ item.label }}</strong><small>{{ item.en }}</small></span>
          <b aria-hidden="true">↗</b>
        </button>
      </nav>

      <div class="sky-sidebar-foot">
        <div><span class="preview-dot" /><strong>界面预览数据</strong></div>
        <p>尚未接入天气与天文计算服务，当前数值仅用于验证页面结构。</p>
        <time>{{ formatClock(now) }} <small>UTC+8</small></time>
      </div>
    </aside>

    <main class="sky-content-scroll">
      <header class="sky-page-header">
        <div>
          <p>{{ currentTitle.kicker }}</p>
          <h1>{{ currentTitle.title }}</h1>
          <span>{{ currentTitle.summary }}</span>
        </div>
        <div class="sky-date">
          <small>2026 / 08 / 09</small>
          <strong>{{ timeLabel }}</strong>
          <span>星期日 · 上海时间</span>
        </div>
      </header>

      <div :key="activePage" class="sky-page-stage">
        <template v-if="activePage === 'tonight'">
          <section class="tonight-sky" :class="skyPhase" aria-label="天空方位模拟">
            <div class="sky-grid" aria-hidden="true">
              <span class="altitude a60">60°</span><span class="altitude a30">30°</span><span class="altitude a0">0°</span>
              <span class="direction east">东</span><span class="direction south">南</span><span class="direction west">西</span>
            </div>
            <div class="celestial-object moon" :style="objectStyle(.18, 66)"><i /><span>月亮</span></div>
            <div class="celestial-object venus" :style="objectStyle(.03, 58)"><i /><span>金星</span></div>
            <div class="celestial-object mars" :style="objectStyle(.24, 48)"><i /><span>火星</span></div>
            <div class="celestial-object saturn" :style="objectStyle(.51, 62)"><i /><span>土星</span></div>
            <div class="horizon" aria-hidden="true" />
            <div class="tonight-verdict">
              <p>今夜判定</p>
              <h2>条件较差，但仍有短暂窗口</h2>
              <span>低云与城市天光是主要限制；优先观察月面、火星和亮行星。</span>
            </div>
          </section>

          <section class="condition-ledger" aria-label="观测条件分解">
            <div class="condition-summary">
              <p>OBSERVING CONDITION</p>
              <strong>04<small>/ 100</small></strong>
              <span>不宜进行暗弱深空目标观测</span>
            </div>
            <dl>
              <div><dt>暗夜窗口</dt><dd>20:12—01:56</dd><span class="good">可用 5h 44m</span></div>
              <div><dt>总云量</dt><dd>92%</dd><span class="poor">主要限制</span></div>
              <div><dt>月光干扰</dt><dd>17%</dd><span class="good">较低</span></div>
              <div><dt>光污染</dt><dd>B8</dd><span class="warn">城市核心</span></div>
              <div><dt>能见度</dt><dd>3 km</dd><span class="poor">较差</span></div>
            </dl>
          </section>

          <section class="time-console">
            <label for="sky-time"><span>模拟时间</span><strong>{{ timeLabel }}</strong></label>
            <input id="sky-time" v-model.number="minuteOfDay" type="range" min="0" max="1439" step="5">
            <div><span>00:00</span><span>06:00</span><span>12:00</span><span>18:00</span><span>24:00</span></div>
          </section>
        </template>

        <template v-else-if="activePage === 'windows'">
          <section class="window-callout">
            <div><p>BEST AVAILABLE WINDOW</p><h2>20:12—22:40</h2></div>
            <p>天色已进入天文黑夜，月光影响较低；22:40 后低云继续增厚，因此窗口虽短但可用于亮目标观测。</p>
            <span>建议提前 20 分钟到达观测地</span>
          </section>

          <section class="window-chart" aria-label="今夜观测时间轴">
            <div class="chart-head"><span>条件 / 时间</span><b>18</b><b>21</b><b>00</b><b>03</b><b>06</b></div>
            <div class="chart-row"><strong>天空亮度<small>暮光与黑夜</small></strong><div class="track darkness"><i /></div></div>
            <div class="chart-row"><strong>低云<small>预览值</small></strong><div class="track clouds"><i /><i /><i /><i /><i /></div></div>
            <div class="chart-row"><strong>月亮<small>17% 亮面</small></strong><div class="track moon-track"><i /></div></div>
            <div class="chart-row"><strong>银河窗口<small>核心位于地平线上</small></strong><div class="track milky"><i /></div></div>
            <div class="chart-row"><strong>火星<small>裸眼</small></strong><div class="track mars-track"><i /></div></div>
            <div class="chart-row"><strong>土星<small>双筒镜</small></strong><div class="track saturn-track"><i /></div></div>
            <div class="window-cursor"><i /><span>20:12</span></div>
          </section>

          <section class="window-legend">
            <div><i class="available" /><span><strong>可用窗口</strong>同时满足黑夜、目标高度与基础环境条件</span></div>
            <div><i class="uncertain" /><span><strong>不确定</strong>预报或地形可能改变实际可见性</span></div>
            <div><i class="blocked" /><span><strong>不建议</strong>目标位于地平线下或受云层明显影响</span></div>
          </section>
        </template>

        <template v-else-if="activePage === 'targets'">
          <section class="target-feature">
            <div class="target-visual" :style="{ '--target-color': selectedTarget.color }">
              <span class="target-orbit" /><i>{{ selectedTarget.symbol }}</i><small>{{ selectedTarget.nameEn }}</small>
            </div>
            <div class="target-copy">
              <p>PRIMARY TARGET / 今夜首选</p>
              <h2>{{ selectedTarget.name }}</h2>
              <span>{{ selectedTarget.reason }}</span>
              <dl>
                <div><dt>窗口</dt><dd>{{ selectedTarget.window }}</dd></div>
                <div><dt>方向</dt><dd>{{ selectedTarget.direction }}</dd></div>
                <div><dt>高度</dt><dd>{{ selectedTarget.altitude }}</dd></div>
                <div><dt>亮度</dt><dd>{{ selectedTarget.magnitude }}</dd></div>
              </dl>
              <strong class="equipment">{{ selectedTarget.equipment }}</strong>
            </div>
          </section>

          <section class="target-directory">
            <div class="directory-head">
              <div><p>VISIBLE OBJECTS</p><h3>候选目标</h3></div>
              <label><span>搜索当前目录</span><input v-model="targetQuery" type="search" placeholder="月亮 / MARS"></label>
            </div>
            <button v-for="target in filteredTargets" :key="target.id" type="button" :class="{ active: selectedTargetId === target.id }" @click="selectedTargetId = target.id">
              <i :style="{ color: target.color }">{{ target.symbol }}</i>
              <span><strong>{{ target.name }}</strong><small>{{ target.nameEn }}</small></span>
              <span>{{ target.window }}</span><span>{{ target.direction }}</span><span>{{ target.equipment }}</span><b>查看 ↗</b>
            </button>
            <p v-if="!filteredTargets.length" class="target-empty">当前演示目录中没有匹配的目标。</p>
          </section>

          <section class="field-note">
            <p>现场提示</p>
            <strong>先找到月亮，再向其右下方寻找火星。</strong>
            <span>关闭手机高亮屏幕并等待至少 15 分钟暗适应；城市核心区域不建议以银河或暗弱深空天体作为首要目标。</span>
          </section>
        </template>

        <template v-else>
          <section class="event-toolbar">
            <div><p>OBSERVATION CALENDAR</p><h2>未来 30 天</h2></div>
            <div role="group" aria-label="天象类型筛选">
              <button v-for="filter in ['全部', '太阳系', '流星雨', '月相'] as const" :key="filter" type="button" :class="{ active: eventFilter === filter }" @click="eventFilter = filter">{{ filter }}</button>
            </div>
          </section>

          <section class="event-list" aria-live="polite">
            <article v-for="event in filteredEvents" :key="event.id">
              <time><strong>{{ event.day }}</strong><span>{{ event.month }}</span></time>
              <div><span>{{ event.type }}</span><h3>{{ event.title }}</h3><p>{{ event.description }}</p></div>
              <dl><div><dt>建议窗口</dt><dd>{{ event.window }}</dd></div><div><dt>天空位置</dt><dd>{{ event.visibility }}</dd></div></dl>
              <button type="button" aria-label="查看天象详情">↗</button>
            </article>
          </section>

          <section class="event-boundary">
            <div><p>DATA BOUNDARY</p><h3>当前为结构与交互预览</h3></div>
            <p>日期、天气与天体位置尚未连接真实服务。正式接入后，每项事件将显示来源、计算时刻、适用地点与可见性说明。</p>
          </section>
        </template>
      </div>
    </main>
  </section>
</template>

<style scoped>
.sky-shell {
  --sky-bg: #02060b;
  --sky-panel: #071019;
  --sky-panel-soft: #0a1520;
  --sky-line: rgba(150, 185, 204, .16);
  --sky-line-strong: rgba(174, 207, 225, .28);
  --sky-ink: #edf5f7;
  --sky-muted: #8da2ad;
  --sky-quiet: #5c727e;
  --sky-lunar: #c9d9e3;
  --sky-amber: #e4b66f;
  display: grid;
  grid-template-columns: 252px minmax(0, 1fr);
  width: 100%;
  height: 100dvh;
  min-height: 680px;
  overflow: hidden;
  background: var(--sky-bg);
  color: var(--sky-ink);
}

.sky-sidebar { position: relative; z-index: 2; display: flex; flex-direction: column; min-width: 0; padding: 30px 24px 22px; border-right: 1px solid var(--sky-line); background: #03080e; }
.sky-sidebar::after { content: ''; position: absolute; left: 0; right: 0; top: 0; height: 38%; pointer-events: none; background: radial-gradient(circle at 36% 0%, rgba(84, 127, 151, .12), transparent 68%); }
.sky-brand, .sky-location, .sky-menu, .sky-sidebar-foot { position: relative; z-index: 1; }
.sky-brand { display: flex; align-items: center; gap: 12px; color: inherit; text-decoration: none; }
.sky-brand-orbit { position: relative; width: 34px; height: 34px; border: 1px solid rgba(201, 217, 227, .42); border-radius: 50%; transition: transform .25s ease, border-color .25s ease; }
.sky-brand-orbit i:first-child { position: absolute; left: 50%; top: 50%; width: 4px; height: 4px; border-radius: 50%; background: var(--sky-lunar); transform: translate(-50%, -50%); }
.sky-brand-orbit i:last-child { position: absolute; inset: 8px -5px; border: 1px solid rgba(228, 182, 111, .45); border-radius: 50%; transform: rotate(-24deg); }
.sky-brand:hover .sky-brand-orbit, .sky-brand:focus-visible .sky-brand-orbit { border-color: var(--sky-lunar); transform: rotate(8deg); }
.sky-brand:focus-visible { outline: 1px solid var(--sky-lunar); outline-offset: 6px; }
.sky-brand strong, .sky-brand small { display: block; }
.sky-brand strong { font-size: 13px; font-weight: 500; letter-spacing: .2em; }
.sky-brand small { margin-top: 4px; color: var(--sky-quiet); font: 400 7px var(--font-mono); letter-spacing: .12em; }

.sky-location { display: grid; grid-template-columns: 26px 1fr auto; gap: 10px; align-items: center; width: 100%; margin: 50px 0 34px; padding: 15px 0; border: 0; border-top: 1px solid var(--sky-line); border-bottom: 1px solid var(--sky-line); background: transparent; color: inherit; text-align: left; }
.sky-location:hover strong, .sky-location:focus-visible strong { color: var(--sky-amber); }
.sky-location:focus-visible { outline: 1px solid var(--sky-amber); outline-offset: 4px; }
.location-mark { position: relative; width: 17px; height: 17px; border: 1px solid var(--sky-lunar); border-radius: 50% 50% 50% 0; transform: rotate(-45deg); }
.location-mark::after { content: ''; position: absolute; left: 5px; top: 5px; width: 5px; height: 5px; border-radius: 50%; background: var(--sky-amber); }
.sky-location strong, .sky-location small { display: block; }
.sky-location strong { overflow: hidden; font-size: 11px; font-weight: 500; text-overflow: ellipsis; white-space: nowrap; transition: color .2s; }
.sky-location small { margin-top: 4px; color: var(--sky-quiet); font: 400 8px var(--font-mono); }
.sky-location > i { color: var(--sky-quiet); font: normal 400 8px var(--font-mono); }

.sky-menu { display: grid; }
.sky-menu button { position: relative; display: grid; grid-template-columns: 30px 1fr auto; gap: 12px; align-items: center; min-height: 72px; padding: 0; border: 0; border-bottom: 1px solid var(--sky-line); background: transparent; color: var(--sky-muted); text-align: left; transition: color .2s; }
.sky-menu button:first-child { border-top: 1px solid var(--sky-line); }
.sky-menu button::before { content: ''; position: absolute; left: -24px; top: 0; bottom: 0; width: 2px; background: var(--sky-amber); transform: scaleY(0); transition: transform .3s cubic-bezier(.16, 1, .3, 1); }
.sky-menu button:hover, .sky-menu button:focus-visible, .sky-menu button.active { color: var(--sky-ink); outline: 0; }
.sky-menu button > * { transition: transform .35s cubic-bezier(.16, 1, .3, 1); }
.sky-menu button:hover > *, .sky-menu button:focus-visible > *, .sky-menu button.active > * { transform: translateX(8px); }
.sky-menu button.active::before { transform: scaleY(1); }
.sky-menu button > i { color: var(--sky-quiet); font: normal 400 10px var(--font-mono); }
.sky-menu button.active > i { color: var(--sky-amber); }
.sky-menu strong, .sky-menu small { display: block; }
.sky-menu strong { font-size: 13px; font-weight: 500; }
.sky-menu small { margin-top: 4px; color: var(--sky-quiet); font: 400 7px var(--font-mono); letter-spacing: .14em; }
.sky-menu b { opacity: 0; color: var(--sky-amber); font-size: 11px; font-weight: 400; transform: translateX(-5px); transition: opacity .2s, transform .25s; }
.sky-menu button.active b, .sky-menu button:hover b { opacity: 1; transform: none; }

.sky-sidebar-foot { margin-top: auto; padding-top: 26px; }
.sky-sidebar-foot > div { display: flex; align-items: center; gap: 8px; }
.preview-dot { width: 5px; height: 5px; border-radius: 50%; background: var(--sky-amber); }
.sky-sidebar-foot strong { color: #b6c5cc; font-size: 9px; font-weight: 500; }
.sky-sidebar-foot p { margin: 9px 0 20px; color: var(--sky-quiet); font-size: 8px; line-height: 1.65; }
.sky-sidebar-foot time { display: block; color: var(--sky-lunar); font: 400 15px var(--font-mono); }
.sky-sidebar-foot time small { color: var(--sky-quiet); font-size: 7px; }

.sky-content-scroll { min-width: 0; overflow: auto; background: radial-gradient(circle at 78% -10%, rgba(65, 101, 122, .12), transparent 38%), var(--sky-bg); scrollbar-color: rgba(158, 191, 208, .24) transparent; }
.sky-page-header { display: flex; justify-content: space-between; align-items: end; min-height: 176px; padding: 48px clamp(48px, 5vw, 86px) 28px; border-bottom: 1px solid var(--sky-line); }
.sky-page-header p, .condition-summary p, .target-copy > p, .window-callout p:first-child, .event-toolbar p, .event-boundary p:first-child, .directory-head p { margin: 0 0 12px; color: var(--sky-amber); font: 500 8px var(--font-mono); letter-spacing: .14em; }
.sky-page-header h1 { margin: 0; font-size: clamp(36px, 4vw, 58px); font-weight: 400; line-height: 1; letter-spacing: -.025em; }
.sky-page-header > div:first-child > span { display: block; max-width: 660px; margin-top: 14px; color: var(--sky-muted); font-size: 12px; }
.sky-date { display: grid; justify-items: end; padding-bottom: 2px; }
.sky-date small { color: var(--sky-quiet); font: 400 8px var(--font-mono); letter-spacing: .08em; }
.sky-date strong { margin: 5px 0; color: var(--sky-lunar); font: 400 28px var(--font-mono); }
.sky-date span { color: var(--sky-quiet); font-size: 9px; }
.sky-page-stage { min-height: calc(100dvh - 176px); padding: 34px clamp(48px, 5vw, 86px) 70px; animation: stage-enter .48s cubic-bezier(.16, 1, .3, 1) both; }

.tonight-sky { position: relative; height: min(48vw, 520px); min-height: 390px; overflow: hidden; border-bottom: 1px solid var(--sky-line-strong); background: radial-gradient(1px 1px at 11% 18%, rgba(230, 241, 246, .68), transparent), radial-gradient(1px 1px at 22% 37%, rgba(230, 241, 246, .5), transparent), radial-gradient(1px 1px at 37% 14%, rgba(230, 241, 246, .58), transparent), radial-gradient(1.3px 1.3px at 58% 24%, rgba(230, 241, 246, .65), transparent), radial-gradient(1px 1px at 76% 12%, rgba(230, 241, 246, .54), transparent), radial-gradient(.8px .8px at 91% 36%, rgba(230, 241, 246, .6), transparent), linear-gradient(180deg, #07101c, #0b1b28 64%, #18232b); transition: background .7s ease; }
.tonight-sky.twilight { background: linear-gradient(180deg, #253c56, #9a8676 70%, #d9b782); }
.tonight-sky.day { background: linear-gradient(180deg, #567e9c, #b5c7cf 70%, #e6cfa6); }
.sky-grid { position: absolute; inset: 8% 0 7%; }
.sky-grid::before, .sky-grid::after { content: ''; position: absolute; left: 0; right: 0; border-top: 1px dashed rgba(196, 217, 227, .14); }
.sky-grid::before { top: 27%; }.sky-grid::after { top: 62%; }
.altitude, .direction { position: absolute; z-index: 2; color: rgba(221, 233, 239, .42); font: 400 9px var(--font-mono); }
.a60 { left: 10px; top: 25%; }.a30 { left: 10px; top: 60%; }.a0 { left: 10px; bottom: 4%; }
.direction { bottom: -2%; font-family: var(--font-sans); font-size: 13px; }.east { left: 20%; }.south { left: 50%; }.west { right: 20%; }
.horizon { position: absolute; z-index: 3; left: -2%; right: -2%; bottom: -26px; height: 92px; background: #02070c; clip-path: polygon(0 37%, 7% 34%, 14% 40%, 23% 30%, 31% 35%, 39% 27%, 48% 38%, 58% 32%, 67% 40%, 77% 29%, 88% 36%, 100% 31%, 100% 100%, 0 100%); }
.celestial-object { position: absolute; z-index: 4; display: grid; justify-items: center; transition: left .45s cubic-bezier(.16, 1, .3, 1), bottom .45s cubic-bezier(.16, 1, .3, 1), opacity .3s; }
.celestial-object i { display: block; width: 18px; height: 18px; border-radius: 50%; box-shadow: 0 4px 18px rgba(0, 0, 0, .4); }
.celestial-object span { margin-top: 5px; padding: 3px 6px; background: rgba(2, 7, 12, .68); color: #d4e1e7; font-size: 8px; }
.celestial-object.moon i { width: 28px; height: 28px; background: radial-gradient(circle at 32% 40%, #c9d4db 0 22%, #111b29 25% 100%); box-shadow: 5px 0 16px rgba(201, 217, 227, .2); }
.celestial-object.venus i { background: #e8d3a2; }.celestial-object.mars i { background: #c87a61; }.celestial-object.saturn i { background: #d8c58c; }
.celestial-object.saturn i::after { content: ''; display: block; width: 28px; height: 8px; border: 1px solid rgba(232, 211, 158, .74); border-radius: 50%; transform: translate(-5px, 5px) rotate(-12deg); }
.tonight-verdict { position: absolute; z-index: 5; left: 34px; bottom: 44px; max-width: 460px; }
.tonight-verdict p { margin: 0 0 8px; color: var(--sky-amber); font: 500 8px var(--font-mono); letter-spacing: .12em; }
.tonight-verdict h2 { margin: 0; font-size: clamp(24px, 2.6vw, 38px); font-weight: 400; letter-spacing: -.02em; }
.tonight-verdict span { display: block; max-width: 440px; margin-top: 12px; color: #aabac2; font-size: 10px; line-height: 1.7; }

.condition-ledger { display: grid; grid-template-columns: minmax(230px, .72fr) 2fr; gap: clamp(36px, 5vw, 84px); padding: 40px 0; border-bottom: 1px solid var(--sky-line); }
.condition-summary strong { display: block; font: 400 clamp(54px, 6vw, 82px)/1 var(--font-mono); letter-spacing: -.06em; }.condition-summary strong small { color: var(--sky-quiet); font-size: 13px; letter-spacing: 0; }
.condition-summary > span { display: block; margin-top: 12px; color: #b07e72; font-size: 9px; }
.condition-ledger dl { display: grid; grid-template-columns: repeat(5, 1fr); margin: 0; border-top: 1px solid var(--sky-line); }
.condition-ledger dl > div { padding: 18px 12px 14px 0; border-bottom: 1px solid var(--sky-line); }
.condition-ledger dt { color: var(--sky-quiet); font-size: 9px; }.condition-ledger dd { margin: 7px 0 6px; color: var(--sky-ink); font: 400 17px var(--font-mono); }
.condition-ledger dl span { font-size: 8px; }.good { color: #7fcab2; }.poor { color: #c37a70; }.warn { color: var(--sky-amber); }
.time-console { padding: 34px 0 0; }.time-console label { display: flex; justify-content: space-between; align-items: baseline; color: var(--sky-muted); font-size: 10px; }.time-console label strong { color: var(--sky-lunar); font: 400 20px var(--font-mono); }
.time-console input { width: 100%; margin: 23px 0 12px; accent-color: var(--sky-amber); }.time-console > div { display: flex; justify-content: space-between; color: var(--sky-quiet); font: 400 8px var(--font-mono); }

.window-callout { display: grid; grid-template-columns: minmax(230px, .7fr) 1.5fr auto; gap: 42px; align-items: end; padding: 12px 0 34px; border-bottom: 1px solid var(--sky-line-strong); }
.window-callout h2 { margin: 0; color: var(--sky-lunar); font: 400 clamp(34px, 4vw, 56px) var(--font-mono); letter-spacing: -.04em; }.window-callout > p { max-width: 620px; margin: 0; color: var(--sky-muted); font-size: 11px; line-height: 1.8; }.window-callout > span { padding-bottom: 4px; color: var(--sky-amber); font-size: 9px; }
.window-chart { position: relative; margin-top: 34px; border-top: 1px solid var(--sky-line); border-bottom: 1px solid var(--sky-line); }
.chart-head, .chart-row { display: grid; grid-template-columns: 150px repeat(4, 1fr); }.chart-head { min-height: 38px; align-items: center; color: var(--sky-quiet); font: 400 8px var(--font-mono); }.chart-head b { font-weight: 400; text-align: right; }
.chart-row { min-height: 76px; border-top: 1px solid var(--sky-line); }.chart-row > strong { align-self: center; color: #c8d6dc; font-size: 10px; font-weight: 500; }.chart-row > strong small { display: block; margin-top: 5px; color: var(--sky-quiet); font-size: 8px; font-weight: 400; }
.track { position: relative; grid-column: 2 / 6; margin: 21px 0; background: rgba(139, 169, 184, .08); }.track > i { position: absolute; top: 0; bottom: 0; }
.darkness i { left: 9%; right: 11%; background: linear-gradient(90deg, #38465c, #10182b 18% 76%, #3e5166); }.clouds { display: grid; grid-template-columns: repeat(5, 1fr); gap: 2px; background: transparent; }.clouds i { position: static; background: rgba(171, 187, 194, .48); }.clouds i:nth-child(2), .clouds i:nth-child(3) { background: rgba(171, 187, 194, .74); }
.moon-track i { left: 0; width: 48%; background: #c8d4dc; }.milky i { left: 10%; width: 32%; background: #756b92; }.mars-track i { left: 9%; width: 28%; background: #b76d5b; }.saturn-track i { left: 26%; right: 9%; background: #ad9d70; }
.window-cursor { position: absolute; left: calc(150px + (100% - 150px) * .18); top: 38px; bottom: 0; width: 1px; background: var(--sky-amber); }.window-cursor i { position: absolute; left: -3px; top: -3px; width: 7px; height: 7px; border-radius: 50%; background: var(--sky-amber); }.window-cursor span { position: absolute; left: 7px; top: 6px; color: var(--sky-amber); font: 400 8px var(--font-mono); }
.window-legend { display: grid; grid-template-columns: repeat(3, 1fr); gap: 32px; padding: 34px 0; }.window-legend > div { display: grid; grid-template-columns: 8px 1fr; gap: 12px; color: var(--sky-quiet); font-size: 9px; line-height: 1.6; }.window-legend i { width: 6px; height: 6px; margin-top: 4px; border-radius: 50%; }.window-legend strong { display: block; color: #b8c7ce; font-weight: 500; }.available { background: #79bda7; }.uncertain { background: var(--sky-amber); }.blocked { background: #a86862; }

.target-feature { display: grid; grid-template-columns: minmax(320px, .9fr) 1.1fr; min-height: 420px; border-bottom: 1px solid var(--sky-line-strong); }
.target-visual { --target-color: #d98b72; position: relative; display: grid; place-items: center; overflow: hidden; background: radial-gradient(circle at center, color-mix(in srgb, var(--target-color) 15%, transparent), transparent 58%); }.target-visual > i { position: relative; z-index: 2; color: var(--target-color); font: normal 400 clamp(78px, 9vw, 132px)/1 var(--font-sans); text-shadow: 18px 24px 45px rgba(0, 0, 0, .48); }.target-visual > small { position: absolute; bottom: 42px; color: var(--sky-quiet); font: 400 8px var(--font-mono); letter-spacing: .24em; }.target-orbit { position: absolute; width: 72%; aspect-ratio: 1; border: 1px solid color-mix(in srgb, var(--target-color) 28%, transparent); border-radius: 50%; transform: rotate(-24deg) scaleY(.38); }.target-orbit::after { content: ''; position: absolute; right: 8%; top: 42%; width: 5px; height: 5px; border-radius: 50%; background: var(--target-color); }
.target-copy { display: flex; flex-direction: column; justify-content: center; padding: 50px clamp(30px, 5vw, 72px); }.target-copy h2 { margin: 0; font-size: clamp(48px, 5vw, 74px); font-weight: 400; letter-spacing: -.03em; }.target-copy > span { display: block; max-width: 590px; margin: 18px 0 28px; color: var(--sky-muted); font-size: 11px; line-height: 1.8; }.target-copy dl { display: grid; grid-template-columns: repeat(4, 1fr); margin: 0; border-top: 1px solid var(--sky-line); border-bottom: 1px solid var(--sky-line); }.target-copy dl > div { padding: 14px 0; }.target-copy dt { color: var(--sky-quiet); font-size: 8px; }.target-copy dd { margin: 6px 0 0; color: #cbd9df; font: 400 10px var(--font-mono); }.equipment { width: max-content; margin-top: 24px; color: var(--sky-amber); font-size: 10px; font-weight: 500; }
.target-directory { margin-top: 42px; }.directory-head { display: flex; justify-content: space-between; align-items: end; padding-bottom: 18px; }.directory-head h3 { margin: 0; font-size: 22px; font-weight: 400; }.directory-head label { display: grid; gap: 6px; color: var(--sky-quiet); font-size: 8px; }.directory-head input { width: 220px; padding: 8px 0; border: 0; border-bottom: 1px solid var(--sky-line-strong); border-radius: 0; background: transparent; color: var(--sky-ink); outline: 0; }.directory-head input:focus { border-bottom-color: var(--sky-amber); }
.target-directory > button { display: grid; grid-template-columns: 36px minmax(130px, .7fr) 1.2fr .8fr 1fr 60px; gap: 22px; align-items: center; width: 100%; min-height: 74px; padding: 0 8px; border: 0; border-top: 1px solid var(--sky-line); background: transparent; color: var(--sky-muted); text-align: left; transition: background .2s; }.target-directory > button:last-of-type { border-bottom: 1px solid var(--sky-line); }.target-directory > button:hover, .target-directory > button:focus-visible, .target-directory > button.active { background: rgba(201, 217, 227, .035); outline: 0; }.target-directory > button > i { font: normal 400 22px var(--font-sans); }.target-directory > button strong, .target-directory > button small { display: block; }.target-directory > button strong { color: var(--sky-ink); font-size: 12px; font-weight: 500; }.target-directory > button small { margin-top: 3px; color: var(--sky-quiet); font: 400 7px var(--font-mono); letter-spacing: .08em; }.target-directory > button > span:not(:nth-child(2)), .target-directory > button b { font: 400 9px var(--font-mono); }.target-directory > button b { color: var(--sky-amber); font-weight: 400; }.target-empty { padding: 40px 0; border-top: 1px solid var(--sky-line); color: var(--sky-quiet); font-size: 10px; }
.field-note { display: grid; grid-template-columns: 120px 1fr 1.5fr; gap: 36px; margin-top: 42px; padding: 26px 0; border-top: 1px solid var(--sky-line-strong); border-bottom: 1px solid var(--sky-line); }.field-note p { margin: 0; color: var(--sky-amber); font: 500 9px var(--font-mono); }.field-note strong { font-size: 12px; font-weight: 500; }.field-note span { color: var(--sky-muted); font-size: 10px; line-height: 1.7; }

.event-toolbar { display: flex; justify-content: space-between; align-items: end; padding: 12px 0 26px; border-bottom: 1px solid var(--sky-line-strong); }.event-toolbar h2 { margin: 0; font-size: 34px; font-weight: 400; }.event-toolbar > div:last-child { display: flex; gap: 4px; }.event-toolbar button { padding: 8px 12px; border: 0; border-radius: 2px; background: transparent; color: var(--sky-quiet); font-size: 9px; }.event-toolbar button:hover, .event-toolbar button:focus-visible, .event-toolbar button.active { background: rgba(201, 217, 227, .08); color: var(--sky-ink); outline: 0; }
.event-list article { display: grid; grid-template-columns: 100px minmax(300px, 1.45fr) minmax(260px, .9fr) 34px; gap: 32px; align-items: center; min-height: 150px; border-bottom: 1px solid var(--sky-line); }.event-list time strong, .event-list time span { display: block; }.event-list time strong { font: 400 42px/1 var(--font-mono); letter-spacing: -.05em; }.event-list time span { margin-top: 7px; color: var(--sky-amber); font: 400 8px var(--font-mono); letter-spacing: .12em; }.event-list article > div > span { color: var(--sky-amber); font: 400 8px var(--font-mono); }.event-list h3 { margin: 8px 0; font-size: 17px; font-weight: 500; }.event-list p { max-width: 620px; margin: 0; color: var(--sky-muted); font-size: 10px; line-height: 1.7; }.event-list dl { display: grid; gap: 12px; margin: 0; }.event-list dl > div { display: grid; grid-template-columns: 64px 1fr; gap: 12px; }.event-list dt { color: var(--sky-quiet); font-size: 8px; }.event-list dd { margin: 0; color: #b8c8cf; font: 400 9px var(--font-mono); }.event-list article > button { width: 30px; height: 30px; border: 1px solid var(--sky-line); border-radius: 50%; background: transparent; color: var(--sky-quiet); }.event-list article > button:hover, .event-list article > button:focus-visible { border-color: var(--sky-amber); color: var(--sky-amber); outline: 0; }
.event-boundary { display: grid; grid-template-columns: minmax(230px, .7fr) 1.5fr; gap: 60px; margin-top: 44px; padding: 28px 0; border-top: 1px solid var(--sky-line-strong); }.event-boundary h3 { margin: 0; font-size: 18px; font-weight: 400; }.event-boundary > p { max-width: 690px; margin: 0; color: var(--sky-muted); font-size: 10px; line-height: 1.8; }

@keyframes stage-enter { from { opacity: .01; transform: translateY(10px); filter: blur(3px); } to { opacity: 1; transform: none; filter: none; } }

@media (max-width: 1280px) {
  .sky-shell { grid-template-columns: 218px minmax(0, 1fr); }
  .sky-sidebar { padding-left: 18px; padding-right: 18px; }
  .sky-menu button::before { left: -18px; }
  .sky-page-header, .sky-page-stage { padding-left: 40px; padding-right: 40px; }
  .condition-ledger dl { grid-template-columns: repeat(3, 1fr); }
  .target-directory > button { grid-template-columns: 32px minmax(110px, .7fr) 1fr .7fr 1fr 50px; gap: 14px; }
}

@media (prefers-reduced-motion: reduce) {
  .sky-page-stage { animation: none; }
  .celestial-object { transition: none; }
}
</style>
