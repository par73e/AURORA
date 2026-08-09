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

export function fetchObserverPlace(latitude: number, longitude: number, signal?: AbortSignal) {
  const search = new URLSearchParams({
    latitude: latitude.toFixed(6),
    longitude: longitude.toFixed(6),
  })
  return requestJSON<ObserverPlace>(`/api/v1/location/reverse?${search}`, signal)
}

export interface ObservingConditions {
  retrievedAt: string
  timezone: string
  source: string
  current: {
    time: string
    temperature: number
    dewPoint: number
    cloudCover: number
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
}

export function fetchObservingConditions(latitude: number, longitude: number, signal?: AbortSignal) {
  const search = new URLSearchParams({ latitude: latitude.toFixed(6), longitude: longitude.toFixed(6) })
  return requestJSON<ObservingConditions>(`/api/v1/astronomy/conditions?${search}`, signal)
}
