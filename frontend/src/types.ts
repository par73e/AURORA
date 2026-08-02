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

export type Selection =
  | { kind: 'spacecraft'; id: string }
  | { kind: 'site'; id: string }
  | { kind: 'event'; id: string }

export interface SceneLayers {
  spacecraft: boolean
  orbits: boolean
  sites: boolean
}
