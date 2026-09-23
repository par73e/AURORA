import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { ENTRY_SPIN_DURATION_MS, entrySpinAngle, entrySpinFinished } from '../src/entrySpin.ts'

test('入场自转在 800 毫秒内保持单向并精确停在最终姿态', () => {
  assert.equal(ENTRY_SPIN_DURATION_MS, 800)
  for (const offset of [-12, -14, 14]) {
    let previous = entrySpinAngle(offset, 1000, 1000)
    assert.equal(previous, offset)
    for (let elapsed = 10; elapsed <= 800; elapsed += 10) {
      const angle = entrySpinAngle(offset, 1000, 1000 + elapsed)
      assert.ok(offset < 0 ? angle >= previous : angle <= previous)
      previous = angle
    }
    assert.equal(previous, 0)
  }
  assert.equal(entrySpinFinished(1000, 1799), false)
  assert.equal(entrySpinFinished(1000, 1800), true)
})

test('全部天体特写使用统一自转时钟，地球附属信息在停转后出现', async () => {
  for (const component of ['OrbitScene.vue', 'MoonScene.vue', 'MarsScene.vue', 'PlanetScene.vue']) {
    const source = await readFile(new URL(`../src/components/${component}`, import.meta.url), 'utf8')
    assert.match(source, /entrySpinAngle\(/, `${component} 应使用共享角度曲线`)
    assert.match(source, /entrySpinFinished\(/, `${component} 应使用共享停转边界`)
  }
  const orbit = await readFile(new URL('../src/components/OrbitScene.vue', import.meta.url), 'utf8')
  assert.match(orbit, /ELEMENTS_REVEAL_DELAY_MS = ENTRY_SPIN_DURATION_MS \+ 20/)
})
