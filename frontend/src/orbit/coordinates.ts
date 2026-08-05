import * as THREE from 'three'
import { eciToGeodetic, gstime, json2satrec, propagate } from 'satellite.js'
import type { DeepSpaceProbe, Spacecraft } from '../types'
import earthDay8kUrl from '../assets/earth/blue-marble-8k.jpg'

export const EARTH_RADIUS = 2.15
const EARTH_RADIUS_KM = 6371
const ALTITUDE_EXAGGERATION = 3.2

/** 地球日间/夜间纹理（远端 CDN，切换页面时提前预热避免卡顿） */
export const EARTH_DAY_TEXTURE_URL = earthDay8kUrl // NASA Blue Marble NG 8k（本地；4k 屏下 8k 已是 4 倍余量，位图 131MB 上传无卡顿）
export const EARTH_NIGHT_TEXTURE_URL = 'https://unpkg.com/three-globe@2.45.2/example/img/earth-night.jpg'

export interface OrbitalPoint {
  position: THREE.Vector3
  latitude: number
  longitude: number
  altitudeKm: number
}

export function latLonToVector(latitude: number, longitude: number, radius = EARTH_RADIUS) {
  const lat = THREE.MathUtils.degToRad(latitude)
  const lon = THREE.MathUtils.degToRad(longitude)
  return new THREE.Vector3(
    radius * Math.cos(lat) * Math.cos(lon),
    radius * Math.sin(lat),
    -radius * Math.cos(lat) * Math.sin(lon),
  )
}

/** 展示高度（km，含夸张）：LEO ≤2000km 沿用 ×3.2 夸张（ISS/天宫/哈勃不变）；
 *  MEO/GEO 若继续 ×3.2 会飞出默认视锥（GPS 20180km → 半径 ~24 单位，完全不可见），
 *  故对 >2000km 的高度做平方根压缩，落到 LEO 之上、默认视角内的可见带。
 *  仍为真实 TLE 传播：轨道面/相位/相对运动真实，仅显示高度带压缩（与月球 ×3 同思路）。 */
function exaggeratedAltitude(altitudeKm: number): number {
  if (altitudeKm <= 2000) return altitudeKm * ALTITUDE_EXAGGERATION
  // 系数 5.5：GPS(20180km)→~5.1、GEO(35786km)→~5.4 单位（比早期 8.3 方案更靠近地球 ~9%，用户要求同比例缩小一点）
  const compressed = 2000 + 5.5 * Math.sqrt(altitudeKm - 2000)
  return compressed * ALTITUDE_EXAGGERATION
}

export function spacecraftPoint(spacecraft: Spacecraft, date: Date, referenceGmst?: number): OrbitalPoint | null {
  try {
    const satrec = json2satrec(spacecraft.omm as never)
    const state = propagate(satrec, date)
    if (!state || !state.position || typeof state.position === 'boolean') return null

    // referenceGmst 提供时用固定参考帧（画轨道环）：逐样本用自身 gmst 会把地球自转混进
    // 采样，一个周期后端点错开 ~ω·P（ISS 约 23°）环不闭合；固定帧下环=真实惯性轨道（闭合）
    const geo = eciToGeodetic(state.position, referenceGmst ?? gstime(date))
    const latitude = THREE.MathUtils.radToDeg(geo.latitude)
    const longitude = THREE.MathUtils.radToDeg(geo.longitude)
    const altitudeKm = Math.max(0, geo.height)
    const radius = EARTH_RADIUS * (1 + exaggeratedAltitude(altitudeKm) / EARTH_RADIUS_KM)
    return { position: latLonToVector(latitude, longitude, radius), latitude, longitude, altitudeKm }
  } catch {
    return null
  }
}

