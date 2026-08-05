export interface Spacecraft {
  id: string
  nameZh: string
  nameEn: string
  noradCatalogId: number
  category: string
  operatorName: string
  description: string
  sourceName: string
  sourceUrl: string
  orbitEpoch: string
  orbitSyncedAt: string
  /** 发射信息（API 可选返回） */
  launchDate?: string
  launchSite?: string
  launchVehicle?: string
  omm: Record<string, string | number>
}

export interface LaunchSite {
  id: string
  nameZh: string
  nameEn: string
  countryCode: string
  countryNameZh: string
  latitude: number
  longitude: number
  description: string
  sourceUrl: string
  tier: number
}

export interface LaunchEvent {
  externalId: string
  name: string
  nameZh: string
  statusName: string
  statusNameZh: string
  statusAbbrev: string
  net: string
  windowStart?: string
  windowEnd?: string
  padName: string
  padNameZh: string
  locationName: string
  locationNameZh: string
  latitude?: number
  longitude?: number
  missionName: string
  missionNameZh: string
  missionType: string
  missionTypeZh: string
  missionDescription: string
  missionDescriptionZh: string
  providerName: string
  sourceUrl: string
  syncedAt: string
  hasOriginal: boolean
}

export interface Freshness {
  sourceCode: string
  sourceName: string
  lastFinishedAt: string
  success: boolean
}

export interface OrbitOverview {
  generatedAt: string
  spacecraft: Spacecraft[]
  launchSites: LaunchSite[]
  events: LaunchEvent[]
  freshness: Freshness[]
}

/** 月球飞行器（来自 /api/v1/moon/spacecraft，镜像地球 Spacecraft 类型） */
export interface MoonLandingSite {
  id: string
  nameZh: string
  nameEn: string
  program: string
  operatorName: string
  landingDate: string
  latitude: number
  longitude: number
  region: string
  description: string
  sortOrder: number
  siteName: string
  officialName: string
  missionName: string
  hardware: string[]
  side: 'NEAR_SIDE' | 'FAR_SIDE'
  category: string
  icon: 'astronaut' | 'lander' | 'rover' | 'sample'
  track: number[][]
}

export interface MoonSpacecraft {
  id: string
  nameZh: string
  nameEn: string
  type: string
  operatorName: string
  description: string
  launchDate: string
  launchSite: string
  launchVehicle: string
  sourceName: string
  displayInclination: string
  displayEccentricity: string
  displayPeriod: string
  kind: 'orbital' | 'stationary'
  orbitA: number
  orbitE: number
  inclinationDeg: number
  raanDeg: number
  argPeriapsisDeg: number
  periodSeconds: number
  stationaryOffset: [number, number, number]
  sortOrder: number
  /** JPL Horizons 日同步快照（无同步时为 null，回退静态参数） */
  snapshot?: {
    epoch: string
    aKm: number
    eccentricity: number
    inclinationDeg: number
    raanDeg: number
    argPeriapsisDeg: number
    meanAnomalyDeg: number
    periodSeconds: number
  } | null
}

export type Selection =
  | { kind: 'spacecraft'; id: string }
  | { kind: 'site'; id: string }
  | { kind: 'event'; id: string }
  | { kind: 'observatory'; id: string }

/** 空间天文台（非地球轨道：日地 L2 / 日心轨道）——地球页外圈示意条目，位置方向来自真实 JPL 数据 */
export interface Observatory {
  id: string
  nameZh: string
  nameEn: string
  operatorName: string
  description: string
  /** 当前距地球距离（AU，真实 JPL 数据） */
  distanceAU: number
  /** 相对反日方向的黄道方位角偏移（度；JWST≈0 即 L2） */
  azOffsetDeg: number
  /** 场景显示方向 × 外圈半径（单位：场景单位） */
  position: { x: number; y: number; z: number }
  /** 位置说明文案（如实标注方向/距离非等比） */
  note: string
  color: number
  syncedAt?: string
}

export interface SceneLayers {
  spacecraft: boolean
  orbits: boolean
  sites: boolean
}

/** 深空探测器（来自 /api/v1/voyage/probes，VOYAGE 太阳系标注） */
export interface DeepSpaceProbe {
  id: string
  nameZh: string
  nameEn: string
  operatorName: string
  launchDate: string
  launchSite: string
  launchVehicle: string
  missionType: string
  target: string
  description: string
  precisionGrade: string
  color: string
  sortOrder: number
  /** 最近一次 JPL Horizons 同步时间（无采样时为空） */
  syncedAt?: string
  /** 轨道绘制方式：ellipse = 拟合椭圆轨道（太阳在焦点）；track = 不绘制轨迹（仅标记） */
  orbitKind: string
  /** 日心黄道位置采样（km，按时间升序，±90 天窗口） */
  positions: Array<{ epoch: string; x: number; y: number; z: number }>
}
