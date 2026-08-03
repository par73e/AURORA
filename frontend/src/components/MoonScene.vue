<template>
  <section class="moon-section" aria-labelledby="moon-title">
    <div id="moon-scene" class="moon-scene-frame">
      <div ref="canvasHost" class="moon-scene-host" :class="{ revealed: sceneRevealed }" role="group" aria-label="月球三维视图，左上角可返回太阳系">
        <!-- 工具栏：与地球页同一套 scene-toolbar 结构（仅颜色走银灰覆盖） -->
        <div class="scene-toolbar" aria-label="场景图层">
          <span>图层</span>
          <label><input v-model="spacecraftEnabled" type="checkbox"><i />航天器</label>
          <label><input v-model="orbitsEnabled" type="checkbox"><i />轨道</label>
          <label><input v-model="terminatorEnabled" type="checkbox"><i class="terminator" />晨昏线</label>
          <label><input v-model="sitesEnabled" type="checkbox"><i class="sites" />着陆点</label>
        </div>

        <!-- 轨道飞行器标签 -->
        <button
          v-for="label in craftLabels"
          v-show="label.visible && spacecraftEnabled"
          :key="label.id"
          class="craft-label"
          :class="{ selected: selectedCraft === label.id }"
          :style="craftLabelStyle(label)"
          :aria-label="`${craftById(label.id)?.nameZh}（${craftById(label.id)?.nameEn}）`"
          @click="selectedCraft = label.id"
        >
          <strong>{{ craftById(label.id)?.nameZh }}</strong>
          <small>{{ craftById(label.id)?.nameEn }}</small>
        </button>

        <!-- 着陆点标签：图标（宇航员/着陆器/月球车/样本）+ 地点名 + 任务名 -->
        <button
          v-for="label in siteLabels"
          v-show="label.visible && sitesEnabled"
          :key="label.id"
          class="craft-label site-label"
          :class="{ selected: selectedSite === label.id }"
          :data-icon="siteById(label.id)?.icon ?? 'lander'"
          :style="siteLabelStyle(label)"
          :aria-label="`${siteById(label.id)?.siteName}（${siteById(label.id)?.missionName}）`"
          @click="selectedSite = selectedSite === label.id ? null : label.id"
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
            <div><dt>月面</dt><dd>{{ siteById(selectedSite)?.side === 'FAR_SIDE' ? '背面（远离地球）' : '正面' }}</dd></div>
            <div><dt>机构</dt><dd>{{ siteById(selectedSite)?.operatorName }}</dd></div>
            <div><dt>简介</dt><dd>{{ siteById(selectedSite)?.description }}</dd></div>
          </dl>
          <div class="site-hardware">
            <h4>遗留设施 / 硬件</h4>
            <ul><li v-for="(h, i) in siteById(selectedSite)?.hardware" :key="i">{{ h }}</li></ul>
          </div>
        </aside>

        <!-- 左下角读数：常驻月球 -->
        <div class="moon-readout" aria-live="polite">
          <span>LUNAR ORBIT</span>
          <strong>月球</strong>
        </div>

        <!-- 右下角：纹理署名（SSS CC BY 4.0，与太阳系页同位置） -->
        <div class="moon-credits" aria-hidden="true">Solar System Scope · CC BY 4.0</div>

        <!-- 右侧信息面板：与地球 context-panel 同结构，内容详尽 -->
        <aside v-if="selectedCraft" class="context-panel" aria-label="所选飞行器详情">
          <button class="panel-close" aria-label="关闭详情" @click="selectedCraft = null">关闭</button>
          <p class="context-type">{{ craftById(selectedCraft)?.type }}</p>
          <h2>{{ craftById(selectedCraft)?.nameZh }}</h2>
          <p class="context-subtitle">{{ craftById(selectedCraft)?.nameEn }}</p>
          <p class="context-description">{{ craftById(selectedCraft)?.description }}</p>
          <dl>
            <div><dt>运营方</dt><dd>{{ craftById(selectedCraft)?.operatorName }}</dd></div>
            <div><dt>发射</dt><dd>{{ craftById(selectedCraft)?.launchDate }} · {{ craftById(selectedCraft)?.launchSite }} · {{ craftById(selectedCraft)?.launchVehicle }}</dd></div>
            <div><dt>轨道倾角</dt><dd>{{ craftById(selectedCraft)?.displayInclination }}°</dd></div>
            <div><dt>偏心率</dt><dd>{{ craftById(selectedCraft)?.displayEccentricity }}</dd></div>
            <div><dt>轨道周期</dt><dd>{{ craftById(selectedCraft)?.displayPeriod }}</dd></div>
          </dl>
          <p class="source-caption">数据来源：{{ craftById(selectedCraft)?.sourceName }}</p>
        </aside>
      </div>
    </div>
  </section>

  <!-- 下方：月球航天器搜索板块（模仿地球的航天器工作区） -->
  <section id="moon-objects" class="content-section moon-objects-section">
    <div class="page-frame">
      <div class="section-heading">
        <div><p class="section-kicker">LUNAR SPACECRAFT</p><h2><i class="sec-num">Ⅰ</i>月球航天器</h2></div>
      </div>
      <div class="catalog-workspace">
        <div class="catalog-controls">
          <label class="search-field">
            <span>名称、英文或数据来源</span>
            <input v-model="craftQuery" type="search" placeholder="输入 LRO、鹊桥…" spellcheck="false" />
          </label>
        </div>
        <div class="catalog-meta">
          <span>共 {{ filteredCrafts.length }} 个对象</span>
          <span>标称轨道参数 · 非实时星历</span>
        </div>
        <div class="object-table" role="table" aria-label="月球航天器列表">
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
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as THREE from 'three'
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'
import { MOON_HD } from '../solar/data'
import { solarTexture } from '../solar/textures'
import type { MoonLandingSite, MoonSpacecraft } from '../types'

