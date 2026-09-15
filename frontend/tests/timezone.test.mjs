import { test } from 'node:test'
import assert from 'node:assert/strict'
import { dateFromZonedLocalTime, zonedDateAtMinute, zonedDateKey, zonedDateKeyAfterDays, zonedMinuteOfDay } from '../src/astronomy.ts'

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

test('明日日期按观测点自然日跨月和跨年', () => {
  const shanghaiYearEnd = new Date('2026-12-31T16:30:00.000Z') // 上海 2027-01-01 00:30
  assert.equal(zonedDateKeyAfterDays(shanghaiYearEnd, 'Asia/Shanghai', 1), '2027-01-02')

  const newYorkMonthEnd = new Date('2026-03-01T02:00:00.000Z') // 纽约 2026-02-28 21:00
  assert.equal(zonedDateKeyAfterDays(newYorkMonthEnd, 'America/New_York', 1), '2026-03-01')
})
