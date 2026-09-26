export type MissionEntityKind = 'spacecraft' | 'surface'

export interface MissionDetailField {
  label: string
  value: string
}

export interface MissionDetail {
  kind: MissionEntityKind
  typeZh: string
  typeEn: string
  status?: string
  nameZh: string
  nameEn?: string
  description?: string
  fields: MissionDetailField[]
  hardware?: string[]
  source?: string
  iconHtml?: string
}

type OptionalValue = string | number | null | undefined

function text(value: OptionalValue): string {
  if (value == null) return ''
  return String(value).trim()
}

export function missionField(label: string, value: OptionalValue): MissionDetailField | null {
  const normalized = text(value)
  return normalized ? { label, value: normalized } : null
}

export function missionFields(...fields: Array<MissionDetailField | null | undefined>): MissionDetailField[] {
  return fields.filter((field): field is MissionDetailField => Boolean(field))
}

export function spacecraftFields(input: {
  operator?: OptionalValue
  launch?: OptionalValue
  target?: OptionalValue
  endpoint?: OptionalValue
  inclination?: OptionalValue
  eccentricity?: OptionalValue
  period?: OptionalValue
  trajectory?: OptionalValue
}): MissionDetailField[] {
  return missionFields(
    missionField('运营机构', input.operator),
    missionField('发射信息', input.launch),
    missionField('任务目标', input.target),
    missionField('任务终点', input.endpoint),
    missionField('轨道倾角', input.inclination),
    missionField('轨道偏心率', input.eccentricity),
    missionField('轨道周期', input.period),
    missionField('轨迹类型', input.trajectory),
  )
}

export function surfaceMissionFields(input: {
  mission?: OptionalValue
  date?: OptionalValue
  operator?: OptionalValue
  region?: OptionalValue
  coordinates?: OptionalValue
  category?: OptionalValue
}): MissionDetailField[] {
  return missionFields(
    missionField('所属任务', input.mission),
    missionField('到达日期', input.date),
    missionField('运营机构', input.operator),
    missionField('任务类型', input.category),
    missionField('区域', input.region),
    missionField('坐标', input.coordinates),
  )
}

/**
 * 3D 圆点的透视补偿。指数高于旧实现的 0.6，使镜头靠近时圆点同步缩小，
 * 避免对象像贴在镜头上的 UI 气泡；上下限只防极端视距下消失或膨胀。
 */
export function missionMarkerScale(distance: number, referenceDistance: number, active = false): number {
  const ratio = Math.max(distance, 0.001) / Math.max(referenceDistance, 0.001)
  const compensated = Math.min(1.55, Math.max(0.42, Math.pow(ratio, 0.78)))
  return compensated * (active ? 1.18 : 1)
}

/** 飞行器特写保留行星与轨道上下文，并保证镜头始终位于对象轨道外侧。 */
export function spacecraftFocusDistance(
  planetRadius: number,
  currentDistance: number,
  objectDistanceFromCenter: number,
): number {
  const contextualDistance = Math.min(currentDistance, planetRadius * 6.8)
  return Math.max(contextualDistance, objectDistanceFromCenter + planetRadius * 1.15)
}

/** 着陆点/任务终点特写统一保留约 2.65 个行星半径的地表上下文。 */
export function surfaceFocusDistance(planetRadius: number): number {
  return planetRadius * 2.65
}
