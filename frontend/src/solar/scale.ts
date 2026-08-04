// 真实日心距离（AU）→ 场景径向单位（示意尺度）的单调映射。
// 场景的行星轨道半径是"近似等差排布"（水星 14 → 海王星 130），不是真实 AU 线性比例；
// 这里用对数空间分段线性插值穿过 8 颗行星锚点（真实 AU → 场景轨道半径），
// 尾部外推——深空探测器（如旅行者号 ~171 AU）落到柯伊伯带之外的深处，
// 语义上正好是"正在离开太阳系"。

export const AU_KM = 149_597_870.7

/** 行星锚点：[真实日心平均距离 AU, 场景轨道半径]（与 solar/data.ts 的 orbitRadius 一致）。
 *  首锚点 [0.05, 7] 为近太阳端：贴近帕克真实近日点（0.046–0.048 AU），
 *  近日弧自然弯曲而非被钳制压平（0.2 钳制会产生硬拐角）；太阳半径 5.5，轨道保持在日面外 */
const ANCHORS: ReadonlyArray<readonly [au: number, units: number]> = [
  [0.05, 7], // 近太阳端（帕克近日点量级）
  [0.387, 14], // 水星
  [0.723, 30], // 金星
  [1.0, 46], // 地球
  [1.524, 60], // 火星
  [5.203, 82], // 木星
  [9.537, 98], // 土星
  [19.19, 114], // 天王星
  [30.07, 130], // 海王星
]

const LOG_ANCHOR_X = ANCHORS.map(([au]) => Math.log10(au))
const LOG_ANCHOR_Y = ANCHORS.map(([, units]) => units)
/** 保单调切线（Fritsch–Carlson）：分段三次 Hermite 处处 C¹ 光滑——
 *  之前的对数分段线性在每个锚点有斜率断点，探测器椭圆穿过锚点时出现拐角（坑坑洼洼） */
const HERMITE_TANGENT = (() => {
  const n = LOG_ANCHOR_X.length
  const secant: number[] = []
  for (let i = 0; i < n - 1; i += 1) {
    secant.push((LOG_ANCHOR_Y[i + 1] - LOG_ANCHOR_Y[i]) / (LOG_ANCHOR_X[i + 1] - LOG_ANCHOR_X[i]))
  }
  const d: number[] = new Array(n)
  d[0] = secant[0]
  d[n - 1] = secant[n - 2]
  for (let i = 1; i < n - 1; i += 1) {
    d[i] = secant[i - 1] * secant[i] <= 0 ? 0 : 2 / (1 / secant[i - 1] + 1 / secant[i])
  }
  return d
})()

/** 真实日心距离（AU）→ 场景径向单位：对数空间保单调三次 Hermite 插值 + 首尾线性外推。
 *  曲线处处 C¹ 光滑（无锚点斜率断点）；输入钳制到首锚点（0.05 AU），映射半径 ≥ 7 */
export function sceneRadiusFromAU(au: number): number {
  const log = Math.log10(Math.max(au, ANCHORS[0][0]))
  const firstX = LOG_ANCHOR_X[0]
  const last = LOG_ANCHOR_X.length - 1
  if (log <= firstX) {
    // 低于首锚点（0.05 AU）：按首段斜率线性外推（输入已被钳制，通常不触发）
    return LOG_ANCHOR_Y[0] + (log - firstX) * HERMITE_TANGENT[0]
  }
  if (log >= LOG_ANCHOR_X[last]) {
    // 超出海王星（旅行者等）：按末段斜率线性外推
    return LOG_ANCHOR_Y[last] + (log - LOG_ANCHOR_X[last]) * HERMITE_TANGENT[last]
  }
  let k = 0
  while (k < last - 1 && log > LOG_ANCHOR_X[k + 1]) k += 1
  const x0 = LOG_ANCHOR_X[k]
  const x1 = LOG_ANCHOR_X[k + 1]
  const h = x1 - x0
  const t = (log - x0) / h
  const t2 = t * t
  const t3 = t2 * t
  // 三次 Hermite 基函数
  const h00 = 2 * t3 - 3 * t2 + 1
  const h10 = t3 - 2 * t2 + t
  const h01 = -2 * t3 + 3 * t2
  const h11 = t3 - t2
  return (
    h00 * LOG_ANCHOR_Y[k] +
    h10 * h * HERMITE_TANGENT[k] +
    h01 * LOG_ANCHOR_Y[k + 1] +
    h11 * h * HERMITE_TANGENT[k + 1]
  )
}

