import * as THREE from 'three'
import { eciToGeodetic, gstime, json2satrec, propagate } from 'satellite.js'
import type { Spacecraft } from '../types'
import earthDay21kUrl from '../assets/earth/blue-marble-21k.jpg'

export const EARTH_RADIUS = 2.15
const EARTH_RADIUS_KM = 6371
const ALTITUDE_EXAGGERATION = 3.2

/** 地球日间/夜间纹理（远端 CDN，切换页面时提前预热避免卡顿） */
export const EARTH_DAY_TEXTURE_URL = earthDay21kUrl // NASA Blue Marble Next Generation 21k（本地化，替代 unpkg 4k 远程）
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

export function spacecraftPoint(spacecraft: Spacecraft, date: Date): OrbitalPoint | null {
  try {
    const satrec = json2satrec(spacecraft.omm as never)
    const state = propagate(satrec, date)
    if (!state || !state.position || typeof state.position === 'boolean') return null

    const geo = eciToGeodetic(state.position, gstime(date))
    const latitude = THREE.MathUtils.radToDeg(geo.latitude)
    const longitude = THREE.MathUtils.radToDeg(geo.longitude)
    const altitudeKm = Math.max(0, geo.height)
    const radius = EARTH_RADIUS * (1 + (altitudeKm / EARTH_RADIUS_KM) * ALTITUDE_EXAGGERATION)
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
