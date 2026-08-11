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

// 目标知识库：每颗天体一句可解释的观测提示（供今夜建议/展开详情引用）。
export const observeTips: Record<BodyId, string> = {
  sun: '太阳观测必须使用合格的全口径太阳滤镜，绝不可直视。',
  moon: '月面适合在明暗交界附近观察；满月虽明亮，地形阴影反而较少。',
  mercury: '水星离太阳很近，最佳观测窗口在日落后/日出前的低空，且常受大气扰动。',
  venus: '金星是除日月外最亮的星，黄昏或黎明低空可见；大气浓密，望远镜中呈均匀乳白色。',
  mars: '火星的极冠与暗区只有在冲日前后大而亮时才有辨识度。',
  jupiter: '木星的四颗伽利略卫星用双筒即可看见排成一条直线。',
  saturn: '土星环用小型望远镜就能看出轮廓，是入门最值得一看的目标。',
  uranus: '天王星亮度约 5.7 等，需要双筒或望远镜，且要在极暗的天空下才有机会。',
  neptune: '海王星亮度约 7.9 等，必须用望远镜，并最好对照星图确认位置。',
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
  samples: Array<{ at: Date; altitude: number; azimuth: number }>
  best: Date | null
}

export interface TwilightTimes {
  sunrise: Date | null
  sunset: Date | null
  astronomicalDawn: Date | null
  astronomicalDusk: Date | null
  // 民用晨昏（太阳 −6°）：视觉上"天基本黑了/亮了"的感知边界，
  // 用于白天系数渐变带（天文 −18° 只适合深空观测判断，不适合当视觉天色）。
  civilDawn: Date | null
  civilDusk: Date | null
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
    const horizontal = Horizon(sampleAt, place, equator.ra, equator.dec, 'normal')
    return { at: sampleAt, altitude: horizontal.altitude, azimuth: horizontal.azimuth }
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
    civilDawn: SearchAltitude(Body.Sun, place, 1, start, 1.1, -6)?.date ?? null,
    civilDusk: SearchAltitude(Body.Sun, place, -1, start, 1.1, -6)?.date ?? null,
  }
}

function clamp01(value: number) {
  return Math.max(0, Math.min(1, value))
}

/**
 * 白昼系数 0（深夜）→ 1（正午）。以当天当地日出/日落为硬边界，
 * 民用晨光/昏影（太阳 −6°）作为 0↔1 的线性渐变过渡带——这是人眼
 * 感知"天黑了/亮了"的边界，日落约 30–40 分钟后视觉上即全黑；
 * 天文 −18° 只用于深空观测判断，不作为视觉天色。
 * 极昼极夜或定位缺失导致 sunrise/sunset 为空时，保守地按黑夜（0）处理。
 */
