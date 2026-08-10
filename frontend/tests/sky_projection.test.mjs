import { test } from 'node:test'
import assert from 'node:assert/strict'
import { projectToSky, trackToPath, SKY_PROJECTION } from '../src/skyProjection.ts'

test('projectToSky 天顶在顶部、地平线在底部', () => {
  const zenith = projectToSky(180, 90, 180)
  assert.ok(zenith.visible)
  assert.ok(zenith.y < 20, `天顶应在画面顶部（实际 y=${zenith.y.toFixed(0)}）`)
  const horizon = projectToSky(180, 0, 180)
  assert.ok(Math.abs(horizon.y - SKY_PROJECTION.horizonY) < 1, `地平线应在 ${SKY_PROJECTION.horizonY} 附近（实际 ${horizon.y.toFixed(1)}）`)
  assert.ok(Math.abs(horizon.x - 50) < 1, '正前方地平线应在画面水平中央')
})

test('projectToSky 天体高度角沿弧线运动，且中心不高于同方位更高弧线', () => {
  // 同一方位（视场中心）上，57° 天体的 y 必须低于（数值更大）60° 弧线的 y
  const body57 = projectToSky(180, 57, 180)
  const line60 = projectToSky(180, 60, 180)
  assert.ok(body57.visible && line60.visible)
  assert.ok(body57.y > line60.y, `57° 天体(y=${body57.y.toFixed(1)})应在 60° 弧线(y=${line60.y.toFixed(1)})下方`)
  // 高度越高 y 越小（越靠上）
  const body30 = projectToSky(180, 30, 180)
  assert.ok(body30.y > body57.y, '30° 应在 57° 下方')
})

test('projectToSky 等高度线呈穹顶弧线（两端高于中心）', () => {
  const center = projectToSky(180, 30, 180)
  const left = projectToSky(150, 30, 180) // 向左 30°
  const right = projectToSky(210, 30, 180)
  assert.ok(center.visible && left.visible && right.visible)
  assert.ok(left.y < center.y && right.y < center.y,
    `同高度 30° 在左右边缘应更高（中心 ${center.y.toFixed(0)}，左 ${left.y.toFixed(0)}，右 ${right.y.toFixed(0)}）——穹顶透视`)
  assert.ok(Math.abs(left.x - (100 - right.x)) < 0.1, `左右应对称（左 ${left.x.toFixed(1)}，右 ${right.x.toFixed(1)}）`)
})

test('projectToSky 观察方位旋转改变画面内容', () => {
  const north = projectToSky(0, 30, 0) // 看向北，天体在北
  const south = projectToSky(0, 30, 180) // 看向南，天体在北（身后）
  assert.ok(north.visible)
  assert.ok(!south.visible, '看向南时北方天体应在视野外')
})

test('trackToPath 生成轨迹 SVG path', () => {
  const path = trackToPath([{ az: 120, alt: 5 }, { az: 140, alt: 30 }, { az: 160, alt: 45 }], 180)
  assert.ok(path.startsWith('M '), `应以 M 开头（实际：${path.slice(0, 20)}）`)
  assert.ok(path.includes(' L '), '采样点之间应以 L 连接')
})
