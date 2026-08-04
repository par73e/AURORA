// 太阳系 Three.js 轨道示意场景：太阳固定在右上（不居中），行星静止在各自轨道上，
// 轨道线、小行星带、柯伊伯带完整绘制；34° 斜俯视构图，仅保留滚轮缩放交互。

import * as THREE from 'three'
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'
import {
  ASTEROID_BELT,
  KUIPER_BELT,
  MOON,
  planets,
  PLANET_LINE_ANGLE_DEG,
  SUN,
  SUN_RADIUS,
  SUN_ROTATION_SECONDS,
  STAR_COUNT,
  VIEW,
  rotationPeriodSeconds,
  SATURN_RING_TEXTURE_URL,
  type PlanetSpec,
} from './data'
import { solarTexture } from './textures'

export interface SolarLabel {
  kind: 'planet' | 'sun' | 'belt'
  id: string
  x: number
  y: number
  /** 屏幕上的物体半径（像素），用于把标签放在轮廓之外 */
  radiusPx: number
  visible: boolean
}

export interface SolarSceneCallbacks {
  onHover(id: string | null): void
  onSelect(id: string): void
  /** 镜头飞向地球过程中，地球放大到一定程度时触发（用于开始变暗） */
  onFlyZoom?(): void
  /** 镜头飞行结束（地球已放大到位）时触发（用于切换页面） */
  onFlyComplete?(): void
}

type LabelSink = (labels: SolarLabel[]) => void

interface PlanetRuntime {
  spec: PlanetSpec
  axial: THREE.Object3D
}

const DEG = Math.PI / 180

/** 轨道线样式：默认所有轨道同一亮度（暗淡），悬停某颗行星时该行星的轨道线变亮；
 *  太阳/月球没有轨道线（月球是卫星，不画独立轨道） */
const ORBIT_COLOR_DEFAULT = new THREE.Color(0x69b0d3)
const ORBIT_COLOR_HOVER = new THREE.Color(0xa9e2ff)
const ORBIT_OPACITY_DEFAULT = 0.4
const ORBIT_OPACITY_HOVER = 0.95

/** 拉远上限 = 当前构图距离的 32 倍（限制最小缩小比例；原 Infinity） */
const MAX_ZOOM_OUT_FACTOR = 32

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

/** 生成一颗形状不规则的小岩石（顶点沿径向随机扰动） */
function makeRockGeometry() {
  const geometry = new THREE.IcosahedronGeometry(1, 0)
  const position = geometry.attributes.position as THREE.BufferAttribute
  const vector = new THREE.Vector3()
  for (let i = 0; i < position.count; i += 1) {
    vector.fromBufferAttribute(position, i).normalize()
    const jitter = 0.68 + Math.random() * 0.58
    vector.multiplyScalar(jitter)
    position.setXYZ(i, vector.x, vector.y, vector.z)
  }
  geometry.computeVertexNormals()
  return geometry
}

function makeGlowTexture() {
  const canvas = document.createElement('canvas')
  canvas.width = 256
  canvas.height = 256
  const context = canvas.getContext('2d')
  if (!context) return null
  const gradient = context.createRadialGradient(128, 128, 0, 128, 128, 128)
  gradient.addColorStop(0, 'rgba(255, 238, 205, 1)')
  gradient.addColorStop(0.16, 'rgba(255, 196, 118, 0.55)')
  gradient.addColorStop(0.48, 'rgba(255, 150, 62, 0.14)')
  gradient.addColorStop(1, 'rgba(255, 120, 40, 0)')
  context.fillStyle = gradient
  context.fillRect(0, 0, 256, 256)
  return new THREE.CanvasTexture(canvas)
}

export class SolarSystemScene {
  private host: HTMLElement
  private callbacks: SolarSceneCallbacks
  private onLabels: LabelSink
  private renderer: THREE.WebGLRenderer
  private scene = new THREE.Scene()
  private camera: THREE.PerspectiveCamera
  private controls: OrbitControls
  private clock = new THREE.Clock()
  private frameId = 0
  private resizeObserver: ResizeObserver
  private raycaster = new THREE.Raycaster()
  private pointer = new THREE.Vector2()
  private pointerStart = new THREE.Vector2()
  private planetRuntimes = new Map<string, PlanetRuntime>()
  private planetMeshes: THREE.Mesh[] = []
  private sunMesh!: THREE.Mesh
  private beltGroups: THREE.InstancedMesh[] = []
  /** 月球锚点（场景级，跟随地球位置但不继承自转/倾角旋转） */
  private moonAnchor: THREE.Object3D | null = null
  /** 月球公转相位时钟（弧度，始终推进；排布模式仅用于投影目标） */
  private moonAngle = 0
  /** 模式切换时的月球角度过渡（easeOut，最短弧） */
  private moonTransition: { from: number; to: number; delta: number; startedAt: number; duration: number } | null = null
  private beltAnchors: THREE.Object3D[] = []
  /** 相机侧补光：让朝向视角的行星面可见（太阳光只照亮朝太阳的一面） */
  private cameraLight!: THREE.DirectionalLight
  private elapsed = 0
  private timeScale = window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 0 : 1
  private motionQuery: MediaQueryList
  private disposed = false
  /** 相机基向量（由俯仰/方位角确定） */
  private dir = new THREE.Vector3()
  private right = new THREE.Vector3()
  private upv = new THREE.Vector3()
  /** 注视点（主视角中心，锚定在小行星带位置附近） */
  private lookAt = new THREE.Vector3()
  /** 小行星带中点的世界坐标（黄道面 XZ 平面上的轨道点），作为构图锚点 */
  private beltAnchor = new THREE.Vector3()
  private fitDistance = 200
  /** 太阳系构图模式：aligned = 一字排布（小行星带锚定视角）；real = 真实公转位置（太阳居中视角） */
  private compositionMode: 'aligned' | 'real' = 'aligned'
  private hoveredId: string | null = null
  /** 各行星轨道线的材质（悬停该行星时轨道线变亮，其余保持暗淡） */
  private orbitMaterials = new Map<string, THREE.LineBasicMaterial>()
  /** 键盘导航（←/→）选中的目标 id；指针悬停优先于键盘选中（hoveredId ?? selectedId） */
  private selectedId: string | null = null
  /** 每颗行星当前展示的轨道角度（弧度，黄道面 XZ 平面，0 = +x） */
  private planetAngles = new Map<string, number>()
  /** 行星角度动画（先加速后减速） */
  private angleAnimation: {
    from: Map<string, number>
    to: Map<string, number>
    startedAt: number
    duration: number
  } | null = null
  /** 用户是否已主动拖拽/缩放（之后 resize 保留其视角，不再重置构图） */
  private userInteracted = false
  /** 飞向地球的镜头动画状态（三次贝塞尔路径：P0 → P1 → P2 → P3，控制点抬升避开火星） */
  private flyState: {
    p0: THREE.Vector3
    p1: THREE.Vector3
    p2: THREE.Vector3
    p3: THREE.Vector3
    fromTarget: THREE.Vector3
    toTarget: THREE.Vector3
    startedAt: number
    duration: number
    zoomed: boolean
    /** 反向飞行（ORBIT → 太阳系）：注视点缓动取镜像（t³），保证与正向逐帧对称 */
    reverse?: boolean
  } | undefined
  private disposables: Array<{ dispose(): void }> = []
  private textures: THREE.Texture[] = []
  private tempWorld = new THREE.Vector3()
  private tempWorldB = new THREE.Vector3()
  private tempProject = new THREE.Vector3()

