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
    civilDawn: SearchAltitude(Body.Sun, place, 1, dayStart, 1.1, -6)?.date ?? null,
    civilDusk: SearchAltitude(Body.Sun, place, -1, dayStart, 1.1, -6)?.date ?? null,
  }
}

function minutesInto(at, dayStart) {
  const value = new Date(dayStart)
  value.setHours(0, 0, 0, 0)
  return Math.round((at.getTime() - value.getTime()) / 60_000)
}

// 边界断言使用真实时刻（不再经过分钟取整）：日出/日落本身是硬边界，
// 构造的时刻若因取整早于真实时刻几十秒，会得到 0.9956 而非 1 的脆弱结果。
function atExact(value) {
  return new Date(value.getTime())
}
function plusMinutes(value, minutes) {
  return new Date(value.getTime() + minutes * 60_000)
}

test('daylightFactor 依据当天当地日出/日落切换昼夜，民用晨昏为渐变过渡', () => {
  const dayStart = new Date()
  dayStart.setHours(0, 0, 0, 0)
  const times = twilightAt(31.23, 121.47, dayStart) // 上海
  assert.ok(times.sunrise && times.sunset && times.civilDawn && times.civilDusk,
    '中纬度当天应能算出日出/日落与民用晨昏')

  const at = (minutes) => new Date(dayStart.getTime() + minutes * 60_000)
  const civilDawn = minutesInto(times.civilDawn, dayStart)
  const sunrise = minutesInto(times.sunrise, dayStart)
  const sunset = minutesInto(times.sunset, dayStart)
  const civilDusk = minutesInto(times.civilDusk, dayStart)
  assert.ok(civilDawn < sunrise && sunrise < sunset && sunset < civilDusk, '时间顺序：民用晨光 < 日出 < 日落 < 民用昏影')

  // 深夜与深夜前后均为 0
  assert.equal(daylightFactor(times, atExact(plusMinutes(times.civilDawn, -1))), 0)
  assert.equal(daylightFactor(times, at(3 * 60)), 0)
  // 日出/日落之间恒为 1（正午取 12:00，保证在区间内）
  assert.equal(daylightFactor(times, at(12 * 60)), 1)
  assert.equal(daylightFactor(times, atExact(times.sunrise)), 1)
  assert.equal(daylightFactor(times, atExact(times.sunset)), 1)
  // 民用昏影之后回到 0
  assert.equal(daylightFactor(times, atExact(plusMinutes(times.civilDusk, 1))), 0)
  assert.equal(daylightFactor(times, at(23 * 60)), 0)

  // 民用晨光→日出 单调上升且严格落在 (0,1)
  const morning = [0.25, 0.5, 0.75].map((ratio) => daylightFactor(times, at(civilDawn + Math.round((sunrise - civilDawn) * ratio))))
  assert.ok(morning.every((value) => value > 0 && value < 1), `晨间渐变应在 (0,1)：${morning}`)
  assert.ok(morning[0] < morning[1] && morning[1] < morning[2], '晨间渐变应单调上升')
  // 日落→民用昏影 单调下降且严格落在 (0,1)
  const evening = [0.25, 0.5, 0.75].map((ratio) => daylightFactor(times, at(sunset + Math.round((civilDusk - sunset) * ratio))))
  assert.ok(evening.every((value) => value > 0 && value < 1), `昏间渐变应在 (0,1)：${evening}`)
  assert.ok(evening[0] > evening[1] && evening[1] > evening[2], '昏间渐变应单调下降')

  // 视觉天色用民用晨昏：日落后 1 小时（太阳约 −10°，已在民用昏影之下）应接近全黑
  const oneHourAfterSunset = atExact(plusMinutes(times.sunset, 60))
  const atOneHour = daylightFactor(times, oneHourAfterSunset)
  assert.ok(atOneHour === 0, `日落 1 小时后应已全黑（实际 ${atOneHour}）`)
})

test('daylightFactor 民用渐变带明显短于天文渐变带（天黑更快）', () => {
  const dayStart = new Date()
  dayStart.setHours(0, 0, 0, 0)
  const times = twilightAt(31.23, 121.47, dayStart)
  assert.ok(times.civilDusk && times.astronomicalDusk, '应有民用与天文昏影')
  const civilSpan = times.civilDusk.getTime() - times.sunset.getTime()
  const astronomicalSpan = times.astronomicalDusk.getTime() - times.sunset.getTime()
  assert.ok(civilSpan < astronomicalSpan, '民用渐变带应比天文渐变带短')
  assert.ok(astronomicalSpan - civilSpan > 30 * 60_000, `两者差距应超过 30 分钟（民用 ${Math.round(civilSpan / 60_000)}min，天文 ${Math.round(astronomicalSpan / 60_000)}min）`)
})

test('daylightFactor 极昼极夜/数据缺失时按黑夜处理', () => {
  const dayStart = new Date()
  dayStart.setHours(0, 0, 0, 0)
  const missing = { sunrise: null, sunset: null, astronomicalDawn: null, astronomicalDusk: null, civilDawn: null, civilDusk: null }
  assert.equal(daylightFactor(missing, new Date(dayStart.getTime() + 12 * 3_600_000)), 0)
})
