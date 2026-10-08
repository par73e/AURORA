import { test } from 'node:test'
import assert from 'node:assert/strict'
import { AdaptiveResolution } from '../src/performance/resolution.ts'

test('startup and a single slow window preserve native resolution', () => {
  const r = new AdaptiveResolution(2)
  assert.equal(r.observe(40, 60, 1000), false)
  assert.equal(r.observe(40, 60, 2000), false)
  assert.equal(r.observe(40, 60, 3000), false)
  assert.equal(r.observe(16.7, 17, 4000), false)
  assert.equal(r.ratio, 2)
})

test('sustained pressure reduces resolution with cooldown and a floor', () => {
  const r = new AdaptiveResolution(2)
  r.observe(30, 35, 3000)
  assert.equal(r.observe(30, 35, 4000), true)
  assert.equal(r.ratio, 1.7)
  assert.equal(r.observe(30, 35, 5000), false)
  for (let t = 6000; t < 40000; t += 1000) r.observe(30, 35, t)
  assert.equal(r.ratio, 1)
})

test('recovery requires consecutive smooth windows, preserves fractional device DPR', () => {
  const r = new AdaptiveResolution(1.25)
  r.observe(30, 35, 3000); r.observe(30, 35, 4000)
  assert.equal(r.ratio, 1.06)
  for (let t = 7000; t < 12000; t += 1000) assert.equal(r.observe(16.7, 18, t), false)
  assert.equal(r.observe(16.7, 18, 12000), true)
  assert.equal(r.ratio, 1.16)
  for (let t = 13000; t < 40000; t += 1000) r.observe(16.7, 18, t)
  assert.equal(r.ratio, 1.25)
})

test('visibility reset prevents old pressure or recovery streaks carrying forward', () => {
  const r = new AdaptiveResolution(2)
  r.observe(30, 35, 3000); r.resetStreaks()
  assert.equal(r.observe(30, 35, 4000), false)
  assert.equal(r.observe(30, 35, 5000), true)
})

test('jittery windows do not trigger recovery and DPR 1 remains at its floor', () => {
  const r = new AdaptiveResolution(2)
  r.observe(30, 35, 3000); r.observe(30, 35, 4000)
  for (let t = 7000; t < 20000; t += 1000) r.observe(16, 40, t)
  assert.equal(r.ratio, 1.7)
  const low = new AdaptiveResolution(1)
  for (let t = 3000; t < 20000; t += 1000) low.observe(40, 50, t)
  assert.equal(low.ratio, 1)
})
