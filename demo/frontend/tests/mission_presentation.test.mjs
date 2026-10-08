import test from 'node:test'
import assert from 'node:assert/strict'
import { landingCategoryLabel, spacecraftTypeLabel } from '../src/missionPresentation.ts'

test('月球和火星任务类型按类别显示，未知类别保留原值', () => {
  assert.equal(spacecraftTypeLabel('CNSA LUNAR ORBITER'), '月球轨道器')
  assert.equal(spacecraftTypeLabel('ESA/ROSCOSMOS MARS ORBITER'), '火星轨道器')
  assert.equal(spacecraftTypeLabel('CNSA RELAY SATELLITE'), '中继卫星')
  assert.equal(spacecraftTypeLabel('TEST PROBE'), 'TEST PROBE')
  assert.equal(landingCategoryLabel('CREWED_LANDING'), '载人登月')
  assert.equal(landingCategoryLabel('STATIC_LANDER'), '静态着陆')
  assert.equal(landingCategoryLabel('UNKNOWN_CATEGORY'), 'UNKNOWN_CATEGORY')
})
