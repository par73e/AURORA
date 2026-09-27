import { test } from 'node:test'
import assert from 'node:assert/strict'
import { localSolarEclipseVisibility } from '../src/solarEclipseVisibility.ts'

const eclipse = { kind: 'solar_eclipse', startsAt: '2026-08-12T17:13:00Z' }

test('2026 年 8 月日食只在当地有食相且太阳升起时标为可观测', () => {
  const iceland = localSolarEclipseVisibility(eclipse, 64.15, -21.94)
  assert.equal(iceland.status, 'observable')
  assert.ok(iceland.windowStart < iceland.bestAt && iceland.bestAt < iceland.windowEnd)
  assert.ok(iceland.altitudeDegrees > 10)

  const madrid = localSolarEclipseVisibility(eclipse, 40.42, -3.7)
  assert.equal(madrid.status, 'limited')
  assert.ok(madrid.altitudeDegrees > 0 && madrid.altitudeDegrees < 10)

  const shanghai = localSolarEclipseVisibility(eclipse, 31.23, 121.47)
  assert.equal(shanghai.status, 'not_visible')
})

test('日食事件日期不能借用地点的下一次日食', () => {
  assert.equal(localSolarEclipseVisibility({ ...eclipse, startsAt: '2027-01-01T12:00:00Z' }, 64.15, -21.94).status, 'not_visible')
  assert.equal(localSolarEclipseVisibility({ ...eclipse, startsAt: 'invalid' }, 64.15, -21.94), null)
})