export function sampleOrbit(spacecraft: Spacecraft, center: Date) {
  const points: THREE.Vector3[] = []
  const meanMotion = Number(spacecraft.omm.MEAN_MOTION) || 15
  const periodMinutes = 1440 / meanMotion
  const samples = 120
  // 固定参考 GMST（中心时刻）：整个轨道环画在场景固定坐标系（≈中心时刻地球固连系），
  // 环 = 真实惯性轨道，端点仅差 J2 进动（~0.3°）；补首点闭合
  const referenceGmst = gstime(center)

  for (let index = 0; index <= samples; index += 1) {
    const offsetMinutes = (index / samples - 0.5) * periodMinutes
    const point = spacecraftPoint(spacecraft, new Date(center.getTime() + offsetMinutes * 60_000), referenceGmst)
    if (point) points.push(point.position)
  }
  // 严格圆形化：真实轨道含偏心率（如先锋1号 e≈0.18 是椭圆）与进动接缝，
  // 按用户要求统一成"严格圆 + 平滑 + 首尾精确闭合"：拟合圆心/轨道面法线/平均半径，
  // 再均匀生成圆周点（参数化 2πk/N，k=0 与 k=N 重合 → 无接缝）
  return fitCirclePoints(points, samples)
}

/** 对采样点做圆拟合，输出严格圆形、均匀、精确闭合的圆周点（保持真实圆心/轨道面/平均半径） */
export function fitCirclePoints(points: THREE.Vector3[], segments: number): THREE.Vector3[] {
  if (points.length < 3) return closeRing(points)
  // 圆心 = 采样点质心
  const centerPoint = new THREE.Vector3()
  for (const p of points) centerPoint.add(p)
  centerPoint.divideScalar(points.length)
  // 轨道面法线 = 相邻差向量叉积之和（必须用独立临时量：cross 就地写 this，别名会叉出零向量）
  const normal = new THREE.Vector3()
  const a = new THREE.Vector3()
  const b = new THREE.Vector3()
  const cross = new THREE.Vector3()
  for (let i = 0; i < points.length - 1; i += 1) {
    a.copy(points[i]).sub(centerPoint)
    b.copy(points[i + 1]).sub(centerPoint)
    cross.crossVectors(a, b)
    normal.add(cross)
  }
  if (normal.lengthSq() < 1e-12) return closeRing(points)
  normal.normalize()
  // 平面内基：u 沿第一个采样点的面内方向，v = n×u
  const toFirst = new THREE.Vector3().copy(points[0]).sub(centerPoint)
  const alongNormal = normal.clone().multiplyScalar(toFirst.dot(normal))
  const u = toFirst.sub(alongNormal)
  if (u.lengthSq() < 1e-12) return closeRing(points)
  u.normalize()
  const v = new THREE.Vector3().crossVectors(normal, u)
  // 平均面内半径
  const radial = new THREE.Vector3()
  let radiusSum = 0
  for (const p of points) {
    radial.copy(p).sub(centerPoint)
    radial.sub(normal.clone().multiplyScalar(radial.dot(normal)))
    radiusSum += radial.length()
  }
  const radius = radiusSum / points.length
  // 均匀圆周点：2πk/N，k=N 与 k=0 重合 → 精确闭合、平滑
  const ring: THREE.Vector3[] = []
  for (let k = 0; k <= segments; k += 1) {
    const theta = (Math.PI * 2 * k) / segments
    ring.push(
      new THREE.Vector3()
        .copy(centerPoint)
        .addScaledVector(u, Math.cos(theta) * radius)
        .addScaledVector(v, Math.sin(theta) * radius),
    )
  }
  return ring
}

/** 回退：不满足拟合条件时原样返回并补首点闭合（避免留缝） */
function closeRing(points: THREE.Vector3[]): THREE.Vector3[] {
  const out = points.map((p) => p.clone())
  if (out.length > 2) out.push(out[0].clone())
  return out
}

// ---------------------------------------------------------------------------
// 空间天文台（非地球轨道）支持：太阳方向 / 地球日心位置 / 探测采样插值
// ---------------------------------------------------------------------------

export const AU_KM = 149_597_870.7