  constructor(host: HTMLElement, callbacks: SolarSceneCallbacks, onLabels: LabelSink) {
    this.host = host
    this.callbacks = callbacks
    this.onLabels = onLabels

    this.motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
    this.motionQuery.addEventListener('change', this.onMotionChange)

    // 构图锚点：小行星带中点（火星与木星轨道之间）在轨道线上的世界位置
    const beltMid = (ASTEROID_BELT.inner + ASTEROID_BELT.outer) / 2
    const anchorAngle = PLANET_LINE_ANGLE_DEG * DEG
    this.beltAnchor.set(Math.cos(anchorAngle) * beltMid, 0, Math.sin(anchorAngle) * beltMid)

    const elevation = VIEW.elevationDeg * DEG
    const azimuth = VIEW.azimuthDeg * DEG
    this.dir.set(Math.sin(azimuth) * Math.cos(elevation), Math.sin(elevation), Math.cos(azimuth) * Math.cos(elevation))
    this.right.set(Math.cos(azimuth), 0, -Math.sin(azimuth))
    this.upv.set(-Math.sin(azimuth) * Math.sin(elevation), Math.cos(elevation), -Math.cos(azimuth) * Math.sin(elevation))

    // 挂载瞬间容器可能尚未布局（0×0），用窗口尺寸兜底避免 aspect 为 NaN
    const initialAspect =
      host.clientWidth > 0 && host.clientHeight > 0 ? host.clientWidth / host.clientHeight : window.innerWidth / window.innerHeight
    // 远裁剪面 120000：容纳 200 倍推镜起点（真实模式约 5 万单位）
    this.camera = new THREE.PerspectiveCamera(VIEW.fov, initialAspect, 0.05, 120000)

    this.renderer = new THREE.WebGLRenderer({
      antialias: true,
      alpha: true,
      powerPreference: 'high-performance',
      logarithmicDepthBuffer: true, // 近 0.05 ~ 远 120000 的跨度过大，对数深度防止 z-fighting
    })
    this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
    this.renderer.setSize(host.clientWidth, host.clientHeight)
    this.renderer.outputColorSpace = THREE.SRGBColorSpace
    this.renderer.toneMapping = THREE.ACESFilmicToneMapping
    this.renderer.toneMappingExposure = 1.0
    // 与页面底色一致的清屏色：canvas 首帧/重建时不会透出白色
    this.renderer.setClearColor(0x03070c, 1)
    host.appendChild(this.renderer.domElement)

    this.controls = new OrbitControls(this.camera, this.renderer.domElement)
    this.controls.enableDamping = true
    this.controls.dampingFactor = 0.06
    // 保持构图：禁止绕中心旋转；左键拖拽为平移
    this.controls.enableRotate = false
    this.controls.enablePan = true
    this.controls.mouseButtons = { LEFT: THREE.MOUSE.PAN, MIDDLE: THREE.MOUSE.DOLLY, RIGHT: THREE.MOUSE.ROTATE }
    this.controls.zoomSpeed = 0.85
    this.controls.minDistance = VIEW.minDistance
    this.controls.addEventListener('start', this.onUserInteract)

    this.buildLighting()
    this.buildSun()
    this.buildOrbits()
    this.buildPlanets()
    this.buildBelts()
    this.buildStars()

    this.refit()

    const dom = this.renderer.domElement
    dom.addEventListener('pointerdown', this.onPointerDown)
    dom.addEventListener('pointerup', this.onPointerUp)
    dom.addEventListener('pointermove', this.onPointerMove)
    dom.addEventListener('pointerleave', this.onPointerLeave)
    dom.addEventListener('dblclick', this.onDoubleClick)

    this.resizeObserver = new ResizeObserver(this.onResize)
    this.resizeObserver.observe(host)

    // 同步预编译全部着色器：把首次渲染的编译卡顿（约 20 个 shader 程序）压缩到
    // 挂载瞬间（遮罩仍为全黑），避免推镜过程中主线程卡顿导致画面跳变
    this.renderer.compile(this.scene, this.camera)
    // 同步渲染首帧：canvas 在浏览器合成前即完成首次清除与绘制，杜绝白闪
    this.renderer.render(this.scene, this.camera)

    this.animate()
  }

  dispose() {
    cancelAnimationFrame(this.frameId)
    this.disposed = true
    this.motionQuery.removeEventListener('change', this.onMotionChange)
    this.resizeObserver.disconnect()
    const dom = this.renderer.domElement
    dom.removeEventListener('pointerdown', this.onPointerDown)
    dom.removeEventListener('pointerup', this.onPointerUp)
    dom.removeEventListener('pointermove', this.onPointerMove)
    dom.removeEventListener('pointerleave', this.onPointerLeave)
    dom.removeEventListener('dblclick', this.onDoubleClick)
    this.controls.removeEventListener('start', this.onUserInteract)
    this.controls.dispose()
    this.renderer.dispose()
    dom.remove()
    for (const belt of this.beltGroups) belt.dispose()
    for (const item of this.disposables) item.dispose()
    for (const texture of this.textures) texture.dispose()
  }

  // ---- 场景构建 ----------------------------------------------------------

  private buildLighting() {
    // 太阳点光源：decay=0 保持全图亮度一致；环境光很弱，让夜面保持黑暗但可辨轮廓。
    const sunLight = new THREE.PointLight(0xfff1dc, 2.8, 0, 0)
    this.scene.add(sunLight)
    const ambient = new THREE.AmbientLight(0x22304a, 0.5)
    this.scene.add(ambient)
    // 相机侧方向光：跟随相机位置，保证视角一侧的行星面被照亮（避免一片黑）
    this.cameraLight = new THREE.DirectionalLight(0xfff1dc, 1.1)
    this.cameraLight.position.copy(this.camera.position)
    this.scene.add(this.cameraLight)
  }

  private buildSun() {
    const texture = solarTexture(SUN.textureUrl)
    texture.colorSpace = THREE.SRGBColorSpace
    texture.anisotropy = 4
    this.textures.push(texture)

    const sunMaterial = new THREE.MeshBasicMaterial({ map: texture })
    this.sunMesh = new THREE.Mesh(new THREE.SphereGeometry(SUN_RADIUS, 64, 64), sunMaterial)
    this.sunMesh.userData = { id: 'sun' }
    this.scene.add(this.sunMesh)
    this.disposables.push(sunMaterial, this.sunMesh.geometry)

    const glowTexture = makeGlowTexture()
    if (glowTexture) {
      this.textures.push(glowTexture)
      const glow = new THREE.Sprite(
        new THREE.SpriteMaterial({
          map: glowTexture,
          color: 0xffb36b,
          transparent: true,
          opacity: 0.85,
          blending: THREE.AdditiveBlending,
          depthWrite: false,
        }),
      )
      glow.scale.setScalar(SUN_RADIUS * 2.4)
      this.scene.add(glow)
      this.disposables.push(glow.material)
    }
  }

  private buildOrbits() {
    for (const spec of planets) {
      const points: THREE.Vector3[] = []
      for (let i = 0; i < 160; i += 1) {
        const theta = (i / 160) * Math.PI * 2
        points.push(new THREE.Vector3(Math.cos(theta) * spec.orbitRadius, 0, Math.sin(theta) * spec.orbitRadius))
      }
      // 所有轨道线默认同一亮度；悬停某颗行星时该轨道线变亮（updateOrbitHighlights 每帧缓动）
      const material = new THREE.LineBasicMaterial({
        color: ORBIT_COLOR_DEFAULT.clone(), // 每颗行星独立 Color 实例，避免 lerp 互相污染
        transparent: true,
        opacity: ORBIT_OPACITY_DEFAULT,
      })
      const orbitLine = new THREE.LineLoop(new THREE.BufferGeometry().setFromPoints(points), material)
      orbitLine.userData = { id: spec.id }
      this.orbitMaterials.set(spec.id, material)
      this.scene.add(orbitLine)
      this.disposables.push(orbitLine.geometry, material)
    }
  }

