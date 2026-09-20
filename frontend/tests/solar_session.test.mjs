import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { createDefaultSceneLayers } from '../src/sceneLayers.ts'

test('所有天体共用地球图层状态，应用首次进入默认显示飞行器', () => {
  const layers = createDefaultSceneLayers()
  assert.deepEqual(layers, { spacecraft: true, orbits: true, sites: true })

  layers.spacecraft = false
  assert.equal(layers.spacecraft, false, '用户的选择由同一个 App 图层对象跟随进入和退出其他天体')
})

test('太阳系、月球、火星和所有其他天体都绑定地球的飞行器图层状态', async () => {
  const appSource = await readFile(new URL('../src/App.vue', import.meta.url), 'utf8')
  const sharedBindings = appSource.match(/v-model:spacecraft-visible="layers\.spacecraft"/g) ?? []

  assert.equal(sharedBindings.length, 10, '月球、火星、七个共用天体页与太阳系都应共用地球图层状态')
  assert.match(appSource, /<input v-model="layers\.spacecraft" type="checkbox">/)

  for (const component of ['SolarSystem.vue', 'MoonScene.vue', 'MarsScene.vue', 'PlanetScene.vue']) {
    const source = await readFile(new URL(`../src/components/${component}`, import.meta.url), 'utf8')
    assert.doesNotMatch(source, /solarSession\.spacecraftVisible|spacecraftVisible\.v\d|spacecraftVisibilityVersion/)
  }
})
