// 真实日心距离（AU）→ 场景径向单位（示意尺度）的单调映射。
// 场景的行星轨道半径是"近似等差排布"（水星 14 → 海王星 130），不是真实 AU 线性比例；
// 这里用对数空间分段线性插值穿过 8 颗行星锚点（真实 AU → 场景轨道半径），
// 尾部外推——深空探测器（如旅行者号 ~171 AU）落到柯伊伯带之外的深处，
// 语义上正好是"正在离开太阳系"。

export const AU_KM = 149_597_870.7

/** 行星锚点：[真实日心平均距离 AU, 场景轨道半径]（与 solar/data.ts 的 orbitRadius 一致）。
 *  首锚点 [0.2, 7] 为近太阳钳制：太阳半径 5.5，标记/轨迹须保持在日面之外；
 *  帕克（近日 0.046 AU）等超近日段被压缩到该圆弧，避免负半径镜像。 */
const ANCHORS: ReadonlyArray<readonly [au: number, units: number]> = [
  [0.2, 7], // 近太阳钳制锚点
  [0.387, 14], // 水星
  [0.723, 30], // 金星
  [1.0, 46], // 地球
  [1.524, 60], // 火星
  [5.203, 82], // 木星
  [9.537, 98], // 土星
  [19.19, 114], // 天王星
  [30.07, 130], // 海王星
]

const LOG_ANCHORS = ANCHORS.map(([au, units]) => [Math.log10(au), units] as const)

/** 真实日心距离（AU）→ 场景径向单位（对数空间分段线性插值 + 首尾外推）。
 *  输入先钳制到首锚点（0.2 AU），保证任何探测器的映射半径 ≥ 7，不会出现负半径 */
export function sceneRadiusFromAU(au: number): number {
  const log = Math.log10(Math.max(au, ANCHORS[0][0]))
  const first = LOG_ANCHORS[0]
  if (log <= first[0]) {
    // 低于水星（帕克近日点等）：按第一段斜率延伸
    const next = LOG_ANCHORS[1]
    return first[1] + ((log - first[0]) * (next[1] - first[1])) / (next[0] - first[0])
  }
  for (let i = 0; i < LOG_ANCHORS.length - 1; i += 1) {
    const a = LOG_ANCHORS[i]
    const b = LOG_ANCHORS[i + 1]
    if (log <= b[0]) {
      return a[1] + ((log - a[0]) * (b[1] - a[1])) / (b[0] - a[0])
    }
  }
  // 超出海王星：按最后一段斜率外推
  const a = LOG_ANCHORS[LOG_ANCHORS.length - 2]
  const b = LOG_ANCHORS[LOG_ANCHORS.length - 1]
  return b[1] + ((log - b[0]) * (b[1] - a[1])) / (b[0] - a[0])
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
