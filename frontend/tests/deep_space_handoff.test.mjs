import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

test('太阳系在封面后预挂载，双向都只在完全变黑后揭示目标页面', async () => {
  const appSource = await readFile(new URL('../src/App.vue', import.meta.url), 'utf8')
  const enterFlow = appSource.match(/function enterSolarSystem\(\)[\s\S]*?\n}\n\nfunction enterSky/)?.[0] ?? ''
  const exitFlow = appSource.match(/function exitSolarSystemToCover[\s\S]*?\n}\n\n\/\*\* SKY/)?.[0] ?? ''

  assert.match(appSource, /const solarHomeEntering = ref\(false\)/)
  assert.match(appSource, /const SOLAR_HOME_ENTRY_VEIL_SECONDS = '0\.32s'/)
  assert.match(appSource, /const SOLAR_HOME_EXIT_VEIL_SECONDS = '0\.46s'/)
  assert.match(appSource, /const SOLAR_HOME_ENTRY_DWELL_MS = 0/)
  assert.match(appSource, /const SOLAR_HOME_EXIT_DWELL_MS = 120/)
  assert.match(appSource, /const SOLAR_HOME_ENTRY_SETTLE_MS = 620/)
  assert.match(appSource, /const solarHomePrewarming = ref\(surface\.value === 'cover'\)/)
  assert.match(appSource, /'for-deep': solarHomeEntering \|\| solarHomeLeaving/)
  assert.match(enterFlow, /veilDuration\.value = reduced \? '0\.04s' : SOLAR_HOME_ENTRY_VEIL_SECONDS[\s\S]*?veilActive\.value = true/)
  assert.match(enterFlow, /waitUntilFullBlack\(resolve, reduced \? 0 : SOLAR_HOME_ENTRY_DWELL_MS\)/)
  assert.match(enterFlow, /const solarReady = \(async \(\) => \{[\s\S]*?await loadSolarSystem\(\)[\s\S]*?await setSurface\('solar-system'\)[\s\S]*?requestAnimationFrame\(\(\) => requestAnimationFrame/)
  assert.match(enterFlow, /await Promise\.all\(\[solarReady, fullBlack\]\)[\s\S]*?coverLingering\.value = false[\s\S]*?veilActive\.value = false/)
  assert.match(exitFlow, /auroraCoverRef\.value\?\.resetLaunchState\(\)[\s\S]*?solarHomeLeaving\.value = true/)
  assert.match(exitFlow, /veilDuration\.value = reduced \? '0\.04s' : SOLAR_HOME_EXIT_VEIL_SECONDS/)
  assert.match(exitFlow, /veilActive\.value = true[\s\S]*?waitUntilFullBlack\(resolve, reduced \? 0 : SOLAR_HOME_EXIT_DWELL_MS\)[\s\S]*?await setSurface\('cover'\)[\s\S]*?coverHomeRevealing\.value = true[\s\S]*?veilActive\.value = false/)
  assert.match(appSource, /surface\.value === 'cover'[\s\S]*?void loadSolarSystem\(\)/)
  assert.match(appSource, /surface\.value === 'cover'[\s\S]*?solarHomePrewarming\.value = true/)
  assert.match(appSource, /v-if="surface === 'solar-system' \|\| solarHomePrewarming"/)
  assert.match(appSource, /'home-revealing': coverHomeRevealing/)
  assert.match(appSource, /1 - Math\.pow\(1 - t, 3\)/)
  assert.doesNotMatch(appSource, /deepCoverTransitioning|deepCoverReturning|deep-transition-end/)
})

test('首页与太阳系各自完成收暗和渐亮，并提供 reduced-motion 路径', async () => {
  const coverSource = await readFile(new URL('../src/components/AuroraCover.vue', import.meta.url), 'utf8')
  const solarSource = await readFile(new URL('../src/components/SolarSystem.vue', import.meta.url), 'utf8')

  assert.match(coverSource, /defineExpose\(\{ resetLaunchState \}\)/)
  assert.match(coverSource, /const settled = ref\(coverEntrancePlayed \|\| !props\.activeHome\)/)
  assert.match(coverSource, /function enterDeepSpace\(\)[\s\S]*?settled\.value = true[\s\S]*?launching\.value = true/)
  assert.match(coverSource, /launching\.value = true\s*\/\/[\s\S]*?emit\('explore'\)/)
  assert.doesNotMatch(coverSource, /setTimeout\(\(\) => emit\('explore'\), 0\)/)
  assert.match(coverSource, /\.aurora-cover\.is-launching::after[\s\S]*?animation: cover-exit-veil \.32s/)
  assert.match(coverSource, /\.aurora-cover\.home-revealing \.cover-earth[\s\S]*?cover-home-earth-return \.72s/)
  assert.match(coverSource, /@keyframes cover-home-content-return[\s\S]*?translateY\(-50%\) translateX\(-14px\)/)
  assert.doesNotMatch(coverSource, /--deep-wipe|cover-to-deep-space/)
  assert.match(solarSource, /\.solar-system\.home-entering \.solar-scene-host/)
  assert.match(solarSource, /solar-home-scene-enter \.58s/)
  assert.match(solarSource, /\.solar-system\.home-leaving \.solar-scene-host[\s\S]*?solar-home-scene-leave \.5s/)
  assert.match(solarSource, /@keyframes solar-home-scene-leave[\s\S]*?to \{ opacity: 0;/)
  assert.match(solarSource, /prefers-reduced-motion: reduce[\s\S]*?\.solar-system\.home-entering/)
})
