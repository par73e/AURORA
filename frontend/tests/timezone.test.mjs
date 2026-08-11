import { test } from 'node:test'
import assert from 'node:assert/strict'
import { dateFromZonedLocalTime, zonedDateAtMinute, zonedDateKey, zonedMinuteOfDay } from '../src/astronomy.ts'

test('地点本地时间按 IANA 时区解释，不依赖运行测试的设备时区', () => {
  const shanghaiNoon = dateFromZonedLocalTime('2026-08-10T12:00', 'Asia/Shanghai')
  assert.equal(shanghaiNoon.toISOString(), '2026-08-10T04:00:00.000Z')
  assert.equal(zonedDateKey(shanghaiNoon, 'Asia/Shanghai'), '2026-08-10')
  assert.equal(zonedMinuteOfDay(shanghaiNoon, 'Asia/Shanghai'), 720)
  assert.equal(zonedMinuteOfDay(shanghaiNoon, 'America/New_York'), 0)
})

test('时间拨杆保留定位点的当地日期并替换分钟', () => {
  const instant = new Date('2026-08-09T16:30:00.000Z') // 上海 8 月 10 日 00:30
  const localNoon = zonedDateAtMinute(instant, 720, 'Asia/Shanghai')
  assert.equal(localNoon.toISOString(), '2026-08-10T04:00:00.000Z')
})
