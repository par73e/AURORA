<template>
  <section class="mars-section" aria-labelledby="mars-title">
    <div id="mars-scene" class="mars-scene-frame">
      <div ref="canvasHost" class="mars-scene-host" :class="{ revealed: sceneRevealed, 'leaving-body': leaving }" role="group" aria-label="火星三维视图，左上角可返回太阳系">
        <!-- 工具栏：与地球页同一套 scene-toolbar 结构（仅颜色走陶土红覆盖） -->
        <div ref="sceneToolbarRef" class="scene-toolbar" :class="{ 'leaving-fade': leaving }" aria-label="场景图层">
          <span>图层</span>
          <label><input v-model="spacecraftEnabled" type="checkbox"><i />飞行器</label>
          <label><input v-model="orbitsEnabled" type="checkbox"><i />轨道</label>
          <label><input v-model="sitesEnabled" type="checkbox"><i class="sites" />着陆点</label>
          <label><input v-model="terminatorEnabled" type="checkbox"><i class="terminator" />晨昏线</label>
        </div>

        <div v-if="dataLoading || dataError" class="scene-data-state" :class="{ error: !!dataError }" role="status">
          <span>{{ dataError || '正在读取火星飞行器与着陆点数据' }}</span>
          <button v-if="dataError" type="button" @click="loadSceneData">重新加载</button>
        </div>

        <!-- 轨道飞行器标签 -->
        <MissionSceneLabel
          v-for="label in craftLabels"
          v-show="label.visible && spacecraftEnabled"
          :key="label.id"
          class="craft-label"
          :class="{ selected: selectedCraft === label.id, 'stage-late': !elementsVisible, 'leaving-fade': leaving }"
          :style="craftLabelStyle(label)"
          kind="spacecraft"
          :name-zh="craftBilingual.get(label.id)?.primary ?? ''"
          :name-en="craftBilingual.get(label.id)?.secondary"
          :selected="selectedCraft === label.id"
          :mode="label.mode"
          :side="label.side"
          :cluster-count="label.clusterCount"
          :cluster-items="label.memberIds.map((id) => ({ id, name: craftBilingual.get(id)?.primary ?? id }))"
          :aria-label="label.mode === 'cluster' ? `${label.clusterCount} 个相近飞行器` : `${craftBilingual.get(label.id)?.primary}${craftBilingual.get(label.id)?.secondary ? `（${craftBilingual.get(label.id)?.secondary}）` : ''}`"
          @click="selectedCraft = label.id"
          @select-member="selectedCraft = $event"
          @pointerenter="hoveredCraftId = label.mode === 'cluster' ? null : label.id"
          @pointerleave="hoveredCraftId = null"
        />

        <!-- 着陆点标签：图标（宇航员/着陆器/月球车/样本）+ 地点名 + 任务名 -->
        <MissionSceneLabel
          v-for="label in siteLabels"
          v-show="label.visible && sitesEnabled"
          :key="label.id"
          class="craft-label site-label"
          :class="{ selected: selectedSite === label.id, 'leaving-fade': leaving }"
          :data-icon="siteById(label.id)?.icon ?? 'lander'"
          :style="siteLabelStyle(label)"
          kind="surface"
          :name-zh="siteBilingual.get(label.id)?.primary ?? ''"
          :name-en="siteBilingual.get(label.id)?.secondary"
          :selected="selectedSite === label.id"
          :mode="label.mode"
          :side="label.side"
          :cluster-count="label.clusterCount"
          :cluster-items="label.memberIds.map((id) => ({ id, name: siteBilingual.get(id)?.primary ?? id }))"
          :icon-html="siteGlyph(siteById(label.id)?.icon ?? 'lander')"
          :aria-label="`${siteBilingual.get(label.id)?.primary}${siteBilingual.get(label.id)?.secondary ? `（${siteBilingual.get(label.id)?.secondary}）` : ''}`"
          @click="selectSite(label.id)"
          @select-member="selectSite($event)"
        />

        <!-- 选中着陆点的信息卡 -->
        <MissionDetailPanel
          v-if="selectedSiteDetail"
          :class="{ 'leaving-fade': leaving }"
          :detail="selectedSiteDetail"
          :style="headerExpanded ? { '--header-overlay-offset': '76px' } : undefined"
          @close="selectedSite = null"
        />

        <!-- 左下角读数：常驻火星（返回时随元素一起淡出） -->
        <div class="mars-readout" :class="{ 'leaving-fade': leaving }" aria-live="polite">
          <span>MARS ORBIT</span>
          <strong>火星</strong>
        </div>

        <!-- 右下角：纹理署名（SSS CC BY 4.0，与太阳系页同位置；返回时随元素一起淡出） -->
        <div class="mars-credits" :class="{ 'leaving-fade': leaving }" aria-hidden="true">Solar System Scope · CC BY 4.0</div>

        <!-- 右侧信息面板：与地球 context-panel 同结构，内容详尽 -->
        <MissionDetailPanel
          v-if="selectedCraftDetail"
          :class="{ 'leaving-fade': leaving }"
          :detail="selectedCraftDetail"
          :style="headerExpanded ? { '--header-overlay-offset': '76px' } : undefined"
          @close="selectedCraft = null"
        />
      </div>
    </div>
  </section>

  <!-- 下方：火星档案板块（与金星/土星/木星页同款：真实静态数据，延续可滚动框架） -->
  <section id="mars-profile" class="content-section mars-profile-section">
    <div class="page-frame">
      <div class="section-heading">
        <div><p class="section-kicker">MARS PROFILE</p><h2><i class="sec-num">Ⅱ</i>火星档案</h2></div>
      </div>
      <div class="profile-grid">
        <dl class="profile-table">
          <div><dt>直径</dt><dd>{{ marsProfile.diameter }}</dd></div>
          <div><dt>距日</dt><dd>{{ marsProfile.distance }}</dd></div>
          <div><dt>自转周期</dt><dd>{{ marsProfile.rotation }}</dd></div>
          <div><dt>太阳日</dt><dd>{{ marsProfile.solarDay }}</dd></div>
          <div><dt>公转周期</dt><dd>{{ marsProfile.orbit }}</dd></div>
          <div><dt>轴倾角</dt><dd>{{ marsProfile.axialTilt }}</dd></div>
          <div><dt>卫星</dt><dd>{{ marsProfile.moons }}</dd></div>
          <div><dt>环</dt><dd>{{ marsProfile.rings }}</dd></div>
          <div><dt>成分</dt><dd>{{ marsProfile.composition }}</dd></div>
          <div class="profile-intro-row"><dt>简介</dt><dd>{{ marsProfile.description }}</dd></div>
        </dl>
      </div>
    </div>
  </section>

  <!-- 下方：火星航天器搜索板块（模仿地球的航天器工作区） -->
  <section id="mars-objects" class="content-section mars-objects-section">
    <div class="page-frame">
      <div class="section-heading">
        <div><p class="section-kicker">MARS SPACECRAFT</p><h2><i class="sec-num">Ⅲ</i>飞行器</h2><p class="section-sub">飞行器 · 轨道与位置用于交互示意；精确状态以数据来源为准。</p></div>
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
          <span>{{ filteredCrafts.length }} 个飞行器</span>
          <span>轨道与位置用于交互示意</span>
        </div>
        <div class="object-table" role="table" aria-label="火星飞行器列表">
          <div class="object-table-head" role="row"><span>对象</span><span>运营方</span><span>类型</span></div>
          <button v-for="craft in pagedCrafts" :key="craft.id" class="object-row" role="row" @click="focusCraft(craft.id)">
            <span><strong>{{ craftBilingual.get(craft.id)?.primary }}</strong><small v-if="craftBilingual.get(craft.id)?.secondary">（{{ craftBilingual.get(craft.id)?.secondary }}）</small></span>
            <span>{{ craft.operatorName }}</span>
            <span>{{ craft.type }}</span>
          </button>
          <div v-if="!filteredCrafts.length" class="catalog-empty">没有符合条件的飞行器。请修改搜索词。</div>
        </div>
        <div class="pagination-space"><span>第 {{ craftPage }} / {{ craftPageCount }} 页 · {{ filteredCrafts.length }} 个飞行器</span><div><button :disabled="craftPage <= 1" @click="craftGotoPage(-1)">上一页</button><button :disabled="craftPage >= craftPageCount" @click="craftGotoPage(1)">下一页</button></div></div>
      </div>
    </div>
  </section>

  <!-- 下方：火星着陆点板块（镜像航天器板块；点击 → 返回火星场景并放大居中该点） -->
  <section id="mars-sites" class="content-section mars-sites-section">
    <div class="page-frame">
      <div class="section-heading">
        <div><p class="section-kicker">MARS LANDING SITES</p><h2><i class="sec-num">Ⅳ</i>着陆点</h2><p class="section-sub">着陆点 · 圆点标示任务位置；坐标精度以数据来源为准。</p></div>
      </div>
      <div class="catalog-workspace">
        <div class="catalog-controls">
          <label class="search-field">
            <span>地点、任务或机构</span>
            <input v-model="siteQuery" type="search" placeholder="输入 杰泽罗、Perseverance、天问一号…" spellcheck="false" />
          </label>
        </div>
        <div class="catalog-meta">
          <span>{{ filteredSites.length }} 个着陆点</span>
          <span>圆点标示任务位置 · 坐标精度以数据来源为准</span>
        </div>
        <div class="object-table" role="table" aria-label="火星着陆点列表">
          <div class="object-table-head" role="row"><span>地点</span><span>任务</span><span>着陆日期</span></div>
          <button v-for="site in pagedSites" :key="site.id" class="object-row site-row" :data-icon="site.icon" role="row" @click="focusSite(site.id)">
            <span class="site-row-name">
              <span class="site-glyph" v-html="siteGlyph(site.icon)" />
              <span><strong>{{ siteBilingual.get(site.id)?.primary }}</strong><small v-if="siteBilingual.get(site.id)?.secondary">（{{ siteBilingual.get(site.id)?.secondary }}）</small></span>
            </span>
            <span>{{ site.missionName }}<small>{{ site.operatorName }}</small></span>
            <span>{{ site.landingDate }}<small>{{ site.category === 'ROVER_LANDING' ? '巡视探测' : site.category === 'SAMPLE_RETURN' ? '采样返回' : site.category === 'AERIAL' ? '动力飞行' : '静态着陆' }}</small></span>
          </button>
          <div v-if="!filteredSites.length" class="catalog-empty">没有符合条件的着陆点。请修改搜索词。</div>
        </div>
        <div class="pagination-space"><span>第 {{ sitePage }} / {{ sitePageCount }} 页 · {{ filteredSites.length }} 个着陆点</span><div><button :disabled="sitePage <= 1" @click="siteGotoPage(-1)">上一页</button><button :disabled="sitePage >= sitePageCount" @click="siteGotoPage(1)">下一页</button></div></div>
      </div>
    </div>
  </section>

  <!-- 页脚：数据源同步时间（与地球页脚一致；右对齐） -->
  <footer class="mars-page-footer">
    <div class="page-frame footer-inner">
      <div><strong>AURORA / MARS</strong></div>
      <div class="source-list"><span><i :class="{ healthy: !!syncedAt }" />{{ orbitDataCaption }}<template v-if="syncedAt"> · {{ formatEpochUTC(syncedAt) }}</template></span></div>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as THREE from 'three'
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'
import { MARS_HD } from '../solar/data'
import { solarTexture } from '../solar/textures'
import { MARS_PAGE } from '../planetPages'
import type { MarsLandingSite, MarsSpacecraft } from '../types'
import { fetchMarsLandingSites, fetchMarsSpacecraft } from '../api'
import { primaryOperator } from '../operators'
import { bilingualName } from '../bilingual'
import { CATALOG_PAGE_SIZE } from '../catalog'
import { usePlanetSceneData } from '../composables/usePlanetSceneData'
import MissionDetailPanel from './MissionDetailPanel.vue'
import MissionSceneLabel from './MissionSceneLabel.vue'
import type { MissionDetail } from '../missionPresentation'
import { spacecraftFields, spacecraftFocusDistance, surfaceMissionFields } from '../missionPresentation'
import type { SceneAnnotationLayout, SurfaceAnnotationLayout } from '../surfaceAnnotations'
import { layoutSceneAnnotations, sceneAnnotationStyle, projectedSphereRadiusPx, orbitMarkerRadiusPx, sceneMarkerWorldRadius, surfaceMarkerRadiusPx, surfaceMarkerWorldRadius } from '../surfaceAnnotations'

