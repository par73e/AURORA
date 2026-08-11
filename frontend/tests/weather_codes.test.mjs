import { test } from 'node:test'
import assert from 'node:assert/strict'
import { conditionDescription, weatherGlyph } from '../src/observatoryWeather.ts'

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
