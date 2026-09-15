import { test } from 'node:test'
import assert from 'node:assert/strict'
import { constellationLines, skyCatalog } from '../src/skyCatalog.ts'

test('星图目录的名称、坐标与星座连线引用完整', () => {
  const ids = new Set(skyCatalog.map(item => item.id))
  assert.ok(skyCatalog.filter(item => item.kind === 'star').length >= 30)
  assert.ok(skyCatalog.filter(item => item.kind === 'messier').length >= 15)
  for (const item of skyCatalog) {
    assert.ok(item.name && item.nameEn && item.constellation)
    assert.ok(item.raHours >= 0 && item.raHours <= 24)
    assert.ok(item.decDegrees >= -90 && item.decDegrees <= 90)
  }
  for (const constellation of constellationLines) {
    for (const [from, to] of constellation.segments) {
      assert.ok(ids.has(from), `${constellation.name} 缺少连线端点 ${from}`)
      assert.ok(ids.has(to), `${constellation.name} 缺少连线端点 ${to}`)
    }
  }
  const bigDipper = constellationLines.find(item => item.name === '北斗七星')
  assert.equal(new Set(bigDipper?.segments.flat()).size, 7)
})
