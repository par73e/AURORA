<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { SolarSystemScene, type SolarLabel } from '../solar/scene'
import { solarSession } from '../solar/session'
import { MOON, planets, SUN, type PlanetSpec } from '../solar/data'
import { solarTexturesReady } from '../solar/textures'
import { fetchDeepSpaceProbes } from '../api'
import { bilingualName } from '../bilingual'
import type { DeepSpaceProbe } from '../types'
import type { ProbeData } from '../solar/scene'
import MissionDetailPanel from './MissionDetailPanel.vue'
import MissionSceneLabel from './MissionSceneLabel.vue'
import type { MissionDetail } from '../missionPresentation'
import { spacecraftFields } from '../missionPresentation'

const props = defineProps<{ spacecraftVisible?: boolean; enterFromOrbit?: boolean; enterFromMoon?: boolean; enterFromMars?: boolean; enterFromVenus?: boolean; enterFromSaturn?: boolean; enterFromJupiter?: boolean; enterFromMercury?: boolean; enterFromUranus?: boolean; enterFromNeptune?: boolean; enterFromSun?: boolean; flyDelay?: number; playEntryFly?: boolean }>()

const emit = defineEmits<{
  'update:spacecraft-visible': [visible: boolean]
  'select-earth': []
  'earth-fly-start': []
  'earth-fly-zoom': []
  'select-moon': []
  'moon-fly-start': []
  'moon-fly-zoom': []
  'select-mars': []
  'mars-fly-start': []
  'mars-fly-zoom': []
  'select-venus': []
  'venus-fly-start': []
  'venus-fly-zoom': []
  'select-saturn': []
  'saturn-fly-start': []
  'saturn-fly-zoom': []
  'select-jupiter': []
  'jupiter-fly-start': []
  'jupiter-fly-zoom': []
  'select-mercury': []
  'mercury-fly-start': []
  'mercury-fly-zoom': []
  'select-uranus': []
  'uranus-fly-start': []
  'uranus-fly-zoom': []
  'select-neptune': []
  'neptune-fly-start': []
  'neptune-fly-zoom': []
  'select-sun': []
  'sun-fly-start': []
  'sun-fly-zoom': []
}>()

const canvasHost = ref<HTMLDivElement | null>(null)
// 选中光标：默认地球；从行星/月球返回时恢复对应星球（组件重新挂载，props 决定初始选中）
const activeId = ref(
  props.enterFromSun ? 'sun'
    : props.enterFromNeptune ? 'neptune' : props.enterFromUranus ? 'uranus' : props.enterFromMercury ? 'mercury'
    : props.enterFromJupiter ? 'jupiter' : props.enterFromSaturn ? 'saturn' : props.enterFromVenus ? 'venus'
    : props.enterFromMars ? 'mars' : props.enterFromMoon ? 'moon' : props.enterFromOrbit ? 'earth' : 'earth',
)
const labels = ref<SolarLabel[]>([])
let scene: SolarSystemScene | undefined
/** 当前飞行动画的目标（事件回调据此分发） */
let flightPlanet: 'moon' | 'mars' | 'venus' | 'saturn' | 'jupiter' | 'mercury' | 'uranus' | 'neptune' | 'sun' | null = null
/** 组件已卸载标记（fetch 回调守卫，避免向已 dispose 的 scene 写数据） */
let unmounted = false
let probesRequest: AbortController | undefined
/** 深空探测器（JPL Horizons 日同步，/api/v1/voyage/probes） */
const probes = ref<DeepSpaceProbe[]>([])
const spacecraftEnabled = computed({
  get: () => props.spacecraftVisible ?? true,
  set: (visible: boolean) => emit('update:spacecraft-visible', visible),
})
/** 点击选中的探测器（信息面板） */
const selectedProbe = ref<DeepSpaceProbe | null>(null)
/** 选中探测器当前距日（AU，打开面板时读取一次） */
const probeDistAU = ref<number | null>(null)
/** 选中探测器轨道参数（倾角/偏心率/周期；无拟合轨道为 null → 显示 —） */
const probeOrbit = ref<{ inclinationDeg: number; eccentricity: number; periodDays: number } | null>(null)
/** 发射信息（日期 · 地点 · 火箭，与地球/月球面板同构） */
const probeLaunchText = computed(() => {
  const p = selectedProbe.value
  if (!p) return ''
  return [p.launchDate, p.launchSite, p.launchVehicle].filter(Boolean).join(' · ')
})
const selectedProbeDetail = computed<MissionDetail | null>(() => {
  const probe = selectedProbe.value
  if (!probe) return null
  const name = bilingualName(probe.nameZh, probe.nameEn)
  return {
    kind: 'spacecraft',
    typeZh: '飞行器',
    typeEn: 'SPACECRAFT',
    status: `${probe.missionType} · 精度 ${probe.precisionGrade}`,
    nameZh: name.primary,
    nameEn: name.secondary,
    description: probe.description,
    fields: spacecraftFields({
      operator: probe.operatorName,
      launch: probeLaunchText.value,
      target: [probe.target, probeDistAU.value == null ? '' : `当前距日 ${probeDistAU.value.toFixed(2)} AU`].filter(Boolean).join(' · '),
      inclination: probeOrbit.value ? `${probeOrbit.value.inclinationDeg.toFixed(2)}°` : '',
      eccentricity: probeOrbit.value?.eccentricity.toFixed(4),
      period: probeOrbit.value ? `${probeOrbit.value.periodDays.toFixed(0)} 天` : '',
    }),
    source: `JPL Horizons · 同步于 ${formatSyncTime(probe.syncedAt)}`,
  }
})

