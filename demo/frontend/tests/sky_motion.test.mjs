import { test } from 'node:test'
import assert from 'node:assert/strict'
import { easeOutExpo, normalizeAzimuth, shortestAzimuthDelta, skyTurnDuration } from '../src/skyMotion.ts'

test('罗盘定位跨越正北时始终选择最短方向', () => {
  assert.equal(shortestAzimuthDelta(350, 10), 20)
  assert.equal(shortestAzimuthDelta(10, 350), -20)
  assert.equal(shortestAzimuthDelta(45, 225), -180)
})

test('罗盘朝向始终归一化到 0–360°', () => {
  assert.equal(normalizeAzimuth(361), 1)
  assert.equal(normalizeAzimuth(-1), 359)
})

test('自动定位时长有上下限，缓动自然减速并准确到达', () => {
  assert.equal(skyTurnDuration(0), 320)
  assert.equal(skyTurnDuration(180), 676)
  assert.ok(easeOutExpo(.5) > .9)
  assert.equal(easeOutExpo(1), 1)
})
