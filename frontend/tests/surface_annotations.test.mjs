import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import {
  layoutSceneAnnotations,
  sceneAnnotationScale,
  sceneAnnotationStyle,
  orbitMarkerRadiusPx,
  surfaceMarkerRadiusPx,
  sceneMarkerWorldRadius,
} from '../src/surfaceAnnotations.ts'

const source = (path) => readFile(new URL(path, import.meta.url), 'utf8')

const viewport = (ratio = 1) => ({ width: 1000, height: 700, currentPlanetRadiusPx: 200 * ratio, referencePlanetRadiusPx: 200 })
const anchor = (id, x = 500, y = 350) => ({ id, anchorX: x, anchorY: y, visible: true })

test('远景明显收小，点、线、标签的比例一致且近景不膨胀', () => {
  const normal = sceneAnnotationScale(200, 200)
  const far = sceneAnnotationScale(100, 200)
  assert.ok(far / normal < 0.6)
  let previousScale = 0
  for (const ratio of [0.01, 0.2, 0.5, 1, 2, 10, 100]) {
    const scale = sceneAnnotationScale(200 * ratio, 200)
    assert.ok(scale >= previousScale && scale >= 0.5 && scale <= 1.05)
    previousScale = scale
    const [layout] = layoutSceneAnnotations([anchor('a')], viewport(ratio))
    assert.equal(layout.scale, scale)
    assert.equal(orbitMarkerRadiusPx(200 * ratio, 200), 4.5 * scale)
    assert.equal(surfaceMarkerRadiusPx(200 * ratio, 200), 5 * scale)
    assert.equal(surfaceMarkerRadiusPx(200 * ratio, 200, true), 5 * scale)
    assert.ok(Math.abs(Math.abs(layout.x - layout.anchorX) - 10 * scale) < 1e-9)
  }
  assert.ok(2 * orbitMarkerRadiusPx(1, 200) >= 4.5, '最远处仍保留可见圆点')
})

test('场景标签避让页头，并在空间充足时保留完整文字', () => {
  const options = { ...viewport(), clusterOverlappingLabels: true, preferFullLabels: true, safeTopPx: 76 }
  const labels = layoutSceneAnnotations([anchor('top', 500, 82), anchor('middle', 500, 350)], options)
  assert.equal(labels[0].visible, false)
  assert.equal(labels[1].visible, true)
  assert.equal(labels[1].mode, 'full')
})

test('不同深度的飞行器换算回像素后半径一致', () => {
  for (const depth of [1, 5, 50, 500]) {
    const radiusPx = orbitMarkerRadiusPx(100, 200)
    const world = sceneMarkerWorldRadius(depth, 42, 700, radiusPx)
    const pixels = world / (2 * depth * Math.tan(42 * Math.PI / 360) / 700)
    assert.ok(Math.abs(pixels - radiusPx) < 1e-9)
  }
})

test('完全相同的两个坐标在放大后自动左右展开', () => {
  const points = [anchor('a'), anchor('b')]
  const far = layoutSceneAnnotations(points, viewport(0.5))
  assert.equal(far.filter((item) => item.visible).length, 1)
  assert.equal(far.find((item) => item.visible).clusterCount, 2)
  const near = layoutSceneAnnotations(points, viewport(1.4), far)
  assert.deepEqual(near.map((item) => [item.visible, item.mode, item.side]), [[true, 'full', 'left'], [true, 'full', 'right']])
  for (const label of near) {
    assert.equal(label.y, 350)
    assert.ok(Math.abs(Math.abs(label.x - 500) - 10 * label.scale) < 1e-9)
  }
  assert.equal(sceneAnnotationStyle(near[0]).transformOrigin, 'right center')
  assert.match(sceneAnnotationStyle(near[0]).transform, /translate\(-100%, -50%\)/)
  assert.equal(sceneAnnotationStyle(near[1]).transformOrigin, 'left center')
})

test('左右位置在选择、悬停、数据重排时保持稳定，缩放临界点不闪烁', () => {
  const points = [anchor('a'), anchor('b')]
  const expanded = layoutSceneAnnotations(points, viewport(1.4))
  for (const state of [{}, { selected: true }, { hovered: true }]) {
    const next = layoutSceneAnnotations([{ ...points[1], ...state }, points[0]], viewport(1.02), expanded)
    assert.equal(next.find((item) => item.id === 'a').side, 'left')
    assert.equal(next.find((item) => item.id === 'b').side, 'right')
    assert.ok(next.every((item) => item.visible && item.mode === 'full'))
  }
  const collapsed = layoutSceneAnnotations(points, viewport(0.9), expanded)
  assert.equal(collapsed.filter((item) => item.visible).length, 1)
  const stillCollapsed = layoutSceneAnnotations(points, viewport(1.02), collapsed)
  assert.equal(stillCollapsed.filter((item) => item.visible).length, 1)
  const crossed = layoutSceneAnnotations([anchor('a', 501), anchor('b', 499)], viewport(1.4), expanded)
  assert.equal(crossed.find((item) => item.id === 'a').side, 'left')
  assert.equal(crossed.find((item) => item.id === 'b').side, 'right')
})

test('相近但不同的两点也能展开，且各自保留真实锚点', () => {
  const points = [anchor('a', 500, 350), anchor('b', 501.2, 351.5)]
  const layouts = layoutSceneAnnotations(points, viewport(1.4))
  assert.ok(layouts.every((item) => item.visible && item.mode === 'full'))
  layouts.forEach((item, index) => {
    assert.equal(item.anchorX, points[index].anchorX)
    assert.equal(item.y, points[index].anchorY)
  })
})

