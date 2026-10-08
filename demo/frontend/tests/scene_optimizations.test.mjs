import { test } from 'node:test'
import assert from 'node:assert/strict'
import { stableLayouts } from '../src/performance/stableLayouts.ts'
import { positionMix } from '../src/orbit/positionInterpolation.ts'
import { wantsDetail } from '../src/solar/textureResolution.ts'

test('unchanged labels reuse nodes and arrays without rounding physical connector geometry', () => {
  const a = [{ id:'a', x:12.345, anchorX:2.123, visible:true, memberIds:['a'] }]
  const b = stableLayouts(a, [{...a[0], memberIds:['a']}])
  assert.equal(b, a)
  assert.equal(b[0].x, 12.345)
  assert.equal(stableLayouts(a, [{...a[0], x:12.35}]), a)
  assert.notEqual(stableLayouts(a, [{...a[0], x:14}]), a)
  assert.notEqual(stableLayouts(a, [{...a[0], visible:false}]), a)
  assert.notEqual(stableLayouts(a, [{...a[0], memberIds:['a','b']}]), a)
})
test('position interpolation remains bounded and rejects stale/invalid times', () => {
  assert.equal(positionMix(1500,1000,2000),.5)
  assert.equal(positionMix(500,1000,2000),0)
  assert.equal(positionMix(2500,1000,2000),1)
  assert.equal(positionMix(8000,1000,2000),null)
  assert.equal(positionMix(1500,2000,2000),null)
  assert.equal(positionMix(NaN,1000,2000),null)
})
test('texture tiers have hysteresis so small wheel/resize changes do not toggle maps', () => {
  assert.equal(wantsDetail(1700,false),false)
  assert.equal(wantsDetail(1900,false),true)
  assert.equal(wantsDetail(1700,true),true)
  assert.equal(wantsDetail(1300,true),false)
})
