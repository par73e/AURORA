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

export function spacecraftPoint(spacecraft: Spacecraft, date: Date): OrbitalPoint | null {
  try {
    const satrec = json2satrec(spacecraft.omm as never)
    const state = propagate(satrec, date)
    if (!state || !state.position || typeof state.position === 'boolean') return null

    const geo = eciToGeodetic(state.position, gstime(date))
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

  for (let index = 0; index <= samples; index += 1) {
    const offsetMinutes = (index / samples - 0.5) * periodMinutes
    const point = spacecraftPoint(spacecraft, new Date(center.getTime() + offsetMinutes * 60_000))
    if (point) points.push(point.position)
  }
  // 强制闭合：真实 SGP4 含 J2 进动（一个周期内轨道面/拱线漂移），±半周期两个端点
  // 不在同一位置，直接连线会留下开口；补首点使轨道环首尾相连（视觉闭合，开口极小）
  if (points.length > 2) points.push(points[0].clone())
  return points
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