const props = defineProps<{ revealTick?: number }>()
const emit = defineEmits<{ 'blank-click': [] }>()

const canvasHost = ref<HTMLDivElement | null>(null)
const terminatorEnabled = ref(false)
const spacecraftEnabled = ref(true)
const orbitsEnabled = ref(true)
const sitesEnabled = ref(true)
const selectedCraft = ref<string | null>(null)
const craftQuery = ref('')
const craftLabels = ref<Array<{ id: string; x: number; y: number; visible: boolean }>>([])
const siteLabels = ref<Array<{ id: string; x: number; y: number; visible: boolean }>>([])
const landingSites = ref<MoonLandingSite[]>([])
const selectedSite = ref<string | null>(null)
const siteMarkers = new Map<string, THREE.Object3D>()

/** 入场渐亮：进入边界（revealTick 递增）时置 true，0.5s 过渡；直接加载默认已亮 */
const sceneRevealed = ref(!props.revealTick)
watch(
  () => props.revealTick,
  (tick) => {
    if (tick) sceneRevealed.value = true
  },
)

/** 月球飞行器列表（API 数据驱动，镜像地球 fetch overview 模式） */
const crafts = ref<MoonSpacecraft[]>([])
const craftById = (id: string) => crafts.value.find((c) => c.id === id)

const filteredCrafts = computed(() => {
  const q = craftQuery.value.trim().toLowerCase()
  if (!q) return crafts.value
  return crafts.value.filter(
    (c) =>
      c.nameZh.toLowerCase().includes(q) ||
      c.nameEn.toLowerCase().includes(q) ||
      c.sourceName.toLowerCase().includes(q),
  )
})

// 占位（craftById 已覆盖原 helper）

