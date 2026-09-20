import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

test('天体退出按附属元素、天体本体、切页黑场三段衔接', async () => {
  const appSource = await readFile(new URL('../src/App.vue', import.meta.url), 'utf8')

  assert.match(appSource, /const CELESTIAL_EXIT_SEQUENCE_MS = 640/)
  assert.match(appSource, /const CELESTIAL_EXIT_REDUCED_MS = 160/)
  assert.match(appSource, /const CELESTIAL_EXIT_VEIL_SECONDS = '0\.16s'/)
  assert.equal(
    (appSource.match(/CELESTIAL_EXIT_REDUCED_MS : CELESTIAL_EXIT_SEQUENCE_MS/g) ?? []).length,
    5,
    '地球、月球、火星、通用行星和返回封面都应等待完整的两拍退出动画',
  )
})

test('每类天体场景都在附属元素退场后延迟渐隐本体', async () => {
  for (const component of ['OrbitScene.vue', 'MoonScene.vue', 'MarsScene.vue', 'PlanetScene.vue']) {
    const source = await readFile(new URL(`../src/components/${component}`, import.meta.url), 'utf8')
    assert.match(source, /'leaving-body': leaving/, `${component} 应接收退出本体状态`)
    assert.match(
      source,
      /\.revealed\.leaving-body\s*\{[\s\S]*?transition: opacity \.32s cubic-bezier\(\.4, 0, 1, 1\) \.3s;/,
      `${component} 应在 300ms 附属元素退场后渐隐天体本体`,
    )
  }
})

test('通用行星的轨道、飞行器与表面标记使用同一退出透明度', async () => {
  const source = await readFile(new URL('../src/components/PlanetScene.vue', import.meta.url), 'utf8')

  assert.match(source, /function exitElementsOpacity/)
  assert.match(source, /dotMat\.opacity = 0\.96 \* elementsOpacity/)
  assert.match(source, /lineMat\.opacity = \(activeId === runtime\.spec\.id \? LINE_ACTIVE_OPACITY : LINE_BASE_OPACITY\) \* elementsOpacity/)
  assert.match(source, /markerMat\.opacity = 0\.95 \* elementsOpacity/)
})

test('地球直接进入时仍优先执行飞行器退出淡出', async () => {
  const orbitSource = await readFile(new URL('../src/components/OrbitScene.vue', import.meta.url), 'utf8')
  const styleSource = await readFile(new URL('../src/style.css', import.meta.url), 'utf8')
  const fadeFunction = orbitSource.match(/function elementsFadeNow[\s\S]*?\n}/)?.[0] ?? ''
  const leavingGuard = fadeFunction.indexOf('if (props.leaving)')
  const directLoadGuard = fadeFunction.indexOf('if (spinReduced || revealTickAt === 0) return 1')

  assert.ok(leavingGuard >= 0 && leavingGuard < directLoadGuard, '退出透明度必须优先于直接加载的入场短路')
  assert.match(orbitSource, /class="scene-spacecraft-label"/)
  assert.match(styleSource, /\.orbit-section\.leaving \.scene-spacecraft-label,[\s\S]*?opacity: 0;/)
})