  private buildPlanets() {
    const lineAngle = PLANET_LINE_ANGLE_DEG * DEG

    for (const spec of planets) {
      const texture = solarTexture(spec.textureUrl)
      texture.colorSpace = THREE.SRGBColorSpace
      texture.anisotropy = 4
      this.textures.push(texture)

      const material = new THREE.MeshStandardMaterial({ map: texture, roughness: 0.92, metalness: 0.02 })
      const mesh = new THREE.Mesh(new THREE.SphereGeometry(spec.radius, 48, 48), material)
      mesh.userData = { id: spec.id }
      this.planetMeshes.push(mesh)
      this.disposables.push(material, mesh.geometry)

      const axial = new THREE.Object3D()
      // ZYX：先自转（local Y）再倾斜，保证自转轴是倾斜后的极轴
      axial.rotation.order = 'ZYX'
      if (spec.id === 'earth') {
        // 地球：轴倾方向指向 +Z（六月夏至方向）——自转轴在空间中固定，
        // 配合真实公转位置即可呈现正确季节（8 月北半球夏至后、太阳直射点在北纬 ~17°）
        axial.rotation.x = spec.axialTiltDeg * DEG
      } else {
        // 其余行星：示意排布，绕 Z 倾斜保持观感
        axial.rotation.z = spec.axialTiltDeg * DEG
      }
      axial.add(mesh)
      if (spec.ring) {
        const ring = this.buildRing(spec)
        axial.add(ring)
      }
      if (spec.id === 'earth') {
        // 月球：小半径球体，锚点置于场景级——跟随地球位置但不随地球自转/倾角旋转
        const moonTexture = solarTexture(MOON.textureUrl)
        moonTexture.colorSpace = THREE.SRGBColorSpace
        moonTexture.anisotropy = 4
        this.textures.push(moonTexture)
        const moonMaterial = new THREE.MeshStandardMaterial({ map: moonTexture, roughness: 0.95, metalness: 0 })
        const moonMesh = new THREE.Mesh(new THREE.SphereGeometry(MOON.radius, 32, 32), moonMaterial)
        moonMesh.userData = { id: 'moon' }
        this.planetMeshes.push(moonMesh)
        this.disposables.push(moonMaterial, moonMesh.geometry)
        this.moonAnchor = new THREE.Object3D()
        this.moonAnchor.add(moonMesh)
        this.scene.add(this.moonAnchor)
      }
      this.scene.add(axial)
      this.planetRuntimes.set(spec.id, { spec, axial })
      // 必须在注册之后再定位（applyPlanetAngle 依赖 planetRuntimes）
      this.planetAngles.set(spec.id, lineAngle)
      this.applyPlanetAngle(spec.id)
    }
  }

  /** 按当前展示角度把行星放到轨道线上（圆形轨道：x = r·cosθ, z = r·sinθ） */
  private applyPlanetAngle(id: string) {
    const runtime = this.planetRuntimes.get(id)
    if (!runtime) return
    const angle = this.planetAngles.get(id) ?? 0
    runtime.axial.position.set(Math.cos(angle) * runtime.spec.orbitRadius, 0, Math.sin(angle) * runtime.spec.orbitRadius)
  }

  /** J2000 历元（2000-01-01 12:00 TT）对应的毫秒数 */
  private static readonly J2000_MS = Date.UTC(2000, 0, 1, 12, 0, 0)

  /** 当前时刻的真实公转黄经（J2000 轨道根数 + 迭代解 Kepler 方程） */
  private realOrbitalAngle(spec: PlanetSpec): number {
    const days = (Date.now() - SolarSystemScene.J2000_MS) / 86400000
    const meanLongitude = (spec.meanLongitudeDeg + (360 / spec.periodDays) * days) % 360
    const M = ((((meanLongitude - spec.perihelionLongitudeDeg) % 360) + 360) % 360) * DEG
    let E = M
    for (let i = 0; i < 8; i += 1) {
      E = E - (E - spec.eccentricity * Math.sin(E) - M) / (1 - spec.eccentricity * Math.cos(E))
    }
    const nu = 2 * Math.atan2(
      Math.sqrt(1 + spec.eccentricity) * Math.sin(E / 2),
      Math.sqrt(1 - spec.eccentricity) * Math.cos(E / 2),
    )
    return (nu + spec.perihelionLongitudeDeg * DEG) % (Math.PI * 2)
  }

  /** 切换到"真实公转位置"模式：行星转到位 + 月球滑入公转相位 + 镜头飞向太阳居中构图 */
  animateToRealPositions() {
    // 起始角取当前显示值（模式未翻转）；目标按目标模式显式传入
    this.startMoonTransition('real')
    this.compositionMode = 'real'
    this.startAngleAnimation((spec) => this.realOrbitalAngle(spec))
    this.flyToModeComposition()
  }

  /** 切换回"一字排布"模式：行星归位 + 月球滑回视角左侧 + 镜头飞回小行星带构图 */
  animateToAligned() {
    this.startMoonTransition('aligned')
    this.compositionMode = 'aligned'
    this.startAngleAnimation(() => PLANET_LINE_ANGLE_DEG * DEG)
    this.flyToModeComposition()
  }

  /** 月球角度过渡（easeOut）：起始角 = 当前显示值；目标按目标模式计算（真实 = 相位外推），走最短弧 */
  private startMoonTransition(targetMode: 'aligned' | 'real') {
    const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    const duration = reduced ? 200 : 1600
    const from = this.currentMoonDisplayAngle()
    const target =
      targetMode === 'real'
        ? this.moonAngle + (Math.PI * 2 / MOON.orbitSeconds) * (duration / 1000)
        : Math.PI
    // 最短弧：差值归一化到 [-π, π]，避免相位累积导致的多圈倒退
    let delta = target - from
    delta = ((((delta + Math.PI) % (Math.PI * 2)) + Math.PI * 2) % (Math.PI * 2)) - Math.PI
    this.moonTransition = { from, to: from + delta, delta, startedAt: performance.now(), duration }
  }

  /** 当前月球显示角度（过渡中取插值） */
  private currentMoonDisplayAngle(): number {
    if (this.moonTransition) {
      const t = Math.min(1, (performance.now() - this.moonTransition.startedAt) / this.moonTransition.duration)
      const eased = 1 - Math.pow(1 - t, 3)
      return this.moonTransition.from + this.moonTransition.delta * eased
    }
    return this.compositionMode === 'real' ? this.moonAngle : Math.PI
  }

  /** 直接放置真实位置 + 太阳居中构图（无动画，用于恢复会话记忆的模式） */
  setRealPositions() {
    this.compositionMode = 'real'
    for (const spec of planets) {
      this.planetAngles.set(spec.id, this.realOrbitalAngle(spec))
      this.applyPlanetAngle(spec.id)
    }
    this.angleAnimation = null
    const host = this.host
    const aspect = host.clientWidth / host.clientHeight
    this.refit()
  }

  /** 镜头沿三次贝塞尔飞向当前模式的目标构图（与行星角度动画同速：1.6s easeInOut） */
  private flyToModeComposition() {
    if (this.flyState) return
    const host = this.host
    const width = host.clientWidth
    const height = host.clientHeight
    if (width === 0 || height === 0) return
    const aspect = width / height
    const { target: destTarget, distance } = this.computeModeComposition(aspect)
    this.fitDistance = distance
    const destPosition = destTarget.clone().addScaledVector(this.dir, distance)
    // 三次贝塞尔拟合：控制点沿位移方向推进，起止切线与位移方向一致
    const p0 = this.camera.position.clone()
    const p3 = destPosition
    const delta = p3.clone().sub(p0)
    const p1 = p0.clone().addScaledVector(delta, 0.3)
    const p2 = p0.clone().addScaledVector(delta, 0.68)
    this.controls.minDistance = VIEW.minDistance
    this.controls.maxDistance = this.fitDistance * MAX_ZOOM_OUT_FACTOR // 拉远上限：构图距离的 32 倍
    this.flyState = {
      p0,
      p1,
      p2,
      p3,
      fromTarget: this.controls.target.clone(),
      toTarget: destTarget,
      startedAt: performance.now(),
      duration: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 250 : 1600,
      zoomed: true, // 模式切换不触发变暗事件
      reverse: true, // 注视点与位置共用同一缓动（锁步运动）
    }
    this.controls.enabled = false
    this.userInteracted = false
  }

