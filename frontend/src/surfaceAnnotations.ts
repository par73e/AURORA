export type SceneAnnotationMode = 'full' | 'compact' | 'cluster'
export type SceneAnnotationSide = 'left' | 'right'

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
  side: SceneAnnotationSide
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
const PAIR_EXPAND_RATIO = 1.1
const PAIR_COLLAPSE_RATIO = 0.95

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
 * 点、固定短线和标签共享视觉缩放。远景明显收小，近景封顶；
 * 最小飞行器圆点直径仍有 4.5px，避免缩小文字时把目标点一并丢失。
 */
export function sceneAnnotationScale(
  currentPlanetRadiusPx: number,
  referencePlanetRadiusPx: number,
): number {
  const ratio = Math.max(currentPlanetRadiusPx, 0.001) / Math.max(referencePlanetRadiusPx, 0.001)
  return clamp(0.9 * Math.pow(ratio, 0.8), 0.5, 1.05)
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

/** x 是靠近圆点的标签边缘；左侧按真实 DOM 宽度定位，不猜测文字宽度。 */
export function sceneAnnotationStyle(label: SceneAnnotationLayout) {
  return {
    left: `${label.x}px`,
    top: `${label.y}px`,
    transform: label.side === 'left'
      ? `translate(-100%, -50%) scale(${label.scale})`
      : `translateY(-50%) scale(${label.scale})`,
    transformOrigin: `${label.side === 'left' ? 'right' : 'left'} center`,
  }
}

/**
 * 所有短线长度固定。近景的两个重叠目标向左右展开，即使坐标完全相同
 * 也不要求先分离；空间不足时使用紧凑标签或可展开的聚合缩略图。
 */
export function layoutSceneAnnotations<T extends SceneAnchorProjection>(
  anchors: T[],
  viewport: SceneAnnotationViewport,
  previous: SceneAnnotationLayout[] = [],
): Array<T & SceneAnnotationLayout> {
  const scale = sceneAnnotationScale(viewport.currentPlanetRadiusPx, viewport.referencePlanetRadiusPx)
  const ratio = Math.max(viewport.currentPlanetRadiusPx, 0.001) / Math.max(viewport.referencePlanetRadiusPx, 0.001)
  const previousById = new Map(previous.map((item) => [item.id, item]))
  const previousClusterById = new Map<string, Set<string>>()
  for (const item of previous) {
    if (item.memberIds.length < 2) continue
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
    if (a.variant === 'observer') continue
    for (let j = i + 1; j < visible.length; j += 1) {
      const b = visible[j]
      if (b.variant === 'observer') continue
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

  const orderedGroups = [...groups.values()].flatMap((group) => {
    if (group.length <= 2) return [group]
    const active = group.filter((item) => item.selected || item.hovered)
    const rest = group.filter((item) => !item.selected && !item.hovered)
    return [...active.map((item) => [item]), ...(rest.length ? [rest] : [])]
  }).sort((a, b) => {
    const aActive = a.some((item) => item.selected || item.hovered) ? 1 : 0
    const bActive = b.some((item) => item.selected || item.hovered) ? 1 : 0
    return bActive - aActive
  })

  for (const group of orderedGroups) {
    // 排序与选中/悬停无关，同坐标的两项不会在交互时互换左右。
    group.sort((a, b) => a.anchorX - b.anchorX || a.id.localeCompare(b.id))
    const wasExpanded = group.length === 2 && group.every((item) => {
      const before = previousById.get(item.id)
      return before?.visible && before.mode !== 'cluster' && before.memberIds.length === 2
    })
    const expandPair = group.length === 2
      && (ratio >= (wasExpanded ? PAIR_COLLAPSE_RATIO : PAIR_EXPAND_RATIO)
        || group.some((item) => item.selected || item.hovered))
    let clustered = group.length > 1 && !expandPair
    const representative = group.find((item) => item.selected || item.hovered)
      ?? group.find((item) => previousById.get(item.id)?.visible && previousById.get(item.id)?.mode === 'cluster')
      ?? group[0]
    const memberIds = group.map((item) => item.id).sort()

    const boxFor = (item: T, mode: SceneAnnotationMode, side: SceneAnnotationSide) => {
      const size = annotationSize(mode, scale)
      const edge = item.anchorX + (side === 'left' ? -1 : 1) * CONNECTOR_LENGTH_PX * scale
      return { x: side === 'left' ? edge - size.width : edge, y: item.anchorY - size.height / 2, ...size }
    }
    const fits = (box: ReturnType<typeof boxFor>) => box.x >= SAFE_INSET
      && box.x + box.width <= viewport.width - SAFE_INSET
      && box.y >= SAFE_INSET
      && box.y + box.height <= viewport.height - SAFE_INSET
    const previousSides = group.map((item) => previousById.get(item.id)?.side)
    const sides: SceneAnnotationSide[] = group.length === 2
      ? wasExpanded && new Set(previousSides).size === 2
        ? previousSides as SceneAnnotationSide[]
        : ['left', 'right']
      : ['right']
    let pairMode: SceneAnnotationMode = 'full'
    if (expandPair) {
      const pairFits = (mode: SceneAnnotationMode) => group.every((item, index) => {
        const box = boxFor(item, mode, sides[index])
        return fits(box) && !accepted.some((other) => overlaps(box, other))
      })
      if (!pairFits('full')) pairMode = 'compact'
      if (!pairFits(pairMode)) clustered = true
    }

    for (const [index, item] of group.entries()) {
      let side: SceneAnnotationSide = expandPair && !clustered ? sides[index] : 'right'
      let mode: SceneAnnotationMode = clustered ? 'cluster' : expandPair ? pairMode : (ratio < 0.82 && !item.selected && !item.hovered ? 'compact' : 'full')
      const isRepresentative = item.id === representative.id
      if (clustered && !isRepresentative) {
        layouts.set(item.id, {
          ...item,
          visible: false,
          x: item.anchorX,
          y: item.anchorY,
          scale,
          side,
          compact: false,
          mode,
          clusterCount: group.length,
          memberIds,
        })
        continue
      }

      let box = boxFor(item, mode, side)
      if (!expandPair || clustered) {
        // 先尝试两个固定方向，再降低信息密度；不会生成斜线或自由位移。
        const preferredSide = previousById.get(item.id)?.side ?? 'right'
        const choices: SceneAnnotationSide[] = [preferredSide, preferredSide === 'right' ? 'left' : 'right']
        const modes: SceneAnnotationMode[] = mode === 'full' ? ['full', 'compact'] : [mode]
        const candidates = modes.flatMap((candidateMode) => choices.map((candidateSide) => ({
          mode: candidateMode,
          side: candidateSide,
          box: boxFor(item, candidateMode, candidateSide),
        })))
        const candidate = candidates.find((entry) => fits(entry.box) && !accepted.some((other) => overlaps(entry.box, other)))
          ?? candidates.find((entry) => fits(entry.box))
        if (candidate) ({ mode, side, box } = candidate)
      }

      const finalVisible = fits(box)
      if (finalVisible) accepted.push(box)
      layouts.set(item.id, {
        ...item,
        visible: finalVisible,
        x: item.anchorX + (side === 'left' ? -1 : 1) * CONNECTOR_LENGTH_PX * scale,
        y: item.anchorY,
        scale,
        side,
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
    side: 'right',
    compact: false,
    mode: 'full',
    clusterCount: 1,
    memberIds: [item.id],
  }) as Array<T & SceneAnnotationLayout>
}
