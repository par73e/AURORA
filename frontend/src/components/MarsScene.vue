<template>
  <section class="mars-section" aria-labelledby="mars-title">
    <div id="mars-scene" class="mars-scene-frame">
      <div ref="canvasHost" class="mars-scene-host" :class="{ revealed: sceneRevealed }" role="group" aria-label="火星三维视图，左上角可返回太阳系">
        <!-- 工具栏：与地球页同一套 scene-toolbar 结构（仅颜色走银灰覆盖） -->
        <div ref="sceneToolbarRef" class="scene-toolbar" :class="{ 'leaving-fade': leaving }" aria-label="场景图层">
          <span>图层</span>
          <label><input v-model="spacecraftEnabled" type="checkbox"><i />航天器</label>
          <label><input v-model="orbitsEnabled" type="checkbox"><i />轨道</label>
          <label><input v-model="sitesEnabled" type="checkbox"><i class="sites" />着陆点</label>
        </div>

        <!-- 轨道飞行器标签 -->
        <button
          v-for="label in craftLabels"
          v-show="label.visible && spacecraftEnabled"
          :key="label.id"
          class="craft-label"
          :class="{ selected: selectedCraft === label.id, 'stage-late': !elementsVisible, 'leaving-fade': leaving }"
          :style="craftLabelStyle(label)"
          :aria-label="`${craftById(label.id)?.nameZh}（${craftById(label.id)?.nameEn}）`"
          @click="selectedCraft = label.id"
          @pointerenter="hoveredCraftId = label.id"
          @pointerleave="hoveredCraftId = null"
        >
          <strong>{{ craftById(label.id)?.nameZh }}</strong>
          <small>{{ craftById(label.id)?.nameEn }}</small>
        </button>

        <!-- 着陆点标签：图标（宇航员/着陆器/月球车/样本）+ 地点名 + 任务名 -->
        <button
          v-for="label in siteLabels"
          v-show="label.visible && selectedSite === label.id"
          :key="label.id"
          class="craft-label site-label"
          :class="{ selected: selectedSite === label.id, 'leaving-fade': leaving }"
          :data-icon="siteById(label.id)?.icon ?? 'lander'"
          :style="siteLabelStyle(label)"
          :aria-label="`${siteById(label.id)?.siteName}（${siteById(label.id)?.missionName}）`"
          @click="selectSite(label.id)"
        >
          <span class="site-glyph" v-html="siteGlyph(siteById(label.id)?.icon ?? 'lander')" />
          <strong>{{ siteById(label.id)?.siteName }}</strong>
          <small>{{ siteById(label.id)?.missionName }}</small>
        </button>

        <!-- 选中着陆点的信息卡 -->
        <aside v-if="selectedSite && siteById(selectedSite)" class="site-panel" :class="{ visible: sceneRevealed }">
          <button class="site-panel-close" aria-label="关闭" @click="selectedSite = null">×</button>
          <div class="site-panel-head">
            <span class="site-glyph large" v-html="siteGlyph(siteById(selectedSite)?.icon ?? 'lander')" />
            <div>
              <h3>{{ siteById(selectedSite)?.siteName }}</h3>
              <p v-if="siteById(selectedSite)?.officialName">{{ siteById(selectedSite)?.officialName }}</p>
            </div>
          </div>
          <dl>
            <div><dt>任务</dt><dd>{{ siteById(selectedSite)?.missionName }}</dd></div>
            <div><dt>着陆日期</dt><dd>{{ siteById(selectedSite)?.landingDate }}</dd></div>
            <div><dt>区域</dt><dd>{{ siteById(selectedSite)?.region }}</dd></div>
            <div><dt>类别</dt><dd>{{ siteById(selectedSite)?.category === 'ROVER_LANDING' ? '巡视探测' : siteById(selectedSite)?.category === 'SAMPLE_RETURN' ? '采样返回' : '静态着陆' }}</dd></div>
            <div><dt>机构</dt><dd>{{ siteById(selectedSite)?.operatorName }}</dd></div>
            <div><dt>简介</dt><dd>{{ siteById(selectedSite)?.description }}</dd></div>
          </dl>
          <div class="site-hardware">
            <h4>遗留设施 / 硬件</h4>
            <ul><li v-for="(h, i) in siteById(selectedSite)?.hardware" :key="i">{{ h }}</li></ul>
          </div>
        </aside>

        <!-- 左下角读数：常驻火星（返回时随元素一起淡出） -->
        <div class="mars-readout" :class="{ 'leaving-fade': leaving }" aria-live="polite">
          <span>MARS ORBIT</span>
          <strong>火星</strong>
        </div>

        <!-- 右下角：纹理署名（SSS CC BY 4.0，与太阳系页同位置；返回时随元素一起淡出） -->
        <div class="mars-credits" :class="{ 'leaving-fade': leaving }" aria-hidden="true">Solar System Scope · CC BY 4.0</div>

        <!-- 右侧信息面板：与地球 context-panel 同结构，内容详尽 -->
        <aside v-if="selectedCraft" class="context-panel" aria-label="所选飞行器详情">
          <button class="panel-close" aria-label="关闭详情" @click="selectedCraft = null">关闭</button>
          <p class="context-type">{{ craftById(selectedCraft)?.type }}</p>
          <h2>{{ craftById(selectedCraft)?.nameZh }}</h2>
          <p class="context-subtitle">{{ craftById(selectedCraft)?.nameEn }}</p>
          <p class="context-description">{{ craftById(selectedCraft)?.description }}</p>
          <dl>
            <div><dt>运营方</dt><dd>{{ craftById(selectedCraft)?.operatorName }}</dd></div>
            <div><dt>发射地点</dt><dd>{{ craftById(selectedCraft)?.launchDate }} · {{ craftById(selectedCraft)?.launchSite }} · {{ craftById(selectedCraft)?.launchVehicle }}</dd></div>
            <div><dt>轨道倾角</dt><dd>{{ craftById(selectedCraft)?.displayInclination }}°</dd></div>
            <div><dt>偏心率</dt><dd>{{ craftById(selectedCraft)?.displayEccentricity }}</dd></div>
            <div><dt>轨道周期</dt><dd>{{ craftById(selectedCraft)?.displayPeriod }}</dd></div>
          </dl>
          <p class="source-caption"><template v-if="selectedCraftInfo?.snapshot">轨道历元 {{ formatEpochUTC(selectedCraftInfo.snapshot.epoch) }}<br></template>数据来源：{{ selectedCraftInfo?.sourceName }}</p>
        </aside>
      </div>
    </div>
  </section>

  <!-- 下方：火星航天器搜索板块（模仿地球的航天器工作区） -->
  <section id="mars-objects" class="content-section mars-objects-section">
    <div class="page-frame">
      <div class="section-heading">
        <div><p class="section-kicker">MARS SPACECRAFT</p><h2><i class="sec-num">Ⅰ</i>火星航天器</h2></div>
      </div>
      <div class="catalog-workspace">
        <div class="catalog-controls">
          <label class="search-field">
            <span>名称、英文或数据来源</span>
            <input v-model="craftQuery" type="search" placeholder="输入 MRO、MAVEN、天问一号…" spellcheck="false" />
          </label>
          <label><span>运营方</span><select v-model="craftOperatorFilter"><option value="all">全部运营方</option><option v-for="operator in craftOperators" :key="operator" :value="operator">{{ operator }}</option></select></label>
          <label><span>排序</span><select v-model="craftSort"><option value="name">名称</option><option value="type">类型</option><option value="operator">运营方</option></select></label>
        </div>
        <div class="catalog-meta">
          <span>共 {{ filteredCrafts.length }} 个对象</span>
          <span>标称轨道参数 · 非实时星历</span>
        </div>
        <div class="object-table" role="table" aria-label="火星航天器列表">
          <div class="object-table-head" role="row"><span>对象</span><span>轨道</span><span>数据来源</span></div>
          <button v-for="craft in filteredCrafts" :key="craft.id" class="object-row" role="row" @click="focusCraft(craft.id)">
            <span><strong>{{ craft.nameZh }}</strong><small>{{ craft.nameEn }}</small></span>
            <span>{{ craft.description }}</span>
            <span>{{ craft.sourceName }}</span>
          </button>
          <div v-if="!filteredCrafts.length" class="catalog-empty">没有符合条件的航天器。请修改搜索词。</div>
        </div>
      </div>
    </div>
  </section>

  <!-- 下方：火星着陆点板块（镜像航天器板块；点击 → 返回火星场景并放大居中该点） -->
  <section id="mars-sites" class="content-section mars-sites-section">
    <div class="page-frame">
      <div class="section-heading">
        <div><p class="section-kicker">MARS LANDING SITES</p><h2><i class="sec-num">Ⅲ</i>着陆点</h2></div>
      </div>
      <div class="catalog-workspace">
        <div class="catalog-controls">
          <label class="search-field">
            <span>地点、任务或机构</span>
            <input v-model="siteQuery" type="search" placeholder="输入 杰泽罗、Perseverance、天问一号…" spellcheck="false" />
          </label>
        </div>
        <div class="catalog-meta">
          <span>共 {{ filteredSites.length }} 个着陆点</span>
          <span>真实历史坐标 · 火星表面足迹</span>
        </div>
        <div class="object-table" role="table" aria-label="火星着陆点列表">
          <div class="object-table-head" role="row"><span>地点</span><span>任务</span><span>着陆日期</span></div>
          <button v-for="site in filteredSites" :key="site.id" class="object-row site-row" :data-icon="site.icon" role="row" @click="focusSite(site.id)">
            <span class="site-row-name">
              <span class="site-glyph" v-html="siteGlyph(site.icon)" />
              <span><strong>{{ site.siteName }}</strong><small>{{ site.officialName || site.region }}</small></span>
            </span>
            <span>{{ site.missionName }}<small>{{ site.operatorName }}</small></span>
            <span>{{ site.landingDate }}<small>{{ site.category === 'ROVER_LANDING' ? '巡视探测' : site.category === 'SAMPLE_RETURN' ? '采样返回' : '静态着陆' }}</small></span>
          </button>
          <div v-if="!filteredSites.length" class="catalog-empty">没有符合条件的着陆点。请修改搜索词。</div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as THREE from 'three'
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'
import { MARS_HD } from '../solar/data'
import { solarTexture } from '../solar/textures'
import type { MarsLandingSite, MarsSpacecraft } from '../types'
import { primaryOperator } from '../operators'