  private startAngleAnimation(targetFor: (spec: PlanetSpec) => number) {
    const from = new Map<string, number>()
    const to = new Map<string, number>()
    for (const spec of planets) {
      const current = this.planetAngles.get(spec.id) ?? 0
      let target = targetFor(spec) % (Math.PI * 2)
      if (target < 0) target += Math.PI * 2
      // 最短路径：把差值归一化到 [-π, π]
      const delta = ((((target - current) + Math.PI) % (Math.PI * 2)) + Math.PI * 2) % (Math.PI * 2) - Math.PI
      from.set(spec.id, current)
      to.set(spec.id, current + delta)
    }
    this.angleAnimation = {
      from,
      to,
      startedAt: performance.now(),
      duration: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 200 : 1600,
    }
  }

  /** 行星角度动画驱动：easeInOutCubic，先加速后减速 */
  private updateAngleAnimation() {
    const anim = this.angleAnimation
    if (!anim) return
    const t = Math.min(1, (performance.now() - anim.startedAt) / anim.duration)
    const eased = t < 0.5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2
    for (const spec of planets) {
      const from = anim.from.get(spec.id) ?? 0
      const to = anim.to.get(spec.id) ?? 0
      this.planetAngles.set(spec.id, from + (to - from) * eased)
      this.applyPlanetAngle(spec.id)
    }
    if (t >= 1) this.angleAnimation = null
  }

  private buildRing(spec: PlanetSpec) {
    const ringSpec = spec.ring!
    const inner = spec.radius * ringSpec.inner
    const outer = spec.radius * ringSpec.outer
    const geometry = radialRingGeometry(inner, outer, 128)

    if (ringSpec.kind === 'saturn') {
      const material = new THREE.MeshBasicMaterial({
        color: 0xd8c9a3,
        transparent: true,
        opacity: 0.9,
        side: THREE.DoubleSide,
        depthWrite: false,
      })
      const ring = new THREE.Mesh(geometry, material)
      ring.userData = { id: spec.id }
      const ringTexture = solarTexture(SATURN_RING_TEXTURE_URL, (texture) => {
        if (this.disposed) {
          texture.dispose()
          return
        }
        material.map = texture
        material.needsUpdate = true
      })
      ringTexture.colorSpace = THREE.SRGBColorSpace
      ringTexture.anisotropy = 4
      this.textures.push(ringTexture)
      // 预热已就绪：同步挂上贴图，让构造器预编译直接覆盖带贴图程序
      if (ringTexture.image) {
        material.map = ringTexture
        material.needsUpdate = true
      }
      this.disposables.push(material, geometry)
      return ring
    }

    const material = new THREE.MeshBasicMaterial({
      color: 0x9fd9de,
      transparent: true,
      opacity: 0.26,
      side: THREE.DoubleSide,
      depthWrite: false,
    })
    const ring = new THREE.Mesh(geometry, material)
    ring.userData = { id: spec.id }
    this.disposables.push(material, geometry)
    return ring
  }

  private buildBelts() {
    const rockGeometry = makeRockGeometry()
    this.disposables.push(rockGeometry)

    const asteroidMaterial = new THREE.MeshStandardMaterial({ color: ASTEROID_BELT.color, roughness: 1, metalness: 0 })
    const kuiperMaterial = new THREE.MeshStandardMaterial({ color: KUIPER_BELT.color, roughness: 1, metalness: 0 })
    this.disposables.push(asteroidMaterial, kuiperMaterial)

    const asteroidGroup = this.createBelt(rockGeometry, asteroidMaterial, ASTEROID_BELT.count, ASTEROID_BELT.inner, ASTEROID_BELT.outer, ASTEROID_BELT.sizeMin, ASTEROID_BELT.sizeMax, ASTEROID_BELT.spreadY)
    const kuiperGroup = this.createBelt(rockGeometry, kuiperMaterial, KUIPER_BELT.count, KUIPER_BELT.inner, KUIPER_BELT.outer, KUIPER_BELT.sizeMin, KUIPER_BELT.sizeMax, KUIPER_BELT.spreadY)
    this.scene.add(asteroidGroup, kuiperGroup)
    this.beltGroups.push(asteroidGroup, kuiperGroup)

    // 标签锚点：位于各自环带的偏左上角（默认视角 elevation 28°/azimuth -35° 下，
    // 屏幕左上对应世界方位角 ≈ 240°——随环带跟随，转动视角时仍在环上）
    const beltLabelAngle = 240 * DEG
    const asteroidAnchor = new THREE.Object3D()
    asteroidAnchor.name = 'asteroid-belt'
    asteroidAnchor.position.set(Math.cos(beltLabelAngle) * (ASTEROID_BELT.outer + 0.4), 2.6, Math.sin(beltLabelAngle) * (ASTEROID_BELT.outer + 0.4))
    const kuiperAnchor = new THREE.Object3D()
    kuiperAnchor.name = 'kuiper-belt'
    kuiperAnchor.position.set(Math.cos(beltLabelAngle) * (KUIPER_BELT.outer + 2), 3.4, Math.sin(beltLabelAngle) * (KUIPER_BELT.outer + 2))
    this.scene.add(asteroidAnchor, kuiperAnchor)
    this.beltAnchors.push(asteroidAnchor, kuiperAnchor)
  }

  private createBelt(geometry: THREE.BufferGeometry, material: THREE.Material, count: number, inner: number, outer: number, sizeMin: number, sizeMax: number, spreadY: number) {
    const mesh = new THREE.InstancedMesh(geometry, material, count)
    const matrix = new THREE.Matrix4()
    const quaternion = new THREE.Quaternion()
    const euler = new THREE.Euler()
    const position = new THREE.Vector3()
    const scale = new THREE.Vector3()
    for (let i = 0; i < count; i += 1) {
      const angle = Math.random() * Math.PI * 2
      const t = 0.15 + 0.7 * Math.random()
      const radius = inner + (outer - inner) * t
      position.set(Math.cos(angle) * radius, (Math.random() - 0.5) * spreadY, Math.sin(angle) * radius)
      euler.set(Math.random() * Math.PI, Math.random() * Math.PI, Math.random() * Math.PI)
      quaternion.setFromEuler(euler)
      const size = sizeMin + (sizeMax - sizeMin) * Math.pow(Math.random(), 1.6)
      scale.setScalar(size)
      matrix.compose(position, quaternion, scale)
      mesh.setMatrixAt(i, matrix)
    }
    mesh.instanceMatrix.setUsage(THREE.StaticDrawUsage)
    mesh.instanceMatrix.needsUpdate = true
    mesh.frustumCulled = false
    return mesh
  }

