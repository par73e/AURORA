// 第一视角天空透视投影（azimuthal perspective）。
//
// 模型：观测者站在地面，抬头看向 `viewAzimuth` 方向、仰角 ALT0 的前方天空。
// 把天球上的 (方位角 az, 高度角 alt) 经透视投影映射到屏幕坐标——
// 同一高度角在画面左右边缘看起来更高（穹顶透视），天体沿真实弧线运动。
// 天顶在画面顶部、地平线在底部、水平视场 H_FOV。
//
// 屏幕坐标系：viewBox 1000×520，x 向右、y 向下。
// 单位：az/alt/viewAzimuth 均为度。

export interface SkyProjection {
  /** 观察仰角（视场中心略上仰），度 */
  alt0: number
  /** 水平半视场，度 */
  halfFov: number
  /** 地平线所在屏幕位置（百分比，0–100；下方为罗盘区） */
  horizonY: number
}

export const SKY_PROJECTION: SkyProjection = {
  alt0: 25,
  halfFov: 60, // 水平视场 120°
  horizonY: 85,
}

const DEG = Math.PI / 180

function unit(az: number, alt: number): [number, number, number] {
  const a = az * DEG
  const e = alt * DEG
  return [Math.cos(e) * Math.cos(a), Math.cos(e) * Math.sin(a), Math.sin(e)]
}

function dot(a: number[], b: number[]): number {
  return a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
}

function sub(a: number[], b: number[]): number[] {
  return [a[0] - b[0], a[1] - b[1], a[2] - b[2]]
}

function normalize(v: number[]): number[] {
  const n = Math.hypot(v[0], v[1], v[2])
  return [v[0] / n, v[1] / n, v[2] / n]
}

/**
 * 把天空中的 (方位角, 高度角) 投影到屏幕坐标。
 * @param az        天体的方位角（度，0=北 90=东 180=南）
 * @param alt       天体的高度角（度，0=地平线 90=天顶）
 * @param viewAzimuth 当前观察方位（罗盘朝向）
 * @returns 屏幕坐标（viewBox 1000×520）与可见性；视野外 visible=false
 */
export function projectToSky(az: number, alt: number, viewAzimuth: number, proj: SkyProjection = SKY_PROJECTION): { x: number; y: number; visible: boolean } {
  // 观察方向向量（alt0 上仰、朝向 viewAzimuth）
  const view = unit(viewAzimuth, proj.alt0)
  // 屏幕基向量：right 沿地平线（az+90）、up 为天顶方向在垂直面的投影
  const right = unit(viewAzimuth + 90, 0)
  const zUp = [0, 0, 1] as number[]
  const up = normalize(sub(zUp, view.map((value) => value * dot(zUp, view))))

  const target = unit(az, alt)
  const depth = dot(target, view) // 目标在观察方向的投影深度
  if (depth <= 0.05) return { x: 0, y: 0, visible: false } // 视野外/身后

  // 透视投影：屏幕坐标 = 垂直分量 / 深度（透视除法）
  const rawX = dot(target, right) / depth
  const rawY = dot(target, up) / depth

  // 屏幕映射（百分比坐标，0–100，同时用于 SVG viewBox 与 HTML 定位）：
  // rawX ∈ [-tan(halfFov), tan(halfFov)] → [0, 100]；垂直方向以
  // "正前方地平线"(alt=0, az=viewAzimuth)为 horizonY(=85%)，天顶向上。
  const x = 50 + rawX / Math.tan(proj.halfFov * DEG) * 46
  const horizonRawY = dot(unit(viewAzimuth, 0), up) / dot(unit(viewAzimuth, 0), view)
  const zenithRawY = dot(unit(viewAzimuth, 90), up) / dot(unit(viewAzimuth, 90), view)
  const scale = proj.horizonY / Math.abs(zenithRawY - horizonRawY)
  const y = proj.horizonY - (rawY - horizonRawY) * scale

  const visible = x >= -4 && x <= 104 && y >= -4 && y <= proj.horizonY + 4
  return { x, y, visible }
}

/**
 * 把一条天空轨迹（升落弧线等）采样点投影成 SVG path。
 * @param samples 轨迹采样点（az/alt 度）
 * @param viewAzimuth 当前观察方位
 * @returns SVG path d；空串表示全部不可见
 */
export function trackToPath(samples: Array<{ az: number; alt: number }>, viewAzimuth: number, proj: SkyProjection = SKY_PROJECTION): string {
  let path = ''
  for (const sample of samples) {
    const p = projectToSky(sample.az, sample.alt, viewAzimuth, proj)
    if (!p.visible) {
      if (path) path += ' Z'
      continue
    }
    const segment = ` ${p.x.toFixed(1)} ${p.y.toFixed(1)}`
    if (!path || path.endsWith(' Z')) {
      path = path ? `${path} M${segment}` : `M${segment}`
    } else {
      path += ` L${segment}`
    }
  }
  return path
}
