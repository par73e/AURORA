import test from 'node:test'
import assert from 'node:assert/strict'
import { landingCategoryLabel, spacecraftTypeLabel, sampleCoverage, snapshotNotice } from '../src/missionPresentation.ts'

test('月球和火星任务类型按类别显示，未知类别保留原值', () => {
  assert.equal(spacecraftTypeLabel('CNSA LUNAR ORBITER'), '月球轨道器')
  assert.equal(spacecraftTypeLabel('ESA/ROSCOSMOS MARS ORBITER'), '火星轨道器')
  assert.equal(spacecraftTypeLabel('CNSA RELAY SATELLITE'), '中继卫星')
  assert.equal(spacecraftTypeLabel('TEST PROBE'), 'TEST PROBE')
  assert.equal(landingCategoryLabel('CREWED_LANDING'), '载人登月')
  assert.equal(landingCategoryLabel('STATIC_LANDER'), '静态着陆')
  assert.equal(landingCategoryLabel('UNKNOWN_CATEGORY'), 'UNKNOWN_CATEGORY')
})

test('采样覆盖取真实首末历元，获取时间不能代替覆盖范围', () => {
  const start = '2026-10-01T00:00:00Z', end = '2026-10-10T00:00:00Z'
  const samples = [{ epoch: end }, { epoch: 'invalid' }, { epoch: start }]
  assert.equal(sampleCoverage(samples, Date.parse(start)).covered, true)
  assert.equal(sampleCoverage(samples, Date.parse(end)).covered, true)
  assert.equal(sampleCoverage(samples, Date.parse('2026-10-12T00:00:00Z')).covered, false)
  assert.match(sampleCoverage(samples).label, /2026-10-01.*2026-10-10/)
  assert.equal(sampleCoverage([]), null)
})

test('同步失败不冒称新鲜，资料年龄提醒不声称物理精度', () => {
  const now = Date.parse('2026-10-12T00:00:00Z')
  assert.match(snapshotNotice('2026-10-08T00:00:00Z', 'failed', now), /保留上次资料.*48 小时/)
  assert.equal(snapshotNotice('2026-10-11T00:00:00Z', 'success', now), undefined)
  assert.match(snapshotNotice(null, 'unsupported', now), /暂无实时星历支持/)
})
