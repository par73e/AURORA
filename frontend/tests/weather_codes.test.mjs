import { test } from 'node:test'
import assert from 'node:assert/strict'
import { conditionDescription, forecastHoursThroughTomorrow, forecastIsStale, weatherGlyph } from '../src/observatoryWeather.ts'

test('WMO 天气代码区分雾、阵雨、阵雪、雷暴与冰雹', () => {
  assert.equal(conditionDescription(3), '阴天')
  assert.equal(conditionDescription(48), '雾')
  assert.equal(conditionDescription(80), '阵雨')
  assert.equal(conditionDescription(85), '阵雪')
  assert.equal(conditionDescription(95), '雷暴')
  assert.equal(conditionDescription(99), '雷暴伴冰雹')
  assert.equal(conditionDescription(999), '天气状况未知')
  assert.equal(weatherGlyph(85), '❄')
  assert.equal(weatherGlyph(99), 'ϟ')
  assert.equal(weatherGlyph(999), '·')
})

test('天气新鲜度同时检查获取时间和当地当前观测时间', () => {
  const now = new Date('2026-10-12T00:10:00Z') // 上海 08:10，已跨当地午夜
  assert.equal(forecastIsStale('2026-10-11T23:30:00Z', '2026-10-12T07:45', 'Asia/Shanghai', now), false)
  assert.equal(forecastIsStale('2026-10-11T22:00:00Z', '2026-10-12T07:45', 'Asia/Shanghai', now), true)
  assert.equal(forecastIsStale('2026-10-11T23:30:00Z', '2026-10-12T05:00', 'Asia/Shanghai', now), true)
  assert.equal(forecastIsStale('', '2026-10-12T07:45', 'Asia/Shanghai', now), true)
})

test('天气栏从用户当地当前整点显示到明日24点', () => {
  const hours = [
    '2026-09-15T16:00',
    '2026-09-15T17:00',
    '2026-09-15T23:00',
    '2026-09-16T00:00',
    '2026-09-16T23:00',
    '2026-09-17T00:00',
  ].map((time) => ({ time }))

  const result = forecastHoursThroughTomorrow(hours, new Date('2026-09-15T09:26:00.000Z'), 'Asia/Shanghai')
  assert.deepEqual(result.map((hour) => hour.time), [
    '2026-09-15T17:00',
    '2026-09-15T23:00',
    '2026-09-16T00:00',
    '2026-09-16T23:00',
  ])
})