/** 日心黄道矢量（km，JPL Horizons 黄道 J2000）→ 场景坐标（黄道面 XZ，+x = 春分点）：
 *  方向保留真实黄经（Horizons 黄道 X → 场景 +x，Y → 场景 +z），
 *  径向距离经 sceneRadiusFromAU 压缩（与行星 real 模式同一坐标系） */
export function heliocentricToScene(xKm: number, yKm: number, zKm: number, out: { x: number; z: number }) {
  const au = Math.sqrt(xKm * xKm + yKm * yKm + zKm * zKm) / AU_KM
  const radius = sceneRadiusFromAU(au)
  const planar = Math.hypot(xKm, yKm) || 1
  out.x = (xKm / planar) * radius
  out.z = (yKm / planar) * radius
  return out
}

/** 日心距离（AU）——信息面板展示用 */
export function distanceAU(xKm: number, yKm: number, zKm: number): number {
  return Math.sqrt(xKm * xKm + yKm * yKm + zKm * zKm) / AU_KM
}

/** 拟合椭圆（太阳位于焦点）：由真实采样线性最小二乘求半长轴/偏心率/近日点方向，
 *  并按"当前真实方向"锚定近日点时刻——加载瞬间标记精确对准真实位置 */
export interface FittedEllipse {
  /** 半长轴（AU） */
  aAU: number
  /** 偏心率 */
  e: number
  /** 近日点方向角（黄道面内，与场景 +x 春分点一致） */
  perihelionAngle: number
  /** 近日点经过时刻（ms，由当前真实方向反推锚定） */
  perihelionEpochMs: number
  /** 公转周期（天，开普勒第三定律 a^1.5） */
  periodDays: number
}

/** 角度归一化到 (-π, π] */
function normPi(a: number) {
  return ((((a + Math.PI) % (Math.PI * 2)) + Math.PI * 2) % (Math.PI * 2)) - Math.PI
}

/** 角度线性插值（跨 ±π 边界走最短弧） */
function interpolateAngle(ts: number[], th: number[], nowMs: number) {
  const last = ts.length - 1
  if (nowMs <= ts[0]) return th[0]
  if (nowMs >= ts[last]) return th[last]
  let lo = 0
  let hi = last
  while (lo + 1 < hi) {
    const mid = (lo + hi) >> 1
    if (ts[mid] <= nowMs) lo = mid
    else hi = mid
  }
  const t = (nowMs - ts[lo]) / (ts[hi] - ts[lo])
  // 返回归一化角度（自包含，跨 ±π 边界走最短弧）
  return normPi(th[lo] + normPi(th[hi] - th[lo]) * t)
}

/** 从日心黄道采样拟合轨道椭圆（黄道面投影 r=hypot(x,y)、θ=atan2(y,x)）。
 *  线性最小二乘 u=1/r = A + C·cosθ + D·sinθ → p=1/A、e=p·√(C²+D²)、ω=atan2(D,C)——
 *  与 JPL osculating 根数交叉验证：帕克/太阳轨道器的近日点黄经误差 <0.1° */
