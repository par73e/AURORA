import {
  Body,
  Equator,
  Horizon,
  Illumination,
  MoonPhase,
  Observer,
  SearchAltitude,
  SearchMoonPhase,
  SearchRiseSet,
} from 'astronomy-engine'

export type BodyId = 'sun' | 'moon' | 'mercury' | 'venus' | 'mars' | 'jupiter' | 'saturn' | 'uranus' | 'neptune'

export interface CelestialBody {
  id: BodyId
  body: Body
  name: string
  nameEn: string
  glyph: string
  tint: string
  archiveHash?: string
}

export interface BodyPosition {
  altitude: number
  azimuth: number
  visible: boolean
  magnitude: number | null
  illumination: number | null
  distanceAu: number | null
}

export interface BodyTrack extends CelestialBody, BodyPosition {
  rise: Date | null
  set: Date | null
  transit: Date | null
  samples: Array<{ at: Date; altitude: number }>
  best: Date | null
}

export interface TwilightTimes {
  sunrise: Date | null
  sunset: Date | null
  astronomicalDawn: Date | null
  astronomicalDusk: Date | null
}

export const bodies: CelestialBody[] = [
  { id: 'sun', body: Body.Sun, name: '太阳', nameEn: 'SUN', glyph: '☉', tint: '#f0bc63', archiveHash: '#sun' },
  { id: 'moon', body: Body.Moon, name: '月球', nameEn: 'MOON', glyph: '◐', tint: '#dce7ed', archiveHash: '#moon' },
  { id: 'mercury', body: Body.Mercury, name: '水星', nameEn: 'MERCURY', glyph: '☿', tint: '#b7a38d', archiveHash: '#mercury' },
  { id: 'venus', body: Body.Venus, name: '金星', nameEn: 'VENUS', glyph: '♀', tint: '#ead5a0', archiveHash: '#venus' },
  { id: 'mars', body: Body.Mars, name: '火星', nameEn: 'MARS', glyph: '♂', tint: '#d88970', archiveHash: '#mars' },
  { id: 'jupiter', body: Body.Jupiter, name: '木星', nameEn: 'JUPITER', glyph: '♃', tint: '#d9bf93', archiveHash: '#jupiter' },
  { id: 'saturn', body: Body.Saturn, name: '土星', nameEn: 'SATURN', glyph: '♄', tint: '#d9cc95', archiveHash: '#saturn' },
  { id: 'uranus', body: Body.Uranus, name: '天王星', nameEn: 'URANUS', glyph: '♅', tint: '#9fced6', archiveHash: '#uranus' },
  { id: 'neptune', body: Body.Neptune, name: '海王星', nameEn: 'NEPTUNE', glyph: '♆', tint: '#7798d6', archiveHash: '#neptune' },
]

function observer(latitude: number, longitude: number, elevation = 0) {
  return new Observer(latitude, longitude, elevation)
}

export function calculatePosition(config: CelestialBody, at: Date, latitude: number, longitude: number, elevation = 0): BodyPosition {
  const place = observer(latitude, longitude, elevation)
  const equator = Equator(config.body, at, place, true, true)
  const horizontal = Horizon(at, place, equator.ra, equator.dec, 'normal')
  const light = Illumination(config.body, at)
  const moonIllumination = config.id === 'moon' ? (1 - Math.cos(MoonPhase(at) * Math.PI / 180)) / 2 : null
  return {
    altitude: horizontal.altitude,
    azimuth: horizontal.azimuth,
    visible: horizontal.altitude > 0,
    magnitude: config.id === 'sun' || config.id === 'moon' ? null : light.mag,
    illumination: config.id === 'moon' ? moonIllumination : config.id === 'sun' ? null : light.phase_fraction,
    distanceAu: light.geo_dist,
  }
}

function localStartOfDay(at: Date) {
  const day = new Date(at)
  day.setHours(0, 0, 0, 0)
  return day
}

function nextRiseOrSet(body: Body, place: Observer, direction: 1 | -1, dayStart: Date) {
  return SearchRiseSet(body, place, direction, dayStart, 1.1)?.date ?? null
}

function transitFor(config: CelestialBody, place: Observer, start: Date): Date | null {
  let best: Date | null = null
  let highest = -Infinity
  for (let minute = 0; minute <= 24 * 60; minute += 5) {
    const at = new Date(start.getTime() + minute * 60_000)
    const equator = Equator(config.body, at, place, true, true)
    const altitude = Horizon(at, place, equator.ra, equator.dec, 'normal').altitude
    if (altitude > highest) {
      highest = altitude
      best = at
    }
  }
  return best
}

