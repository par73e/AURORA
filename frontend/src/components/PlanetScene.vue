<template>
  <section class="planet-section" :class="planet.themeKey" :aria-labelledby="`${planet.key}-title`">
    <div :id="`${planet.key}-scene`" class="planet-scene-frame">
      <div ref="canvasHost" class="planet-scene-host" :class="{ revealed: sceneRevealed, 'leaving-body': leaving }" role="group" :aria-label="`${planet.name}三维视图，左上角可返回太阳系`">
        <!-- 工具栏：行星页统一图层控制；只为确实存在的数据提供开关 -->
        <div ref="sceneToolbarRef" class="scene-toolbar" :class="{ 'leaving-fade': leaving }" aria-label="场景图层">
          <span>图层</span>
          <label v-if="planet.spacecraft"><input v-model="spacecraftEnabled" type="checkbox"><i />飞行器</label>
          <label v-if="planet.spacecraft"><input v-model="orbitsEnabled" type="checkbox"><i class="orbits" />轨道</label>
          <label v-if="planet.exploration"><input v-model="sitesEnabled" type="checkbox"><i class="sites" />{{ planet.exploration.title }}</label>
          <label v-if="!planet.star"><input v-model="terminatorEnabled" type="checkbox"><i class="terminator" />晨昏线</label>
        </div>

        <!-- 探测器标签：只有前半球且对应图层开启时出现 -->
        <MissionSceneLabel
          v-for="label in craftLabels"
          v-show="label.visible && spacecraftEnabled"
          :key="label.id"
          class="planet-craft-label"
          :class="{ selected: selectedCraft === label.id, 'leaving-fade': leaving }"
          :style="labelStyle(label)"
          kind="spacecraft"
          :name-zh="label.name"
          :name-en="label.nameEn"
          :selected="selectedCraft === label.id"
          :mode="label.mode"
          :side="label.side"
          :cluster-count="label.clusterCount"
          :cluster-items="label.memberIds.map((id) => ({ id, name: craftById(id)?.name ?? id }))"
          :aria-label="label.mode === 'cluster' ? `${label.clusterCount} 个相近飞行器` : `${label.name}${label.nameEn !== label.name ? `（${label.nameEn}）` : ''}`"
          @pointerenter="hoveredCraft = label.mode === 'cluster' ? null : label.id"
          @pointerleave="hoveredCraft = null"
          @click.stop="selectCraft(label.id)"
          @select-member="selectCraft($event)"
        />

        <!-- 足迹标签：有坐标且在行星前半球才出现，背面由球体遮挡 -->
        <MissionSceneLabel
          v-for="label in siteLabels"
          :key="label.id"
          class="planet-site-label"
          v-show="label.visible && sitesEnabled"
          :data-icon="label.icon"
          :class="{ selected: selectedSite === label.id, 'leaving-fade': leaving }"
          :style="surfaceLabelStyle(label)"
          kind="surface"
          :name-zh="label.name"
          :name-en="label.nameEn"
          :selected="selectedSite === label.id"
          :mode="label.mode"
          :side="label.side"
          :cluster-count="label.clusterCount"
          :cluster-items="label.memberIds.map((id) => ({ id, name: markerSites.find((site) => site.id === id)?.name ?? id }))"
          :icon-html="siteGlyph(label.icon)"
          :aria-label="`${label.name}${label.nameEn !== label.name ? `（${label.nameEn}）` : ''}${planet.exploration?.title === '任务终点' ? '，任务终点' : ''}`"
          @click.stop="selectSite(label.id)"
          @select-member="selectSite($event)"
        />

        <!-- 选中探测器的信息卡：与月球/火星场景保持同一互斥选择逻辑 -->
        <MissionDetailPanel
          v-if="selectedCraftDetail"
          :class="{ 'leaving-fade': leaving }"
          :detail="selectedCraftDetail"
          :style="panelHeaderOffset"
          @close="selectedCraft = null"
        />

        <!-- 选中着陆点/撞击点的信息卡 -->
        <MissionDetailPanel
          v-if="selectedSiteDetail"
          :class="{ 'leaving-fade': leaving }"
          :detail="selectedSiteDetail"
          :style="panelHeaderOffset"
          @close="selectedSite = null"
        />

        <!-- 左下角读数：常驻行星（返回时随元素一起淡出） -->
        <div class="planet-readout" :class="{ 'leaving-fade': leaving }" aria-live="polite">
          <span>{{ planet.nameEn }} ORBIT</span>
          <strong>{{ planet.name }}</strong>
        </div>

        <!-- 右下角：纹理署名（SSS CC BY 4.0，与太阳系页同位置；返回时随元素一起淡出） -->
        <div class="planet-credits" :class="{ 'leaving-fade': leaving }" aria-hidden="true">Solar System Scope · CC BY 4.0</div>
      </div>
    </div>
  </section>

  <!-- 下方：行星档案板块（简单静态真实数据，延续月球/火星页面可滚动框架） -->
  <section :id="`${planet.key}-profile`" class="content-section planet-profile-section" :class="planet.themeKey">
    <div class="page-frame">
      <div class="section-heading">
        <div><p class="section-kicker">{{ planet.profile.kicker }}</p><h2><i class="sec-num">Ⅱ</i>{{ planet.name }}档案</h2></div>
      </div>
      <div class="profile-grid">
        <dl class="profile-table">
          <!-- 恒星（太阳）专属档案行 -->
          <template v-if="planet.star">
            <div><dt>类型</dt><dd>{{ planet.profile.type }}</dd></div>
            <div><dt>直径</dt><dd>{{ planet.profile.diameter }}</dd></div>
            <div><dt>质量</dt><dd>{{ planet.profile.mass }}</dd></div>
            <div><dt>表面温度</dt><dd>{{ planet.profile.surfaceTemp }}</dd></div>
            <div><dt>核心温度</dt><dd>{{ planet.profile.coreTemp }}</dd></div>
            <div><dt>距地距离</dt><dd>{{ planet.profile.distance }}</dd></div>
            <div><dt>年龄</dt><dd>{{ planet.profile.age }}</dd></div>
            <div><dt>自转周期</dt><dd>{{ planet.profile.rotation }}</dd></div>
            <div><dt>成分</dt><dd>{{ planet.profile.composition }}</dd></div>
          </template>
          <!-- 行星档案行 -->
          <template v-else>
            <div><dt>直径</dt><dd>{{ planet.profile.diameter }}</dd></div>
            <div><dt>距日</dt><dd>{{ planet.profile.distance }}</dd></div>
            <div><dt>自转周期</dt><dd>{{ planet.profile.rotation }}</dd></div>
            <div><dt>太阳日</dt><dd>{{ planet.profile.solarDay }}</dd></div>
            <div><dt>公转周期</dt><dd>{{ planet.profile.orbit }}</dd></div>
            <div><dt>轴倾角</dt><dd>{{ planet.profile.axialTilt }}</dd></div>
            <div><dt>卫星</dt><dd>{{ planet.profile.moons }}</dd></div>
            <div><dt>环</dt><dd>{{ planet.profile.rings }}</dd></div>
            <div><dt>成分</dt><dd>{{ planet.profile.composition }}</dd></div>
          </template>
          <div class="profile-intro-row"><dt>简介</dt><dd>{{ planet.profile.description }}</dd></div>
        </dl>
      </div>
    </div>
  </section>

  <!-- 下方：探测器板块（所有七个天体均有对应任务资料） -->
  <section v-if="planet.spacecraft" :id="`${planet.key}-objects`" class="content-section planet-spacecraft-section" :class="planet.themeKey">
    <div class="page-frame">
      <div class="section-heading">
        <div><p class="section-kicker">{{ planet.spacecraft.kicker }}</p><h2><i class="sec-num">Ⅲ</i>{{ planet.spacecraft.title }}</h2><p class="section-sub">{{ planet.spacecraft.sub }}</p></div>
      </div>
      <div class="catalog-workspace" :class="{ compact: planet.spacecraft.compact }">
        <div v-if="!planet.spacecraft.compact" class="catalog-controls">
          <label class="search-field">
            <span>名称、任务、机构或正则表达式</span>
            <input v-model="craftQuery" type="search" :placeholder="craftSearchPlaceholder" spellcheck="false" />
          </label>
          <label><span>状态</span><select v-model="craftStatusFilter"><option value="all">全部状态</option><option v-for="status in craftStatuses" :key="status" :value="status">{{ status }}</option></select></label>
          <label><span>排序</span><select v-model="craftSort"><option value="name">名称</option><option value="type">类型</option><option value="operator">机构</option></select></label>
        </div>
        <div class="object-table planet-craft-table" role="table" :aria-label="`${planet.name}探测器列表`">
          <div class="object-table-head" role="row"><span>对象</span><span>机构</span><span>状态</span><span>类型</span></div>
          <button v-for="craft in pagedCrafts" :key="craft.id" class="object-row planet-craft-row" role="row" @click="focusCraft(craft.id)">
            <span><strong>{{ craft.name }}</strong><small v-if="craft.nameEn !== craft.name">{{ craft.nameEn }}</small></span>
            <span>{{ craft.operator }}</span>
            <span><i class="craft-status-dot" :class="`status-${craft.status}`" />{{ craft.status }}</span>
            <span>{{ craft.type }}</span>
          </button>
          <div v-if="!filteredCrafts.length" class="catalog-empty">没有符合条件的飞行器。请修改搜索词。</div>
        </div>
        <div v-if="!planet.spacecraft.compact" class="pagination-space"><span>第 {{ craftPage }} / {{ craftPageCount }} 页 · {{ filteredCrafts.length }} 个飞行器</span><div><button :disabled="craftPage <= 1" @click="craftGotoPage(-1)">上一页</button><button :disabled="craftPage >= craftPageCount" @click="craftGotoPage(1)">下一页</button></div></div>
      </div>
    </div>
  </section>

  <!-- 下方：人类探索板块（着陆点/任务终点；有坐标的点击 → 返回场景并放大居中该点） -->
  <section v-if="planet.exploration" :id="`${planet.key}-sites`" class="content-section planet-sites-section" :class="planet.themeKey">
    <div class="page-frame">
      <div class="section-heading">
        <div><p class="section-kicker">{{ planet.exploration.kicker }}</p><h2><i class="sec-num">Ⅳ</i>{{ planet.exploration.title }}</h2><p class="section-sub">{{ planet.exploration.sub }}</p></div>
      </div>
      <div class="catalog-workspace" :class="{ compact: planet.exploration.compact }">
        <div v-if="!planet.exploration.compact" class="catalog-controls">
          <label class="search-field">
            <span>名称、任务或机构</span>
            <input v-model="siteQuery" type="search" :placeholder="`输入 ${planet.exploration.sites[0]?.name ?? ''}…`" spellcheck="false" />
          </label>
        </div>
        <div class="object-table" role="table" :aria-label="`${planet.name}${planet.exploration.title}列表`">
          <div class="object-table-head" role="row"><span>名称</span><span>任务</span><span>日期</span><span>类型</span></div>
          <button v-for="site in pagedSites" :key="site.id" class="object-row site-row" :data-icon="site.icon" role="row" @click="focusSite(site.id)">
            <span class="site-row-name">
              <span class="planet-site-glyph" v-html="siteGlyph(site.icon)" />
              <span><strong>{{ site.name }}</strong><small v-if="site.nameEn !== site.name">{{ site.nameEn }}</small></span>
            </span>
            <span>{{ site.mission }}<small>{{ site.operator }}</small></span>
            <span>{{ site.date }}</span>
            <span><small>{{ siteKindLabel(site.kind) }}</small></span>
          </button>
          <div v-if="!filteredSites.length" class="catalog-empty">没有符合条件的记录。请修改搜索词。</div>
        </div>
      </div>
    </div>
  </section>

  <!-- 页脚：仅品牌（纹理署名在场景右下角 planet-credits，与火星/月球一致，不在页脚重复） -->
  <footer class="planet-page-footer">
    <div class="page-frame footer-inner">
      <div><strong>AURORA / {{ planet.nameEn }}</strong></div>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as THREE from 'three'
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'
import { solarTexture } from '../solar/textures'
import type { PlanetCraft, PlanetCraftTrajectory, PlanetCraftTrajectoryKind, PlanetPageConfig } from '../planetPages'
import MissionDetailPanel from './MissionDetailPanel.vue'
import MissionSceneLabel from './MissionSceneLabel.vue'
import type { MissionDetail } from '../missionPresentation'
import { ENDPOINT_SCENE_NOTE, spacecraftFields, spacecraftFocusDistance, surfaceFocusDistance, surfaceMissionFields } from '../missionPresentation'
import type { SceneAnnotationLayout, SurfaceAnnotationLayout } from '../surfaceAnnotations'
import { layoutSceneAnnotations, sceneAnnotationStyle, projectedSphereRadiusPx, orbitMarkerRadiusPx, sceneMarkerWorldRadius, surfaceMarkerRadiusPx, surfaceMarkerWorldRadius } from '../surfaceAnnotations'