/** 点击搜索结果/场景标签：选中并聚焦（滚回主视图 → 飞行器居中 → 右侧面板） */
function focusCraft(id: string) {
  selectedCraft.value = id
  document.getElementById('moon-scene')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

/** 启动聚焦动画：相机移到飞行器外侧 3.8 单位（月球在正后方作背景 → 居中且放大） */
function startCraftFocus(id: string) {
  const runtime = craftRuntimes.find((r) => r.spec.id === id)
  if (!runtime || !camera || !controls) return
  const world = runtime.dot.getWorldPosition(focusTmp).clone()
  const radial = world.clone().normalize()
  focusAnimation = {
    fromPos: camera.position.clone(),
    toPos: world.clone().addScaledVector(radial, 3.8),
    fromTarget: controls.target.clone(),
    toTarget: world.clone(),
    startedAt: performance.now(),
    duration: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 150 : 900,
  }
}

watch(selectedCraft, (id) => {
  if (id) startCraftFocus(id)
})

let renderer: THREE.WebGLRenderer | undefined
let scene: THREE.Scene | undefined
let camera: THREE.PerspectiveCamera | undefined
let controls: OrbitControls | undefined
let moonMesh: THREE.Mesh | undefined
let moonMaterial: THREE.MeshStandardMaterial | undefined
let ambientLight: THREE.AmbientLight | undefined
let sunLight: THREE.DirectionalLight | undefined
let observationLight: THREE.DirectionalLight | undefined
let resizeObserver: ResizeObserver | undefined
let frameId = 0
/** 聚焦后用户开始拖拽：注视点快速滑回月球中心（拖拽恢复绕月球旋转） */
let dragResetTarget = false
/** 飞行器聚焦动画：相机移到飞行器外侧（月球在正后方作背景，居中且放大） */
let focusAnimation: {
  fromPos: THREE.Vector3
  toPos: THREE.Vector3
  fromTarget: THREE.Vector3
  toTarget: THREE.Vector3
  startedAt: number
  duration: number
} | null = null
const focusTmp = new THREE.Vector3()
const raycaster = new THREE.Raycaster()
const pointerNDC = new THREE.Vector2()
/** 飞行器拾取球（不可见，挂在圆点上，扩大点击命中区域） */
const craftHitMeshes: THREE.Mesh[] = []

interface CraftRuntime {
  spec: MoonSpacecraft
  plane: THREE.Object3D
  dot: THREE.Object3D
  line: THREE.Line | null
  nu: number
}

const craftRuntimes: CraftRuntime[] = []

const MOON_FOV = 42
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
  camera = new THREE.PerspectiveCamera(MOON_FOV, initialWidth / initialHeight, 0.1, 2000)
  camera.position.set(0, 1.6, 9)

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

  // 无自动自转：拖拽旋转；滚轮由 onSceneWheel 按区域接管（月球上缩放、边缘滚动页面）
  controls = new OrbitControls(camera, renderer.domElement)
  controls.enableDamping = true
  controls.dampingFactor = 0.06
  controls.enablePan = false
  controls.enableZoom = false
  controls.addEventListener('start', () => {
    dragResetTarget = true
  })
  controls.minDistance = 3.5
  controls.maxDistance = 60

  // 月球本体：8k 贴图 + PBR 材质（保留质感，同地球模式）
  const texture = solarTexture(MOON_HD.textureUrl)
  texture.colorSpace = THREE.SRGBColorSpace
  texture.anisotropy = 8
  moonMaterial = new THREE.MeshStandardMaterial({ map: texture, roughness: 0.92, metalness: 0.02 })
  moonMesh = new THREE.Mesh(new THREE.SphereGeometry(2.6, 96, 96), moonMaterial)
  scene.add(moonMesh)

  // 光照（镜像地球）：固定环境光 + 太阳方向光 + 跟随相机的观测光
  //  - 晨昏线关闭（默认）：观测光照亮相机侧 → 360° 全亮（明暗边界落在球体轮廓之外）
  //  - 晨昏线打开：太阳光产生真实阴影
  ambientLight = new THREE.AmbientLight(0x24344a, 0.8)
  scene.add(ambientLight)
  // 初始即晨昏线关闭状态：观测光 3.1 + 太阳光 0（360° 全亮）
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

  // 轨道飞行器数据来自 /api/v1/moon/spacecraft（数据库 → Go → API → 前端），
  // 挂载后异步拉取并按数据构建轨道/圆点（镜像地球的数据链路）
  fetch('/api/v1/moon/spacecraft')
    .then((res) => res.json())
    .then((data: { spacecraft: MoonSpacecraft[] }) => {
      crafts.value = data.spacecraft ?? []
      if (!scene) return
      for (const spec of crafts.value) buildCraft(spec)
    })
    .catch((error) => {
      console.error('加载月球飞行器数据失败:', error)
    })
  fetch('/api/v1/moon/landing-sites')
    .then((res) => res.json())
    .then((data: { landingSites: MoonLandingSite[] }) => {
      landingSites.value = data.landingSites ?? []
      buildSiteMarkers()
    })
    .catch((error) => {
      console.error('加载月球着陆点数据失败:', error)
    })

  renderer.render(scene, camera)

  let lastTime = performance.now()
  const animate = () => {
    frameId = requestAnimationFrame(animate)
    if (!renderer || !scene || !camera) return
    const now = performance.now()
    const delta = Math.min((now - lastTime) / 1000, 0.05)
    lastTime = now

    // 航天器公转（仅绕月轨道；定点不动）
    for (const runtime of craftRuntimes) {
      if (runtime.spec.kind !== 'orbital') continue
      runtime.nu += (Math.PI * 2 / runtime.spec.periodSeconds) * delta
      const sn = runtime.spec.snapshot ?? null
      const a = sn ? sn.aKm * MOON_SCENE_SCALE : runtime.spec.orbitA
      const e = sn ? sn.eccentricity : runtime.spec.orbitE
      const r = (a * (1 - e * e)) / (1 + e * Math.cos(runtime.nu))
      runtime.dot.position.set(r * Math.cos(runtime.nu), r * Math.sin(runtime.nu), 0)
    }
    // 观测光跟随相机：明暗边界始终落在球体轮廓之外（关闭晨昏线时 360° 全亮）
    if (observationLight && camera) observationLight.position.copy(camera.position)

    // 聚焦：相机与注视点双缓动（点击瞬间飞行器居中、月球背景放大）。
    // 动画完成后不再跟随；用户拖拽时注视点滑回月球中心——拖拽始终绕月球旋转
    if (controls && camera) {
      if (dragResetTarget) {
        controls.target.lerp(new THREE.Vector3(0, 0, 0), 0.12)
        if (controls.target.lengthSq() < 0.002) {
          controls.target.set(0, 0, 0)
          dragResetTarget = false
        }
      }
      const runtime = selectedCraft.value ? craftRuntimes.find((r) => r.spec.id === selectedCraft.value) : undefined
      if (runtime && focusAnimation) {
        const world = runtime.dot.getWorldPosition(focusTmp)
        const t = Math.min(1, (now - focusAnimation.startedAt) / focusAnimation.duration)
        const eased = 1 - Math.pow(1 - t, 3)
        camera.position.lerpVectors(focusAnimation.fromPos, focusAnimation.toPos, eased)
        controls.target.lerpVectors(focusAnimation.fromTarget, world, eased)
        if (t >= 1) focusAnimation = null
      } else if (!runtime && focusAnimation) {
        // 取消选中：相机与注视点缓动回月球（默认距离 9、中心原点）
        const t = Math.min(1, (now - focusAnimation.startedAt) / focusAnimation.duration)
        const eased = 1 - Math.pow(1 - t, 3)
        camera.position.lerpVectors(focusAnimation.fromPos, focusAnimation.fromPos.clone().normalize().multiplyScalar(9), eased)
        controls.target.lerpVectors(focusAnimation.fromTarget, new THREE.Vector3(0, 0, 0), eased)
        if (t >= 1) focusAnimation = null
      }
    }

    controls?.update()
    updateLabels()
    renderer.render(scene, camera)
  }
  animate()
})