export function calculateTrack(config: CelestialBody, at: Date, latitude: number, longitude: number, elevation = 0): BodyTrack {
  const place = observer(latitude, longitude, elevation)
  const dayStart = localStartOfDay(at)
  const samples = Array.from({ length: 97 }, (_, index) => {
    const sampleAt = new Date(dayStart.getTime() + index * 15 * 60_000)
    const equator = Equator(config.body, sampleAt, place, true, true)
    return { at: sampleAt, altitude: Horizon(sampleAt, place, equator.ra, equator.dec, 'normal').altitude }
  })
  const current = calculatePosition(config, at, latitude, longitude, elevation)
  const transit = transitFor(config, place, dayStart)
  const best = samples.reduce<{ at: Date; altitude: number } | null>((bestSample, sample) => !bestSample || sample.altitude > bestSample.altitude ? sample : bestSample, null)
  return {
    ...config,
    ...current,
    rise: nextRiseOrSet(config.body, place, 1, dayStart),
    set: nextRiseOrSet(config.body, place, -1, dayStart),
    transit,
    samples,
    best: best?.altitude && best.altitude > 0 ? best.at : null,
  }
}

export function calculateTwilight(at: Date, latitude: number, longitude: number, elevation = 0): TwilightTimes {
  const place = observer(latitude, longitude, elevation)
  const start = localStartOfDay(at)
  return {
    sunrise: SearchRiseSet(Body.Sun, place, 1, start, 1.1)?.date ?? null,
    sunset: SearchRiseSet(Body.Sun, place, -1, start, 1.1)?.date ?? null,
    astronomicalDawn: SearchAltitude(Body.Sun, place, 1, start, 1.1, -18)?.date ?? null,
    astronomicalDusk: SearchAltitude(Body.Sun, place, -1, start, 1.1, -18)?.date ?? null,
  }
}

function clamp01(value: number) {
  return Math.max(0, Math.min(1, value))
}

/**
 * 白昼系数 0（深夜）→ 1（正午）。以当天当地日出/日落为硬边界，
 * 天文晨光/昏影作为 0↔1 的线性渐变过渡带。
 * 极昼极夜或定位缺失导致 sunrise/sunset 为空时，保守地按黑夜（0）处理。
 */
export function daylightFactor(times: TwilightTimes, at: Date): number {
  if (!times.sunrise || !times.sunset || !times.astronomicalDawn || !times.astronomicalDusk) return 0
  const moment = at.getTime()
  const dawn = times.astronomicalDawn.getTime()
  const sunrise = times.sunrise.getTime()
  const sunset = times.sunset.getTime()
  const dusk = times.astronomicalDusk.getTime()
  if (moment <= dawn || moment >= dusk) return 0
  if (moment >= sunrise && moment <= sunset) return 1
  if (moment < sunrise) return clamp01((moment - dawn) / (sunrise - dawn))
  return 1 - clamp01((moment - sunset) / (dusk - sunset))
}

export function moonPhase(at: Date) {
  const phase = MoonPhase(at)
  const illumination = (1 - Math.cos(phase * Math.PI / 180)) / 2
  const label = phase < 18 || phase >= 342 ? '朔月' : phase < 72 ? '娥眉月' : phase < 108 ? '上弦月' : phase < 162 ? '盈凸月' : phase < 198 ? '满月' : phase < 252 ? '亏凸月' : phase < 288 ? '下弦月' : '残月'
  return { phase, illumination, label, age: phase / 360 * 29.53059 }
}

export function upcomingMoonPhases(at: Date) {
  return [
    { target: 0, label: '朔月', description: '月光最少的深空观测时段即将开始。' },
    { target: 90, label: '上弦月', description: '适合沿月面明暗交界观察地形。' },
    { target: 180, label: '满月', description: '月面与夜景题材更适合优先安排。' },
    { target: 270, label: '下弦月', description: '清晨可见，月面阴影层次较强。' },
  ].map((phase) => ({ ...phase, at: SearchMoonPhase(phase.target, at, 32)?.date ?? null }))
    .filter((phase): phase is { target: number; label: string; description: string; at: Date } => phase.at !== null)
    .sort((left, right) => left.at.getTime() - right.at.getTime())
}

export function observingStatus(track: BodyTrack) {
  if (track.altitude <= 0) return '地平线下'
  if (track.altitude < 8) return track.azimuth < 180 ? '正在升起' : '正在下落'
  return '可观测'
}

export function bearing(azimuth: number) {
  const points = ['北', '东北', '东', '东南', '南', '西南', '西', '西北']
  return points[Math.round(azimuth / 45) % 8]
}