const marsProfile = MARS_PAGE.profile

const props = defineProps<{ spacecraftVisible?: boolean; revealTick?: number; enterFromSolar?: boolean; leaving?: boolean; headerExpanded?: boolean }>()
const emit = defineEmits<{
  'blank-click': []
  'update:spacecraft-visible': [visible: boolean]
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
const spacecraftEnabled = computed({
  get: () => props.spacecraftVisible ?? true,
  set: (visible: boolean) => emit('update:spacecraft-visible', visible),
})
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
const craftLabels = ref<SceneAnnotationLayout[]>([])
const siteLabels = ref<SurfaceAnnotationLayout[]>([])
const landingSites = ref<MarsLandingSite[]>([])
const selectedSite = ref<string | null>(null)
const siteMarkers = new Map<string, THREE.Object3D>()
/** 拾取用：所有站点圆点（含拾取球） */
function siteMarkersArray() {
  return [...siteMarkers.values()]
}

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
/** 火星入场自转（自西向东 = 火星真实自转方向，绕自转轴）：
 *  转速 14.4°/s（≈1.45s 转正，与地球入场时长相当），渐入开始时从 -18° 偏角匀速转，
 *  角度剩减速位移时线性匀减速，终点 0°（初始姿态），全程线性无突快突慢 */
const MARS_SPIN_SPEED = THREE.MathUtils.degToRad(14.4) // ≈14.4°/s，自西向东
const MARS_SPIN_DECEL_MS = 400 // 匀减速段
const MARS_SPIN_DECEL_SWEEP = (MARS_SPIN_SPEED * MARS_SPIN_DECEL_MS) / 2000 // ≈2.88°（匀减速位移）
const MARS_SPIN_OFFSET = -THREE.MathUtils.degToRad(18) // 预设偏角（渐入前偏 18°，转正）
let marsSpinPhase: 'spin' | 'stop' | 'done' = 'done'
let marsSpinStartAt = 0
let marsSpinStopAt = 0
let marsSpinStopFrom = 0
/** 元素弹出延迟 = 旋转停稳（≈1.45s）+ 50ms 缓冲 */
const MARS_ELEMENTS_DELAY_MS = 1500
/** 标记点距离补偿基准（默认相机距离 ≈ 9）：部分透视补偿（远小近大不过度） */
const MARS_MARKER_REF_DISTANCE = 9
/** 距离透明度（与地球统一）：远处（默认视角及更远）70% 半透明，放大到极限后渐变为实色 */
function distOpacity(d: number): number {
  return 0.7 + 0.3 * THREE.MathUtils.clamp((MARS_MARKER_REF_DISTANCE - d) / (MARS_MARKER_REF_DISTANCE - 2.2), 0, 1)
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
  if (!swingPivot || marsSpinPhase !== 'done') return
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  marsSpinPhase = reduced ? 'done' : 'spin'
  marsSpinStartAt = performance.now()
  swingPivot.rotation.y = MARS_SPIN_OFFSET
}
/** 入场自转推进：匀速（自西向东）→ 角度剩减速位移时线性匀减速 → 终点 0°（初始姿态） */
function updateMarsSpin(now: number) {
  if (!swingPivot || marsSpinPhase === 'done') return
  if (marsSpinPhase === 'spin') {
    const t = Math.max(0, (now - marsSpinStartAt) / 1000)
    swingPivot.rotation.y = MARS_SPIN_OFFSET + MARS_SPIN_SPEED * t
    if (swingPivot.rotation.y >= -MARS_SPIN_DECEL_SWEEP) {
      marsSpinPhase = 'stop'
      marsSpinStopAt = now
      // 对齐精确阈值：终点精确落在 0°（初始姿态），不受帧偏差/后台标签页帧迟到影响
      marsSpinStopFrom = -MARS_SPIN_DECEL_SWEEP
    }
  } else {
    const t = Math.min(1, (now - marsSpinStopAt) / MARS_SPIN_DECEL_MS)
    swingPivot.rotation.y = marsSpinStopFrom + MARS_SPIN_SPEED * (MARS_SPIN_DECEL_MS / 1000) * (t - (t * t) / 2)
    if (t >= 1) marsSpinPhase = 'done'
  }
}
// 进入：裸火星先 0.3s 渐入（scene-host），随后所有元素（着陆点+飞行器+轨道+标签）一次性淡入
let elementsRevealTimer: number | undefined
let focusTimer: number | undefined
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
const terminatorEnabled = ref(false)
/** 火星数据与月球走同一原子提交规则，避免网络抖动时场景出现半套实体。 */
const { crafts, syncedAt, dataLoading, dataError, loadSceneData, abortSceneData } = usePlanetSceneData<MarsSpacecraft, MarsLandingSite>({
  loadSpacecraft: fetchMarsSpacecraft,
  loadLandingSites: fetchMarsLandingSites,
  canCommit: () => Boolean(scene),
  onLoaded: (nextCrafts, nextSites) => {
    landingSites.value = nextSites
    for (const spec of nextCrafts) buildCraft(spec)
    buildSiteMarkers()
  },
  unavailableLabel: '数据暂时不可用',
})
const craftById = (id: string) => crafts.value.find((c) => c.id === id)
/** 当前选中飞行器（模板多次取用） */
const selectedCraftInfo = computed(() => (selectedCraft.value ? craftById(selectedCraft.value) : undefined))
const selectedCraftDetail = computed<MissionDetail | null>(() => {
  const craft = selectedCraftInfo.value
  if (!craft) return null
  const name = bilingualName(craft.nameZh, craft.nameEn)
  const launch = [craft.launchDate, craft.launchSite, craft.launchVehicle].filter(Boolean).join(' · ')
  const hasOrbit = craft.kind === 'orbital' || craft.kind === 'catalog'
  const epoch = craft.snapshot?.epoch ? `轨道历元 ${formatEpochUTC(craft.snapshot.epoch)} · ` : ''
  return {
    kind: 'spacecraft',
    typeZh: '飞行器',
    typeEn: 'SPACECRAFT',
    status: craft.type,
    nameZh: name.primary,
    nameEn: name.secondary,
    description: craft.description,
    fields: spacecraftFields({
      operator: craft.operatorName,
      launch,
      inclination: hasOrbit ? `${craft.displayInclination}°` : '',
      eccentricity: hasOrbit ? craft.displayEccentricity : '',
      period: hasOrbit ? craft.displayPeriod : '',
    }),
    source: `${epoch}${craft.sourceName}`,
  }
})
const orbitDataCaption = computed(() => {
  if (!crafts.value.length) return '火星轨道数据'
  return 'JPL Horizons'
})

/** 轨道历元统一 UTC 显示（与探测器面板同步时间格式一致，避免本地/UTC 混用） */
function formatEpochUTC(iso?: string) {
  if (!iso) return ''
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())} ${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())} UTC`
}

// 注：formatUTCDate 曾用于航天器名录"轨道历元"列，该列已移除（标称轨道无历元）

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

/** 双语名称（统一规则）：全部中文主，外国对象附英文括号注释 */
const craftBilingual = computed(() => new Map(crafts.value.map((c) => [c.id, bilingualName(c.nameZh, c.nameEn)])))
const siteBilingual = computed(() => new Map(landingSites.value.map((s) => [s.id, bilingualName(s.siteName, s.officialName || s.region)])))

/** 点击搜索结果/场景标签：选中并聚焦（滚回主视图 → 飞行器居中 → 右侧面板） */
function focusCraft(id: string) {
  if (focusTimer !== undefined) clearTimeout(focusTimer)
  selectedCraft.value = id
  selectedSite.value = null // 选中互斥：聚焦飞行器时取消着陆点选中
  document.getElementById('mars-scene')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

/** 板块点击着陆点：返回火星场景 + 选中 + 镜头放大居中该点
 *  先平滑滚动回场景，滚动结束后再启动聚焦动画（并行会掉帧） */
function focusSite(id: string) {
  if (focusTimer !== undefined) clearTimeout(focusTimer)
  selectedSite.value = id
  document.getElementById('mars-scene')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  focusTimer = window.setTimeout(() => {
    focusTimer = undefined
    if (selectedSite.value === id) startSiteFocus(id)
  }, 520)
}

/** 场景标签点击：切换选中（再次点击关闭），选中时镜头聚焦该点 */
function selectSite(id: string) {
  if (focusTimer !== undefined) clearTimeout(focusTimer)
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

/** 飞行器聚焦：镜头沿球面弧线转到飞行器方向，并停在飞行器外侧（正面可见）。
 *  月球/地球轨道贴面（r << 8），镜头停 8 即可；火星大轨道 r 可达 20+，
 *  若仍停 8 会被火星挡在背后 → "透过火星看到虚空"。故聚焦距离 = max(8, 飞行器当前半径 + 0.8)，
 *  保证飞行器始终在火星与相机之间、正面可见（方向球面插值绕行星弧线，不穿星球）。 */
function startCraftFocus(id: string) {
  const runtime = craftRuntimes.find((r) => r.spec.id === id)
  if (!runtime || !camera) return
  const world = runtime.dot.getWorldPosition(focusTmp).clone()
  planFocusMotion(world, spacecraftFocusDistance(MARS_RADIUS, camera.position.length(), world.length()))
}

/** 着陆点聚焦：方向对准着陆点（观察距离 5.2——火面区域与周边地形整体可见） */
function startSiteFocus(id: string) {
  const marker = siteMarkers.get(id)
  if (!marker || !camera) return
  const world = marker.getWorldPosition(focusTmp).clone()
  planFocusMotion(world, 7.4) // 着陆点聚焦距离统一 ≈2.5R（地球 5.4/2.15≈2.5R、月球 6.4/2.6≈2.5R）
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
/** 轴倾角组：rotation.z = 25.19°（真实火星轴倾角，与太阳系场景轴向一致），自转轴随之倾斜 */
let tiltPivot: THREE.Object3D | undefined
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
  // 初始视角：距火星中心 9（视半径 ~10°）——与类地行星同距离基准（金星 13.5° > 火星 10° > 水星 8.5° > 月球 7.1°），
  // 只要求大小关系正确（火星 < 地球 < 金星），不做严格比例
  camera.position.set(0, 0.95, 9)

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
  controls.minDistance = MARS_RADIUS * 1.4 // 拉近极限（与地球视大小一致）：地球 3.0 → 视半径 45.8°；火星 2.2 → 45.8°
  controls.maxDistance = 12 // 缩到最远：与地球视大小统一（地球 12 → 视半径 10.3°；火星 12 → 7.5°）——需 > 天问一号远心（新尺度 ≈9.4），保证镜头能越过飞行器聚焦

  // 火星本体：8k 贴图 + PBR 材质（保留质感，同地球模式）
  const texture = solarTexture(MARS_HD.textureUrl, () => emitTexturesReady())
  texture.colorSpace = THREE.SRGBColorSpace
  texture.anisotropy = 16
  marsMaterial = new THREE.MeshStandardMaterial({ map: texture, roughness: 0.95, metalness: 0.02 })
  // 细分 256 段：8k 贴图在 96 段球体上贴面时三角形过粗导致模糊，256 段显著提升贴面清晰度
  marsMesh = new THREE.Mesh(new THREE.SphereGeometry(MARS_RADIUS, 256, 256), marsMaterial)
  // 初始朝向：绕自转轴（局部 Y）旋转，让 lon 0° 子午线朝向相机——
  // 用绕 Y 轴的四元数（北极保持在局部 +Y = 自转轴，不产生极轴漂移；
  // 不能 setFromUnitVectors((1,0,0), cameraDir)——那会把北极也转离自转轴）
  marsMesh.quaternion.setFromAxisAngle(new THREE.Vector3(0, 1, 0), Math.atan2(-camera.position.z, camera.position.x))
  // 轴倾角组：rotation.z = 25.19°（真实火星轴倾角，黄道面参考；与太阳系场景 axial 一致）。
  // 层级 = tiltPivot(轴倾角) → swingPivot(自转) → marsMesh，自转绕倾斜后的火星极轴。
  tiltPivot = new THREE.Object3D()
  tiltPivot.name = 'mars-tilt-pivot'
  tiltPivot.rotation.order = 'ZYX'
  tiltPivot.rotation.z = 25.19 * DEG
  // 自转轴：火星自西向东慢速自转（真实周期 24.6h，场景做慢速可见旋转）
  swingPivot = new THREE.Object3D()
  swingPivot.name = 'mars-swing-pivot'
  swingPivot.add(marsMesh)
  tiltPivot.add(swingPivot)
  scene.add(tiltPivot)

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

  // 数据和 3D 对象在同一成功边界提交：任一接口失败就保留可恢复错误态，不渲染半套场景。
  void loadSceneData()

  renderer.render(scene, camera)

  let lastTime = performance.now()
  let lastSunUpdate = 0
  const animate = () => {
    frameId = requestAnimationFrame(animate)
    if (!renderer || !scene || !camera) return
    const now = performance.now()
    const delta = Math.min((now - lastTime) / 1000, 0.05)
    lastTime = now

    // 入场自转（自西向东绕自转轴，停稳后静止）
    updateMarsSpin(now)

    // 航天器公转（仅绕火轨道）
    for (const runtime of craftRuntimes) {
      if (runtime.spec.kind !== 'orbital' && runtime.spec.kind !== 'catalog') continue
      const sn = runtime.spec.snapshot ?? null
      // 快照存在时用真实公转周期（JPL 日同步，如 MRO 约 112 分钟）；
      // 无快照时回退静态轨道周期（迁移 034 起已改为真实周期）
      const periodSec = sn ? sn.periodSeconds : runtime.spec.periodSeconds
      runtime.nu += (Math.PI * 2 / periodSec) * delta
      const a = exaggeratedA(sn ? sn.aKm * MARS_SCENE_SCALE : runtime.spec.orbitA)
      const e = sn ? sn.eccentricity : runtime.spec.orbitE
      const argp = sn ? sn.argPeriapsisDeg : runtime.spec.argPeriapsisDeg
      const r = (a * (1 - e * e)) / (1 + e * Math.cos(runtime.nu))
      // 位置角度 = 真近点角 + 近点幅角（与轨道环一致：环整体旋转 argp；否则圆点在错误的椭圆上飞 → 空轨道）
      runtime.dot.position.set(r * Math.cos(runtime.nu + argp * DEG), r * Math.sin(runtime.nu + argp * DEG), 0)
    }
    // 观测光跟随相机：明暗边界始终落在球体轮廓之外（关闭晨昏线时 360° 全亮）
    if (observationLight && camera) observationLight.position.copy(camera.position)
    // 晨昏线开启：太阳方向随火日持续推进（每 60s 刷新，与地球页 updateSun 同节奏）
    if (terminatorEnabled.value && sunLight && now - lastSunUpdate > 60_000) {
      lastSunUpdate = now
      sunLight.position.copy(marsSunDirection()).multiplyScalar(10)
    }

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

    // 动态拖动灵敏度（与地球一致）：近处降敏、远处提速；默认视角 9 处 ≈ 0.46
    if (controls && camera) {
      const t = THREE.MathUtils.clamp((camera.position.length() - controls.minDistance) / (controls.maxDistance - controls.minDistance), 0, 1)
      controls.rotateSpeed = 0.2 + t * 0.5
    }
    // 统一元素淡入淡出：进入时星球渐入完成后一次性浮现；退出时全部一起消失（只留裸火星）
    const elementsFadeNow = updateElementsFade()
    for (const runtime of craftRuntimes) {
      const dotChild = runtime.dot.children[0] as THREE.Mesh | undefined
      const dotMat = dotChild?.material as THREE.MeshBasicMaterial | undefined
      // 高亮：悬停 > 选中（与地球页语义一致）；仅点亮不运镜
      const active = runtime.spec.id === (hoveredCraftId.value ?? selectedCraft.value)
      if (dotMat) dotMat.opacity = elementsFadeNow
      if (runtime.line) (runtime.line.material as THREE.LineBasicMaterial).opacity = (active ? 0.95 : 0.55) * elementsFadeNow
      // 点与标签共享同一屏幕空间缩放；高亮只改变亮度，不改变几何大小。
      const d = runtime.dot.getWorldPosition(focusTmp).distanceTo(camera.position)
      const viewportHeight = canvasHost.value?.clientHeight ?? 0
      const currentPlanetRadiusPx = projectedSphereRadiusPx(MARS_RADIUS, camera.position.length(), MARS_FOV, viewportHeight)
      const referencePlanetRadiusPx = projectedSphereRadiusPx(MARS_RADIUS, MARS_MARKER_REF_DISTANCE, MARS_FOV, viewportHeight)
      runtime.dot.scale.setScalar(sceneMarkerWorldRadius(d, MARS_FOV, viewportHeight, orbitMarkerRadiusPx(currentPlanetRadiusPx, referencePlanetRadiusPx)))
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

/** 场景单位 ↔ 真实尺寸：火星半径 1.57（场景，按地球 2.15 的 sqrt 压缩）↔ 3389.5 km（真实） */
const MARS_RADIUS = 1.57
const MARS_SCENE_SCALE = MARS_RADIUS / 3389.5
/** 轨道高度夸张（与地球 ALTITUDE_EXAGGERATION=3.2 同思路）：超出火面的部分放大 1.5 倍——
 *  MRO 真实轨道仅高出火面 ~8% 半径，视觉上贴面飞行；MAVEN/天问一号轨道本身达 2–3 倍
 *  半径，夸张取 1.5 兼顾可辨识度与取景 */
const MARS_ALTITUDE_EXAGGERATION = 1.5
/** 轨道半径（场景单位，含高度夸张）：火心 + 超出火面部分 × 夸张系数 */
function exaggeratedA(a: number) {
  return MARS_RADIUS + Math.max(0, a - MARS_RADIUS) * MARS_ALTITUDE_EXAGGERATION
}

/** 平近点角 → 真近点角（Kepler 方程，牛顿迭代） */
function keplerToTrueAnomaly(M: number, e: number): number {
  let E = M
  for (let k = 0; k < 8; k += 1) E = E - (E - e * Math.sin(E) - M) / (1 - e * Math.cos(E))
  return 2 * Math.atan2(Math.sqrt(1 + e) * Math.sin(E / 2), Math.sqrt(1 - e) * Math.cos(E / 2))
}

/** 按 API 数据构建单个飞行器（轨道平面/轨道线/运动点/拾取球）
 *  优先使用 JPL Horizons 日同步快照（真实形状 + 真实相位），无快照回退静态参数。
 *  surface（地表探测器）/ catalog（历史名录）无 3D 呈现：只入名录，点击看面板 */
function buildCraft(spec: MarsSpacecraft) {
  if (!scene) return
  if (spec.kind !== 'orbital' && spec.kind !== 'stationary' && spec.kind !== 'catalog') return
  const sn = spec.snapshot ?? null
  // 真实轨道根数（快照优先）：半长轴 km → 场景单位 → 轨道高度夸张（贴面飞行观感修正）
  const a = exaggeratedA(sn ? sn.aKm * MARS_SCENE_SCALE : spec.orbitA)
  const e = sn ? sn.eccentricity : spec.orbitE
  const inc = sn ? sn.inclinationDeg : spec.inclinationDeg
  const raan = sn ? sn.raanDeg : spec.raanDeg
  const argp = sn ? sn.argPeriapsisDeg : spec.argPeriapsisDeg

  const plane = new THREE.Object3D()
  if (spec.kind === 'orbital' || spec.kind === 'catalog') {
    plane.rotation.order = 'YXZ'
    plane.rotation.y = raan * DEG
    plane.rotation.x = inc * DEG
  }

  const dot = new THREE.Object3D()
  // 近点幅角不进 dot.rotation（只旋转球体无意义）：位置角度统一 = 真近点角 + argp（见 animate 循环）
  plane.add(dot)
  const dotMesh = new THREE.Mesh(
    new THREE.SphereGeometry(1, 16, 16),
    // transparent 必须为 true：否则分阶段揭示的 opacity=0 被忽略，圆点提前出现
    new THREE.MeshBasicMaterial({ color: spec.kind === 'stationary' ? 0xffd9a0 : 0xffb27d, transparent: true }),
  )
  dot.add(dotMesh)
  const hitSphere = new THREE.Mesh(
    new THREE.SphereGeometry(5.5, 8, 8),
    new THREE.MeshBasicMaterial({ transparent: true, opacity: 0, depthWrite: false }),
  )
  hitSphere.userData.craftId = spec.id
  dot.add(hitSphere)
  craftHitMeshes.push(hitSphere)

  let line: THREE.Line | null = null
  let initialNu = 0
  if (spec.kind === 'orbital' || spec.kind === 'catalog') {
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
    // 初始相位（首帧即被 animate 覆盖）：近点方向 + argp
    dot.position.set(a * (1 - e) * Math.cos(argp * DEG), a * (1 - e) * Math.sin(argp * DEG), 0)
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
      new THREE.SphereGeometry(1, 12, 12),
      new THREE.MeshBasicMaterial({ color, transparent: true, opacity: elementsFade }),
    )
    // 球心落在火面半径上（1.57）：球体一半嵌进表面（被火星深度遮挡）、一半露出——
    // "镶嵌"在火面上的观感；露出半球深度 < 表面 → 通过深度测试，无 z-fighting
    marker.position.copy(sitePosition(site.latitude, site.longitude, MARS_RADIUS))
    marker.userData = { kind: 'landing-site', siteId: site.id }
    marsMesh.add(marker)
    siteMarkers.set(site.id, marker)
    // 拾取球：扩大点击命中区域（点击圆点 → 选中并聚焦，标签随选中出现）
    const siteHit = new THREE.Mesh(
      new THREE.SphereGeometry(5, 8, 8),
      new THREE.MeshBasicMaterial({ transparent: true, opacity: 0, depthWrite: false }),
    )
    siteHit.userData.siteId = site.id
    marker.add(siteHit)

    // 火星车行驶轨迹：虚线折线（示意图，数据存库可替换真实遥测）
    if (site.track && site.track.length >= 2) {
      const points = site.track.map(([lat, lon]) => sitePosition(lat, lon, MARS_RADIUS * 1.008))
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

function siteCategoryLabel(category: string) {
  return category === 'ROVER_LANDING' ? '巡视探测' : category === 'SAMPLE_RETURN' ? '采样返回' : category === 'AERIAL' ? '动力飞行' : '静态着陆'
}

const selectedSiteDetail = computed<MissionDetail | null>(() => {
  if (!selectedSite.value) return null
  const site = siteById(selectedSite.value)
  if (!site) return null
  const name = bilingualName(site.siteName, site.officialName || site.nameEn)
  const coordinates = `${Math.abs(site.latitude).toFixed(2)}°${site.latitude >= 0 ? 'N' : 'S'} ${Math.abs(site.longitude).toFixed(2)}°${site.longitude >= 0 ? 'E' : 'W'}`
  return {
    kind: 'surface',
    typeZh: '着陆点',
    typeEn: 'LANDING SITE',
    nameZh: name.primary,
    nameEn: name.secondary,
    description: site.description,
    iconHtml: siteGlyph(site.icon),
    fields: surfaceMissionFields({
      mission: site.missionName,
      date: site.landingDate,
      operator: site.operatorName,
      region: site.region,
      coordinates,
      category: siteCategoryLabel(site.category),
    }),
    hardware: site.hardware,
  }
})

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

// ===== 统一分页（与地球目录/月球一致：每页 CATALOG_PAGE_SIZE 条）=====
const craftPage = ref(1)
const craftPageCount = computed(() => Math.max(1, Math.ceil(filteredCrafts.value.length / CATALOG_PAGE_SIZE)))
const pagedCrafts = computed(() =>
  filteredCrafts.value.slice((craftPage.value - 1) * CATALOG_PAGE_SIZE, craftPage.value * CATALOG_PAGE_SIZE),
)
const craftGotoPage = (delta: number) => {
  craftPage.value = Math.min(craftPageCount.value, Math.max(1, craftPage.value + delta))
}
const sitePage = ref(1)
const sitePageCount = computed(() => Math.max(1, Math.ceil(filteredSites.value.length / CATALOG_PAGE_SIZE)))
const pagedSites = computed(() =>
  filteredSites.value.slice((sitePage.value - 1) * CATALOG_PAGE_SIZE, sitePage.value * CATALOG_PAGE_SIZE),
)
const siteGotoPage = (delta: number) => {
  sitePage.value = Math.min(sitePageCount.value, Math.max(1, sitePage.value + delta))
}
watch([craftQuery, craftOperatorFilter, craftSort], () => { craftPage.value = 1 })
watch(siteQuery, () => { sitePage.value = 1 })

/** 所有标签都以圆点为唯一锚点；固定短线与标签一起缩放。 */
const annotationLabelStyle = sceneAnnotationStyle

const siteLabelStyle = annotationLabelStyle
const craftLabelStyle = annotationLabelStyle

/** 着陆点圆点按火星屏幕半径统一换算为目标像素尺寸。 */
function updateSiteMarkerProximity() {
  if (!camera || !sitesEnabled.value) return
  const viewportHeight = canvasHost.value?.clientHeight ?? 0
  const currentPlanetRadiusPx = projectedSphereRadiusPx(MARS_RADIUS, camera.position.length(), MARS_FOV, viewportHeight)
  const referencePlanetRadiusPx = projectedSphereRadiusPx(MARS_RADIUS, MARS_MARKER_REF_DISTANCE, MARS_FOV, viewportHeight)
  for (const site of landingSites.value) {
    const marker = siteMarkers.get(site.id)
    if (!marker) continue
    const world = marker.getWorldPosition(focusTmp)
    const d = world.distanceTo(camera.position)
    const material = (marker as THREE.Mesh).material as THREE.MeshBasicMaterial
    // 距离透明度（远处 70% 半透明、放大后实色）× 统一元素淡入淡出（进入一次性浮现 / 退出一次性消失）
    material.opacity = distOpacity(d) * elementsFade
    // 部分透视补偿（远小近大、不过度）：k=0.6；去掉原"贴面微缩"（近处缩小的观感反物理）
    const markerRadiusPx = surfaceMarkerRadiusPx(currentPlanetRadiusPx, referencePlanetRadiusPx, selectedSite.value === site.id)
    marker.scale.setScalar(surfaceMarkerWorldRadius(d, MARS_FOV, viewportHeight, markerRadiusPx))
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
// 晨昏线开关（镜像地球 applyDayNightMode / 月球 terminatorEnabled）：
// 关闭 = 观测光(相机方向) 3.1 + 太阳光 0 → 360° 全亮；
// 打开 = 太阳光 3.1 + 观测光 0 → 真实昼夜阴影

/** 真实太阳方向（火星参数）：按当前日期/时刻计算太阳在火星固连坐标系中的方向
 *  （北 = +y），用于晨昏线——黄赤交角 25.19°、火星年 687 地球日、火日 24.6229h。
 *  子日点黄经按火日推进（14.622°/h），赤纬按火星季节（25.19°·sin 火星年相位）；
 *  经 marsMesh 世界四元数变换，晨昏线落在火星正确位置 */
function marsSunDirection(): THREE.Vector3 {
  const now = new Date()
  const yearStart = Date.UTC(now.getUTCFullYear(), 0, 0)
  const dayOfYear = Math.floor((now.getTime() - yearStart) / 86_400_000)
  // 火星年相位：以真实春分 Ls=0 为锚（火星年 39 春分 ≈ 2026-10-01，dayOfYear 274）
  const declination = 25.19 * Math.sin(DEG * ((360 / 687) * (dayOfYear - 274)))
  const utcHours = now.getUTCHours() + now.getUTCMinutes() / 60 + now.getUTCSeconds() / 3600
  const subsolarLongitude = 180 - utcHours * (360 / 24.6229)
  const dir = sitePosition(declination, subsolarLongitude, 10)
  const q = new THREE.Quaternion()
  if (marsMesh) marsMesh.getWorldQuaternion(q)
  return dir.applyQuaternion(q)
}

watch(terminatorEnabled, (enabled) => {
  if (!observationLight || !sunLight) return
  observationLight.intensity = enabled ? 0 : 3.1
  sunLight.intensity = enabled ? 3.1 : 0
  if (enabled) {
    // 真实昼夜方向：晨昏线位置 = 此刻太阳方位（严格按时间，火星白天黑夜随火日推进）
    sunLight.position.copy(marsSunDirection()).multiplyScalar(10)
  }
})

watch(spacecraftEnabled, (enabled) => {
  for (const runtime of craftRuntimes) runtime.dot.visible = enabled
  if (!enabled) hoveredCraftId.value = null
})
// 轨道开关：显示/隐藏轨道线
watch(orbitsEnabled, (enabled) => {
  for (const runtime of craftRuntimes) if (runtime.line) runtime.line.visible = enabled
})
// 返回太阳系：全部多余元素 300ms 一次性淡出（统一 elementsFade），只留裸火星；
// host 随后接力渐隐火星本体；离开被中止时 leaving 回 false → 恢复显示
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
    .multiplyScalar(MARS_RADIUS * 1.08)
    .project(camera)
  const centerX = bounds.left + (projectedCenter.x * 0.5 + 0.5) * bounds.width
  const centerY = bounds.top + (-projectedCenter.y * 0.5 + 0.5) * bounds.height
  const radius = Math.abs(cameraRight.x - projectedCenter.x) * bounds.width * 0.5
  return Math.hypot(clientX - centerX, clientY - centerY) <= radius * 1.12
}

/** 飞行器是否被火星遮挡：视线段（相机→飞行器）与火星球体（半径 MARS_RADIUS）相交 */
function isCraftOccluded(world: THREE.Vector3) {
  if (!camera) return false
  // 0) 位于火星内部（大偏心轨道近日段 r<MARS_RADIUS，如 MOM 近日 r≈0.97）：球内绝不可见。
  //    射线-球体判定对"球体与相机之间的球内点"会漏判（最近点越过目标点）。
  //    严格按球面 3.0 判定（留 1e-3 浮点余量，cos²+sin² 表面点可能 ≈2.9999）——
  //    不能带 3.04 圆点余量：着陆点在表面上 r=3.0，带余量会把全部着陆点误隐藏
  if (world.length() < MARS_RADIUS - 1e-3) return true
  // 1) 视线段与火星球体相交（含掠射带 1.61 = 星球 1.57 + 圆点半径 0.04）：
  //    与地球 isOccludedByEarth / 月球 isCraftOccluded 同款"射线-球体"判定——
  //    视线被球挡住才隐藏，飞行器一出火星边缘立即可见（透明圆点另由 GPU 深度兜底盘面像素）。
  //    （历史：曾加过"背半球判定 world·camera<0"，会把飞行器藏到越过球心平面才显示，
  //     即转到盘面正中才见——过度隐藏，已移除；本函数历史上还有"盘面角锥"版本同理被弃）
  const toDot = world.clone().sub(camera.position)
  const dir = toDot.clone().normalize()
  const t = -camera.position.dot(dir)
  if (t > 0 && t < toDot.length()) {
    const closest = camera.position.clone().addScaledVector(dir, t)
    if (closest.length() < MARS_RADIUS + 0.04) return true
  }
  return false
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
  const tmp = new THREE.Vector3()
  const next: Array<{ id: string; anchorX: number; anchorY: number; visible: boolean; selected: boolean; hovered: boolean }> = []
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
      anchorX: (p.x * 0.5 + 0.5) * width,
      anchorY: (-p.y * 0.5 + 0.5) * height,
      visible: p.z > -1 && p.z < 1 && !occluded,
      selected: selectedCraft.value === runtime.spec.id,
      hovered: hoveredCraftId.value === runtime.spec.id,
    })
  }

  const annotationViewport = {
    width,
    height,
    currentPlanetRadiusPx: projectedSphereRadiusPx(MARS_RADIUS, camera.position.length(), MARS_FOV, height),
    referencePlanetRadiusPx: projectedSphereRadiusPx(MARS_RADIUS, MARS_MARKER_REF_DISTANCE, MARS_FOV, height),
  }
  craftLabels.value = layoutSceneAnnotations(next, annotationViewport, craftLabels.value)

  // 着陆点标签：背面隐藏（圆点本体由材质深度测试自然遮挡）
  const siteNext: Array<{ id: string; anchorX: number; anchorY: number; visible: boolean; selected: boolean }> = []
  const siteTmp = new THREE.Vector3()
  for (const site of landingSites.value) {
    const marker = siteMarkers.get(site.id)
    if (!marker) continue
    const world = marker.getWorldPosition(siteTmp)
    const cp = world.clone().project(camera)
    const occluded = isCraftOccluded(world) // 同款背面判定：法线朝向相机才显示
    siteNext.push({
      id: site.id,
      anchorX: (cp.x * 0.5 + 0.5) * width,
      anchorY: (-cp.y * 0.5 + 0.5) * height,
      visible: cp.z > -1 && cp.z < 1 && !occluded,
      selected: selectedSite.value === site.id,
    })
  }
  siteLabels.value = layoutSceneAnnotations(siteNext, annotationViewport, siteLabels.value)
}

onBeforeUnmount(() => {
  abortSceneData()
  if (elementsRevealTimer !== undefined) clearTimeout(elementsRevealTimer)
  if (focusTimer !== undefined) clearTimeout(focusTimer)
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
  --mission-accent: var(--mars-accent);
  --mission-accent-dim: var(--mars-accent-dim);
  --mission-line: var(--mars-line);
  --mission-text: var(--mars-text);
  --mission-quiet: var(--mars-quiet);
  --mission-body: #c8ad97;
  --mission-panel-surface: rgba(24, 16, 10, .95);
  --mission-label-surface: rgba(18, 12, 8, .8);
  --mission-label-surface-active: rgba(31, 21, 14, .94);
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
.mars-objects-section .pagination-space,
.mars-sites-section .pagination-space { border-top: 1px solid rgba(224, 168, 120, .15); color: #a89078; }
.mars-objects-section .pagination-space button,
.mars-sites-section .pagination-space button { border-color: rgba(224, 168, 120, .3); color: #c9a17a; background: transparent; }
.mars-objects-section .pagination-space button:not(:disabled):hover,
.mars-sites-section .pagination-space button:not(:disabled):hover {
  border-color: #e0a878;
  color: #e0a878;
  background: rgba(224, 168, 120, .1);
}
.mars-objects-section .pagination-space button:not(:disabled):active,
.mars-sites-section .pagination-space button:not(:disabled):active {
  transform: scale(.93);
  background: rgba(224, 168, 120, .18);
  box-shadow: 0 0 8px rgba(224, 168, 120, .25);
}
.mars-objects-section .section-kicker,
.mars-sites-section .section-kicker { color: #d0a080; }
/* 火星档案板块（与金星/土星/木星页同款 profile-grid，主题色陶土红） */
.mars-profile-section {
  min-height: 0;
  padding-bottom: 48px;
}
.mars-profile-section .section-kicker { color: #d0a080; }
.mars-profile-section .sec-num { color: #c09070; }
.mars-profile-section .profile-grid {
  max-width: 640px;
  padding: 28px 0 44px;
}
.mars-profile-section .profile-table {
  display: grid;
  gap: 0;
  margin: 0;
  border: 1px solid rgba(224, 168, 120, .22);
  border-radius: 8px;
  background: rgba(20, 12, 7, .55);
  overflow: hidden;
}
.mars-profile-section .profile-table > div {
  display: grid;
  grid-template-columns: 100px 1fr;
  gap: 16px;
  align-items: baseline;
  padding: 12px 18px;
  border-bottom: 1px solid rgba(224, 168, 120, .22);
}
.mars-profile-section .profile-table > div:last-child { border-bottom: 0; }
.mars-profile-section .profile-table dt {
  color: #b09880;
  font: 500 10px var(--font-mono);
  letter-spacing: .1em;
  padding-top: 2px;
}
.mars-profile-section .profile-table dd {
  margin: 0;
  color: #ecd9c8;
  font-size: 13px;
  line-height: 1.6;
}
/* 简介行：并入表格最后一行（消除右侧独立文字），文字用 quiet 色、放宽行距更耐读 */
.mars-profile-section .profile-intro-row {
  align-items: start;
  background: rgba(255, 255, 255, .02);
}
.mars-profile-section .profile-intro-row dd {
  color: #b09880;
  font-size: 12px;
  line-height: 1.9;
}
/* 板块小字标注：航天器=飞行器 / 着陆点=着陆器 */
.mars-objects-section .section-sub,
.mars-sites-section .section-sub {
  margin: 6px 0 0;
  color: var(--mars-quiet);
  font: 400 10px/1.5 var(--font-mono);
  letter-spacing: .08em;
}
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
.mars-objects-section .object-row {
  grid-template-columns: minmax(260px, 1.6fr) minmax(200px, 1.1fr) minmax(180px, 1fr);
}
/* 对象列现在是首列：全局 first-child mono 字体仅应作用于编码类列，名称用正文字体 */
.mars-objects-section .object-row > span:first-child { font: inherit; }
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
.mars-scene-host.revealed.leaving-body {
  opacity: 0;
  pointer-events: none;
  transition: opacity .32s cubic-bezier(.4, 0, 1, 1) .3s;
}
.mars-scene-host canvas { display: block; }
.scene-data-state {
  position: absolute;
  z-index: 7;
  left: 32px;
  top: 82px;
  display: flex;
  align-items: center;
  gap: 12px;
  max-width: min(440px, calc(100% - 64px));
  color: var(--mars-quiet);
  font: 500 10px var(--font-mono);
  letter-spacing: .06em;
}
.scene-data-state.error { color: #ffcf8f; }
.scene-data-state button {
  border: 1px solid var(--mars-line);
  border-radius: 3px;
  padding: 5px 8px;
  background: rgba(18, 12, 8, .72);
  color: var(--mars-text);
  font: inherit;
  cursor: pointer;
}

/* 标注尺寸、布局和左右短线统一由 MissionSceneLabel 管理。 */

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

/* 返回渐隐：工具栏与标签同节奏淡出，随后 host 接力渐隐火星本体 */
.scene-toolbar.leaving-fade { opacity: 0; pointer-events: none; transition: opacity .3s ease; }
/* 返回渐隐：左下角读数/右下角署名随元素一起淡出 */
.mars-readout.leaving-fade,
.mars-credits.leaving-fade { opacity: 0; transition: opacity .3s ease; }
.mission-detail-panel.leaving-fade {
  animation: none !important;
  opacity: 0 !important;
  pointer-events: none;
  transition: opacity .3s ease;
}

@media (prefers-reduced-motion: reduce) {
  .mars-scene-host.revealed.leaving-body { transition: opacity .1s linear .04s; }
}

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
  /* 低对比深色底：右上略深 → 左下微亮，幅度极小，面板边缘与背景无色差（暖褐底，无蓝色元素） */
  background: linear-gradient(200deg, rgba(18, 12, 8, .97) 0%, rgba(32, 22, 14, .95) 100%);
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
.mars-page-footer { padding: 10px 0 56px; }
.mars-page-footer .source-list { color: #a89078; }
.mars-page-footer .source-list i.healthy { background: #79e3bd; }
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
