import { test } from 'node:test'
import assert from 'node:assert/strict'
import { solarSession } from '../src/solar/session.ts'

test('飞行器偏好升级后默认显示，并继续记忆用户的新选择', () => {
  const originalLocalStorage = Object.getOwnPropertyDescriptor(globalThis, 'localStorage')
  const values = new Map([
    ['aurora.solar.spacecraftVisible', '0'],
    ['aurora.solar.spacecraftVisible.v2', '0'],
    ['aurora.solar.spacecraftVisible.v3', '0'],
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
    assert.equal(solarSession.spacecraftVisible, true, '旧版隐藏记录不应覆盖新的默认展示方式')

    solarSession.spacecraftVisible = false
    assert.equal(values.get('aurora.solar.spacecraftVisible.v4'), '0')
    assert.equal(values.get('aurora.solar.spacecraftVisibilityVersion'), '1')
    assert.equal(solarSession.spacecraftVisible, false)

    solarSession.spacecraftVisible = true
    assert.equal(values.get('aurora.solar.spacecraftVisible.v4'), '1')
    assert.equal(solarSession.spacecraftVisible, true)
  } finally {
    if (originalLocalStorage === undefined) delete globalThis.localStorage
    else Object.defineProperty(globalThis, 'localStorage', originalLocalStorage)
  }
})