const props = defineProps<{ revealTick?: number; enterFromSolar?: boolean; leaving?: boolean; headerExpanded?: boolean }>()
const emit = defineEmits<{
  'blank-click': []
  /** 场景首帧贴图渲染完成（16k 解码 + GPU 上传后）——过渡遮罩等待此信号再揭示 */
  'textures-ready': []
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
const spacecraftEnabled = ref(true)
const orbitsEnabled = ref(true)
const sitesEnabled = ref(true)
const selectedCraft = ref<string | null>(null)
/** 悬停预览的飞行器（不运镜，仅驱动高亮：标记放大/实色 + 轨道线点亮；悬停优先于选中） */
const hoveredCraftId = ref<string | null>(null)
const craftQuery = ref('')
/** 航天器目录：运营方筛选 + 排序（与地球页一致） */
const craftOperatorFilter = ref('all')
const craftSort = ref('name')
const craftOperators = computed(() => [...new Set(crafts.value.map((c) => primaryOperator(c.operatorName)))].sort())
const siteQuery = ref('')
const craftLabels = ref<Array<{ id: string; x: number; y: number; visible: boolean }>>([])
const siteLabels = ref<Array<{ id: string; x: number; y: number; visible: boolean }>>([])
const landingSites = ref<MarsLandingSite[]>([])
const selectedSite = ref<string | null>(null)
const siteMarkers = new Map<string, THREE.Object3D>()
/** 拾取用：所有站点圆点（含拾取球） */
function siteMarkersArray() {
  return [...siteMarkers.values()]
}
/** 标签避让偏移缓存：每帧向目标偏移 lerp，避免重叠判定在阈值边缘抖动导致标签乱跳 */
const siteLabelOffsets = new Map<string, number>()

/** 入场渐亮：从太阳系进入（enterFromSolar）时等待 revealTick 递增；直接加载默认已亮。
 *  不能用 revealTick 判初始态——它只增不减，第二次进入时非 0 会误判为"直接加载" */
/** 工具栏被页头"推下/推回"：rAF 逐帧插值（CSS transition 被系统减弱动态效果禁用，JS 动画不受影响） */
const sceneToolbarRef = ref<HTMLElement | null>(null)
let moonToolbarShift = 0
let moonToolbarAnim: number | undefined
watch(
  () => props.headerExpanded,
  (expanded) => {
    if (moonToolbarAnim !== undefined) cancelAnimationFrame(moonToolbarAnim)
    const target = expanded ? 76 : 0
    const from = moonToolbarShift
    const start = performance.now()
    const tick = (now: number) => {
      const t = Math.min(1, (now - start) / 380)
      const eased = 1 - Math.pow(1 - t, 3)
      moonToolbarShift = from + (target - from) * eased
      if (sceneToolbarRef.value) {
        sceneToolbarRef.value.style.transform = moonToolbarShift > 0.5 ? `translateY(${moonToolbarShift.toFixed(2)}px)` : ''
      }
      moonToolbarAnim = t < 1 ? requestAnimationFrame(tick) : undefined
    }
    moonToolbarAnim = requestAnimationFrame(tick)
  },
)

const sceneRevealed = ref(!props.enterFromSolar)
/** 元素整体可见标记（DOM 标签用）：星球渐入完成后置 true；退出时立即 false */
const elementsVisible = ref(!props.enterFromSolar)
/** 统一元素淡入淡出进度（0..1）：1 = 全部元素可见；0 = 只剩裸火星。
 *  进入：星球渐入完成后 0→1（300ms）；退出：leaving 时 1→0（300ms）。
 *  直接加载/刷新默认全亮（无时间轴）。 */
let elementsFade = props.enterFromSolar ? 0 : 1
let elementsAnim: { from: number; to: number; startedAt: number; duration: number } | null = null
/** 火星自转：真实周期 24.6h，场景中做慢速可见旋转（非潮汐锁定，纯视觉） */
const MARS_SPIN_SPEED = THREE.MathUtils.degToRad(1.6) // ≈1.6°/s，约 3.8 分钟一圈（可见但不过度）
/** 元素弹出延迟 = 星球渐入（0.3s）+ 缓冲 */
const MARS_ELEMENTS_DELAY_MS = 900
/** 标记点距离补偿基准（默认相机距离 ≈ 17）：部分透视补偿（远小近大不过度） */
const MARS_MARKER_REF_DISTANCE = 17
/** 距离透明度（与地球统一）：远处（默认视角及更远）70% 半透明，放大到极限后渐变为实色 */
function distOpacity(d: number): number {
  return 0.7 + 0.3 * THREE.MathUtils.clamp((MARS_MARKER_REF_DISTANCE - d) / (MARS_MARKER_REF_DISTANCE - 2.85), 0, 1)
}
function animateElements(to: number, duration: number) {
  elementsAnim = { from: elementsFade, to, startedAt: performance.now(), duration }
}
/** 每帧推进并返回当前元素淡入淡出值（无动画时直接返回当前值） */
function updateElementsFade(): number {
  if (!elementsAnim) return elementsFade
  const t = Math.min(1, (performance.now() - elementsAnim.startedAt) / elementsAnim.duration)
  const eased = 1 - Math.pow(1 - t, 3)
  elementsFade = elementsAnim.from + (elementsAnim.to - elementsAnim.from) * eased
  if (t >= 1) elementsAnim = null
  return elementsFade
}
watch(
  () => props.revealTick,
  (tick) => {
    if (tick) sceneRevealed.value = true
  },
)
function startMarsSpin() {
  if (!swingPivot) return
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  swingPivot.rotation.y = reduced ? 0 : -THREE.MathUtils.degToRad(15)
}
/** 自转推进：匀速（自西向东 = 火星真实自转方向） */
function updateMarsSpin(delta: number) {
  if (!swingPivot) return
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  swingPivot.rotation.y += (reduced ? 0 : MARS_SPIN_SPEED) * delta
}
// 进入：裸火星先 0.3s 渐入（scene-host），随后所有元素（着陆点+飞行器+轨道+标签）一次性淡入
let elementsRevealTimer: number | undefined
watch(sceneRevealed, (revealed) => {
  if (!revealed || elementsVisible.value) return
  startMarsSpin()
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  elementsRevealTimer = window.setTimeout(() => {
    elementsRevealTimer = undefined
    elementsVisible.value = true
    animateElements(1, reduced ? 1 : 300)
  }, reduced ? 0 : MARS_ELEMENTS_DELAY_MS)
})

/** 火星飞行器列表（API 数据驱动，镜像地球 fetch overview 模式） */
const crafts = ref<MarsSpacecraft[]>([])
const craftById = (id: string) => crafts.value.find((c) => c.id === id)
/** 当前选中飞行器（模板多次取用） */
const selectedCraftInfo = computed(() => (selectedCraft.value ? craftById(selectedCraft.value) : undefined))

/** 轨道历元统一 UTC 显示（与探测器面板同步时间格式一致，避免本地/UTC 混用） */
function formatEpochUTC(iso?: string) {
  if (!iso) return ''
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())} ${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())} UTC`
}

const filteredCrafts = computed(() => {
  let items = crafts.value
  if (craftOperatorFilter.value !== 'all') {
    items = items.filter((c) => primaryOperator(c.operatorName) === craftOperatorFilter.value)
  }
  const q = craftQuery.value.trim().toLowerCase()
  if (q) {
    items = items.filter(
      (c) =>
        c.nameZh.toLowerCase().includes(q) ||
        c.nameEn.toLowerCase().includes(q) ||
        c.sourceName.toLowerCase().includes(q),
    )
  }
  const sorted = [...items]
  if (craftSort.value === 'operator') sorted.sort((a, b) => primaryOperator(a.operatorName).localeCompare(primaryOperator(b.operatorName), 'zh-CN'))
  else if (craftSort.value === 'type') sorted.sort((a, b) => (a.type ?? '').localeCompare(b.type ?? '', 'zh-CN'))
  else sorted.sort((a, b) => a.nameZh.localeCompare(b.nameZh, 'zh-CN'))
  return sorted
})

// 占位（craftById 已覆盖原 helper）

/** 点击搜索结果/场景标签：选中并聚焦（滚回主视图 → 飞行器居中 → 右侧面板） */
function focusCraft(id: string) {
  selectedCraft.value = id
  selectedSite.value = null // 选中互斥：聚焦飞行器时取消着陆点选中
  document.getElementById('mars-scene')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

/** 板块点击着陆点：返回火星场景 + 选中 + 镜头放大居中该点
 *  先平滑滚动回场景，滚动结束后再启动聚焦动画（并行会掉帧） */
function focusSite(id: string) {
  selectedSite.value = id
  document.getElementById('mars-scene')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  window.setTimeout(() => {
    if (selectedSite.value === id) startSiteFocus(id)
  }, 520)
}

/** 场景标签点击：切换选中（再次点击关闭），选中时镜头聚焦该点 */
function selectSite(id: string) {
  const next = selectedSite.value === id ? null : id
  selectedSite.value = next
  if (next) {
    selectedCraft.value = null // 选中互斥：聚焦着陆点时取消飞行器选中（否则注视点追飞行器）
    startSiteFocus(next)
  }
}


/** 通用运镜规划（学习地球 beginFocus）：任何目标（飞行器/着陆点）统一走此函数——
 *  相机方向球面插值（方向 lerp+normalize，不穿星球）+ 距离独立插值 + 注视火星中心。
 *  任意时刻被新目标覆盖时，fromPos=当前相机位置 → 平滑续接，无抽搐 */
function planFocusMotion(targetPos: THREE.Vector3, targetDistance: number) {
  if (!camera || !controls) return
  const distance = THREE.MathUtils.clamp(targetDistance, controls.minDistance, controls.maxDistance)
  focusAnimation = {
    fromPos: camera.position.clone(),
    toPos: targetPos.clone().normalize().multiplyScalar(distance),
    startedAt: performance.now(),
    duration: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 150 : 950,
  }
  controls.enabled = false
}

/** 飞行器聚焦：方向对准飞行器，观察距离取"当前距离与 8 的较小值"——稍作放大
 *  （初始 17 → 8，不至于太小；已放大时保持用户距离） */
function startCraftFocus(id: string) {
  const runtime = craftRuntimes.find((r) => r.spec.id === id)
  if (!runtime || !camera) return
  const world = runtime.dot.getWorldPosition(focusTmp).clone()
  planFocusMotion(world, Math.min(camera.position.length(), 8))
}

/** 着陆点聚焦：方向对准着陆点（观察距离 5.2——火面区域与周边地形整体可见） */
function startSiteFocus(id: string) {
  const marker = siteMarkers.get(id)
  if (!marker || !camera) return
  const world = marker.getWorldPosition(focusTmp).clone()
  planFocusMotion(world, 5.2)
}

watch(selectedCraft, (id) => {
  if (id) startCraftFocus(id)
})

// 着陆点聚焦改由 selectSite/focusSite 显式触发（watch 有 flush 时序与同 id 不触发的问题）

let renderer: THREE.WebGLRenderer | undefined
let scene: THREE.Scene | undefined
let camera: THREE.PerspectiveCamera | undefined
let controls: OrbitControls | undefined
let marsMesh: THREE.Mesh | undefined
/** 自转轴：marsMesh 挂其下，rotation.y 自西向东慢速推进（火星真实自转方向）；着陆点/轨迹随球面转 */
let swingPivot: THREE.Object3D | undefined
let marsMaterial: THREE.MeshStandardMaterial | undefined
let ambientLight: THREE.AmbientLight | undefined
let sunLight: THREE.DirectionalLight | undefined
let observationLight: THREE.DirectionalLight | undefined
let resizeObserver: ResizeObserver | undefined
let frameId = 0
/** 聚焦后用户开始拖拽：注视点快速滑回火星中心（拖拽恢复绕火星旋转） */
let dragResetTarget = false
/** 飞行器聚焦动画：相机移到飞行器外侧（火星在正后方作背景，居中且放大） */
let focusAnimation: {
  fromPos: THREE.Vector3
  toPos: THREE.Vector3
  startedAt: number
  duration: number
} | null = null
const focusTmp = new THREE.Vector3()
const focusTmp2 = new THREE.Vector3()
const focusTmp3 = new THREE.Vector3()
const focusTmp4 = new THREE.Vector3()
const raycaster = new THREE.Raycaster()
const pointerNDC = new THREE.Vector2()
/** 飞行器拾取球（不可见，挂在圆点上，扩大点击命中区域） */
const craftHitMeshes: THREE.Mesh[] = []

interface CraftRuntime {
  spec: MarsSpacecraft
  plane: THREE.Object3D
  dot: THREE.Object3D
  line: THREE.Line | null
  nu: number
}

const craftRuntimes: CraftRuntime[] = []

const MARS_FOV = 42
const DEG = Math.PI / 180

onMounted(() => {
  const host = canvasHost.value
  if (!host) return

  renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true, powerPreference: 'high-performance' })
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
  const initialWidth = host.clientWidth || window.innerWidth
  const initialHeight = host.clientHeight || window.innerHeight
  renderer.setSize(initialWidth, initialHeight)
  renderer.outputColorSpace = THREE.SRGBColorSpace
  renderer.setClearColor(0x010307, 1)
  host.appendChild(renderer.domElement)

  scene = new THREE.Scene()
  camera = new THREE.PerspectiveCamera(MARS_FOV, initialWidth / initialHeight, 0.1, 2000)
  // 初始视角：距火星中心 17（视半径 ~10.1°）——火星比月球大，轨道也更宽，视野相应拉远
  camera.position.set(0, 1.8, 17)

  resizeObserver = new ResizeObserver(() => {
    const width = host.clientWidth
    const height = host.clientHeight
    if (width === 0 || height === 0 || !renderer || !camera) return
    camera.aspect = width / height
    camera.updateProjectionMatrix()
    renderer.setSize(width, height)
  })
  resizeObserver.observe(host)
  renderer.domElement.addEventListener('wheel', onSceneWheel, { passive: false })
  renderer.domElement.addEventListener('pointerdown', onPointerDown)
  renderer.domElement.addEventListener('pointerup', onPointerUp)
  renderer.domElement.addEventListener('pointermove', onPointerMove)
  renderer.domElement.addEventListener('pointerleave', onPointerLeave)

  // 无自动自转：拖拽旋转；滚轮由 onSceneWheel 按区域接管（火星上缩放、边缘滚动页面）
  controls = new OrbitControls(camera, renderer.domElement)
  controls.enableDamping = true
  controls.dampingFactor = 0.06
  controls.enablePan = false
  controls.enableZoom = false
  controls.addEventListener('start', () => {
    dragResetTarget = true
  })
  controls.minDistance = 3.25 // 拉近极限：距火星表面（半径 3.0）仅 0.25，可贴面观察纹理
  controls.maxDistance = 60

  // 火星本体：8k 贴图 + PBR 材质（保留质感，同地球模式）
  const texture = solarTexture(MARS_HD.textureUrl, () => emitTexturesReady())
  texture.colorSpace = THREE.SRGBColorSpace
  texture.anisotropy = 16
  marsMaterial = new THREE.MeshStandardMaterial({ map: texture, roughness: 0.95, metalness: 0.02 })
  // 细分 256 段：8k 贴图在 96 段球体上贴面时三角形过粗导致模糊，256 段显著提升贴面清晰度
  marsMesh = new THREE.Mesh(new THREE.SphereGeometry(3.0, 256, 256), marsMaterial)
  // 初始朝向：lon 0° 子午线朝向相机（任意初始姿态，着陆点作为子节点随球面转）
  marsMesh.quaternion.setFromUnitVectors(new THREE.Vector3(1, 0, 0), camera.position.clone().normalize())
  // 自转轴：火星自西向东慢速自转（真实周期 24.6h，场景做慢速可见旋转）
  swingPivot = new THREE.Object3D()
  swingPivot.name = 'mars-swing-pivot'
  swingPivot.add(marsMesh)
  scene.add(swingPivot)

  // 光照（镜像地球）：固定环境光 + 太阳方向光 + 跟随相机的观测光（360° 全亮，无晨昏线）
  ambientLight = new THREE.AmbientLight(0x3a2a22, 0.8)
  scene.add(ambientLight)
  observationLight = new THREE.DirectionalLight(0xfff3dd, 3.1)
  observationLight.position.copy(camera.position)
  scene.add(observationLight)
  sunLight = new THREE.DirectionalLight(0xfff3dd, 0)
  sunLight.position.set(-6, 4, 8)
  scene.add(sunLight)

  // 星空粒子球（镜像地球 OrbitScene）：3000 颗、壳层 60–150、银灰主题色
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
  const starPoints = new THREE.Points(starGeometry, new THREE.PointsMaterial({ color: 0xc7d1db, size: 0.15, transparent: true, opacity: 0.75 }))
  starPoints.name = 'background-stars'
  scene.add(camera)
  camera.add(starPoints)

  // 轨道飞行器数据来自 /api/v1/mars/spacecraft（数据库 → Go → API → 前端），
  // 挂载后异步拉取并按数据构建轨道/圆点（镜像地球的数据链路）
  fetch('/api/v1/mars/spacecraft')
    .then((res) => res.json())
    .then((data: { spacecraft: MarsSpacecraft[] }) => {
      crafts.value = data.spacecraft ?? []
      if (!scene) return
      for (const spec of crafts.value) buildCraft(spec)
    })
    .catch((error) => {
      console.error('加载火星飞行器数据失败:', error)
    })
  fetch('/api/v1/mars/landing-sites')
    .then((res) => res.json())
    .then((data: { landingSites: MarsLandingSite[] }) => {
      landingSites.value = data.landingSites ?? []
      buildSiteMarkers()
    })
    .catch((error) => {
      console.error('加载火星着陆点数据失败:', error)
    })

  renderer.render(scene, camera)

  let lastTime = performance.now()
  const animate = () => {
    frameId = requestAnimationFrame(animate)
    if (!renderer || !scene || !camera) return
    const now = performance.now()
    const delta = Math.min((now - lastTime) / 1000, 0.05)
    lastTime = now

    // 航天器公转（仅绕火轨道）
    for (const runtime of craftRuntimes) {
      if (runtime.spec.kind !== 'orbital') continue
      const sn = runtime.spec.snapshot ?? null
      // 快照存在时用真实公转周期（JPL 日同步，如 MRO 约 112 分钟）；
      // 无快照时回退静态轨道周期（迁移 034 起已改为真实周期）
      const periodSec = sn ? sn.periodSeconds : runtime.spec.periodSeconds
      runtime.nu += (Math.PI * 2 / periodSec) * delta
      const a = exaggeratedA(sn ? sn.aKm * MARS_SCENE_SCALE : runtime.spec.orbitA)
      const e = sn ? sn.eccentricity : runtime.spec.orbitE
      const r = (a * (1 - e * e)) / (1 + e * Math.cos(runtime.nu))
      runtime.dot.position.set(r * Math.cos(runtime.nu), r * Math.sin(runtime.nu), 0)
    }
    // 观测光跟随相机：明暗边界始终落在球体轮廓之外（关闭晨昏线时 360° 全亮）
    if (observationLight && camera) observationLight.position.copy(camera.position)

    // 聚焦：相机与注视点双缓动（点击瞬间飞行器居中、火星背景放大）。
    // 动画完成后不再跟随；用户拖拽时注视点滑回火星中心——拖拽始终绕火星旋转
    if (controls && camera) {
      if (dragResetTarget) {
        controls.target.lerp(new THREE.Vector3(0, 0, 0), 0.12)
        if (controls.target.lengthSq() < 0.002) {
          controls.target.set(0, 0, 0)
          dragResetTarget = false
        }
      }
      if (focusAnimation) {
        const t = Math.min(1, (now - focusAnimation.startedAt) / focusAnimation.duration)
        const eased = 1 - Math.pow(1 - t, 3)
        // 方向球面插值（lerp + normalize = 短弧滑动，不穿星球）+ 距离独立插值（无突兀缩放）
        const dir = focusTmp2
          .copy(focusAnimation.fromPos)
          .normalize()
          .lerp(focusTmp3.copy(focusAnimation.toPos).normalize(), eased)
          .normalize()
        const dist = THREE.MathUtils.lerp(focusAnimation.fromPos.length(), focusAnimation.toPos.length(), eased)
        camera.position.copy(dir.multiplyScalar(dist))
        camera.lookAt(0, 0, 0) // 注视火星中心（与地球 lookAt 中心一致，永远稳定）
        controls.target.multiplyScalar(1 - eased) // 注视点平滑衰减回火星中心
        if (t >= 1) {
          focusAnimation = null
          controls.target.set(0, 0, 0) // 结束后绕火星中心旋转（与地球一致）
          controls.enabled = true
          controls.update()
        }
      }
    }

    // 动态拖动灵敏度（与地球一致）：近处降敏、远处提速；默认视角 13.6 处 ≈ 0.41（与地球默认手感一致）
    if (controls && camera) {
      const t = THREE.MathUtils.clamp((camera.position.length() - controls.minDistance) / 25, 0, 1)
      controls.rotateSpeed = 0.2 + t * 0.5
    }
    // 统一元素淡入淡出：进入时星球渐入完成后一次性浮现；退出时全部一起消失（只留裸火星）
    const elementsFadeNow = updateElementsFade()
    updateMarsSpin(delta)
    for (const runtime of craftRuntimes) {
      const dotChild = runtime.dot.children[0] as THREE.Mesh | undefined
      const dotMat = dotChild?.material as THREE.MeshBasicMaterial | undefined
      // 高亮：悬停 > 选中（与地球页语义一致）；仅点亮不运镜
      const active = runtime.spec.id === (hoveredCraftId.value ?? selectedCraft.value)
      if (dotMat) dotMat.opacity = elementsFadeNow
      if (runtime.line) (runtime.line.material as THREE.LineBasicMaterial).opacity = (active ? 0.95 : 0.55) * elementsFadeNow
      // 部分透视补偿（远小近大、不过度）：scale = (d/基准)^0.6；高亮时放大 35%
      // 距离透明度：远处 70% 半透明、放大后实色（与地球统一）；隐藏期不渲染（visible 兜底）
      const d = runtime.dot.getWorldPosition(focusTmp).distanceTo(camera.position)
      runtime.dot.scale.setScalar(Math.pow(d / MARS_MARKER_REF_DISTANCE, 0.6) * (active ? 1.35 : 1))
      if (dotMat) dotMat.opacity = (active ? 1 : distOpacity(d)) * elementsFadeNow
      runtime.dot.visible = spacecraftEnabled.value && elementsFadeNow > 0.001
      if (runtime.line) runtime.line.visible = orbitsEnabled.value && elementsFadeNow > 0.001
    }
    // 着陆点虚线轨迹随元素整体淡入淡出
    for (const child of marsMesh?.children ?? []) {
      if (child.name && child.name.startsWith('track:')) {
        const trackMat = (child as THREE.Line).material as THREE.LineDashedMaterial | undefined
        if (trackMat) trackMat.opacity = 0.85 * elementsFadeNow
        child.visible = sitesEnabled.value && elementsFadeNow > 0.001
      }
    }
    // （已移除）近距锐化切换：minFilter + needsUpdate 会触发 16k 纹理整体重传，
    // 放大跨越阈值时产生明显卡顿——收益远小于代价
    controls?.update()
    updateLabels()
    updateSiteMarkerProximity()
    renderer.render(scene, camera)
  }
  animate()
})

/** 场景单位 ↔ 真实尺寸：火星半径 3.0（场景）↔ 3389.5 km（真实） */
const MARS_SCENE_SCALE = 3.0 / 3389.5
/** 轨道高度夸张（与地球 ALTITUDE_EXAGGERATION=3.2 同思路）：超出火面的部分放大 1.5 倍——
 *  MRO 真实轨道仅高出火面 ~8% 半径，视觉上贴面飞行；MAVEN/天问一号轨道本身达 2–3 倍
 *  半径，夸张取 1.5 兼顾可辨识度与取景（初始视距 17） */
const MARS_ALTITUDE_EXAGGERATION = 1.5
/** 轨道半径（场景单位，含高度夸张）：火心 + 超出火面部分 × 夸张系数 */
function exaggeratedA(a: number) {
  return 3.0 + Math.max(0, a - 3.0) * MARS_ALTITUDE_EXAGGERATION
}

/** 平近点角 → 真近点角（Kepler 方程，牛顿迭代） */
function keplerToTrueAnomaly(M: number, e: number): number {
  let E = M
  for (let k = 0; k < 8; k += 1) E = E - (E - e * Math.sin(E) - M) / (1 - e * Math.cos(E))
  return 2 * Math.atan2(Math.sqrt(1 + e) * Math.sin(E / 2), Math.sqrt(1 - e) * Math.cos(E / 2))
}

/** 按 API 数据构建单个飞行器（轨道平面/轨道线/运动点/拾取球）
 *  优先使用 JPL Horizons 日同步快照（真实形状 + 真实相位），无快照回退静态参数 */
function buildCraft(spec: MarsSpacecraft) {
  if (!scene) return
  const sn = spec.snapshot ?? null
  // 真实轨道根数（快照优先）：半长轴 km → 场景单位 → 轨道高度夸张（贴面飞行观感修正）
  const a = exaggeratedA(sn ? sn.aKm * MARS_SCENE_SCALE : spec.orbitA)
  const e = sn ? sn.eccentricity : spec.orbitE
  const inc = sn ? sn.inclinationDeg : spec.inclinationDeg
  const raan = sn ? sn.raanDeg : spec.raanDeg
  const argp = sn ? sn.argPeriapsisDeg : spec.argPeriapsisDeg

  const plane = new THREE.Object3D()
  if (spec.kind === 'orbital') {
    plane.rotation.order = 'YXZ'
    plane.rotation.y = raan * DEG
    plane.rotation.x = inc * DEG
  }

  const dot = new THREE.Object3D()
  if (spec.kind === 'orbital') dot.rotation.z = argp * DEG
  plane.add(dot)
  const dotMesh = new THREE.Mesh(
    new THREE.SphereGeometry(spec.kind === 'stationary' ? 0.045 : 0.04, 16, 16),
    // transparent 必须为 true：否则分阶段揭示的 opacity=0 被忽略，圆点提前出现
    new THREE.MeshBasicMaterial({ color: spec.kind === 'stationary' ? 0xffd9a0 : 0xffb27d, transparent: true }),
  )
  dot.add(dotMesh)
  const hitSphere = new THREE.Mesh(
    new THREE.SphereGeometry(0.22, 8, 8),
    new THREE.MeshBasicMaterial({ transparent: true, opacity: 0, depthWrite: false }),
  )
  hitSphere.userData.craftId = spec.id
  dot.add(hitSphere)
  craftHitMeshes.push(hitSphere)

  let line: THREE.Line | null = null
  let initialNu = 0
  if (spec.kind === 'orbital') {
    const linePoints: THREE.Vector3[] = []
    for (let i = 0; i <= 180; i += 1) {
      const nu = (i / 180) * Math.PI * 2
      const r = (a * (1 - e * e)) / (1 + e * Math.cos(nu))
      linePoints.push(new THREE.Vector3(r * Math.cos(nu), r * Math.sin(nu), 0))
    }
    line = new THREE.Line(
      new THREE.BufferGeometry().setFromPoints(linePoints),
      new THREE.LineBasicMaterial({ color: 0x9a5a3c, transparent: true, opacity: 0.55 }),
    )
    line.rotation.z = argp * DEG
    plane.add(line)
    // 真实初始相位：M = M0 + n·Δt（历元传播到当前时刻）→ 真近点角
    if (sn) {
      const elapsedSec = (Date.now() - Date.parse(sn.epoch)) / 1000
      const n = (Math.PI * 2) / sn.periodSeconds
      const M = (sn.meanAnomalyDeg * DEG + n * elapsedSec) % (Math.PI * 2)
      initialNu = keplerToTrueAnomaly(M, e)
    }
    dot.position.set(a * (1 - e), 0, 0)
  } else {
    // 定点：固定在火星外侧（不参与公转）
    dot.position.set(spec.stationaryOffset[0], spec.stationaryOffset[1], spec.stationaryOffset[2])
  }

  scene.add(plane)
  craftRuntimes.push({ spec, plane, dot, line, nu: initialNu })
}

/** 经纬度 → 球面坐标（与地球页 latLonToVector 同公式） */
function sitePosition(latitude: number, longitude: number, radius: number) {
  const lat = latitude * DEG
  const lon = longitude * DEG
  return new THREE.Vector3(
    radius * Math.cos(lat) * Math.cos(lon),
    radius * Math.sin(lat),
    -radius * Math.cos(lat) * Math.sin(lon),
  )
}

/** 着陆点标记：小圆点贴在火面（marsMesh 子节点，天然随球面），不参与任何旋转 */
function buildSiteMarkers() {
  if (!scene || !marsMesh) return
  for (const site of landingSites.value) {
    if (siteMarkers.has(site.id)) continue
    // 图标类型着色：astronaut 金 / rover 橙 / sample 青 / lander 银
    const color = site.icon === 'astronaut' ? 0xffcf8f : site.icon === 'rover' ? 0xffb27d : site.icon === 'sample' ? 0x8fd6c2 : 0xcfd8e2
    const marker = new THREE.Mesh(
      new THREE.SphereGeometry(0.03, 12, 12),
      new THREE.MeshBasicMaterial({ color, transparent: true, opacity: elementsFade }),
    )
    // 球心落在火面半径上（3.0）：球体一半嵌进表面（被火星深度遮挡）、一半露出——
    // "镶嵌"在火面上的观感；露出半球深度 < 表面 → 通过深度测试，无 z-fighting
    marker.position.copy(sitePosition(site.latitude, site.longitude, 3.0))
    marker.userData = { kind: 'landing-site', siteId: site.id }
    marsMesh.add(marker)
    siteMarkers.set(site.id, marker)
    // 拾取球：扩大点击命中区域（点击圆点 → 选中并聚焦，标签随选中出现）
    const siteHit = new THREE.Mesh(
      new THREE.SphereGeometry(0.12, 8, 8),
      new THREE.MeshBasicMaterial({ transparent: true, opacity: 0, depthWrite: false }),
    )
    siteHit.userData.siteId = site.id
    marker.add(siteHit)

    // 火星车行驶轨迹：虚线折线（示意图，数据存库可替换真实遥测）
    if (site.track && site.track.length >= 2) {
      const points = site.track.map(([lat, lon]) => sitePosition(lat, lon, 3.0 * 1.008))
      const trackLine = new THREE.Line(
        new THREE.BufferGeometry().setFromPoints(points),
        new THREE.LineDashedMaterial({ color, dashSize: 0.055, gapSize: 0.05, transparent: true, opacity: elementsFade * 0.85 }),
      )
      trackLine.computeLineDistances()
      trackLine.name = `track:${site.id}`
      marsMesh.add(trackLine)
    }
  }
}

function siteById(id: string) {
  return landingSites.value.find((site) => site.id === id)
}

const filteredSites = computed(() => {
  const q = siteQuery.value.trim().toLowerCase()
  if (!q) return landingSites.value
  return landingSites.value.filter(
    (site) =>
      site.siteName.toLowerCase().includes(q) ||
      site.missionName.toLowerCase().includes(q) ||
      site.officialName.toLowerCase().includes(q) ||
      site.operatorName.toLowerCase().includes(q) ||
      site.region.toLowerCase().includes(q),
  )
})

/** 着陆点标签样式：右侧偏移，垂直对齐圆点 */
function siteLabelStyle(label: { id: string; x: number; y: number }) {
  return { transform: `translate(calc(${label.x}px + 10px), ${label.y - 14}px)` }
}

/** 着陆点圆点随镜头距离淡出：远视正常 → 凑近半透明并缩小 → 贴面消失（不遮挡月面观察）
 *  距离 > 4.5 完全显示；4.5 → 3.2 线性淡出 + 缩至 45%；< 3.2 完全消失 */
function updateSiteMarkerProximity() {
  if (!camera || !sitesEnabled.value) return
  for (const site of landingSites.value) {
    const marker = siteMarkers.get(site.id)
    if (!marker) continue
    const world = marker.getWorldPosition(focusTmp)
    const d = world.distanceTo(camera.position)
    const material = (marker as THREE.Mesh).material as THREE.MeshBasicMaterial
    // 距离透明度（远处 70% 半透明、放大后实色）× 统一元素淡入淡出（进入一次性浮现 / 退出一次性消失）
    material.opacity = distOpacity(d) * elementsFade
    // 部分透视补偿（远小近大、不过度）：k=0.6；去掉原"贴面微缩"（近处缩小的观感反物理）
    marker.scale.setScalar(Math.pow(d / MARS_MARKER_REF_DISTANCE, 0.6))
    marker.visible = sitesEnabled.value && elementsFade > 0.001
  }
}

/** 站点图标（内联 SVG）：宇航员 / 着陆器 / 月球车 / 样本返回舱 */
function siteGlyph(icon: 'astronaut' | 'lander' | 'rover' | 'sample') {
  const stroke = 'currentColor'
  switch (icon) {
    case 'astronaut':
      return `<svg viewBox="0 0 12 12" width="12" height="12" fill="none" stroke="${stroke}" stroke-width="1.1" stroke-linecap="round"><circle cx="6" cy="3.6" r="2.3"/><path d="M2.6 11c0-2.1 1.5-3.4 3.4-3.4s3.4 1.3 3.4 3.4"/></svg>`
    case 'rover':
      return `<svg viewBox="0 0 12 12" width="12" height="12" fill="none" stroke="${stroke}" stroke-width="1.1" stroke-linecap="round"><rect x="2.8" y="4.4" width="6.4" height="3" rx="0.6"/><path d="M3.6 2.6h2.2M6.4 2.6h2"/><circle cx="4.2" cy="8.4" r="1.1"/><circle cx="7.8" cy="8.4" r="1.1"/></svg>`
    case 'sample':
      return `<svg viewBox="0 0 12 12" width="12" height="12" fill="none" stroke="${stroke}" stroke-width="1.1" stroke-linecap="round"><path d="M5 1.6h2l1.4 2v6a1 1 0 0 1-1 1H4.6a1 1 0 0 1-1-1v-6z"/><path d="M3.6 5.4h4.8"/></svg>`
    default:
      return `<svg viewBox="0 0 12 12" width="12" height="12" fill="none" stroke="${stroke}" stroke-width="1.1" stroke-linecap="round" stroke-linejoin="round"><path d="M6 1.6 9.4 10.4H2.6z"/><path d="M6 5.4v5"/></svg>`
  }
}

// 火星无晨昏线开关：观测光(相机方向) 3.1 + 太阳光 0 → 360° 全亮

// 航天器开关：显示/隐藏飞行器圆点（标签由 v-show 联动）；关闭时清悬停避免轨道线残留点亮
watch(spacecraftEnabled, (enabled) => {
  for (const runtime of craftRuntimes) runtime.dot.visible = enabled
  if (!enabled) hoveredCraftId.value = null
})
// 轨道开关：显示/隐藏轨道线
watch(orbitsEnabled, (enabled) => {
  for (const runtime of craftRuntimes) if (runtime.line) runtime.line.visible = enabled
})
// 返回太阳系：全部多余元素 300ms 一次性淡出（统一 elementsFade），只留裸火星——
// 随后由 App 遮罩完成星球渐暗切页；离开被中止（hash 守卫失败）时 leaving 回 false → 恢复显示
watch(
  () => props.leaving,
  (leaving) => {
    const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    if (!leaving) {
      if (elementsFade < 1) {
        elementsVisible.value = true
        animateElements(1, reduced ? 1 : 300)
      }
      return
    }
    if (elementsRevealTimer !== undefined) {
      clearTimeout(elementsRevealTimer) // 防止入场延迟定时器在退出后把元素拉回
      elementsRevealTimer = undefined
    }
    elementsVisible.value = false
    animateElements(0, reduced ? 1 : 300)
    selectedSite.value = null
    selectedCraft.value = null
  },
)

// 着陆点开关：同步控制月面圆点 + 虚线轨迹（不只是标签）
watch(sitesEnabled, (enabled) => {
  for (const site of landingSites.value) {
    const marker = siteMarkers.get(site.id)
    if (marker) marker.visible = enabled
  }
  for (const child of marsMesh?.children ?? []) {
    if (child.name && child.name.startsWith('track:')) child.visible = enabled
  }
})

const marsPointerStart = new THREE.Vector2()

/** 按下瞬间：在火星表面 → 立即收起页头并清除选中（与地球一致） */
function onPointerDown(event: PointerEvent) {
  marsPointerStart.set(event.clientX, event.clientY)
  hoveredCraftId.value = null // 按下即清悬停：拖拽期间不残留点亮；点击选中由 selectedCraft 继续驱动高亮
  if (isNearMars(event.clientX, event.clientY)) {
    selectedCraft.value = null
    emit('blank-click')
  }
}

/** 悬停预览：不拖拽时射线拾取飞行器 → 点亮（标记放大+实色、轨道线变亮）；航天器图层关闭不触发 */
function onPointerMove(event: PointerEvent) {
  if (!renderer || !camera || !spacecraftEnabled.value) {
    hoveredCraftId.value = null // 图层关闭时清悬停：避免轨道线残留点亮
    return
  }
  if (event.buttons !== 0) {
    hoveredCraftId.value = null
    return
  }
  const bounds = renderer.domElement.getBoundingClientRect()
  pointerNDC.set(((event.clientX - bounds.left) / bounds.width) * 2 - 1, -((event.clientY - bounds.top) / bounds.height) * 2 + 1)
  raycaster.setFromCamera(pointerNDC, camera)
  const hits = raycaster.intersectObjects(craftHitMeshes)
  const hit = hits.find((h) => {
    // Raycaster 不检查 visible 且不继承父级：命中球本身 visible 恒 true，需查父级（dot）
    if (h.object.parent && !h.object.parent.visible) return false
    const world = h.object.getWorldPosition(focusTmp)
    return !isCraftOccluded(world)
  })
  hoveredCraftId.value = hit?.object.userData.craftId ? String(hit.object.userData.craftId) : null
}

/** 指针离开画布：清除悬停 */
function onPointerLeave() {
  hoveredCraftId.value = null
}

/** 松开：未拖拽（点按）→ 射线拾取飞行器（点击圆点选中）或清除；四周拖拽保持展开 */
function onPointerUp(event: PointerEvent) {
  if (!renderer || !camera) return
  if (marsPointerStart.distanceTo(new THREE.Vector2(event.clientX, event.clientY)) > 5) return
  const bounds = renderer.domElement.getBoundingClientRect()
  pointerNDC.set(((event.clientX - bounds.left) / bounds.width) * 2 - 1, -((event.clientY - bounds.top) / bounds.height) * 2 + 1)
  raycaster.setFromCamera(pointerNDC, camera)
  const hits = raycaster.intersectObjects(craftHitMeshes)
  const hit = hits.find((h) => {
    const world = h.object.getWorldPosition(focusTmp)
    return !isCraftOccluded(world)
  })
  if (hit && hit.object.userData.craftId) {
    selectedCraft.value = hit.object.userData.craftId
    return
  }
  // 站点圆点拾取：点击圆点 → 选中并聚焦（标签随选中出现）
  const siteHits = raycaster.intersectObjects(siteMarkersArray())
  const siteHit = siteHits.find((h) => {
    const world = h.object.getWorldPosition(focusTmp)
    return !isCraftOccluded(world)
  })
  if (siteHit && siteHit.object.userData.siteId) {
    selectSite(siteHit.object.userData.siteId)
    return
  }
  selectedCraft.value = null
  emit('blank-click')
}

/** 鼠标是否在火星投影范围内（镜像地球 isNearEarth） */
function isNearMars(clientX: number, clientY: number) {
  if (!renderer || !camera) return false
  const bounds = renderer.domElement.getBoundingClientRect()
  const projectedCenter = new THREE.Vector3(0, 0, 0).project(camera)
  const cameraRight = new THREE.Vector3(1, 0, 0)
    .applyQuaternion(camera.quaternion)
    .multiplyScalar(3.0 * 1.08)
    .project(camera)
  const centerX = bounds.left + (projectedCenter.x * 0.5 + 0.5) * bounds.width
  const centerY = bounds.top + (-projectedCenter.y * 0.5 + 0.5) * bounds.height
  const radius = Math.abs(cameraRight.x - projectedCenter.x) * bounds.width * 0.5
  return Math.hypot(clientX - centerX, clientY - centerY) <= radius * 1.12
}

/** 飞行器是否被火星遮挡：视线段（相机→飞行器）与火星球体（半径 3.0）相交 */
function isCraftOccluded(world: THREE.Vector3) {
  if (!camera) return false
  const dir = world.clone().sub(camera.position)
  const distance = dir.length()
  dir.normalize()
  // 最近点必须在视线段之内（否则是飞行器后面的火星，不算遮挡）
  const t = -camera.position.dot(dir)
  if (t <= 0 || t >= distance) return false
  const closest = camera.position.clone().addScaledVector(dir, t)
  return closest.length() < 3.0
}

/** 滚轮：在火星上 → 缩放火星；在边缘区域 → 交给页面滚动（与地球一致） */
function onSceneWheel(event: WheelEvent) {
  if (!camera || !controls || !isNearMars(event.clientX, event.clientY)) return
  event.preventDefault()
  const normalizedDelta = event.deltaMode === WheelEvent.DOM_DELTA_LINE ? event.deltaY * 16 : event.deltaY
  const nextDistance = THREE.MathUtils.clamp(
    camera.position.length() * Math.exp(normalizedDelta * 0.0012),
    controls.minDistance,
    controls.maxDistance,
  )
  camera.position.setLength(nextDistance)
  controls.update()
}

function updateLabels() {
  const host = canvasHost.value
  if (!host || !camera) return
  const width = host.clientWidth
  const height = host.clientHeight
  if (width === 0 || height === 0) return
  const halfFovTan = Math.tan((MARS_FOV / 2) * DEG)
  const tmp = new THREE.Vector3()
  const next: Array<{ id: string; x: number; y: number; visible: boolean }> = []
  for (const runtime of craftRuntimes) {
    const world = runtime.dot.getWorldPosition(tmp)
    // 先算遮挡（世界坐标），再投影（project 会原地改写向量）
    const occluded = isCraftOccluded(world)
    const p = world.project(camera)
    // 标签与圆点一体：同一遮挡判定，背面一起隐藏、正面一起出现；
    // 隐藏期（元素未揭示）同样不渲染（visible 兜底，与 craft 循环一致）
    runtime.dot.visible = spacecraftEnabled.value && !occluded && elementsFade > 0.001
    next.push({
      id: runtime.spec.id,
      x: (p.x * 0.5 + 0.5) * width,
      y: (-p.y * 0.5 + 0.5) * height,
      visible: p.z > -1 && p.z < 1 && !occluded,
    })
  }
  craftLabels.value = next

  // 着陆点标签：背面隐藏（圆点本体由材质深度测试自然遮挡）
  const siteNext: Array<{ id: string; x: number; y: number; visible: boolean }> = []
  const siteTmp = new THREE.Vector3()
  for (const site of landingSites.value) {
    const marker = siteMarkers.get(site.id)
    if (!marker) continue
    const world = marker.getWorldPosition(siteTmp)
    const cp = world.clone().project(camera)
    const occluded = isCraftOccluded(world) // 同款背面判定：法线朝向相机才显示
    siteNext.push({
      id: site.id,
      x: (cp.x * 0.5 + 0.5) * width,
      y: (-cp.y * 0.5 + 0.5) * height,
      visible: cp.z > -1 && cp.z < 1 && !occluded,
    })
  }
  // 标签避让：屏幕距离过近的可见标签对，后者向下错开一档（链式处理多重重叠）。
  // 真实站点可能相距仅 180m（阿波罗 12 与勘测者 3），投影后完全重叠——错开保证可读
  // 阈值按实际标签盒取（向右展开约 170px 宽、两行文字约 36px 高）
  const SITE_LABEL_W = 170
  const SITE_LABEL_H = 36
  const targetOffsets = new Map<string, number>()
  for (let i = 0; i < siteNext.length; i += 1) {
    const a = siteNext[i]
    if (!a.visible) continue
    for (let j = i + 1; j < siteNext.length; j += 1) {
      const b = siteNext[j]
      if (!b.visible) continue
      if (Math.abs(a.x - b.x) < SITE_LABEL_W && Math.abs(a.y - b.y) < SITE_LABEL_H) {
        targetOffsets.set(b.id, (targetOffsets.get(b.id) ?? 0) + SITE_LABEL_H)
      }
    }
  }
  // 平滑过渡：偏移向目标 lerp，转动时标签缓慢归位而非跳变
  for (const label of siteNext) {
    const target = targetOffsets.get(label.id) ?? 0
    const current = siteLabelOffsets.get(label.id) ?? 0
    const next = current + (target - current) * 0.25
    siteLabelOffsets.set(label.id, next)
    if (Math.abs(next) > 1) label.y += next
  }
  siteLabels.value = siteNext
}

function craftLabelStyle(label: { id: string; x: number; y: number }) {
  // 标签垂直中心与圆点对齐（标签高约 28px，上移一半）
  return { transform: `translate(calc(${label.x}px + 10px), ${label.y - 14}px)` }
}

onBeforeUnmount(() => {
  if (elementsRevealTimer !== undefined) clearTimeout(elementsRevealTimer)
  cancelAnimationFrame(frameId)
  resizeObserver?.disconnect()
  renderer?.domElement.removeEventListener('wheel', onSceneWheel)
  renderer?.domElement.removeEventListener('pointerdown', onPointerDown)
  renderer?.domElement.removeEventListener('pointerup', onPointerUp)
  renderer?.domElement.removeEventListener('pointermove', onPointerMove)
  renderer?.domElement.removeEventListener('pointerleave', onPointerLeave)
  controls?.dispose()
  marsMaterial?.dispose()
  marsMesh?.geometry.dispose()
  renderer?.dispose()
})
</script>

<style scoped>
/* 暖红主题：火星界面统一用陶土暖色（区别于地球浅蓝、月球银灰） */
.mars-section {
  --mars-accent: #e0a878;
  --mars-accent-dim: rgba(224, 168, 120, 0.4);
  --mars-line: rgba(224, 168, 120, 0.22);
  --mars-text: #ecd9c8;
  --mars-quiet: #a89078;
  position: relative;
  height: 150dvh;
  min-height: 990px;
  /* 暖红星野：火星界面统一暖色星点 */
  background:
    radial-gradient(1.2px 1.2px at 12% 22%, rgba(230, 178, 130, .4), transparent 100%),
    radial-gradient(1px 1px at 23% 64%, rgba(230, 178, 130, .3), transparent 100%),
    radial-gradient(.8px .8px at 31% 38%, rgba(230, 178, 130, .25), transparent 100%),
    radial-gradient(1.4px 1.4px at 41% 82%, rgba(230, 178, 130, .36), transparent 100%),
    radial-gradient(1px 1px at 55% 15%, rgba(230, 178, 130, .28), transparent 100%),
    radial-gradient(.9px .9px at 62% 48%, rgba(230, 178, 130, .24), transparent 100%),
    radial-gradient(1.3px 1.3px at 71% 74%, rgba(230, 178, 130, .32), transparent 100%),
    radial-gradient(1px 1px at 79% 29%, rgba(230, 178, 130, .26), transparent 100%),
    radial-gradient(.8px .8px at 88% 58%, rgba(230, 178, 130, .28), transparent 100%),
    radial-gradient(1.1px 1.1px at 94% 12%, rgba(230, 178, 130, .32), transparent 100%),
    radial-gradient(1px 1px at 7% 86%, rgba(230, 178, 130, .26), transparent 100%),
    radial-gradient(.9px .9px at 49% 92%, rgba(230, 178, 130, .24), transparent 100%),
    radial-gradient(1.2px 1.2px at 66% 4%, rgba(230, 178, 130, .3), transparent 100%),
    radial-gradient(ellipse at 50% 50%, #120a06 0%, #050302 100%);
}
/* 航天器/着陆点板块 UI 全暖红（覆盖全局浅蓝主题色） */
.mars-objects-section .catalog-workspace,
.mars-sites-section .catalog-workspace { background: #16100a; }
.mars-objects-section .section-kicker,
.mars-sites-section .section-kicker { color: #d0a080; }
.mars-objects-section .sec-num,
.mars-sites-section .sec-num { color: #c09070; }
.mars-objects-section .catalog-controls label > span,
.mars-sites-section .catalog-controls label > span { color: #b09880; }
.mars-objects-section .catalog-controls input,
.mars-sites-section .catalog-controls input {
  border-color: rgba(224, 168, 120, .25);
  background: #120c07;
  color: #ecd9c8;
}
.mars-objects-section .catalog-controls select,
.mars-sites-section .catalog-controls select {
  border-color: rgba(224, 168, 120, .25);
  background-color: #120c07;
  color: #ecd9c8;
}
.mars-objects-section .catalog-controls input:focus,
.mars-sites-section .catalog-controls input:focus {
  border-color: rgba(224, 168, 120, .6);
  box-shadow: 0 0 0 3px rgba(224, 168, 120, .08);
}
.mars-objects-section .catalog-meta,
.mars-sites-section .catalog-meta {
  border-top-color: rgba(224, 168, 120, .15);
  color: #a89078;
}
.mars-objects-section .object-table-head,
.mars-objects-section .object-row,
.mars-sites-section .object-table-head,
.mars-sites-section .object-row {
  grid-template-columns: 150px minmax(260px, 1.6fr) minmax(180px, 1fr);
}
.mars-objects-section .object-table-head,
.mars-sites-section .object-table-head {
  border-top-color: rgba(224, 168, 120, .15);
  border-bottom-color: rgba(224, 168, 120, .15);
  color: #a89078;
}
.mars-objects-section .object-row,
.mars-sites-section .object-row {
  border-bottom-color: rgba(224, 168, 120, .12);
  color: #c09070;
}
.mars-objects-section .object-row:hover,
.mars-objects-section .object-row:focus-visible,
.mars-sites-section .object-row:hover,
.mars-sites-section .object-row:focus-visible {
  background: rgba(224, 168, 120, .06);
  color: #ecd9c8;
}
.mars-objects-section .object-row small,
.mars-sites-section .object-row small { color: #a89078; }
.mars-objects-section .catalog-empty,
.mars-sites-section .catalog-empty { color: #a89078; }

/* 粘性场景区：首屏 100dvh，下滑进入航天器板块 */
.mars-scene-frame {
  position: sticky;
  top: 0;
  height: 100dvh;
  min-height: 660px;
  overflow: hidden;
}
.mars-scene-host {
  position: absolute;
  inset: 0;
  opacity: 0;
  transition: opacity 0.3s ease;
  cursor: grab;
}
.mars-scene-host.revealed {
  opacity: 1;
}
.mars-scene-host canvas { display: block; }

/* 航天器标签（银灰，位于小点右侧，连接线水平指向左侧的圆点） */
.craft-label {
  position: absolute;
  left: 0;
  top: 0;
  z-index: 3;
  display: grid;
  justify-items: center;
  gap: 2px;
  padding: 3px 8px;
  border: 1px solid var(--mars-line);
  border-radius: 3px;
  background: rgba(6, 10, 14, .72);
  color: var(--mars-text);
  cursor: pointer;
  transition: border-color .2s, color .2s, background .2s;
  backdrop-filter: blur(8px);
}
.craft-label strong {
  font-size: 10px;
  font-weight: 500;
  white-space: nowrap;
  text-shadow: 0 1px 4px rgba(0, 0, 0, .9);
}
.craft-label small {
  color: var(--mars-quiet);
  font: 400 7px var(--font-mono);
  letter-spacing: .12em;
}
.craft-label::before {
  content: '';
  position: absolute;
  right: 100%;
  top: 50%;
  width: 8px;
  height: 1px;
  background: var(--mars-accent-dim);
  transform: translateY(-50%);
}
.craft-label:hover,
.craft-label.selected {
  border-color: rgba(224, 168, 120, .65);
  background: rgba(16, 22, 28, .85);
}

/* 着陆点板块行：图标 + 名称两行 */
.site-row { grid-template-columns: minmax(260px, 1.4fr) minmax(220px, 1fr) 150px !important; }
.site-row .site-glyph { color: #d8b090; flex-shrink: 0; }
.site-row[data-icon='astronaut'] .site-glyph { color: #ffcf8f; }
.site-row[data-icon='rover'] .site-glyph { color: #ffb27d; }
.site-row[data-icon='sample'] .site-glyph { color: #8fd6c2; }
.site-row-name { display: flex; align-items: center; gap: 10px; }

/* 分阶段揭示：阶段 3 前的标签淡入（透明度过渡，不抢占点击） */
.craft-label.stage-late { opacity: 0 !important; pointer-events: none; }
.craft-label { transition: opacity .45s ease; }
/* 返回渐隐：标签 300ms 淡出 */
.craft-label.leaving-fade { opacity: 0 !important; pointer-events: none; }

/* 返回渐隐：工具栏与标签同节奏淡出（只留裸火星，随后由遮罩完成球体渐暗） */
.scene-toolbar.leaving-fade { opacity: 0; pointer-events: none; transition: opacity .3s ease; }
/* 返回渐隐：左下角读数/右下角署名随元素一起淡出 */
.mars-readout.leaving-fade,
.mars-credits.leaving-fade { opacity: 0; transition: opacity .3s ease; }

/* 着陆点标签：图标着色 + 银灰主题 */
.site-label { gap: 5px !important; }
.site-label .site-glyph { display: inline-flex; flex-shrink: 0; }
.site-label strong { color: #ecd9c8 !important; }
.site-label small { color: var(--mars-quiet) !important; }
.site-label[class*='selected'] .site-glyph { color: #ffd9a0 !important; }
/* 图标类型颜色：astronaut 金 / rover 橙 / sample 青 / lander 银 */
.site-label .site-glyph { color: #d8b090; }
.site-label[data-icon='astronaut'] .site-glyph { color: #ffcf8f; }
.site-label[data-icon='rover'] .site-glyph { color: #ffb27d; }
.site-label[data-icon='sample'] .site-glyph { color: #8fd6c2; }

/* ===== 火星右侧信息面板（重构：统一低对比深色底，无黑边/白边，去掉嵌套底块） ===== */
.site-panel,
.mars-scene-host .context-panel {
  position: absolute;
  z-index: 8;
  top: 18px;
  right: 32px;
  width: clamp(300px, 22vw, 380px);
  max-height: calc(100% - 36px - var(--header-overlay-offset));
  overflow-y: auto;
  padding: 22px 24px;
  border: 1px solid rgba(224, 168, 120, .1); /* 四边统一弱描边 */
  border-radius: 10px;
  /* 低对比深色底：右上略深 → 左下微亮，幅度极小，面板边缘与背景无色差 */
  background: linear-gradient(200deg, rgba(9, 13, 18, .97) 0%, rgba(20, 27, 34, .95) 100%);
  /* 外投影浮起 + 顶部极微弱光（无底部高光/黑线） */
  box-shadow: 0 24px 70px rgba(0, 0, 0, .5), inset 0 1px 0 rgba(255, 255, 255, .04);
  backdrop-filter: blur(18px);
  color: var(--mars-text);
  font-size: 12px;
  /* 顶部菜单栏展开时整体下移（与地球页信息卡同机制） */
  transform: translateY(var(--header-overlay-offset, 0px));
  transition: transform .38s cubic-bezier(.22, 1, .36, 1);
}
/* 设施区：只用分隔线区分，去掉嵌套底块（原浅灰底块 + 圆角 + 边框） */
.site-panel .site-hardware {
  padding: 12px 0 0;
  border: 0;
  border-top: 1px solid var(--mars-line);
  border-radius: 0;
  background: none;
}
.site-panel-head h3 { text-shadow: 0 1px 8px rgba(0, 0, 0, .6); }
.site-panel .site-panel-close {
  position: absolute;
  top: 10px;
  right: 14px;
  border: 0;
  background: none;
  color: var(--mars-quiet);
  font-size: 18px;
  line-height: 1;
  cursor: pointer;
}
.site-panel .site-panel-close:hover { color: var(--mars-text); }
.site-panel-head { display: flex; gap: 12px; align-items: center; padding-bottom: 14px; border-bottom: 1px solid var(--mars-line); }
.site-panel-head .site-glyph.large { color: #ffd9a0; }
.site-panel-head h3 { margin: 0; font-size: 15px; font-weight: 500; color: #e8edf2; }
.site-panel-head p { margin: 3px 0 0; color: var(--mars-quiet); font: 400 10px var(--font-mono); letter-spacing: .06em; }
.site-panel dl { display: grid; gap: 8px; padding: 14px 0; margin: 0; }
.site-panel dl > div { display: grid; grid-template-columns: 64px 1fr; gap: 10px; }
.site-panel dt { color: var(--mars-quiet); font-size: 11px; }
.site-panel dd { margin: 0; color: var(--mars-text); font-size: 11px; line-height: 1.5; }
.site-hardware h4 { margin: 0 0 8px; color: var(--mars-quiet); font: 500 9px var(--font-mono); letter-spacing: .12em; }
.site-hardware ul { margin: 0; padding-left: 16px; display: grid; gap: 5px; }
.site-hardware li { color: var(--mars-text); font-size: 11px; line-height: 1.5; }

/* 右下角署名（银灰，与太阳系页同位置同风格） */
.mars-credits {
  position: absolute;
  z-index: 3;
  right: 34px;
  bottom: 30px;
  color: var(--mars-quiet);
  font: 400 7px var(--font-mono);
  letter-spacing: .08em;
  text-align: right;
  pointer-events: none;
}

/* 读数区（银灰） */
.mars-readout {
  position: absolute;
  z-index: 4;
  left: 32px;
  bottom: 28px;
  color: var(--mars-text);
}
.mars-readout > span {
  color: var(--mars-quiet);
  font: 500 8px var(--font-mono);
  letter-spacing: .15em;
}
.mars-readout strong {
  display: block;
  margin-top: 5px;
  font-size: 17px;
  font-weight: 500;
}
.mars-readout p {
  max-width: 320px;
  margin: 6px 0 0;
  color: var(--mars-quiet);
  font-size: 10px;
  line-height: 1.7;
}
</style>
