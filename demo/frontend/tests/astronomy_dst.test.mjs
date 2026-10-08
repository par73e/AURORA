import { test } from 'node:test'
import assert from 'node:assert/strict'
import { bodies, calculateTrack, zonedDateKey } from '../src/astronomy.ts'

test('星图整日轨迹在夏令时切换日按相邻当地午夜采样', () => {
  const moon = bodies.find((body) => body.id === 'moon')
  const timezone = 'America/New_York'
  for (const { at, end, count } of [
    { at: '2026-03-08T12:00:00Z', end: '2026-03-09T04:00:00.000Z', count: 93 },
    { at: '2026-11-01T12:00:00Z', end: '2026-11-02T05:00:00.000Z', count: 101 },
  ]) {
    const track = calculateTrack(moon, new Date(at), 40.7, -74, 0, timezone)
    assert.equal(track.samples.length, count)
    assert.equal(track.samples.at(-1).at.toISOString(), end)
    assert.equal(zonedDateKey(track.samples.at(-2).at, timezone), zonedDateKey(new Date(at), timezone))
    for (const event of [track.rise, track.set, track.transit, track.best]) {
      if (event) assert.equal(zonedDateKey(event, timezone), zonedDateKey(new Date(at), timezone))
    }
  }
})
