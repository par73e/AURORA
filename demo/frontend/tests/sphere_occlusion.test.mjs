// 球面遮挡判据的回归守卫。
//
// 背景：土卫六（TitanLandingScene）原先用
//   `normalize(point)·normalize(camera.position) > 0.12`
// 这种「法线点积 > 固定阈值」来判可见性。真实可见边界是 n·û = R/d，
// 随相机距离变化：默认距离 5.2 时阈值 0.308，拉到最近 2.3 时是 0.696。
// 写死 0.12 只在某一个特定距离上勉强成立，拉近镜头后会把背面的圆点误判为可见 ——
// 表现为「圆点被球体挡住、标签却还浮在上面」。
//
// 这里同时用「显式几何用例」和「与 THREE.Ray/Sphere 独立实现对拍」两种方式钉住边界。
import test from 'node:test'
import assert from 'node:assert/strict'
import * as THREE from 'three'
import { isOccludedBySphere } from '../src/sphereOcclusion.ts'

const ORIGIN = new THREE.Vector3(0, 0, 0)
/** 土卫六在场景里的相对半径（与 TitanLandingScene 的 RADIUS 同量级）。 */
const R = 1.6

/** 在 +z 轴相机下，取半径为 radius、与 +z 夹角为 theta 的点。radius=R 时 n·û 恰为 cos(theta)。 */
function surfacePoint(theta, radius = R) {
  return new THREE.Vector3(Math.sin(theta) * radius, 0, Math.cos(theta) * radius)
}

function occluded(point, distance, margin = 0) {
  return isOccludedBySphere(point, new THREE.Vector3(0, 0, distance), ORIGIN, R, margin)
}

/** 扫描球面（或给定半径的球壳），返回第一个被判为遮挡的角度（度）。 */
function shadowStartDeg(distance, { radius = R, margin = 0, step = 0.001 } = {}) {
  for (let deg = 0; deg <= 180; deg += step) {
    if (occluded(surfacePoint((deg * Math.PI) / 180, radius), distance, margin)) return deg
  }
  return Infinity
}

test('正对相机的点可见，背对相机的点被遮挡', () => {
  assert.equal(occluded(surfacePoint(0), 5.2), false, '最近点应可见')
  assert.equal(occluded(surfacePoint(Math.PI), 5.2), true, '最远点应被遮挡')
})

test('球体内部的点必须判为遮挡（射线-球体判定会漏判这种情况）', () => {
  // 点在球内时，射线最近点会越过目标点，纯射线判定会返回「可见」，
  // 必须靠单独的球内分支兜住。
  assert.equal(occluded(new THREE.Vector3(0, 0, 0), 5.2), true)
  assert.equal(occluded(new THREE.Vector3(0, 0, R * 0.5), 5.2), true)
})

test('球面上的点不得被误判为球内（浮点余量 1e-3 的作用）', () => {
  // cos²+sin² 计算的表面点距离可能略小于 R，若用 `radius + margin` 做球内判定会误伤
  for (const deg of [0, 15, 30, 45, 60, 90, 120, 150, 180]) {
    const p = surfacePoint((deg * Math.PI) / 180)
    assert.ok(p.distanceTo(ORIGIN) <= R + 1e-9)
    // 恰好正对相机的表面点必须可见
    if (deg === 0) assert.equal(occluded(p, 5.2), false)
  }
})

test('可见边界随相机距离移动：n·û = R/d（土卫六固定阈值 0.12 的回归守卫）', () => {
  // n·û = 0.5 的点：在默认距离 5.2 时可见（边界 0.308），拉到最近 2.3 时被遮挡（边界 0.696）。
  // 旧的固定阈值 0.12 在两个距离上都会判为可见 —— 这正是土卫六标签穿模的成因。
  const p = surfacePoint(Math.PI / 3) // cos(60°) = 0.5
  assert.ok(Math.abs(p.clone().normalize().dot(new THREE.Vector3(0, 0, 1)) - 0.5) < 1e-12)

  assert.equal(occluded(p, 5.2), false, 'd=5.2 时边界 0.308 < 0.5，应可见')
  assert.equal(occluded(p, 2.3), true, 'd=2.3 时边界 0.696 > 0.5，应被遮挡')
  assert.ok(R / 2.3 > 0.5 && R / 5.2 < 0.5, '边界确实跨越了 n·û = 0.5')
})

test('边界位置与 R/d 一致：稍微偏内被遮挡，稍微偏外可见', () => {
  for (const d of [2.3, 3.4, 5.2, 8.5]) {
    const boundary = Math.acos(R / d)
    const eps = 1e-3
    assert.equal(occluded(surfacePoint(boundary + eps), d), true, `d=${d} 边界内侧应被遮挡`)
    assert.equal(occluded(surfacePoint(boundary - eps), d), false, `d=${d} 边界外侧应可见`)
  }
})

