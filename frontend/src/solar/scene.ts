// 太阳系 Three.js 轨道示意场景：太阳固定在右上（不居中），行星静止在各自轨道上，
// 轨道线、小行星带、柯伊伯带完整绘制；34° 斜俯视构图，仅保留滚轮缩放交互。

import * as THREE from 'three'
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'
import {
  ASTEROID_BELT,
  KUIPER_BELT,
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
}

type LabelSink = (labels: SolarLabel[]) => void

interface PlanetRuntime {
  spec: PlanetSpec
  axial: THREE.Object3D
}

const DEG = Math.PI / 180

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
  private hoveredId: string | null = null
  /** 用户是否已主动拖拽/缩放（之后 resize 保留其视角，不再重置构图） */
  private userInteracted = false
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

    this.camera = new THREE.PerspectiveCamera(VIEW.fov, host.clientWidth / host.clientHeight, 0.5, 4000)

    this.renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true, powerPreference: 'high-performance' })
    this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
    this.renderer.setSize(host.clientWidth, host.clientHeight)
    this.renderer.outputColorSpace = THREE.SRGBColorSpace
    this.renderer.toneMapping = THREE.ACESFilmicToneMapping
    this.renderer.toneMappingExposure = 1.0
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
    const loader = new THREE.TextureLoader()
    const texture = loader.load(SUN.textureUrl)
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
      const isEarth = spec.id === 'earth'
      const orbitLine = new THREE.LineLoop(
        new THREE.BufferGeometry().setFromPoints(points),
        new THREE.LineBasicMaterial({
          color: isEarth ? 0x6fd2ed : 0x69b0d3,
          transparent: true,
          opacity: isEarth ? 0.62 : 0.4,
        }),
      )
      orbitLine.userData = { id: spec.id }
      this.scene.add(orbitLine)
      this.disposables.push(orbitLine.geometry, orbitLine.material as THREE.Material)
    }
  }

  private buildPlanets() {
    const loader = new THREE.TextureLoader()
    const ringLoader = new THREE.TextureLoader()
    const lineAngle = PLANET_LINE_ANGLE_DEG * DEG

    for (const spec of planets) {
      const texture = loader.load(spec.textureUrl)
      texture.colorSpace = THREE.SRGBColorSpace
      texture.anisotropy = 4
      this.textures.push(texture)

      const material = new THREE.MeshStandardMaterial({ map: texture, roughness: 0.92, metalness: 0.02 })
      const mesh = new THREE.Mesh(new THREE.SphereGeometry(spec.radius, 48, 48), material)
      mesh.userData = { id: spec.id }
      this.planetMeshes.push(mesh)
      this.disposables.push(material, mesh.geometry)

      const axial = new THREE.Object3D()
      // ZYX：先自转（local Y）再倾斜（Z），保证自转轴是倾斜后的极轴
      axial.rotation.order = 'ZYX'
      axial.rotation.z = spec.axialTiltDeg * DEG
      // 行星必须落在黄道面（XZ 平面）的轨道线上：x = r·cosθ, z = r·sinθ
      axial.position.set(Math.cos(lineAngle) * spec.orbitRadius, 0, Math.sin(lineAngle) * spec.orbitRadius)
      axial.add(mesh)

      if (spec.ring) {
        const ring = this.buildRing(spec, ringLoader)
        axial.add(ring)
      }

      this.scene.add(axial)
      this.planetRuntimes.set(spec.id, { spec, axial })
    }
  }

  private buildRing(spec: PlanetSpec, loader: THREE.TextureLoader) {
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
      loader.load(SATURN_RING_TEXTURE_URL, (texture) => {
        if (this.disposed) {
          texture.dispose()
          return
        }
        texture.colorSpace = THREE.SRGBColorSpace
        texture.anisotropy = 4
        this.textures.push(texture)
        material.map = texture
        material.needsUpdate = true
      })
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

    // 标签锚点：位于行星连线下方的带边缘
    const lineAngle = PLANET_LINE_ANGLE_DEG * DEG
    const asteroidAnchor = new THREE.Object3D()
    asteroidAnchor.name = 'asteroid-belt'
    const asteroidAngle = lineAngle + 12 * DEG
    asteroidAnchor.position.set(Math.cos(asteroidAngle) * (ASTEROID_BELT.outer + 0.4), 2.6, Math.sin(asteroidAngle) * (ASTEROID_BELT.outer + 0.4))
    const kuiperAnchor = new THREE.Object3D()
    kuiperAnchor.name = 'kuiper-belt'
    const kuiperAngle = lineAngle + 8 * DEG
    kuiperAnchor.position.set(Math.cos(kuiperAngle) * (KUIPER_BELT.outer + 2), 3.4, Math.sin(kuiperAngle) * (KUIPER_BELT.outer + 2))
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
    const delta = Math.min(this.clock.getDelta(), 0.05) * this.timeScale
    this.elapsed += delta

    this.sunMesh.rotation.y += (Math.PI * 2 / SUN_ROTATION_SECONDS) * delta
    // 相机侧补光跟随相机
    this.cameraLight.position.copy(this.camera.position)
    for (const runtime of this.planetRuntimes.values()) {
      runtime.axial.rotation.y += (Math.PI * 2 / rotationPeriodSeconds(runtime.spec.rotationHours)) * delta
    }
    const periods = [ASTEROID_BELT.periodSeconds, KUIPER_BELT.periodSeconds]
    this.beltGroups.forEach((group, index) => {
      group.rotation.y += (Math.PI * 2 / periods[index]) * delta
    })
    this.controls.update()
    // 同步注视点：平移会移动 controls.target，标签与 resize 逻辑依赖 lookAt
    this.lookAt.copy(this.controls.target)
    this.updateLabels()
    this.renderer.render(this.scene, this.camera)
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
    }
  }

  private updateLabels() {
    const width = this.host.clientWidth
    const height = this.host.clientHeight
    if (width === 0 || height === 0) return
    const halfFovTan = Math.tan((VIEW.fov / 2) * DEG)
    const labels: SolarLabel[] = []

    labels.push(this.projectLabel('sun', 'sun', this.tempWorld.set(0, 0, 0), SUN_RADIUS, width, height, halfFovTan))

    for (const runtime of this.planetRuntimes.values()) {
      const world = runtime.axial.getWorldPosition(this.tempWorld)
      labels.push(this.projectLabel('planet', runtime.spec.id, world, runtime.spec.radius, width, height, halfFovTan))
    }

    for (const anchor of this.beltAnchors) {
      const world = anchor.getWorldPosition(this.tempWorldB)
      labels.push(this.projectLabel('belt', anchor.name, world, 1.6, width, height, halfFovTan))
    }

    this.onLabels(labels)
  }

  // ---- 相机与构图 --------------------------------------------------------

  /** 数值构图：主视角中心对准小行星带（锚定在画面目标位置），距离取构图下限 */
  private refit() {
    const host = this.host
    const width = host.clientWidth
    const height = host.clientHeight
    if (width === 0 || height === 0) return
    const aspect = width / height

    this.fitDistance = VIEW.composeMinDistance
    this.placeCameraForBelt(this.fitDistance, aspect)
    this.controls.maxDistance = Math.max(600, this.fitDistance * 2.2)
  }

  /** 解析反推注视点：让小行星带（主视角中心）精确落在目标屏幕位置（针孔相机模型） */
  private placeCameraForBelt(distance: number, aspect: number) {
    const tanHalf = Math.tan((VIEW.fov / 2) * DEG)
    const ndcX = 2 * VIEW.anchorScreenX - 1
    const ndcY = -(2 * VIEW.anchorScreenY - 1)
    const alpha = -ndcX * tanHalf * aspect
    const beta = -ndcY * tanHalf
    this.lookAt.copy(this.beltAnchor).addScaledVector(this.right, distance * alpha).addScaledVector(this.upv, distance * beta)
    this.camera.position.copy(this.lookAt).addScaledVector(this.dir, distance)
    this.camera.lookAt(this.lookAt)
    // lookAt 只刷新 matrixWorld；project() 依赖 matrixWorldInverse，必须显式更新
    this.camera.updateMatrixWorld(true)
    // OrbitControls 每帧强制 camera.lookAt(target)，必须同步，否则构图只存活一帧
    this.controls.target.copy(this.lookAt)
  }

  private onResize = () => {
    const host = this.host
    if (host.clientWidth === 0 || host.clientHeight === 0) return
    this.camera.aspect = host.clientWidth / host.clientHeight
    this.camera.updateProjectionMatrix()
    this.renderer.setSize(host.clientWidth, host.clientHeight)
    if (!this.userInteracted) {
      // 用户尚未操作：resize 后沿用默认构图
      this.refit()
      return
    }
    // 用户已操作：保留当前取景方向与距离，仅限制在合法范围
    const offset = this.camera.position.clone().sub(this.lookAt)
    const length = offset.length()
    if (length < 0.01) return
    const clamped = Math.min(Math.max(length, VIEW.minDistance), Math.max(600, this.fitDistance * 2.2))
    this.camera.position.copy(this.lookAt).addScaledVector(offset.normalize(), clamped)
    this.camera.lookAt(this.lookAt)
    this.camera.updateMatrixWorld(true)
    this.controls.target.copy(this.lookAt)
  }

  /** 用户首次拖拽/缩放时标记，之后不再重置其视角 */
  private onUserInteract = () => {
    this.userInteracted = true
  }

  /** 双击复位：回到默认的斜俯视构图 */
  private onDoubleClick = () => {
    this.refit()
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

  private onMotionChange = (event: MediaQueryListEvent) => {
    this.timeScale = event.matches ? 0 : 1
  }
}
