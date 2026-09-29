import { AngleBetween, Body, Equator, Horizon, KM_PER_AU, Observer, SearchLocalSolarEclipse } from 'astronomy-engine'
import type { AstronomyEvent, EventLocalVisibility } from './api'

const minute = 60_000
const eventTolerance = 36 * 60 * minute

/** 只对匹配的 NASA 日食日期给出本地结论，避免把下一次日食误配给当前事件。 */
export function localSolarEclipseVisibility(event: AstronomyEvent, latitude: number, longitude: number): EventLocalVisibility | null {
  if (event.kind !== 'solar_eclipse') return null
  const eventAt = new Date(event.startsAt).getTime()
  if (!Number.isFinite(eventAt) || !Number.isFinite(latitude) || !Number.isFinite(longitude)) return null

  try {
    const observer = new Observer(latitude, longitude, 0)
    const eclipse = SearchLocalSolarEclipse(new Date(eventAt - eventTolerance), observer)
    const peakAt = eclipse.peak.time.date.getTime()
    if (Math.abs(peakAt - eventAt) > eventTolerance) {
      return { status: 'not_visible', reason: '此次日食的月球半影未经过当前地点。' }
    }

    const first = eclipse.partial_begin.time.date.getTime()
    const last = eclipse.partial_end.time.date.getTime()
    const peakTime = eclipse.peak.time.date
    const sun = Equator(Body.Sun, peakTime, observer, true, true)
    const moon = Equator(Body.Moon, peakTime, observer, true, true)
    const sunRadius = Math.asin(695700 / (sun.dist * KM_PER_AU))
    const moonRadius = Math.asin(1738.1 / (moon.dist * KM_PER_AU))
    const separation = AngleBetween(sun.vec, moon.vec) * Math.PI / 180
    // NASA 的食分是被遮住的太阳视直径比例，与面积遮挡率不同。
    const magnitude = eclipse.total_begin
      ? moonRadius / sunRadius
      : Math.max(0, Math.min(1, (sunRadius + moonRadius - separation) / (2 * sunRadius)))
    const contacts = {
      partialBegin: eclipse.partial_begin.time.date.toISOString(),
      peak: eclipse.peak.time.date.toISOString(),
      partialEnd: eclipse.partial_end.time.date.toISOString(),
      partialBeginVisible: eclipse.partial_begin.altitude > 0,
      peakVisible: eclipse.peak.altitude > 0,
      partialEndVisible: eclipse.partial_end.altitude > 0,
      ...(eclipse.total_begin && eclipse.total_end ? {
        centralBegin: eclipse.total_begin.time.date.toISOString(),
        centralEnd: eclipse.total_end.time.date.toISOString(),
        centralBeginVisible: eclipse.total_begin.altitude > 0,
        centralEndVisible: eclipse.total_end.altitude > 0,
      } : {}),
      obscurationPercent: eclipse.kind === 'total' ? 100 : Math.min(99.9, Math.round(eclipse.obscuration * 1000) / 10),
      magnitude: Math.round(magnitude * 10000) / 10000,
      kind: eclipse.kind === 'total' ? 'total' as const : eclipse.kind === 'annular' ? 'annular' as const : 'partial' as const,
    }
    const visible: Array<{ at: number; altitude: number; azimuth: number }> = []
    const sample = (at: number) => {
      const date = new Date(at)
      const equator = Equator(Body.Sun, date, observer, true, true)
      const horizon = Horizon(date, observer, equator.ra, equator.dec, 'normal')
      if (horizon.altitude > 0) visible.push({ at, altitude: horizon.altitude, azimuth: horizon.azimuth })
    }
    sample(first)
    for (let at = Math.ceil(first / minute) * minute; at < last; at += minute) sample(at)
    sample(last)
    if (!visible.length) {
      return { status: 'not_visible', reason: '日食发生时，当前地点的太阳位于地平线下。' }
    }

    const best = visible.reduce((candidate, point) => Math.abs(point.at - peakAt) < Math.abs(candidate.at - peakAt) ? point : candidate)
    const limited = best.altitude < 10
    return {
      status: limited ? 'limited' : 'observable',
      bestAt: new Date(best.at).toISOString(),
      windowStart: new Date(visible[0]!.at).toISOString(),
      windowEnd: new Date(visible.at(-1)!.at).toISOString(),
      azimuthDegrees: best.azimuth,
      altitudeDegrees: best.altitude,
      reason: limited
        ? '此次日食在当前地点可见，但太阳位置较低；观测全程须使用合格的太阳滤镜。'
        : '此次日食在当前地点可见；观测全程须使用合格的太阳滤镜。',
      // 日出/日落时可能仅看到部分食相；食甚在地平线下也要保留接触时刻供说明。
      eclipseContacts: contacts,
    }
  } catch {
    return null // 星历无法计算时沿用后端明确的待计算状态。
  }
}
