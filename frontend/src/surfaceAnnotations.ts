export interface SurfaceAnchorProjection {
  id: string
  anchorX: number
  anchorY: number
  visible: boolean
  selected?: boolean
  variant?: 'default' | 'observer'
  inactive?: boolean
}

export interface SurfaceAnnotationLayout extends SurfaceAnchorProjection {
  x: number
  y: number
  leaderX: number
  leaderY: number
  scale: number
  compact: boolean
  side: 'left' | 'right'
}

export interface SurfaceAnnotationViewport {
  width: number
  height: number
  currentPlanetRadiusPx: number
  referencePlanetRadiusPx: number
}

const LABEL_WIDTH = 170
const LABEL_HEIGHT = 36
const COMPACT_WIDTH = 112
const COMPACT_HEIGHT = 18
const EDGE_GAP = 10
const SAFE_INSET = 8

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value))
}

/** 行星球面在当前透视相机中的近似屏幕半径。 */
export function projectedSphereRadiusPx(
  sphereRadius: number,
  centerDistance: number,
  verticalFovDegrees: number,
  viewportHeight: number,
): number {
  if (sphereRadius <= 0 || centerDistance <= 0 || viewportHeight <= 0) return 0
  const ratio = clamp(sphereRadius / centerDistance, 0, 0.999999)
  const angularRadius = Math.asin(ratio)
  const halfFov = (verticalFovDegrees * Math.PI) / 360
  return (Math.tan(angularRadius) / Math.tan(halfFov)) * viewportHeight * 0.5
}

/**
 * 标签随行星在屏幕中的实际大小柔和缩放。选中项保留更高的可读下限，
 * 避免不同天体半径与相机距离让同一套标注呈现出不同观感。
 */
export function surfaceAnnotationScale(
  currentPlanetRadiusPx: number,
  referencePlanetRadiusPx: number,
  selected = false,
): number {
  const ratio = Math.max(currentPlanetRadiusPx, 0.001) / Math.max(referencePlanetRadiusPx, 0.001)
  return clamp(Math.pow(ratio, 0.55), selected ? 0.88 : 0.72, 1.08)
}

/** 视觉圆点的目标屏幕半径；拾取体由调用方独立保留，不受此上限影响。 */
export function surfaceMarkerRadiusPx(
  currentPlanetRadiusPx: number,
  referencePlanetRadiusPx: number,
  selected = false,
): number {
  const ratio = Math.max(currentPlanetRadiusPx, 0.001) / Math.max(referencePlanetRadiusPx, 0.001)
  const activeMultiplier = selected ? 1.18 : 1
  return clamp(5.5 * Math.pow(ratio, 0.55) * activeMultiplier, 3, 7)
}

/** 把目标像素半径转换为标记所在深度的世界空间半径。 */
export function surfaceMarkerWorldRadius(
  markerDistance: number,
  verticalFovDegrees: number,
  viewportHeight: number,
  markerRadiusPx: number,
): number {
  if (markerDistance <= 0 || viewportHeight <= 0) return 0
  const halfFov = (verticalFovDegrees * Math.PI) / 360
  return markerRadiusPx * ((2 * markerDistance * Math.tan(halfFov)) / viewportHeight)
}

/**
 * 只移动标签，不修改真实锚点。左右两列分别做稳定的纵向排布，最终引线端点
 * 取标签靠近锚点的一侧，因此缩放、旋转和避让后仍精准指回圆点。
 */
export function layoutSurfaceAnnotations<T extends SurfaceAnchorProjection>(
  anchors: T[],
  viewport: SurfaceAnnotationViewport,
): Array<T & SurfaceAnnotationLayout> {
  const ratio = Math.max(viewport.currentPlanetRadiusPx, 0.001) / Math.max(viewport.referencePlanetRadiusPx, 0.001)
  const visible = anchors
    .filter((item) => item.visible)
    .map((item) => {
      const scale = surfaceAnnotationScale(
        viewport.currentPlanetRadiusPx,
        viewport.referencePlanetRadiusPx,
        item.selected,
      )
      const compact = !item.selected && ratio < 0.78
      const width = (compact ? COMPACT_WIDTH : LABEL_WIDTH) * scale
      const height = (compact ? COMPACT_HEIGHT : LABEL_HEIGHT) * scale
      const side: 'left' | 'right' = item.anchorX + EDGE_GAP + width <= viewport.width - SAFE_INSET ? 'right' : 'left'
      const x = side === 'right' ? item.anchorX + EDGE_GAP : item.anchorX - EDGE_GAP - width
      return {
        ...item,
        scale,
        compact,
        side,
        width,
        height,
        x: clamp(x, SAFE_INSET, Math.max(SAFE_INSET, viewport.width - width - SAFE_INSET)),
        y: clamp(item.anchorY - height * 0.5, SAFE_INSET, Math.max(SAFE_INSET, viewport.height - height - SAFE_INSET)),
      }
    })

  for (const side of ['left', 'right'] as const) {
    const column = visible.filter((item) => item.side === side).sort((a, b) => a.anchorY - b.anchorY)
    let cursor = SAFE_INSET
    for (const item of column) {
      item.y = Math.max(item.y, cursor)
      cursor = item.y + item.height + 4
    }
    const overflow = cursor - 4 - (viewport.height - SAFE_INSET)
    if (overflow > 0) {
      for (const item of column) item.y = Math.max(SAFE_INSET, item.y - overflow)
    }
  }

  const layouts = visible.map(({ width, height, ...item }) => ({
    ...item,
    leaderX: item.side === 'right' ? item.x : item.x + width,
    leaderY: clamp(item.anchorY, item.y + 4 * item.scale, item.y + height - 4 * item.scale),
  }))
  const byId = new Map(layouts.map((item) => [item.id, item]))
  return anchors.map((item) => byId.get(item.id) ?? {
    ...item,
    x: item.anchorX,
    y: item.anchorY,
    leaderX: item.anchorX,
    leaderY: item.anchorY,
    scale: 1,
    compact: false,
    side: 'right',
  }) as Array<T & SurfaceAnnotationLayout>
}
