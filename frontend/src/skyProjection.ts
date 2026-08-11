export interface SkyCamera {
  /** 视野中央的地理方位角：0° 北、90° 东。 */
  heading: number
  /** 圆柱全景展开的水平视野，单位为度。 */
  horizontalFov: number
}

export interface SkyProjection {
  /** 归一化屏幕坐标；0/1 分别对应画布左右边缘。 */
  x: number
  /** 归一化屏幕坐标；0/1 分别对应画布上下边缘。 */
  y: number
  /** 朝向视野前方的深度分量，小于等于 0 表示超过前方半球。 */
  depth: number
  inFront: boolean
  inViewport: boolean
}

export interface ProjectedAltitudePoint extends SkyProjection {
  relativeAzimuth: number
}

export interface ProjectedAltitudeGuide {
  altitude: number
  points: ProjectedAltitudePoint[]
  path: string
  label: ProjectedAltitudePoint | null
}

export function normalizeSkyAngle(value: number) {
  return (value % 360 + 360) % 360
}

// 为山体和航向拨盘留出低空空间；天顶也稍离开圆角画框，避免标记被裁切。
const HORIZON_Y = .88
const ZENITH_Y = .04
const ALTITUDE_EASING = 1.08
// 轻度使用 S 曲线展开中段高度：拉开 30° / 60°，但不回到针孔超广角的夸张间距。
const MIDDLE_ALTITUDE_SPREAD = .25
// 视野边缘最多上提约 2.8% 画面高度，保留天幕感而不制造超广角畸变。
const ALTITUDE_ARC_LIFT = .028

/**
 * 将真实地平坐标展开到带轻微穹顶弧度的圆柱全景天幕。
 *
 * 0° 地平线抬到山体之上；30° / 60° 分布在接近三等分的位置。
 * 同一高度角在视野两侧轻微上扬，天体转动时始终贴合对应参考弧线。
 */
export function projectHorizontalDirection(azimuth: number, altitude: number, camera: SkyCamera): SkyProjection {
  const horizontalFov = Math.max(2, Math.min(180, camera.horizontalFov))
  const relativeAzimuth = normalizeSkyAngle(azimuth - camera.heading + 180) - 180
  const depth = Math.cos(relativeAzimuth * Math.PI / 180)
  const inFront = depth > 1e-6

  if (!inFront) {
    return { x: Number.NaN, y: Number.NaN, depth, inFront: false, inViewport: false }
  }

  const x = .5 + relativeAzimuth / horizontalFov
  const normalizedAltitude = Math.max(0, Math.min(1, altitude / 90))
  const easedAltitude = normalizedAltitude ** ALTITUDE_EASING
  const smoothAltitude = normalizedAltitude ** 2 * (3 - 2 * normalizedAltitude)
  const altitudeProgress = easedAltitude * (1 - MIDDLE_ALTITUDE_SPREAD) + smoothAltitude * MIDDLE_ALTITUDE_SPREAD
  const baseY = HORIZON_Y - altitudeProgress * (HORIZON_Y - ZENITH_Y)
  const normalizedEdgeDistance = Math.min(1, Math.abs(relativeAzimuth) / (horizontalFov / 2))
  const y = baseY - ALTITUDE_ARC_LIFT * normalizedEdgeDistance ** 2
  const inAltitudeRange = altitude >= 0 && altitude <= 90
  return {
    x,
    y,
    depth,
    inFront: true,
    inViewport: inAltitudeRange && x >= 0 && x <= 1 && y >= 0 && y <= 1,
  }
}

/**
 * 采样一条真实“恒高度圈”。视觉弧度与天体共用同一投影，不是独立装饰线。
 */
export function projectAltitudeGuide(altitude: number, camera: SkyCamera, step = 1): ProjectedAltitudeGuide {
  const margin = 18
  const halfRange = Math.min(89, camera.horizontalFov / 2 + margin)
  const safeStep = Math.max(.25, Math.min(5, step))
  const points: ProjectedAltitudePoint[] = []

  for (let relativeAzimuth = -halfRange; relativeAzimuth <= halfRange + 1e-6; relativeAzimuth += safeStep) {
    const projection = projectHorizontalDirection(camera.heading + relativeAzimuth, altitude, camera)
    if (!projection.inFront) continue
    points.push({ ...projection, relativeAzimuth })
  }

  const path = points.length
    ? `M ${points.map((point) => `${(point.x * 1000).toFixed(2)} ${(point.y * 1000).toFixed(2)}`).join(' L ')}`
    : ''
  const label = points.find((point) => point.x >= .035 && point.x <= .965 && point.y >= .045 && point.y <= .93) ?? null

  return { altitude, points, path, label }
}
