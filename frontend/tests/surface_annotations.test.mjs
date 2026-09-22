import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

const source = (path) => readFile(new URL(path, import.meta.url), 'utf8')

test('表面标注按行星屏幕半径统一缩放并保留可读上下限', async () => {
  const annotations = await source('../src/surfaceAnnotations.ts')

  assert.match(annotations, /Math\.pow\(ratio, 0\.55\)/)
  assert.match(annotations, /selected \? 0\.88 : 0\.72, 1\.08/)
  assert.match(annotations, /clamp\(5\.5 \* Math\.pow\(ratio, 0\.55\) \* activeMultiplier, 3, 7\)/)
  assert.match(annotations, /markerRadiusPx \* \(\(2 \* markerDistance \* Math\.tan\(halfFov\)\) \/ viewportHeight\)/)
})

test('标签避让与真实锚点分离，引线始终从锚点连接标签边缘', async () => {
  const [annotations, leaders, label] = await Promise.all([
    source('../src/surfaceAnnotations.ts'),
    source('../src/components/SurfaceLeaderLayer.vue'),
    source('../src/components/MissionSceneLabel.vue'),
  ])

  assert.match(annotations, /anchorX: number/)
  assert.match(annotations, /leaderX: number/)
  assert.match(annotations, /leaderY: clamp\(item\.anchorY/)
  assert.match(leaders, /:x1="label\.anchorX"/)
  assert.match(leaders, /:y1="label\.anchorY"/)
  assert.match(leaders, /:x2="label\.leaderX"/)
  assert.match(leaders, /:y2="label\.leaderY"/)
  assert.match(label, /\.mission-scene-label\.is-spacecraft::before/)
  assert.doesNotMatch(label, /\.mission-scene-label::before/)
})

test('地球、月球、火星与通用行星共用表面标注契约', async () => {
  const scenes = await Promise.all([
    source('../src/components/OrbitScene.vue'),
    source('../src/components/MoonScene.vue'),
    source('../src/components/MarsScene.vue'),
    source('../src/components/PlanetScene.vue'),
  ])

  for (const scene of scenes) {
    assert.match(scene, /layoutSurfaceAnnotations/)
    assert.match(scene, /projectedSphereRadiusPx/)
    assert.match(scene, /surfaceMarkerRadiusPx/)
    assert.match(scene, /surfaceMarkerWorldRadius/)
    assert.match(scene, /<SurfaceLeaderLayer/)
  }

  assert.match(scenes[0], /latLonToVector\(site\.latitude, site\.longitude, EARTH_RADIUS\)/)
  assert.doesNotMatch(scenes[0], /EARTH_RADIUS \* 1\.006/)
  assert.match(scenes[0], /props\.observerTarget\.longitude,\s*EARTH_RADIUS,/)
  assert.doesNotMatch(scenes[0], /EARTH_RADIUS \* 1\.008/)
  assert.match(scenes[0], /kind: 'observer'/)
  assert.match(scenes[0], /:labels="surfaceLabels"/)
  assert.match(scenes[0], /new THREE\.SphereGeometry\(3\.4, 8, 8\)/)
  assert.match(scenes[1], /new THREE\.SphereGeometry\(5, 8, 8\)/)
  assert.match(scenes[2], /new THREE\.SphereGeometry\(5, 8, 8\)/)
  assert.match(scenes[3], /new THREE\.SphereGeometry\(7, 8, 8\)/)
  assert.doesNotMatch(scenes[1], /siteLabelOffsets/)
  assert.doesNotMatch(scenes[2], /siteLabelOffsets/)
})
