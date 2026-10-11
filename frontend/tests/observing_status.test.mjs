import test from 'node:test'
import assert from 'node:assert/strict'
import { observingStatus } from '../src/astronomy.ts'

test('高度角状态只描述位置，不把地平线上误写成适合观测', () => {
  assert.equal(observingStatus({ altitude: -0.1 }), '地平线下')
  assert.equal(observingStatus({ altitude: 3 }), '地平线上 · 低空')
  assert.equal(observingStatus({ altitude: 45 }), '地平线上')
})