  private buildStars() {
    const dimGeometry = new THREE.BufferGeometry()
    const dimPositions = new Float32Array(STAR_COUNT * 3)
    for (let i = 0; i < STAR_COUNT; i += 1) {
      const radius = 700 + Math.random() * 400
      const theta = Math.random() * Math.PI * 2
      const phi = Math.acos(2 * Math.random() - 1)
      dimPositions[i * 3] = radius * Math.sin(phi) * Math.cos(theta)
      dimPositions[i * 3 + 1] = radius * Math.cos(phi)
      dimPositions[i * 3 + 2] = radius * Math.sin(phi) * Math.sin(theta)
    }
    dimGeometry.setAttribute('position', new THREE.BufferAttribute(dimPositions, 3))
    const dimMaterial = new THREE.PointsMaterial({
      color: 0xa6c9df,
      size: 0.9,
      sizeAttenuation: true,
      transparent: true,
      opacity: 0.8,
      depthWrite: false,
    })
    const dimStars = new THREE.Points(dimGeometry, dimMaterial)
    this.scene.add(dimStars)
    this.disposables.push(dimGeometry, dimMaterial)

    const brightCount = 420
    const brightGeometry = new THREE.BufferGeometry()
    const brightPositions = new Float32Array(brightCount * 3)
    const brightColors = new Float32Array(brightCount * 3)
    const warm = new THREE.Color(0xffe3b8)
    const cool = new THREE.Color(0xcfe8f5)
    for (let i = 0; i < brightCount; i += 1) {
      const radius = 700 + Math.random() * 400
      const theta = Math.random() * Math.PI * 2
      const phi = Math.acos(2 * Math.random() - 1)
      brightPositions[i * 3] = radius * Math.sin(phi) * Math.cos(theta)
      brightPositions[i * 3 + 1] = radius * Math.cos(phi)
      brightPositions[i * 3 + 2] = radius * Math.sin(phi) * Math.sin(theta)
      const color = Math.random() < 0.35 ? warm : cool
      brightColors[i * 3] = color.r
      brightColors[i * 3 + 1] = color.g
      brightColors[i * 3 + 2] = color.b
    }
    brightGeometry.setAttribute('position', new THREE.BufferAttribute(brightPositions, 3))
    brightGeometry.setAttribute('color', new THREE.BufferAttribute(brightColors, 3))
    const brightMaterial = new THREE.PointsMaterial({
      vertexColors: true,
      size: 1.9,
      sizeAttenuation: true,
      transparent: true,
      opacity: 0.95,
      depthWrite: false,
    })
    const brightStars = new THREE.Points(brightGeometry, brightMaterial)
    this.scene.add(brightStars)
    this.disposables.push(brightGeometry, brightMaterial)
  }

  // ---- 帧循环 ------------------------------------------------------------

  private animate = () => {
    this.frameId = requestAnimationFrame(this.animate)
    try {
      this.tick()
    } catch (error) {
      console.error('SolarSystemScene animate 异常:', error)
    }
  }

  private tick() {
    const rawDelta = Math.min(this.clock.getDelta(), 0.05)
    const delta = rawDelta * this.timeScale
    this.elapsed += delta

    this.sunMesh.rotation.y += (Math.PI * 2 / SUN_ROTATION_SECONDS) * delta
    // 相机侧补光跟随相机
    this.cameraLight.position.copy(this.camera.position)
    for (const runtime of this.planetRuntimes.values()) {
      runtime.axial.rotation.y += (Math.PI * 2 / rotationPeriodSeconds(runtime.spec.rotationHours)) * delta
    }
    // 月球：相位时钟始终推进；显示角度按模式/过渡决定（排布 = π 固定视角左侧）
    this.moonAngle += (Math.PI * 2 / MOON.orbitSeconds) * delta
    let moonDisplayAngle = this.compositionMode === 'real' ? this.moonAngle : Math.PI
    if (this.moonTransition) {
      const t = Math.min(1, (performance.now() - this.moonTransition.startedAt) / this.moonTransition.duration)
      // easeOut：起步快、末端柔和落定，避免 easeInOut 末尾爬行造成的"卡一下"感
      const eased = 1 - Math.pow(1 - t, 3)
      moonDisplayAngle = this.moonTransition.from + this.moonTransition.delta * eased
      if (t >= 1) {
        // 仅真实模式收尾时对齐相位时钟（消除漂移跳变）；
        // 排布模式收尾不动时钟——相位被保留，下次进真实模式落点各不相同
        if (this.compositionMode === 'real') this.moonAngle = this.moonTransition.to
        this.moonTransition = null
      }
    }
    const earthRuntime = this.planetRuntimes.get('earth')
    if (earthRuntime && this.moonAnchor) {
      const earthPos = earthRuntime.axial.getWorldPosition(this.tempWorld)
      this.moonAnchor.position.set(
        earthPos.x + Math.cos(moonDisplayAngle) * MOON.distance,
        earthPos.y,
        earthPos.z + Math.sin(moonDisplayAngle) * MOON.distance,
      )
    }
    const periods = [ASTEROID_BELT.periodSeconds, KUIPER_BELT.periodSeconds]
    this.beltGroups.forEach((group, index) => {
      group.rotation.y += (Math.PI * 2 / periods[index]) * delta
    })
    if (this.pendingEntryFly && this.host.clientWidth > 0 && this.host.clientHeight > 0) {
      const pending = this.pendingEntryFly
      this.pendingEntryFly = null
      this.startEntryFly(pending.delayMs)
    }
    this.updateAngleAnimation()
    this.updateFly()
    // 自适应灵敏度：OrbitControls 的旋转/缩放是"每像素固定角度/等比"，距离越近
    // 同样的角度在屏幕上的位移越大，操作会显得迟钝——按距离动态提速补偿
    if (!this.flyState) {
      const dist = this.camera.position.distanceTo(this.controls.target)
      const t = THREE.MathUtils.clamp(dist / this.fitDistance, 0, 1)
      // 近端 boost（rotate/zoom）：小目标放大时相机更近 → boost≈1，限制上限避免过灵
      const boost = 1 - t * t
      this.controls.rotateSpeed = 1.5 + boost * 4
      this.controls.zoomSpeed = 1.2 + boost * 1.6
      // pan 灵敏度补偿（左键拖动）：OrbitControls 的 pan 位移 ∝ 相机到注视点距离——
      // 内圈行星（相机近，dist 小）pan 慢、边缘（dist 大）pan 快，正是"放大内行星
      // 灵敏度低、边缘过高"的来源。panSpeed ∝ 1/dist 抵消 → 全距离屏幕手感一致
      // （构图处为原手感；贴脸时 dist 下限 4 防极端）
      this.controls.panSpeed = THREE.MathUtils.clamp(this.fitDistance / Math.max(dist, 4), 0.4, 6)
    }
    this.controls.update()
    // 轨道线悬停高亮：悬停行星的轨道每帧向高亮样式缓动，其余回到暗淡
    this.updateOrbitHighlights(rawDelta)
    // 同步注视点：平移会移动 controls.target，标签与 resize 逻辑依赖 lookAt
    this.lookAt.copy(this.controls.target)
    this.updateLabels()
    this.renderer.render(this.scene, this.camera)
  }

  /** 轨道线悬停高亮：悬停行星的轨道线变亮、其余保持暗淡。
   *  每帧按 rawDelta 指数缓动（与时间缩放无关）；reduced-motion 下直接切换（同 veil 处理） */
  private updateOrbitHighlights(rawDelta: number) {
    // 悬停优先于键盘选中：指针在某颗行星上时高亮跟随指针；指针离开后回到键盘选中的目标
    const highlightId = this.hoveredId ?? this.selectedId
    // timeScale 为 0（系统减弱动态效果）：不做缓动，状态直接切换
    const factor = this.timeScale === 0 ? 1 : 1 - Math.exp(-rawDelta * 10)
    for (const [id, material] of this.orbitMaterials) {
      const active = id === highlightId
      const targetOpacity = active ? ORBIT_OPACITY_HOVER : ORBIT_OPACITY_DEFAULT
      material.opacity += (targetOpacity - material.opacity) * factor
      material.color.lerp(active ? ORBIT_COLOR_HOVER : ORBIT_COLOR_DEFAULT, factor)
    }
  }

  // ---- 标签投影 ----------------------------------------------------------