const activePlanet = computed<PlanetSpec>(() => planets.find((p) => p.id === activeId.value) ?? planets[2])
const activeName = computed(() => {
  if (activeId.value === 'sun') return SUN.name
  if (activeId.value === 'moon') return MOON.name
  return activePlanet.value.name
})
const activeNameEn = computed(() => {
  if (activeId.value === 'sun') return SUN.nameEn
  if (activeId.value === 'moon') return MOON.nameEn
  return activePlanet.value.nameEn
})

const planetLabels = computed(() => labels.value.filter((l) => l.kind === 'planet'))
const probeLabels = computed(() => spacecraftEnabled.value ? labels.value.filter((l) => l.kind === 'probe') : [])
const sunLabel = computed(() => labels.value.find((l) => l.kind === 'sun') ?? null)
const beltLabels = computed(() => labels.value.filter((l) => l.kind === 'belt'))
const earthLabel = computed(() => labels.value.find((l) => l.kind === 'planet' && l.id === 'earth') ?? null)

const nameById = new Map(planets.map((p) => [p.id, p.name]))
const nameEnById = new Map(planets.map((p) => [p.id, p.nameEn]))
nameById.set(MOON.id, MOON.name)
nameEnById.set(MOON.id, MOON.nameEn)

/** 双语名称（统一规则）：全部中文主，外国对象附英文括号注释 */
const probeBilingual = computed(() => new Map(probes.value.map((p) => [p.id, bilingualName(p.nameZh, p.nameEn)])))

function choosePlanet(id: string) {
  activeId.value = id
  selectedProbe.value = null
  scene?.clearProbeSelection()
  if (id === 'earth') {
    // 地球：先在太阳系场景内放大地球，飞行到位后再由 App 切换页面
    flightPlanet = null
    scene?.flyToEarth()
    emit('earth-fly-start')
  } else if (id === 'moon') {
    // 月球：镜像地球流程——太阳系内推近月球 → 渐暗 → 切到月球页面
    flightPlanet = 'moon'
    scene?.flyToMoon()
    emit('moon-fly-start')
  } else if (id === 'mars') {
    // 火星：镜像月球流程——太阳系内推近火星 → 渐暗 → 切到火星页面
    flightPlanet = 'mars'
    scene?.flyToMars()
    emit('mars-fly-start')
  } else if (id === 'venus') {
    // 金星：镜像月球/火星流程——太阳系内推近金星 → 渐暗 → 切到金星页面
    flightPlanet = 'venus'
    scene?.flyToVenus()
    emit('venus-fly-start')
  } else if (id === 'saturn') {
    // 土星：镜像月球/火星流程——太阳系内推近土星 → 渐暗 → 切到土星页面
    flightPlanet = 'saturn'
    scene?.flyToSaturn()
    emit('saturn-fly-start')
  } else if (id === 'jupiter') {
    // 木星：镜像月球/火星流程——太阳系内推近木星 → 渐暗 → 切到木星页面
    flightPlanet = 'jupiter'
    scene?.flyToJupiter()
    emit('jupiter-fly-start')
  } else if (id === 'mercury') {
    // 水星：镜像月球/火星流程——太阳系内推近水星 → 渐暗 → 切到水星页面
    flightPlanet = 'mercury'
    scene?.flyToMercury()
    emit('mercury-fly-start')
  } else if (id === 'uranus') {
    // 天王星：镜像月球/火星流程——太阳系内推近天王星 → 渐暗 → 切到天王星页面
    flightPlanet = 'uranus'
    scene?.flyToUranus()
    emit('uranus-fly-start')
  } else if (id === 'neptune') {
    // 海王星：镜像月球/火星流程——太阳系内推近海王星 → 渐暗 → 切到海王星页面
    flightPlanet = 'neptune'
    scene?.flyToNeptune()
    emit('neptune-fly-start')
  } else if (id === 'sun') {
    // 太阳：镜像月球/火星流程——太阳系内推近太阳 → 渐暗 → 切到太阳页面
    flightPlanet = 'sun'
    scene?.flyToSun()
    emit('sun-fly-start')
  }
}
/** 探测器标签悬停/移开：点亮/熄灭对应探测器轨迹（与悬停 3D 标记一致） */
function hoverProbe(id: string | null) {
  scene?.setHover(id)
}

