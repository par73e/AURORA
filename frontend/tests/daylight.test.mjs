import { test } from 'node:test'
import assert from 'node:assert/strict'
import { Body, Observer, SearchRiseSet, SearchAltitude } from 'astronomy-engine'
import { daylightFactor } from '../src/astronomy.ts'

// 与 src/astronomy.ts 中 calculateTwilight 相同的调用方式（上海坐标）。
function twilightAt(latitude, longitude, dayStart) {
  const place = new Observer(latitude, longitude, 0)
  return {
    sunrise: SearchRiseSet(Body.Sun, place, 1, dayStart, 1.1)?.date ?? null,
    sunset: SearchRiseSet(Body.Sun, place, -1, dayStart, 1.1)?.date ?? null,
    astronomicalDawn: SearchAltitude(Body.Sun, place, 1, dayStart, 1.1, -18)?.date ?? null,
    astronomicalDusk: SearchAltitude(Body.Sun, place, -1, dayStart, 1.1, -18)?.date ?? null,
  }
}

function minutesInto(at, dayStart) {
  const value = new Date(dayStart)
  value.setHours(0, 0, 0, 0)
  return Math.round((at.getTime() - value.getTime()) / 60_000)
}

test('daylightFactor 依据当天当地日出/日落切换昼夜，晨昏为渐变过渡', () => {
  const dayStart = new Date()
  dayStart.setHours(0, 0, 0, 0)
  const times = twilightAt(31.23, 121.47, dayStart) // 上海
  assert.ok(times.sunrise && times.sunset && times.astronomicalDawn && times.astronomicalDusk,
    '中纬度当天应能算出日出/日落与天文晨昏')

  const at = (minutes) => new Date(dayStart.getTime() + minutes * 60_000)
  const dawn = minutesInto(times.astronomicalDawn, dayStart)
  const sunrise = minutesInto(times.sunrise, dayStart)
  const sunset = minutesInto(times.sunset, dayStart)
  const dusk = minutesInto(times.astronomicalDusk, dayStart)
  assert.ok(dawn < sunrise && sunrise < sunset && sunset < dusk, '时间顺序：晨光 < 日出 < 日落 < 昏影')

  // 深夜与深夜前后均为 0
  assert.equal(daylightFactor(times, at(dawn - 1)), 0)
  assert.equal(daylightFactor(times, at(3 * 60)), 0)
  // 日出/日落之间恒为 1（正午取 12:00，保证在区间内）
  assert.equal(daylightFactor(times, at(12 * 60)), 1)
  assert.equal(daylightFactor(times, at(sunrise)), 1)
  assert.equal(daylightFactor(times, at(sunset)), 1)
  // 昏影之后回到 0
  assert.equal(daylightFactor(times, at(dusk + 1)), 0)
  assert.equal(daylightFactor(times, at(23 * 60)), 0)

  // 晨光→日出 单调上升且严格落在 (0,1)
  const morning = [0.25, 0.5, 0.75].map((ratio) => daylightFactor(times, at(dawn + Math.round((sunrise - dawn) * ratio))))
  assert.ok(morning.every((value) => value > 0 && value < 1), `晨间渐变应在 (0,1)：${morning}`)
  assert.ok(morning[0] < morning[1] && morning[1] < morning[2], '晨间渐变应单调上升')
  // 日落→昏影 单调下降且严格落在 (0,1)
  const evening = [0.25, 0.5, 0.75].map((ratio) => daylightFactor(times, at(sunset + Math.round((dusk - sunset) * ratio))))
  assert.ok(evening.every((value) => value > 0 && value < 1), `昏间渐变应在 (0,1)：${evening}`)
  assert.ok(evening[0] > evening[1] && evening[1] > evening[2], '昏间渐变应单调下降')
})

test('daylightFactor 极昼极夜/数据缺失时按黑夜处理', () => {
  const dayStart = new Date()
  dayStart.setHours(0, 0, 0, 0)
  const missing = { sunrise: null, sunset: null, astronomicalDawn: null, astronomicalDusk: null }
  assert.equal(daylightFactor(missing, new Date(dayStart.getTime() + 12 * 3_600_000)), 0)
})
