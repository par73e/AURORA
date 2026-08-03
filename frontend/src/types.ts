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

export interface SceneLayers {
  spacecraft: boolean
  orbits: boolean
  sites: boolean
}