/** 点击深空探测器（3D 标记或标签）：镜头飞近 + （椭圆轨道）点亮轨道 + 打开信息面板，一个入口。
 *  飞行中/运镜未启动时不弹面板，保持状态一致 */
function onProbeClick(id: string) {
  if (scene?.flyToProbe(id) === true) selectProbe(id)
}

/** 点击深空探测器：打开信息面板 + 读取当前距日（AU）与轨道参数 */
function selectProbe(id: string) {
  const probe = probes.value.find((p) => p.id === id) ?? null
  selectedProbe.value = probe
  probeDistAU.value = null
  probeOrbit.value = null
  if (probe && scene) {
    const info = scene.getProbeInfo(id)
    if (info) probeDistAU.value = info.distAU
    probeOrbit.value = scene.getProbeOrbit(id)
  }
}

/** 关闭探测器面板：同时熄灭其轨道高亮 */
function closeProbePanel() {
  selectedProbe.value = null
  probeDistAU.value = null
  probeOrbit.value = null
  scene?.clearProbeSelection()
}

/** 同步时间展示（UTC） */
function formatSyncTime(iso?: string) {
  if (!iso) return '暂无同步'
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())} ${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())} UTC`
}

// 排列线（屏幕方向：右上 → 左下）的单位法向（右下），标签沿该方向放在球体轮廓之外，避免压到相邻行星；
// 数值按 28° 俯视、35° 方位角下的投影方向近似
const LABEL_OFFSET_X = 0.5
const LABEL_OFFSET_Y = 0.87

/** 标签放在球体轮廓之外：沿排列线法向（右下）偏移 */
function planetLabelStyle(label: SolarLabel) {
  // 月球与地球在这张总览图中有意保持真实邻近关系；标签略向左上避让，
  // 既不盖住地球入口，也保持紧贴月球轮廓。
  if (label.id === 'moon') {
    return { opacity: label.opacity, transform: `translate(calc(${label.x - 24}px - 50%), ${label.y - label.radiusPx - 18}px)` }
  }
  const offset = label.radiusPx + 14
  return { opacity: label.opacity, transform: `translate(calc(${label.x + offset * LABEL_OFFSET_X}px - 50%), ${label.y + offset * LABEL_OFFSET_Y}px)` }
}

function sunLabelStyle(label: SolarLabel) {
  const offset = label.radiusPx + 16
  return { opacity: label.opacity, transform: `translate(calc(${label.x + offset * LABEL_OFFSET_X}px - 50%), ${label.y + offset * LABEL_OFFSET_Y}px)` }
}

function beltLabelStyle(label: SolarLabel) {
  return { opacity: label.opacity, transform: `translate(calc(${label.x}px - 50%), ${label.y - 12}px)` }
}

/** YOU ARE HERE 标记：位于地球正上方，底部小箭头指向地球顶端 */
function youMarkerStyle(label: SolarLabel) {
  return { opacity: label.opacity, transform: `translate(calc(${label.x}px - 50%), ${label.y - label.radiusPx - 27}px)` }
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown)
  if (!canvasHost.value) return
  scene = new SolarSystemScene(
    canvasHost.value,
    {
      onHover: (id) => {
        // 探测器不进入 activeId（键盘导航顺序只含太阳/行星/月球）；悬停高亮由 scene 处理
        if (id && !probes.value.some((p) => p.id === id)) activeId.value = id
      },
      onProbeSelect: (id) => {
        // 联动：镜头飞近探测器 + 点亮绕日轨道 + 弹出左下角信息面板
        onProbeClick(id)
      },
      onProbeDeselect: () => {
        selectedProbe.value = null
        probeDistAU.value = null
        probeOrbit.value = null
      },
      onSelect: choosePlanet,
      onFlyZoom: () => {
        if (flightPlanet === 'moon') emit('moon-fly-zoom')
        else if (flightPlanet === 'mars') emit('mars-fly-zoom')
        else if (flightPlanet === 'venus') emit('venus-fly-zoom')
        else if (flightPlanet === 'saturn') emit('saturn-fly-zoom')
        else if (flightPlanet === 'jupiter') emit('jupiter-fly-zoom')
        else if (flightPlanet === 'mercury') emit('mercury-fly-zoom')
        else if (flightPlanet === 'uranus') emit('uranus-fly-zoom')
        else if (flightPlanet === 'neptune') emit('neptune-fly-zoom')
        else if (flightPlanet === 'sun') emit('sun-fly-zoom')
        else emit('earth-fly-zoom')
      },
      onFlyComplete: () => {
        if (flightPlanet === 'moon') emit('select-moon')
        else if (flightPlanet === 'mars') emit('select-mars')
        else if (flightPlanet === 'venus') emit('select-venus')
        else if (flightPlanet === 'saturn') emit('select-saturn')
        else if (flightPlanet === 'jupiter') emit('select-jupiter')
        else if (flightPlanet === 'mercury') emit('select-mercury')
        else if (flightPlanet === 'uranus') emit('select-uranus')
        else if (flightPlanet === 'neptune') emit('select-neptune')
        else if (flightPlanet === 'sun') emit('select-sun')
        else emit('select-earth')
      },
    },
    (next) => {
      labels.value = next
    },
  )
  scene.setProbesVisible(spacecraftEnabled.value)
  // 会话记忆：上次是"真实公转位置"模式则直接恢复（无动画）；刷新/首次访问为默认排布
  if (!alignedPositions.value) scene.setRealPositions()
  if (props.enterFromSun) {
    // 从太阳页面返回：镜头从太阳近景拉回默认构图（太阳缩回太阳系）
    scene.flyFromSun()
  } else if (props.enterFromMercury) {
    // 从水星页面返回：镜头从水星近景拉回默认构图（水星缩回太阳系）
    scene.flyFromMercury()
  } else if (props.enterFromUranus) {
    // 从天王星页面返回：镜头从天王星近景拉回默认构图（天王星缩回太阳系）
    scene.flyFromUranus()
  } else if (props.enterFromNeptune) {
    // 从海王星页面返回：镜头从海王星近景拉回默认构图（海王星缩回太阳系）
    scene.flyFromNeptune()
  } else if (props.enterFromVenus) {
    // 从金星页面返回：镜头从金星近景拉回默认构图（金星缩回太阳系）
    scene.flyFromVenus()
  } else if (props.enterFromSaturn) {
    // 从土星页面返回：镜头从土星近景拉回默认构图（土星缩回太阳系）
    scene.flyFromSaturn()
  } else if (props.enterFromJupiter) {
    // 从木星页面返回：镜头从木星近景拉回默认构图（木星缩回太阳系）
    scene.flyFromJupiter()
  } else if (props.enterFromMars) {
    // 从火星页面返回：镜头从火星近景拉回默认构图（火星缩回太阳系）
    scene.flyFromMars()
  } else if (props.enterFromMoon) {
    // 从月球页面返回：镜头从月球近景拉回默认构图（月球缩回太阳系）
    scene.flyFromMoon()
  } else if (props.enterFromOrbit) {
    // 从 ORBIT 返回：镜头从地球近景拉回默认构图（地球缩回太阳系）
    scene.flyFromEarth()
  } else if (props.playEntryFly) {
    // 封面路径：由远及近飞入默认视角
    scene.flyInFromDistance(props.flyDelay ?? 0)
  }
  // 把组件的初始选中同步给 Three.js 场景：首次从封面进入时 activeId 默认是地球，
  // 也应立即点亮地球轨道；从其他页面返回时则恢复对应天体的选中状态。
  scene.setSelected(activeId.value)
  // 刷新/直接加载：不播推镜，静态恢复默认构图（resize 触发 refit 定位）
  // 深空探测器：挂载后拉取 JPL Horizons 位置采样并传入场景（标记点 + 轨迹线）
  const controller = new AbortController()
  probesRequest = controller
  fetchDeepSpaceProbes(controller.signal)
    .then((items) => {
      if (unmounted) return
      probes.value = items
      if (scene) scene.setProbes(items)
    })
    .catch((error) => {
      if (controller.signal.aborted) return
      console.error('加载深空探测器数据失败:', error)
    })
})

onBeforeUnmount(() => {
  unmounted = true
  probesRequest?.abort()
  probesRequest = undefined
  window.removeEventListener('keydown', onKeydown)
  scene?.dispose()
})

/** 行星是否处于"一字排布"模式；点击切换为当前真实公转位置（模式存入会话记忆） */
const alignedPositions = ref(!solarSession.realPositions)

function togglePositions() {
  alignedPositions.value = !alignedPositions.value
  solarSession.realPositions = !alignedPositions.value
  if (alignedPositions.value) {
    scene?.animateToAligned()
  } else {
    scene?.animateToRealPositions()
  }
}

/** 键盘导航的选中顺序：太阳 + 8 颗行星 + 月球（紧跟地球） */
const SELECTION_ORDER = ['sun', ...planets.map((p) => p.id)]
SELECTION_ORDER.splice(SELECTION_ORDER.indexOf('earth') + 1, 0, 'moon')

/** 左右方向键切换选中行星（联动左下角读数），回车进入（目前仅地球开放） */
function onKeydown(event: KeyboardEvent) {
  const target = event.target as HTMLElement | null
  if (target && ['BUTTON', 'INPUT', 'SELECT', 'TEXTAREA', 'A'].includes(target.tagName)) return
  const index = SELECTION_ORDER.indexOf(activeId.value)
  if (index < 0) return
  if (event.key === 'ArrowRight') {
    // 右箭头 = 向内侧（水星方向）
    event.preventDefault()
    activeId.value = SELECTION_ORDER[(index - 1 + SELECTION_ORDER.length) % SELECTION_ORDER.length]
    // 方向键选中同样点亮对应行星的轨道线（scene 内指针悬停优先于键盘选中）
    scene?.setSelected(activeId.value)
  } else if (event.key === 'ArrowLeft') {
    // 左箭头 = 向外侧（海王星方向）
    event.preventDefault()
    activeId.value = SELECTION_ORDER[(index + 1) % SELECTION_ORDER.length]
    // 方向键选中同样点亮对应行星的轨道线（scene 内指针悬停优先于键盘选中）
    scene?.setSelected(activeId.value)
  } else if (event.key === 'Enter') {
    if (activeId.value === 'earth') choosePlanet('earth')
    else if (activeId.value === 'moon') choosePlanet('moon')
    else if (activeId.value === 'mars') choosePlanet('mars')
    else if (activeId.value === 'venus') choosePlanet('venus')
    else if (activeId.value === 'saturn') choosePlanet('saturn')
    else if (activeId.value === 'jupiter') choosePlanet('jupiter')
    else if (activeId.value === 'mercury') choosePlanet('mercury')
    else if (activeId.value === 'uranus') choosePlanet('uranus')
    else if (activeId.value === 'neptune') choosePlanet('neptune')
    else if (activeId.value === 'sun') choosePlanet('sun')
  }
}

/** 复位视角：恢复到当前模式的默认构图（一字排开 → 小行星带锚定；真实位置 → 太阳居中），
 *  同时清掉探测器选中状态（由页头图标/重置按钮触发） */
function resetView() {
  selectedProbe.value = null
  probeDistAU.value = null
  scene?.clearProbeSelection()
  scene?.resetView()
}

watch(spacecraftEnabled, (enabled) => {
  scene?.setProbesVisible(enabled)
  if (!enabled) closeProbePanel()
})

function toggleSpacecraftVisibility() {
  spacecraftEnabled.value = !spacecraftEnabled.value
}

defineExpose({ resetView })
</script>

<template>
  <section class="solar-system" aria-labelledby="solar-system-title">
    <header class="solar-intro">
      <p>SOLAR SYSTEM · OBSERVATORY</p>
      <h1 id="solar-system-title">太阳系</h1>
    </header>

    <div ref="canvasHost" class="solar-scene-host" role="group" aria-label="太阳系示意排布场景，点击地球可进入 ORBIT 观测界面">
      <button
        v-for="label in planetLabels"
        v-show="label.visible"
        :key="label.id"
        class="solar-label planet-label"
        :class="{ earth: label.id === 'earth', active: activeId === label.id }"
        :style="planetLabelStyle(label)"
        :aria-label="label.id === 'earth' ? '进入地球 ORBIT 观测界面' : `${nameById.get(label.id)}（${nameEnById.get(label.id)}）`"
        @mouseenter="activeId = label.id"
        @focus="activeId = label.id"
        @click="choosePlanet(label.id)"
      >
        <strong>{{ nameById.get(label.id) }}</strong>
        <small>{{ nameEnById.get(label.id) }}</small>
        <i class="earth-entry">ENTER {{ nameEnById.get(label.id) }} ↗</i>
      </button>

      <MissionSceneLabel
        v-for="label in probeLabels"
        v-show="label.visible"
        :key="label.id"
        class="probe-label"
        :class="{ active: selectedProbe?.id === label.id }"
        :style="planetLabelStyle(label)"
        kind="spacecraft"
        compact
        :name-zh="probeBilingual.get(label.id)?.primary ?? ''"
        :name-en="probeBilingual.get(label.id)?.secondary"
        :selected="selectedProbe?.id === label.id"
        :aria-label="`${probeBilingual.get(label.id)?.primary}${probeBilingual.get(label.id)?.secondary ? `（${probeBilingual.get(label.id)?.secondary}）` : ''}`"
        @mouseenter="hoverProbe(label.id)"
        @mouseleave="hoverProbe(null)"
        @focus="hoverProbe(label.id)"
        @blur="hoverProbe(null)"
        @click="onProbeClick(label.id)"
      />

      <div
        v-if="sunLabel"
        v-show="sunLabel.visible"
        class="solar-label sun-label"
        :class="{ active: activeId === 'sun' }"
        :style="sunLabelStyle(sunLabel)"
        role="button"
        tabindex="0"
        :aria-label="'进入太阳页面'"
        @click="choosePlanet('sun')"
        @keydown.enter="choosePlanet('sun')"
      >
        <strong>太阳</strong>
        <small>SUN</small>
        <i class="earth-entry">ENTER SUN ↗</i>
      </div>

      <div
        v-for="belt in beltLabels"
        v-show="belt.visible"
        :key="belt.id"
        class="belt-label"
        :style="beltLabelStyle(belt)"
        aria-hidden="true"
      >
        {{ belt.id === 'asteroid-belt' ? '小行星主带' : '柯伊伯带' }}
        <small>{{ belt.id === 'asteroid-belt' ? 'THE BELT' : 'KUIPER BELT' }}</small>
      </div>

      <div
        v-if="earthLabel"
        v-show="earthLabel.visible"
        class="you-marker"
        :style="youMarkerStyle(earthLabel)"
        aria-hidden="true"
      >
        <strong>YOU ARE HERE</strong>
        <svg viewBox="0 0 24 14" role="presentation">
          <path d="M12 1 V8" />
          <path d="M8 6 L12 11 L16 6" />
        </svg>
      </div>
    </div>

    <MissionDetailPanel v-if="selectedProbeDetail" :detail="selectedProbeDetail" @close="closeProbePanel" />

    <button class="reset-view" type="button" @click="resetView">重置</button>
    <button class="spacecraft-toggle" :class="{ off: !spacecraftEnabled }" type="button" :aria-pressed="spacecraftEnabled" @click="toggleSpacecraftVisibility"><i aria-hidden="true" />显示飞行器</button>
    <button class="position-toggle" type="button" @click="togglePositions">
      <i :class="{ real: !alignedPositions }" aria-hidden="true" />
      显示行星当前位置
    </button>

    <div class="solar-credits" aria-hidden="true">
      <span>Solar System Scope · CC BY 4.0</span>
    </div>

    <footer class="solar-readout" aria-live="polite">
      <span>{{ activeNameEn }}</span>
      <strong>{{ activeName }}</strong>
    </footer>
  </section>
</template>

<style scoped>
.solar-system {
  position: relative;
  height: 100dvh;
  min-height: 680px;
  overflow: hidden;
  isolation: isolate;
  padding-top: 72px;
  color: #e8f2f5;
  --mission-accent: #72d7ff;
  --mission-accent-dim: rgba(114, 215, 255, .38);
  --mission-line: rgba(139, 180, 202, .2);
  --mission-text: #ecf5f9;
  --mission-quiet: #7f98a7;
  --mission-body: #a8c0cc;
  --mission-panel-surface: rgba(5, 14, 22, .95);
  background:
    radial-gradient(ellipse at 82% 10%, rgba(255, 169, 70, .085), transparent 16%),
    radial-gradient(ellipse at 45% 68%, rgba(24, 94, 133, .12), transparent 42%),
    linear-gradient(134deg, #010408 0%, #020812 50%, #010408 100%);
}

.solar-system::before {
  content: '';
  position: absolute;
  z-index: -2;
  inset: 72px 0 0;
  pointer-events: none;
  background: linear-gradient(156deg, transparent 36%, rgba(32, 96, 128, .035) 59%, transparent 60%);
}

/* 首页交接的第二拍：Three.js 负责真实镜头推进，这里只让场景容器与信息层
   从同一空间方向接稳，避免整页同时“啪”地出现。 */
.solar-system.home-entering,
.solar-system.home-leaving { pointer-events: none; }

.solar-system.home-entering .solar-scene-host {
  animation: solar-home-scene-enter 1.28s cubic-bezier(.16, 1, .3, 1) both;
  transform-origin: 54% 44%;
}

.solar-system.home-entering .solar-intro,
.solar-system.home-entering .reset-view,
.solar-system.home-entering .spacecraft-toggle,
.solar-system.home-entering .position-toggle,
.solar-system.home-entering .solar-credits,
.solar-system.home-entering .solar-readout {
  animation: solar-home-ui-enter .52s .56s cubic-bezier(.16, 1, .3, 1) both;
}

.solar-system.home-leaving .solar-intro,
.solar-system.home-leaving .reset-view,
.solar-system.home-leaving .spacecraft-toggle,
.solar-system.home-leaving .position-toggle,
.solar-system.home-leaving .solar-credits,
.solar-system.home-leaving .solar-readout,
.solar-system.home-leaving :deep(.mission-detail-panel) {
  animation: solar-home-ui-leave .28s cubic-bezier(.4, 0, 1, 1) both;
}

.solar-system.home-leaving .solar-scene-host {
  animation: solar-home-scene-leave .62s cubic-bezier(.4, 0, .2, 1) both;
  transform-origin: 54% 44%;
}

@keyframes solar-home-scene-enter {
  from { opacity: .18; transform: translate3d(14px, 0, 0) scale(.985); }
  to { opacity: 1; transform: translate3d(0, 0, 0) scale(1); }
}

@keyframes solar-home-ui-enter {
  from { opacity: 0; filter: blur(5px); transform: translateY(8px); }
  to { opacity: 1; filter: blur(0); transform: translateY(0); }
}

@keyframes solar-home-ui-leave {
  from { opacity: 1; filter: blur(0); transform: translateY(0); }
  to { opacity: 0; filter: blur(4px); transform: translateY(-7px); }
}

@keyframes solar-home-scene-leave {
  from { opacity: 1; transform: translate3d(0, 0, 0) scale(1); }
  to { opacity: 0; transform: translate3d(-12px, 0, 0) scale(.982); }
}

@media (prefers-reduced-motion: reduce) {
  .solar-system.home-entering .solar-scene-host,
  .solar-system.home-entering .solar-intro,
  .solar-system.home-entering .reset-view,
  .solar-system.home-entering .spacecraft-toggle,
  .solar-system.home-entering .position-toggle,
  .solar-system.home-entering .solar-credits,
  .solar-system.home-entering .solar-readout,
  .solar-system.home-leaving .solar-scene-host,
  .solar-system.home-leaving .solar-intro,
  .solar-system.home-leaving .reset-view,
  .solar-system.home-leaving .spacecraft-toggle,
  .solar-system.home-leaving .position-toggle,
  .solar-system.home-leaving .solar-credits,
  .solar-system.home-leaving .solar-readout,
  .solar-system.home-leaving :deep(.mission-detail-panel) {
    animation-duration: .04s;
    animation-delay: 0s;
  }
}

.solar-intro { position: absolute; z-index: 4; top: 110px; left: 32px; }
.solar-intro p { margin: 0 0 9px; color: rgba(112, 179, 209, .72); font: 500 8px var(--font-mono); letter-spacing: .2em; }
.solar-intro h1 { margin: 0; font-size: 34px; font-weight: 400; letter-spacing: .08em; }

.solar-scene-host {
  position: absolute;
  inset: 72px 0 0;
  overflow: hidden;
  cursor: default;
}
/* 明确把 WebGL 画布放在标签层下：否则某些浏览器的 canvas 合成层会抢到标签的指针事件，
   出现“点地球却命中附近月球”的错位交互。 */
.solar-scene-host :deep(canvas) { position: absolute; inset: 0; z-index: 0; display: block; }

.solar-label {
  position: absolute;
  left: 0;
  top: 0;
  z-index: 4;
  display: grid;
  justify-items: center;
  gap: 3px;
  margin: 0;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  white-space: nowrap;
  pointer-events: none;
  appearance: none;
  text-shadow: 0 2px 8px rgba(0, 0, 0, .85);
}
.planet-label { pointer-events: auto; cursor: pointer; }
.planet-label strong { color: rgba(224, 235, 240, .88); font-size: 10px; font-weight: 500; letter-spacing: .08em; }
.planet-label small { color: rgba(104, 138, 153, .74); font: 400 7px var(--font-mono); letter-spacing: .12em; }
.planet-label:focus-visible { outline: 1px dashed rgba(115, 223, 255, .6); outline-offset: 4px; border-radius: 2px; }

.sun-label strong { color: rgba(255, 218, 159, .9); font-size: 10px; font-weight: 500; letter-spacing: .08em; }
.sun-label small { color: rgba(213, 147, 75, .7); font: 400 7px var(--font-mono); letter-spacing: .12em; }

.earth-entry {
  display: flex;
  gap: 5px;
  align-items: center;
  margin-top: 2px;
  color: rgba(126, 222, 252, .92);
  font: 500 7px var(--font-mono);
  letter-spacing: .1em;
  opacity: 0;
  transform: translateY(3px);
  transition: opacity .25s, transform .4s cubic-bezier(.16, 1, .3, 1);
}
.planet-label:hover .earth-entry,
.planet-label:focus-visible .earth-entry,
.planet-label.active .earth-entry,
.sun-label:hover .earth-entry,
.sun-label.active .earth-entry { opacity: 1; transform: translateY(0); }
.sun-label .earth-entry { color: rgba(255, 205, 130, .95); }

.you-marker {
  position: absolute;
  left: 0;
  top: 0;
  z-index: 6;
  display: grid;
  justify-items: center;
  gap: 2px;
  pointer-events: none;
}
.you-marker strong {
  color: rgba(214, 240, 250, .95);
  font: 600 9px var(--font-mono);
  letter-spacing: .14em;
  text-transform: uppercase;
  text-shadow: 0 1px 6px rgba(0, 0, 0, .9), 0 0 12px rgba(120, 200, 235, .45);
}
.you-marker svg { width: 22px; height: 13px; overflow: visible; }
.you-marker path {
  fill: none;
  stroke: rgba(149, 211, 235, .6);
  stroke-width: 0.9;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.belt-label {
  position: absolute;
  left: 0;
  top: 0;
  z-index: 3;
  color: rgba(143, 163, 170, .64);
  font-size: 8px;
  letter-spacing: .08em;
  text-align: center;
  pointer-events: none;
  text-shadow: 0 1px 6px rgba(0, 0, 0, .8);
}
.belt-label small { display: block; margin-top: 4px; color: rgba(76, 109, 123, .65); font: 400 6px var(--font-mono); letter-spacing: .1em; }

.probe-panel {
  position: absolute;
  z-index: 9;
  left: 32px;
  bottom: 120px;
  width: min(340px, calc(100vw - 64px));
  padding: 18px 20px 16px;
  border: 1px solid rgba(120, 165, 190, .22);
  border-radius: 12px;
  background: linear-gradient(200deg, rgba(6, 14, 22, .94), rgba(9, 20, 30, .9));
  color: #d8e6ee;
  backdrop-filter: blur(14px);
  box-shadow: 0 18px 48px rgba(0, 0, 0, .45);
}
.probe-panel-close {
  position: absolute;
  top: 10px;
  right: 12px;
  width: 22px;
  height: 22px;
  border: 1px solid rgba(120, 165, 190, .25);
  border-radius: 4px;
  background: transparent;
  color: rgba(150, 178, 192, .8);
  font-size: 14px;
  line-height: 1;
  cursor: pointer;
}
.probe-panel-close:hover { color: #e8f5fb; border-color: rgba(120, 165, 190, .5); }
.probe-panel-kicker {
  margin: 0 0 8px;
  color: rgba(112, 179, 209, .72);
  font: 500 8px var(--font-mono);
  letter-spacing: .16em;
}
.probe-panel h2 { margin: 0 0 12px; font-size: 17px; font-weight: 500; letter-spacing: .04em; }
.probe-panel h2 small { margin-left: 8px; color: rgba(122, 152, 168, .72); font: 400 9px var(--font-mono); letter-spacing: .14em; }
.probe-panel dl { display: grid; gap: 6px; margin: 0 0 12px; }
.probe-panel dl > div { display: grid; grid-template-columns: 56px 1fr; gap: 10px; }
.probe-panel dt { color: rgba(112, 145, 162, .78); font: 500 8px var(--font-mono); letter-spacing: .08em; }
.probe-panel dd { margin: 0; color: rgba(214, 228, 236, .92); font-size: 11px; line-height: 1.5; }
.probe-panel-desc { margin: 0 0 10px; color: rgba(178, 200, 212, .82); font-size: 11px; line-height: 1.7; }
.probe-panel-note { margin: 0; color: rgba(100, 128, 144, .6); font: 400 8px var(--font-mono); letter-spacing: .04em; }

.position-toggle,
.spacecraft-toggle,
.reset-view {
  position: absolute;
  z-index: 8;
  right: 32px;
  bottom: 72px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border: 1px solid var(--line);
  border-radius: 4px;
  background: rgba(3, 9, 15, .8);
  color: #9bb0bb;
  font: 500 9px var(--font-mono);
  letter-spacing: .1em;
  cursor: pointer;
  transition: border-color .2s, color .2s, background .2s;
  backdrop-filter: blur(12px);
}
.reset-view {
  bottom: 162px;
}
.spacecraft-toggle { bottom: 118px; }
.position-toggle:hover,
.spacecraft-toggle:hover,
.reset-view:hover {
  border-color: rgba(114, 215, 255, .4);
  background: rgba(3, 9, 15, .92);
  color: #d8e9f2;
}
.position-toggle i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #355161;
  transition: background .2s, box-shadow .2s;
}
.position-toggle i.real {
  background: var(--blue);
  box-shadow: 0 0 8px rgba(114, 215, 255, .6);
}
.spacecraft-toggle i { width: 6px; height: 6px; border-radius: 50%; background: #355161; transition: background .2s, box-shadow .2s; }
.spacecraft-toggle:not(.off) i { background: var(--blue); box-shadow: 0 0 8px rgba(114, 215, 255, .6); }
.spacecraft-toggle:focus-visible { outline: 1px solid rgba(114, 215, 255, .9); outline-offset: 3px; }

.solar-credits {
  position: absolute;
  z-index: 3;
  right: 34px;
  bottom: 30px;
  color: rgba(105, 146, 165, .54);
  font: 400 7px var(--font-mono);
  letter-spacing: .08em;
  text-align: right;
  pointer-events: none;
}

.solar-readout { position: absolute; z-index: 4; left: 32px; bottom: 28px; display: grid; grid-template-columns: 100px auto; align-items: end; column-gap: 18px; }
.solar-readout > span { grid-row: 1; color: rgba(92, 153, 179, .64); font: 500 8px var(--font-mono); letter-spacing: .15em; }
.solar-readout strong { grid-column: 1; grid-row: 2; margin-top: 5px; color: rgba(235, 242, 245, .9); font-size: 17px; font-weight: 500; }

@media (prefers-reduced-motion: no-preference) {
  .solar-intro, .solar-readout { animation: copy-arrive .7s .12s cubic-bezier(.16, 1, .3, 1) both; }
}

@keyframes copy-arrive { from { opacity: 0; filter: blur(5px); transform: translateY(12px); } }

@media (max-width: 1120px) {
  .solar-intro { left: 24px; }
  .solar-readout { left: 24px; }
  .solar-credits { right: 24px; }
.position-toggle { right: 24px; }
.spacecraft-toggle { right: 24px; }
.reset-view { right: 24px; }
}
</style>
