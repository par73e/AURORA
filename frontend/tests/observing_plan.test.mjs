import test from 'node:test'
import assert from 'node:assert/strict'
import { remainingNightWindow, forecastAt, nightForecast } from '../src/observingPlan.ts'

const at = s => new Date(s)
const previous = { start: at('2026-10-11T19:00:00+08:00'), end: at('2026-10-12T05:00:00+08:00') }
const tonight = { start: at('2026-10-12T19:00:00+08:00'), end: at('2026-10-13T05:00:00+08:00') }

test('晚间只规划剩余时段，午夜后继续本夜，晨光后再进入下一夜', () => {
  for (const now of ['2026-10-11T23:30:20+08:00', '2026-10-12T00:30:20+08:00']) {
    const window = remainingNightWindow(at(now), now.includes('11T') ? previous : tonight, previous)
    assert.equal(window.start.getTime(), at(now).getTime() + 40_000)
    assert.equal(window.end.getTime(), previous.end.getTime())
    assert.ok(window.start >= at(now))
  }
  assert.deepEqual(remainingNightWindow(at('2026-10-12T05:00:00+08:00'), tonight, previous), tonight)
  assert.equal(remainingNightWindow(at('2026-10-12T18:00:00+08:00'), null, null), null)
})

test('计划不采用过去窗口的高分，当前小时覆盖与未来预报边界明确', () => {
  const now = at('2026-10-11T23:30:00+08:00')
  const window = remainingNightWindow(now, previous, null)
  const hours = [
    { time: '2026-10-11T19:00:00+08:00', score: 100 },
    { time: '2026-10-11T23:00:00+08:00', score: 50 },
    { time: '2026-10-12T00:00:00+08:00', score: 70 },
    { time: '2026-10-12T05:00:00+08:00', score: 90 },
  ]
  const candidates = nightForecast(hours, window, at)
  assert.deepEqual(candidates.map(h => h.score), [50, 70])
  assert.equal(forecastAt(candidates, now, at).score, 50)
  assert.equal(forecastAt(candidates, at('2026-10-12T00:00:00+08:00'), at).score, 70)
  assert.equal(forecastAt(candidates, at('2026-10-12T01:00:00+08:00'), at), null)
})