const props = defineProps<{ planet: PlanetPageConfig; spacecraftVisible?: boolean; revealTick?: number; enterFromSolar?: boolean; leaving?: boolean; headerExpanded?: boolean }>()
const emit = defineEmits<{
  'blank-click': []
  'update:spacecraft-visible': [visible: boolean]
  /** 场景首帧贴图渲染完成（解码 + GPU 上传后）——过渡遮罩等待此信号再揭示 */
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
const terminatorEnabled = ref(false)
const spacecraftEnabled = computed({
  get: () => props.spacecraftVisible ?? true,
  set: (visible: boolean) => emit('update:spacecraft-visible', visible),
})
const orbitsEnabled = ref(true)
/** 着陆点/任务终点开关；大气坠毁仅在有官方发布或可靠复算坐标时绘制标记。 */
const sitesEnabled = ref(true)
const selectedCraft = ref<string | null>(null)
/** 鼠标悬停的飞行器（标签或 3D 圆点）：高亮优先于选中，移开即恢复 */
const hoveredCraft = ref<string | null>(null)
const selectedSite = ref<string | null>(null)
const craftQuery = ref('')
const craftStatusFilter = ref<PlanetCraft['status'] | 'all'>('all')
const craftSort = ref<'name' | 'type' | 'operator'>('name')
const siteQuery = ref('')

const craftStatuses: PlanetCraft['status'][] = ['运行中', '即将入轨', '飞掠', '已结束']
const planetCrafts = computed(() => props.planet.spacecraft?.items ?? [])
const craftSearchPlaceholder = computed(() => {
  const examples = planetCrafts.value
    .slice(0, 2)
    .map((craft) => craft.nameEn || craft.name)
    .filter(Boolean)
    .join('、')
  return examples ? `输入 ${examples}，或使用正则表达式` : '输入名称、任务或机构'
})
function matchesCraftQuery(craft: PlanetCraft, query: string) {
  const haystack = [craft.name, craft.nameEn, craft.operator, craft.type, craft.description, craft.endpoint ?? ''].join(' ')
  const regexMatch = query.match(/^\/(.*)\/([dgimsuvy]*)$/i)
  if (regexMatch) {
    try {
      return new RegExp(regexMatch[1], regexMatch[2]).test(haystack)
    } catch {
      return false
    }
  }
  return haystack.toLowerCase().includes(query.toLowerCase())
}
const filteredCrafts = computed(() => {
  const q = craftQuery.value.trim()
  const matching = planetCrafts.value.filter((craft) => {
    if (craftStatusFilter.value !== 'all' && craft.status !== craftStatusFilter.value) return false
    if (!q) return true
    return matchesCraftQuery(craft, q)
  })
  return [...matching].sort((a, b) => {
    if (craftSort.value === 'operator') return a.operator.localeCompare(b.operator, 'zh-CN')
    if (craftSort.value === 'type') return a.type.localeCompare(b.type, 'zh-CN')
    return a.name.localeCompare(b.name, 'zh-CN')
  })
})
const CRAFT_PAGE_SIZE = 8
const craftPage = ref(1)
const craftPageCount = computed(() => Math.max(1, Math.ceil(filteredCrafts.value.length / CRAFT_PAGE_SIZE)))
const pagedCrafts = computed(() => filteredCrafts.value.slice((craftPage.value - 1) * CRAFT_PAGE_SIZE, craftPage.value * CRAFT_PAGE_SIZE))
watch(filteredCrafts, () => { craftPage.value = 1 })

function craftById(id: string) {
  return planetCrafts.value.find((craft) => craft.id === id)
}
function craftTrajectoryLabel(kind?: PlanetCraftTrajectoryKind) {
  return kind === 'flyby' ? '飞掠弧线（示意）' : kind === 'orbit' ? '环绕轨道（示意）' : '任务资料'
}
function formatPeriod(days: number) {
  if (days >= 365) return `${(days / 365.25).toFixed(1)} 年`
  if (days >= 1) return `${days.toFixed(days % 1 ? 1 : 0)} 天`
  return `${(days * 24).toFixed(1)} 小时`
}
function craftGotoPage(delta: number) {
  craftPage.value = Math.min(craftPageCount.value, Math.max(1, craftPage.value + delta))
}

/** 有坐标的足迹（可画 3D 标记 + 标签）：landing/impact 有坐标，atmospheric 无 */
const markerSites = computed(() => props.planet.exploration?.sites.filter((s) => s.latitude != null && s.longitude != null) ?? [])
/** 标签 overlay 数据（含屏幕投影坐标，rAF 更新） */
type OverlayLabel = SurfaceAnnotationLayout & { name: string; nameEn: string; mission: string; type: string; icon: 'lander' | 'probe' | 'impact' }
type CraftOverlayLabel = SceneAnnotationLayout & { name: string; nameEn: string; type: string }
const siteLabels = ref<OverlayLabel[]>([])
const craftLabels = ref<CraftOverlayLabel[]>([])

/** 目录搜索过滤 */
const filteredSites = computed(() => {
  const q = siteQuery.value.trim().toLowerCase()
  const sites = props.planet.exploration?.sites ?? []
  if (!q) return sites
  return sites.filter(
    (s) =>
      s.name.toLowerCase().includes(q) ||
      s.nameEn.toLowerCase().includes(q) ||
      s.mission.toLowerCase().includes(q) ||
      s.operator.toLowerCase().includes(q),
  )
})
const PAGE_SIZE = 8
const sitePage = ref(1)
const pagedSites = computed(() => {
  const start = (sitePage.value - 1) * PAGE_SIZE
  return filteredSites.value.slice(start, start + PAGE_SIZE)
})
watch(filteredSites, () => {
  sitePage.value = 1
})

function siteById(id: string) {
  return props.planet.exploration?.sites.find((s) => s.id === id)
}

const selectedCraftDetail = computed<MissionDetail | null>(() => {
  if (!selectedCraft.value) return null
  const craft = craftById(selectedCraft.value)
  if (!craft) return null
  const launch = craft.date ? `${craft.date} · 发射日期` : ''
  return {
    kind: 'spacecraft',
    typeZh: '飞行器',
    typeEn: 'SPACECRAFT',
    status: `${craft.status} · ${craft.type}`,
    nameZh: craft.name,
    nameEn: craft.nameEn,
    description: craft.description,
    fields: spacecraftFields({
      operator: craft.operator,
      launch,
      endpoint: craft.endpoint,
      period: craft.trajectory?.periodDays ? `约 ${formatPeriod(craft.trajectory.periodDays)}` : '',
      trajectory: craft.trajectory ? craftTrajectoryLabel(craft.trajectory.kind) : '',
    }),
    source: `${craft.verifiedAt ? `${craft.verifiedAt} · ` : ''}${craft.source ?? '公开任务档案'}`,
  }
})

const selectedSiteDetail = computed<MissionDetail | null>(() => {
  if (!selectedSite.value) return null
  const site = siteById(selectedSite.value)
  if (!site) return null
  const endpoint = props.planet.exploration?.title === '任务终点'
  const coordinates = site.latitude != null && site.longitude != null ? formatCoordinate(site.latitude, site.longitude) : ''
  return {
    kind: 'surface',
    typeZh: endpoint ? '任务终点' : '着陆点',
    typeEn: endpoint ? 'MISSION ENDPOINT' : 'LANDING SITE',
    nameZh: site.name,
    nameEn: site.nameEn,
    description: site.description,
    iconHtml: siteGlyph(site.icon),
    fields: surfaceMissionFields({
      mission: site.mission,
      date: site.date,
      operator: site.operator,
      category: siteKindLabel(site.kind),
      coordinates,
    }),
    source: `${site.verifiedAt ? `${site.verifiedAt} · ` : ''}${site.source ?? '公开任务档案'}`,
    note: endpoint ? ENDPOINT_SCENE_NOTE : undefined,
  }
})

function returnToPlanetScene() {
  document.getElementById(`${props.planet.key}-scene`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

/** 场景中的飞行器标签：切换选择，并立即围绕当前真实位置运镜。 */
function selectCraft(id: string) {
  if (focusTimer !== undefined) clearTimeout(focusTimer)
  const next = selectedCraft.value === id ? null : id
  selectedCraft.value = next
  if (next) {
    selectedSite.value = null
    startCraftFocus(next)
  }
}

/** 目录中的飞行器：先返回主场景，再以当前点位完成聚焦，节奏与月球/火星一致。 */
function focusCraft(id: string) {
  if (focusTimer !== undefined) clearTimeout(focusTimer)
  selectedCraft.value = id
  selectedSite.value = null
  emit('blank-click')
  returnToPlanetScene()
  focusTimer = window.setTimeout(() => {
    focusTimer = undefined
    if (selectedCraft.value === id) startCraftFocus(id)
  }, 520)
}
function siteKindLabel(kind?: 'landing' | 'impact' | 'atmospheric') {
  return kind === 'landing' ? '软着陆' : kind === 'impact' ? '表面撞击' : kind === 'atmospheric' ? '大气层坠毁' : ''
}
function formatCoordinate(lat: number, lon: number) {
  return `${Math.abs(lat).toFixed(2)}°${lat >= 0 ? 'N' : 'S'} ${lon.toFixed(2)}°E`
}
function siteGlyph(icon: 'lander' | 'probe' | 'impact') {
  const stroke = 'currentColor'
  switch (icon) {
    case 'impact':
      return `<svg viewBox="0 0 12 12" width="12" height="12" fill="none" stroke="${stroke}" stroke-width="1.1" stroke-linecap="round"><circle cx="6" cy="6" r="4.2"/><path d="M6 2.6v2.2M6 7.2v2.2M2.6 6h2.2M7.2 6h2.2"/></svg>`
    case 'probe':
      return `<svg viewBox="0 0 12 12" width="12" height="12" fill="none" stroke="${stroke}" stroke-width="1.1" stroke-linecap="round" stroke-linejoin="round"><circle cx="6" cy="6" r="1.6"/><path d="M6 1.8v1.6M6 8.6v1.6M1.8 6h1.6M8.6 6h1.6"/></svg>`
    default:
      return `<svg viewBox="0 0 12 12" width="12" height="12" fill="none" stroke="${stroke}" stroke-width="1.1" stroke-linecap="round" stroke-linejoin="round"><path d="M6 1.6 9.4 10.4H2.6z"/><path d="M6 5.4v5"/></svg>`
  }
}

/** 标签紧贴圆点两侧，固定短线与标签使用相同的缩放原点。 */
const annotationLabelStyle = sceneAnnotationStyle

const labelStyle = annotationLabelStyle
const surfaceLabelStyle = annotationLabelStyle

/** 场景中的表面航天器：切换选择，避免重复点击仍强制启动一次运镜。 */
function selectSite(id: string) {
  if (focusTimer !== undefined) clearTimeout(focusTimer)
  const next = selectedSite.value === id ? null : id
  selectedSite.value = next
  if (next) {
    selectedCraft.value = null
    startSiteFocus(next)
  }
}

/** 目录中的表面航天器：回到场景后再聚焦，避免滚动和 WebGL 运镜互相抢帧。 */
function focusSite(id: string) {
  if (focusTimer !== undefined) clearTimeout(focusTimer)
  selectedSite.value = id
  selectedCraft.value = null
  emit('blank-click')
  returnToPlanetScene()
  focusTimer = window.setTimeout(() => {
    focusTimer = undefined
    if (selectedSite.value === id) startSiteFocus(id)
  }, 520)
}

/** 所有对象统一使用围绕行星中心的短弧运镜，避免相机直穿行星或飞到对象内部。 */
function planFocusMotion(targetPos: THREE.Vector3, targetDistance: number) {
  if (!camera || !controls) return
  const distance = THREE.MathUtils.clamp(targetDistance, controls.minDistance, controls.maxDistance)
  focusAnimation = {
    fromPos: camera.position.clone(),
    toPos: targetPos.clone().normalize().multiplyScalar(distance),
    startedAt: performance.now(),
    duration: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 150 : 920,
  }
  dragResetTarget = false
  controls.enabled = false
}

/** 聚焦着陆点/任务终点：只有具备可靠坐标的记录才会进入 markerSites。 */
function startSiteFocus(id: string) {
  const marker = siteMarkers.get(id)
  if (!marker || !camera || !controls) return
  // marker 已经过轴倾角和入场自转的父级变换；必须取世界坐标，不能把未变换的经纬度向量直接当镜头目标。
  planFocusMotion(marker.getWorldPosition(focusTmp).clone(), surfaceFocusDistance(props.planet.radius))
}

/** 飞行器聚焦取当前 Three.js 点位，而不是固定轨迹相位；动态任务的标签与镜头因此永远指向同一对象。 */
function startCraftFocus(id: string) {
  const runtime = craftRuntimes.get(id)
  if (!runtime || !camera || !controls) return
  const world = runtime.dot.getWorldPosition(focusTmp).clone()
  planFocusMotion(world, spacecraftFocusDistance(props.planet.radius, camera.position.length(), world.length()))
}
const focusTmp = new THREE.Vector3()
const focusTmp2 = new THREE.Vector3()
const focusTmp3 = new THREE.Vector3()
const focusTmp4 = new THREE.Vector3()
const origin = new THREE.Vector3()
let focusTimer: number | undefined

/** 入场渐亮：从太阳系进入（enterFromSolar）时等待 revealTick 递增；直接加载默认已亮。
 *  不能用 revealTick 判初始态——它只增不减，第二次进入时非 0 会误判为"直接加载" */
const sceneRevealed = ref(!props.enterFromSolar)
const EXIT_ELEMENTS_MS = 300
let leavingStartedAt = 0

/** 第一拍只清退轨道、飞行器与表面标记；天体本体由 host 的延迟 opacity 接力。 */
function exitElementsOpacity(now = performance.now()) {
  if (!props.leaving || leavingStartedAt === 0) return 1
  return THREE.MathUtils.clamp(1 - (now - leavingStartedAt) / EXIT_ELEMENTS_MS, 0, 1)
}

/** 入场自转（镜像火星 8014e13）：
 *  转速 14.4°/s（≈1.45s 转正），渐入开始时从 ±18° 偏角匀速转，
 *  角度剩减速位移时线性匀减速，终点 0°（初始姿态）。
 *  方向与太阳系场景一致（绕倾斜后的极轴正方向自转；金星的逆向由 177.4° 轴倾角表达）。 */
const SPIN_SPEED = THREE.MathUtils.degToRad(14.4) // ≈14.4°/s
const SPIN_DECEL_MS = 400 // 匀减速段
const SPIN_DECEL_SWEEP = (SPIN_SPEED * SPIN_DECEL_MS) / 2000 // ≈2.88°（匀减速位移）
// 预设偏角与速度方向相反（火星 -18° + 正速度 → 转回 0°）；金星 177.4° 轴倾角已表达逆向，
// 自转方向与太阳系一致（spinSign 恒为 +1，入场与持续方向统一为正方向）
const SPIN_OFFSET = -THREE.MathUtils.degToRad(18) * props.planet.spinSign
const craftMotionStartedAt = performance.now()
let spinPhase: 'spin' | 'stop' | 'done' = 'done'
let spinStartAt = 0
let spinStopAt = 0
let spinStopFrom = 0

watch(
  () => props.revealTick,
  (tick) => {
    if (tick) sceneRevealed.value = true
  },
)

function startSpin() {
  if (!swingPivot || spinPhase !== 'done') return
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  spinPhase = reduced ? 'done' : 'spin'
  spinStartAt = performance.now()
  swingPivot.rotation.y = SPIN_OFFSET
}
/** 入场自转推进：匀速（自西向东）→ 角度剩减速位移时线性匀减速 → 终点 0°（初始姿态） */
function updateSpin(now: number) {
  if (!swingPivot || spinPhase === 'done') return
  if (spinPhase === 'spin') {
    const t = Math.max(0, (now - spinStartAt) / 1000)
    swingPivot.rotation.y = SPIN_OFFSET + SPIN_SPEED * props.planet.spinSign * t
    // 终点精确落在 0°（初始姿态）：进入减速段时对齐精确阈值，不受帧偏差/后台标签页帧迟到影响
    if (Math.abs(swingPivot.rotation.y) <= SPIN_DECEL_SWEEP) {
      spinPhase = 'stop'
      spinStopAt = now
      spinStopFrom = -SPIN_DECEL_SWEEP * props.planet.spinSign
    }
  } else {
    const t = Math.min(1, (now - spinStopAt) / SPIN_DECEL_MS)
    swingPivot.rotation.y = spinStopFrom + SPIN_SPEED * props.planet.spinSign * (SPIN_DECEL_MS / 1000) * (t - (t * t) / 2)
    if (t >= 1) spinPhase = 'done'
  }
}

// 进入：星球渐入（scene-host）完成即启动入场自转（无航天器/着陆点，无元素弹出阶段）
watch(sceneRevealed, (revealed) => {
  if (!revealed) return
  startSpin()
})

/** 工具栏被页头"推下/推回"：rAF 逐帧插值（CSS transition 被系统减弱动态效果禁用，JS 动画不受影响） */
const sceneToolbarRef = ref<HTMLElement | null>(null)
let toolbarShift = 0
let toolbarAnim: number | undefined

/** 弹出信息卡随页头"推下/推回"（地球页 context-panel 同款：headerExpanded → translateY(76px)） */
const panelHeaderOffset = computed(() => (props.headerExpanded ? { '--header-overlay-offset': '76px' } : undefined))
watch(
  () => props.headerExpanded,
  (expanded) => {
    if (toolbarAnim !== undefined) cancelAnimationFrame(toolbarAnim)
    const target = expanded ? 76 : 0
    const from = toolbarShift
    const start = performance.now()
    const tick = (now: number) => {
      const t = Math.min(1, (now - start) / 380)
      const eased = 1 - Math.pow(1 - t, 3)
      toolbarShift = from + (target - from) * eased
      if (sceneToolbarRef.value) {
        sceneToolbarRef.value.style.transform = toolbarShift > 0.5 ? `translateY(${toolbarShift.toFixed(2)}px)` : ''
      }
      toolbarAnim = t < 1 ? requestAnimationFrame(tick) : undefined
    }
    toolbarAnim = requestAnimationFrame(tick)
  },
  { immediate: true },
)

let renderer: THREE.WebGLRenderer | undefined
let scene: THREE.Scene | undefined
let camera: THREE.PerspectiveCamera | undefined
let controls: OrbitControls | undefined
let planetMesh: THREE.Mesh | undefined
/** 自转轴：tiltPivot（真实轴倾角）→ swingPivot（入场自转绕倾斜后的极轴）→ 行星 */
let tiltPivot: THREE.Object3D | undefined
let swingPivot: THREE.Object3D | undefined
let planetMaterial: THREE.MeshStandardMaterial | THREE.MeshBasicMaterial | undefined
let ambientLight: THREE.AmbientLight | undefined
let sunLight: THREE.DirectionalLight | undefined
let observationLight: THREE.DirectionalLight | undefined
let resizeObserver: ResizeObserver | undefined
let frameId = 0
let lastSunUpdate = 0
/** 拖拽后注视点滑回行星中心 */
let dragResetTarget = false
/** 目录/标签触发的短弧运镜：与月球、火星一致，由主渲染循环推进。 */
let focusAnimation: {
  fromPos: THREE.Vector3
  toPos: THREE.Vector3
  startedAt: number
  duration: number
} | null = null
type CraftRuntime = { spec: PlanetCraft; line?: THREE.Line; dot: THREE.Mesh; hit: THREE.Mesh; path: THREE.Vector3[] }
const craftRuntimes = new Map<string, CraftRuntime>()
/** 表面任务标记在场景生命周期内唯一；目录聚焦与每帧标签投影共用同一真实 Three.js 对象。 */
const siteMarkers = new Map<string, THREE.Mesh>()

const FOV = 42
const DEG = Math.PI / 180
/** 统一加速时钟：1 秒代表 0.1 个地球日——运行中探测器按真实轨道周期绕行
 *  （Parker 88 天 ≈ 14.7 分钟一圈、BepiColombo 120 天 ≈ 20 分钟、Juno 53 天 ≈ 8.8 分钟），
 *  视觉周期与真实周期严格成比例（88:120:53），clamp 上限只防极端长周期，
 *  不在正常范围内截断比例。符合"视觉克制、不冒充实时"的原则。 */
const SIMULATED_DAYS_PER_SECOND = 0.1
const MIN_VISUAL_PERIOD_SECONDS = 30
const MAX_VISUAL_PERIOD_SECONDS = 3000

const pointerStart = new THREE.Vector2()

/** 判断世界坐标是否位于行星朝向相机的一侧。命中测试也复用它，避免透明拾取球让背面点可点击。 */
function isWorldPointFrontFacing(worldPoint: THREE.Vector3) {
  if (!camera || !tiltPivot) return false
  const center = new THREE.Vector3()
  const normal = worldPoint.clone().sub(tiltPivot.getWorldPosition(center)).normalize()
  const cameraForward = new THREE.Vector3()
  camera.getWorldDirection(cameraForward)
  return normal.dot(cameraForward) < -0.03
}function craftTrajectoryPosition(trajectory: PlanetCraftTrajectory, progress: number) {
  const radius = props.planet.radius * trajectory.radius
  const phase = THREE.MathUtils.degToRad(trajectory.phaseDeg ?? 0)
  const inclination = THREE.MathUtils.degToRad(trajectory.inclinationDeg ?? 0)
  if (trajectory.kind === 'flyby') {
    const span = THREE.MathUtils.degToRad(trajectory.spanDeg ?? 140)
    const angle = -span / 2 + span * progress
    // 飞掠弧线：圆心在行星中心，半径恒为 radius（> 行星半径），弧线全程在行星外侧。
    // 不能写成 radius*cos(angle) - radius——那是以 (0,0,-radius) 为圆心的圆弧，
    // 中点 (progress=0.5) 会落在行星中心 (0,0,0)，弧线直接从行星正中间穿过。
    const x = radius * Math.sin(angle)
    const z = radius * Math.cos(angle)
    const y = Math.sin(Math.PI * progress) * props.planet.radius * 0.42
    return new THREE.Vector3(x, y, z).applyAxisAngle(new THREE.Vector3(0, 1, 0), phase).applyAxisAngle(new THREE.Vector3(1, 0, 0), inclination)
  }
  const angle = phase + Math.PI * 2 * progress
  const eccentricity = THREE.MathUtils.clamp(trajectory.eccentricity ?? 0.12, 0, 0.85)
  const radial = radius * (1 - eccentricity * eccentricity) / (1 + eccentricity * Math.cos(angle - phase))
  return new THREE.Vector3(radial * Math.cos(angle), 0, -radial * Math.sin(angle))
    .applyAxisAngle(new THREE.Vector3(0, 1, 0), 0)
    .applyAxisAngle(new THREE.Vector3(1, 0, 0), inclination)
}

function trajectoryPoints(trajectory: PlanetCraftTrajectory) {
  const count = trajectory.kind === 'flyby' ? 96 : 144
  return Array.from({ length: count }, (_, index) => craftTrajectoryPosition(trajectory, index / (count - 1)))
}

function visualTrajectoryPeriodSeconds(trajectory: PlanetCraftTrajectory) {
  if (trajectory.periodDays != null) {
    return THREE.MathUtils.clamp(trajectory.periodDays / SIMULATED_DAYS_PER_SECOND, MIN_VISUAL_PERIOD_SECONDS, MAX_VISUAL_PERIOD_SECONDS)
  }
  return Math.max(MIN_VISUAL_PERIOD_SECONDS, trajectory.periodSeconds ?? 60)
}

/** 把 RingGeometry 的平面 UV 改写为径向条带 UV（u = 内缘 → 外缘），以匹配环带纹理 */
function radialRingGeometry(inner: number, outer: number, segments: number) {
  const geometry = new THREE.RingGeometry(inner, outer, segments, 1)
  const position = geometry.attributes.position as THREE.BufferAttribute
  const uv = geometry.attributes.uv as THREE.BufferAttribute
  for (let i = 0; i < position.count; i += 1) {
    const x = position.getX(i)
    const y = position.getY(i)
    const radius = Math.sqrt(x * x + y * y)
    uv.setXY(i, (radius - inner) / (outer - inner), 0.5)
  }
  return geometry
}

/** 程序化生成天王星环纹理：13 条细环（Zeta/6/5/4/α/β/η/γ/δ/λ/ε/ν/μ，由内到外），
 *  环间暗隙 + 细环亮线，模拟真实 Uranus ring system（NASA 命名）。
 *  输入为环径向位置（以行星半径为单位，内缘→外缘归一化 0..1），输出 CanvasTexture 供径向 UV 使用。 */
function proceduralUranusRingTexture(inner: number, outer: number): THREE.CanvasTexture {
  // 13 条环的真实径向位置（Uranus 半径单位；ε 环最亮、μ/ν 是外侧两条弱环）
  const ringPositions = [
    1.592, 1.604, 1.612, 1.625, 1.657, 1.681, 1.703, 1.723, 1.742, // 9 条内环（Zeta..δ）
    1.954, // ε（最亮）
    2.136, // ν
    2.377, // μ
  ]
  const width = 1024
  const height = 4
  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('canvas 2d unavailable')
  ctx.clearRect(0, 0, width, height)
  const span = outer - inner
  for (const r of ringPositions) {
    const t = (r - inner) / span // 0..1 归一化
    if (t < -0.02 || t > 1.02) continue
    const x = Math.round(t * (width - 1))
    // 每条环 2-3px 宽（ε 环更宽更亮）；细环线用半透明白，α 通道决定亮度
    const isEpsilon = Math.abs(r - 1.954) < 0.01
    const isOuter = r > 2.1
    const bandWidth = isEpsilon ? 6 : 3
    const alpha = isEpsilon ? 0.95 : isOuter ? 0.4 : 0.75
    ctx.fillStyle = `rgba(210, 225, 235, ${alpha})`
    ctx.fillRect(x - bandWidth / 2, 0, bandWidth, height)
  }
  // ε 环外缘的微弱晕（稀薄尘埃）
  ctx.fillStyle = 'rgba(180, 200, 215, 0.12)'
  ctx.fillRect(Math.round(((1.95 - inner) / span) * width), 0, Math.round(((2.05 - inner) / span) * width), height)
  const texture = new THREE.CanvasTexture(canvas)
  texture.wrapS = THREE.ClampToEdgeWrapping
  texture.colorSpace = THREE.SRGBColorSpace
  return texture
}

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
  camera = new THREE.PerspectiveCamera(FOV, initialWidth / initialHeight, 0.1, 2000)
  // 初始视角：距行星中心 defaultDistance（视半径与火星页接近；土星带环略远保证环完整入画）
  camera.position.set(0, 1.8, props.planet.defaultDistance)

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

  controls = new OrbitControls(camera, renderer.domElement)
  controls.enableDamping = true
  controls.dampingFactor = 0.06
  controls.enablePan = false
  controls.enableZoom = false
  controls.addEventListener('start', () => {
    dragResetTarget = true
  })
  // 拉近极限：视半径上限与火星一致（45.8°，即近限 = 半径×1.4）；有环行星近限必须 > 环外缘，
  // 避免镜头穿入环平面造成"环横贯画面"的观感（土星环外缘 6.6×2.33≈15.4，天王星环 4.3×2.4≈10.3）
  controls.minDistance = Math.max(
    props.planet.radius * 1.4,
    props.planet.ring ? props.planet.radius * props.planet.ring.outer * 1.08 : 0,
  )
  // 缩到最远：保持"缩到最远视大小统一"（约 8.5°，即最远 = 半径×6.7，与地球 12/2.15 同档）；
  // 且不小于默认距离（水星 1.33×6.7≈8.9 < 默认 9，OrbitControls 要求 max ≥ 当前位置）
  controls.maxDistance = Math.max(props.planet.radius * 6.7, props.planet.defaultDistance)

  // 行星本体：行星用 PBR 材质（受光照，晨昏线依赖明暗）；恒星（太阳）用自发光 Basic 材质（不受光照）
  const texture = solarTexture(props.planet.textureUrl, () => emitTexturesReady())
  texture.colorSpace = THREE.SRGBColorSpace
  texture.anisotropy = 16
  planetMaterial = props.planet.star
    ? new THREE.MeshBasicMaterial({ map: texture })
    : new THREE.MeshStandardMaterial({ map: texture, roughness: 0.95, metalness: 0.02 })
  planetMesh = new THREE.Mesh(new THREE.SphereGeometry(props.planet.radius, 256, 256), planetMaterial)
  // 初始朝向：绕自转轴（局部 Y）旋转，让 lon 0° 子午线朝向相机——
  // 用绕 Y 轴的四元数（北极保持在局部 +Y = 自转轴，不产生极轴漂移；
  // 不能 setFromUnitVectors((1,0,0), cameraDir)——那会把北极也转离自转轴）
  planetMesh.quaternion.setFromAxisAngle(
    new THREE.Vector3(0, 1, 0),
    Math.atan2(-camera.position.z, camera.position.x) + THREE.MathUtils.degToRad(props.planet.surfaceYawDeg ?? 0),
  )

  // 轴倾角组（真实轴倾角；金星 177.4° = 倒置自转轴）→ 自转组（入场自转绕倾斜后的极轴）
  tiltPivot = new THREE.Object3D()
  tiltPivot.name = `${props.planet.key}-tilt-pivot`
  tiltPivot.rotation.order = 'ZYX'
  tiltPivot.rotation.z = props.planet.axialTiltDeg * DEG
  swingPivot = new THREE.Object3D()
  swingPivot.name = `${props.planet.key}-swing-pivot`
  swingPivot.add(planetMesh)

  // 行星环：土星用环带纹理（径向条带 UV），天王星用程序化 13 细环纹理（模拟真实环系）。
  // 挂在自转组下随轴倾角倾斜（与太阳系场景同实现）。
  if (props.planet.ring) {
    const inner = props.planet.radius * props.planet.ring.inner
    const outer = props.planet.radius * props.planet.ring.outer
    const ringGeometry = radialRingGeometry(inner, outer, 256)
    const ringMaterial = new THREE.MeshBasicMaterial({
      color: props.planet.ring.color ?? 0xd8c9a3,
      transparent: true,
      // 土星环贴图未就绪时不显示材质底色，避免直达页面首帧出现实心圆盘。
      opacity: props.planet.ring.kind === 'saturn' && props.planet.ring.textureUrl
        ? 0
        : (props.planet.ring.opacity ?? 0.9),
      side: THREE.DoubleSide,
      depthWrite: false,
    })
    const ringMesh = new THREE.Mesh(ringGeometry, ringMaterial)
    ringMesh.name = `${props.planet.key}-ring`
    if (props.planet.ring.kind === 'uranus') {
      // 程序化 13 细环：Canvas 生成（真实环系位置），无外部素材依赖
      const ringTexture = proceduralUranusRingTexture(props.planet.ring.inner, props.planet.ring.outer)
      ringMaterial.map = ringTexture
      ringMaterial.needsUpdate = true
    } else if (props.planet.ring.textureUrl) {
      const ringTexture = solarTexture(props.planet.ring.textureUrl, (t: THREE.Texture) => {
        if (!ringMesh.material) return
        ringMaterial.map = t
        ringMaterial.opacity = props.planet.ring?.opacity ?? 0.9
        ringMaterial.needsUpdate = true
      })
      ringTexture.colorSpace = THREE.SRGBColorSpace
      ringTexture.anisotropy = 4
      // 即使预加载纹理仍在解码，也先绑定同一个 Texture 实例；加载完成后 Three.js 会自动上传图像。
      // 否则直接通过 hash 进入土星页时，环会短暂甚至持续显示成一整块纯色圆盘。
      ringMaterial.map = ringTexture
      ringMaterial.needsUpdate = true
    }
    swingPivot.add(ringMesh)
  }

  tiltPivot.add(swingPivot)
  scene.add(tiltPivot)

  // 探测器轨迹：轨道/飞掠线与行星共用深度缓冲，背面自然被球体遮挡。
  for (const craft of planetCrafts.value) {
    const trajectory = craft.trajectory
    if (!trajectory) continue
    const path = trajectoryPoints(trajectory)
    const line = new THREE.Line(
      new THREE.BufferGeometry().setFromPoints(path),
      new THREE.LineBasicMaterial({ color: props.planet.sceneAccent, transparent: true, opacity: 0.34, depthTest: true, depthWrite: false }),
    )
    line.name = `${craft.id}-trajectory`
    line.visible = orbitsEnabled.value
    line.userData = { kind: 'planet-trajectory', craftId: craft.id }
    swingPivot.add(line)
    const dot = new THREE.Mesh(
      new THREE.SphereGeometry(1, 12, 12),
      new THREE.MeshBasicMaterial({
        color: props.planet.star ? 0xfff0c2 : (craft.status === '运行中' ? props.planet.sceneAccent : 0xd7e4ea),
        transparent: true,
        opacity: 0.96,
        // 太阳页的轨道是压缩示意图：自发光太阳盘会遮掉全部近太阳标记，故以 HUD 标记置顶；
        // 普通行星仍保持真实深度遮挡，背面飞行器不可见。
        depthTest: !props.planet.star,
        depthWrite: false,
      }),
    )
    dot.userData = { kind: 'planet-craft', craftId: craft.id }
    const displayProgress = THREE.MathUtils.clamp(
      trajectory.displayProgress ?? (craft.status === '运行中' || craft.status === '即将入轨' ? 12 / (path.length - 1) : 0.78),
      0,
      1,
    )
    dot.position.copy(path[Math.min(path.length - 1, Math.floor(path.length * displayProgress))])
    // 视觉圆点保持克制；透明拾取球沿用月球/火星，确保鼠标命中不依赖像素级精度。
    const hit = new THREE.Mesh(
      new THREE.SphereGeometry(4, 10, 10),
      new THREE.MeshBasicMaterial({ transparent: true, opacity: 0, depthWrite: false }),
    )
    hit.userData = { kind: 'planet-craft-hit', craftId: craft.id }
    dot.add(hit)
    swingPivot.add(dot)
    // 大气终点（如 Magellan/Pioneer Venus 坠入金星大气）无真实经纬度，只入目录与信息卡
    // （endpoint 文案），不画 3D 标记——避免行星表面出现无标签的悬浮圆点（"残留点"）
    craftRuntimes.set(craft.id, { spec: craft, line, dot, hit, path })
  }
  watch(orbitsEnabled, (enabled) => {
    for (const runtime of craftRuntimes.values()) runtime.line && (runtime.line.visible = enabled)
  })
  watch(spacecraftEnabled, (enabled) => {
    for (const runtime of craftRuntimes.values()) runtime.dot.visible = enabled
    if (!enabled) selectedCraft.value = null
  })

  // 轨道高亮（地球页同款交互）：悬停优先、其次选中——飞行器轨道线提亮 + 圆点放大；
  // 悬停移开恢复选中状态（若有），无选中则恢复默认（线 0.34、圆点原尺寸）
  const LINE_BASE_OPACITY = 0.34
  const LINE_ACTIVE_OPACITY = 0.95
  const applyCraftHighlight = () => {
    const activeId = hoveredCraft.value ?? selectedCraft.value
    for (const runtime of craftRuntimes.values()) {
      const lineMat = runtime.line?.material as THREE.LineBasicMaterial | undefined
      if (lineMat) lineMat.opacity = activeId === runtime.spec.id ? LINE_ACTIVE_OPACITY : LINE_BASE_OPACITY
    }
  }
  watch([hoveredCraft, selectedCraft], () => applyCraftHighlight())
  applyCraftHighlight()

  // 人类足迹标记（着陆点/撞击点）：小圆球嵌在行星表面（与火星着陆点同实现）。
  // 挂在 swingPivot 下随行星自转/轴倾角；大气坠毁（气态行星无表面坐标）不画标记
  for (const site of markerSites.value) {
    const marker = new THREE.Mesh(
      new THREE.SphereGeometry(1, 12, 12),
      new THREE.MeshBasicMaterial({ color: props.planet.sceneAccent, transparent: true, opacity: 0.95 }),
    )
    // 球心落在行星表面半径上：球体一半嵌进表面、一半露出（被行星深度遮挡，无 z-fighting）
    marker.position.copy(sitePosition(site.latitude!, site.longitude!, props.planet.radius))
    marker.userData = { kind: 'planet-site', siteId: site.id }
    swingPivot.add(marker)
    siteMarkers.set(site.id, marker)
    // 拾取球：扩大点击命中区域
    const hit = new THREE.Mesh(
      new THREE.SphereGeometry(7, 8, 8),
      new THREE.MeshBasicMaterial({ transparent: true, opacity: 0, depthWrite: false }),
    )
    hit.userData.siteId = site.id
    marker.add(hit)
  }
  // 开关联动：隐藏/显示标记
  watch(sitesEnabled, (enabled) => {
    for (const marker of siteMarkers.values()) marker.visible = enabled
    if (!enabled) selectedSite.value = null
  })

  // 标签 overlay 投影更新（每帧）：标记的世界坐标 → 屏幕坐标。
  // 关键点：标签可见性用"视线遮挡"判断（地球页 isOccludedByEarth 同款）——
  // 从相机向目标发射线，若与行星球体相交且在目标之前 → 被行星遮挡 → 隐藏。
  // 不能只用球面法线朝相机：轨道飞行器在水星边界附近时法线接近垂直视线（dot≈0），
  // 圆点明明在球体外可见，标签却被 -0.03 阈值误判为背面而隐藏。
  const labelTmp = new THREE.Vector3()
  const labelWorld = new THREE.Vector3()
  const labelOcclusionDir = new THREE.Vector3()
  const labelRay = new THREE.Raycaster()
  const labelOcclusionSphere = new THREE.Sphere(new THREE.Vector3(0, 0, 0), props.planet.radius)
  const labelOcclusionHit = new THREE.Vector3()
  const isNotOccluded = (worldPoint: THREE.Vector3) => {
    if (!camera) return false
    // 用独立临时向量计算方向，绝不就地修改传入的 worldPoint——
    // 否则调用方（如 site 标签的 marker.getWorldPosition(labelWorld) 后传入）的
    // 世界坐标会被减成"相对相机的向量"，后续投影全错（标签落到视口中心/行星中间）
    const toTarget = labelOcclusionDir.copy(worldPoint).sub(camera.position)
    const targetDistance = toTarget.length()
    if (targetDistance === 0) return false
    labelRay.set(camera.position, toTarget.normalize())
    const hit = labelRay.ray.intersectSphere(labelOcclusionSphere, labelOcclusionHit)
    if (!hit) return true
    return camera.position.distanceTo(labelOcclusionHit) >= targetDistance - 0.035
  }
  const updateSiteLabels = () => {
    if (!renderer || !camera || !sitesEnabled.value) {
      siteLabels.value = []
      return
    }
    const cam = camera
    const bounds = renderer.domElement.getBoundingClientRect()
    const rawLabels = markerSites.value
      .map((site) => {
        const marker = siteMarkers.get(site.id)
        if (!marker) return null
        marker.getWorldPosition(labelWorld)
        if (!isNotOccluded(labelWorld)) return null
        labelTmp.copy(labelWorld)
        labelTmp.project(cam)
        if (labelTmp.z > 1) return null
        return {
          id: site.id,
          name: site.name,
          nameEn: site.nameEn,
          mission: site.mission,
          type: siteKindLabel(site.kind),
          icon: site.icon,
          anchorX: (labelTmp.x * 0.5 + 0.5) * bounds.width,
          anchorY: (-labelTmp.y * 0.5 + 0.5) * bounds.height,
          visible: true,
          selected: selectedSite.value === site.id,
        }
      })
      .filter((l): l is NonNullable<typeof l> => l !== null)
    const currentPlanetRadiusPx = projectedSphereRadiusPx(props.planet.radius, cam.position.length(), FOV, bounds.height)
    const referencePlanetRadiusPx = projectedSphereRadiusPx(props.planet.radius, props.planet.defaultDistance, FOV, bounds.height)
    siteLabels.value = layoutSceneAnnotations(rawLabels, {
      width: bounds.width,
      height: bounds.height,
      currentPlanetRadiusPx,
      referencePlanetRadiusPx,
    }, siteLabels.value)
  }
  const updateCraftLabels = () => {
    if (!renderer || !camera || !spacecraftEnabled.value) {
      craftLabels.value = []
      return
    }
    const cam = camera
    const bounds = renderer.domElement.getBoundingClientRect()
    const rawLabels = Array.from(craftRuntimes.values()).map((runtime) => {
      runtime.dot.getWorldPosition(labelTmp)
      // 太阳页与圆点采用同一 HUD 语义，标签不被自发光球体吞掉；其他行星继续做球体遮挡判断。
      const notOccluded = props.planet.star || isNotOccluded(labelTmp)
      labelTmp.project(cam)
      const anchorX = (labelTmp.x * 0.5 + 0.5) * bounds.width
      const anchorY = (-labelTmp.y * 0.5 + 0.5) * bounds.height
      return {
        id: runtime.spec.id,
        name: runtime.spec.name,
        nameEn: runtime.spec.nameEn,
        type: runtime.spec.status === '运行中' ? '运行中' : runtime.spec.type,
        anchorX,
        anchorY,
        visible: labelTmp.z <= 1 && notOccluded,
        selected: selectedCraft.value === runtime.spec.id,
        hovered: hoveredCraft.value === runtime.spec.id,
      }
    })
    const currentPlanetRadiusPx = projectedSphereRadiusPx(props.planet.radius, cam.position.length(), FOV, bounds.height)
    const referencePlanetRadiusPx = projectedSphereRadiusPx(props.planet.radius, props.planet.defaultDistance, FOV, bounds.height)
    craftLabels.value = layoutSceneAnnotations(rawLabels, {
      width: bounds.width,
      height: bounds.height,
      currentPlanetRadiusPx,
      referencePlanetRadiusPx,
    }, craftLabels.value)
  }

  // 光照：行星用固定环境光 + 太阳方向光 + 跟随相机的观测光（360° 全亮，无晨昏线）；
  // 恒星（太阳）自发光，无需任何光照
  if (!props.planet.star) {
    ambientLight = new THREE.AmbientLight(0x3a2a22, 0.8)
    scene.add(ambientLight)
    observationLight = new THREE.DirectionalLight(0xfff3dd, props.planet.observationLightIntensity ?? 3.1)
    observationLight.position.copy(camera.position)
    scene.add(observationLight)
    sunLight = new THREE.DirectionalLight(0xfff3dd, 0)
    sunLight.position.set(-6, 4, 8)
    scene.add(sunLight)
  }

  // 星空粒子球（镜像地球/火星）：3000 颗、壳层 60–150
  const starGeometry = new THREE.BufferGeometry()
  const starData: number[] = []
  for (let index = 0; index < 3000; index += 1) {
    const radius = 60 + Math.random() * 90
    const theta = Math.random() * Math.PI * 2
    const phi = Math.acos(2 * Math.random() - 1)
    starData.push(radius * Math.sin(phi) * Math.cos(theta), radius * Math.cos(phi), radius * Math.sin(phi) * Math.sin(theta))
  }
  starGeometry.setAttribute('position', new THREE.Float32BufferAttribute(starData, 3))
  // 背景星空挂在相机上：屏幕固定，不随星球/相机旋转
  const starPoints = new THREE.Points(starGeometry, new THREE.PointsMaterial({ color: 0xc7d1db, size: 0.15, transparent: true, opacity: 0.75 }))
  starPoints.name = 'background-stars'
  scene.add(camera)
  camera.add(starPoints)

  renderer.render(scene, camera)

  let lastTime = performance.now()
  const animate = () => {
    frameId = requestAnimationFrame(animate)
    if (!renderer || !scene || !camera) return
    const now = performance.now()
    lastTime = now
    const elementsOpacity = exitElementsOpacity(now)

    // 入场自转（按行星真实自转方向，停稳后静止）
    updateSpin(now)

    // 观测光跟随相机：明暗边界始终落在球体轮廓之外（关闭晨昏线时 360° 全亮）
    if (observationLight && camera) observationLight.position.copy(camera.position)
    // 晨昏线开启：太阳方向按行星太阳日持续推进（每 60s 刷新，与地球/火星同节奏）
    if (terminatorEnabled.value && sunLight && now - lastSunUpdate > 60_000) {
      lastSunUpdate = now
      sunLight.position.copy(sunDirection()).multiplyScalar(10)
    }

    // 拖拽恢复：注视点滑回行星中心
    if (controls && camera) {
      if (dragResetTarget) {
        controls.target.lerp(origin, 0.12)
        if (controls.target.lengthSq() < 0.002) {
          controls.target.set(0, 0, 0)
          dragResetTarget = false
        }
      }
      // 动态拖动灵敏度（与地球/火星一致）：近处降敏、远处提速
      const t = THREE.MathUtils.clamp((camera.position.length() - controls.minDistance) / (controls.maxDistance - controls.minDistance), 0, 1)
      controls.rotateSpeed = 0.2 + t * 0.5
    }

    // 目录点击后的聚焦：按相机球面方向插值，不直穿行星；结束后恢复正常拖拽。
    if (focusAnimation && camera && controls) {
      const t = Math.min(1, (now - focusAnimation.startedAt) / focusAnimation.duration)
      const eased = 1 - Math.pow(1 - t, 3)
      const direction = focusTmp2
        .copy(focusAnimation.fromPos)
        .normalize()
        .lerp(focusTmp3.copy(focusAnimation.toPos).normalize(), eased)
        .normalize()
      const distance = THREE.MathUtils.lerp(focusAnimation.fromPos.length(), focusAnimation.toPos.length(), eased)
      camera.position.copy(direction.multiplyScalar(distance))
      camera.lookAt(origin)
      controls.target.multiplyScalar(1 - eased)
      if (t >= 1) {
        focusAnimation = null
        controls.target.copy(origin)
        controls.enabled = true
        controls.update()
      }
    }

    const markerViewportHeight = renderer.domElement.clientHeight
    const currentPlanetRadiusPx = projectedSphereRadiusPx(props.planet.radius, camera.position.length(), FOV, markerViewportHeight)
    const referencePlanetRadiusPx = projectedSphereRadiusPx(props.planet.radius, props.planet.defaultDistance, FOV, markerViewportHeight)
    for (const runtime of craftRuntimes.values()) {
      const trajectory = runtime.spec.trajectory
      if (!trajectory) continue
      if ((runtime.spec.status === '运行中' || runtime.spec.status === '即将入轨') && trajectory.kind === 'orbit') {
        const period = visualTrajectoryPeriodSeconds(trajectory)
        const progress = trajectory.displayProgress == null
          ? (now / 1000 / period) % 1
          : (trajectory.displayProgress + (now - craftMotionStartedAt) / 1000 / period) % 1
        runtime.dot.position.copy(craftTrajectoryPosition(trajectory, progress))
      } else {
        // 已结束的飞行器：圆点固定停在自己的轨道/弧线上（path 78% 处）——
        // 不能把圆点覆盖到大气终点（endpoint），否则飞行器会脱离轨道、轨道上看起来没有对应点
        const displayProgress = THREE.MathUtils.clamp(trajectory.displayProgress ?? 0.78, 0, 1)
        runtime.dot.position.copy(runtime.path[Math.min(runtime.path.length - 1, Math.floor(runtime.path.length * displayProgress))])
      }
      const activeId = hoveredCraft.value ?? selectedCraft.value
      const world = runtime.dot.getWorldPosition(focusTmp4)
      runtime.dot.scale.setScalar(sceneMarkerWorldRadius(
        world.distanceTo(camera.position),
        FOV,
        markerViewportHeight,
        orbitMarkerRadiusPx(currentPlanetRadiusPx, referencePlanetRadiusPx),
      ))
      const dotMat = runtime.dot.material as THREE.MeshBasicMaterial
      dotMat.opacity = 0.96 * elementsOpacity
      const lineMat = runtime.line?.material as THREE.LineBasicMaterial | undefined
      if (lineMat) lineMat.opacity = (activeId === runtime.spec.id ? LINE_ACTIVE_OPACITY : LINE_BASE_OPACITY) * elementsOpacity
      runtime.dot.visible = spacecraftEnabled.value && elementsOpacity > 0.001
      if (runtime.line) runtime.line.visible = orbitsEnabled.value && elementsOpacity > 0.001
    }

    for (const [id, marker] of siteMarkers) {
      const world = marker.getWorldPosition(focusTmp4)
      const markerRadiusPx = surfaceMarkerRadiusPx(currentPlanetRadiusPx, referencePlanetRadiusPx, selectedSite.value === id)
      marker.scale.setScalar(surfaceMarkerWorldRadius(world.distanceTo(camera.position), FOV, markerViewportHeight, markerRadiusPx))
      const markerMat = marker.material as THREE.MeshBasicMaterial
      markerMat.opacity = 0.95 * elementsOpacity
      marker.visible = sitesEnabled.value && elementsOpacity > 0.001
    }

    controls?.update()
    // 相机阻尼更新后再投影标签，避免飞行器高速移动时标签滞后一帧显得离点很远。
    updateSiteLabels()
    updateCraftLabels()
    renderer.render(scene, camera)
  }
  animate()
})

/** 经纬度 → 球面坐标（与地球页 latLonToVector / 火星 sitePosition 同公式） */
function sitePosition(latitude: number, longitude: number, radius: number) {
  const lat = latitude * DEG
  const lon = longitude * DEG
  return new THREE.Vector3(
    radius * Math.cos(lat) * Math.cos(lon),
    radius * Math.sin(lat),
    -radius * Math.cos(lat) * Math.sin(lon),
  )
}

/** 真实太阳方向（行星参数）：按当前日期/时刻计算太阳在行星固连坐标系中的方向
 *  （北 = +y），用于晨昏线——子日点黄经按太阳日推进（360/太阳日每小时），
 *  赤纬按行星季节（黄赤交角 · sin 季节相位，锚定真实春分/近似锚点）；
 *  经行星网格世界四元数变换，晨昏线落在行星正确位置（镜像火星 marsSunDirection） */
function sunDirection(): THREE.Vector3 {
  const now = new Date()
  const sun = props.planet.sun
  // 等效季节倾角：顺行行星（轴倾角 ≤90°）直接用轴倾角（水星 0.03°/火星 25.19°/土星 26.73°）；
  // 逆向行星（轴倾角 >90°，金星 177.4°/天王星 97.77°）用 |180-轴倾角|（金星 2.64°/天王星 82.23°，季节反转）
  const effectiveTilt = Math.abs(sun.axialTiltDeg) > 90 ? 180 - Math.abs(sun.axialTiltDeg) : Math.abs(sun.axialTiltDeg)
  const daysSinceAnchor = (now.getTime() - sun.seasonAnchorMs) / 86_400_000
  const declination = effectiveTilt * Math.sin(DEG * ((360 / sun.seasonPeriodDays) * daysSinceAnchor))
  const utcHours = now.getUTCHours() + now.getUTCMinutes() / 60 + now.getUTCSeconds() / 3600
  const subsolarLongitude = 180 - utcHours * (360 / sun.solarDayHours)
  const dir = sitePosition(declination, subsolarLongitude, 10)
  const q = new THREE.Quaternion()
  if (planetMesh) planetMesh.getWorldQuaternion(q)
  return dir.applyQuaternion(q)
}

watch(terminatorEnabled, (enabled) => {
  if (!observationLight || !sunLight) return
  if (props.planet.star) return // 恒星自发光，无昼夜
  observationLight.intensity = enabled ? 0 : 3.1
  sunLight.intensity = enabled ? 3.1 : 0
  if (enabled) {
    // 真实昼夜方向：晨昏线位置 = 此刻太阳方位（严格按时间，昼夜随行星太阳日推进）
    sunLight.position.copy(sunDirection()).multiplyScalar(10)
  }
})

// 返回太阳系：第一拍清退轨道、飞行器、足迹与界面信息，第二拍由 host 延迟渐隐天体本身。
// 离开被中止时 leaving 回 false，透明度计算和 host class 都会恢复。
watch(
  () => props.leaving,
  (leaving) => {
    if (!leaving) {
      leavingStartedAt = 0
      return
    }
    leavingStartedAt = performance.now()
    hoveredCraft.value = null
    spinPhase = 'done' // 退出时若入场自转仍在进行，立即停住（避免返回过渡期间继续转）
  },
)

/** 鼠标移动：悬停飞行器圆点 → 轨道线高亮（地球页同款；标签 hover 走模板事件）。
 *  拖拽中不更新避免闪烁（与地球页一致）。 */
function onPointerMove(event: PointerEvent) {
  if (event.buttons !== 0 || !renderer || !camera || !swingPivot || !spacecraftEnabled.value) return
  const rect = renderer.domElement.getBoundingClientRect()
  const ndc = new THREE.Vector2(((event.clientX - rect.left) / rect.width) * 2 - 1, -((event.clientY - rect.top) / rect.height) * 2 + 1)
  const raycaster = new THREE.Raycaster()
  raycaster.setFromCamera(ndc, camera)
  const hits = raycaster.intersectObjects(swingPivot.children, true)
  for (const hit of hits) {
    let obj: THREE.Object3D | null = hit.object
    while (obj) {
      // 只允许圆点/拾取球触发 hover（kind: planet-craft / planet-craft-hit）——
      // 轨道线（planet-trajectory）虽然也带 craftId，但整条线太长，
      // 远处轻轻碰到轨道任意位置就会高亮，触发范围过大
      const kind = obj.userData?.kind as string | undefined
      const craftId = obj.userData?.craftId as string | undefined
      if (craftId && (kind === 'planet-craft' || kind === 'planet-craft-hit')) {
        hoveredCraft.value = craftId
        return
      }
      obj = obj.parent
    }
  }
  if (hoveredCraft.value) hoveredCraft.value = null
}

/** 按下瞬间：在行星表面 → 立即收起页头（与地球/火星一致） */
function onPointerDown(event: PointerEvent) {
  pointerStart.set(event.clientX, event.clientY)
  if (isNearPlanet(event.clientX, event.clientY)) {
    emit('blank-click')
  }
}

/** 松开：未拖拽（点按）→ 拾取着陆点/撞击点标记，否则空白处清除并收起页头 */
function onPointerUp(event: PointerEvent) {
  if (!renderer || !camera) return
  if (pointerStart.distanceTo(new THREE.Vector2(event.clientX, event.clientY)) > 5) return
  // 拾取：优先命中探测器/足迹标记（与月球/火星同款，背面仍由 Three.js 深度测试挡住）
  const rect = renderer.domElement.getBoundingClientRect()
  const ndc = new THREE.Vector2(((event.clientX - rect.left) / rect.width) * 2 - 1, -((event.clientY - rect.top) / rect.height) * 2 + 1)
  const raycaster = new THREE.Raycaster()
  raycaster.setFromCamera(ndc, camera)
  const hits = raycaster.intersectObjects(swingPivot ? swingPivot.children : [], true)
  for (const hit of hits) {
    let obj: THREE.Object3D | null = hit.object
    while (obj) {
      const craftId = obj.userData?.craftId as string | undefined
      if (craftId && spacecraftEnabled.value) {
        const runtime = craftRuntimes.get(craftId)
        const point = new THREE.Vector3()
        runtime?.dot.getWorldPosition(point)
        if (!runtime || !isWorldPointFrontFacing(point)) {
          obj = obj.parent
          continue
        }
        selectCraft(craftId)
        return
      }
      const siteId = obj.userData?.siteId as string | undefined
      if (siteId && sitesEnabled.value) {
        const point = new THREE.Vector3()
        obj.getWorldPosition(point)
        if (!isWorldPointFrontFacing(point)) {
          obj = obj.parent
          continue
        }
        selectSite(siteId)
        return
      }
      obj = obj.parent
    }
  }
  selectedCraft.value = null
  selectedSite.value = null
  focusAnimation = null
  if (controls) controls.enabled = true
  emit('blank-click')
}

/** 鼠标是否在行星投影范围内（镜像火星 isNearMars） */
function isNearPlanet(clientX: number, clientY: number) {
  if (!renderer || !camera) return false
  const bounds = renderer.domElement.getBoundingClientRect()
  const projectedCenter = new THREE.Vector3(0, 0, 0).project(camera)
  const cameraRight = new THREE.Vector3(1, 0, 0)
    .applyQuaternion(camera.quaternion)
    .multiplyScalar(props.planet.radius * 1.08)
    .project(camera)
  const centerX = bounds.left + (projectedCenter.x * 0.5 + 0.5) * bounds.width
  const centerY = bounds.top + (-projectedCenter.y * 0.5 + 0.5) * bounds.height
  const radius = Math.abs(cameraRight.x - projectedCenter.x) * bounds.width * 0.5
  return Math.hypot(clientX - centerX, clientY - centerY) <= radius * 1.12
}

/** 滚轮：在行星上 → 缩放行星；在边缘区域 → 交给页面滚动（与地球/火星一致） */
function onSceneWheel(event: WheelEvent) {
  if (!camera || !controls || !isNearPlanet(event.clientX, event.clientY)) return
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

onBeforeUnmount(() => {
  if (focusTimer !== undefined) clearTimeout(focusTimer)
  cancelAnimationFrame(frameId)
  resizeObserver?.disconnect()
  renderer?.domElement.removeEventListener('wheel', onSceneWheel)
  renderer?.domElement.removeEventListener('pointerdown', onPointerDown)
  renderer?.domElement.removeEventListener('pointerup', onPointerUp)
  renderer?.domElement.removeEventListener('pointermove', onPointerMove)
  controls?.dispose()
  planetMaterial?.dispose()
  planetMesh?.geometry.dispose()
  for (const runtime of craftRuntimes.values()) {
    runtime.line?.geometry.dispose()
    ;(runtime.line?.material as THREE.Material | undefined)?.dispose()
    runtime.dot.geometry.dispose()
    ;(runtime.dot.material as THREE.Material).dispose()
    runtime.hit.geometry.dispose()
    ;(runtime.hit.material as THREE.Material).dispose()
  }
  craftRuntimes.clear()
  for (const marker of siteMarkers.values()) {
    marker.geometry.dispose()
    ;(marker.material as THREE.Material).dispose()
    for (const child of marker.children) {
      if (child instanceof THREE.Mesh) {
        child.geometry.dispose()
        ;(child.material as THREE.Material).dispose()
      }
    }
  }
  siteMarkers.clear()
  renderer?.dispose()
})
</script>

<style scoped>
/* 行星主题色 CSS 变量：场景区（.planet-section）与档案区（.planet-profile-section）为兄弟节点，
   两者都挂 themeKey class，变量在两处均可解析 */
.planet-section,
.planet-profile-section,
.planet-spacecraft-section,
.planet-sites-section {
  --planet-accent: #f0e0b2;
  --planet-accent-dim: rgba(240, 224, 178, .4);
  --planet-line: rgba(240, 224, 178, .22);
  --planet-text: #f7efd8;
  --planet-quiet: #c4b184;
}
.planet-section {
  --mission-accent: var(--planet-accent);
  --mission-accent-dim: var(--planet-accent-dim);
  --mission-line: var(--planet-line);
  --mission-text: var(--planet-text);
  --mission-quiet: var(--planet-quiet);
  --mission-body: color-mix(in srgb, var(--planet-text) 72%, var(--planet-quiet));
  --mission-panel-surface: color-mix(in srgb, var(--planet-accent) 4%, rgba(4, 9, 15, .95));
  --mission-label-surface: color-mix(in srgb, var(--planet-accent) 3%, rgba(3, 10, 17, .82));
  --mission-label-surface-active: color-mix(in srgb, var(--planet-accent) 8%, rgba(6, 17, 26, .92));
}
.planet-section.venus {
  --mission-label-text-stroke: 1.5px #000;
  --mission-label-text-shadow: 0 1px 2px #000;
}
.planet-section.saturn,
.planet-profile-section.saturn,
.planet-spacecraft-section.saturn,
.planet-sites-section.saturn {
  --planet-accent: #e7c987;
  --planet-accent-dim: rgba(231, 201, 135, .4);
  --planet-line: rgba(231, 201, 135, .22);
  --planet-text: #f5ead0;
  --planet-quiet: #b39c6e;
}
.planet-section.jupiter,
.planet-profile-section.jupiter,
.planet-spacecraft-section.jupiter,
.planet-sites-section.jupiter {
  --planet-accent: #cf9257;
  --planet-accent-dim: rgba(207, 146, 87, .4);
  --planet-line: rgba(207, 146, 87, .22);
  --planet-text: #f0ddc4;
  --planet-quiet: #a37c50;
}
.planet-section.mercury,
.planet-profile-section.mercury,
.planet-spacecraft-section.mercury,
.planet-sites-section.mercury {
  --planet-accent: #c8b8a0;
  --planet-accent-dim: rgba(200, 184, 160, .4);
  --planet-line: rgba(200, 184, 160, .22);
  --planet-text: #efe6d8;
  --planet-quiet: #b3a48f;
}
.planet-section.uranus,
.planet-profile-section.uranus,
.planet-spacecraft-section.uranus,
.planet-sites-section.uranus {
  --planet-accent: #8fd8d0;
  --planet-accent-dim: rgba(143, 216, 208, .4);
  --planet-line: rgba(143, 216, 208, .22);
  --planet-text: #d8f2ee;
  --planet-quiet: #8fb8b4;
}
.planet-section.neptune,
.planet-profile-section.neptune,
.planet-spacecraft-section.neptune,
.planet-sites-section.neptune {
  --planet-accent: #6aa8e0;
  --planet-accent-dim: rgba(106, 168, 224, .4);
  --planet-line: rgba(106, 168, 224, .22);
  --planet-text: #d8e8f8;
  --planet-quiet: #8fb0d4;
}
.planet-section.sun,
.planet-profile-section.sun,
.planet-spacecraft-section.sun,
.planet-sites-section.sun {
  --planet-accent: #ff6b4a;
  --planet-accent-dim: rgba(255, 107, 74, .4);
  --planet-line: rgba(255, 107, 74, .22);
  --planet-text: #ffe3d6;
  --planet-quiet: #e0957d;
}

/* 场景区布局（档案区为兄弟节点，不参与 sticky） */
.planet-section {
  position: relative;
  height: 150dvh;
  min-height: 990px;
}

/* 天体档案是进入目录前的简明资料，不应像独立工作区一样占满一整屏并留下大块空白。 */
.planet-profile-section {
  min-height: 0;
  padding-bottom: 48px;
}

/* 星野背景（按行星主题色，与火星暖红星野同构） */
.planet-section {
  background:
    radial-gradient(1.2px 1.2px at 12% 22%, rgba(240, 224, 178, .4), transparent 100%),
    radial-gradient(1px 1px at 23% 64%, rgba(240, 224, 178, .3), transparent 100%),
    radial-gradient(.8px .8px at 31% 38%, rgba(240, 224, 178, .25), transparent 100%),
    radial-gradient(1.4px 1.4px at 41% 82%, rgba(240, 224, 178, .36), transparent 100%),
    radial-gradient(1px 1px at 55% 15%, rgba(240, 224, 178, .28), transparent 100%),
    radial-gradient(.9px .9px at 62% 48%, rgba(240, 224, 178, .24), transparent 100%),
    radial-gradient(1.3px 1.3px at 71% 74%, rgba(240, 224, 178, .32), transparent 100%),
    radial-gradient(1px 1px at 79% 29%, rgba(240, 224, 178, .26), transparent 100%),
    radial-gradient(.8px .8px at 88% 58%, rgba(240, 224, 178, .28), transparent 100%),
    radial-gradient(1.1px 1.1px at 94% 12%, rgba(240, 224, 178, .32), transparent 100%),
    radial-gradient(1px 1px at 7% 86%, rgba(240, 224, 178, .26), transparent 100%),
    radial-gradient(.9px .9px at 49% 92%, rgba(240, 224, 178, .24), transparent 100%),
    radial-gradient(1.2px 1.2px at 66% 4%, rgba(240, 224, 178, .3), transparent 100%),
    radial-gradient(ellipse at 50% 50%, #14100a 0%, #050302 100%);
}
.planet-section.saturn {
  background:
    radial-gradient(1.2px 1.2px at 12% 22%, rgba(231, 201, 135, .4), transparent 100%),
    radial-gradient(1px 1px at 23% 64%, rgba(231, 201, 135, .3), transparent 100%),
    radial-gradient(.8px .8px at 31% 38%, rgba(231, 201, 135, .25), transparent 100%),
    radial-gradient(1.4px 1.4px at 41% 82%, rgba(231, 201, 135, .36), transparent 100%),
    radial-gradient(1px 1px at 55% 15%, rgba(231, 201, 135, .28), transparent 100%),
    radial-gradient(.9px .9px at 62% 48%, rgba(231, 201, 135, .24), transparent 100%),
    radial-gradient(1.3px 1.3px at 71% 74%, rgba(231, 201, 135, .32), transparent 100%),
    radial-gradient(1px 1px at 79% 29%, rgba(231, 201, 135, .26), transparent 100%),
    radial-gradient(.8px .8px at 88% 58%, rgba(231, 201, 135, .28), transparent 100%),
    radial-gradient(1.1px 1.1px at 94% 12%, rgba(231, 201, 135, .32), transparent 100%),
    radial-gradient(1px 1px at 7% 86%, rgba(231, 201, 135, .26), transparent 100%),
    radial-gradient(.9px .9px at 49% 92%, rgba(231, 201, 135, .24), transparent 100%),
    radial-gradient(1.2px 1.2px at 66% 4%, rgba(231, 201, 135, .3), transparent 100%),
    radial-gradient(ellipse at 50% 50%, #171209 0%, #050302 100%);
}
.planet-section.jupiter {
  background:
    radial-gradient(1.2px 1.2px at 12% 22%, rgba(207, 146, 87, .4), transparent 100%),
    radial-gradient(1px 1px at 23% 64%, rgba(207, 146, 87, .3), transparent 100%),
    radial-gradient(.8px .8px at 31% 38%, rgba(207, 146, 87, .25), transparent 100%),
    radial-gradient(1.4px 1.4px at 41% 82%, rgba(207, 146, 87, .36), transparent 100%),
    radial-gradient(1px 1px at 55% 15%, rgba(207, 146, 87, .28), transparent 100%),
    radial-gradient(.9px .9px at 62% 48%, rgba(207, 146, 87, .24), transparent 100%),
    radial-gradient(1.3px 1.3px at 71% 74%, rgba(207, 146, 87, .32), transparent 100%),
    radial-gradient(1px 1px at 79% 29%, rgba(207, 146, 87, .26), transparent 100%),
    radial-gradient(.8px .8px at 88% 58%, rgba(207, 146, 87, .28), transparent 100%),
    radial-gradient(1.1px 1.1px at 94% 12%, rgba(207, 146, 87, .32), transparent 100%),
    radial-gradient(1px 1px at 7% 86%, rgba(207, 146, 87, .26), transparent 100%),
    radial-gradient(.9px .9px at 49% 92%, rgba(207, 146, 87, .24), transparent 100%),
    radial-gradient(1.2px 1.2px at 66% 4%, rgba(207, 146, 87, .3), transparent 100%),
    radial-gradient(ellipse at 50% 50%, #160e06 0%, #050302 100%);
}
.planet-section.mercury {
  --planet-accent: #c8b8a0;
  --planet-accent-dim: rgba(200, 184, 160, .4);
  --planet-line: rgba(200, 184, 160, .22);
  --planet-text: #efe6d8;
  --planet-quiet: #b3a48f;
  background:
    radial-gradient(1.2px 1.2px at 12% 22%, rgba(200, 184, 160, .4), transparent 100%),
    radial-gradient(1px 1px at 23% 64%, rgba(200, 184, 160, .3), transparent 100%),
    radial-gradient(.8px .8px at 31% 38%, rgba(200, 184, 160, .25), transparent 100%),
    radial-gradient(1.4px 1.4px at 41% 82%, rgba(200, 184, 160, .36), transparent 100%),
    radial-gradient(1px 1px at 55% 15%, rgba(200, 184, 160, .28), transparent 100%),
    radial-gradient(.9px .9px at 62% 48%, rgba(200, 184, 160, .24), transparent 100%),
    radial-gradient(1.3px 1.3px at 71% 74%, rgba(200, 184, 160, .32), transparent 100%),
    radial-gradient(1px 1px at 79% 29%, rgba(200, 184, 160, .26), transparent 100%),
    radial-gradient(.8px .8px at 88% 58%, rgba(200, 184, 160, .28), transparent 100%),
    radial-gradient(1.1px 1.1px at 94% 12%, rgba(200, 184, 160, .32), transparent 100%),
    radial-gradient(1px 1px at 7% 86%, rgba(200, 184, 160, .26), transparent 100%),
    radial-gradient(.9px .9px at 49% 92%, rgba(200, 184, 160, .24), transparent 100%),
    radial-gradient(1.2px 1.2px at 66% 4%, rgba(200, 184, 160, .3), transparent 100%),
    radial-gradient(ellipse at 50% 50%, #17130d 0%, #050302 100%);
}
.planet-section.uranus {
  --planet-accent: #8fd8d0;
  --planet-accent-dim: rgba(143, 216, 208, .4);
  --planet-line: rgba(143, 216, 208, .22);
  --planet-text: #d8f2ee;
  --planet-quiet: #8fb8b4;
  background:
    radial-gradient(1.2px 1.2px at 12% 22%, rgba(143, 216, 208, .4), transparent 100%),
    radial-gradient(1px 1px at 23% 64%, rgba(143, 216, 208, .3), transparent 100%),
    radial-gradient(.8px .8px at 31% 38%, rgba(143, 216, 208, .25), transparent 100%),
    radial-gradient(1.4px 1.4px at 41% 82%, rgba(143, 216, 208, .36), transparent 100%),
    radial-gradient(1px 1px at 55% 15%, rgba(143, 216, 208, .28), transparent 100%),
    radial-gradient(.9px .9px at 62% 48%, rgba(143, 216, 208, .24), transparent 100%),
    radial-gradient(1.3px 1.3px at 71% 74%, rgba(143, 216, 208, .32), transparent 100%),
    radial-gradient(1px 1px at 79% 29%, rgba(143, 216, 208, .26), transparent 100%),
    radial-gradient(.8px .8px at 88% 58%, rgba(143, 216, 208, .28), transparent 100%),
    radial-gradient(1.1px 1.1px at 94% 12%, rgba(143, 216, 208, .32), transparent 100%),
    radial-gradient(1px 1px at 7% 86%, rgba(143, 216, 208, .26), transparent 100%),
    radial-gradient(.9px .9px at 49% 92%, rgba(143, 216, 208, .24), transparent 100%),
    radial-gradient(1.2px 1.2px at 66% 4%, rgba(143, 216, 208, .3), transparent 100%),
    radial-gradient(ellipse at 50% 50%, #0a1816 0%, #050302 100%);
}
.planet-section.neptune {
  --planet-accent: #6aa8e0;
  --planet-accent-dim: rgba(106, 168, 224, .4);
  --planet-line: rgba(106, 168, 224, .22);
  --planet-text: #d8e8f8;
  --planet-quiet: #8fb0d4;
  background:
    radial-gradient(1.2px 1.2px at 12% 22%, rgba(106, 168, 224, .4), transparent 100%),
    radial-gradient(1px 1px at 23% 64%, rgba(106, 168, 224, .3), transparent 100%),
    radial-gradient(.8px .8px at 31% 38%, rgba(106, 168, 224, .25), transparent 100%),
    radial-gradient(1.4px 1.4px at 41% 82%, rgba(106, 168, 224, .36), transparent 100%),
    radial-gradient(1px 1px at 55% 15%, rgba(106, 168, 224, .28), transparent 100%),
    radial-gradient(.9px .9px at 62% 48%, rgba(106, 168, 224, .24), transparent 100%),
    radial-gradient(1.3px 1.3px at 71% 74%, rgba(106, 168, 224, .32), transparent 100%),
    radial-gradient(1px 1px at 79% 29%, rgba(106, 168, 224, .26), transparent 100%),
    radial-gradient(.8px .8px at 88% 58%, rgba(106, 168, 224, .28), transparent 100%),
    radial-gradient(1.1px 1.1px at 94% 12%, rgba(106, 168, 224, .32), transparent 100%),
    radial-gradient(1px 1px at 7% 86%, rgba(106, 168, 224, .26), transparent 100%),
    radial-gradient(.9px .9px at 49% 92%, rgba(106, 168, 224, .24), transparent 100%),
    radial-gradient(1.2px 1.2px at 66% 4%, rgba(106, 168, 224, .3), transparent 100%),
    radial-gradient(ellipse at 50% 50%, #0a1420 0%, #050302 100%);
}
.planet-section.sun {
  background:
    radial-gradient(1.2px 1.2px at 12% 22%, rgba(255, 107, 74, .4), transparent 100%),
    radial-gradient(1px 1px at 23% 64%, rgba(255, 107, 74, .3), transparent 100%),
    radial-gradient(.8px .8px at 31% 38%, rgba(255, 107, 74, .25), transparent 100%),
    radial-gradient(1.4px 1.4px at 41% 82%, rgba(255, 107, 74, .36), transparent 100%),
    radial-gradient(1px 1px at 55% 15%, rgba(255, 107, 74, .28), transparent 100%),
    radial-gradient(.9px .9px at 62% 48%, rgba(255, 107, 74, .24), transparent 100%),
    radial-gradient(1.3px 1.3px at 71% 74%, rgba(255, 107, 74, .32), transparent 100%),
    radial-gradient(1px 1px at 79% 29%, rgba(255, 107, 74, .26), transparent 100%),
    radial-gradient(.8px .8px at 88% 58%, rgba(255, 107, 74, .28), transparent 100%),
    radial-gradient(1.1px 1.1px at 94% 12%, rgba(255, 107, 74, .32), transparent 100%),
    radial-gradient(1px 1px at 7% 86%, rgba(255, 107, 74, .26), transparent 100%),
    radial-gradient(.9px .9px at 49% 92%, rgba(255, 107, 74, .24), transparent 100%),
    radial-gradient(1.2px 1.2px at 66% 4%, rgba(255, 107, 74, .3), transparent 100%),
    radial-gradient(ellipse at 50% 50%, #3d1508 0%, #050302 100%);
}

/* 粘性场景区：首屏 100dvh，下滑进入档案板块 */
.planet-scene-frame {
  position: sticky;
  top: 0;
  height: 100dvh;
  min-height: 660px;
  overflow: hidden;
}
.planet-scene-host {
  position: absolute;
  inset: 0;
  opacity: 0;
  transition: opacity 0.3s ease;
  cursor: grab;
}
.planet-scene-host.revealed {
  opacity: 1;
}
.planet-scene-host.revealed.leaving-body {
  opacity: 0;
  pointer-events: none;
  transition: opacity .32s cubic-bezier(.4, 0, 1, 1) .3s;
}
.planet-scene-host canvas { display: block; }

/* 档案板块：延续月球/火星板块框架（背景/边框/文字走行星主题色） */
.planet-profile-section .section-kicker { color: var(--planet-accent); }
.planet-profile-section .sec-num { color: var(--planet-accent); }
/* 探测器/探索板块：kicker 与编号走行星主题色（全局 --blue 是地球蓝，会破坏行星主题） */
.planet-spacecraft-section .section-kicker,
.planet-sites-section .section-kicker { color: var(--planet-accent); }
.planet-spacecraft-section .sec-num,
.planet-sites-section .sec-num { color: var(--planet-accent); }
.profile-grid {
  max-width: 640px;
  padding: 28px 0 44px;
}
.profile-table {
  display: grid;
  gap: 0;
  margin: 0;
  border: 1px solid var(--planet-line);
  border-radius: 8px;
  background: rgba(8, 6, 4, .55);
  overflow: hidden;
}
.profile-table > div {
  display: grid;
  grid-template-columns: 100px 1fr;
  gap: 16px;
  align-items: baseline;
  padding: 12px 18px;
  border-bottom: 1px solid var(--planet-line);
}
.profile-table > div:last-child { border-bottom: 0; }
.profile-table dt {
  color: var(--planet-quiet);
  font: 500 10px var(--font-mono);
  letter-spacing: .1em;
  padding-top: 2px;
}
.profile-table dd {
  margin: 0;
  color: var(--planet-text);
  font-size: 13px;
  line-height: 1.6;
}
/* 简介行：并入表格最后一行（消除右侧独立文字），文字用 quiet 色、放宽行距更耐读 */
.profile-intro-row {
  align-items: start;
  background: rgba(255, 255, 255, .02);
}
.profile-intro-row dd {
  color: var(--planet-quiet);
  font-size: 12px;
  line-height: 1.9;
}

/* 返回渐隐：工具栏/读数/署名与标签同节奏淡出（只留裸行星，随后由遮罩完成星球渐暗） */
.scene-toolbar.leaving-fade { opacity: 0; pointer-events: none; transition: opacity .3s ease; }
.planet-site-label.leaving-fade,
.planet-craft-label.leaving-fade { opacity: 0; pointer-events: none; transition: opacity .3s ease; }
.planet-readout.leaving-fade,
.planet-credits.leaving-fade { opacity: 0; transition: opacity .3s ease; }
.mission-detail-panel.leaving-fade {
  animation: none !important;
  opacity: 0 !important;
  pointer-events: none;
  transition: opacity .3s ease;
}

@media (prefers-reduced-motion: reduce) {
  .planet-scene-host.revealed.leaving-body { transition: opacity .1s linear .04s; }
}

/* 页脚：仅品牌（署名在场景右下角） */
.planet-page-footer { padding: 10px 0 56px; }

/* 右下角署名 */
.planet-credits {
  position: absolute;
  z-index: 3;
  right: 34px;
  bottom: 30px;
  color: var(--planet-quiet);
  font: 400 7px var(--font-mono);
  letter-spacing: .08em;
  text-align: right;
  pointer-events: none;
}

/* 读数区 */
.planet-readout {
  position: absolute;
  z-index: 4;
  left: 32px;
  bottom: 28px;
  color: var(--planet-text);
}
.planet-readout > span {
  color: var(--planet-quiet);
  font: 500 8px var(--font-mono);
  letter-spacing: .15em;
}
.planet-readout strong {
  display: block;
  margin-top: 5px;
  font-size: 17px;
  font-weight: 500;
}

/* 标注的完整、紧凑、聚合尺寸及左右短线统一由 MissionSceneLabel 管理。 */

/* ===== 探测器详情 ===== */
.planet-craft-panel {
  position: absolute;
  z-index: 8;
  top: 18px;
  right: 32px;
  width: clamp(368px, 25vw, 460px);
  max-height: calc(100% - 36px - var(--header-overlay-offset, 72px));
  overflow-y: auto;
  padding: 24px 28px;
  border: 1px solid var(--planet-line);
  border-radius: 12px;
  background: color-mix(in srgb, var(--planet-accent) 4%, rgba(4, 9, 15, .92));
  box-shadow: 0 24px 70px rgba(0, 8, 18, .45);
  backdrop-filter: blur(20px);
  opacity: 0;
  transform: translateY(8px);
  transition: opacity .28s, transform .38s cubic-bezier(.22, 1, .36, 1);
  scrollbar-width: thin;
}
.planet-craft-panel.visible { opacity: 1; transform: none; }
.planet-craft-panel-close {
  position: absolute;
  top: 8px;
  right: 12px;
  border: 0;
  background: transparent;
  color: var(--planet-quiet);
  font-size: 16px;
  cursor: pointer;
}
.planet-craft-panel-close:hover { color: var(--planet-text); }
.planet-context-type { margin: 0 0 8px; color: var(--planet-accent); font: 500 9px var(--font-mono); letter-spacing: .12em; text-transform: uppercase; }
.planet-craft-panel h2 { margin: 0; color: var(--planet-text); font-size: 22px; font-weight: 500; line-height: 1.15; }
.planet-context-subtitle { margin: 4px 0 0; color: var(--planet-quiet); font: 400 10px var(--font-mono); }
.planet-context-description { margin: 16px 0; color: var(--planet-quiet); font-size: 12px; line-height: 1.75; }
.planet-craft-panel dl { display: grid; gap: 8px; margin: 0; padding-top: 14px; border-top: 1px solid var(--planet-line); }
.planet-craft-panel dl > div { display: grid; grid-template-columns: 46px 1fr; gap: 10px; }
.planet-craft-panel dt { color: var(--planet-quiet); font-size: 11px; }
.planet-craft-panel dd { margin: 0; color: var(--planet-text); font-size: 11px; line-height: 1.5; }
.planet-source-caption { margin: 14px 0 0; color: var(--planet-quiet); font: 400 9px var(--font-mono); line-height: 1.6; }

/* ===== 探测器目录 ===== */
.planet-spacecraft-section .catalog-controls label { text-align: left; }
/* 目录整体走行星主题色：背景去蓝（全局 --surface 深蓝黑 → 主题色透明底），
   表格边框/行分隔/文字从全局蓝色系切到行星主题色系 */
.planet-spacecraft-section .catalog-workspace {
  background: color-mix(in srgb, var(--planet-accent) 4%, rgba(5, 11, 17, .82));
  border: 1px solid var(--planet-line);
}
.planet-spacecraft-section .catalog-workspace.compact,
.planet-sites-section .catalog-workspace.compact { max-width: 980px; }
.catalog-workspace.compact .object-row { min-height: 76px; }
.planet-spacecraft-section .object-table-head,
.planet-spacecraft-section .object-row { border-color: var(--planet-line); }
.planet-spacecraft-section .object-table-head { color: var(--planet-quiet); }
.planet-spacecraft-section .object-row { color: var(--planet-text); }
.planet-spacecraft-section .object-row:hover,
.planet-spacecraft-section .object-row:focus-visible {
  background: color-mix(in srgb, var(--planet-accent) 7%, transparent);
  color: var(--planet-text);
}
.planet-spacecraft-section .object-row strong { color: var(--planet-text); }
.planet-spacecraft-section .object-row small { color: var(--planet-quiet); }
.planet-spacecraft-section .catalog-empty { color: var(--planet-quiet); }
.planet-spacecraft-section .pagination-space { color: var(--planet-quiet); }
.planet-spacecraft-section .pagination-space button { border-color: var(--planet-line); color: var(--planet-quiet); }
.planet-spacecraft-section .pagination-space button:hover { border-color: var(--planet-accent); color: var(--planet-text); }
.planet-spacecraft-section .catalog-controls label > span { color: var(--planet-quiet); }
.planet-spacecraft-section .catalog-controls input,
.planet-spacecraft-section .catalog-controls select { border-color: var(--planet-line); color: var(--planet-text); }
.planet-spacecraft-section .catalog-controls input,
.planet-spacecraft-section .catalog-controls select {
  /* 覆盖全局 #06111a 藏青背景与 --ink 冷白文字：深色底带主题色微光 */
  background-color: color-mix(in srgb, var(--planet-accent) 5%, #0a0d12);
  color: var(--planet-text);
}
.planet-spacecraft-section .catalog-controls input:focus,
.planet-spacecraft-section .catalog-controls select:focus {
  border-color: var(--planet-accent);
  box-shadow: 0 0 0 3px var(--planet-accent-dim);
}
.planet-spacecraft-section .catalog-controls select {
  background-image: url("data:image/svg+xml;charset=utf-8,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6' viewBox='0 0 10 6'%3E%3Cpath d='M1 1l4 4 4-4' fill='none' stroke='%23b0a698' stroke-width='1.5' stroke-linecap='round' stroke-linejoin='round'/%3E%3C/svg%3E");
}
.planet-spacecraft-section .catalog-controls select:hover { border-color: var(--planet-accent); }
.planet-spacecraft-section .catalog-controls input {
  text-align: left;
  font: 400 12px var(--font-mono);
}
.planet-spacecraft-section .catalog-controls input::placeholder {
  color: var(--planet-quiet);
  font-weight: 400;
  opacity: 1;
}
.planet-spacecraft-section .catalog-controls input:focus,
.planet-spacecraft-section .catalog-controls select:focus { border-color: var(--planet-accent); }
.planet-craft-table .object-table-head,
.planet-craft-table .planet-craft-row { grid-template-columns: minmax(250px, 1.35fr) minmax(160px, 1fr) 130px minmax(160px, 1fr); }
.planet-craft-row > span:first-child strong { display: block; color: var(--planet-text); font-size: 13px; font-weight: 500; }
.planet-craft-row > span:first-child small { display: block; margin-top: 3px; color: var(--planet-quiet); font: 400 8px var(--font-mono); }
.planet-craft-row > span:nth-child(3) { display: inline-flex; align-items: center; gap: 7px; }
.craft-status-dot { width: 6px; height: 6px; flex: 0 0 6px; border-radius: 50%; background: #94a7b0; }
.craft-status-dot.status-运行中 { background: var(--planet-accent); box-shadow: 0 0 8px var(--planet-accent-dim); }
.craft-status-dot.status-即将入轨 { background: var(--planet-accent); }
.craft-status-dot.status-飞掠 { background: var(--planet-accent-dim); }
.craft-status-dot.status-已结束 { background: #6e7980; }

/* 足迹图标（沿用火星 SVG 风格，主题色描边） */
.planet-site-glyph svg { display: block; }

/* ===== 选中足迹信息卡 ===== */
.planet-site-panel {
  position: absolute;
  z-index: 7;
  top: 18px;
  right: 32px;
  width: 300px;
  padding: 18px 20px;
  border: 1px solid var(--planet-line);
  border-radius: 12px;
  background: color-mix(in srgb, var(--planet-accent) 4%, rgba(4, 9, 15, .9));
  box-shadow: 0 24px 70px rgba(0, 8, 18, .45);
  backdrop-filter: blur(20px);
  opacity: 0;
  transform: translateY(8px);
  transition: opacity .28s, transform .38s cubic-bezier(.22, 1, .36, 1);
}
.planet-site-panel.visible { opacity: 1; transform: none; }
.planet-site-panel-close {
  position: absolute;
  top: 8px;
  right: 12px;
  border: 0;
  background: transparent;
  color: var(--planet-quiet);
  font-size: 16px;
  cursor: pointer;
}
.planet-site-panel-close:hover { color: var(--planet-text); }
.planet-site-panel-head { display: flex; gap: 12px; align-items: center; padding-bottom: 14px; border-bottom: 1px solid var(--planet-line); }
.planet-site-panel-head .planet-site-glyph.large { color: var(--planet-accent); }
.planet-site-panel-head .planet-site-glyph.large svg { width: 26px; height: 26px; }
.planet-site-panel-head h3 { margin: 0; font-size: 15px; font-weight: 500; color: var(--planet-text); }
.planet-site-panel-head p { margin: 3px 0 0; color: var(--planet-quiet); font: 400 10px var(--font-mono); letter-spacing: .06em; }
.planet-site-panel dl { display: grid; gap: 8px; padding: 14px 0 0; margin: 0; }
.planet-site-panel dl > div { display: grid; grid-template-columns: 52px 1fr; gap: 10px; }
.planet-site-panel dt { color: var(--planet-quiet); font-size: 11px; }
.planet-site-panel dd { margin: 0; color: var(--planet-text); font-size: 11px; line-height: 1.55; }

/* ===== 探索板块：目录行（4 列，含图标列） ===== */
.planet-sites-section .site-row {
  grid-template-columns: minmax(260px, 1.4fr) minmax(220px, 1fr) 150px 130px !important;
  width: 100%;
  text-align: left;
}
.planet-sites-section .site-row .planet-site-glyph { color: var(--planet-accent); flex-shrink: 0; }
.planet-sites-section .site-row-name { display: flex; align-items: center; gap: 10px; }
.planet-sites-section .site-row-name strong { color: var(--planet-text); font-size: 13px; }
.planet-sites-section .site-row-name small { display: block; margin-top: 2px; color: var(--planet-quiet); font: 400 8px var(--font-mono); }
.planet-sites-section .site-row small { display: block; margin-top: 3px; color: var(--planet-quiet); font: 400 8px var(--font-mono); }
/* 着陆点目录同样去蓝：背景/边框/行色走行星主题 */
.planet-sites-section .catalog-workspace {
  background: color-mix(in srgb, var(--planet-accent) 4%, rgba(5, 11, 17, .82));
  border: 1px solid var(--planet-line);
}
.planet-sites-section .object-table-head,
.planet-sites-section .object-row { border-color: var(--planet-line); }
.planet-sites-section .object-table-head { color: var(--planet-quiet); }
.planet-sites-section .object-row { color: var(--planet-text); }
.planet-sites-section .object-row:hover,
.planet-sites-section .object-row:focus-visible {
  background: color-mix(in srgb, var(--planet-accent) 7%, transparent);
  color: var(--planet-text);
}
.planet-sites-section .catalog-empty { color: var(--planet-quiet); }
.planet-sites-section .catalog-controls label > span { color: var(--planet-quiet); }
.planet-sites-section .catalog-controls input,
.planet-sites-section .catalog-controls select {
  border-color: var(--planet-line);
  color: var(--planet-text);
  /* 覆盖全局 #06111a 藏青背景与 --ink 冷白文字 */
  background-color: color-mix(in srgb, var(--planet-accent) 5%, #0a0d12);
}
.planet-sites-section .catalog-controls input::placeholder { color: var(--planet-quiet); opacity: 1; }
.planet-sites-section .catalog-controls input:focus,
.planet-sites-section .catalog-controls select:focus {
  border-color: var(--planet-accent);
  box-shadow: 0 0 0 3px var(--planet-accent-dim);
}
.planet-sites-section .catalog-controls select {
  background-image: url("data:image/svg+xml;charset=utf-8,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6' viewBox='0 0 10 6'%3E%3Cpath d='M1 1l4 4 4-4' fill='none' stroke='%23b0a698' stroke-width='1.5' stroke-linecap='round' stroke-linejoin='round'/%3E%3C/svg%3E");
}
.planet-sites-section .catalog-controls select:hover { border-color: var(--planet-accent); }
</style>
