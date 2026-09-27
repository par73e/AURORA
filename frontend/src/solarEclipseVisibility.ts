import { Body, Equator, Horizon, Observer, SearchLocalSolarEclipse } from 'astronomy-engine'
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
    }
  } catch {
    return null // 星历无法计算时沿用后端明确的待计算状态。
  }
}