/** 场景惯性系中的地球轴倾（与 OrbitScene EARTH_TILT 同轴角，保证太阳方向一致） */
export const EARTH_TILT_QUATERNION = new THREE.Quaternion().setFromAxisAngle(
  new THREE.Vector3(0, 0, 1),
  THREE.MathUtils.degToRad(23.44),
)

/** 太阳在场景惯性系中的单位方向（与 OrbitScene.updateSun 同公式：太阳赤纬 + 子午线 + 轴倾） */
export function sunSceneDirection(date = new Date()): THREE.Vector3 {
  const yearStart = Date.UTC(date.getUTCFullYear(), 0, 0)
  const dayOfYear = Math.floor((date.getTime() - yearStart) / 86_400_000)
  const declination = 23.44 * Math.sin(THREE.MathUtils.degToRad((360 / 365) * (dayOfYear - 81)))
  const utcHours = date.getUTCHours() + date.getUTCMinutes() / 60 + date.getUTCSeconds() / 3600
  const subsolarLongitude = 180 - utcHours * 15
  return latLonToVector(declination, subsolarLongitude, 1).applyQuaternion(EARTH_TILT_QUATERNION).normalize()
}

/** 地球当前日心黄道位置（km）：J2000 轨道根数 + 迭代开普勒方程（与太阳系页同款公式） */
export function earthHeliocentricEclipticKm(date = new Date()): THREE.Vector3 {
  const J2000_MS = Date.UTC(2000, 0, 1, 12)
  const days = (date.getTime() - J2000_MS) / 86_400_000
  const meanLongitude = (100.46435 + (360 / 365.256) * days) % 360
  const M = THREE.MathUtils.degToRad((((meanLongitude - 102.93735) % 360) + 360) % 360)
  let E = M
  for (let i = 0; i < 8; i += 1) E = E - (E - 0.016708 * Math.sin(E) - M) / (1 - 0.016708 * Math.cos(E))
  const nu = 2 * Math.atan2(Math.sqrt(1 + 0.016708) * Math.sin(E / 2), Math.sqrt(1 - 0.016708) * Math.cos(E / 2))
  const rAU = 1.000001018 * (1 - 0.016708 * Math.cos(E))
  const trueLongitude = nu + THREE.MathUtils.degToRad(102.93735)
  return new THREE.Vector3(Math.cos(trueLongitude), Math.sin(trueLongitude), 0).multiplyScalar(rAU * AU_KM)
}

/** 探测采样线性插值到指定时刻（km 日心黄道）；越界返回最近端点；无采样返回 null */
export function interpolateProbeKm(probe: DeepSpaceProbe, date = new Date()): THREE.Vector3 | null {
  const pos = probe.positions
  if (!pos.length) return null
  const t = date.getTime()
  const first = pos[0]
  const last = pos[pos.length - 1]
  const finite = (v: { x: number; y: number; z: number }) => Number.isFinite(v.x) && Number.isFinite(v.y) && Number.isFinite(v.z)
  if (t <= Date.parse(first.epoch) && finite(first)) return new THREE.Vector3(first.x, first.y, first.z)
  if (t >= Date.parse(last.epoch) && finite(last)) return new THREE.Vector3(last.x, last.y, last.z)
  for (let i = 0; i < pos.length - 1; i += 1) {
    const a = Date.parse(pos[i].epoch)
    const b = Date.parse(pos[i + 1].epoch)
    if (a <= t && t <= b && b - a > 0) {
      const f = (t - a) / (b - a)
      const p = new THREE.Vector3(
        pos[i].x + (pos[i + 1].x - pos[i].x) * f,
        pos[i].y + (pos[i + 1].y - pos[i].y) * f,
        pos[i].z + (pos[i + 1].z - pos[i].z) * f,
      )
      // 防御：服务端异常采样（非有限值）不流入场景
      if (Number.isFinite(p.x) && Number.isFinite(p.y) && Number.isFinite(p.z)) return p
    }
  }
  return null
}
