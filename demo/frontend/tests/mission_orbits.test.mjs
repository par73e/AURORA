import test from 'node:test'
import assert from 'node:assert/strict'
import { orbitProgress, orbitPeriodSeconds, planetOrbitPeriodSeconds, orbitPeriodText, normalizeMoonOrbit, moonOrbitStage } from '../src/missionOrbits.ts'

test('真实周期：短周期与长周期都在半圈/一圈的真实时刻到位，无速度截断', () => {
  for (const seconds of [195 * 60, 7200, 86400, 6.5 * 86400, 2260 * 86400]) {
    assert.equal(orbitProgress(seconds / 2, seconds), 0.5)
    assert.equal(orbitProgress(seconds, seconds), 0)
    assert.ok(Math.abs(orbitProgress(seconds * 100 + seconds / 4, seconds, 0.1) - 0.35) < 1e-12)
  }
})

test('掉帧与后台恢复：以经过时间决定进度，不依赖渲染帧数', () => {
  const sampleTimes = [0, 0.016, 30, 3600, 7200, 10800]
  assert.deepEqual(sampleTimes.map(t => orbitProgress(t, 7200)), [0, 0.016 / 7200, 30 / 7200, 0.5, 0, 0.5].map(p => ((p % 1) + 1) % 1))
})

test('周期与初始相位安全边界：无效周期不产生 NaN，负相位也规范到轨道上', () => {
  for (const period of [0, -1, NaN, Infinity]) assert.ok(Math.abs(orbitProgress(100, period, 0.3) - 0.3) < 1e-12)
  assert.equal(orbitProgress(0, 3600, -0.25), 0.75)
})

test('面板与动画同源：有效快照优先，无效快照回退标称周期', () => {
  assert.equal(orbitPeriodSeconds({periodSeconds:6780,snapshot:{periodSeconds:7010}}),7010)
  for (const p of [0,-1,NaN,Infinity]) assert.equal(orbitPeriodSeconds({periodSeconds:6780,snapshot:{periodSeconds:p}}),6780)
  assert.equal(orbitPeriodText(orbitPeriodSeconds({periodSeconds:6780,snapshot:{periodSeconds:7010}})), '约 1.9 小时')
})

test('普通行星周期使用真实秒数，贝皮科伦坡保留旧速度', () => {
  assert.equal(planetOrbitPeriodSeconds('magellan',195 / 1440),195 * 60)
  assert.equal(planetOrbitPeriodSeconds('cassini',6.5),6.5 * 86400)
  assert.equal(planetOrbitPeriodSeconds('ulysses',2260),2260 * 86400)
  assert.equal(planetOrbitPeriodSeconds('bepicolombo',120),1200)
})

test('鹊桥二号兼容旧定点数据：生成 24 小时闭合使命轨道，不修改原始对象或鹊桥一号', () => {
  const original = {id:'queqiao2',kind:'stationary',periodSeconds:0}
  const craft = normalizeMoonOrbit(original)
  assert.equal(original.kind,'stationary')
  assert.equal(craft.kind,'orbital')
  assert.equal(orbitPeriodSeconds(craft),86400)
  // MoonScene 将月面以上的半长轴高度放大 3 倍；检查实际显示轨道的近月点。
  const displayedA = 1.12 + (craft.orbitA - 1.12) * 3
  assert.ok(displayedA * (1 - craft.orbitE) > 1.12)
  const queqiao = {id:'queqiao',kind:'stationary'}
  assert.equal(normalizeMoonOrbit(queqiao),queqiao)
  assert.match(moonOrbitStage({id:'apollo-8'}), /1968.*历史/)
  assert.equal(orbitPeriodSeconds(normalizeMoonOrbit({id:'luna-10',kind:'historical_orbit',periodSeconds:10800})),178 * 60)
})