  private projectToScreen(x: number, y: number, z: number, width: number, height: number) {
    this.tempProject.set(x, y, z).project(this.camera)
    return {
      x: (this.tempProject.x * 0.5 + 0.5) * width,
      y: (-this.tempProject.y * 0.5 + 0.5) * height,
      z: this.tempProject.z,
    }
  }

  private projectLabel(kind: SolarLabel['kind'], id: string, worldPosition: THREE.Vector3, radius: number, width: number, height: number, halfFovTan: number): SolarLabel {
    const projected = worldPosition.clone().project(this.camera)
    const distance = this.camera.position.distanceTo(worldPosition)
    const radiusPx = (radius / (distance * halfFovTan)) * (height / 2)
    return {
      kind,
      id,
      x: (projected.x * 0.5 + 0.5) * width,
      y: (-projected.y * 0.5 + 0.5) * height,
      radiusPx,
      visible: projected.z > -1 && projected.z < 1 && radiusPx > 0.4,
      opacity: 1,
    }
  }

  private updateLabels() {
    const width = this.host.clientWidth
    const height = this.host.clientHeight
    if (width === 0 || height === 0) return
    const halfFovTan = Math.tan((VIEW.fov / 2) * DEG)
    const labels: SolarLabel[] = []
    // 入场推镜：标签在推进 15%–60% 之间渐显（前期行星挤在中央，标签会叠成一团）
    const labelOpacity = this.entryFlyEased === null ? 1 : THREE.MathUtils.clamp((this.entryFlyEased - 0.15) / 0.45, 0, 1)

    labels.push(this.projectLabel('sun', 'sun', this.tempWorld.set(0, 0, 0), SUN_RADIUS, width, height, halfFovTan))

    for (const runtime of this.planetRuntimes.values()) {
      const world = runtime.axial.getWorldPosition(this.tempWorld)
      labels.push(this.projectLabel('planet', runtime.spec.id, world, runtime.spec.radius, width, height, halfFovTan))
    }

    if (this.moonAnchor) {
      const world = this.moonAnchor.getWorldPosition(this.tempWorldB)
      labels.push(this.projectLabel('planet', 'moon', world, MOON.radius, width, height, halfFovTan))
    }

    for (const anchor of this.beltAnchors) {
      const world = anchor.getWorldPosition(this.tempWorldB)
      labels.push(this.projectLabel('belt', anchor.name, world, 1.6, width, height, halfFovTan))
    }

    for (const label of labels) label.opacity = labelOpacity
    this.onLabels(labels)
  }

  /** 封面入场：镜头从远端沿视线方向飞入当前模式的默认构图（由远及近） */
  /** 封面入场推镜：delayMs 毫秒后从 40 倍远处匀速高速冲入默认构图（延迟期停在起点）。
   *  挂载瞬间容器可能 0×0（尚未布局），此时挂起等待，容器有尺寸后自动启动 */
  flyInFromDistance(delayMs = 0) {
    if (this.flyState) return
    const width = this.host.clientWidth
    const height = this.host.clientHeight
    if (width === 0 || height === 0) {
      this.pendingEntryFly = { delayMs }
      return
    }
    this.startEntryFly(delayMs)
  }

  /** 真正启动入场推镜（容器尺寸已就绪） */
  private startEntryFly(delayMs: number) {
    if (this.flyState) return
    const host = this.host
    const width = host.clientWidth
    const height = host.clientHeight
    const aspect = width / height
    const { target, distance } = this.computeModeComposition(aspect)
    this.fitDistance = distance
    // 起点：15 倍构图距离（原 20×——揭幕首帧太远；12× 后揭幕首帧约 6× 正合适，
    // 但起点太近纵深不足，故取 15× 折中：起点仍有由远及近的纵深，
    // 推镜 1.3s（原 1.6s——推进偏慢）下揭幕首帧（约 46% 进度）仍约 6×。
    // 想更近/更快可调小（如 12 / 1200），想更远/更缓可调大）
    const destPosition = target.clone().addScaledVector(this.dir, distance)
    const p0 = target.clone().addScaledVector(this.dir, distance * 15)
    const delta = destPosition.clone().sub(p0)
    const p1 = p0.clone().addScaledVector(delta, 0.3)
    const p2 = p0.clone().addScaledVector(delta, 0.68)
    this.controls.minDistance = VIEW.minDistance
    this.controls.maxDistance = this.fitDistance * MAX_ZOOM_OUT_FACTOR // 拉远上限：构图距离的 32 倍
    this.flyState = {
      p0,
      p1,
      p2,
      p3: destPosition,
      fromTarget: target.clone(),
      toTarget: target.clone(),
      // startedAt 带延迟：全黑期间镜头停在起点，延迟结束才开始推进
      startedAt: performance.now() + delayMs,
      // 1.6s 推镜：整体入场压缩到 1.6s——黑幕 600ms（挂载/编译/首帧都在其中，可见时不卡顿）
      // + 渐亮 0.9s（600→1500ms），渐亮完全结束时推镜约 94%（放大到最大之前一点点）
      duration: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 250 : 1600,
      zoomed: true,
      reverse: true,
      entry: true,
      easeOut: 'linear-out',
    }
    this.controls.enabled = false
    this.userInteracted = false
  }

  // ---- 相机与构图 --------------------------------------------------------

  /** 当前模式的目标构图：aligned = 小行星带锚定（右上太阳/对角线行星）；
   *  real = 太阳居中，画面左右边界刚好到达柯伊伯带外缘（+4% 微边距） */
  private computeModeComposition(aspect: number): { target: THREE.Vector3; distance: number } {
    if (this.compositionMode === 'real') {
      const tanHalfV = Math.tan((VIEW.fov / 2) * DEG)
      // 距离由水平方向决定：边界到达柯伊伯带外缘（0.92 = 略微拉近放大）
      const distance = (KUIPER_BELT.outer / (tanHalfV * aspect)) * VIEW.realFitMargin
      // 太阳锚定在 40%（从下往上 60%），椭圆中心随之、短轴竖向居中平衡
      const beta = (2 * VIEW.realAnchorScreenY - 1) * tanHalfV
      const target = new THREE.Vector3(0, 0, 0).addScaledVector(this.upv, distance * beta)
      return { target, distance }
    }
    const tanHalf = Math.tan((VIEW.fov / 2) * DEG)
    const ndcX = 2 * VIEW.anchorScreenX - 1
    const ndcY = -(2 * VIEW.anchorScreenY - 1)
    const alpha = -ndcX * tanHalf * aspect
    const beta = -ndcY * tanHalf
    const target = this.beltAnchor
      .clone()
      .addScaledVector(this.right, VIEW.composeMinDistance * alpha)
      .addScaledVector(this.upv, VIEW.composeMinDistance * beta)
    return { target, distance: VIEW.composeMinDistance }
  }

  /** 数值构图：按当前模式摆放默认视角 */
  private refit() {
    const host = this.host
    const width = host.clientWidth
    const height = host.clientHeight
    if (width === 0 || height === 0) return
    const aspect = width / height

    const { target, distance } = this.computeModeComposition(aspect)
    this.fitDistance = distance
    this.lookAt.copy(target)
    this.camera.position.copy(this.lookAt).addScaledVector(this.dir, distance)
    this.camera.lookAt(this.lookAt)
    this.camera.updateMatrixWorld(true)
    this.controls.target.copy(this.lookAt)
    this.controls.minDistance = VIEW.minDistance
    // 拉远上限 600：星空球半径 700–1100，超出会穿出星幕坠入虚空
    this.controls.maxDistance = this.fitDistance * MAX_ZOOM_OUT_FACTOR // 拉远上限：构图距离的 32 倍
  }

