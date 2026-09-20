import { test } from 'node:test'
import assert from 'node:assert/strict'
import { solarSession } from '../src/solar/session.ts'

test('所有天体首次进入默认显示飞行器，只记忆用户在新版中的主动选择', () => {
  const originalLocalStorage = Object.getOwnPropertyDescriptor(globalThis, 'localStorage')
  const values = new Map([
    ['aurora.solar.spacecraftVisible', '0'],
    ['aurora.solar.spacecraftVisible.v2', '0'],
    ['aurora.solar.spacecraftVisible.v3', '0'],
    ['aurora.solar.spacecraftVisible.v4', '0'],
    ['aurora.solar.spacecraftVisibilityVersion', '3'],
  ])
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: {
      getItem(key) {
        return values.get(key) ?? null
      },
      setItem(key, value) {
        values.set(key, String(value))
      },
    },
  })

  try {
    assert.equal(solarSession.spacecraftVisible, true, '所有旧键和旧版本记录都不应覆盖新的默认展示方式')

    solarSession.spacecraftVisible = false
    assert.equal(values.get('aurora.solar.spacecraftVisible.v5'), '0')
    assert.equal(solarSession.spacecraftVisible, false)

    solarSession.spacecraftVisible = true
    assert.equal(values.get('aurora.solar.spacecraftVisible.v5'), '1')
    assert.equal(solarSession.spacecraftVisible, true)
  } finally {
    if (originalLocalStorage === undefined) delete globalThis.localStorage
    else Object.defineProperty(globalThis, 'localStorage', originalLocalStorage)
  }
})
