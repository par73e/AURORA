import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { createDefaultSceneLayers } from '../src/sceneLayers.ts'

test('所有天体特写页共用地球图层状态，应用首次进入默认显示飞行器', () => {
  const layers = createDefaultSceneLayers()
  assert.deepEqual(layers, { spacecraft: true, orbits: true, sites: true })

  layers.spacecraft = false
  assert.equal(layers.spacecraft, false, '用户的选择由同一个 App 图层对象跟随进入和退出其他天体')
})

test('太阳系总览与天体特写页的飞行器状态相互独立', async () => {
  const appSource = await readFile(new URL('../src/App.vue', import.meta.url), 'utf8')
  const sharedBindings = appSource.match(/v-model:spacecraft-visible="layers\.spacecraft"/g) ?? []

  assert.equal(sharedBindings.length, 9, '月球、火星与七个共用天体页应继续共用地球图层状态')
  assert.match(appSource, /const solarSpacecraftVisible = ref\(true\)/)
  assert.match(appSource, /<SolarSystem[\s\S]*?v-model:spacecraft-visible="solarSpacecraftVisible"/)
  assert.match(appSource, /<input v-model="layers\.spacecraft" type="checkbox">/)

  for (const component of ['SolarSystem.vue', 'MoonScene.vue', 'MarsScene.vue', 'PlanetScene.vue']) {
    const source = await readFile(new URL(`../src/components/${component}`, import.meta.url), 'utf8')
    assert.doesNotMatch(source, /solarSession\.spacecraftVisible|spacecraftVisible\.v\d|spacecraftVisibilityVersion/)
  }
})

test('太阳系位置模式首次进入默认排布，应用内离开再返回仍保留选择', async () => {
  const sessionSource = await readFile(new URL('../src/solar/session.ts', import.meta.url), 'utf8')
  const solarSource = await readFile(new URL('../src/components/SolarSystem.vue', import.meta.url), 'utf8')

  assert.match(sessionSource, /let realPositions = false/)
  assert.match(sessionSource, /get realPositions\(\): boolean \{\s*return realPositions\s*\}/)
  assert.match(sessionSource, /set realPositions\(value: boolean\) \{\s*realPositions = value\s*\}/)
  assert.doesNotMatch(sessionSource, /localStorage|sessionStorage/)
  assert.match(solarSource, /const alignedPositions = ref\(!solarSession\.realPositions\)/)
  assert.match(solarSource, /solarSession\.realPositions = !alignedPositions\.value/)
})
