export type SceneAnnotationMode = 'full' | 'compact' | 'cluster'

export interface SceneAnchorProjection {
  id: string
  anchorX: number
  anchorY: number
  visible: boolean
  selected?: boolean
  hovered?: boolean
  variant?: 'default' | 'observer'
  inactive?: boolean
}

export type SurfaceAnchorProjection = SceneAnchorProjection

export interface SceneAnnotationLayout extends SceneAnchorProjection {
  x: number
  y: number
  scale: number
  compact: boolean
  mode: SceneAnnotationMode
  clusterCount: number
  memberIds: string[]
}

export type SurfaceAnnotationLayout = SceneAnnotationLayout

export interface SceneAnnotationViewport {
  width: number
  height: number
  currentPlanetRadiusPx: number
  referencePlanetRadiusPx: number
}

export type SurfaceAnnotationViewport = SceneAnnotationViewport

const LABEL_WIDTH = 170
const LABEL_HEIGHT = 36
const COMPACT_WIDTH = 112
const COMPACT_HEIGHT = 18
const CLUSTER_WIDTH = 42
const CLUSTER_HEIGHT = 22
const CONNECTOR_LENGTH_PX = 10
const SAFE_INSET = 8
const CLUSTER_ENTER_PX = 16
const CLUSTER_EXIT_PX = 22

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
 * 点、固定短线和标签共享的视觉缩放。指数刻意较小，并限制在窄区间内，
 * 让远处仍可辨认、近处也不会突然膨胀；这是一种可读性透视而非物理透视。
 */
export function sceneAnnotationScale(
  currentPlanetRadiusPx: number,
  referencePlanetRadiusPx: number,
): number {
  const ratio = Math.max(currentPlanetRadiusPx, 0.001) / Math.max(referencePlanetRadiusPx, 0.001)
  return clamp(Math.pow(ratio, 0.28), 0.75, 1.05)
}

/** 视觉圆点的目标屏幕半径；选中状态只改变样式，不改变几何大小。 */
export function sceneMarkerRadiusPx(
  currentPlanetRadiusPx: number,
  referencePlanetRadiusPx: number,
  baseRadiusPx: number,
): number {
  return baseRadiusPx * sceneAnnotationScale(currentPlanetRadiusPx, referencePlanetRadiusPx)
}

export function surfaceMarkerRadiusPx(
  currentPlanetRadiusPx: number,
  referencePlanetRadiusPx: number,
  _selected = false,
): number {
  return sceneMarkerRadiusPx(currentPlanetRadiusPx, referencePlanetRadiusPx, 5)
}

export function orbitMarkerRadiusPx(
  currentPlanetRadiusPx: number,
  referencePlanetRadiusPx: number,
): number {
  return sceneMarkerRadiusPx(currentPlanetRadiusPx, referencePlanetRadiusPx, 4.5)
}

/** 把目标像素半径转换为标记所在深度的世界空间半径。 */
export function sceneMarkerWorldRadius(
  markerDistance: number,
  verticalFovDegrees: number,
  viewportHeight: number,
  markerRadiusPx: number,
): number {
  if (markerDistance <= 0 || viewportHeight <= 0) return 0
  const halfFov = (verticalFovDegrees * Math.PI) / 360
  return markerRadiusPx * ((2 * markerDistance * Math.tan(halfFov)) / viewportHeight)
}

export const surfaceMarkerWorldRadius = sceneMarkerWorldRadius

function annotationSize(mode: SceneAnnotationMode, scale: number) {
  if (mode === 'cluster') return { width: CLUSTER_WIDTH * scale, height: CLUSTER_HEIGHT * scale }
  if (mode === 'compact') return { width: COMPACT_WIDTH * scale, height: COMPACT_HEIGHT * scale }
  return { width: LABEL_WIDTH * scale, height: LABEL_HEIGHT * scale }
}

function overlaps(
  a: { x: number; y: number; width: number; height: number },
  b: { x: number; y: number; width: number; height: number },
) {
  return a.x < b.x + b.width && a.x + a.width > b.x && a.y < b.y + b.height && a.y + a.height > b.y
}

function distance(a: SceneAnchorProjection, b: SceneAnchorProjection) {
  return Math.hypot(a.anchorX - b.anchorX, a.anchorY - b.anchorY)
}

/**
 * 标签的几何锚点始终固定在圆点右侧。空间不足时只切换信息密度：
 * full -> compact -> cluster；不再通过移动标签或改变引线角度来避让。
 */
