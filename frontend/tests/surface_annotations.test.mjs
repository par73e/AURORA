import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

const source = (path) => readFile(new URL(path, import.meta.url), 'utf8')

test('圆点、固定短线与标签共享一套克制的视觉缩放', async () => {
  const [annotations, label] = await Promise.all([
    source('../src/surfaceAnnotations.ts'),
    source('../src/components/MissionSceneLabel.vue'),
  ])

  assert.match(annotations, /Math\.pow\(ratio, 0\.28\), 0\.75, 1\.05/)
  assert.match(annotations, /baseRadiusPx \* sceneAnnotationScale/)
  assert.match(annotations, /export function orbitMarkerRadiusPx/)
  assert.match(annotations, /export function surfaceMarkerRadiusPx/)
  assert.match(annotations, /markerRadiusPx \* \(\(2 \* markerDistance \* Math\.tan\(halfFov\)\) \/ viewportHeight\)/)
  assert.doesNotMatch(annotations, /activeMultiplier|selected \? 1\./)
  assert.match(label, /\.mission-scene-label\.has-connector::before/)
  assert.match(label, /width: 10px/)
  assert.doesNotMatch(label, /is-compact::before \{ display: none/)
})

test('标签只切换 full compact cluster，不再移动锚点或生成动态引线', async () => {
  const annotations = await source('../src/surfaceAnnotations.ts')

  assert.match(annotations, /'full' \| 'compact' \| 'cluster'/)
  assert.match(annotations, /CLUSTER_ENTER_PX = 16/)
  assert.match(annotations, /CLUSTER_EXIT_PX = 22/)
  assert.match(annotations, /x = item\.anchorX \+ CONNECTOR_LENGTH_PX \* scale/)
  assert.match(annotations, /a\.selected \|\| a\.hovered/)
  assert.doesNotMatch(annotations, /leaderX|leaderY|side: 'left'/)
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
