import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

test('太阳系在封面后预挂载，双向都只在完全变黑后揭示目标页面', async () => {
  const appSource = await readFile(new URL('../src/App.vue', import.meta.url), 'utf8')
  const solarSource = await readFile(new URL('../src/components/SolarSystem.vue', import.meta.url), 'utf8')
  const enterFlow = appSource.match(/function enterSolarSystem\(\)[\s\S]*?\n}\n\nfunction enterSky/)?.[0] ?? ''
  const exitFlow = appSource.match(/function exitSolarSystemToCover[\s\S]*?\n}\n\n\/\*\* SKY/)?.[0] ?? ''

  assert.match(appSource, /const solarHomeEntering = ref\(false\)/)
  assert.match(appSource, /const SOLAR_HOME_ENTRY_VEIL_SECONDS = '0\.08s'/)
  assert.match(appSource, /const SOLAR_HOME_EXIT_VEIL_SECONDS = '0\.46s'/)
  assert.match(appSource, /const SOLAR_HOME_ENTRY_DWELL_MS = 0/)
  assert.match(appSource, /const SOLAR_HOME_EXIT_DWELL_MS = 120/)
  assert.match(appSource, /const SOLAR_HOME_ENTRY_SETTLE_MS = 700/)
  assert.match(appSource, /const solarHomePrewarming = ref\(surface\.value === 'cover'\)/)
  assert.match(appSource, /'for-deep': veilTarget === 'solar-system' \|\| solarHomeLeaving/)
  assert.match(enterFlow, /veilDuration\.value = reduced \? '0\.04s' : SOLAR_HOME_ENTRY_VEIL_SECONDS[\s\S]*?veilActive\.value = true/)
  assert.match(enterFlow, /solarEntryFly\.value = false[\s\S]*?veilActive\.value = true[\s\S]*?waitUntilFullBlack[\s\S]*?solarHomeEntering\.value = true[\s\S]*?solarEntryFly\.value = true[\s\S]*?await nextTick\(\)[\s\S]*?requestAnimationFrame/)
  assert.match(appSource, /async function setSurface\(nextSurface: AppSurface\) \{[\s\S]*?if \(nextSurface === 'cover'\) solarEntryFly\.value = false[\s\S]*?surface\.value = nextSurface/)
  assert.match(solarSource, /watch\([\s\S]*?\(\) => props\.playEntryFly[\s\S]*?if \(shouldPlay && !wasPlaying\) scene\?\.flyInFromDistance\(props\.flyDelay \?\? 0\)/)
  assert.match(enterFlow, /waitUntilFullBlack\(resolve, reduced \? 0 : SOLAR_HOME_ENTRY_DWELL_MS\)/)
  assert.match(enterFlow, /const solarReady = \(async \(\) => \{[\s\S]*?await setSurface\('solar-system'\)[\s\S]*?requestAnimationFrame\(\(\) => requestAnimationFrame/)
  assert.doesNotMatch(enterFlow, /const solarReady = \(async \(\) => \{[\s\S]*?await loadSolarSystem\(\)/)
  assert.match(enterFlow, /await Promise\.all\(\[solarReady, fullBlack, solarComponentReady\]\)[\s\S]*?await solarSystemRef\.value\?\.waitForTexturesReady\?\.\(\)[\s\S]*?solarEntryFly\.value = true[\s\S]*?coverLingering\.value = false[\s\S]*?veilActive\.value = false/)
  assert.match(exitFlow, /auroraCoverRef\.value\?\.resetLaunchState\(\)[\s\S]*?solarHomeLeaving\.value = true/)
  assert.match(exitFlow, /veilDuration\.value = reduced \? '0\.04s' : SOLAR_HOME_EXIT_VEIL_SECONDS/)
  assert.match(exitFlow, /veilActive\.value = true[\s\S]*?waitUntilFullBlack\(resolve, reduced \? 0 : SOLAR_HOME_EXIT_DWELL_MS\)[\s\S]*?await setSurface\('cover'\)[\s\S]*?coverHomeRevealing\.value = true[\s\S]*?veilActive\.value = false/)
  assert.match(appSource, /surface\.value === 'cover'[\s\S]*?void loadSolarSystem\(\)/)
  assert.match(appSource, /surface\.value === 'cover'[\s\S]*?solarHomePrewarming\.value = true/)
  assert.match(appSource, /v-if="surface === 'solar-system' \|\| \(surface === 'cover' && solarHomePrewarming\)"/)
  assert.doesNotMatch(appSource, /v-if="surface === 'solar-system' \|\| solarHomePrewarming"/)
  assert.match(appSource, /'home-revealing': coverHomeRevealing/)
  assert.match(appSource, /1 - Math\.pow\(1 - t, 3\)/)
  assert.doesNotMatch(appSource, /deepCoverTransitioning|deepCoverReturning|deep-transition-end/)
})

test('首页与太阳系各自完成收暗和渐亮，并提供 reduced-motion 路径', async () => {
  const coverSource = await readFile(new URL('../src/components/AuroraCover.vue', import.meta.url), 'utf8')
  const solarSource = await readFile(new URL('../src/components/SolarSystem.vue', import.meta.url), 'utf8')
  const styleSource = await readFile(new URL('../src/style.css', import.meta.url), 'utf8')

  assert.match(coverSource, /defineExpose\(\{ resetLaunchState \}\)/)
  assert.match(coverSource, /const settled = ref\(coverEntrancePlayed \|\| !props\.activeHome\)/)
  assert.match(coverSource, /function enterDeepSpace\(\)[\s\S]*?settled\.value = true[\s\S]*?launching\.value = true/)
  assert.match(coverSource, /launching\.value = true\s*\/\/[\s\S]*?emit\('explore'\)/)
  assert.doesNotMatch(coverSource, /setTimeout\(\(\) => emit\('explore'\), 0\)/)
  assert.match(coverSource, /\.aurora-cover\.is-launching::after[\s\S]*?animation: cover-exit-veil \.08s/)
  assert.match(coverSource, /\.aurora-cover\.home-revealing \.cover-earth[\s\S]*?cover-home-earth-return \.72s/)
  assert.match(coverSource, /@keyframes cover-home-content-return[\s\S]*?translateY\(-50%\) translateX\(-14px\)/)
  assert.doesNotMatch(coverSource, /--deep-wipe|cover-to-deep-space/)
  assert.match(solarSource, /\.solar-system\.home-entering \.solar-scene-host/)
  assert.match(solarSource, /solar-home-scene-enter \.7s cubic-bezier\(\.4, 0, \.2, 1\)/)
  assert.match(solarSource, /solar-home-ui-enter \.36s \.2s/)
  assert.match(solarSource, /@keyframes solar-home-scene-enter\s*\{\s*from \{ opacity: 0; \}\s*to \{ opacity: 1; \}/)
  assert.match(styleSource, /deep-header-enter \.18s \.02s/)
  assert.match(styleSource, /\.desktop-app\.deep-prewarming\s*\{[\s\S]*?position:\s*fixed;[\s\S]*?inset:\s*0;[\s\S]*?overflow:\s*hidden;/)
  assert.match(solarSource, /\.solar-system\.home-leaving \.solar-scene-host[\s\S]*?solar-home-scene-leave \.5s/)
  assert.match(solarSource, /@keyframes solar-home-scene-leave[\s\S]*?to \{ opacity: 0;/)
  assert.match(solarSource, /prefers-reduced-motion: reduce[\s\S]*?\.solar-system\.home-entering/)
})

test('太阳系示意排布复现目标近景，真实位置模式保留原有距离', async () => {
  const dataSource = await readFile(new URL('../src/solar/data.ts', import.meta.url), 'utf8')
  const sceneSource = await readFile(new URL('../src/solar/scene.ts', import.meta.url), 'utf8')

  assert.match(dataSource, /composeMinDistance: 50/)
  assert.match(dataSource, /alignedFitMargin: 0\.544/)
  assert.match(dataSource, /anchorScreenX: 0\.398/)
  assert.match(dataSource, /anchorScreenY: 0\.556/)
  assert.match(
    sceneSource,
    /const viewportRadiusDistance = KUIPER_BELT\.outer \/ \(tanHalf \* aspect\)/,
  )
  assert.match(
    sceneSource,
    /if \(this\.compositionMode === 'real'\) \{[\s\S]*?viewportRadiusDistance \* VIEW\.realFitMargin/,
  )
  assert.match(sceneSource, /viewportRadiusDistance \* VIEW\.alignedFitMargin/)
  assert.match(
    sceneSource,
    /addScaledVector\(this\.right, distance \* alpha\)[\s\S]*?addScaledVector\(this\.upv, distance \* beta\)[\s\S]*?return \{ target, distance \}/,
  )
  assert.doesNotMatch(sceneSource, /return \{ target, distance: VIEW\.composeMinDistance \}/)
  assert.match(sceneSource, /private refit\(\)[\s\S]*?computeModeComposition\(aspect\)/)
  assert.match(sceneSource, /resetView\(\)[\s\S]*?computeModeComposition\(aspect\)/)
})

test('太阳系入场用 0.7 秒完成适度远景推进与同步渐亮', async () => {
  const dataSource = await readFile(new URL('../src/solar/data.ts', import.meta.url), 'utf8')
  const sceneSource = await readFile(new URL('../src/solar/scene.ts', import.meta.url), 'utf8')

  assert.match(dataSource, /entryStartOffset: 250/)
  assert.match(sceneSource, /const p0 = target\.clone\(\)\.addScaledVector\(this\.dir, distance \+ VIEW\.entryStartOffset\)/)
  assert.match(sceneSource, /duration: window\.matchMedia\('\(prefers-reduced-motion: reduce\)'\)\.matches \? 250 : 700/)
  assert.doesNotMatch(sceneSource, /easeOut: 'linear-out'/)
  assert.doesNotMatch(sceneSource, /const p0 = target\.clone\(\)\.addScaledVector\(this\.dir, distance \* 15\)/)
})

test('首页品牌字使用本地预载字体，不再等待远程 Montserrat 后替换字形', async () => {
  const indexSource = await readFile(new URL('../index.html', import.meta.url), 'utf8')
  const styleSource = await readFile(new URL('../src/style.css', import.meta.url), 'utf8')
  const coverSource = await readFile(new URL('../src/components/AuroraCover.vue', import.meta.url), 'utf8')
  const fontBytes = await readFile(new URL('../public/fonts/aurora-montserrat-200-latin.woff2', import.meta.url))

  assert.match(indexSource, /rel="preload" href="\/fonts\/aurora-montserrat-200-latin\.woff2" as="font" type="font\/woff2" crossorigin/)
  assert.match(styleSource, /font-family: 'Aurora Wordmark';[\s\S]*?font-display: block;/)
  assert.match(coverSource, /font-family: 'Aurora Wordmark', sans-serif;/)
  assert.doesNotMatch(styleSource, /family=Montserrat|font-family:\s*Montserrat/)
  assert.ok(fontBytes.byteLength > 1_000)
})