export function fitEllipseFromSamples(positions: Array<{ epoch: string; x: number; y: number; z: number }>, nowMs: number): FittedEllipse | null {
  const th: number[] = []
  const invR: number[] = []
  const ts: number[] = []
  for (const s of positions) {
    const epochMs = Date.parse(s.epoch)
    const r = Math.hypot(s.x, s.y)
    // NaN 防护：坏历元/坏坐标直接跳过，避免污染拟合与锚定
    if (!Number.isFinite(epochMs) || !Number.isFinite(r) || r <= 0) continue
    th.push(Math.atan2(s.y, s.x))
    invR.push(AU_KM / r)
    ts.push(epochMs)
  }
  const n = th.length
  if (n < 5) return null

  // 法方程（3×3）高斯消元（列主元）
  let s1 = 0
  let sc = 0
  let ss = 0
  let scc = 0
  let sss = 0
  let scs = 0
  let su = 0
  let suc = 0
  let sus = 0
  for (let i = 0; i < n; i += 1) {
    const c = Math.cos(th[i])
    const s = Math.sin(th[i])
    s1 += 1
    sc += c
    ss += s
    scc += c * c
    sss += s * s
    scs += c * s
    su += invR[i]
    suc += invR[i] * c
    sus += invR[i] * s
  }
  const m = [
    [s1, sc, ss],
    [sc, scc, scs],
    [ss, scs, sss],
  ]
  const b = [su, suc, sus]
  for (let col = 0; col < 3; col += 1) {
    let piv = col
    for (let r = col + 1; r < 3; r += 1) if (Math.abs(m[r][col]) > Math.abs(m[piv][col])) piv = r
    ;[m[col], m[piv]] = [m[piv], m[col]]
    ;[b[col], b[piv]] = [b[piv], b[col]]
    const diag = m[col][col]
    if (Math.abs(diag) < 1e-12) return null
    for (let r = 0; r < 3; r += 1) {
      if (r === col || Math.abs(m[r][col]) < 1e-12) continue
      const f = m[r][col] / diag
      for (let c = col; c < 3; c += 1) m[r][c] -= f * m[col][c]
      b[r] -= f * b[col]
    }
  }
  const A = b[0] / m[0][0]
  const C = b[1] / m[1][1]
  const D = b[2] / m[2][2]
  if (!(A > 0)) return null
  const p = 1 / A
  const e = p * Math.hypot(C, D)
  if (!(e > 0) || e >= 1) return null
  const w = Math.atan2(D, C)
  const aAU = p / (1 - e * e)
  const periodDays = Math.pow(aAU, 1.5) * 365.25
  const n0 = (Math.PI * 2) / (periodDays * 86400000)

  // 锚定：当前真实方向落在椭圆上 → 反推近日点时刻（加载瞬间方向精确对准真实位置，
  // 半径取拟合值；之后按真实周期沿椭圆运行，帕克传播 44 天误差 <0.1°）
  const nu = normPi(interpolateAngle(ts, th, nowMs) - w)
  const E = 2 * Math.atan2(Math.sqrt(1 - e) * Math.sin(nu / 2), Math.sqrt(1 + e) * Math.cos(nu / 2))
  const M = E - e * Math.sin(E)
  return { aAU, e, perihelionAngle: w, perihelionEpochMs: nowMs - M / n0, periodDays }
}

/** 椭圆轨道上某时刻的位置（极坐标）：开普勒方程解真近点角，太阳位于焦点。
 *  用于探测器实时位置传播——标记严格落在拟合椭圆上 */
export function ellipsePositionAt(fit: FittedEllipse, timeMs: number): { rAU: number; theta: number } {
  const n = (Math.PI * 2) / (fit.periodDays * 86400000) // 平均角速度 rad/ms
  let M = (n * (timeMs - fit.perihelionEpochMs)) % (Math.PI * 2)
  if (M < 0) M += Math.PI * 2
  // 开普勒方程 M = E - e·sinE（牛顿迭代，e<1 快速收敛）
  let E = M
  for (let i = 0; i < 10; i += 1) {
    const next = E - (E - fit.e * Math.sin(E) - M) / (1 - fit.e * Math.cos(E))
    if (Math.abs(next - E) < 1e-9) {
      E = next
      break
    }
    E = next
  }
  const nu = 2 * Math.atan2(Math.sqrt(1 + fit.e) * Math.sin(E / 2), Math.sqrt(1 - fit.e) * Math.cos(E / 2))
  const rAU = (fit.aAU * (1 - fit.e * fit.e)) / (1 + fit.e * Math.cos(nu))
  return { rAU, theta: fit.perihelionAngle + nu }
}

/** 拟合椭圆采样 → 场景坐标点（太阳位于原点即焦点，极坐标 r=a(1-e²)/(1+e·cosν)）
 *  经光滑径向映射 + 黄经方向映射，生成闭合椭圆轨道；240 段保证高偏心率轨道近日段也平滑 */
export function ellipseScenePoints(fit: FittedEllipse, count = 240): Array<{ x: number; z: number }> {
  const p = fit.aAU * (1 - fit.e * fit.e)
  const points: Array<{ x: number; z: number }> = []
  for (let i = 0; i < count; i += 1) {
    const nu = (i / count) * Math.PI * 2
    const rAU = p / (1 + fit.e * Math.cos(nu))
    const theta = fit.perihelionAngle + nu
    const radius = sceneRadiusFromAU(rAU)
    points.push({ x: radius * Math.cos(theta), z: radius * Math.sin(theta) })
  }
  return points
}