export function layoutSceneAnnotations<T extends SceneAnchorProjection>(
  anchors: T[],
  viewport: SceneAnnotationViewport,
  previous: SceneAnnotationLayout[] = [],
): Array<T & SceneAnnotationLayout> {
  const scale = sceneAnnotationScale(viewport.currentPlanetRadiusPx, viewport.referencePlanetRadiusPx)
  const ratio = Math.max(viewport.currentPlanetRadiusPx, 0.001) / Math.max(viewport.referencePlanetRadiusPx, 0.001)
  const previousClusterById = new Map<string, Set<string>>()
  for (const item of previous) {
    if (item.mode !== 'cluster') continue
    const members = new Set(item.memberIds)
    for (const id of members) previousClusterById.set(id, members)
  }

  const visible = anchors.filter((item) => item.visible)
  const parent = new Map(visible.map((item) => [item.id, item.id]))
  const find = (id: string): string => {
    const current = parent.get(id) ?? id
    if (current === id) return id
    const root = find(current)
    parent.set(id, root)
    return root
  }
  const union = (a: string, b: string) => {
    const rootA = find(a)
    const rootB = find(b)
    if (rootA !== rootB) parent.set(rootB, rootA)
  }

  for (let i = 0; i < visible.length; i += 1) {
    const a = visible[i]
    if (a.selected || a.hovered) continue
    for (let j = i + 1; j < visible.length; j += 1) {
      const b = visible[j]
      if (b.selected || b.hovered) continue
      const wasTogether = previousClusterById.get(a.id)?.has(b.id) ?? false
      const threshold = (wasTogether ? CLUSTER_EXIT_PX : CLUSTER_ENTER_PX) * scale
      if (distance(a, b) <= threshold) union(a.id, b.id)
    }
  }

  const groups = new Map<string, T[]>()
  for (const item of visible) {
    const root = find(item.id)
    const group = groups.get(root) ?? []
    group.push(item)
    groups.set(root, group)
  }

  const accepted: Array<{ x: number; y: number; width: number; height: number }> = []
  const layouts = new Map<string, T & SceneAnnotationLayout>()

  const orderedGroups = [...groups.values()].sort((a, b) => {
    const aActive = a.some((item) => item.selected || item.hovered) ? 1 : 0
    const bActive = b.some((item) => item.selected || item.hovered) ? 1 : 0
    return bActive - aActive
  })

  for (const group of orderedGroups) {
    const clustered = group.length > 1
    const representative = group.find((item) => item.selected || item.hovered) ?? group[0]
    const memberIds = group.map((item) => item.id)

    for (const item of group) {
      let mode: SceneAnnotationMode = clustered ? 'cluster' : (ratio < 0.82 && !item.selected && !item.hovered ? 'compact' : 'full')
      const isRepresentative = item.id === representative.id
      if (clustered && !isRepresentative) {
        layouts.set(item.id, {
          ...item,
          visible: false,
          x: item.anchorX,
          y: item.anchorY,
          scale,
          compact: false,
          mode,
          clusterCount: group.length,
          memberIds,
        })
        continue
      }

      let size = annotationSize(mode, scale)
      const x = item.anchorX + CONNECTOR_LENGTH_PX * scale
      const y = item.anchorY
      let box = { x, y: y - size.height * 0.5, ...size }
      const fits = () => box.x >= SAFE_INSET
        && box.x + box.width <= viewport.width - SAFE_INSET
        && box.y >= SAFE_INSET
        && box.y + box.height <= viewport.height - SAFE_INSET

      if (mode === 'full' && !item.selected && !item.hovered && (!fits() || accepted.some((other) => overlaps(box, other)))) {
        mode = 'compact'
        size = annotationSize(mode, scale)
        box = { x, y: y - size.height * 0.5, ...size }
      }

      const finalVisible = fits() || item.selected || item.hovered
      if (finalVisible) accepted.push(box)
      layouts.set(item.id, {
        ...item,
        visible: finalVisible,
        x,
        y,
        scale,
        compact: mode === 'compact',
        mode,
        clusterCount: clustered ? group.length : 1,
        memberIds,
      })
    }
  }

  return anchors.map((item) => layouts.get(item.id) ?? {
    ...item,
    x: item.anchorX,
    y: item.anchorY,
    scale,
    compact: false,
    mode: 'full',
    clusterCount: 1,
    memberIds: [item.id],
  }) as Array<T & SceneAnnotationLayout>
}
