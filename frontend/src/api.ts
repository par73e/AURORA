import type { DeepSpaceProbe, MarsLandingSite, MarsSpacecraft, MoonLandingSite, MoonSpacecraft, OrbitOverview, SpacecraftCatalogPage } from './types'

async function requestJSON<T>(endpoint: string, signal?: AbortSignal): Promise<T> {
  const response = await fetch(endpoint, { signal })
  if (!response.ok) {
    throw new Error(`接口返回 ${response.status}`)
  }
  return response.json() as Promise<T>
}

export async function fetchOrbitOverview(): Promise<OrbitOverview> {
  return requestJSON<OrbitOverview>('/api/v1/orbit/overview')
}

export interface SpacecraftCatalogParams {
  page: number
  pageSize: number
  query?: string
  operator?: string
  sort?: 'name' | 'norad' | 'operator'
  mode?: 'keyword' | 'regex'
}

/** 大目录查询走后端分页；不再要求场景首屏把全部目录载入浏览器。 */
export async function fetchSpacecraftCatalog(params: SpacecraftCatalogParams, signal?: AbortSignal): Promise<SpacecraftCatalogPage> {
  const search = new URLSearchParams({
    page: String(params.page),
    pageSize: String(params.pageSize),
    sort: params.sort ?? 'name',
    mode: params.mode ?? 'keyword',
  })
  if (params.query) search.set('q', params.query)
  if (params.operator && params.operator !== 'all') search.set('operator', params.operator)
  return requestJSON<SpacecraftCatalogPage>(`/api/v1/orbit/spacecraft?${search}`, signal)
}

export async function fetchDeepSpaceProbes(signal?: AbortSignal): Promise<DeepSpaceProbe[]> {
  const data = await requestJSON<{ probes: DeepSpaceProbe[] }>('/api/v1/voyage/probes', signal)
  return data.probes ?? []
}

export function fetchMoonSpacecraft(signal?: AbortSignal) {
  return requestJSON<{ spacecraft: MoonSpacecraft[]; syncedAt?: string | null }>('/api/v1/moon/spacecraft', signal)
}

export function fetchMoonLandingSites(signal?: AbortSignal) {
  return requestJSON<{ landingSites: MoonLandingSite[] }>('/api/v1/moon/landing-sites', signal)
}

export function fetchMarsSpacecraft(signal?: AbortSignal) {
  return requestJSON<{ spacecraft: MarsSpacecraft[]; syncedAt?: string | null }>('/api/v1/mars/spacecraft', signal)
}

export function fetchMarsLandingSites(signal?: AbortSignal) {
  return requestJSON<{ landingSites: MarsLandingSite[] }>('/api/v1/mars/landing-sites', signal)
}

export interface ObserverPlace {
  label: string
  province: string
  city: string
  district: string
  adcode: string
}

export interface ObserverPlaceCandidate extends ObserverPlace {
  level: string
  latitude: number
  longitude: number
}

export function fetchObserverPlace(latitude: number, longitude: number, signal?: AbortSignal) {
  const search = new URLSearchParams({
    latitude: latitude.toFixed(6),
    longitude: longitude.toFixed(6),
  })
  return requestJSON<ObserverPlace>(`/api/v1/location/reverse?${search}`, signal)
}

export function searchObserverPlaces(query: string, signal?: AbortSignal) {
  const search = new URLSearchParams({ q: query.trim() })
  return requestJSON<{ places: ObserverPlaceCandidate[] }>(`/api/v1/location/search?${search}`, signal)
}

export interface ObservingConditions {
  retrievedAt: string
  timezone: string
  source: string
  elevation?: number // Open-Meteo 返回的观测点海拔（米）
  current: {
    time: string
    temperature: number
    dewPoint: number
    cloudCover: number
    cloudCoverLow: number
    cloudCoverMid: number
    cloudCoverHigh: number
    visibilityMeters: number
    humidity: number
    precipitation: number
    windSpeed: number
    windGusts: number
    weatherCode: number
  }
  hourly: Array<{
    time: string
    temperature: number
    dewPoint: number
    cloudCover: number
    cloudCoverLow: number
    cloudCoverMid: number
    cloudCoverHigh: number
    visibilityMeters: number
    humidity: number
    precipitation: number
    windSpeed: number
    windDirection: number
    windGusts: number
    pressure: number
    weatherCode: number
  }>
  airQuality: Array<{
    time: string
    pm25: number
    pm10: number
    aerosolOpticalDepth: number
  }>
  /** 携带 time 参数时返回：该时刻最近的逐小时预报快照。 */
  selected?: {
    hour: ObservingConditions['hourly'][number]
    withinForecastWindow: boolean
  }
  /** 携带 scores=1 时返回：逐小时观测评分（动态推荐和今夜评分的输入）。 */
  scores?: Array<{
    time: string
    score: number
    verdict: string
    factors: ScoreFactors
    weather: ObservingConditions['hourly'][number]
  }>
}

export function fetchObservingConditions(latitude: number, longitude: number, signal?: AbortSignal, scores = false) {
  const search = new URLSearchParams({ latitude: latitude.toFixed(6), longitude: longitude.toFixed(6) })
  if (scores) search.set('scores', '1')
  return requestJSON<ObservingConditions>(`/api/v1/astronomy/conditions?${search}`, signal)
}

export interface MoonPhaseSnapshot {
  phase: number // 相位角 0–360（0 朔、180 望）
  illumination: number // 亮面占比 0–1
  age: number // 月龄（天）
  label: string
}

export interface MoonDay {
  date: string
  timezone: string
  phase: number
  illumination: number
  age: number
  label: string
  moonrise: number | null // Unix 秒
  moonset: number | null
  transit: number | null
  computedAt: number
}