test('三个同位任务的缩略图保留全部成员，选择任一项都能显示完整标签', () => {
  const points = [anchor('a'), anchor('b'), anchor('c')]
  const initial = layoutSceneAnnotations(points, viewport(1.4))
  assert.deepEqual(initial.find((item) => item.visible).memberIds, ['a', 'b', 'c'])
  for (const selected of points) {
    const next = layoutSceneAnnotations(points.map((item) => ({ ...item, selected: item.id === selected.id })), viewport(1.4), initial)
    const label = next.find((item) => item.id === selected.id)
    assert.equal(label.visible, true)
    assert.equal(label.mode, 'full')
    const remaining = new Set(next.filter((item) => item.visible).flatMap((item) => item.memberIds))
    assert.equal(remaining.size, 3, '其余成员仍可由缩略图访问')
  }
})

test('边缘空间不足时保留可点击的聚合入口，不把标签挪离坐标', () => {
  const points = [anchor('a', 18), anchor('b', 18)]
  const labels = layoutSceneAnnotations(points, viewport(1.4))
  const shown = labels.filter((item) => item.visible)
  assert.equal(shown.length, 1)
  assert.equal(shown[0].mode, 'cluster')
  assert.equal(shown[0].clusterCount, 2)
  assert.equal(shown[0].y, points[0].anchorY)
  assert.equal(layoutSceneAnnotations([{ ...anchor('back'), visible: false }], viewport(2))[0].visible, false)
})

test('月球远景按紧凑标签占用空间聚合，并用滞回边界避免临界闪烁', () => {
  const points = [anchor('a', 500, 350), anchor('b', 540, 350)]
  const defaultLayout = layoutSceneAnnotations(points, viewport(0.5))
  assert.equal(defaultLayout.filter((item) => item.visible).length, 2, '普通场景不扩大聚合范围')

  const moonViewport = { ...viewport(0.5), clusterOverlappingLabels: true }
  const clustered = layoutSceneAnnotations(points, moonViewport)
  const thumbnail = clustered.find((item) => item.visible)
  assert.equal(clustered.filter((item) => item.visible).length, 1)
  assert.equal(thumbnail.mode, 'cluster')
  assert.deepEqual(thumbnail.memberIds, ['a', 'b'])

  const held = layoutSceneAnnotations(points, { ...viewport(1.15), clusterOverlappingLabels: true }, clustered)
  assert.equal(held.filter((item) => item.visible).length, 1, '默认视距附近仍保持缩略态')
  const released = layoutSceneAnnotations(points, { ...viewport(1.25), clusterOverlappingLabels: true }, held)
  assert.equal(released.filter((item) => item.visible).length, 2)
})

test('月球着陆点避开已排布的飞行器标签', () => {
  const moonViewport = { ...viewport(1), clusterOverlappingLabels: true }
  const craft = layoutSceneAnnotations([anchor('craft', 500, 350)], moonViewport)
  const [site] = layoutSceneAnnotations([anchor('site', 510, 350)], moonViewport, [], craft)
  assert.equal(craft[0].side, 'right')
  assert.equal(site.side, 'left')
  assert.equal(site.mode, 'compact')
})

test('金星的表面及飞行器文字使用黑色描边，短线支持镜像', async () => {
  const label = await source('../src/components/MissionSceneLabel.vue')
  const scene = await source('../src/components/PlanetScene.vue')
  assert.match(label, /has-connector\.on-left::before/)
  assert.match(label, /-webkit-text-stroke: var\(--mission-label-text-stroke/)
  assert.match(label, /paint-order: stroke fill/)
  assert.match(scene, /\.planet-section\.venus\s*\{[^}]*--mission-label-text-stroke: 1\.5px #000/s)
})

test('地球、月球、火星与通用行星共用轨道及表面标注契约', async () => {
  const scenes = await Promise.all([
    source('../src/components/OrbitScene.vue'),
    source('../src/components/MoonScene.vue'),
    source('../src/components/MarsScene.vue'),
    source('../src/components/PlanetScene.vue'),
  ])

  for (const scene of scenes) {
    assert.match(scene, /layoutSceneAnnotations/)
    assert.match(scene, /projectedSphereRadiusPx/)
    assert.match(scene, /orbitMarkerRadiusPx/)
    assert.match(scene, /sceneMarkerWorldRadius/)
    assert.match(scene, /surfaceMarkerRadiusPx/)
    assert.match(scene, /surfaceMarkerWorldRadius/)
    assert.match(scene, /:mode="label\.mode"/)
    assert.match(scene, /cluster-count/)
    assert.match(scene, /:side="label\.side"/)
    assert.match(scene, /:cluster-items=/)
    assert.match(scene, /@select-member=/)
    assert.doesNotMatch(scene, /SurfaceLeaderLayer|missionMarkerScale/)
  }

  assert.match(scenes[0], /new THREE\.SphereGeometry\(1, 16, 16\)/)
  assert.match(scenes[0], /new THREE\.SphereGeometry\(3\.4, 8, 8\)/)
  assert.match(scenes[1], /new THREE\.SphereGeometry\(5\.5, 8, 8\)/)
  assert.match(scenes[2], /new THREE\.SphereGeometry\(5\.5, 8, 8\)/)
  assert.match(scenes[3], /new THREE\.SphereGeometry\(4, 10, 10\)/)
})

test('地球表面坐标仍落在真实半径而非悬浮偏移', async () => {
  const scene = await source('../src/components/OrbitScene.vue')

  assert.match(scene, /latLonToVector\(site\.latitude, site\.longitude, EARTH_RADIUS\)/)
  assert.doesNotMatch(scene, /EARTH_RADIUS \* 1\.006/)
  assert.match(scene, /props\.observerTarget\.longitude,\s*EARTH_RADIUS,/)
  assert.doesNotMatch(scene, /EARTH_RADIUS \* 1\.008/)
})
