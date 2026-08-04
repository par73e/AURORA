import * as THREE from 'three'
import { eciToGeodetic, gstime, json2satrec, propagate } from 'satellite.js'
import type { Spacecraft } from '../types'
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
  return points
}