/** 后端每日月相数据（按经纬度+海拔+时区，每天缓存一次）。 */
export function fetchMoonDay(latitude: number, longitude: number, elevation: number, timezone: string, at?: number, signal?: AbortSignal) {
  const search = new URLSearchParams({
    latitude: latitude.toFixed(6),
    longitude: longitude.toFixed(6),
    elevation: String(elevation),
    timezone,
  })
  if (at) search.set('at', String(at))
  return requestJSON<MoonDay>(`/api/v1/astronomy/moon?${search}`, signal)
}

export interface ScoreFactors {
  visibilityPenalty: number
  cloudPenalty: number
  moonPenalty: number
  precipitationPenalty: number
  aerosolPenalty: number
  windPenalty: number
  dewPenalty: number
  weatherPenalty: number
}

export interface ObservingScore {
  at: number
  timezone: string
  score: number | null
  verdict: string
  withinForecastWindow: boolean
  factors: ScoreFactors
  weather: ObservingConditions['hourly'][number] | null
  moon: MoonPhaseSnapshot
  moonAltitude: number
  moonAboveHorizon: boolean
  lightPollution?: LightPollution
}

/** 后端动态观测评分（天气×月相×时刻）；光污染作为地点长期环境数据单独返回。 */
export function fetchObservingScore(latitude: number, longitude: number, at?: number, signal?: AbortSignal) {
  const search = new URLSearchParams({ latitude: latitude.toFixed(6), longitude: longitude.toFixed(6) })
  if (at) search.set('at', String(at))
  return requestJSON<ObservingScore>(`/api/v1/astronomy/score?${search}`, signal)
}

export interface LightPollution {
  bortle: number
  sqm: number
  radiance: number
  radianceUnit: string
  dataYear?: number
  resolutionMeters?: number
  estimated: boolean
  model?: string
  source: string
  retrievedAt: number
}

/** 年度卫星辐射与估算光污染；未配置或超出本地栅格覆盖时返回 null。 */
export async function fetchLightPollution(latitude: number, longitude: number, signal?: AbortSignal): Promise<LightPollution | null> {
  const search = new URLSearchParams({ latitude: latitude.toFixed(6), longitude: longitude.toFixed(6) })
  try {
    return await requestJSON<LightPollution>(`/api/v1/astronomy/light-pollution?${search}`, signal)
  } catch {
    return null // 未配置或失败：保持诚实占位
  }
}

/** 天象事件：后端每日生成/校验，前端只请求 AURORA API。 */
export interface AstronomyEvent {
  id: string
  kind: string
  title: string
  titleEn: string
  startsAt: string
  endsAt?: string | null
  dateLabel: string
  summary: string
  origin: 'computed' | 'external_forecast' | 'curated'
  sourceName: string
  sourceUrl: string
  verifiedAt: string
  geometry: Record<string, unknown>
  global: Record<string, unknown> & { description: string }
  source: {
    kind: 'computed' | 'external_forecast' | 'curated'
    name: string
    url: string
    verifiedAt: string
  }
  local?: EventLocalVisibility
}

export interface EventLocalVisibility {
  status: 'observable' | 'limited' | 'not_visible' | 'non_visual' | 'not_calculated'
  bestAt?: string
  windowStart?: string
  windowEnd?: string
  azimuthDegrees?: number
  altitudeDegrees?: number
  reason: string
}

export interface AstronomyEventsResponse {
  events: AstronomyEvent[]
  range: { from: string; to: string }
  locationVisibility: 'partial' | 'location_required'
  sources?: AstronomyEventSourceStatus[]
}


export interface AstronomyEventSourceStatus {
  code: string
  name: string
  url: string
  lastAttemptAt?: string
  lastSuccessAt?: string
  success?: boolean
  recordsWritten: number
  error?: string
  coverageStart?: string
  coverageEnd?: string
}

/**
 * 拉取天象事件。未提供坐标时返回全球日历（locationVisibility=location_required）；
 * 提供坐标时返回带 local 字段的事件（locationVisibility=partial）。
 */
export async function fetchAstronomyEvents(params: {
  from?: string
  to?: string
  latitude?: number
  longitude?: number
  timezone?: string
}, signal?: AbortSignal): Promise<AstronomyEventsResponse> {
  const search = new URLSearchParams()
  if (params.from) search.set('from', params.from)
  if (params.to) search.set('to', params.to)
  if (params.latitude !== undefined) search.set('latitude', String(params.latitude))
  if (params.longitude !== undefined) search.set('longitude', String(params.longitude))
  if (params.timezone) search.set('timezone', params.timezone)
  return requestJSON<AstronomyEventsResponse>(`/api/v1/astronomy/events?${search}`, signal)
}

/** 图像墙由 AURORA 后端聚合；NASA APOD 的 Key 从不进入浏览器。 */
export interface ImageWindow {
  id: string
  sourceId: string
  sourceName: string
  title: string
  publishedAt?: string
  imageUrl?: string
  thumbnailUrl?: string
  mediaType: 'image' | 'video'
  credit: string
  licenseNote?: string
  sourceUrl: string
  hdUrl?: string
  selectionMode: 'daily' | 'rotating' | 'curated'
  summary?: string
  status: 'ready' | 'error'
  error?: string
  isFallback?: boolean
}

export interface ImageWall {
  recent: ImageWindow[]
  collection: ImageWindow[]
  generatedAt: string
}

export function fetchImageWall(signal?: AbortSignal) {
  return requestJSON<ImageWall>('/api/v1/astronomy/image-wall', signal)
}