test('margin 只放宽掠射带：把球外标记的遮挡影锥略微加宽', () => {
  // margin 是「标记自身的圆点半径」：它把判定球放大，让贴着轮廓的标记更早被隐藏。
  // 火星用它容纳圆点半径 0.04；这里的 r_c = 3.0 对应火星飞行器（远在球外）。
  const d = 5.2
  const CRAFT_RADIUS = 3.0

  const base = shadowStartDeg(d, { radius: CRAFT_RADIUS, margin: 0 })
  const wide = shadowStartDeg(d, { radius: CRAFT_RADIUS, margin: 0.04 })

  assert.ok(Number.isFinite(base) && Number.isFinite(wide), '球外标记应存在遮挡边界')
  assert.ok(wide < base, `margin 应把影锥加宽（${base.toFixed(3)}° -> ${wide.toFixed(3)}°）`)
  assert.ok(base - wide < 2, '加宽幅度应是一个贴边窄带，而不是把大半个球面算成遮挡')

  // 单调性：加了 margin 之后，不会有任何已遮挡的点变回可见
  for (let deg = 0; deg <= 180; deg += 1) {
    const p = surfacePoint((deg * Math.PI) / 180, CRAFT_RADIUS)
    if (occluded(p, d, 0)) assert.equal(occluded(p, d, 0.04), true, `${deg}° 不应因 margin 变可见`)
  }

  // 正面仍可见、背面仍遮挡
  assert.equal(occluded(surfacePoint(0, CRAFT_RADIUS), d, 0.04), false)
  assert.equal(occluded(surfacePoint(Math.PI, CRAFT_RADIUS), d, 0.04), true)
})

test('margin 对球面上的点是空操作（火星着陆点依赖这一点）', () => {
  // 解析结论：r_c == R 时「最近点距离 = R + margin」无实数解，放大判定球不会改变任何
  // 表面点的可见性。火星着陆点在表面上（r == MARS_RADIUS），正是靠这个性质才不会被
  // 0.04 的余量整片误隐藏 —— MarsScene 的注释依赖该结论，这里把它钉住。
  const d = 5.2
  for (const deg of [0, 30, 60, 71, 72.1, 73, 90, 120, 180]) {
    const p = surfacePoint((deg * Math.PI) / 180)
    assert.equal(
      occluded(p, d, 0.04),
      occluded(p, d, 0),
      `${deg}° 在 margin=0.04 与 margin=0 下应一致`,
    )
  }
  assert.equal(shadowStartDeg(d, { margin: 0.04 }), shadowStartDeg(d, { margin: 0 }))
})

test('球体不在相机与目标点之间时不遮挡（退化与顺序守卫）', () => {
  const cam = new THREE.Vector3(0, 0, 5)
  // 目标点在球体前方（相机与球心之间）：球体在这段视线之外
  assert.equal(isOccludedBySphere(new THREE.Vector3(0, 0, 3), cam, ORIGIN, R), false)
  // 相机恰好在目标点上：方向退化，不应判为遮挡
  const p = new THREE.Vector3(0, 0, 3)
  assert.equal(isOccludedBySphere(p, p.clone(), ORIGIN, R), false)
  // 球心在相机背后
  assert.equal(isOccludedBySphere(new THREE.Vector3(0, 0, -3), new THREE.Vector3(0, 0, 5), new THREE.Vector3(0, 0, 9), R), false)
})

test('与 THREE.Ray/Sphere 独立实现对拍（确定性随机采样）', () => {
  // 线性同余伪随机，保证可复现
  let seed = 20261001
  const rand = () => ((seed = (seed * 1103515245 + 12345) & 0x7fffffff) / 0x7fffffff)

  /**
   * 独立参考实现：射线从相机指向目标，若首个交点比目标点更近即遮挡。
   *
   * 该参考只在目标点位于判定球之外时与共享判据等价。目标点落在判定球内部时，
   * 纯射线判定会把目标自身的圆点当成遮挡物，得出「自己被自己挡住」的结论；
   * 共享判据用 `t >= targetDistance` 短路规避了这一点，因此采样必须取
   * r_c > R + margin 的球外点（这也是该判据在火星飞行器上的真实用法）。
   */
  function referenceOccluded(point, camera, center, radius, margin) {
    const occluder = radius + margin
    const toTarget = point.clone().sub(camera)
    const len = toTarget.length()
    if (len === 0) return false
    toTarget.divideScalar(len)
    const hit = new THREE.Ray(camera, toTarget).intersectSphere(new THREE.Sphere(center, occluder), new THREE.Vector3())
    return hit != null && hit.distanceTo(camera) < len - 1e-6
  }

  let checked = 0
  for (let i = 0; i < 2000; i += 1) {
    const margin = rand() < 0.5 ? 0 : 0.04
    // 采样球外点：从判定球外一点点，到 4 倍半径
    const radius = R + margin + 1e-3 + rand() * (3 * R)
    const theta = Math.acos(2 * rand() - 1)
    const phi = rand() * Math.PI * 2
    const point = new THREE.Vector3(
      Math.sin(theta) * Math.cos(phi) * radius,
      Math.sin(theta) * Math.sin(phi) * radius,
      Math.cos(theta) * radius,
    )
    const distance = R + margin + rand() * 12 // 保证相机在判定球外
    const camera = new THREE.Vector3(0, 0, distance)

    assert.equal(
      isOccludedBySphere(point, camera, ORIGIN, R, margin),
      referenceOccluded(point, camera, ORIGIN, R, margin),
      `采样不一致：d=${distance} margin=${margin} r=${radius} point=(${point.toArray().join(', ')})`,
    )
    checked += 1
  }
  assert.equal(checked, 2000)
})
