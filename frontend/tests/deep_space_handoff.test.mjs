import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

test('首页与太阳系使用双端参与的对称空间交接', async () => {
  const appSource = await readFile(new URL('../src/App.vue', import.meta.url), 'utf8')

  assert.match(appSource, /const deepCoverTransitioning = ref\(false\)/)
  assert.match(appSource, /const solarHomeEntering = ref\(false\)/)
  assert.match(appSource, /function exitSolarSystemToCover/)
  assert.match(appSource, /deepCoverReturning\.value = true/)
  assert.match(appSource, /solarHomeLeaving\.value = true/)
  assert.match(appSource, /@deep-transition-end="onDeepCoverTransitionEnd"/)
  assert.match(appSource, /waitForDeepCoverAnimation\(handoffMs \+ 240\)/)
  assert.match(appSource, /waitForDeepCoverAnimation\(coverReturnMs \+ 240\)/)
  assert.match(appSource, /auroraCoverRef\.value\?\.resetLaunchState\(\)[\s\S]*?await nextTick\(\)[\s\S]*?solarHomeLeaving\.value = true/)
  assert.doesNotMatch(appSource, /scheduleForNavigation\(generation, \(\) => \{[\s\S]{0,120}deepCoverTransitioning\.value = false[\s\S]{0,40}\}, handoffMs\)/)
  assert.doesNotMatch(
    appSource.match(/function enterSolarSystem\(\)[\s\S]*?\n}\n\nfunction enterSky/)?.[0] ?? '',
    /veilActive\.value = true/,
    '首页进入太阳系不应再经过独立黑场',
  )
})

test('深空交接复用同一斜向蒙版并提供 reduced-motion 路径', async () => {
  const coverSource = await readFile(new URL('../src/components/AuroraCover.vue', import.meta.url), 'utf8')
  const solarSource = await readFile(new URL('../src/components/SolarSystem.vue', import.meta.url), 'utf8')

  assert.match(coverSource, /@property --deep-wipe/)
  assert.match(coverSource, /\.aurora-cover\.deep-transitioning,[\s\S]*?linear-gradient\(\s*112deg/)
  assert.match(coverSource, /from \{ --deep-wipe: -24%; \}/)
  assert.match(coverSource, /to \{ --deep-wipe: 118%; \}/)
  assert.match(coverSource, /\.aurora-cover\.deep-returning[\s\S]*?animation-direction: reverse/)
  assert.match(coverSource, /if \(event\.animationName\.startsWith\('cover-to-deep-space'\)\) emit\('deepTransitionEnd'\)/)
  assert.match(coverSource, /defineExpose\(\{ resetLaunchState \}\)/)
  assert.match(coverSource, /const settled = ref\(coverEntrancePlayed \|\| !props\.activeHome\)/)
  assert.match(coverSource, /function enterDeepSpace\(\)[\s\S]*?settled\.value = true[\s\S]*?launching\.value = true/)
  assert.match(coverSource, /launching\.value = true\s*\/\/[\s\S]*?emit\('explore'\)/)
  assert.doesNotMatch(coverSource, /setTimeout\(\(\) => emit\('explore'\), 0\)/)
  assert.match(coverSource, /prefers-reduced-motion: reduce[\s\S]*?\.aurora-cover\.deep-transitioning/)
  assert.match(solarSource, /\.solar-system\.home-entering \.solar-scene-host/)
  assert.match(solarSource, /\.solar-system\.home-leaving \.solar-scene-host/)
  assert.match(solarSource, /prefers-reduced-motion: reduce[\s\S]*?\.solar-system\.home-entering/)
})
