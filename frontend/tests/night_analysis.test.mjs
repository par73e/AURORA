import { test } from 'node:test'
import assert from 'node:assert/strict'
import { analyzeNight } from '../src/astronomy.ts'

// 上海：北纬 31.23、东经 121.47，夏季（8 月）——银河核心（赤纬 −29°）在
// 当地最高约 30°，应存在天文夜内的可见窗口。
const SHANGHAI = { latitude: 31.23, longitude: 121.47 }

function localDate(month, day, hour = 12) {
  return new Date(2026, month - 1, day, hour, 0, 0)
}

test('analyzeNight 天文夜跨午夜、无月黑夜与银河核心窗口结构完整', () => {
  const result = analyzeNight(localDate(8, 9), SHANGHAI.latitude, SHANGHAI.longitude)
  assert.ok(result.astronomicalNight, '上海 8 月应有天文夜')
  const start = result.astronomicalNight.start
  const end = result.astronomicalNight.end
  assert.ok(start.getTime() < end.getTime(), '天文夜起点早于终点')
  assert.ok(start.getHours() >= 18 || start.getHours() <= 4, '昏影约在傍晚/深夜开始（当地时间）')
  assert.ok(result.moonlessWindows.length >= 1, '天文夜内应有无月黑夜窗口（月出/月落之外）')
  for (const window of result.moonlessWindows) {
    assert.ok(window.start.getTime() < window.end.getTime(), '无月黑夜窗口有序')
    assert.ok(window.start.getTime() >= start.getTime(), '无月黑夜不早于天文夜开始')
    assert.ok(window.end.getTime() <= end.getTime(), '无月黑夜不晚于天文夜结束')
  }
  // 银河核心最高约 30°（90 − |31.23 + 29| = 29.8°），应落在合理区间
  assert.ok(result.galacticCore.maxAltitude >= 20, `银河核心最高高度应可观（实际 ${result.galacticCore.maxAltitude}°）`)
  assert.ok(result.galacticCore.maxAltitude <= 32, `银河核心最高高度不应超过地理上限（实际 ${result.galacticCore.maxAltitude}°）`)
  assert.ok(result.galacticCore.maxTime, '应有最高高度对应时刻')
  assert.ok(result.galacticCore.windows.length >= 1, '天文夜内应有银河核心可见窗口')
  // 银河核心最高时刻应在天文夜窗口内
  const maxAt = result.galacticCore.maxTime.getTime()
  assert.ok(maxAt >= start.getTime() && maxAt <= end.getTime(), '银河核心最高时刻应在天文夜内')
})

test('analyzeNight 高纬度冬季无天文夜时返回空结构', () => {
  // 北极圈内 8 月极昼：太阳不落到 −18° 以下
  const polar = analyzeNight(localDate(8, 9), 78, 16)
  assert.equal(polar.astronomicalNight, null)
  assert.deepEqual(polar.moonlessWindows, [])
  assert.equal(polar.galacticCore.maxAltitude, 0)
  assert.equal(polar.galacticCore.maxTime, null)
  assert.deepEqual(polar.galacticCore.windows, [])
})