  private onResize = () => {
    const host = this.host
    if (host.clientWidth === 0 || host.clientHeight === 0) return
    this.camera.aspect = host.clientWidth / host.clientHeight
    this.camera.updateProjectionMatrix()
    this.renderer.setSize(host.clientWidth, host.clientHeight)
    if (this.flyState) return // 飞行中不重置镜头
    if (!this.userInteracted) {
      // 用户尚未操作：resize 后沿用默认构图
      this.refit()
      return
    }
    // 用户已操作：保留当前取景方向与距离，仅限制在合法范围
    const offset = this.camera.position.clone().sub(this.lookAt)
    const length = offset.length()
    if (length < 0.01) return
    const clamped = Math.max(length, VIEW.minDistance) // 仅保留最小距离限制
    this.camera.position.copy(this.lookAt).addScaledVector(offset.normalize(), clamped)
    this.camera.lookAt(this.lookAt)
    this.camera.updateMatrixWorld(true)
    this.controls.target.copy(this.lookAt)
  }

  /** 用户首次拖拽/缩放时标记，之后不再重置其视角 */
  private onUserInteract = () => {
    this.userInteracted = true
  }

  /** 计算飞向地球的路径（起点 = 当前相机位置，终点 = 地球近景） */
  private computeEarthFlyPath() {
    const earthRuntime = this.planetRuntimes.get('earth')
    if (!earthRuntime) return null
    const earthPosition = earthRuntime.axial.getWorldPosition(this.tempWorldB)
    const p0 = this.camera.position.clone()
    const earthDir = earthPosition.clone().normalize()
    // 终点：太阳→地球连线上、距地球中心 17.5（地球视半径约 8.2°，带 1 单位仰角）
    const p3 = earthPosition.clone().addScaledVector(earthDir, -17.5)
    p3.y += 1
    // 控制点整体抬升：路径保持在高空滑过火星与小行星带，再俯冲进入地球
    const delta = p3.clone().sub(p0)
    const p1 = p0.clone().addScaledVector(delta, 0.3)
    p1.y += 4
    const p2 = p0.clone().addScaledVector(delta, 0.72)
    p2.y += 1.5
    return { p0, p1, p2, p3, fromTarget: this.controls.target.clone(), toTarget: earthPosition.clone() }
  }

  /** 点击地球：镜头沿抬升的三次贝塞尔路径推近——避开与地球同在行星连线上的火星，
   *  末端俯冲进地球（随后由调用方切页） */
  /** 点击月球：镜头沿抬升的三次贝塞尔路径推近月球（终点在月球近旁） */
  flyToMoon() {
    if (this.flyState) return
    const moonPosition = this.moonAnchor?.getWorldPosition(this.tempWorldB)
    if (!moonPosition) return
    const p0 = this.camera.position.clone()
    const moonDir = moonPosition.clone().normalize()
    // 终点：太阳→月球连线上、距月球中心 5.2（视半径约 7.3°，占画面 ~35%——
    // 比月球页初始（~51%）小，变黑衔接自然；带 1 单位仰角）
    const p3 = moonPosition.clone().addScaledVector(moonDir, -5.2)
    p3.y += 1
    const delta = p3.clone().sub(p0)
    const p1 = p0.clone().addScaledVector(delta, 0.3)
    p1.y += 1.5
    const p2 = p0.clone().addScaledVector(delta, 0.72)
    p2.y += 0.5
    this.controls.minDistance = 3
    this.flyState = {
      p0,
      p1,
      p2,
      p3,
      fromTarget: this.controls.target.clone(),
      toTarget: moonPosition.clone(),
      startedAt: performance.now(),
      duration: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 250 : 1300,
      zoomed: false,
      enterPlanet: true, // 仅此运镜完成时触发进入回调（onFlyComplete）
    }
    this.controls.enabled = false
  }

  /** 反向飞行（月球 → 太阳系）：从月球近景拉回默认构图 */
  flyFromMoon() {
    if (this.flyState) return
    const moonPosition = this.moonAnchor?.getWorldPosition(this.tempWorldB)
    if (!moonPosition) return
    // 终点：按当前模式构图（排布 = 小行星带锚定；真实位置 = 太阳居中）
    const aspect = this.host.clientWidth / this.host.clientHeight
    const { target, distance } = this.computeModeComposition(aspect)
    this.fitDistance = distance
    this.lookAt.copy(target)
    this.camera.position.copy(this.lookAt).addScaledVector(this.dir, distance)
    this.camera.lookAt(this.lookAt)
    this.camera.updateMatrixWorld(true)
    this.controls.target.copy(this.lookAt)
    const p3 = this.camera.position.clone()
    const toTarget = this.lookAt.clone()
    // 起点：从月球沿"朝向默认视角"的水平方向外移 6 单位、带 1 单位仰角
    const viewerDir = new THREE.Vector3(p3.x - moonPosition.x, 0, p3.z - moonPosition.z).normalize()
    const p0 = moonPosition.clone().addScaledVector(viewerDir, 6)
    p0.y += 1
    this.camera.position.copy(p0)
    this.camera.lookAt(moonPosition)
    this.controls.target.copy(moonPosition)
    this.controls.minDistance = 3
    const delta = p3.clone().sub(p0)
    const p1 = p0.clone().addScaledVector(delta, 0.3)
    p1.y += 1.5
    const p2 = p0.clone().addScaledVector(delta, 0.72)
    p2.y += 0.5
    this.flyState = {
      p0,
      p1,
      p2,
      p3,
      fromTarget: moonPosition.clone(),
      toTarget,
      startedAt: performance.now(),
      duration: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 250 : 1300,
      zoomed: true,
      reverse: true,
    }
    this.controls.enabled = false
  }

  flyToEarth() {
    if (this.flyState) return
    const path = this.computeEarthFlyPath()
    if (!path) return
    // OrbitControls.update() 每帧都会把相机距目标的距离钳制在 [minDistance, maxDistance]；
    // 飞行终点距地球 17.5 远大于近限 2，无需放宽；此处仍保留近限兜底
    this.controls.minDistance = 3
    this.flyState = {
      ...path,
      startedAt: performance.now(),
      // 1.6s 推镜（用户感知速度：由远及近缓推）
      duration: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 250 : 1600,
      zoomed: false,
      enterPlanet: true, // 仅此运镜完成时触发进入回调（onFlyComplete）
    }
    this.controls.enabled = false
  }

  /** 取消飞行（回到可交互状态） */
  cancelFly() {
    if (this.flyState) {
      this.flyState = undefined
      this.controls.enabled = true
    }
  }