export function daylightFactor(times: TwilightTimes, at: Date): number {
  if (!times.sunrise || !times.sunset || !times.civilDawn || !times.civilDusk) return 0
  const moment = at.getTime()
  const dawn = times.civilDawn.getTime()
  const sunrise = times.sunrise.getTime()
  const sunset = times.sunset.getTime()
  const dusk = times.civilDusk.getTime()
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

/* ---------- 今夜夜空分析：天文夜、无月黑夜与银河核心窗口 ---------- */

// 银河核心（人马座 A* 方向）固定赤道坐标：RA 17h45m40s / Dec −29°00′28″。
// 用固定坐标可直接经 Horizon 求地平高度，不依赖具体天体。
export const GALACTIC_CORE = { ra: 266.4167, dec: -29.0078 }

export interface NightWindow {
  start: Date
  end: Date
}

export interface NightAnalysis {
  // 天文夜（太阳低于地平线 −18°）：当晚昏影 → 次日晨光（可能跨午夜）。
  astronomicalNight: NightWindow | null
  // 天文夜内月亮在地平线以下的连续时段（无月黑夜）。
  moonlessWindows: NightWindow[]
  // 银河核心：天文夜内高度 ≥ 20° 的可见窗口，及窗口内最高高度与时刻。
  galacticCore: {
    maxAltitude: number
    maxTime: Date | null
    windows: NightWindow[]
  }
}

const NIGHT_SAMPLE_MINUTES = 10

function collectWindows(points: Array<{ at: Date; active: boolean }>): NightWindow[] {
  const windows: NightWindow[] = []
  let runStart: Date | null = null
  for (const point of points) {
    if (point.active) {
      if (runStart == null) runStart = point.at
    } else if (runStart != null) {
      windows.push({ start: runStart, end: point.at })
      runStart = null
    }
  }
  if (runStart != null && points.length) windows.push({ start: runStart, end: points[points.length - 1].at })
  return windows
}

/**
 * 基于本地星历计算今夜的分析结果：天文夜窗口、无月黑夜与银河核心可见时段。
 * 全部为几何计算（太阳/月亮/固定赤道坐标 → 地平高度），不依赖天气或外部 Key。
 *
 * "今夜"窗口 = 自当前时刻起下一个天文昏影（太阳降至 −18°）→ 其后的天文晨光（次日）。
 */
export function analyzeNight(at: Date, latitude: number, longitude: number, elevation = 0): NightAnalysis {
  const empty: NightAnalysis = { astronomicalNight: null, moonlessWindows: [], galacticCore: { maxAltitude: 0, maxTime: null, windows: [] } }
  const place = observer(latitude, longitude, elevation)
  // 太阳低于 −18° 的下降交点 = 今晚天文昏影；从该时刻起的下一个上升交点 = 次日天文晨光。
  const dusk = SearchAltitude(Body.Sun, place, -1, at, 1.1, -18)?.date ?? null
  if (!dusk) return empty
  const dawn = SearchAltitude(Body.Sun, place, 1, new Date(dusk.getTime() + 60_000), 1.1, -18)?.date ?? null
  if (!dawn) return empty
  const nightStart = dusk.getTime()
  const nightEnd = dawn.getTime()
  const dayStart = localStartOfDay(at).getTime()

  const points: Array<{ at: Date; moonless: boolean; coreVisible: boolean; coreAltitude: number }> = []
  let maxCore = { altitude: -Infinity, time: null as Date | null }
  // 天文夜可能跨午夜（如 20:10 → 次日 03:46），采样必须覆盖到 nightEnd，
  // 不能只到 dayStart+24h（那会漏掉午夜后到晨光之间的时段）。
  const lastSampleMinute = Math.ceil((nightEnd - dayStart) / 60_000)
  for (let minute = 0; minute <= lastSampleMinute; minute += NIGHT_SAMPLE_MINUTES) {
    const sampleAt = new Date(dayStart + minute * 60_000)
    const moment = sampleAt.getTime()
    const dark = moment >= nightStart && moment <= nightEnd
    const moonEquator = Equator(Body.Moon, sampleAt, place, true, true)
    const moonAltitude = Horizon(sampleAt, place, moonEquator.ra, moonEquator.dec, 'normal').altitude
    const coreAltitude = Horizon(sampleAt, place, GALACTIC_CORE.ra, GALACTIC_CORE.dec, 'normal').altitude
    if (dark && coreAltitude > maxCore.altitude) maxCore = { altitude: coreAltitude, time: sampleAt }
    points.push({
      at: sampleAt,
      moonless: dark && moonAltitude <= 0,
      coreVisible: dark && coreAltitude >= 20,
      coreAltitude,
    })
  }

  return {
    astronomicalNight: { start: dusk, end: dawn },
    moonlessWindows: collectWindows(points.map((point) => ({ at: point.at, active: point.moonless }))),
    galacticCore: {
      maxAltitude: maxCore.altitude === -Infinity ? 0 : Math.round(maxCore.altitude),
      maxTime: maxCore.time,
      windows: collectWindows(points.map((point) => ({ at: point.at, active: point.coreVisible }))),
    },
  }
}