/** 场景单位 ↔ 真实尺寸：月球半径 2.6（场景）↔ 1737.4 km（真实） */
const MOON_SCENE_SCALE = 2.6 / 1737.4

/** 平近点角 → 真近点角（Kepler 方程，牛顿迭代） */
function keplerToTrueAnomaly(M: number, e: number): number {
  let E = M
  for (let k = 0; k < 8; k += 1) E = E - (E - e * Math.sin(E) - M) / (1 - e * Math.cos(E))
  return 2 * Math.atan2(Math.sqrt(1 + e) * Math.sin(E / 2), Math.sqrt(1 - e) * Math.cos(E / 2))
}

/** 按 API 数据构建单个飞行器（轨道平面/轨道线/运动点/拾取球）
 *  优先使用 JPL Horizons 日同步快照（真实形状 + 真实相位），无快照回退静态参数 */
function buildCraft(spec: MoonSpacecraft) {
  if (!scene) return
  const sn = spec.snapshot ?? null
  // 真实轨道根数（快照优先）：半长轴 km → 场景单位
  const a = sn ? sn.aKm * MOON_SCENE_SCALE : spec.orbitA
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
    new THREE.MeshBasicMaterial({ color: spec.kind === 'stationary' ? 0xf0f4f8 : 0xe6edf4 }),
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
      new THREE.LineBasicMaterial({ color: 0xb9c4cf, transparent: true, opacity: 0.5 }),
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
    // 定点：固定在月球外侧（不参与公转）
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

/** 着陆点标记：小圆点贴在月面（moonMesh 子节点，天然随球面），不参与任何旋转 */
function buildSiteMarkers() {
  if (!scene || !moonMesh) return
  for (const site of landingSites.value) {
    if (siteMarkers.has(site.id)) continue
    // 图标类型着色：astronaut 金 / rover 橙 / sample 青 / lander 银
    const color = site.icon === 'astronaut' ? 0xffcf8f : site.icon === 'rover' ? 0xffb27d : site.icon === 'sample' ? 0x8fd6c2 : 0xcfd8e2
    const marker = new THREE.Mesh(
      new THREE.SphereGeometry(0.035, 12, 12),
      new THREE.MeshBasicMaterial({ color }),
    )
    marker.position.copy(sitePosition(site.latitude, site.longitude, 2.6 * 1.004))
    marker.userData = { kind: 'landing-site', siteId: site.id }
    moonMesh.add(marker)
    siteMarkers.set(site.id, marker)

    // 月球车行驶轨迹：虚线折线（示意图，数据存库可替换真实遥测）
    if (site.track && site.track.length >= 2) {
      const points = site.track.map(([lat, lon]) => sitePosition(lat, lon, 2.6 * 1.006))
      const trackLine = new THREE.Line(
        new THREE.BufferGeometry().setFromPoints(points),
        new THREE.LineDashedMaterial({ color, dashSize: 0.055, gapSize: 0.05, transparent: true, opacity: 0.85 }),
      )
      trackLine.computeLineDistances()
      trackLine.name = `track:${site.id}`
      moonMesh.add(trackLine)
    }
  }
}

function siteById(id: string) {
  return landingSites.value.find((site) => site.id === id)
}

/** 着陆点标签样式：右侧偏移，垂直对齐圆点 */
function siteLabelStyle(label: { id: string; x: number; y: number }) {
  return { transform: `translate(calc(${label.x}px + 10px), ${label.y - 14}px)` }
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

// 晨昏线开关（镜像地球 applyDayNightMode）：
// 关闭 = 观测光(相机方向) 3.1 + 太阳光 0 → 360° 全亮；
// 打开 = 太阳光 3.1 + 观测光 0 → 真实阴影
watch(terminatorEnabled, (enabled) => {
  if (!observationLight || !sunLight) return
  observationLight.intensity = enabled ? 0 : 3.1
  sunLight.intensity = enabled ? 3.1 : 0
})

// 航天器开关：显示/隐藏飞行器圆点（标签由 v-show 联动）
watch(spacecraftEnabled, (enabled) => {
  for (const runtime of craftRuntimes) runtime.dot.visible = enabled
})
// 轨道开关：显示/隐藏轨道线
watch(orbitsEnabled, (enabled) => {
  for (const runtime of craftRuntimes) if (runtime.line) runtime.line.visible = enabled
})

const moonPointerStart = new THREE.Vector2()

/** 按下瞬间：在月球表面 → 立即收起页头并清除选中（与地球一致） */
function onPointerDown(event: PointerEvent) {
  moonPointerStart.set(event.clientX, event.clientY)
  if (isNearMoon(event.clientX, event.clientY)) {
    selectedCraft.value = null
    emit('blank-click')
  }
}

/** 松开：未拖拽（点按）→ 射线拾取飞行器（点击圆点选中）或清除；四周拖拽保持展开 */
function onPointerUp(event: PointerEvent) {
  if (!renderer || !camera) return
  if (moonPointerStart.distanceTo(new THREE.Vector2(event.clientX, event.clientY)) > 5) return
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
  selectedCraft.value = null
  emit('blank-click')
}

/** 鼠标是否在月球投影范围内（镜像地球 isNearEarth） */
function isNearMoon(clientX: number, clientY: number) {
  if (!renderer || !camera) return false
  const bounds = renderer.domElement.getBoundingClientRect()
  const projectedCenter = new THREE.Vector3(0, 0, 0).project(camera)
  const cameraRight = new THREE.Vector3(1, 0, 0)
    .applyQuaternion(camera.quaternion)
    .multiplyScalar(2.6 * 1.08)
    .project(camera)
  const centerX = bounds.left + (projectedCenter.x * 0.5 + 0.5) * bounds.width
  const centerY = bounds.top + (-projectedCenter.y * 0.5 + 0.5) * bounds.height
  const radius = Math.abs(cameraRight.x - projectedCenter.x) * bounds.width * 0.5
  return Math.hypot(clientX - centerX, clientY - centerY) <= radius * 1.12
}

/** 飞行器是否被月球遮挡：视线段（相机→飞行器）与月球球体（半径 2.6）相交 */
function isCraftOccluded(world: THREE.Vector3) {
  if (!camera) return false
  const dir = world.clone().sub(camera.position)
  const distance = dir.length()
  dir.normalize()
  // 最近点必须在视线段之内（否则是飞行器后面的月球，不算遮挡）
  const t = -camera.position.dot(dir)
  if (t <= 0 || t >= distance) return false
  const closest = camera.position.clone().addScaledVector(dir, t)
  return closest.length() < 2.6
}

/** 滚轮：在月球上 → 缩放月球；在边缘区域 → 交给页面滚动（与地球一致） */
function onSceneWheel(event: WheelEvent) {
  if (!camera || !controls || !isNearMoon(event.clientX, event.clientY)) return
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
  const halfFovTan = Math.tan((MOON_FOV / 2) * DEG)
  const tmp = new THREE.Vector3()
  const next: Array<{ id: string; x: number; y: number; visible: boolean }> = []
  for (const runtime of craftRuntimes) {
    const world = runtime.dot.getWorldPosition(tmp)
    // 先算遮挡（世界坐标），再投影（project 会原地改写向量）
    const occluded = isCraftOccluded(world)
    const p = world.project(camera)
    // 标签与圆点一体：同一遮挡判定，背面一起隐藏、正面一起出现
    runtime.dot.visible = spacecraftEnabled.value && !occluded
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
  siteLabels.value = siteNext
}

function craftLabelStyle(label: { id: string; x: number; y: number }) {
  // 标签垂直中心与圆点对齐（标签高约 28px，上移一半）
  return { transform: `translate(calc(${label.x}px + 10px), ${label.y - 14}px)` }
}

onBeforeUnmount(() => {
  cancelAnimationFrame(frameId)
  resizeObserver?.disconnect()
  renderer?.domElement.removeEventListener('wheel', onSceneWheel)
  renderer?.domElement.removeEventListener('pointerdown', onPointerDown)
  renderer?.domElement.removeEventListener('pointerup', onPointerUp)
  controls?.dispose()
  moonMaterial?.dispose()
  moonMesh?.geometry.dispose()
  renderer?.dispose()
})
</script>

<style scoped>
/* 银灰主题：月球界面统一用银灰配色（区别于地球的浅蓝） */
.moon-section {
  --moon-accent: #c8d0d8;
  --moon-accent-dim: rgba(200, 208, 216, 0.4);
  --moon-line: rgba(200, 208, 216, 0.22);
  --moon-text: #d5dce3;
  --moon-quiet: #8b959f;
  position: relative;
  height: 150dvh;
  min-height: 990px;
  /* 银灰星野：月球界面放弃浅蓝星星，统一银灰 */
  background:
    radial-gradient(1.2px 1.2px at 12% 22%, rgba(212, 218, 224, .4), transparent 100%),
    radial-gradient(1px 1px at 23% 64%, rgba(212, 218, 224, .3), transparent 100%),
    radial-gradient(.8px .8px at 31% 38%, rgba(212, 218, 224, .25), transparent 100%),
    radial-gradient(1.4px 1.4px at 41% 82%, rgba(212, 218, 224, .36), transparent 100%),
    radial-gradient(1px 1px at 55% 15%, rgba(212, 218, 224, .28), transparent 100%),
    radial-gradient(.9px .9px at 62% 48%, rgba(212, 218, 224, .24), transparent 100%),
    radial-gradient(1.3px 1.3px at 71% 74%, rgba(212, 218, 224, .32), transparent 100%),
    radial-gradient(1px 1px at 79% 29%, rgba(212, 218, 224, .26), transparent 100%),
    radial-gradient(.8px .8px at 88% 58%, rgba(212, 218, 224, .28), transparent 100%),
    radial-gradient(1.1px 1.1px at 94% 12%, rgba(212, 218, 224, .32), transparent 100%),
    radial-gradient(1px 1px at 7% 86%, rgba(212, 218, 224, .26), transparent 100%),
    radial-gradient(.9px .9px at 49% 92%, rgba(212, 218, 224, .24), transparent 100%),
    radial-gradient(1.2px 1.2px at 66% 4%, rgba(212, 218, 224, .3), transparent 100%),
    radial-gradient(ellipse at 50% 50%, #060b13 0%, #010307 100%);
}
/* 航天器板块 UI 全银灰（覆盖全局浅蓝） */
.moon-objects-section .catalog-workspace { background: #0d1217; }
.moon-objects-section .section-kicker { color: #b6bfc8; }
.moon-objects-section .sec-num { color: #aab4be; }
.moon-objects-section .catalog-controls label > span { color: #9aa4ae; }
.moon-objects-section .catalog-controls input {
  border-color: rgba(200, 208, 216, .25);
  background: #0a0f14;
  color: #e2e7ec;
}
.moon-objects-section .catalog-controls input:focus {
  border-color: rgba(200, 208, 216, .6);
  box-shadow: 0 0 0 3px rgba(200, 208, 216, .08);
}
.moon-objects-section .catalog-meta {
  border-top-color: rgba(200, 208, 216, .15);
  color: #8b959f;
}
.moon-objects-section .object-table-head,
.moon-objects-section .object-row {
  grid-template-columns: 150px minmax(260px, 1.6fr) minmax(180px, 1fr);
}
.moon-objects-section .object-table-head {
  border-top-color: rgba(200, 208, 216, .15);
  border-bottom-color: rgba(200, 208, 216, .15);
  color: #8b959f;
}
.moon-objects-section .object-row {
  border-bottom-color: rgba(200, 208, 216, .12);
  color: #aab4be;
}
.moon-objects-section .object-row:hover,
.moon-objects-section .object-row:focus-visible {
  background: rgba(200, 208, 216, .06);
  color: #e6ebf0;
}
.moon-objects-section .object-row small { color: #7c8791; }
.moon-objects-section .catalog-empty { color: #8b959f; }

/* 粘性场景区：首屏 100dvh，下滑进入航天器板块 */
.moon-scene-frame {
  position: sticky;
  top: 0;
  height: 100dvh;
  min-height: 660px;
  overflow: hidden;
}
.moon-scene-host {
  position: absolute;
  inset: 0;
  opacity: 0;
  transition: opacity 0.5s ease;
  cursor: grab;
}
.moon-scene-host.revealed {
  opacity: 1;
}
.moon-scene-host canvas { display: block; }

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
  border: 1px solid var(--moon-line);
  border-radius: 3px;
  background: rgba(6, 10, 14, .72);
  color: var(--moon-text);
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
  color: var(--moon-quiet);
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
  background: var(--moon-accent-dim);
  transform: translateY(-50%);
}
.craft-label:hover,
.craft-label.selected {
  border-color: rgba(200, 208, 216, .65);
  background: rgba(16, 22, 28, .85);
}

/* 着陆点标签：图标着色 + 银灰主题 */
.site-label { gap: 5px !important; }
.site-label .site-glyph { display: inline-flex; flex-shrink: 0; }
.site-label strong { color: #e2e8ee !important; }
.site-label small { color: var(--moon-quiet) !important; }
.site-label[class*='selected'] .site-glyph { color: #ffd9a0 !important; }
/* 图标类型颜色：astronaut 金 / rover 橙 / sample 青 / lander 银 */
.site-label .site-glyph { color: #cfd8e2; }
.site-label[data-icon='astronaut'] .site-glyph { color: #ffcf8f; }
.site-label[data-icon='rover'] .site-glyph { color: #ffb27d; }
.site-label[data-icon='sample'] .site-glyph { color: #8fd6c2; }

/* 选中着陆点信息卡（银灰主题，右侧） */
.site-panel {
  position: absolute;
  z-index: 8;
  top: 18px;
  right: 32px;
  width: clamp(300px, 22vw, 380px);
  padding: 22px 24px;
  border: 1px solid var(--moon-line);
  border-radius: 10px;
  background: rgba(10, 14, 18, .9);
  box-shadow: 0 24px 70px rgba(0, 0, 0, .5);
  backdrop-filter: blur(18px);
  color: var(--moon-text);
  font-size: 12px;
  transition: transform .3s cubic-bezier(.22, 1, .36, 1);
}
.site-panel .site-panel-close {
  position: absolute;
  top: 10px;
  right: 14px;
  border: 0;
  background: none;
  color: var(--moon-quiet);
  font-size: 18px;
  line-height: 1;
  cursor: pointer;
}
.site-panel .site-panel-close:hover { color: var(--moon-text); }
.site-panel-head { display: flex; gap: 12px; align-items: center; padding-bottom: 14px; border-bottom: 1px solid var(--moon-line); }
.site-panel-head .site-glyph.large { color: #ffd9a0; }
.site-panel-head h3 { margin: 0; font-size: 15px; font-weight: 500; color: #e8edf2; }
.site-panel-head p { margin: 3px 0 0; color: var(--moon-quiet); font: 400 10px var(--font-mono); letter-spacing: .06em; }
.site-panel dl { display: grid; gap: 8px; padding: 14px 0; margin: 0; }
.site-panel dl > div { display: grid; grid-template-columns: 64px 1fr; gap: 10px; }
.site-panel dt { color: var(--moon-quiet); font-size: 11px; }
.site-panel dd { margin: 0; color: var(--moon-text); font-size: 11px; line-height: 1.5; }
.site-hardware { padding-top: 12px; border-top: 1px solid var(--moon-line); }
.site-hardware h4 { margin: 0 0 8px; color: var(--moon-quiet); font: 500 9px var(--font-mono); letter-spacing: .12em; }
.site-hardware ul { margin: 0; padding-left: 16px; display: grid; gap: 5px; }
.site-hardware li { color: var(--moon-text); font-size: 11px; line-height: 1.5; }

/* 右下角署名（银灰，与太阳系页同位置同风格） */
.moon-credits {
  position: absolute;
  z-index: 3;
  right: 34px;
  bottom: 30px;
  color: var(--moon-quiet);
  font: 400 7px var(--font-mono);
  letter-spacing: .08em;
  text-align: right;
  pointer-events: none;
}

/* 读数区（银灰） */
.moon-readout {
  position: absolute;
  z-index: 4;
  left: 32px;
  bottom: 28px;
  color: var(--moon-text);
}
.moon-readout > span {
  color: var(--moon-quiet);
  font: 500 8px var(--font-mono);
  letter-spacing: .15em;
}
.moon-readout strong {
  display: block;
  margin-top: 5px;
  font-size: 17px;
  font-weight: 500;
}
.moon-readout p {
  max-width: 320px;
  margin: 6px 0 0;
  color: var(--moon-quiet);
  font-size: 10px;
  line-height: 1.7;
}
</style>
