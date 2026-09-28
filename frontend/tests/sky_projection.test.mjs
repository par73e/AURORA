import { test } from 'node:test'
import assert from 'node:assert/strict'
import { projectAltitudeGuide, projectHorizontalDirection, projectSkyTrajectoryBranch } from '../src/skyProjection.ts'

const baseCamera = { heading: 180, horizontalFov: 120 }

function near(actual, expected, tolerance = 1e-10, message = '') {
  assert.ok(Math.abs(actual - expected) <= tolerance, `${message}：${actual} ≉ ${expected}`)
}

test('地平线抬离底边，30° / 60° 保持在接近三等分的区域', () => {
  const horizon = projectHorizontalDirection(baseCamera.heading, 0, baseCamera)
  const lowerThird = projectHorizontalDirection(baseCamera.heading, 30, baseCamera)
  const upperThird = projectHorizontalDirection(baseCamera.heading, 60, baseCamera)
  const zenith = projectHorizontalDirection(baseCamera.heading, 90, baseCamera)

  near(horizon.y, .88, 1e-10, '0° 应位于山体之上的低空基准')
  assert.ok(lowerThird.y > .60 && lowerThird.y < .64, `30° 应接近并略高于下三等分区域：${lowerThird.y}`)
  assert.ok(upperThird.y > .31 && upperThird.y < .35, `60° 应接近并略高于上三等分区域：${upperThird.y}`)
  assert.ok(lowerThird.y - upperThird.y > .30 && lowerThird.y - upperThird.y < .33, '30° / 60° 间距应适度展开但不过分夸张')
  near(zenith.y, .04, 1e-10, '90° 应稍离开画框顶边')
})

test('同一高度角在视野两侧形成轻微、对称的向上弧度', () => {
  for (const altitude of [15, 30, 60, 82.5]) {
    const center = projectHorizontalDirection(baseCamera.heading, altitude, baseCamera)
    const left = projectHorizontalDirection(baseCamera.heading - 60, altitude, baseCamera)
    const right = projectHorizontalDirection(baseCamera.heading + 60, altitude, baseCamera)
    near(left.y, right.y, 1e-10, `${altitude}° 弧线应左右对称`)
    near(center.y - left.y, .028, 1e-10, `${altitude}° 边缘上提幅度`)
  }
})

test('任意固定高度的天体在旋转罗盘时始终落在同一恒高度参考线上', () => {
  for (const altitude of [0, 12.7, 30, 37.4, 60, 72.1]) {
    for (const heading of [0, 45, 90, 180, 270, 315]) {
      const camera = { ...baseCamera, heading }
      const guide = projectAltitudeGuide(altitude, camera, 1)
      for (const relative of [-50, -25, 0, 25, 50]) {
        const body = projectHorizontalDirection(heading + relative, altitude, camera)
        const guidePoint = guide.points.find((point) => Math.abs(point.relativeAzimuth - relative) < 1e-8)
        assert.ok(guidePoint, `应采样到相对方位 ${relative}°`)
        near(body.x, guidePoint.x, 1e-10, `${altitude}° 天体横坐标`)
        near(body.y, guidePoint.y, 1e-10, `${altitude}° 天体纵坐标`)
      }
    }
  }
})

test('改变相机朝向只改变屏幕投影，不改变输入的真实高度角', () => {
  const altitude = 30
  const azimuth = 132
  const positions = [0, 45, 90, 135, 180, 270].map((heading) =>
    projectHorizontalDirection(azimuth, altitude, { ...baseCamera, heading }))
  assert.ok(positions.some((point, index) => index > 0 && point.inFront && point.x !== positions[0].x))
  assert.equal(altitude, 30)
})

test('地平线下方和水平视野外的天体不会进入当前画面', () => {
  assert.equal(projectHorizontalDirection(baseCamera.heading, -1, baseCamera).inViewport, false)
  assert.equal(projectHorizontalDirection(baseCamera.heading + 61, 30, baseCamera).inViewport, false)
  assert.equal(projectHorizontalDirection(baseCamera.heading + 60, 30, baseCamera).inViewport, true)
})

test('星轨跨过稀疏采样点时仍连续画到视野边界', () => {
  const path = projectSkyTrajectoryBranch([
    { azimuth: 180, altitude: 30 },
    { azimuth: 220, altitude: 30 },
    { azimuth: 250, altitude: 30 },
    { azimuth: 280, altitude: 30 },
  ], baseCamera)
  assert.ok(path)
  assert.match(path, /^M 500\.00 /)
  assert.match(path, /833\.33 /)
  assert.match(path, / L 1000\.00 /)
  assert.doesNotMatch(path, /L 1083\.33 /)
})

test('星轨落到地平线下时精确结束在地平线，后续时段不接到下一次升起', () => {
  const path = projectSkyTrajectoryBranch([
    { azimuth: 180, altitude: 20 },
    { azimuth: 190, altitude: -20 },
    { azimuth: 200, altitude: -30 },
    { azimuth: 210, altitude: 20 },
  ], baseCamera)
  assert.ok(path)
  const horizon = projectHorizontalDirection(185, 0, baseCamera)
  assert.ok(path.endsWith(`L ${(horizon.x * 1000).toFixed(2)} ${(horizon.y * 1000).toFixed(2)}`))
  assert.doesNotMatch(path, /750\.00/)
})

test('当前天体不可见时不绘制其他时段的轨迹', () => {
  const belowHorizon = projectSkyTrajectoryBranch([
    { azimuth: 180, altitude: -20 },
    { azimuth: 185, altitude: -10 },
    { azimuth: 190, altitude: 10 },
    { azimuth: 200, altitude: 25 },
  ], baseCamera)
  const outsideView = projectSkyTrajectoryBranch([
    { azimuth: 250, altitude: 25 },
    { azimuth: 240, altitude: 25 },
    { azimuth: 230, altitude: 25 },
    { azimuth: 220, altitude: 25 },
  ], baseCamera)

  assert.equal(belowHorizon, null)
  assert.equal(outsideView, null)
})