  /** 反向飞行（ORBIT → 太阳系）：起点在地球外侧（太阳-地球连线之外、朝向视角的一端），
   *  镜头从地球外侧拉回默认构图——太阳在视野中自然显现，不再被地球挡在镜头背后 */
  flyFromEarth() {
    if (this.flyState) return
    const earthRuntime = this.planetRuntimes.get('earth')
    if (!earthRuntime) return
    const earthPosition = earthRuntime.axial.getWorldPosition(this.tempWorldB)
    // 终点：默认初始构图（computeModeComposition——与封面入场推镜一致的视角）；
    // 运镜过程中注视点从地球（起点近景）缓动转回构图中心（reverse t³：前期紧盯地球、后期转回）
    const host = this.host
    const aspect =
      host.clientWidth > 0 && host.clientHeight > 0 ? host.clientWidth / host.clientHeight : window.innerWidth / window.innerHeight
    const { target, distance } = this.computeModeComposition(aspect)
    this.fitDistance = distance
    this.lookAt.copy(target)
    this.camera.position.copy(this.lookAt).addScaledVector(this.dir, distance)
    this.camera.lookAt(this.lookAt)
    this.camera.updateMatrixWorld(true)
    this.controls.target.copy(this.lookAt)
    const p3 = this.camera.position.clone()
    const toTarget = this.lookAt.clone()
    // 起点：从地球沿"朝向默认视角"的水平方向外移 10 单位、带 1 单位仰角
    // （位于太阳-地球连线外侧，太阳在起点即处于视野边缘，拉远时自然滑入画面）
    const viewerDir = new THREE.Vector3(p3.x - earthPosition.x, 0, p3.z - earthPosition.z).normalize()
    const p0 = earthPosition.clone().addScaledVector(viewerDir, 10)
    p0.y += 1
    this.camera.position.copy(p0)
    this.camera.lookAt(earthPosition)
    this.controls.target.copy(earthPosition)
    this.controls.minDistance = 3 // 近限兜底（终点距地球 17.5，正常不会触发）
    // 控制点抬升：路径在高空滑过火星与小行星带
    const delta = p3.clone().sub(p0)
    const p1 = p0.clone().addScaledVector(delta, 0.3)
    p1.y += 4
    const p2 = p0.clone().addScaledVector(delta, 0.72)
    p2.y += 1.5
    this.flyState = {
      p0,
      p1,
      p2,
      p3,
      fromTarget: earthPosition.clone(),
      toTarget,
      startedAt: performance.now(),
      // 1.6s 推镜（用户感知速度：由远及近缓推）
      duration: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 250 : 1600,
      zoomed: true, // 反向飞行不再触发变暗事件（遮罩由 ORBIT 侧控制）
      reverse: true, // 注视点缓动取 t³：前期紧盯地球、后期转回默认构图
    }
    this.controls.enabled = false
  }

  /** 飞行进度驱动：沿抬升的三次贝塞尔路径推进（避开火星），注视点缓动过渡 */
  private updateFly() {
    const fly = this.flyState
    if (!fly) return
    const t = Math.min(1, (performance.now() - fly.startedAt) / fly.duration)
    if (t < 0) {
      // 延迟等待期：镜头停在起点（全黑期间）
      this.camera.position.copy(fly.p0)
      this.controls.target.copy(fly.fromTarget)
      return
    }
    // 入场推镜：'linear-out' = 前 50% 匀速高速（0→70% 路径），后 50% easeOut（70%→100%）；
    // 'cubic' = 纯 easeOut；其余飞行用 easeInOut
    let eased: number
    if (fly.easeOut === 'linear-out') {
      eased = t < 0.5 ? 1.4 * t : 0.7 + 0.3 * (1 - Math.pow(1 - (t - 0.5) / 0.5, 3))
    } else if (fly.easeOut === 'cubic') {
      eased = 1 - Math.pow(1 - t, 3)
    } else {
      eased = t < 0.5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2
    }
    // B = u³·P0 + 3u²e·P1 + 3ue²·P2 + e³·P3
    const u = 1 - eased
    this.camera.position
      .copy(fly.p0).multiplyScalar(u * u * u)
      .addScaledVector(fly.p1, 3 * u * u * eased)
      .addScaledVector(fly.p2, 3 * u * eased * eased)
      .addScaledVector(fly.p3, eased * eased * eased)
    // 注视点缓动：正向用 easeOut（地球前 80% 基本滑入正中）；
    // 反向与相机位置共用同一 eased——两者锁步运动，视线角速度单调（慢→快→慢），无摆动抖动
    const targetEase = fly.reverse ? eased : 1 - Math.pow(1 - t, 3)
    this.controls.target.lerpVectors(fly.fromTarget, fly.toTarget, targetEase)
    if (!fly.zoomed && eased >= 0.6) { // 渐暗提前：镜头推进约 60% 时触发变暗（原 78% 太晚）
      fly.zoomed = true
      this.callbacks.onFlyZoom?.()
    }
    if (fly.entry) this.entryFlyEased = eased
    if (t >= 1) {
      this.flyState = undefined
      this.entryFlyEased = null
      this.controls.enabled = true
      // 白名单：仅 flyToEarth/flyToMoon（enterPlanet）完成时触发进入回调——
      // 入场推镜/模式切换构图飞行/返回运镜一律不得触发（否则自动进入地球）
      if (fly.enterPlanet) this.callbacks.onFlyComplete?.()
    }
  }

  /** 复位视角：沿平滑曲线飞回默认的斜俯视构图（飞行中不响应；供双击与外部图标点击调用） */
  resetView() {
    if (this.flyState) return
    const host = this.host
    const width = host.clientWidth
    const height = host.clientHeight
    if (width === 0 || height === 0) return
    const aspect = width / height
    const { target: destTarget, distance } = this.computeModeComposition(aspect)
    this.fitDistance = distance
    const destPosition = destTarget.clone().addScaledVector(this.dir, distance)
    // 三次贝塞尔拟合：控制点沿位移方向推进，起止切线与位移方向一致，无折角无抖动
    const p0 = this.camera.position.clone()
    const p3 = destPosition
    const delta = p3.clone().sub(p0)
    const p1 = p0.clone().addScaledVector(delta, 0.3)
    const p2 = p0.clone().addScaledVector(delta, 0.68)
    this.controls.minDistance = VIEW.minDistance
    // 拉远上限 600：星空球半径 700–1100，超出会穿出星幕坠入虚空
    this.controls.maxDistance = this.fitDistance * MAX_ZOOM_OUT_FACTOR // 拉远上限：构图距离的 32 倍
    this.flyState = {
      p0,
      p1,
      p2,
      p3,
      fromTarget: this.controls.target.clone(),
      toTarget: destTarget,
      startedAt: performance.now(),
      duration: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 250 : 900,
      zoomed: true, // 复位不触发变暗事件
      reverse: true, // 注视点与位置共用同一缓动（锁步运动，视线角速度单调）
    }
    this.controls.enabled = false
    this.userInteracted = false // 复位后视为初始状态（后续 resize 沿用默认构图）
  }

  /** 双击复位：回到默认的斜俯视构图（飞行中不响应） */
  private onDoubleClick = () => {
    this.resetView()
  }

  // ---- 交互 --------------------------------------------------------------

  private raycastFromPointer(event: PointerEvent) {
    const bounds = this.renderer.domElement.getBoundingClientRect()
    this.pointer.set(
      ((event.clientX - bounds.left) / bounds.width) * 2 - 1,
      -((event.clientY - bounds.top) / bounds.height) * 2 + 1,
    )
    this.raycaster.setFromCamera(this.pointer, this.camera)
    const hits = this.raycaster.intersectObjects([this.sunMesh, ...this.planetMeshes])
    return (hits[0]?.object.userData.id as string | undefined) ?? null
  }

  private onPointerDown = (event: PointerEvent) => {
    this.pointerStart.set(event.clientX, event.clientY)
    this.renderer.domElement.style.cursor = 'grabbing'
  }

  private onPointerUp = (event: PointerEvent) => {
    this.renderer.domElement.style.cursor = 'grab'
    if (this.pointerStart.distanceTo(new THREE.Vector2(event.clientX, event.clientY)) > 5) return
    const id = this.raycastFromPointer(event)
    if (id) this.callbacks.onSelect(id)
  }

  private onPointerMove = (event: PointerEvent) => {
    if (event.buttons !== 0) return
    const id = this.raycastFromPointer(event)
    this.renderer.domElement.style.cursor = id ? 'pointer' : 'grab'
    if (id !== this.hoveredId) {
      this.hoveredId = id
      this.callbacks.onHover(id)
    }
  }

  private onPointerLeave = () => {
    this.renderer.domElement.style.cursor = 'grab'
    if (this.hoveredId !== null) {
      this.hoveredId = null
      this.callbacks.onHover(null)
    }
  }

  /** 设置键盘导航选中的目标（←/→ 切换时由组件调用）；仅作为高亮来源，不改指针状态 */
  setSelected(id: string | null) {
    this.selectedId = id
  }

  private onMotionChange = (event: MediaQueryListEvent) => {
    this.timeScale = event.matches ? 0 : 1
  }
}
