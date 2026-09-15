<!--
THESIS: SKY is a local observing decision room, not a weather dashboard or a generic star-map clone.
OWN-WORLD: midnight blue-black, lunar white, horizon amber, flat instrument rails, and one continuous sky field.
STORY: establish local conditions, inspect the sky and its 24-hour rhythm, then plan around the next event.
FIRST VIEWPORT: a permanent left observatory rail frames a wide lunar condition reading and a plain-language verdict.
FORM: desktop field observatory; four focused workspaces share one clock, one location, and evidence boundaries.
-->
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { fetchObserverPlace, searchObserverPlaces, fetchObservingConditions, fetchMoonDay, fetchLightPollution, fetchAstronomyEvents, fetchImageWall, type ObservingConditions, type MoonDay, type LightPollution, type AstronomyEvent, type AstronomyEventSourceStatus, type ImageWall, type ObserverPlaceCandidate } from '../api'
import { analyzeNight, bearing, bodies, calculateFixedObjectPosition, calculatePosition, calculateTrack, calculateTwilight, dateFromZonedLocalTime, daylightFactor, moonPhase, observeTips, observingStatus, upcomingMoonPhases, zonedDateAtMinute, zonedDateKey, zonedDateKeyAfterDays, zonedMinuteOfDay, type BodyId, type BodyTrack, type NightAnalysis } from '../astronomy'
import { conditionDescription, forecastHoursThroughTomorrow, weatherGlyph } from '../observatoryWeather'
import { projectAltitudeGuide, projectHorizontalDirection, type SkyCamera } from '../skyProjection'
import { easeOutExpo, normalizeAzimuth, shortestAzimuthDelta, skyTurnDuration } from '../skyMotion'
import { constellationLines, milkyWayCenterline, skyCatalog, type SkyCatalogObject } from '../skyCatalog'
import moonNearsideTexture from '../assets/solar/2k_moon.jpg'
import AuroraBrand from './AuroraBrand.vue'
import ObservatoryClock from './ObservatoryClock.vue'

type SkyPage = 'conditions' | 'sky' | 'events' | 'daily-image'

const emit = defineEmits<{ home: [] }>()

const menu = [
  { id: 'conditions' as const, numeral: 'Ⅰ', label: '观星条件', en: 'CONDITIONS', icon: 'gauge' },
  { id: 'sky' as const, numeral: 'Ⅱ', label: '星图', en: 'SKY MAP', icon: 'constellation' },
  { id: 'events' as const, numeral: 'Ⅲ', label: '天象', en: 'SKY EVENTS', icon: 'calendar' },
  { id: 'daily-image' as const, numeral: 'Ⅳ', label: '每日一图', en: 'DAILY IMAGE', icon: 'image' },
]

function pageFromHash(): SkyPage {
  const hash = window.location.hash
  if (hash === '#astronomy-daily-image') return 'daily-image'
  if (hash === '#astronomy-events') return 'events'
  if (hash === '#astronomy-sky' || hash === '#astronomy-tonight' || hash === '#astronomy-windows' || hash === '#astronomy-targets') return 'sky'
  return 'conditions'
}

const activePage = ref<SkyPage>(pageFromHash())
const now = ref(new Date())
// 仿真时钟跟随真实时间：进度条每分钟前进一格、行星位置随之移动。
// 用户拖动进度条时暂停跟随（预览未来/过去时刻）；点"现在"恢复。
const followingRealTime = ref(true)
const minuteOfDay = ref(now.value.getHours() * 60 + now.value.getMinutes())
const skyViewAzimuth = ref(180)
const skyViewDragging = ref(false)
const skyViewAutoTurning = ref(false)
const locationLabel = ref('等待位置授权')
const latitude = ref<number | null>(null)
const longitude = ref<number | null>(null)
const locationStatus = ref<'idle' | 'locating' | 'resolving' | 'located' | 'partial' | 'denied' | 'unavailable'>('idle')
const showLocationEditor = ref(false)
const locationControl = ref<HTMLElement | null>(null)
const locationTrigger = ref<HTMLButtonElement | null>(null)
const skySearchControl = ref<HTMLElement | null>(null)
const showSkySearchResults = ref(false)
const locationQuery = ref('')
const manualLatitude = ref('')
const manualLongitude = ref('')
const locationSearchStatus = ref<'idle' | 'loading' | 'ready' | 'error'>('idle')
const locationSearchResults = ref<ObserverPlaceCandidate[]>([])
const locationFormError = ref('')
const conditions = ref<ObservingConditions | null>(null)
const conditionsStatus = ref<'idle' | 'loading' | 'ready' | 'error'>('idle')
// 后端数据源：月相每日、逐小时评分、光污染按地点；任一失败均回退本地计算/诚实占位。
const moonDay = ref<MoonDay | null>(null)
const lightPollution = ref<LightPollution | null>(null)
const expandedBodyId = ref<BodyId | null>(null)
const expandedEventId = ref<string | null>(null)
const showAllCuratedEvents = ref(false)
const imageWall = ref<ImageWall | null>(null)
const imageWallStatus = ref<'idle' | 'loading' | 'ready' | 'error'>('idle')
const recentImageWindows = computed(() => imageWall.value?.recent.filter((window) => window.mediaType === 'image') ?? [])
const skyLeaving = ref(false)
const moonCanvas = ref<HTMLCanvasElement | null>(null)
let minuteClock: number | undefined
let minuteAnimationFrame: number | undefined
let timeScrubAnimationFrame: number | undefined
let pendingMinuteOfDay: number | undefined
let locationRevision = 0
let locationLookupController: AbortController | undefined
let locationSearchController: AbortController | undefined
let conditionsController: AbortController | undefined
let homeExitTimer: number | undefined
let moonTexturePixels: ImageData | undefined
let moonTextureWidth = 0
let moonTextureHeight = 0
let skyViewStartX = 0
let skyViewStartAzimuth = 180
let skyViewAnimationFrame: number | undefined
let pendingSkyViewAzimuth: number | undefined
let skyViewTurnAnimationFrame: number | undefined
let imageWallController: AbortController | undefined
const skyViewFieldOfView = 120
const fallbackTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'

const skyCamera = computed<SkyCamera>(() => ({
  heading: skyViewAzimuth.value,
  horizontalFov: skyViewFieldOfView,
}))

const activeCoordinates = computed(() => latitude.value != null && longitude.value != null ? { latitude: latitude.value, longitude: longitude.value } : null)
// Open-Meteo timezone=auto 返回定位点的 IANA 时区；数据到达前才暂用设备时区。
const observatoryTimezone = computed(() => conditions.value?.timezone || fallbackTimezone)
const simulatedTime = computed(() => zonedDateAtMinute(now.value, minuteOfDay.value, observatoryTimezone.value))
const simulatedDateKey = computed(() => zonedDateKey(simulatedTime.value, observatoryTimezone.value))
const simulatedDayAnchor = computed(() => dateFromZonedLocalTime(`${simulatedDateKey.value}T12:00`, observatoryTimezone.value))
const tomorrowDateKey = computed(() => zonedDateKeyAfterDays(now.value, observatoryTimezone.value, 1))
const tomorrowDayAnchor = computed(() => dateFromZonedLocalTime(`${tomorrowDateKey.value}T12:00`, observatoryTimezone.value))
const timeLabel = computed(() => formatTime(simulatedTime.value))
const scrubFraction = computed(() => (minuteOfDay.value / 1439).toFixed(4))
const coordinateLabel = computed(() => activeCoordinates.value
  ? `${activeCoordinates.value.latitude >= 0 ? '北纬' : '南纬'} ${Math.abs(activeCoordinates.value.latitude).toFixed(1)}° · ${activeCoordinates.value.longitude >= 0 ? '东经' : '西经'} ${Math.abs(activeCoordinates.value.longitude).toFixed(1)}°`
  : '允许定位后生成本地数据')
const moon = computed(() => moonPhase(now.value))
const elevation = computed(() => conditions.value?.elevation ?? 0)
// 升落、整日采样与中天只随“当地日期/地点”变化，不应在时间条每移动一分钟时重算。
const dailyTracks = computed<BodyTrack[]>(() => activeCoordinates.value
  ? bodies.map((body) => calculateTrack(body, simulatedDayAnchor.value, activeCoordinates.value!.latitude, activeCoordinates.value!.longitude, elevation.value, observatoryTimezone.value))
  : [])
// 拖动时间条时只计算 9 个天体在当前时刻的位置，再复用当天的轨迹与升落结果。
const tracks = computed<BodyTrack[]>(() => dailyTracks.value.map((track) => ({
  ...track,
  ...calculatePosition(track, simulatedTime.value, activeCoordinates.value!.latitude, activeCoordinates.value!.longitude, elevation.value),
})))
const moonTrack = computed(() => tracks.value.find((track) => track.id === 'moon') ?? null)
// 月相面板：后端每日数据优先（相位/亮度/月龄/月出月落），未就绪或失败时退回本地 Astronomy Engine。
const moonPanel = computed(() => {
  const day = moonDay.value
  if (!day) {
    return {
      phase: moon.value.phase,
      illumination: moon.value.illumination,
      age: moon.value.age,
      label: moon.value.label,
      moonrise: moonTrack.value?.rise ?? null,
      moonset: moonTrack.value?.set ?? null,
    }
  }
  return {
    phase: day.phase,
    illumination: day.illumination,
    age: day.age,
    label: day.label,
    moonrise: day.moonrise ? new Date(day.moonrise * 1000) : null,
    moonset: day.moonset ? new Date(day.moonset * 1000) : null,
  }
})
const moonIllumination = computed(() => (moonPanel.value.illumination * 100).toFixed(1))
// 这些轨迹几何只随日期/地点变化，不放进每次拖动进度条的热路径。
const visibleSegmentMap = computed(() => new Map(
  dailyTracks.value.map((track) => [track.id, visibleSegments(track)]),
))
const expandedWaveGeometry = computed(() => {
  const track = expandedBodyId.value ? dailyTracks.value.find((item) => item.id === expandedBodyId.value) : null
  return track ? altitudeCurve(track, ALTITUDE_CHART_ROW) : null
})
const expandedWaveCurrent = computed(() => {
  const track = expandedBodyId.value ? dailyTracks.value.find((item) => item.id === expandedBodyId.value) : null
  if (!track) return null
  const altitude = idealWaveAltitude(track, minuteOfDay.value)
  return {
    x: minuteOfDay.value / 1439 * ALTITUDE_CHART_ROW.width,
    y: altitudeToY(altitude, ALTITUDE_CHART_ROW),
    aboveHorizon: altitude > 0,
  }
})
// 收起态时间条上的"当前时刻"小点：与展开波形同一平滑正弦源（idealWaveAltitude > 0 判定地平线上/下，x 与 minuteOfDay/1439 同源）
const railCurrentMarkers = computed(() => new Map(
  tracks.value.map((track) => [
    track.id,
    { left: minuteOfDay.value / 1439 * 100, aboveHorizon: idealWaveAltitude(track, minuteOfDay.value) > 0 },
  ]),
))
const twilight = computed(() => activeCoordinates.value ? calculateTwilight(simulatedDayAnchor.value, activeCoordinates.value.latitude, activeCoordinates.value.longitude, elevation.value, observatoryTimezone.value) : null)
// 今夜夜空分析：天文夜窗口、无月黑夜与银河核心可见时段（纯本地星历）。
const nightAnalysis = computed<NightAnalysis | null>(() => activeCoordinates.value ? analyzeNight(now.value, activeCoordinates.value.latitude, activeCoordinates.value.longitude, elevation.value, observatoryTimezone.value) : null)
const tomorrowNightAnalysis = computed<NightAnalysis | null>(() => activeCoordinates.value ? analyzeNight(tomorrowDayAnchor.value, activeCoordinates.value.latitude, activeCoordinates.value.longitude, elevation.value, observatoryTimezone.value) : null)
function windowRange(window: NightAnalysis['astronomicalNight']) {
  return window ? `${formatTime(window.start)} – ${formatTime(window.end)}` : '—'
}
// 白昼系数 0（深夜）→ 1（正午）：以当天当地日出/日落为边界，天文晨光/昏影为渐变过渡带；星图视场与天体淡出共用。
const daylight = computed(() => twilight.value ? daylightFactor(twilight.value, simulatedTime.value) : 0)
const visibleSkyBodies = computed(() => tracks.value.filter((track) => track.visible))
const horizonBodies = computed(() => visibleSkyBodies.value.filter((track) => projectHorizontalDirection(track.azimuth, track.altitude, skyCamera.value).inViewport))
const selectedCatalogId = ref<string | null>(null)
const skySearchQuery = ref('')
const catalogPositions = computed(() => {
  const coords = activeCoordinates.value
  if (!coords) return new Map<string, ReturnType<typeof calculateFixedObjectPosition>>()
  return new Map(skyCatalog.map((item) => [item.id, calculateFixedObjectPosition(item.raHours, item.decDegrees, simulatedTime.value, coords.latitude, coords.longitude, elevation.value)]))
})
const projectedCatalog = computed(() => skyCatalog.flatMap((item) => {
  const position = catalogPositions.value.get(item.id)
  if (!position?.visible) return []
  const projection = projectHorizontalDirection(position.azimuth, position.altitude, skyCamera.value)
  return projection.inViewport ? [{ item, position, projection }] : []
}))
const visibleCatalogStars = computed(() => projectedCatalog.value.filter(({ item }) => item.kind === 'star'))
const visibleMessierObjects = computed(() => projectedCatalog.value.filter(({ item }) => item.kind === 'messier'))
interface SkySearchResult {
  key: string
  name: string
  nameEn: string
  group: string
  visible: boolean
  bodyId?: BodyId
  catalogObject?: SkyCatalogObject
}
const skySearchResults = computed<SkySearchResult[]>(() => {
  const query = skySearchQuery.value.trim().toLocaleLowerCase()
  if (!query) return []
  const bodyResults = tracks.value
    .filter((item) => `${item.name} ${item.nameEn}`.toLocaleLowerCase().includes(query))
    .map((item) => ({ key: `body-${item.id}`, name: item.name, nameEn: item.nameEn, group: '太阳系天体', visible: item.visible, bodyId: item.id }))
  const catalogResults = skyCatalog
    .filter((item) => `${item.name} ${item.nameEn} ${item.constellation}`.toLocaleLowerCase().includes(query))
    .map((item) => ({ key: `catalog-${item.id}`, name: item.name, nameEn: item.nameEn, group: item.constellation, visible: catalogPositions.value.get(item.id)?.visible ?? false, catalogObject: item }))
  return [...bodyResults, ...catalogResults].slice(0, 8)
})
const constellationGeometry = computed(() => constellationLines.map((constellation) => {
  const paths = constellation.segments.flatMap(([fromId, toId]) => {
    const from = projectedCatalog.value.find(({ item }) => item.id === fromId)?.projection
    const to = projectedCatalog.value.find(({ item }) => item.id === toId)?.projection
    if (!from || !to || Math.abs(from.x - to.x) > .35) return []
    return [`M ${(from.x * 1000).toFixed(2)} ${(from.y * 1000).toFixed(2)} L ${(to.x * 1000).toFixed(2)} ${(to.y * 1000).toFixed(2)}`]
  })
  const points = projectedCatalog.value.filter(({ item }) => item.constellation === constellation.name).map(({ projection }) => projection)
  const label = points.length ? { x: points.reduce((sum, point) => sum + point.x, 0) / points.length, y: points.reduce((sum, point) => sum + point.y, 0) / points.length } : null
  return { ...constellation, paths, label }
}).filter((item) => item.paths.length))
const milkyWayPaths = computed(() => {
  const coords = activeCoordinates.value
  if (!coords) return []
  const paths: string[] = []
  let run: Array<{ x: number; y: number }> = []
  const flush = () => {
    if (run.length > 1) paths.push(`M ${run.map((point) => `${(point.x * 1000).toFixed(2)} ${(point.y * 1000).toFixed(2)}`).join(' L ')}`)
    run = []
  }
  for (const point of milkyWayCenterline) {
    const position = calculateFixedObjectPosition(point.raHours % 24, point.decDegrees, simulatedTime.value, coords.latitude, coords.longitude, elevation.value)
    const projection = projectHorizontalDirection(position.azimuth, position.altitude, skyCamera.value)
    if (!position.visible || !projection.inViewport || (run.at(-1) && Math.abs(projection.x - run.at(-1)!.x) > .3)) {
      flush()
      if (!position.visible || !projection.inViewport) continue
    }
    run.push(projection)
  }
  flush()
  return paths
})
const altitudeGuides = computed(() => [30, 60].map((altitude) => projectAltitudeGuide(altitude, skyCamera.value, .75)))
const horizonFieldStyle = computed(() => ({
  '--sky-daylight': String(daylight.value),
}))
const selectedSkyTrajectory = computed(() => {
  const track = expandedBodyId.value ? tracks.value.find((item) => item.id === expandedBodyId.value) : null
  if (!track) return null
  const currentTime = simulatedTime.value.getTime()
  const current = { at: simulatedTime.value, altitude: track.altitude, azimuth: track.azimuth }
  const past = [...track.samples.filter((sample) => sample.at.getTime() < currentTime), current]
  const future = [current, ...track.samples.filter((sample) => sample.at.getTime() > currentTime)]
  return {
    tint: track.tint,
    pastPaths: projectSkyTrackRuns(past, skyCamera.value),
    futurePaths: projectSkyTrackRuns(future, skyCamera.value),
  }
})
const skyViewDirection = computed(() => bearing(skyViewAzimuth.value))
const skyViewHeadingLabel = computed(() => `${skyViewDirection.value} ${skyViewAzimuth.value.toFixed(2)}°`)
const directionNames: Record<number, string> = { 0:'北', 45:'东北', 90:'东', 135:'东南', 180:'南', 225:'西南', 270:'西', 315:'西北' }
const skyHeadingTicks = computed(() => {
  const halfView = skyViewFieldOfView / 2
  const firstDegree = Math.floor((skyViewAzimuth.value - halfView) / 5) * 5
  const lastDegree = Math.ceil((skyViewAzimuth.value + halfView) / 5) * 5
  const ticks = []
  for (let rawDegree = firstDegree; rawDegree <= lastDegree; rawDegree += 5) {
    const relativeDegree = rawDegree - skyViewAzimuth.value
    const roundedDegree = Math.round(normalizeAzimuth(rawDegree))
    const ratio = Math.max(-1, Math.min(1, relativeDegree / halfView))
    const direction = directionNames[roundedDegree] ?? ''
    const isDirection = direction !== ''
    const hasLabel = isDirection
      ? Math.abs(relativeDegree) > 12 && Math.abs(relativeDegree) <= halfView - 3
      : roundedDegree % 30 === 0 && Math.abs(relativeDegree) > 13
    ticks.push({
      key: rawDegree,
      position: 50 + ratio * 49,
      lift: 4 + (1 - ratio * ratio) * 34,
      major: roundedDegree % 10 === 0 || isDirection,
      direction: isDirection,
      label: hasLabel ? (direction || String(roundedDegree || 360).padStart(3, '0')) : '',
    })
  }
  return ticks
})
// 顶部评分只回答“今夜天文夜里，最值得安排的时段条件如何”。
// 白天和星图预览不参与这张卡的分数，避免把“此刻天气”误解为“此刻适合观星”。
type TonightScore = NonNullable<ObservingConditions['scores']>[number]
const tonightScore = computed<TonightScore | null>(() => {
  const night = nightAnalysis.value?.astronomicalNight
  const scores = conditions.value?.scores ?? []
  if (!night || !scores.length) return null
  const candidates = scores.filter((item) => {
    const at = dateFromZonedLocalTime(item.time, observatoryTimezone.value).getTime()
    return at >= night.start.getTime() && at <= night.end.getTime()
  })
  if (!candidates.length) return null
  return candidates.reduce((best, item) => item.score > best.score ? item : best)
})
function bestScoreForNight(night: NightAnalysis['astronomicalNight'] | undefined, localDateKey?: string) {
  const scores = conditions.value?.scores ?? []
  if (!night || !scores.length) return null
  const candidates = scores.filter((item) => {
    if (localDateKey && item.time.slice(0, 10) !== localDateKey) return false
    const at = dateFromZonedLocalTime(item.time, observatoryTimezone.value).getTime()
    return at >= night.start.getTime() && at <= night.end.getTime()
  })
  return candidates.reduce<TonightScore | null>((best, item) => !best || item.score > best.score ? item : best, null)
}
// “明日夜间”只使用明天当地自然日内的夜间小时，不把后天凌晨并入明日评分。
const tomorrowScore = computed(() => bestScoreForNight(tomorrowNightAnalysis.value?.astronomicalNight, tomorrowDateKey.value))
const hasTonightScoreDetails = computed(() => Boolean(tonightScore.value?.factors && tonightScore.value?.weather))
const scoreWeather = computed(() => hasTonightScoreDetails.value ? tonightScore.value?.weather ?? null : null)
const displayedConditions = computed<ObservingConditions['current'] | ObservingConditions['hourly'][number] | null>(() => {
  const forecast = conditions.value
  if (!forecast) return null
  if (followingRealTime.value) return forecast.current
  const target = simulatedTime.value.getTime()
  return forecast.hourly.reduce((nearest, item) => {
    const itemAt = dateFromZonedLocalTime(item.time, observatoryTimezone.value).getTime()
    const nearestAt = dateFromZonedLocalTime(nearest.time, observatoryTimezone.value).getTime()
    return Math.abs(itemAt - target) < Math.abs(nearestAt - target) ? item : nearest
  }, forecast.hourly[0] ?? null as ObservingConditions['hourly'][number] | null)
})
const conditionsMomentLabel = computed(() => followingRealTime.value ? '当前时刻' : '预览时刻')
const currentAir = computed(() => {
  const forecast = conditions.value
  const weather = displayedConditions.value
  if (!forecast?.airQuality.length || !weather) return null
  const target = dateFromZonedLocalTime(weather.time, observatoryTimezone.value).getTime()
  if (!Number.isFinite(target)) return forecast.airQuality[0]
  return forecast.airQuality.reduce((nearest, item) => {
    const itemTime = dateFromZonedLocalTime(item.time, observatoryTimezone.value).getTime()
    const nearestTime = dateFromZonedLocalTime(nearest.time, observatoryTimezone.value).getTime()
    return Math.abs(itemTime - target) < Math.abs(nearestTime - target) ? item : nearest
  })
})
const scoreVerdict = computed(() => hasTonightScoreDetails.value ? tonightScore.value?.verdict ?? '正在计算今夜条件' : (tonightScore.value ? '请重启本地后端以更新评分模型' : conditionsStatus.value === 'error' ? '天气源暂不可用' : '正在计算今夜条件'))
const scorePanel = computed(() => hasTonightScoreDetails.value ? tonightScore.value?.score ?? null : null)
const scoreWindowLabel = computed(() => hasTonightScoreDetails.value && tonightScore.value ? `今夜最佳 · ${tonightScore.value.time.slice(11, 16)}` : '今夜天文夜')
// 评分因子直接使用“今夜最佳时段”的后端分解，确保卡片、文案和分数同源。
interface ScoreFactorPart {
  key: string
  label: string
  value: number // 仅展示负数扣分
  kind: 'penalty'
}
const scoreFactors = computed<ScoreFactorPart[] | null>(() => {
  const score = tonightScore.value
  if (!score || !hasTonightScoreDetails.value) return null
  const parts: ScoreFactorPart[] = []
  if (score.factors.cloudPenalty > 0) parts.push({ key: 'cloud', label: '云层遮挡', value: -score.factors.cloudPenalty, kind: 'penalty' })
  if (score.factors.precipitationPenalty > 0) parts.push({ key: 'precip', label: '降水', value: -score.factors.precipitationPenalty, kind: 'penalty' })
  if (score.factors.visibilityPenalty > 0) parts.push({ key: 'visibility', label: '通透度', value: -score.factors.visibilityPenalty, kind: 'penalty' })
  if (score.factors.moonPenalty > 0) parts.push({ key: 'moon', label: '月光', value: -score.factors.moonPenalty, kind: 'penalty' })
  if (score.factors.aerosolPenalty > 0) parts.push({ key: 'aerosol', label: '气溶胶', value: -score.factors.aerosolPenalty, kind: 'penalty' })
  if (score.factors.windPenalty > 0) parts.push({ key: 'wind', label: '风与阵风', value: -score.factors.windPenalty, kind: 'penalty' })
  if (score.factors.dewPenalty > 0) parts.push({ key: 'dew', label: '结露风险', value: -score.factors.dewPenalty, kind: 'penalty' })
  if (score.factors.weatherPenalty > 0) parts.push({ key: 'weather', label: '天气现象', value: -score.factors.weatherPenalty, kind: 'penalty' })
  return parts
})
function factorBarWidth(part: ScoreFactorPart) {
  // 以当前扣分项中绝对值最大者为满刻度，让用户快速辨认主要限制因素。
  const scale = scoreFactors.value?.reduce((max, item) => Math.max(max, Math.abs(item.value)), 0) ?? 30
  return `${Math.max(6, Math.abs(part.value) / Math.max(1, scale) * 100)}%`
}
// 今夜建议（动态推荐）：逐小时评分 × 天文夜窗口 × 逐目标月光/曙暮光 → 最佳窗口与推荐目标。
// 推荐目标 → 星图定位：切到星图页、平滑转动罗盘到该天体当前方位、并展开该行详情。
function locateRecommendedBody(body: BodyId) {
  const track = tracks.value.find((item) => item.id === body)
  // 与星图页"在星图定位"一致：当前在地平线以下的天体无法定位，直接忽略。
  if (!track || !track.visible) return
  if (activePage.value !== 'sky') selectPage('sky')
  expandedBodyId.value = body
  revealSkyDirection(track.azimuth)
}
const recommendation = computed(() => {
  const hourly = conditions.value?.scores ?? []
  // 天文夜窗口必须用 analyzeNight 的跨午夜结果（今晚昏影 → 次日晨光）；
  // calculateTwilight 的 astronomicalDawn 是当天早晨的晨光，直接用会得到反向窗口。
  const night = nightAnalysis.value?.astronomicalNight
  const coords = activeCoordinates.value
  if (!tracks.value.length || !night || !coords) return null
  const nightStart = night.start
  const nightEnd = night.end
  // 仅认天文夜内已覆盖的逐小时预报；绝不以白天的高分代替“今夜”结果。
  const darkHours = hourly.filter((item) => {
    const at = dateFromZonedLocalTime(item.time, observatoryTimezone.value)
    return at.getTime() >= nightStart.getTime() && at.getTime() <= nightEnd.getTime()
  })
  const weatherAvailable = darkHours.length > 0
  const weatherCoverage = weatherAvailable ? 'covered' : hourly.length ? 'outside' : 'unavailable'

  // 天气不可用或预报未覆盖今夜时，窗口信息诚实退化为天文夜时段本身。
  let bestHour: (typeof hourly)[number] | null = null
  let windowLabel: string
  if (weatherAvailable) {
    const bestScore = Math.max(...darkHours.map((item) => item.score))
    if (bestScore >= 40) {
      bestHour = darkHours.find((item) => item.score === bestScore) ?? null
      windowLabel = bestHour ? `最佳窗口 ${bestHour.time.slice(11, 16)} · 评分 ${bestScore}` : '今夜未见理想窗口'
    } else {
      windowLabel = '今夜天气与月光条件有限'
    }
  } else {
    windowLabel = hourly.length ? `今夜天文夜暂未进入天气预报范围 · ${formatTime(nightStart)} – ${formatTime(nightEnd)}` : `天气源暂不可用，天文夜 ${formatTime(nightStart)} – ${formatTime(nightEnd)}`
  }

  // 逐目标：在天文夜窗口内每 15 分钟用星历重算 暗夜 × 高度 × 月光，取综合最佳时刻。
  // 用 calculatePosition 而非 track.samples——samples 只覆盖当天 0–24 点，
  // 会漏掉跨午夜窗口凌晨（次日 0:00–晨光）的高质量时段。
  const moonBody = bodies.find((body) => body.id === 'moon')!
  const targets = bodies
    .filter((body) => body.id !== 'sun')
    .map((body) => {
      let best: { at: Date; altitude: number; magnitude: number | null; score: number } | null = null
      for (let sampleAt = nightStart.getTime(); sampleAt <= nightEnd.getTime(); sampleAt += 15 * 60_000) {
        const at = new Date(sampleAt)
        const position = calculatePosition(body, at, coords.latitude, coords.longitude, elevation.value)
        if (position.altitude <= 10) continue
        const moonPosition = calculatePosition(moonBody, at, coords.latitude, coords.longitude, elevation.value)
        const moonInterference = moonPosition.altitude > 0 ? moonPanel.value.illumination * .65 : 0
        const score = position.altitude * (1 - moonInterference)
        if (!best || score > best.score) best = { at, altitude: position.altitude, magnitude: position.magnitude, score }
      }
      return { body, best }
    })
    .filter((entry) => entry.best)
    .sort((a, b) => (b.best?.score ?? -Infinity) - (a.best?.score ?? -Infinity))
    .slice(0, 3)
    .map((entry) => {
      const best = entry.best!
      const moonPosition = calculatePosition(moonBody, best.at, coords.latitude, coords.longitude, elevation.value)
      const moonNote = moonPosition.altitude > 0 ? (moonPanel.value.illumination > .5 ? '月光较强' : '月光较弱') : '无月光干扰'
      // 定位按钮的可用性由"当前模拟时刻"的地平高度决定（此刻能转到才有意义）；
      // 最佳时刻可能在未来凌晨，此刻在地平线下时按钮应置灰。
      const current = tracks.value.find((item) => item.id === entry.body.id)
      return {
        id: entry.body.id,
        name: entry.body.name,
        glyph: entry.body.glyph,
        tint: entry.body.tint,
        summary: `最佳 ${formatTime(best.at)} · ${Math.round(best.altitude)}° 高 · ${moonNote}`,
        tip: observeTips[entry.body.id],
        visible: current?.visible ?? false,
      }
    })
  return { window: bestHour, windowLabel, targets, weatherAvailable, weatherCoverage }
})
// 预报矩阵从用户所在地当前整点开始，保留今天剩余小时，并显示到明日 24:00。
const hourlyForecast = computed(() => forecastHoursThroughTomorrow(
  conditions.value?.hourly ?? [],
  now.value,
  observatoryTimezone.value,
))
const forecastDateGroups = computed(() => {
  const groups: Array<{ date: string; hours: number }> = []
  for (const hour of hourlyForecast.value) {
    const date = forecastDay(hour.time)
    const previous = groups.at(-1)
    if (previous?.date === date) previous.hours += 1
    else groups.push({ date, hours: 1 })
  }
  return groups
})
const airQualityByTime = computed(() => new Map((conditions.value?.airQuality ?? []).map((item) => [item.time, item])))
const moonEvents = computed(() => upcomingMoonPhases(now.value).slice(0, 4))
const nextMoonEvent = computed(() => moonEvents.value[0] ?? null)
const curatedEvents = computed(() => astronomyEvents.value
  .filter((event) => {
    const start = new Date(event.startsAt).getTime()
    return start >= now.value.getTime() - 36 * 60 * 60_000 && start <= now.value.getTime() + 31 * 24 * 60 * 60_000
  }))
const visibleCuratedEvents = computed(() => {
  if (showAllCuratedEvents.value) return curatedEvents.value
  // 近地小天体数量很多，默认摘要先保留对普通观测者更可执行的事件；展开后仍按日期完整展示。
  const highlights = curatedEvents.value.filter((event) => event.kind !== 'small_body_close_approach')
  const closeApproaches = curatedEvents.value.filter((event) => event.kind === 'small_body_close_approach')
  return [...highlights, ...closeApproaches].slice(0, 5)
})
const hasHiddenCuratedEvents = computed(() => curatedEvents.value.length > 5)

function formatTime(value: Date | null) {
  if (!value) return '—'
  return new Intl.DateTimeFormat('zh-CN', { timeZone: observatoryTimezone.value, hour: '2-digit', minute: '2-digit', hour12: false }).format(value)
}

function formatEventDay(value: Date) {
  return new Intl.DateTimeFormat('en', { timeZone: observatoryTimezone.value, day: '2-digit' }).format(value)
}

function formatEventMonth(value: Date) {
  return new Intl.DateTimeFormat('en', { timeZone: observatoryTimezone.value, month: 'short' }).format(value).toUpperCase()
}

function formatEventDate(value: string | Date) {
  const at = typeof value === 'string' ? new Date(value) : value
  return new Intl.DateTimeFormat('zh-CN', { timeZone: observatoryTimezone.value, month: 'long', day: 'numeric' }).format(at)
}

function formatEventMoment(value?: string) {
  if (!value) return '—'
  const at = new Date(value)
  if (Number.isNaN(at.getTime())) return '—'
  return new Intl.DateTimeFormat('zh-CN', {
    timeZone: observatoryTimezone.value,
    month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit', hour12: false,
  }).format(at)
}

function formatSourceDate(value?: string) {
  if (!value) return '尚无成功记录'
  const at = new Date(value)
  return Number.isNaN(at.getTime()) ? value : new Intl.DateTimeFormat('zh-CN', { timeZone: observatoryTimezone.value, month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit', hour12: false }).format(at)
}

function sourceStatusMessage(source: AstronomyEventSourceStatus) {
  if (source.success !== false) return `最近同步 ${formatSourceDate(source.lastSuccessAt)}`
  if (source.code === 'imo_meteor_calendar' && source.error?.includes('not a PDF')) return 'IMO 年度 PDF 当前返回网页；继续使用上次缓存'
  if (source.code === 'imo_meteor_calendar') return 'IMO 年度日历当前不可用；继续使用上次缓存'
  return '部分资料同步失败；继续使用上次缓存'
}

function formatImageWindowDate(value?: string) {
  if (!value) return '发布日期未提供'
  const date = new Date(`${value}T12:00:00Z`)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', { month: 'long', day: 'numeric', year: 'numeric', timeZone: 'UTC' }).format(date)
}

function eventPrecision(event: AstronomyEvent) {
  const precision = event.geometry.precision
  if (precision === 'refined') return '日采样插值精化'
  if (precision === 'daily_sample') return '日采样候选'
  if (typeof precision === 'string') return precision.replaceAll('_', ' ')
  return event.origin === 'computed' ? 'AURORA 计算' : '来源资料'
}

function eventVisibilityTone(event: AstronomyEvent) {
  switch (event.local?.status) {
    case 'observable':
      return 'observable'
    case 'limited':
      return 'limited'
    case 'not_visible':
      return 'not-visible'
    case 'non_visual':
      return 'non-visual'
    case 'not_calculated':
      return 'unknown'
    default:
      return 'pending'
  }
}

function eventVisibilityLabel(event: AstronomyEvent) {
  switch (event.local?.status) {
    case 'observable':
      return '适合观看'
    case 'limited':
      return '条件有限'
    case 'not_visible':
      return '本地不可见'
    case 'non_visual':
      return '非视觉事件'
    case 'not_calculated':
      return '待计算'
    default:
      return '需要定位'
  }
}

function toggleEvent(event: AstronomyEvent) {
  expandedEventId.value = expandedEventId.value === event.id ? null : event.id
}

function toggleAllCuratedEvents() {
  showAllCuratedEvents.value = !showAllCuratedEvents.value
}

function formatDistance(distanceAu: number | null) {
  if (distanceAu == null) return '—'
  if (distanceAu < .01) return `${Math.round(distanceAu * 149_597_870)} km`
  return `${distanceAu.toFixed(2)} AU`
}

function forecastDay(value: string) {
  const [, month, day] = value.match(/\d{4}-(\d{2})-(\d{2})/) ?? []
  return month && day ? `${Number(month)}月${Number(day)}日` : value.slice(0, 10)
}

function airFor(time: string) {
  return airQualityByTime.value.get(time)
}

function forecastTone(metric: 'weather' | 'cloud' | 'humidity' | 'precipitation' | 'wind' | 'visibility' | 'aod' | 'dew', hour: ObservingConditions['hourly'][number]) {
  const air = airFor(hour.time)
  if (metric === 'weather') return hour.weatherCode <= 1 ? 'good' : hour.weatherCode <= 3 ? 'caution' : 'poor'
  if (metric === 'cloud') return hour.cloudCover <= 20 ? 'good' : hour.cloudCover <= 55 ? 'caution' : 'poor'
  if (metric === 'humidity') return hour.humidity <= 70 ? 'good' : hour.humidity <= 85 ? 'caution' : 'poor'
  if (metric === 'precipitation') return hour.precipitation === 0 ? 'good' : hour.precipitation <= .2 ? 'caution' : 'poor'
  if (metric === 'wind') return hour.windSpeed <= 15 ? 'good' : hour.windSpeed <= 25 ? 'caution' : 'poor'
  if (metric === 'visibility') return hour.visibilityMeters >= 15_000 ? 'good' : hour.visibilityMeters >= 7_000 ? 'caution' : 'poor'
  if (metric === 'aod') return !air ? 'neutral' : air.aerosolOpticalDepth <= .1 ? 'good' : air.aerosolOpticalDepth <= .25 ? 'caution' : 'poor'
  const spread = hour.temperature - hour.dewPoint
  return spread >= 5 ? 'good' : spread >= 2 ? 'caution' : 'poor'
}

function clamp(value: number, minimum = 0, maximum = 1) {
  return Math.max(minimum, Math.min(maximum, value))
}

/**
 * 以月球近侧贴图为材质，在画布上正交投影为圆盘；每个像素用太阳-月球相对角度计算 Lambert 受光。
 * 相位 0° 为朔、180° 为望，因此在北半球相位超过 180° 时会自然得到左侧受光的亏月。
 */
function renderMoon() {
  const canvas = moonCanvas.value
  if (!canvas || !moonTexturePixels || !moonTextureWidth || !moonTextureHeight) return

  const size = 320
  const center = size / 2
  const radius = center - .5
  canvas.width = size
  canvas.height = size
  const context = canvas.getContext('2d', { alpha: true })
  if (!context) return
  const output = context.createImageData(size, size)
  const source = moonTexturePixels.data
  const phaseRadians = moonPanel.value.phase * Math.PI / 180
  const sunX = Math.sin(phaseRadians)
  const sunZ = -Math.cos(phaseRadians)

  for (let pixelY = 0; pixelY < size; pixelY += 1) {
    const normalY = (center - pixelY) / radius
    for (let pixelX = 0; pixelX < size; pixelX += 1) {
      const normalX = (pixelX - center) / radius
      const radiusSquared = normalX * normalX + normalY * normalY
      const targetIndex = (pixelY * size + pixelX) * 4
      if (radiusSquared > 1) continue

      const normalZ = Math.sqrt(1 - radiusSquared)
      const longitude = Math.atan2(normalX, normalZ)
      const latitude = Math.asin(normalY)
      const sourceX = Math.min(moonTextureWidth - 1, Math.max(0, Math.floor((longitude / (2 * Math.PI) + .5) * moonTextureWidth)))
      const sourceY = Math.min(moonTextureHeight - 1, Math.max(0, Math.floor((.5 - latitude / Math.PI) * moonTextureHeight)))
      const sourceIndex = (sourceY * moonTextureWidth + sourceX) * 4
      const directLight = normalX * sunX + normalZ * sunZ
      // 约 4% 的地球照保留暗面地形；窄过渡带避免上一版生硬的黑色圆遮罩。
      const terminator = clamp((directLight + .055) / .11)
      const illumination = .04 + .96 * Math.pow(terminator, .78)
      // 月缘只保留轻微的球面压暗，不再人为制造一圈黑边。
      const limbDarkening = .9 + .1 * normalZ
      const light = illumination * limbDarkening
      const edge = clamp((1 - Math.sqrt(radiusSquared)) / .005)
      output.data[targetIndex] = Math.round(source[sourceIndex] * light)
      output.data[targetIndex + 1] = Math.round(source[sourceIndex + 1] * light)
      output.data[targetIndex + 2] = Math.round(source[sourceIndex + 2] * light)
      output.data[targetIndex + 3] = Math.round(255 * edge)
    }
  }
  context.putImageData(output, 0, 0)
}

function loadMoonTexture() {
  const image = new Image()
  image.onload = () => {
    const textureCanvas = document.createElement('canvas')
    moonTextureWidth = 1024
    moonTextureHeight = 512
    textureCanvas.width = moonTextureWidth
    textureCanvas.height = moonTextureHeight
    const textureContext = textureCanvas.getContext('2d', { willReadFrequently: true })
    if (!textureContext) return
    textureContext.drawImage(image, 0, 0, moonTextureWidth, moonTextureHeight)
    moonTexturePixels = textureContext.getImageData(0, 0, moonTextureWidth, moonTextureHeight)
    renderMoon()
  }
  image.src = moonNearsideTexture
}

function visibleSegments(track: BodyTrack) {
  // 与展开波形同一平滑正弦源（idealWaveAltitude，193 点同分辨率）：段边界取正弦的 0° 穿越点，
  // 保证收起态色块条的"在地平线上"区间与展开波形完全一致。
  const pointCount = 193
  const altitudeAt = (index: number) => idealWaveAltitude(track, index / pointCount * 1440)
  const minuteOf = (sampleIndex: number) => sampleIndex / pointCount * 1440
  const crossingIndex = (fromIndex: number, toIndex: number) => {
    const a = altitudeAt(fromIndex)
    const b = altitudeAt(toIndex)
    return fromIndex + (0 - a) / (b - a) * (toIndex - fromIndex)
  }
  const segments: Array<{ left: number; width: number }> = []
  let runStart = -1
  for (let index = 0; index <= pointCount; index++) {
    const above = altitudeAt(index) > 0
    if (above && runStart < 0) runStart = index
    const closes = runStart >= 0 && (!above || index === pointCount)
    if (closes) {
      const last = above ? index : index - 1
      const startMinute = runStart > 0 ? minuteOf(crossingIndex(runStart - 1, runStart)) : 0
      const endMinute = last < pointCount ? minuteOf(crossingIndex(last, last + 1)) : 1440
      segments.push({ left: startMinute / 1440 * 100, width: Math.max(.9, (endMinute - startMinute) / 1440 * 100) })
      runStart = -1
    }
  }
  return segments
}

/* 高度角-时间曲线只表达一天内的升降节奏：以升起时刻作为相位起点，
   24 小时绘制一个完整正弦周期。详情中的升落时刻仍来自真实星历。 */
interface AltitudeChartMetrics {
  width: number
  height: number
  horizonY: number
  scale: number // 每度高度角对应的纵向 viewBox 单位
}

// 会意式全天波形使用完整的天球高度范围：顶部 +90°、中线 0°、底部 -90°。
// 上下各留 2 个 viewBox 单位，避免圆点和描边贴住裁切边缘。
const ALTITUDE_CHART_ROW: AltitudeChartMetrics = { width: 960, height: 88, horizonY: 44, scale: 42 / 90 }
const IDEAL_WAVE_AMPLITUDE = 90

interface AltitudeCurve {
  fullPath: string
  belowPaths: string[]
  abovePaths: string[]
}

function altitudeToY(altitude: number, metrics: AltitudeChartMetrics) {
  return Math.max(0, Math.min(metrics.height, metrics.horizonY - altitude * metrics.scale))
}

function smoothPath(points: Array<{ x: number; y: number }>) {
  if (points.length < 2) return ''
  let path = `M ${points[0].x.toFixed(1)} ${points[0].y.toFixed(1)}`
  for (let index = 0; index < points.length - 1; index += 1) {
    const previous = points[Math.max(0, index - 1)]
    const current = points[index]
    const next = points[index + 1]
    const following = points[Math.min(points.length - 1, index + 2)]
    const controlOneX = current.x + (next.x - previous.x) / 6
    const controlOneY = current.y + (next.y - previous.y) / 6
    const controlTwoX = next.x - (following.x - current.x) / 6
    const controlTwoY = next.y - (following.y - current.y) / 6
    path += ` C ${controlOneX.toFixed(1)} ${controlOneY.toFixed(1)} ${controlTwoX.toFixed(1)} ${controlTwoY.toFixed(1)} ${next.x.toFixed(1)} ${next.y.toFixed(1)}`
  }
  return path
}

function minuteInDay(value: Date) {
  return zonedMinuteOfDay(value, observatoryTimezone.value)
}

function positiveMinuteDelta(value: number) {
  return (value % 1440 + 1440) % 1440
}

function idealWaveAltitude(track: BodyTrack, minute: number) {
  const riseMinute = track.rise ? minuteInDay(track.rise) : null
  const setMinute = track.set ? minuteInDay(track.set) : null
  if (riseMinute != null && setMinute != null) {
    // 一天 = 一条连续正弦波：升起→落下为峰值 A 的拱弧（实线），其余时间为同幅反向
    // 正弦凹谷（虚线）——虚线延续拱弧的弧度、在升/落点平滑接续；不会因取模产生
    // 多余的第二次拱起，地平线以下的虚线自然连续。
    const duration = Math.max(60, positiveMinuteDelta(setMinute - riseMinute))
    const elapsed = positiveMinuteDelta(minute - riseMinute)
    if (elapsed <= duration) {
      return IDEAL_WAVE_AMPLITUDE * Math.sin(Math.PI * elapsed / duration)
    }
    const troughSpan = 1440 - duration
    const belowElapsed = elapsed - duration
    return -IDEAL_WAVE_AMPLITUDE * Math.sin(Math.PI * belowElapsed / troughSpan)
  }
  // 极昼极夜或数据缺失：以中天为中心取一段理想可见弧（与原实现一致）
  const transitMinute = track.transit ? minuteInDay(track.transit) : 720
  const elapsed = positiveMinuteDelta(minute - positiveMinuteDelta(transitMinute - 360))
  return IDEAL_WAVE_AMPLITUDE * Math.sin(Math.PI * elapsed / 720)
}

function altitudeCurve(track: BodyTrack, metrics: AltitudeChartMetrics): AltitudeCurve {
  const pointCount = 193
  const points = Array.from({ length: pointCount }, (_, index) => {
    const minute = index / (pointCount - 1) * 1440
    return {
      x: index / (pointCount - 1) * metrics.width,
      y: altitudeToY(idealWaveAltitude(track, minute), metrics),
    }
  })
  // 平滑曲线（Catmull-Rom）采样，并在地平线 y=horizonY 处切分为上/下独立段路径。
  // 实线与虚线共享同一穿越点、各自成段，不依赖 clip-path url() 引用（避免浏览器接缝渲染问题）。
  const samples: Array<{ x: number; y: number }> = []
  for (let index = 0; index < points.length - 1; index++) {
    const previous = points[Math.max(0, index - 1)]
    const current = points[index]
    const next = points[index + 1]
    const following = points[Math.min(points.length - 1, index + 2)]
    const c1 = { x: current.x + (next.x - previous.x) / 6, y: current.y + (next.y - previous.y) / 6 }
    const c2 = { x: next.x - (following.x - current.x) / 6, y: next.y - (following.y - current.y) / 6 }
    for (let step = 0; step < 12; step++) {
      const u = step / 12
      const w = 1 - u
      samples.push({
        x: w * w * w * current.x + 3 * w * w * u * c1.x + 3 * w * u * u * c2.x + u * u * u * next.x,
        y: w * w * w * current.y + 3 * w * w * u * c1.y + 3 * w * u * u * c2.y + u * u * u * next.y,
      })
    }
  }
  samples.push(points[points.length - 1])

  const belowRuns: string[] = []
  const aboveRuns: string[] = []
  let below: Array<{ x: number; y: number }> = []
  let above: Array<{ x: number; y: number }> = []
  const toPath = (run: Array<{ x: number; y: number }>) => run.length ? `M ${run.map((p) => `${p.x.toFixed(1)} ${p.y.toFixed(1)}`).join(' L ')}` : ''
  const flush = () => {
    if (below.length) belowRuns.push(toPath(below))
    if (above.length) aboveRuns.push(toPath(above))
    below = []
    above = []
  }
  for (let index = 0; index < samples.length - 1; index++) {
    const a = samples[index]
    const b = samples[index + 1]
    const aAbove = a.y < metrics.horizonY
    const bAbove = b.y < metrics.horizonY
    ;(aAbove ? above : below).push(a)
    if (aAbove !== bAbove) {
      const ratio = (metrics.horizonY - a.y) / (b.y - a.y)
      const crossing = { x: a.x + (b.x - a.x) * ratio, y: metrics.horizonY }
      // 穿越点同时收进旧段结尾与新段开头，保证实/虚线共享同一接缝点
      ;(aAbove ? above : below).push(crossing)
      flush()
      ;(bAbove ? above : below).push(crossing)
    }
  }
  const last = samples[samples.length - 1]
  ;(last.y < metrics.horizonY ? above : below).push(last)
  flush()

  return {
    fullPath: smoothPath(points),
    belowPaths: belowRuns,
    abovePaths: aboveRuns,
  }
}

function horizonStyle(track: BodyTrack) {
  const projection = projectHorizontalDirection(track.azimuth, track.altitude, skyCamera.value)
  // 太阳恒为不透明；其余天体白天也标注位置：正午（daylight=1）保持 50%，
  // 随天黑（daylight→0）线性变亮到 100%——行星常在白天天空，不应完全隐藏。
  const opacity = track.id === 'sun' ? 1 : 1 - .5 * daylight.value
  return {
    left: `${projection.x * 100}%`,
    top: `${projection.y * 100}%`,
    '--body-tint': track.tint,
    '--body-opacity': String(opacity), // 入场动画终点跟随静止透明度，避免黄昏时"先亮后暗"
    opacity: String(opacity),
  }
}

function altitudeLabelStyle(label: { x: number; y: number } | null) {
  return label ? { left: `${label.x * 100}%`, top: `${label.y * 100}%` } : undefined
}

function projectSkyTrackRuns(samples: Array<{ altitude: number; azimuth: number }>, camera: SkyCamera) {
  const paths: string[] = []
  let run: Array<{ x: number; y: number }> = []
  const flush = () => {
    if (run.length >= 2) paths.push(`M ${run.map((point) => `${(point.x * 1000).toFixed(2)} ${(point.y * 1000).toFixed(2)}`).join(' L ')}`)
    run = []
  }

  for (const sample of samples) {
    const projection = projectHorizontalDirection(sample.azimuth, sample.altitude, camera)
    const drawable = sample.altitude >= 0 && projection.inFront
      && projection.x >= -.04 && projection.x <= 1.04
      && projection.y >= -.08 && projection.y <= 1.04
    const previous = run[run.length - 1]
    if (!drawable || (previous && Math.abs(projection.x - previous.x) > .28)) {
      flush()
      if (!drawable) continue
    }
    run.push({ x: projection.x, y: projection.y })
  }
  flush()
  return paths
}

function commitPendingSkyView() {
  if (pendingSkyViewAzimuth !== undefined) {
    skyViewAzimuth.value = pendingSkyViewAzimuth
    pendingSkyViewAzimuth = undefined
  }
  skyViewAnimationFrame = undefined
}

// Pointer / wheel 事件可能远高于屏幕刷新率；同一帧只提交最后一次朝向，避免无意义重复投影与 DOM 更新。
function scheduleSkyView(nextAzimuth: number) {
  pendingSkyViewAzimuth = normalizeAzimuth(nextAzimuth)
  if (skyViewAnimationFrame === undefined) skyViewAnimationFrame = requestAnimationFrame(commitPendingSkyView)
}

function flushPendingSkyView() {
  if (skyViewAnimationFrame !== undefined) cancelAnimationFrame(skyViewAnimationFrame)
  commitPendingSkyView()
}

function cancelPendingSkyView() {
  if (skyViewAnimationFrame !== undefined) cancelAnimationFrame(skyViewAnimationFrame)
  skyViewAnimationFrame = undefined
  pendingSkyViewAzimuth = undefined
}

function cancelSkyViewTurn() {
  if (skyViewTurnAnimationFrame !== undefined) cancelAnimationFrame(skyViewTurnAnimationFrame)
  skyViewTurnAnimationFrame = undefined
  skyViewAutoTurning.value = false
}

function animateSkyViewTo(targetAzimuth: number) {
  cancelPendingSkyView()
  cancelSkyViewTurn()
  const start = skyViewAzimuth.value
  const delta = shortestAzimuthDelta(start, targetAzimuth)
  if (Math.abs(delta) < .01 || window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    skyViewAzimuth.value = normalizeAzimuth(targetAzimuth)
    return
  }

  const duration = skyTurnDuration(delta)
  const startedAt = performance.now()
  skyViewAutoTurning.value = true
  const step = (timestamp: number) => {
    const progress = Math.min(1, (timestamp - startedAt) / duration)
    skyViewAzimuth.value = normalizeAzimuth(start + delta * easeOutExpo(progress))
    if (progress < 1) {
      skyViewTurnAnimationFrame = requestAnimationFrame(step)
    } else {
      skyViewTurnAnimationFrame = undefined
      skyViewAutoTurning.value = false
      skyViewAzimuth.value = normalizeAzimuth(targetAzimuth)
    }
  }
  skyViewTurnAnimationFrame = requestAnimationFrame(step)
}

function revealSkyDirection(targetAzimuth: number) {
  window.setTimeout(() => {
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    document.querySelector('.horizon-section')?.scrollIntoView({ behavior: reducedMotion ? 'auto' : 'smooth', block: 'start' })
    requestAnimationFrame(() => animateSkyViewTo(targetAzimuth))
  }, 0)
}

// 搜索结果定位只旋转星图，不改变页面的纵向滚动位置。
function revealCatalogDirection(targetAzimuth: number) {
  requestAnimationFrame(() => animateSkyViewTo(targetAzimuth))
}

function rotateSkyView(change: number) {
  cancelSkyViewTurn()
  scheduleSkyView((pendingSkyViewAzimuth ?? skyViewAzimuth.value) + change)
}

function turnSkyViewByWheel(event: WheelEvent) {
  cancelSkyViewTurn()
  const movement = Math.abs(event.deltaX) > Math.abs(event.deltaY) ? event.deltaX : event.deltaY
  rotateSkyView(movement * .08)
}

function beginSkyViewDrag(event: PointerEvent) {
  if (!activeCoordinates.value) return
  cancelSkyViewTurn()
  flushPendingSkyView()
  skyViewDragging.value = true
  skyViewStartX = event.clientX
  skyViewStartAzimuth = skyViewAzimuth.value
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
}

function dragSkyView(event: PointerEvent) {
  if (!skyViewDragging.value) return
  scheduleSkyView(skyViewStartAzimuth - (event.clientX - skyViewStartX) * .28)
}

function endSkyViewDrag(event: PointerEvent) {
  if (!skyViewDragging.value) return
  flushPendingSkyView()
  skyViewDragging.value = false
  const field = event.currentTarget as HTMLElement
  if (field.hasPointerCapture(event.pointerId)) field.releasePointerCapture(event.pointerId)
}

function selectPage(page: SkyPage) {
  if (page !== 'sky') cancelSkyViewTurn()
  activePage.value = page
  window.history.pushState(null, '', `#astronomy-${page}`)
  document.querySelector('.sky-content-scroll')?.scrollTo({ top: 0, behavior: 'smooth' })
}

function selectBody(body: BodyId) {
  expandedBodyId.value = expandedBodyId.value === body ? null : body
}

// 从星图点击行星图标：展开下方对应行星行并平滑滚动到该行。
function revealBody(body: BodyId) {
  expandedBodyId.value = body
  window.setTimeout(() => {
    document.getElementById(`track-trigger-${body}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }, 0)
}

function locateBody(body: BodyId) {
  const track = tracks.value.find((item) => item.id === body)
  // 只有当前在地平线以上的天体才值得定位：平滑转动罗盘到其方位，再滚动到视场。
  if (!track || !track.visible) return
  expandedBodyId.value = body
  revealSkyDirection(track.azimuth)
}

function stopMinuteAnimation() {
  if (minuteAnimationFrame !== undefined) {
    cancelAnimationFrame(minuteAnimationFrame)
    minuteAnimationFrame = undefined
  }
}

function cancelPendingTimeScrub() {
  if (timeScrubAnimationFrame !== undefined) {
    cancelAnimationFrame(timeScrubAnimationFrame)
    timeScrubAnimationFrame = undefined
  }
  pendingMinuteOfDay = undefined
}

// 原生 range 可能在一帧内派发多次 input；星历与 SVG 投影最多每帧更新一次。
function scheduleMinuteOfDay(event: Event) {
  const target = event.currentTarget as HTMLInputElement
  stopMinuteAnimation()
  followingRealTime.value = false
  pendingMinuteOfDay = Number(target.value)
  if (timeScrubAnimationFrame !== undefined) return
  timeScrubAnimationFrame = requestAnimationFrame(() => {
    if (pendingMinuteOfDay !== undefined) minuteOfDay.value = pendingMinuteOfDay
    pendingMinuteOfDay = undefined
    timeScrubAnimationFrame = undefined
  })
}

function commitMinuteOfDay(event: Event) {
  cancelPendingTimeScrub()
  followingRealTime.value = false
  minuteOfDay.value = Number((event.currentTarget as HTMLInputElement).value)
}

function jumpToNow() {
  const current = new Date()
  const target = Math.floor(zonedMinuteOfDay(current, observatoryTimezone.value))
  const start = minuteOfDay.value
  cancelPendingTimeScrub()
  stopMinuteAnimation()
  followingRealTime.value = true
  now.value = current
  if (start === target) return
  const duration = Math.min(1500, Math.max(450, Math.abs(target - start) * 5))
  const startTime = performance.now()
  const step = (timestamp: number) => {
    const progress = Math.min(1, (timestamp - startTime) / duration)
    const eased = progress < .5 ? 4 * progress ** 3 : 1 - Math.pow(-2 * progress + 2, 3) / 2
    minuteOfDay.value = Math.round(start + (target - start) * eased)
    if (progress < 1) {
      minuteAnimationFrame = requestAnimationFrame(step)
    } else {
      minuteAnimationFrame = undefined
      minuteOfDay.value = target
    }
  }
  minuteAnimationFrame = requestAnimationFrame(step)
}

async function resolveLocationName(currentLatitude: number, currentLongitude: number, revision: number) {
  locationLookupController?.abort()
  const controller = new AbortController()
  locationLookupController = controller
  locationStatus.value = 'resolving'
  locationLabel.value = '正在确认城市区县'
  try {
    const place = await fetchObserverPlace(currentLatitude, currentLongitude, controller.signal)
    if (revision !== locationRevision) return
    locationLabel.value = place.label
    locationStatus.value = 'located'
  } catch {
    if (controller.signal.aborted || revision !== locationRevision) return
    locationLabel.value = '地名暂不可用'
    locationStatus.value = 'partial'
  } finally {
    if (locationLookupController === controller) locationLookupController = undefined
  }
}

function requestLocation() {
  if (locationStatus.value === 'locating' || locationStatus.value === 'resolving') return
  if (!navigator.geolocation) {
    locationStatus.value = 'unavailable'
    locationLabel.value = '浏览器不支持定位'
    return
  }
  const revision = ++locationRevision
  locationLookupController?.abort()
  locationStatus.value = 'locating'
  locationLabel.value = '正在获取位置'
  navigator.geolocation.getCurrentPosition(
    ({ coords }) => {
      if (revision !== locationRevision) return
      latitude.value = coords.latitude
      longitude.value = coords.longitude
      showLocationEditor.value = false
      void resolveLocationName(coords.latitude, coords.longitude, revision)
    },
    (error) => {
      if (revision !== locationRevision) return
      if (error.code === error.PERMISSION_DENIED) {
        locationStatus.value = 'denied'
        locationLabel.value = '定位未授权'
      } else {
        locationStatus.value = 'unavailable'
        locationLabel.value = error.code === error.TIMEOUT ? '定位请求超时' : '暂时无法获取位置'
      }
    },
    { enableHighAccuracy: true, timeout: 10_000, maximumAge: 300_000 },
  )
}

function toggleLocationEditor() {
  showLocationEditor.value = !showLocationEditor.value
}

function closeLocationEditor(returnFocus = false) {
  if (!showLocationEditor.value) return
  showLocationEditor.value = false
  if (returnFocus) locationTrigger.value?.focus()
}

function onLocationOutsidePointerDown(event: PointerEvent) {
  const target = event.target
  if (!(target instanceof Node)) return
  if (!locationControl.value?.contains(target)) closeLocationEditor()
  if (!skySearchControl.value?.contains(target)) showSkySearchResults.value = false
}

function onLocationEditorKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape') return
  const handledLocation = showLocationEditor.value
  const handledSearch = showSkySearchResults.value
  if (!handledLocation && !handledSearch) return
  event.preventDefault()
  event.stopPropagation()
  if (handledLocation) closeLocationEditor(true)
  showSkySearchResults.value = false
}

function useCoordinates(nextLatitude: number, nextLongitude: number, label?: string) {
  if (!Number.isFinite(nextLatitude) || nextLatitude < -90 || nextLatitude > 90 || !Number.isFinite(nextLongitude) || nextLongitude < -180 || nextLongitude > 180) {
    locationFormError.value = '纬度需在 -90～90，经度需在 -180～180。'
    return
  }
  const revision = ++locationRevision
  locationLookupController?.abort()
  latitude.value = nextLatitude
  longitude.value = nextLongitude
  manualLatitude.value = nextLatitude.toFixed(6)
  manualLongitude.value = nextLongitude.toFixed(6)
  locationFormError.value = ''
  showLocationEditor.value = false
  if (label) {
    locationLabel.value = label
    locationStatus.value = 'located'
  } else {
    void resolveLocationName(nextLatitude, nextLongitude, revision)
  }
}

function submitManualCoordinates() {
  useCoordinates(Number(manualLatitude.value), Number(manualLongitude.value))
}

async function submitLocationSearch() {
  const query = locationQuery.value.trim()
  if (query.length < 2) {
    locationFormError.value = '请输入至少两个字的乡、镇、区县或城市名称。'
    return
  }
  locationSearchController?.abort()
  const controller = new AbortController()
  locationSearchController = controller
  locationSearchStatus.value = 'loading'
  locationFormError.value = ''
  try {
    const response = await searchObserverPlaces(query, controller.signal)
    if (controller.signal.aborted) return
    locationSearchResults.value = response.places
    locationSearchStatus.value = 'ready'
    if (!response.places.length) locationFormError.value = '没有匹配到行政区划，请补充省市名称后重试。'
  } catch {
    if (!controller.signal.aborted) {
      locationSearchStatus.value = 'error'
      locationFormError.value = '地点搜索暂不可用，请输入经纬度。'
    }
  } finally {
    if (locationSearchController === controller) locationSearchController = undefined
  }
}

function selectLocationCandidate(place: ObserverPlaceCandidate) {
  useCoordinates(place.latitude, place.longitude, place.label)
  locationSearchResults.value = []
}

function locateCatalogObject(item: SkyCatalogObject) {
  const position = catalogPositions.value.get(item.id)
  if (!position?.visible) return
  selectedCatalogId.value = item.id
  skySearchQuery.value = item.name
  showSkySearchResults.value = false
  revealCatalogDirection(position.azimuth)
}

function locateSkySearchResult(result: SkySearchResult) {
  if (!result.visible) return
  if (result.bodyId) {
    const track = tracks.value.find((item) => item.id === result.bodyId)
    if (!track?.visible) return
    expandedBodyId.value = result.bodyId
    selectedCatalogId.value = null
    skySearchQuery.value = result.name
    showSkySearchResults.value = false
    revealCatalogDirection(track.azimuth)
    return
  }
  if (result.catalogObject) locateCatalogObject(result.catalogObject)
}

function locateFirstSkySearchMatch() {
  const firstVisible = skySearchResults.value.find((item) => item.visible)
  if (firstVisible) locateSkySearchResult(firstVisible)
}

async function loadConditions(currentLatitude: number, currentLongitude: number) {
  conditionsController?.abort()
  const controller = new AbortController()
  conditionsController = controller
  conditionsStatus.value = 'loading'
  try {
    conditions.value = await fetchObservingConditions(currentLatitude, currentLongitude, controller.signal, true)
    conditionsStatus.value = 'ready'
  } catch {
    if (!controller.signal.aborted) conditionsStatus.value = 'error'
  } finally {
    if (conditionsController === controller) conditionsController = undefined
  }
}

async function loadMoonDay(currentLatitude: number, currentLongitude: number) {
  try {
    moonDay.value = await fetchMoonDay(currentLatitude, currentLongitude, elevation.value, observatoryTimezone.value, Math.floor(now.value.getTime() / 1000))
  } catch {
    moonDay.value = null // 后端不可用时退回本地计算
  }
}

// 天象事件：后端每日生成/校验，前端只请求 AURORA API。
// 未定位时返回全球日历（locationVisibility=location_required）。
const astronomyEvents = ref<AstronomyEvent[]>([])
const astronomyEventSources = ref<AstronomyEventSourceStatus[]>([])
let astronomyEventsController: AbortController | undefined
const astronomyEventsStatus = ref<'idle' | 'loading' | 'ready' | 'error'>('idle')

async function loadAstronomyEvents() {
  astronomyEventsController?.abort()
  const controller = new AbortController()
  astronomyEventsController = controller
  astronomyEventsStatus.value = 'loading'
  try {
    const response = await fetchAstronomyEvents({
      latitude: activeCoordinates.value?.latitude,
      longitude: activeCoordinates.value?.longitude,
      timezone: observatoryTimezone.value,
    }, controller.signal)
    if (!controller.signal.aborted) {
      astronomyEvents.value = response.events
      astronomyEventSources.value = response.sources ?? []
      astronomyEventsStatus.value = 'ready'
    }
  } catch {
    // 后端不可用时不清空已加载事件，仍明确告知这次刷新失败。
    if (!controller.signal.aborted) astronomyEventsStatus.value = 'error'
  } finally {
    if (astronomyEventsController === controller) astronomyEventsController = undefined
  }
}

async function loadImageWall() {
  imageWallController?.abort()
  const controller = new AbortController()
  imageWallController = controller
  imageWallStatus.value = 'loading'
  try {
    const wall = await fetchImageWall(controller.signal)
    if (!controller.signal.aborted) {
      imageWall.value = wall
      imageWallStatus.value = 'ready'
    }
  } catch {
    if (!controller.signal.aborted) imageWallStatus.value = 'error'
  } finally {
    if (imageWallController === controller) imageWallController = undefined
  }
}

async function loadLightPollution(currentLatitude: number, currentLongitude: number) {
  lightPollution.value = await fetchLightPollution(currentLatitude, currentLongitude)
}

function onPopState() {
  if (window.location.hash.startsWith('#astronomy')) activePage.value = pageFromHash()
}

function beginHomeExit() {
  if (skyLeaving.value) return
  skyLeaving.value = true
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  homeExitTimer = window.setTimeout(() => emit('home'), reduced ? 0 : 180)
}

watch(activeCoordinates, (coordinates) => {
  if (coordinates) {
    void loadConditions(coordinates.latitude, coordinates.longitude)
    void loadMoonDay(coordinates.latitude, coordinates.longitude)
    void loadLightPollution(coordinates.latitude, coordinates.longitude)
  }
  // 无定位授权也必须载入全球日历；坐标可用时会自动附加本地可见性。
  void loadAstronomyEvents()
}, { immediate: true })
// 天气接口返回真实海拔后，用该海拔重取每日月相。
watch(elevation, () => {
  if (activeCoordinates.value) void loadMoonDay(activeCoordinates.value.latitude, activeCoordinates.value.longitude)
})
// 定位点跨时区时，重置“现在”的当地分钟并重取按当地日期缓存的月相。
watch(observatoryTimezone, () => {
  if (followingRealTime.value) minuteOfDay.value = Math.floor(zonedMinuteOfDay(now.value, observatoryTimezone.value))
  if (activeCoordinates.value) void loadMoonDay(activeCoordinates.value.latitude, activeCoordinates.value.longitude)
  if (activeCoordinates.value) void loadAstronomyEvents()
})
watch(activePage, (page) => {
  if (page === 'daily-image' && imageWallStatus.value === 'idle') void loadImageWall()
})
watch(moon, renderMoon)
watch(moonDay, renderMoon)
watch(moonCanvas, (canvas) => {
  if (canvas) renderMoon()
})

onMounted(() => {
  window.addEventListener('popstate', onPopState)
  document.addEventListener('pointerdown', onLocationOutsidePointerDown)
  document.addEventListener('keydown', onLocationEditorKeydown)
  // 与左下角秒钟共用真实时间节奏；星历仅在当地分钟变化时更新。
  minuteClock = window.setInterval(() => {
    const real = new Date()
    if (followingRealTime.value) {
      const localMinute = Math.floor(zonedMinuteOfDay(real, observatoryTimezone.value))
      if (localMinute !== minuteOfDay.value) {
        now.value = real
        minuteOfDay.value = localMinute
      }
    }
  }, 1_000)
  loadMoonTexture()
  requestLocation()
})

onBeforeUnmount(() => {
  locationRevision += 1
  locationLookupController?.abort()
  locationSearchController?.abort()
  conditionsController?.abort()
  imageWallController?.abort()
  if (homeExitTimer !== undefined) window.clearTimeout(homeExitTimer)
  window.removeEventListener('popstate', onPopState)
  document.removeEventListener('pointerdown', onLocationOutsidePointerDown)
  document.removeEventListener('keydown', onLocationEditorKeydown)
  if (minuteClock !== undefined) window.clearInterval(minuteClock)
  cancelPendingSkyView()
  cancelSkyViewTurn()
  cancelPendingTimeScrub()
  stopMinuteAnimation()
})
</script>

<template>
  <section class="sky-shell" :class="{ 'is-leaving': skyLeaving }" aria-label="AURORA 天文观测">
    <aside class="sky-sidebar">
      <AuroraBrand class="sky-brand" subtitle="SKY OBSERVATORY" variant="sky" :leaving="skyLeaving" @click="beginHomeExit" />
      <div class="sidebar-divider" aria-hidden="true" />

      <div ref="locationControl" class="location-control">
        <button ref="locationTrigger" class="sky-location" type="button" aria-haspopup="dialog" :aria-expanded="showLocationEditor" aria-controls="location-editor" @click="toggleLocationEditor">
          <span class="location-mark" aria-hidden="true" />
          <span aria-live="polite"><strong>{{ locationLabel }}</strong><small>{{ coordinateLabel }}</small></span>
          <i>{{ showLocationEditor ? '收起' : '更改' }}</i>
        </button>
        <Transition name="location-panel">
          <section v-if="showLocationEditor" id="location-editor" class="location-editor" role="dialog" aria-label="设置观测地点">
            <button class="location-gps" type="button" :disabled="locationStatus === 'locating' || locationStatus === 'resolving'" @click="requestLocation">{{ locationStatus === 'locating' || locationStatus === 'resolving' ? '正在定位…' : '使用设备定位' }}</button>
            <form @submit.prevent="submitLocationSearch">
              <label for="location-query">搜索乡镇、区县或城市</label>
              <div><input id="location-query" v-model="locationQuery" autocomplete="address-level2" placeholder="例：上海市崇明区" /><button type="submit" :disabled="locationSearchStatus === 'loading'">搜索</button></div>
            </form>
            <ul v-if="locationSearchResults.length" class="location-results">
              <li v-for="place in locationSearchResults" :key="`${place.adcode}-${place.latitude}-${place.longitude}`"><button type="button" @click="selectLocationCandidate(place)"><strong>{{ place.label }}</strong><small>{{ place.latitude.toFixed(4) }}, {{ place.longitude.toFixed(4) }} · WGS84</small></button></li>
            </ul>
            <form @submit.prevent="submitManualCoordinates">
              <label>请输入经纬度</label>
              <div class="coordinate-inputs"><input v-model="manualLatitude" inputmode="decimal" aria-label="纬度" placeholder="纬度" /><input v-model="manualLongitude" inputmode="decimal" aria-label="经度" placeholder="经度" /><button type="submit">使用</button></div>
            </form>
            <p v-if="locationFormError" class="location-form-error" role="alert">{{ locationFormError }}</p>
          </section>
        </Transition>
      </div>

      <nav class="sky-menu" aria-label="天文观测页面">
        <button v-for="item in menu" :key="item.id" type="button" :class="{ active: activePage === item.id }" @click="selectPage(item.id)">
          <b>{{ item.numeral }}</b><span>{{ item.label }}<small>{{ item.en }}</small></span><i class="menu-icon" :class="`menu-icon-${item.icon}`" aria-hidden="true"><em /></i>
        </button>
      </nav>

      <div class="sidebar-source"><span />{{ activeCoordinates ? '本地星历计算' : '需要地点以计算本地天空' }}</div>
      <ObservatoryClock :timezone="observatoryTimezone" />
    </aside>

    <main class="sky-content-scroll">
      <section v-if="activePage === 'conditions'" class="conditions-page page-stack">
        <div class="condition-hero">
          <div class="moon-disc" role="img" :aria-label="`${moonPanel.label}，月球面向地球的一面，亮面占比 ${moonIllumination}%`">
            <canvas ref="moonCanvas" aria-hidden="true" />
          </div>
          <div class="moon-copy"><p>MOON / 今日月相</p><h2>{{ moonPanel.label }}</h2><span>月球亮面占比 {{ moonIllumination }}% · 月龄 {{ moonPanel.age.toFixed(1) }} 日</span></div>
          <div class="moon-rise" v-if="moonPanel.moonrise || moonPanel.moonset">
            <div class="moon-rise-times"><strong><i>↑</i> 月出 <time>{{ formatTime(moonPanel.moonrise) }}</time></strong><strong><i>↓</i> 月落 <time>{{ formatTime(moonPanel.moonset) }}</time></strong></div>
            <small>亮面占比 <b>{{ moonIllumination }}%</b></small>
          </div>
        </div>

        <section class="condition-verdict" :class="{ loading: conditionsStatus === 'loading' }">
          <div class="score-now"><strong>{{ scorePanel == null ? '—' : String(scorePanel).padStart(2, '0') }}<small>/100</small></strong><i>{{ scoreWindowLabel }}</i></div>
          <div><h2>今夜观测评分 · {{ scoreVerdict }}</h2><p v-if="scoreWeather">{{ scoreWindowLabel }}：云量 {{ Math.round(scoreWeather.cloudCover) }}%，能见度 {{ (scoreWeather.visibilityMeters / 1000).toFixed(1) }} km；{{ conditionDescription(scoreWeather.weatherCode) }}是该时段的主导条件。</p><p v-else-if="conditionsStatus === 'error'">天气源暂不可用；本地星历仍可计算天体位置与升落。</p><p v-else>正在读取今晚天文夜内的云层、能见度、降水与月光条件。</p></div>
          <div v-if="scoreFactors" class="score-factors" aria-label="今夜观测评分扣分项">
            <p>今夜最佳时段扣分项</p>
            <span v-for="part in scoreFactors" :key="part.key" :class="part.kind"><i :style="{ width: factorBarWidth(part) }" /><small>{{ part.label }}</small><strong>{{ part.value.toFixed(1) }}</strong></span>
            <em v-if="!scoreFactors.length">当前没有明显扣分项</em>
          </div>
          <article class="tomorrow-score" aria-label="明日夜间观测条件预估">
            <span>明日夜间</span><strong>{{ tomorrowScore ? `${tomorrowScore.score}/100` : '暂未覆盖' }}</strong><small v-if="tomorrowScore">最佳 {{ tomorrowScore.time.slice(11, 16) }} · {{ tomorrowScore.verdict }}</small><small v-else>明日 00:00–23:00 内暂无可用于夜间评分的天气数据。</small>
          </article>
        </section>

        <section class="observing-console" aria-label="今晚建议与当前观测环境">
          <div class="observing-brief">
            <div class="advice-window">
              <p>今夜建议</p>
              <h3>{{ recommendation?.windowLabel ?? '等待生成本地观测建议' }}</h3>
              <span v-if="recommendation?.window">{{ recommendation.window.verdict }}</span>
              <span v-else-if="!activeCoordinates">允许定位后计算今晚的目标与窗口</span>
              <em v-if="recommendation && !recommendation.weatherAvailable" class="advice-degraded">{{ recommendation.weatherCoverage === 'outside' ? '天气预报暂未覆盖这段天文夜，以下目标来自本地星历' : '天气源不可用，以下目标与窗口来自本地星历' }}</em>
            </div>
          </div>
          <ul v-if="recommendation?.targets.length" class="observing-targets">
            <li v-for="target in recommendation.targets" :key="target.id"><i :style="{ color: target.tint }">{{ target.glyph }}</i><span><strong>{{ target.name }}</strong><small>{{ target.summary }}</small><em v-if="target.tip">{{ target.tip }}</em></span><button type="button" :disabled="!target.visible" :title="target.visible ? `在星图中定位${target.name}` : '当前在地平线下，无法定位'" @click="locateRecommendedBody(target.id)">定位</button></li>
          </ul>
          <p v-else-if="recommendation" class="observing-targets-empty">{{ recommendation.weatherAvailable ? '今夜天气与月光条件有限，建议短时观察亮目标，或改日再安排。' : '天文夜内暂无可观测的亮目标，可改日再安排。' }}</p>

          <div class="current-observation">
            <header><div><p>CONDITION SNAPSHOT / {{ conditionsMomentLabel }}</p><h3>{{ followingRealTime ? '当前' : '预览' }}观测环境</h3></div><span v-if="displayedConditions">{{ displayedConditions.time.slice(11, 16) }} · {{ conditions?.source }}</span><span v-else>等待环境数据</span></header>
            <div class="instrument-grid" :aria-label="`${conditionsMomentLabel}观测条件`">
              <article><span>总云量</span><strong>{{ displayedConditions ? `${Math.round(displayedConditions.cloudCover)}%` : '—' }}</strong><small>低 / 中 / 高云详见下方预报</small></article>
              <article><span>能见度</span><strong>{{ displayedConditions ? `${(displayedConditions.visibilityMeters / 1000).toFixed(1)} km` : '—' }}</strong><small>影响暗星与银河辨识</small></article>
              <article><span>温度 / 露点</span><strong>{{ displayedConditions ? `${displayedConditions.temperature.toFixed(1)}°` : '—' }}</strong><small>{{ displayedConditions ? `露点 ${displayedConditions.dewPoint.toFixed(1)}°` : '等待气温数据' }}</small></article>
              <article><span>湿度</span><strong>{{ displayedConditions ? `${Math.round(displayedConditions.humidity)}%` : '—' }}</strong><small>高湿可能造成镜片结露</small></article>
              <article><span>风速 / 阵风</span><strong>{{ displayedConditions ? `${Math.round(displayedConditions.windSpeed)} / ${Math.round(displayedConditions.windGusts)}` : '—' }}<b v-if="displayedConditions"> km/h</b></strong><small>影响脚架稳定与体感</small></article>
              <article><span>降水</span><strong>{{ displayedConditions ? `${displayedConditions.precipitation.toFixed(1)} mm` : '—' }}</strong><small>{{ conditionsMomentLabel }}预报时段</small></article>
              <article><span>PM₂.₅ / 气溶胶</span><strong>{{ currentAir ? `${Math.round(currentAir.pm25)} μg/m³` : '—' }}</strong><small>{{ currentAir ? `AOD ${currentAir.aerosolOpticalDepth.toFixed(2)} · 透明度参考` : '空气质量源暂不可用' }}</small></article>
              <article class="light-reading" :title="lightPollution ? `${lightPollution.source} · 辐射 ${lightPollution.radiance.toFixed(1)} ${lightPollution.radianceUnit} · SQM/Bortle 为模型估算 · 不计入动态评分` : '当前位置暂无年度卫星光污染数据'"><span>光污染 · 长期环境</span><strong>{{ lightPollution ? `Bortle ≈ ${lightPollution.bortle}` : '—' }}</strong><small>{{ lightPollution ? `SQM ≈ ${lightPollution.sqm.toFixed(1)} · VIIRS ${lightPollution.dataYear ?? ''} · 不计入动态评分` : '年度卫星数据暂不可用' }}</small></article>
            </div>
          </div>
        </section>

        <section class="night-analysis" v-if="nightAnalysis" aria-label="今夜夜空分析">
          <div class="section-heading night-heading"><div><p>NIGHT ANALYSIS / 本地星历</p><h2>今夜夜空</h2></div></div>
          <div class="night-grid">
            <article v-if="nightAnalysis.astronomicalNight"><span>天文夜</span><strong>{{ windowRange(nightAnalysis.astronomicalNight) }}</strong><small>太阳低于地平线 −18°，深空目标不受曙暮光干扰</small></article>
            <article><span>无月黑夜</span><strong>{{ nightAnalysis.moonlessWindows.length ? nightAnalysis.moonlessWindows.map((window) => windowRange(window)).join(' / ') : '今夜无' }}</strong><small>天文夜内月亮在地平线以下，暗弱目标辨识最佳</small></article>
            <article><span>银河核心</span><strong>{{ nightAnalysis.galacticCore.maxTime ? `最高 ${nightAnalysis.galacticCore.maxAltitude}°（${formatTime(nightAnalysis.galacticCore.maxTime)}）` : '今夜低于 20°' }}</strong><small>{{ nightAnalysis.galacticCore.windows.length ? `可见窗口：${nightAnalysis.galacticCore.windows.map((window) => windowRange(window)).join(' / ')}` : '银河核心需要更南方的观测点或更暗的天空' }}</small></article>
          </div>
        </section>

        <section class="forecast-section">
          <div v-if="hourlyForecast.length" class="forecast-matrix" role="table" aria-label="当前整点至明日24点的逐小时观测天气预报">
            <div class="matrix-labels" aria-hidden="true">
              <span><strong>日期</strong><small>月 / 日</small></span>
              <span><strong>时间</strong><small>HH:mm</small></span>
              <span><strong>天气</strong><small>WMO</small></span>
              <span><strong>云量</strong><small>%</small></span>
              <span><strong>高云</strong><small>%</small></span>
              <span><strong>中云</strong><small>%</small></span>
              <span><strong>低云</strong><small>%</small></span>
              <span><strong>气温</strong><small>°C</small></span>
              <span><strong>露点</strong><small>°C</small></span>
              <span><strong>湿度</strong><small>%</small></span>
              <span><strong>降水</strong><small>mm</small></span>
              <span><strong>风速 / 阵风</strong><small>km/h</small></span>
              <span><strong>风向</strong><small>方位</small></span>
              <span><strong>能见度</strong><small>km</small></span>
              <span><strong>气溶胶</strong><small>AOD · 无量纲</small></span>
            </div>
            <div class="matrix-scroll"><div class="matrix-grid">
              <div class="matrix-date-row"><span v-for="group in forecastDateGroups" :key="group.date" :style="{ width: `${group.hours * 66}px` }">{{ group.date }}</span></div>
              <div class="matrix-hours">
              <template v-for="hour in hourlyForecast" :key="hour.time">
                <div class="matrix-hour">
                  <strong class="matrix-cell matrix-time">{{ hour.time.slice(11, 16) }}</strong>
                  <span class="matrix-cell matrix-weather" :class="`tone-${forecastTone('weather', hour)}`" :title="conditionDescription(hour.weatherCode)"><i>{{ weatherGlyph(hour.weatherCode) }}</i><small>{{ conditionDescription(hour.weatherCode) }}</small></span>
                  <span class="matrix-cell" :class="`tone-${forecastTone('cloud', hour)}`">{{ Math.round(hour.cloudCover) }}%</span>
                  <span class="matrix-cell" :class="`tone-${hour.cloudCoverHigh <= 20 ? 'good' : hour.cloudCoverHigh <= 55 ? 'caution' : 'poor'}`">{{ Math.round(hour.cloudCoverHigh) }}%</span>
                  <span class="matrix-cell" :class="`tone-${hour.cloudCoverMid <= 20 ? 'good' : hour.cloudCoverMid <= 55 ? 'caution' : 'poor'}`">{{ Math.round(hour.cloudCoverMid) }}%</span>
                  <span class="matrix-cell" :class="`tone-${hour.cloudCoverLow <= 20 ? 'good' : hour.cloudCoverLow <= 55 ? 'caution' : 'poor'}`">{{ Math.round(hour.cloudCoverLow) }}%</span>
                  <span class="matrix-cell tone-neutral">{{ hour.temperature.toFixed(0) }}°</span>
                  <span class="matrix-cell" :class="`tone-${forecastTone('dew', hour)}`">{{ hour.dewPoint.toFixed(0) }}°</span>
                  <span class="matrix-cell" :class="`tone-${forecastTone('humidity', hour)}`">{{ Math.round(hour.humidity) }}%</span>
                  <span class="matrix-cell" :class="`tone-${forecastTone('precipitation', hour)}`">{{ hour.precipitation.toFixed(1) }}</span>
                  <span class="matrix-cell" :class="`tone-${forecastTone('wind', hour)}`">{{ Math.round(hour.windSpeed) }}<small>{{ Math.round(hour.windGusts) }}</small></span>
                  <span class="matrix-cell tone-neutral matrix-wind"><i :style="{ transform: `rotate(${hour.windDirection + 180}deg)` }">↑</i></span>
                  <span class="matrix-cell" :class="`tone-${forecastTone('visibility', hour)}`">{{ (hour.visibilityMeters / 1000).toFixed(1) }}</span>
                  <span class="matrix-cell" :class="`tone-${forecastTone('aod', hour)}`">{{ airFor(hour.time)?.aerosolOpticalDepth.toFixed(2) ?? '—' }}</span>
                </div>
              </template>
              </div>
            </div></div>
          </div>
          <div v-else class="integration-state"><span>01</span><div><h3>{{ conditionsStatus === 'error' ? '天气预报暂不可用' : '正在连接天气预报' }}</h3><p>从当前整点到明日 24:00 的逐小时预报将在数据加载后显示。</p></div></div>
        </section>
      </section>

      <section v-else-if="activePage === 'sky'" class="sky-map-page page-stack">
        <div class="section-heading"><h2>星图</h2></div>
        <div ref="skySearchControl" class="sky-catalog-search">
          <label for="sky-object-search">搜索天体</label><input id="sky-object-search" v-model="skySearchQuery" role="combobox" aria-autocomplete="list" aria-controls="sky-object-results" :aria-expanded="showSkySearchResults && skySearchResults.length > 0" placeholder="木星、天狼星、M31…" autocomplete="off" @focus="showSkySearchResults = true" @input="showSkySearchResults = true" @keydown.enter.prevent="locateFirstSkySearchMatch" />
          <ul v-if="showSkySearchResults && skySearchResults.length" id="sky-object-results" role="listbox"><li v-for="result in skySearchResults" :key="result.key"><button type="button" role="option" :aria-disabled="!result.visible" :disabled="!result.visible" @click="locateSkySearchResult(result)"><span><strong>{{ result.name }}</strong><small>{{ result.nameEn }} · {{ result.group }}</small></span><i>{{ result.visible ? '定位' : '地平线下' }}</i></button></li></ul>
        </div>
        <section class="horizon-section">
          <div class="horizon-field" :class="{ 'has-location': activeCoordinates, 'is-dragging': skyViewDragging, 'is-auto-turning': skyViewAutoTurning }" :style="horizonFieldStyle" @pointerdown="beginSkyViewDrag" @pointermove="dragSkyView" @pointerup="endSkyViewDrag" @pointercancel="endSkyViewDrag">
            <div class="sky-night" aria-hidden="true" />
            <div class="star-grain" aria-hidden="true" />
            <svg class="altitude-guides" viewBox="0 0 1000 1000" preserveAspectRatio="none" aria-hidden="true">
              <g class="milky-way-band"><path v-for="(path, index) in milkyWayPaths" :key="`milky-${index}`" :d="path" vector-effect="non-scaling-stroke" /></g>
              <g v-for="constellation in constellationGeometry" :key="constellation.name" class="constellation-lines"><path v-for="(path, index) in constellation.paths" :key="`${constellation.name}-${index}`" :d="path" vector-effect="non-scaling-stroke" /></g>
              <path v-for="guide in altitudeGuides" :key="guide.altitude" class="altitude-guide" :d="guide.path" vector-effect="non-scaling-stroke" />
              <g v-if="selectedSkyTrajectory" class="sky-trajectory" :style="{ '--trajectory-tint': selectedSkyTrajectory.tint }">
                <path v-for="(path, index) in selectedSkyTrajectory.pastPaths" :key="`past-${index}`" class="trajectory-past" :d="path" vector-effect="non-scaling-stroke" />
                <path v-for="(path, index) in selectedSkyTrajectory.futurePaths" :key="`future-${index}`" class="trajectory-future" :d="path" vector-effect="non-scaling-stroke" />
              </g>
            </svg>
            <span v-for="constellation in constellationGeometry" :key="`name-${constellation.name}`" v-show="constellation.label" class="constellation-name" :style="{ left: `${(constellation.label?.x ?? 0) * 100}%`, top: `${(constellation.label?.y ?? 0) * 100}%` }">{{ constellation.name }}</span>
            <button v-for="entry in visibleCatalogStars" :key="entry.item.id" class="catalog-star" :class="{ selected: selectedCatalogId === entry.item.id }" type="button" :style="{ left: `${entry.projection.x * 100}%`, top: `${entry.projection.y * 100}%`, '--star-size': `${Math.max(2, 5.4 - entry.item.magnitude)}px` }" :aria-label="`${entry.item.name}，${entry.item.constellation}`" :title="`${entry.item.name} / ${entry.item.nameEn} · ${entry.item.magnitude.toFixed(1)} 等`" @click="selectedCatalogId = entry.item.id" @pointerdown.stop><i /><span>{{ entry.item.name }}</span></button>
            <button v-for="entry in visibleMessierObjects" :key="entry.item.id" class="catalog-messier" :class="{ selected: selectedCatalogId === entry.item.id }" type="button" :style="{ left: `${entry.projection.x * 100}%`, top: `${entry.projection.y * 100}%` }" :title="`${entry.item.name} / ${entry.item.nameEn}`" @click="selectedCatalogId = entry.item.id" @pointerdown.stop><i>◇</i><span>{{ entry.item.nameEn }}</span></button>
            <template v-for="guide in altitudeGuides" :key="`label-${guide.altitude}`"><span v-if="guide.label" class="altitude-label" :style="altitudeLabelStyle(guide.label)">{{ guide.altitude }}°</span></template>
            <div v-for="body in horizonBodies" :key="body.id" class="sky-body" :class="{ 'is-active': expandedBodyId === body.id }" :style="horizonStyle(body)" role="button" tabindex="0" :aria-label="`查看${body.name}详情`" :aria-expanded="expandedBodyId === body.id" @click="revealBody(body.id)" @keydown.enter.prevent="revealBody(body.id)" @keydown.space.prevent="revealBody(body.id)" @pointerdown.stop><i>{{ body.glyph }}</i><span>{{ body.name }}</span></div>
            <div class="horizon-ridge horizon-ridge-far" aria-hidden="true" />
            <div class="horizon-ridge horizon-ridge-near" aria-hidden="true" />
            <div v-if="!activeCoordinates" class="sky-empty"><strong>允许定位后生成本地地平天空</strong><span>星历计算不需要 Key；它只需要你的经纬度与时刻。</span><button type="button" @click="requestLocation">请求位置</button></div>
            <span v-if="activeCoordinates" class="sky-visible-count"><small>当前视野</small>{{ horizonBodies.length + visibleCatalogStars.length + visibleMessierObjects.length }}<small>目标</small></span>
            <div
              v-if="activeCoordinates"
              class="heading-dial"
              role="slider"
              tabindex="0"
              aria-label="星图朝向"
              aria-valuemin="0"
              aria-valuemax="360"
              :aria-valuenow="Number(skyViewAzimuth.toFixed(2))"
              :aria-valuetext="skyViewHeadingLabel"
              @pointerdown.stop="beginSkyViewDrag"
              @pointermove.stop="dragSkyView"
              @pointerup.stop="endSkyViewDrag"
              @pointercancel.stop="endSkyViewDrag"
              @wheel.prevent.stop="turnSkyViewByWheel"
              @keydown.left.prevent="rotateSkyView(-1)"
              @keydown.right.prevent="rotateSkyView(1)"
            >
              <div class="heading-scale" aria-hidden="true">
                <svg class="heading-arc" viewBox="0 0 1000 48" preserveAspectRatio="none" shape-rendering="geometricPrecision" aria-hidden="true"><path d="M 0 45 C 205 24 346 3 500 3 C 654 3 795 24 1000 45" fill="none" stroke="currentColor" stroke-width="1" vector-effect="non-scaling-stroke" stroke-linecap="round" stroke-linejoin="round" /></svg>
                <i v-for="tick in skyHeadingTicks" :key="tick.key" class="heading-tick" :class="{ major: tick.major, direction: tick.direction }" :style="{ left: `${tick.position}%`, bottom: `${tick.lift}px` }"><span v-if="tick.label">{{ tick.label }}</span></i>
              </div>
              <div class="heading-readout"><span>{{ skyViewDirection }}</span><strong>{{ skyViewAzimuth.toFixed(2) }}°</strong></div>
              <i class="heading-lubber" aria-hidden="true" />
            </div>
          </div>
        </section>
        <div class="time-scrubber"><div class="time-scrubber-inner"><div><span>时刻</span><strong>{{ timeLabel }}</strong></div><div class="time-scrubber-track"><output class="time-scrubber-bubble" :style="{ '--scrub-f': scrubFraction }">{{ timeLabel }}</output><input :value="minuteOfDay" type="range" min="0" max="1439" step="1" aria-label="时刻" @input="scheduleMinuteOfDay" @change="commitMinuteOfDay" /></div><button class="time-scrubber-now" type="button" @click="jumpToNow">现在</button><div><span>00:00</span><span>06:00</span><span>12:00</span><span>18:00</span><span>24:00</span></div></div></div>

        <section class="window-section">
          <div class="section-heading"><h2>行星升落</h2></div>
          <div class="window-summary" v-if="twilight"><article><span>日出</span><strong>{{ formatTime(twilight.sunrise) }}</strong></article><article><span>日落</span><strong>{{ formatTime(twilight.sunset) }}</strong></article><article><span>天文晨光</span><strong>{{ formatTime(twilight.astronomicalDawn) }}</strong></article><article><span>天文昏影</span><strong>{{ formatTime(twilight.astronomicalDusk) }}</strong></article></div>
              <div class="time-axis"><div class="axis-labels"><span>00</span><span>06</span><span>12</span><span>18</span><span>24</span></div><article v-for="track in tracks" :key="track.id" class="time-track" :class="{ expanded: expandedBodyId === track.id }"><button :id="`track-trigger-${track.id}`" class="time-track-trigger" type="button" :aria-expanded="expandedBodyId === track.id" :aria-controls="`track-detail-${track.id}`" @click="selectBody(track.id)"><span class="time-track-label"><i :style="{ color: track.tint }">{{ track.glyph }}</i><span>{{ track.name }}<small>{{ observingStatus(track) }}</small></span><strong>{{ track.visible ? `${Math.round(track.altitude)}° ${bearing(track.azimuth)}` : '地平线下' }}</strong></span><span v-if="expandedBodyId !== track.id" class="track-rail" aria-hidden="true"><i v-for="segment in visibleSegmentMap.get(track.id) ?? []" :key="`${segment.left}-${segment.width}`" :style="{ left: `${segment.left}%`, width: `${segment.width}%`, backgroundColor: track.tint }" /><i class="rail-current" :class="{ 'is-below': !(railCurrentMarkers.get(track.id)?.aboveHorizon ?? true) }" :style="{ left: `${railCurrentMarkers.get(track.id)?.left ?? 0}%`, '--chart-tint': track.tint }" /></span><svg v-else class="track-wave" viewBox="0 0 960 88" aria-hidden="true" :style="{ '--chart-tint': track.tint }"><line class="altitude-horizon" x1="0" y1="44" x2="960" y2="44" /><path v-for="(path, index) in expandedWaveGeometry?.belowPaths ?? []" :key="`below-${index}`" class="altitude-below" :d="path" /><path v-for="(path, index) in expandedWaveGeometry?.abovePaths ?? []" :key="`above-${index}`" class="altitude-above" :d="path" /><circle v-if="expandedWaveCurrent" class="altitude-current" :class="{ 'is-below': !expandedWaveCurrent.aboveHorizon }" :cx="expandedWaveCurrent.x" :cy="expandedWaveCurrent.y" r="5" /></svg></button><div v-if="expandedBodyId === track.id" :id="`track-detail-${track.id}`" class="track-detail" role="region" :aria-labelledby="`track-trigger-${track.id}`"><div class="track-detail-state"><span :style="{ backgroundColor: track.tint }" /><div><p>当前位置</p><h3>{{ observingStatus(track) }} · {{ Math.round(track.altitude) }}° 高度角</h3><small>方位角 {{ Math.round(track.azimuth) }}° · {{ bearing(track.azimuth) }}方</small></div></div><dl><div><dt>升起</dt><dd>{{ formatTime(track.rise) }}</dd></div><div><dt>中天</dt><dd>{{ formatTime(track.transit) }}</dd></div><div><dt>落下</dt><dd>{{ formatTime(track.set) }}</dd></div><div><dt>最佳高度</dt><dd>{{ formatTime(track.best) }}</dd></div><div v-if="track.id === 'moon'"><dt>月面亮度</dt><dd>{{ Math.round((track.illumination ?? 0) * 100) }}%</dd></div><div v-else-if="track.id !== 'sun'"><dt>视星等</dt><dd>{{ track.magnitude?.toFixed(1) ?? '—' }}</dd></div><div><dt>{{ track.id === 'moon' ? '地月距离' : '地心距离' }}</dt><dd>{{ formatDistance(track.distanceAu) }}</dd></div></dl><p class="track-caveat" v-if="track.id === 'sun'">太阳观测必须使用合格的全口径太阳滤镜；绝不可用裸眼、墨镜或未加滤镜的器材直视太阳。</p><p class="track-caveat" v-else-if="track.id === 'moon'">月面适合在明暗交界附近观察；满月虽明亮，地形阴影反而较少。</p><p class="track-caveat" v-else>实际可见性还取决于云层、曙暮光、地平线遮挡与本地光污染。</p><div class="track-detail-actions"><button type="button" :disabled="!track.visible" :title="track.visible ? '转动罗盘定位该天体' : '当前在地平线下，无法定位'" @click="locateBody(track.id)">在星图定位</button></div></div></article></div>
        </section>

      </section>

      <section v-else-if="activePage === 'events'" class="events-page page-stack">
        <section class="events-lead">
          <div>
            <h2>未来 30 天的夜空</h2>
          </div>
          <article v-if="nextMoonEvent" class="next-moon-event"><span>下一月相</span><strong>{{ nextMoonEvent.label }}</strong><small>{{ formatEventDate(nextMoonEvent.at) }} · {{ formatTime(nextMoonEvent.at) }}</small></article>
        </section>
        <section class="moon-rhythm" aria-labelledby="moon-rhythm-title">
          <div class="section-heading"><h2 id="moon-rhythm-title">未来月相</h2></div>
          <div class="event-list">
            <article v-for="event in moonEvents" :key="event.target"><time><strong>{{ formatEventDay(event.at) }}</strong><span>{{ formatEventMonth(event.at) }}</span></time><div><p>月相</p><h3>{{ event.label }}</h3><span>{{ event.description }}</span></div><strong>{{ formatTime(event.at) }}</strong></article>
          </div>
        </section>
        <section class="curated-events" aria-labelledby="curated-events-title">
          <div class="section-heading"><h2 id="curated-events-title">未来天象事件</h2></div>
          <div v-if="astronomyEventSources.length" class="event-source-status" aria-label="天象资料源同步状态">
            <article v-for="source in astronomyEventSources" :key="source.code" :class="{ failed: source.success === false }"><i aria-hidden="true" /><div><strong>{{ source.name }}</strong><small>{{ sourceStatusMessage(source) }}</small></div><span>{{ source.coverageStart && source.coverageEnd ? `${source.coverageStart.slice(0, 10)} → ${source.coverageEnd.slice(0, 10)}` : `${source.recordsWritten} 条` }}</span></article>
          </div>
          <div v-if="curatedEvents.length" class="curated-event-list" aria-live="polite">
            <article v-for="event in visibleCuratedEvents" :key="event.id" :class="{ expanded: expandedEventId === event.id }">
              <button type="button" :aria-expanded="expandedEventId === event.id" :aria-controls="`event-detail-${event.id}`" @click="toggleEvent(event)">
                <time>{{ event.dateLabel }}</time><span class="event-kind">{{ event.kind === 'meteor_shower' ? '流星雨' : event.kind.endsWith('eclipse') ? '食' : event.kind.includes('conjunction') || event.kind.startsWith('planetary') ? '行星' : '天象' }}</span><div><h3>{{ event.title }}</h3><p>{{ event.summary }}</p></div><div class="event-visibility" :class="`visibility-${eventVisibilityTone(event)}`"><strong>{{ eventVisibilityLabel(event) }}</strong></div><i aria-hidden="true">⌄</i>
              </button>
              <div v-if="expandedEventId === event.id" :id="`event-detail-${event.id}`" class="curated-event-detail" role="region">
                <dl><div><dt>最佳时段</dt><dd>{{ formatEventMoment(event.local?.bestAt) }}</dd></div><div><dt>可见窗口</dt><dd>{{ event.local?.windowStart ? `${formatEventMoment(event.local.windowStart)} – ${formatEventMoment(event.local.windowEnd)}` : '—' }}</dd></div><div><dt>方位</dt><dd>{{ event.local?.azimuthDegrees != null ? `${Math.round(event.local.azimuthDegrees)}°` : '—' }}</dd></div><div><dt>高度</dt><dd>{{ event.local?.altitudeDegrees != null ? `${Math.round(event.local.altitudeDegrees)}°` : '—' }}</dd></div><div><dt>精度</dt><dd>{{ eventPrecision(event) }}</dd></div><div><dt>核验</dt><dd>{{ event.verifiedAt }}</dd></div><div><dt>来源</dt><dd>{{ event.sourceName }}</dd></div></dl>
                <div class="event-detail-actions"><a :href="event.sourceUrl" target="_blank" rel="noreferrer">查看 {{ event.sourceName }}</a></div>
              </div>
            </article>
          </div>
          <button v-if="hasHiddenCuratedEvents" class="show-all-events" type="button" :aria-expanded="showAllCuratedEvents" @click="toggleAllCuratedEvents">{{ showAllCuratedEvents ? '收起其余天象' : `展开其余 ${curatedEvents.length - 5} 条天象` }}</button>
          <div v-else class="events-empty"><strong>{{ astronomyEventsStatus === 'loading' ? '正在读取未来天象' : '未来 30 天暂无已校订的重点天象' }}</strong><span>{{ astronomyEventsStatus === 'error' ? '天象服务暂不可用；请稍后刷新。' : '无定位时仍会展示全球日历；允许定位后可判断本地可见性。' }}</span><button v-if="astronomyEventsStatus === 'error'" type="button" @click="loadAstronomyEvents">重新加载</button></div>
        </section>
      </section>

      <section v-else class="daily-image-page page-stack" aria-label="每日一图">
        <header class="daily-image-heading">
          <div>
            <h1>宇宙图像窗</h1>
          </div>
        </header>

        <div v-if="imageWallStatus === 'loading' && !imageWall" class="daily-image-state" aria-live="polite"><strong>正在开启图像窗</strong><span>各来源独立读取；某一扇窗延迟不会阻塞其他图像。</span></div>
        <div v-else-if="imageWall" class="image-stream" :class="{ 'is-refreshing': imageWallStatus === 'loading' }" aria-live="polite">
          <section class="image-stream-section" aria-labelledby="recent-images-title">
            <header class="image-stream-heading"><h2 id="recent-images-title">NASA每日一图</h2><p>NASA APOD每日更新</p></header>
            <div class="image-wall">
              <article v-for="window in recentImageWindows" :key="window.id" class="image-window" :class="[`image-window--${window.sourceId}`, { 'is-unavailable': window.status === 'error' }]">
                <div class="image-window-meta"><span>{{ window.sourceName }}</span></div>
                <a v-if="window.status === 'ready'" class="image-window-media" :href="window.sourceUrl" target="_blank" rel="noreferrer" :aria-label="`在来源网站打开：${window.title}`">
                  <img v-if="window.mediaType === 'image' ? window.imageUrl : window.thumbnailUrl" :src="window.mediaType === 'image' ? window.imageUrl : window.thumbnailUrl" :alt="window.title" loading="eager" />
                  <span v-else class="image-window-video">该来源提供视频内容<br />前往官方页面观看</span>
                  <span v-if="window.mediaType === 'video'" class="image-window-play" aria-hidden="true">观看视频</span>
                </a>
                <div v-else class="image-window-missing"><span>×</span><strong>{{ window.title }}</strong><p>{{ window.error }}</p><button type="button" @click="loadImageWall">重新连接</button></div>
                <div class="image-window-copy">
                  <time v-if="window.status === 'ready'">{{ formatImageWindowDate(window.publishedAt) }}</time>
                  <h3>{{ window.title }}</h3>
                  <p v-if="window.summary">{{ window.summary }}</p>
                  <dl v-if="window.status === 'ready'"><div><dt>完整署名</dt><dd>{{ window.credit }}</dd></div><div v-if="window.licenseNote"><dt>使用说明</dt><dd>{{ window.licenseNote }}</dd></div></dl>
                  <div class="daily-image-links"><a :href="window.sourceUrl" target="_blank" rel="noreferrer">打开原始内容</a><a v-if="window.hdUrl" :href="window.hdUrl" target="_blank" rel="noreferrer">高清原图</a></div>
                </div>
              </article>
            </div>
          </section>
          <section class="image-stream-section" aria-labelledby="collection-images-title">
            <header class="image-stream-heading"><h2 id="collection-images-title">继续下潜</h2><p>NASA 图库 · ESO、Webb 与 Hubble</p></header>
            <div class="image-wall">
              <article v-for="window in imageWall.collection" :key="window.id" class="image-window" :class="[`image-window--${window.sourceId}`, { 'is-unavailable': window.status === 'error' }]">
            <div class="image-window-meta"><span>{{ window.sourceName }}</span></div>
            <a v-if="window.status === 'ready'" class="image-window-media" :href="window.sourceUrl" target="_blank" rel="noreferrer" :aria-label="`在来源网站打开：${window.title}`">
              <img v-if="window.mediaType === 'image' ? window.imageUrl : window.thumbnailUrl" :src="window.mediaType === 'image' ? window.imageUrl : window.thumbnailUrl" :alt="window.title" loading="lazy" />
              <span v-else class="image-window-video">该来源提供视频内容<br />前往官方页面观看</span>
              <span v-if="window.mediaType === 'video'" class="image-window-play" aria-hidden="true">观看视频</span>
            </a>
            <div v-else class="image-window-missing"><span>×</span><strong>{{ window.title }}</strong><p>{{ window.error }}</p><button type="button" @click="loadImageWall">重新连接</button></div>
            <div class="image-window-copy">
              <time v-if="window.status === 'ready'">{{ formatImageWindowDate(window.publishedAt) }}</time>
              <h3>{{ window.title }}</h3>
              <p v-if="window.summary">{{ window.summary }}</p>
              <dl v-if="window.status === 'ready'"><div><dt>完整署名</dt><dd>{{ window.credit }}</dd></div><div v-if="window.licenseNote"><dt>使用说明</dt><dd>{{ window.licenseNote }}</dd></div></dl>
              <div class="daily-image-links"><a :href="window.sourceUrl" target="_blank" rel="noreferrer">打开原始内容</a><a v-if="window.hdUrl" :href="window.hdUrl" target="_blank" rel="noreferrer">高清原图</a></div>
            </div>
              </article>
            </div>
          </section>
        </div>
        <div v-else class="daily-image-state is-error" aria-live="polite"><strong>宇宙图像窗暂不可用</strong><span>请检查 AURORA 后端连接后重新加载；不会要求浏览器持有 NASA API Key。</span><button type="button" @click="loadImageWall">重新加载</button></div>
      </section>
    </main>
  </section>
</template>

<style scoped>
/* ============================================================
   AURORA 天文观测 · 单一样式体系（已合并去重）
   填充层级：--sky-deep 页面底 → --sky-sunken 内嵌井 → --sky-panel 浮起面板
   场景例外：星图夜空覆盖层 .sky-night、白天渐变、星点（真实场景色，非 UI 填充）
   ============================================================ */

/* ---------- 主题令牌与整体框架 ---------- */
.sky-shell {
  --sky-ink:#e3e9f1; --sky-muted:#8a96ab; --sky-line:rgba(165,188,222,.16);
  --sky-panel:#131d2e; --sky-sunken:#0d1828; --sky-deep:#0b1322;
  --sky-lunar:#dde3ec; --sky-amber:#c8a361; --sky-cyan:#9db8e8;
  display:grid; grid-template-columns:210px minmax(0,1fr); min-height:100dvh;
  color:var(--sky-ink); background:var(--sky-deep); font-family:var(--font-sans,system-ui,sans-serif);
}
.sky-sidebar { position:sticky; z-index:30; top:0; display:flex; flex-direction:column; min-height:100dvh; padding:30px 20px 18px; border-right:1px solid var(--sky-line); background:linear-gradient(180deg,var(--sky-panel) 0%,var(--sky-deep) 100%); }
.sky-brand { margin-left:5px; color:var(--sky-ink); } /* 图标轨道环向左探出约 5px，右移品牌使图标最左端与下方分隔线左端对齐 */
.location-control { width:100%; }
.sky-location { display:grid; grid-template-columns:22px 1fr auto; gap:10px; align-items:center; width:100%; margin:0 0 28px; padding:0; color:inherit; text-align:left; background:none; border:0; cursor:pointer; }
.sidebar-divider { margin:24px 0 16px; border-top:1px solid var(--sky-line); }
.location-mark { width:14px; height:14px; border:1px solid var(--sky-amber); border-radius:50% 50% 50% 0; transform:rotate(-45deg); }
.location-mark::after { content:""; display:block; width:4px; height:4px; margin:4px; border-radius:50%; background:var(--sky-amber); }
.sky-location strong,.sky-location small { display:block; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.sky-location strong { font-size:11px; font-weight:600; }
.sky-location small { margin-top:4px; color:var(--sky-muted); font:9px var(--font-mono,monospace); }
.sky-location i { color:var(--sky-cyan); font:8px var(--font-mono,monospace); font-style:normal; letter-spacing:.08em; }
.location-editor { position:absolute; z-index:20; top:146px; left:18px; width:286px; padding:13px; color:var(--sky-ink); background:var(--sky-sunken); border:1px solid rgba(157,184,232,.24); border-radius:7px; box-shadow:0 16px 38px rgba(2,8,18,.42); }
.location-editor form + form { margin-top:12px; padding-top:11px; border-top:1px solid var(--sky-line); }
.location-editor label { display:block; margin-bottom:6px; color:var(--sky-muted); font:8px/1.45 var(--font-mono,monospace); letter-spacing:.04em; }
.location-editor form > div { display:flex; gap:5px; }
.location-editor input { min-width:0; width:100%; padding:7px 8px; color:var(--sky-ink); background:var(--sky-deep); border:1px solid var(--sky-line); border-radius:4px; outline:0; font:10px/1.4 var(--font-sans,system-ui,sans-serif); }
.location-editor input::placeholder { color:rgba(138,150,171,.72); }
.location-editor input:focus-visible { border-color:var(--sky-cyan); box-shadow:0 0 0 1px rgba(157,184,232,.12); }
.location-editor button { flex:none; padding:7px 9px; color:var(--sky-cyan); background:rgba(157,184,232,.035); border:1px solid var(--sky-line); border-radius:4px; cursor:pointer; font:9px/1.25 var(--font-sans,system-ui,sans-serif); }
.location-editor button:hover:not(:disabled),.location-editor button:focus-visible { color:var(--sky-ink); background:rgba(157,184,232,.075); border-color:rgba(157,184,232,.58); outline:0; }
.location-editor button:disabled { opacity:.5; cursor:wait; }
.location-gps { width:100%; margin-bottom:12px; letter-spacing:.03em; }
.coordinate-inputs input { width:76px; }
.location-results { max-height:168px; margin:7px 0 0; padding:0; overflow:auto; list-style:none; border-block:1px solid var(--sky-line); }
.location-results li + li { border-top:1px solid var(--sky-line); }
.location-results button { display:flex; justify-content:space-between; gap:10px; width:100%; padding:8px 2px; text-align:left; border:0; border-radius:0; }
.location-results strong,.location-results small { display:block; }
.location-results strong { font-size:9px; font-weight:600; }
.location-results small { color:var(--sky-muted); font:7px/1.4 var(--font-mono,monospace); white-space:nowrap; }
.location-form-error { margin:8px 0 0; color:#f28f84; font-size:8px; line-height:1.5; }
.location-panel-enter-active,.location-panel-leave-active { transition:opacity .14s ease,transform .14s ease; }
.location-panel-enter-from,.location-panel-leave-to { opacity:0; transform:translateY(-4px); }
@media (prefers-reduced-motion:reduce) { .location-panel-enter-active,.location-panel-leave-active { transition:none; } }
.sky-menu { border-top:1px solid var(--sky-line); }
.sky-menu button { position:relative; display:grid; grid-template-columns:25px 1fr auto; align-items:center; width:calc(100% + 40px); min-height:60px; margin-left:-20px; padding:0 20px 0 28px; color:var(--sky-muted); text-align:left; background:none; border:0; border-bottom:1px solid var(--sky-line); cursor:pointer; transition:background .2s,color .2s; }
.sky-menu button::before { position:absolute; top:0; bottom:0; left:0; width:2px; background:var(--sky-amber); content:""; transform:scaleY(0); transform-origin:center; transition:transform .2s; }
.sky-menu button:hover,.sky-menu button.active { color:var(--sky-ink); background:rgba(160,182,216,.045); }
.sky-menu button:focus-visible { outline:1px solid var(--sky-cyan); outline-offset:-2px; }
.sky-menu button.active::before { transform:scaleY(1); }
.sky-menu b { font:9px var(--font-mono,monospace); color:var(--sky-cyan); }
.sky-menu span { font-size:12px; font-weight:600; }
.sky-menu small { display:block; margin-top:3px; color:var(--sky-muted); font:8px var(--font-mono,monospace); letter-spacing:.13em; }
.sky-menu .menu-icon,.sky-menu .menu-icon::before,.sky-menu .menu-icon::after,.sky-menu .menu-icon em,.sky-menu .menu-icon em::before,.sky-menu .menu-icon em::after { position:absolute; display:block; box-sizing:border-box; content:""; }
.sky-menu .menu-icon { position:relative; width:24px; height:24px; color:var(--sky-muted); font-style:normal; opacity:.72; transition:color .2s,opacity .2s,transform .2s; }
.sky-menu button:not(.active):hover .menu-icon { color:var(--sky-cyan); opacity:1; }
.sky-menu button.active .menu-icon { color:var(--sky-amber); opacity:1; transform:translateX(1px); }
.menu-icon-gauge::before { left:2px; bottom:3px; width:20px; height:11px; border:1.5px solid currentColor; border-bottom:0; border-radius:20px 20px 0 0; }
.menu-icon-gauge::after { left:11px; bottom:4px; width:1.5px; height:9px; background:currentColor; border-radius:2px; transform:rotate(38deg); transform-origin:50% 100%; }
.menu-icon-gauge em { left:9px; bottom:1px; width:6px; height:6px; border:1.5px solid currentColor; border-radius:50%; }
.menu-icon-constellation { background:radial-gradient(circle at 12.5% 71%,currentColor 0 1.7px,transparent 1.9px),radial-gradient(circle at 37.5% 42%,currentColor 0 1.2px,transparent 1.45px),radial-gradient(circle at 62.5% 58%,currentColor 0 1.45px,transparent 1.7px),radial-gradient(circle at 87.5% 21%,currentColor 0 2px,transparent 2.25px); }
.menu-icon-constellation::before { top:16.5px; left:3px; width:9.2px; height:1.2px; background:currentColor; border-radius:2px; opacity:.5; transform:rotate(-49.4deg); transform-origin:left center; }
.menu-icon-constellation::after { top:9.7px; left:9px; width:7.2px; height:1.2px; background:currentColor; border-radius:2px; opacity:.5; transform:rotate(33.7deg); transform-origin:left center; }
.menu-icon-constellation em { top:13.8px; left:15px; width:10.8px; height:1.2px; background:currentColor; border-radius:2px; opacity:.5; transform:rotate(-56.3deg); transform-origin:left center; }
.menu-icon-calendar::before { top:4px; left:3px; width:18px; height:17px; border:1.5px solid currentColor; border-radius:3px; background:linear-gradient(currentColor,currentColor) 0 5px / 100% 1.5px no-repeat; }
.menu-icon-calendar::after { top:2px; left:7px; width:1.5px; height:5px; background:currentColor; box-shadow:9px 0 0 currentColor; }
.menu-icon-calendar em { top:13px; left:10px; width:4px; height:4px; background:currentColor; transform:rotate(45deg); }
.menu-icon-image::before { top:3px; left:2px; width:19px; height:17px; border:1.5px solid currentColor; border-radius:2px; }
.menu-icon-image::after { bottom:5px; left:5px; width:13px; height:8px; background:currentColor; clip-path:polygon(0 100%,34% 28%,53% 60%,74% 0,100% 100%); opacity:.78; }
.menu-icon-image em { top:7px; right:5px; width:3.5px; height:3.5px; background:currentColor; border-radius:50%; }
.sidebar-source { margin-top:auto; color:var(--sky-muted); font-size:9px; line-height:1.5; }
.sidebar-source span { display:inline-block; width:5px; height:5px; margin-right:7px; border-radius:50%; background:var(--sky-amber); }
.sky-sidebar > time { margin-top:16px; font:16px var(--font-mono,monospace); color:var(--sky-ink); }
.sky-sidebar > time small { color:var(--sky-muted); font-size:7px; }

/* ---------- 观星条件页 ---------- */
.sky-content-scroll { min-width:0; height:100dvh; overflow:auto; }
.section-heading p,.events-lead p,.moon-copy p,.condition-verdict p,.sky-reading p { margin:0; color:var(--sky-amber); font:9px var(--font-mono,monospace); letter-spacing:.12em; }
.page-stack { max-width:1160px; margin:0 auto; padding:30px clamp(28px,5vw,72px) 80px; }
.condition-hero { display:grid; grid-template-columns:120px 1fr auto; gap:28px; align-items:center; min-height:178px; padding:28px; background:radial-gradient(circle at 10% 20%,rgba(172,193,226,.07),transparent 34%),var(--sky-panel); border:1px solid var(--sky-line); }
.moon-disc { position:relative; width:106px; height:106px; overflow:hidden; border-radius:50%; filter:drop-shadow(0 8px 14px rgba(0,0,0,.26)); }
.moon-disc canvas { display:block; width:100%; height:100%; }
.moon-copy h2 { margin:6px 0; font-size:34px; font-weight:500; }
.moon-copy > span,.moon-copy small { display:block; color:var(--sky-muted); font-size:12px; }
.moon-copy small { max-width:470px; margin-top:17px; line-height:1.6; }
.moon-rise { align-self:stretch; display:flex; flex-direction:column; justify-content:center; min-width:208px; padding-left:28px; border-left:1px solid var(--sky-line); }
.moon-rise-times { display:grid; gap:8px; margin:0 0 8px; }
.moon-rise-times strong { display:grid; grid-template-columns:17px 32px 1fr; gap:4px; align-items:center; color:var(--sky-muted); font-size:10px; font-weight:500; }
.moon-rise-times i { color:var(--sky-amber); font:16px var(--font-mono,monospace); font-style:normal; }
.moon-rise-times time { color:var(--sky-ink); font:15px var(--font-mono,monospace); }
.moon-rise small { color:var(--sky-muted); font-size:10px; }
.moon-rise small b { color:var(--sky-lunar); font:12px var(--font-mono,monospace); font-weight:500; }
.condition-verdict { display:grid; grid-template-columns:180px 1fr; gap:28px; margin-top:18px; padding:26px 28px 22px; border-top:1px solid var(--sky-amber); border-bottom:1px solid var(--sky-line); background:rgba(172,193,226,.03); }
.condition-verdict > div:first-child strong { display:block; margin-top:7px; font:58px/.9 var(--font-mono,monospace); }
.condition-verdict > div:first-child small { color:var(--sky-muted); font-size:15px; }
.score-now i { display:block; margin-top:6px; color:var(--sky-muted); font:8px var(--font-mono,monospace); font-style:normal; letter-spacing:.1em; }
.condition-verdict h2 { margin:2px 0 8px; font-size:24px; font-weight:500; }
.condition-verdict div:nth-child(2) p { color:var(--sky-muted); font-family:inherit; letter-spacing:0; line-height:1.6; }
.score-factors { grid-column:1 / -1; display:grid; grid-template-columns:auto repeat(4,minmax(108px,1fr)); gap:8px; align-items:stretch; margin-top:16px; }
.score-factors > p { display:flex; align-items:center; margin:0; padding-right:10px; color:var(--sky-muted); font:9px var(--font-mono,monospace); letter-spacing:.08em; }
.score-factors span { position:relative; display:grid; grid-template-columns:1fr auto; gap:2px 8px; align-items:end; padding:8px 10px; overflow:hidden; border:1px solid var(--sky-line); border-radius:8px; background:var(--sky-sunken); }
.score-factors span > i { position:absolute; inset:auto 0 0; height:2px; background:#d77d6a; opacity:.62; }
.score-factors small { grid-column:1; color:var(--sky-muted); font-size:9px; }
.score-factors strong { grid-column:2; font:11px var(--font-mono,monospace); }
.score-factors > em { grid-column:2 / -1; display:flex; align-items:center; min-height:34px; color:var(--sky-muted); font-size:10px; font-style:normal; }
.tomorrow-score { grid-column:1 / -1; display:grid; grid-template-columns:auto auto 1fr; gap:12px; align-items:baseline; padding-top:14px; border-top:1px solid var(--sky-line); }
.tomorrow-score span { color:var(--sky-amber); font:9px var(--font-mono,monospace); letter-spacing:.08em; }
.tomorrow-score strong { font:15px var(--font-mono,monospace); }
.tomorrow-score small { color:var(--sky-muted); font-size:10px; }
.observing-console { margin-top:18px; overflow:hidden; border:1px solid var(--sky-line); border-radius:18px; background:rgba(172,193,226,.025); }
.observing-brief { display:grid; grid-template-columns:minmax(0,1fr) auto; gap:28px; align-items:end; padding:25px 28px 20px; }
.advice-window p { margin:0; color:var(--sky-muted); font-size:10px; }
.advice-window h3 { max-width:760px; margin:6px 0 4px; font-size:21px; font-weight:500; letter-spacing:-.02em; }
.advice-window > span { color:var(--sky-muted); font-size:11px; }
.advice-degraded { display:block; margin-top:6px; color:var(--sky-amber); font-size:10px; font-style:normal; }
.observing-targets { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:22px; margin:0; padding:0 28px 24px; list-style:none; }
.observing-targets li { display:grid; grid-template-columns:auto minmax(0,1fr) auto; gap:10px; align-items:start; min-width:0; }
.observing-targets li i { font-style:normal; font-size:22px; line-height:1; }
.observing-targets li strong { display:block; font-size:14px; font-weight:500; }
.observing-targets li small { display:block; margin-top:2px; color:var(--sky-muted); font-size:10px; }
.observing-targets li em { display:block; margin-top:4px; max-width:34ch; color:var(--sky-muted); font-size:9px; font-style:normal; line-height:1.5; }
.observing-targets li > button { align-self:center; padding:5px 10px; color:var(--sky-cyan); font-size:9px; white-space:nowrap; border:1px solid var(--sky-line); border-radius:6px; background:transparent; cursor:pointer; }
.observing-targets li > button:hover:not(:disabled) { border-color:rgba(157,184,232,.5); background:rgba(157,184,232,.08); }
.observing-targets li > button:disabled { opacity:.35; cursor:not-allowed; }
.observing-targets-empty { margin:0; padding:0 28px 24px; color:var(--sky-muted); font-size:12px; line-height:1.6; }
.current-observation { border-top:1px solid var(--sky-line); background:color-mix(in srgb,var(--sky-sunken) 56%,transparent); }
.current-observation > header { display:flex; justify-content:space-between; gap:20px; align-items:end; padding:22px 28px 18px; }
.current-observation > header p { margin:0; color:var(--sky-amber); font:8px var(--font-mono,monospace); letter-spacing:.1em; }
.current-observation > header h3 { margin:5px 0 0; font-size:18px; font-weight:500; }
.current-observation > header > span { color:var(--sky-muted); font-size:9px; text-align:right; }
.instrument-grid { display:grid; grid-template-columns:repeat(4,minmax(0,1fr)); border-top:1px solid var(--sky-line); }
.instrument-grid article { min-height:116px; padding:18px 28px 20px; border-right:1px solid var(--sky-line); }
.instrument-grid article:nth-child(4n) { border-right:0; }
.instrument-grid article:nth-child(n+5) { border-top:1px solid var(--sky-line); }
.instrument-grid span,.instrument-grid small { display:block; color:var(--sky-muted); font-size:10px; }
.instrument-grid strong { display:block; margin:16px 0 8px; font:21px var(--font-mono,monospace); font-weight:500; letter-spacing:-.025em; white-space:nowrap; }
.instrument-grid strong b { color:var(--sky-muted); font-size:10px; font-weight:500; letter-spacing:0; }
.instrument-grid small { line-height:1.4; }
.instrument-grid .light-reading { background:rgba(234,196,120,.025); }
.instrument-grid .light-reading strong { color:var(--sky-amber); font-size:18px; }
.forecast-section,.window-section,.curated-events { margin-top:54px; }
.night-analysis { margin-top:54px; }
.night-grid { display:grid; grid-template-columns:repeat(3,1fr); border:1px solid var(--sky-line); border-radius:18px; overflow:hidden; background:var(--sky-sunken); }
.night-grid article { min-height:118px; padding:18px 20px; border-right:1px solid var(--sky-line); }
.night-grid article:last-child { border-right:0; }
.night-grid span { display:block; color:var(--sky-amber); font:9px var(--font-mono,monospace); letter-spacing:.1em; }
.night-grid strong { display:block; margin:10px 0 8px; font:15px var(--font-mono,monospace); line-height:1.45; }
.night-grid small { display:block; color:var(--sky-muted); font-size:10px; line-height:1.55; }
.section-heading { display:flex; align-items:end; margin-bottom:18px; }
.section-heading h2 { margin:6px 0 0; font-size:25px; font-weight:500; letter-spacing:-.04em; }
.forecast-matrix { display:grid; grid-template-columns:102px minmax(0,1fr); overflow:hidden; border:1px solid var(--sky-line); background:var(--sky-sunken); }
.matrix-labels { display:grid; grid-template-rows:34px 38px 58px repeat(12,44px); background:var(--sky-panel); border-right:1px solid var(--sky-line); }
.matrix-labels span { display:grid; place-content:center; justify-items:center; gap:2px; padding:0 8px; color:var(--sky-muted); text-align:center; border-bottom:1px solid rgba(165,188,222,.1); }
.matrix-labels strong { font-size:10px; font-weight:500; line-height:1.1; }
.matrix-labels small { color:currentColor; font:8px var(--font-mono,monospace); line-height:1.1; opacity:.72; white-space:nowrap; }
.matrix-labels span:nth-child(1),.matrix-labels span:nth-child(2) { color:var(--sky-cyan); }
.matrix-scroll { min-width:0; overflow-x:auto; }
.matrix-grid { width:max-content; }
.matrix-date-row,.matrix-hours { display:flex; }
.matrix-date-row { height:34px; }
.matrix-date-row span { display:grid; flex:0 0 auto; place-items:center; color:var(--sky-muted); font:9px var(--font-mono,monospace); background:rgba(160,182,216,.045); border-right:1px solid rgba(165,188,222,.11); border-bottom:1px solid rgba(165,188,222,.1); }
.matrix-hour { display:grid; grid-template-rows:38px 58px repeat(12,44px); width:66px; flex:0 0 66px; }
.matrix-cell { display:grid; place-items:center; min-width:66px; color:var(--sky-ink); font:12px var(--font-mono,monospace); border-right:1px solid rgba(165,188,222,.11); border-bottom:1px solid rgba(165,188,222,.1); }
.matrix-time { color:var(--sky-ink); font-size:13px; font-weight:500; background:rgba(160,182,216,.06); }
.matrix-weather { gap:2px; background:rgba(160,182,216,.06); }
.matrix-weather i { font-size:20px; font-style:normal; line-height:1; }
.matrix-weather small,.matrix-cell small { display:block; margin-top:2px; color:currentColor; font-size:8px; opacity:.75; }
.matrix-wind i { display:block; color:var(--sky-ink); font-size:20px; font-style:normal; }
.tone-good { color:#63d9a4; background:rgba(52,168,116,.28); }
.tone-caution { color:#ecc257; background:rgba(196,150,46,.28); }
.tone-poor { color:#f28f84; background:rgba(198,74,64,.3); }
.tone-neutral { color:#c8cbcd; background:rgba(160,182,216,.06); }
.integration-state { display:flex; gap:22px; padding:28px; border:1px solid var(--sky-line); background:rgba(172,193,226,.025); }
.integration-state > span { color:var(--sky-amber); font:18px var(--font-mono,monospace); }
.integration-state h3 { margin:0 0 7px; font-size:16px; font-weight:500; }
.integration-state p { max-width:590px; margin:0; color:var(--sky-muted); font-size:12px; line-height:1.65; }

/* ---------- 星图（天幕） ---------- */
.sky-catalog-search { position:relative; z-index:8; display:grid; grid-template-columns:auto minmax(180px,320px); gap:10px; align-items:center; width:max-content; max-width:100%; margin:-4px 0 12px auto; }
.sky-catalog-search label { color:var(--sky-muted); font-size:10px; font-weight:600; letter-spacing:.02em; }
.sky-catalog-search input { width:100%; padding:8px 10px; color:var(--sky-ink); background:var(--sky-sunken); border:1px solid var(--sky-line); border-radius:6px; outline:0; font:10px/1.4 var(--font-sans,system-ui,sans-serif); }
.sky-catalog-search input:focus { border-color:var(--sky-cyan); }
.sky-catalog-search ul { position:absolute; z-index:12; top:calc(100% + 6px); right:0; width:320px; max-width:85vw; margin:0; padding:0; overflow:hidden; list-style:none; background:rgba(13,24,40,.98); border:1px solid rgba(157,184,232,.28); border-radius:8px; box-shadow:0 18px 42px rgba(0,0,0,.34); }
.sky-catalog-search li + li { border-top:1px solid var(--sky-line); }
.sky-catalog-search li button { display:flex; justify-content:space-between; align-items:center; gap:14px; width:100%; padding:9px 11px; color:inherit; text-align:left; background:transparent; border:0; cursor:pointer; }
.sky-catalog-search li button:hover:not(:disabled),.sky-catalog-search li button:focus-visible { background:rgba(157,184,232,.08); outline:0; }
.sky-catalog-search li button:disabled { opacity:.42; cursor:not-allowed; }
.sky-catalog-search li strong,.sky-catalog-search li small { display:block; }
.sky-catalog-search li strong { font-size:10px; font-weight:500; }
.sky-catalog-search li small { margin-top:2px; color:var(--sky-muted); font:8px var(--font-mono,monospace); }
.sky-catalog-search li i { color:var(--sky-cyan); font:8px var(--font-mono,monospace); font-style:normal; }
.horizon-section { border:1px solid var(--sky-line); background:var(--sky-panel); border-radius:18px; overflow:hidden; }
.horizon-field { position:relative; min-height:440px; overflow:hidden; background:linear-gradient(180deg,#2c5074 0%,#4c7295 55%,#9db2c3 100%); }
.horizon-field::before { position:absolute; inset:0; background:radial-gradient(circle at 18% 13%,rgba(172,193,226,.2) 0 1px,transparent 1.5px),radial-gradient(circle at 76% 24%,rgba(172,193,226,.14) 0 1px,transparent 1.5px),radial-gradient(circle at 61% 9%,rgba(172,193,226,.2) 0 1px,transparent 1.5px); content:""; opacity:calc(.7 * (1 - var(--sky-daylight,0))); }
.star-grain { position:absolute; inset:0; opacity:calc(.24 * (1 - var(--sky-daylight,0))); background-image:radial-gradient(#c9d8ee 1px,transparent 1px); background-size:79px 83px; }
.sky-night { position:absolute; inset:0; background:linear-gradient(180deg,#081322 0%,#0e2034 66%,#0a1322 100%); opacity:calc(1 - var(--sky-daylight,0)); pointer-events:none; transition:opacity .45s ease; }
.altitude-guides { position:absolute; z-index:1; inset:0; width:100%; height:100%; overflow:hidden; pointer-events:none; }
.altitude-guide,.sky-trajectory path { fill:none; stroke-linecap:round; stroke-linejoin:round; }
.altitude-guide { stroke:color-mix(in srgb,rgba(10,20,36,.72) calc(var(--sky-daylight,0) * 100%),rgba(165,188,222,.3)); stroke-width:1; stroke-dasharray:3 4; }
.milky-way-band path { fill:none; stroke:rgba(207,220,238,.11); stroke-width:36; stroke-linecap:round; filter:blur(8px); opacity:calc(1 - var(--sky-daylight,0)); }
.constellation-lines path { fill:none; stroke:rgba(157,184,232,.3); stroke-width:1; stroke-linecap:round; }
.constellation-name { position:absolute; z-index:2; padding:2px 5px; color:rgba(199,214,234,.58); font:8px var(--font-mono,monospace); letter-spacing:.12em; background:rgba(7,17,30,.45); transform:translate(-50%,-50%); pointer-events:none; }
.catalog-star,.catalog-messier { position:absolute; z-index:2; width:0; height:0; padding:0; color:var(--sky-ink); background:transparent; border:0; cursor:pointer; }
.catalog-star > i { position:absolute; display:block; width:var(--star-size); height:var(--star-size); border-radius:50%; background:#e7eef8; box-shadow:0 0 calc(var(--star-size) * 2) rgba(205,224,248,.75); transform:translate(-50%,-50%); }
.catalog-star > span,.catalog-messier > span { position:absolute; top:6px; left:5px; padding:2px 4px; color:rgba(224,234,247,.72); font:8px var(--font-mono,monospace); white-space:nowrap; background:rgba(7,17,30,.58); }
.catalog-star:hover > i,.catalog-star:focus-visible > i,.catalog-star.selected > i { background:#fff; box-shadow:0 0 12px #d8e8ff; transform:translate(-50%,-50%) scale(1.5); }
.catalog-star:focus-visible,.catalog-messier:focus-visible { outline:1px solid var(--sky-cyan); outline-offset:7px; }
.catalog-messier > i { position:absolute; color:rgba(200,163,97,.9); font:15px var(--font-mono,monospace); font-style:normal; transform:translate(-50%,-50%); }
.catalog-messier:hover > i,.catalog-messier.selected > i { color:#f0c777; text-shadow:0 0 12px rgba(240,199,119,.7); }
.sky-trajectory .trajectory-past { stroke:var(--trajectory-tint); stroke-width:1.35; stroke-dasharray:3 5; opacity:.3; }
.sky-trajectory .trajectory-future { stroke:var(--trajectory-tint); stroke-width:1.65; opacity:.72; }
.altitude-label { position:absolute; z-index:2; margin-left:4px; padding:2px 4px; color:color-mix(in srgb,rgba(10,20,36,.88) calc(var(--sky-daylight,0) * 100%),rgba(184,202,227,.78)); font:8px var(--font-mono,monospace); white-space:nowrap; background:color-mix(in srgb,rgba(236,243,249,.72) calc(var(--sky-daylight,0) * 100%),rgba(7,17,30,.62)); border-radius:2px; pointer-events:none; transform:translateY(-50%); }
.horizon-ridge { position:absolute; z-index:0; right:-3%; left:-3%; pointer-events:none; }
.horizon-ridge-far { bottom:3px; height:52px; background:linear-gradient(180deg,rgba(53,82,112,.72),rgba(18,34,52,.92)); clip-path:polygon(0 86%,8% 70%,17% 78%,29% 45%,38% 68%,47% 52%,57% 76%,68% 54%,78% 74%,89% 48%,100% 72%,100% 100%,0 100%); opacity:.48; }
.horizon-ridge-near { bottom:0; height:39px; background:linear-gradient(180deg,#17283b 0%,#0a1422 100%); clip-path:polygon(0 82%,11% 58%,21% 76%,33% 51%,43% 82%,56% 63%,66% 79%,79% 54%,90% 74%,100% 62%,100% 100%,0 100%); opacity:.62; }
.compass { position:absolute; right:28px; bottom:14px; left:28px; display:flex; justify-content:space-between; color:rgba(165,188,222,.48); font:9px var(--font-mono,monospace); }
.sky-body { position:absolute; z-index:3; width:0; height:0; cursor:pointer; animation:body-arrive .5s cubic-bezier(.22,1,.36,1); transition:opacity .45s ease; }
.sky-body i { position:absolute; left:0; top:0; display:grid; width:25px; height:25px; place-items:center; transform:translate(-50%,-50%); border-radius:50%; color:#0d1828; background:var(--body-tint); box-shadow:0 0 18px color-mix(in srgb,var(--body-tint) 45%,transparent); font-size:16px; font-style:normal; transition:transform .18s ease, box-shadow .18s ease; }
.sky-body:hover i, .sky-body:focus-visible i { transform:translate(-50%,-50%) scale(1.22); box-shadow:0 0 24px color-mix(in srgb,var(--body-tint) 75%,transparent); }
.sky-body:focus-visible { outline:2px solid color-mix(in srgb,var(--body-tint) 60%,white); outline-offset:6px; border-radius:8px; }
.sky-body.is-active i { transform:translate(-50%,-50%) scale(1.3); box-shadow:0 0 30px color-mix(in srgb,var(--body-tint) 90%,transparent); }
.sky-body span { position:absolute; left:0; top:17px; padding:2px 5px; color:var(--sky-ink); background:rgba(11,19,34,.8); font-size:9px; white-space:nowrap; transform:translateX(-50%); }
.sky-reading { position:absolute; z-index:3; left:28px; bottom:38px; }
.sky-reading strong { display:block; margin:5px 0; font:36px var(--font-mono,monospace); }
.sky-reading span { color:var(--sky-muted); font-size:11px; }
.sky-empty { position:absolute; z-index:3; top:50%; left:50%; display:grid; gap:10px; width:min(380px,82%); transform:translate(-50%,-50%); text-align:center; }
.sky-empty strong { font-size:18px; font-weight:500; }
.sky-empty span { color:var(--sky-muted); font-size:12px; line-height:1.5; }
.sky-empty button { justify-self:center; padding:8px 12px; color:var(--sky-amber); border:1px solid color-mix(in srgb,var(--sky-amber) 45%,transparent); background:transparent; cursor:pointer; }
.sky-visible-count { position:absolute; z-index:5; top:18px; right:48px; display:flex; gap:5px; align-items:baseline; color:var(--sky-ink); font:12px var(--font-mono,monospace); white-space:nowrap; pointer-events:none; }
.sky-visible-count small { color:rgba(199,214,234,.62); font:8px var(--font-mono,monospace); }
.horizon-field.has-location { cursor:grab; touch-action:none; }
.horizon-field.is-dragging { cursor:grabbing; }

/* ---------- 航向拨盘：地平线下方的贯穿式半圆刻度，不占用低空天体区域 ---------- */
.heading-dial { position:absolute; z-index:4; right:28px; bottom:2px; left:28px; height:58px; color:var(--sky-ink); cursor:ew-resize; user-select:none; touch-action:none; outline:none; }
.heading-dial:focus-visible { outline:none; }
.heading-dial:focus-visible .heading-readout { outline:1px solid var(--sky-cyan); outline-offset:3px; }
.horizon-field.is-dragging .heading-dial { cursor:grabbing; }
.heading-scale { position:absolute; z-index:5; inset:0 0 11px; overflow:visible; }
.heading-arc { position:absolute; inset:0; width:100%; height:100%; color:color-mix(in srgb,rgba(214,228,240,.58) calc(var(--sky-daylight,0) * 100%),rgba(165,188,222,.32)); pointer-events:none; }
.heading-tick { position:absolute; width:1px; height:5px; background:color-mix(in srgb,rgba(207,223,237,.44) calc(var(--sky-daylight,0) * 100%),rgba(165,188,222,.32)); transform:translateX(-50%); }
.heading-tick.major { height:10px; background:color-mix(in srgb,rgba(218,232,243,.66) calc(var(--sky-daylight,0) * 100%),rgba(165,188,222,.68)); }
.heading-tick span { position:absolute; z-index:7; top:calc(100% + 3px); left:50%; padding:0 2px; color:color-mix(in srgb,#0a1422 calc(var(--sky-daylight,0) * 100%),rgba(184,202,227,.72)); font:7px var(--font-mono,monospace); white-space:nowrap; background:color-mix(in srgb,rgba(244,248,252,.9) calc(var(--sky-daylight,0) * 100%),rgba(7,17,30,.52)); border-radius:2px; text-shadow:color-mix(in srgb,rgba(255,255,255,.55) calc(var(--sky-daylight,0) * 100%),rgba(0,0,0,.9)); transform:translateX(-50%); }
.heading-tick.direction span { color:var(--sky-amber); font-size:9px; font-weight:600; letter-spacing:.08em; background:color-mix(in srgb,rgba(244,248,252,.95) calc(var(--sky-daylight,0) * 100%),rgba(7,17,30,.86)); }
.heading-readout { position:absolute; top:11px; left:50%; z-index:5; display:flex; gap:5px; align-items:baseline; justify-content:center; color:color-mix(in srgb,#0a1422 calc(var(--sky-daylight,0) * 100%),var(--sky-ink)); white-space:nowrap; text-shadow:color-mix(in srgb,rgba(255,255,255,.5) calc(var(--sky-daylight,0) * 100%),rgba(0,0,0,.72)); transform:translateX(-50%); }
.heading-readout span { color:var(--sky-amber); font-size:9px; font-weight:600; }
.heading-readout strong { font:12px var(--font-mono,monospace); font-weight:500; }
.heading-lubber { position:absolute; top:-1px; left:50%; z-index:6; width:8px; height:7px; background:var(--sky-amber); clip-path:polygon(0 0,100% 0,50% 100%); transform:translateX(-50%); }
.heading-dial:hover .heading-arc,.horizon-field.is-dragging .heading-arc { color:color-mix(in srgb,rgba(229,238,246,.72) calc(var(--sky-daylight,0) * 100%),rgba(165,188,222,.42)); }

/* ---------- 时间条（时刻 · 现在 · 进度条） ---------- */
.time-scrubber { margin-top:16px; border:1px solid var(--sky-line); border-radius:18px; background:var(--sky-sunken); }.time-scrubber-inner { display:grid; grid-template-columns:88px minmax(0,1fr) 66px; column-gap:3px; row-gap:0; align-items:center; width:100%; margin:0; padding:14px 28px 11px 24px; }
.time-scrubber-inner > div:first-child span,.time-scrubber-inner > div:last-child { color:var(--sky-muted); font:9px var(--font-mono,monospace); }
.time-scrubber-inner > div:first-child { padding-left:8px; }
.time-scrubber-inner > div:first-child strong { display:block; margin-top:4px; font:17px var(--font-mono,monospace); }
.time-scrubber-inner > div:last-child { grid-column:2; display:flex; justify-content:space-between; margin-top:-6px; padding:0 1px; }
.time-scrubber-now { margin-left:7px; padding:7px 14px; color:var(--sky-cyan); font:9px var(--font-mono,monospace); letter-spacing:.1em; background:rgba(157,184,232,.06); border:1px solid rgba(165,188,222,.3); border-radius:6px; cursor:pointer; transition:background .2s,border-color .2s; }
.time-scrubber-now:hover { background:rgba(157,184,232,.12); border-color:rgba(165,188,222,.5); }
.time-scrubber-track { position:relative; display:flex; align-items:center; min-width:0; }
.time-scrubber input { flex:1; min-width:0; width:100%; height:16px; appearance:none; -webkit-appearance:none; background:transparent; }
.time-scrubber input::-webkit-slider-runnable-track { height:4px; border-radius:2px; background:rgba(157,184,232,.25); }
.time-scrubber input::-webkit-slider-thumb { -webkit-appearance:none; width:14px; height:14px; margin-top:-5px; border-radius:50%; background:var(--sky-cyan); box-shadow:0 0 0 3px rgba(157,184,232,.18); }
.time-scrubber input::-moz-range-track { height:4px; border-radius:2px; background:rgba(157,184,232,.25); }
.time-scrubber input::-moz-range-progress { height:4px; border-radius:2px; background:rgba(157,184,232,.45); }
.time-scrubber input::-moz-range-thumb { width:14px; height:14px; border:0; border-radius:50%; background:var(--sky-cyan); }
.time-scrubber-bubble { position:absolute; z-index:2; bottom:calc(100% + 6px); left:calc(var(--scrub-f,.5) * (100% - 14px) + 7px); transform:translateX(-50%); padding:2px 6px; color:var(--sky-cyan); font:9px var(--font-mono,monospace); white-space:nowrap; background:rgba(11,19,34,.94); border:1px solid rgba(157,184,232,.35); border-radius:3px; pointer-events:none; }
.time-scrubber-bubble::after { content:""; position:absolute; top:100%; left:50%; transform:translateX(-50%); border:4px solid transparent; border-top-color:rgba(157,184,232,.35); }

/* ---------- 行星升落（24h 时间带 + 原位展开） ---------- */
.window-summary { display:grid; grid-template-columns:repeat(4,1fr); border:1px solid var(--sky-line); background:var(--sky-sunken); border-radius:18px; overflow:hidden; }
.window-summary article { padding:14px; border-right:1px solid var(--sky-line); }
.window-summary article:last-child { border-right:0; }
.window-summary span { display:block; color:var(--sky-muted); font-size:9px; }
.window-summary strong { display:block; margin-top:7px; font:17px var(--font-mono,monospace); }
.time-axis { display:grid; width:100%; gap:6px; margin:20px auto 0; border-top:0; }
.axis-labels { display:flex; justify-content:space-between; padding:0 12px 2px 174px; color:var(--sky-muted); font:9px var(--font-mono,monospace); }
.time-track { overflow:hidden; border:1px solid rgba(165,188,222,.14); border-radius:18px; background:var(--sky-panel); }
.time-track:hover { border-color:rgba(165,188,222,.3); }
.time-track.expanded { border-color:rgba(165,188,222,.4); }
.time-track-trigger { display:grid; grid-template-columns:156px minmax(0,1fr); width:100%; min-height:46px; padding:0; color:var(--sky-ink); text-align:left; background:none; border:0; cursor:pointer; }
.time-track-trigger:hover { background:color-mix(in srgb,var(--sky-cyan) 5%,var(--sky-panel)); }
.time-track.expanded .time-track-trigger { background:transparent; }
.time-track-trigger:focus-visible { position:relative; z-index:1; outline:1px solid var(--sky-cyan); outline-offset:-1px; }
.time-track-label { display:grid; grid-template-columns:22px minmax(0,1fr) auto; gap:8px; align-items:center; padding:0 14px; border-right:0; }
.time-track-label > i { font-size:1.125rem; font-style:normal; }
.time-track-label > span { font-size:.875rem; font-weight:600; }
.time-track-label small { display:block; margin-top:2px; color:var(--sky-muted); font-size:.625rem; }
.time-track-label strong { color:var(--sky-muted); font:9px var(--font-mono,monospace); }
.track-rail { position:relative; align-self:center; height:8px; min-height:8px; margin:0 14px 0 0; overflow:hidden; border-radius:999px; background:var(--sky-sunken); }
.track-rail i { position:absolute; top:0; bottom:0; opacity:.8; border-radius:inherit; }
.track-rail .rail-current { position:absolute; top:50%; bottom:auto; z-index:1; width:7px; height:7px; margin:0; opacity:1; border-radius:50%; background:var(--chart-tint); box-shadow:0 0 6px color-mix(in srgb,var(--chart-tint) 55%,transparent); transform:translate(-50%,-50%); }
.track-rail .rail-current.is-below { background:var(--sky-muted); box-shadow:none; opacity:.72; }
.track-wave { display:block; align-self:center; width:100%; height:88px; margin:3px 14px 3px 0; animation:wave-reveal .38s cubic-bezier(.22,1,.36,1) both; }
.track-detail { display:grid; grid-template-columns:minmax(0,1fr) auto; gap:14px 26px; padding:12px 20px 18px; background:transparent; animation:track-detail-reveal .3s cubic-bezier(.22,1,.36,1) both; }
.track-detail-state { display:flex; align-items:center; gap:11px; grid-column:1 / -1; }
.track-detail-state > span { width:8px; height:8px; border-radius:50%; }
.track-detail-state p { margin:0; color:var(--sky-amber); font:8px var(--font-mono,monospace); letter-spacing:.1em; }
.track-detail-state h3 { margin:3px 0; font-size:1rem; font-weight:500; }
.track-detail-state small { color:var(--sky-muted); font-size:10px; }
.track-detail dl { grid-column:1 / -1; display:grid; grid-template-columns:repeat(6,minmax(0,1fr)); gap:16px 22px; margin:0; border:0; }
.track-detail dl div { padding:0; border:0; }
.track-detail dt { color:var(--sky-muted); font-size:9px; }
.track-detail dd { margin:5px 0 0; font:13px var(--font-mono,monospace); }
.track-caveat { grid-column:1; margin:0; color:var(--sky-muted); font-size:10px; line-height:1.6; align-self:end; max-width:66ch; padding-top:2px; }
.track-detail-actions { display:flex; flex-wrap:wrap; align-items:start; justify-content:end; gap:8px; align-self:end; }
.track-detail-actions button { padding:8px 10px; color:var(--sky-cyan); font-size:10px; border:1px solid var(--sky-line); background:transparent; cursor:pointer; }
.track-detail-actions button:hover:not(:disabled) { background:color-mix(in srgb,var(--sky-cyan) 8%,transparent); }
.track-detail-actions button:disabled { opacity:.35; cursor:not-allowed; }
.track-detail-actions button:focus-visible { outline:1px solid var(--sky-cyan); outline-offset:2px; }

/* 展开详情：高度角-时间曲线（从地平线下穿出、拱起、再回到地平线下） */
.altitude-horizon { stroke:rgba(226,230,232,.3); stroke-width:1; }
.altitude-below { fill:none; stroke:rgba(226,230,232,.2); stroke-width:1.5; stroke-dasharray:3 4; }
.altitude-above { fill:none; stroke:var(--chart-tint); stroke-width:2.6; stroke-linecap:round; stroke-linejoin:round; }
.altitude-current { fill:var(--chart-tint); stroke:var(--sky-deep); stroke-width:2; filter:drop-shadow(0 2px 4px rgba(0,0,0,.42)); }
.altitude-current.is-below { fill:var(--sky-muted); opacity:.76; }

/* ---------- 天象 ---------- */
.events-lead { display:grid; grid-template-columns:minmax(0,1fr) 184px; align-items:end; gap:36px; padding:34px 0 30px; border-top:2px solid var(--sky-amber); }
.events-lead > div { min-width:0; }
.events-lead h2 { max-width:620px; margin:0 0 10px; font-size:34px; font-weight:500; }
.events-lead span { display:block; max-width:620px; color:var(--sky-muted); font-size:12px; line-height:1.6; }
.next-moon-event { flex:0 0 184px; padding:0 0 2px 20px; border-left:1px solid var(--sky-line); }
.next-moon-event span,.next-moon-event small { display:block; color:var(--sky-muted); font-size:10px; }
.next-moon-event strong { display:block; margin:6px 0 4px; color:var(--sky-lunar); font-size:21px; font-weight:500; }
.moon-rhythm { padding-top:16px; }
.event-list { margin-top:18px; border-top:1px solid var(--sky-line); }
.event-list article { display:grid; grid-template-columns:82px 1fr auto; gap:22px; align-items:center; min-height:100px; border-bottom:1px solid var(--sky-line); }
.event-list time { padding:0 14px; border-right:1px solid var(--sky-line); }
.event-list time strong,.event-list time span { display:block; }
.event-list time strong { font:29px var(--font-mono,monospace); }
.event-list time span { margin-top:3px; color:var(--sky-muted); font:9px var(--font-mono,monospace); }
.event-list article p { margin:0; color:var(--sky-amber); font:8px var(--font-mono,monospace); }
.event-list h3 { margin:5px 0; font-size:17px; font-weight:500; }
.event-list article div span { color:var(--sky-muted); font-size:11px; }
.event-list > article > strong { padding-right:10px; color:var(--sky-cyan); font:12px var(--font-mono,monospace); }
.curated-events { margin-top:64px; }
.curated-event-list { margin-top:18px; border-top:1px solid var(--sky-line); }
.curated-event-list > article { border-bottom:1px solid var(--sky-line); }
.curated-event-list > article > button { display:grid; grid-template-columns:115px 76px minmax(0,1fr) minmax(168px,226px) 20px; gap:16px; width:100%; min-height:106px; padding:16px 8px 16px 0; color:inherit; text-align:left; background:transparent; border:0; cursor:pointer; }
.curated-event-list time { align-self:start; color:var(--sky-ink); font:13px var(--font-mono,monospace); line-height:1.45; }
.event-kind { align-self:start; color:var(--sky-amber); font:9px var(--font-mono,monospace); letter-spacing:.1em; }
.curated-event-list h3 { margin:0 0 5px; font-size:18px; font-weight:500; }
.curated-event-list p { margin:0; color:var(--sky-muted); font-size:11px; line-height:1.6; }
.event-visibility { --visibility-color:var(--sky-muted); align-self:center; display:inline-grid; justify-self:start; min-height:0; padding:6px 10px 7px; border:1px solid color-mix(in srgb,var(--visibility-color) 34%,var(--sky-line)); border-radius:999px; background:color-mix(in srgb,var(--visibility-color) 8%,transparent); }
.event-visibility strong { color:var(--visibility-color); font-size:12px; font-weight:600; line-height:1.2; }
.event-visibility.visibility-observable { --visibility-color:#63d9a4; }
.event-visibility.visibility-limited { --visibility-color:#ecc257; }
.event-visibility.visibility-not-visible { --visibility-color:#f28f84; }
.event-visibility.visibility-non-visual { --visibility-color:#9db8e8; }
.event-visibility.visibility-unknown,.event-visibility.visibility-pending { --visibility-color:#91a0a9; }
.curated-event-list > article > button > i { align-self:start; color:var(--sky-muted); font-style:normal; transition:transform .2s ease; }
.curated-event-list > article.expanded > button > i { transform:rotate(180deg); }
.curated-event-list > article > button:hover h3,.curated-event-list > article > button:focus-visible h3 { color:var(--sky-cyan); }
.curated-event-list > article > button:focus-visible { outline:1px solid var(--sky-cyan); outline-offset:-1px; }
.curated-event-detail { padding:0 44px 22px 207px; animation:track-detail-reveal .3s cubic-bezier(.22,1,.36,1) both; }
.curated-event-detail dl { display:grid; grid-template-columns:repeat(4,minmax(0,1fr)); gap:14px; margin:0; }
.curated-event-detail dt { color:var(--sky-muted); font-size:9px; }
.curated-event-detail dd { margin:5px 0 0; color:var(--sky-ink); font-size:11px; line-height:1.55; }
.curated-event-detail > p { max-width:70ch; margin:20px 0; color:var(--sky-muted); font-size:11px; line-height:1.7; }
.event-source-status { display:grid; grid-template-columns:repeat(auto-fit,minmax(180px,1fr)); gap:1px; margin-bottom:18px; overflow:hidden; background:var(--sky-line); border:1px solid var(--sky-line); }
.event-source-status article { display:grid; grid-template-columns:7px minmax(0,1fr); gap:9px; align-items:start; min-height:76px; padding:13px; background:var(--sky-sunken); }
.event-source-status article > i { width:6px; height:6px; margin-top:4px; border-radius:50%; background:#63d9a4; }
.event-source-status article.failed > i { background:#f28f84; }
.event-source-status strong,.event-source-status small,.event-source-status span { display:block; }
.event-source-status strong { font-size:10px; font-weight:500; }
.event-source-status small { margin-top:4px; color:var(--sky-muted); font-size:8px; line-height:1.45; }
.event-source-status span { grid-column:2; color:var(--sky-cyan); font:8px var(--font-mono,monospace); }
.event-detail-actions { display:flex; gap:18px; align-items:center; }
.event-detail-actions button,.event-detail-actions a { padding:0; color:var(--sky-cyan); font-size:10px; text-decoration:none; background:transparent; border:0; cursor:pointer; }
.event-detail-actions button:hover,.event-detail-actions a:hover,.event-detail-actions button:focus-visible,.event-detail-actions a:focus-visible { color:var(--sky-ink); outline:0; }
.events-empty { display:grid; gap:6px; margin-top:18px; padding:26px 0; border-top:1px solid var(--sky-line); border-bottom:1px solid var(--sky-line); }
.events-empty strong { font-size:15px; font-weight:500; }
.events-empty span { color:var(--sky-muted); font-size:11px; }
.events-empty button { justify-self:start; padding:0; color:var(--sky-cyan); font-size:11px; background:transparent; border:0; cursor:pointer; }
.events-empty button:hover,.events-empty button:focus-visible { color:var(--sky-ink); outline:0; }
.show-all-events { display:block; width:100%; padding:16px 0; color:var(--sky-cyan); font-size:11px; text-align:left; background:transparent; border:0; border-bottom:1px solid var(--sky-line); cursor:pointer; }
.show-all-events:hover,.show-all-events:focus-visible { color:var(--sky-ink); outline:0; }
/* ---------- 宇宙图像窗：一面来源可追溯的连续观测墙 ---------- */
.daily-image-page { padding-top:52px; }
.daily-image-heading { padding-bottom:30px; border-bottom:1px solid var(--sky-line); }
.daily-image-heading h1 { max-width:560px; margin:0; font-size:clamp(2.2rem,4vw,4.25rem); font-weight:500; letter-spacing:-.04em; line-height:.98; }
.daily-image-heading p { max-width:57ch; margin:15px 0 0; color:var(--sky-muted); font-size:12px; line-height:1.7; }
.image-stream.is-refreshing { opacity:.66; }
.image-stream-section + .image-stream-section { margin-top:68px; }
.image-stream-heading { display:flex; align-items:center; justify-content:space-between; gap:24px; margin-bottom:0; padding:18px 0; }
.image-stream-heading h2 { margin:0; font-size:clamp(1.45rem,2.2vw,2rem); font-weight:500; letter-spacing:-.03em; }
.image-stream-heading p { max-width:54ch; margin:0; color:var(--sky-muted); font-size:11px; line-height:1.65; text-align:right; }
.image-wall { display:grid; grid-template-columns:repeat(6,minmax(0,1fr)); border-top:1px solid var(--sky-line); border-bottom:1px solid var(--sky-line); }
.image-window { grid-column:span 2; min-width:0; padding:25px 24px 31px; border-right:1px solid var(--sky-line); border-bottom:1px solid var(--sky-line); }
.image-window:nth-child(5n + 3),.image-window:nth-child(5n + 5) { border-right:0; }
.image-window:nth-child(5n + 4),.image-window:nth-child(5n + 5) { grid-column:span 3; }
.image-window-meta { display:flex; justify-content:space-between; gap:14px; min-height:18px; margin-bottom:14px; color:var(--sky-cyan); font:9px var(--font-mono,monospace); letter-spacing:.045em; line-height:1.4; }
.image-window-meta time { color:var(--sky-muted); text-align:right; }
.image-window-media { position:relative; display:block; overflow:hidden; aspect-ratio:1.36; color:var(--sky-ink); background:#05080d; text-decoration:none; }
.image-window-media::after { position:absolute; inset:0; background:linear-gradient(180deg,transparent 58%,rgba(3,7,12,.62)); content:""; pointer-events:none; }
.image-window-media img { display:block; width:100%; height:100%; object-fit:cover; transition:transform .7s cubic-bezier(.16,1,.3,1); }
.image-window-media:hover img { transform:scale(1.025); }
.image-window-media:focus-visible { outline:1px solid var(--sky-ink); outline-offset:3px; }
.image-window-video { display:grid; place-items:center; width:100%; height:100%; color:var(--sky-muted); font-size:12px; line-height:1.7; text-align:center; }
.image-window-play { position:absolute; z-index:1; right:13px; bottom:12px; padding:7px 9px; color:var(--sky-ink); font:9px var(--font-mono,monospace); letter-spacing:.05em; background:rgba(5,8,13,.78); border:1px solid rgba(226,233,241,.28); }
.image-window-copy { padding-top:17px; }
.image-window-copy > time { display:block; margin-bottom:10px; color:var(--sky-amber); font:9px var(--font-mono,monospace); letter-spacing:.05em; }
.image-window-copy h3 { margin:0; font-size:clamp(1.2rem,1.85vw,1.7rem); font-weight:500; letter-spacing:-.025em; line-height:1.08; }
.image-window-copy > p { display:-webkit-box; -webkit-box-orient:vertical; -webkit-line-clamp:3; overflow:hidden; margin:12px 0 18px; color:var(--sky-muted); font-size:11px; line-height:1.7; }
.image-window-copy dl { display:grid; gap:12px; margin:0; padding:14px 0; border-top:1px solid var(--sky-line); }
.image-window-copy dt { color:var(--sky-muted); font-size:9px; }
.image-window-copy dd { margin:4px 0 0; font-size:10px; line-height:1.55; }
.image-window-missing { display:grid; min-height:220px; align-content:center; gap:10px; padding:24px; color:var(--sky-muted); background:rgba(7,12,19,.68); border:1px solid var(--sky-line); }
.image-window-missing > span { color:var(--sky-amber); font-size:24px; line-height:1; }
.image-window-missing strong { color:var(--sky-ink); font-size:15px; font-weight:500; }
.image-window-missing p { margin:0; font-size:11px; line-height:1.65; }
.image-window-missing button { justify-self:start; padding:0; color:var(--sky-cyan); font:11px inherit; background:transparent; border:0; cursor:pointer; }
.image-window-missing button:hover,.image-window-missing button:focus-visible { color:var(--sky-ink); outline:0; }
.daily-image-links { display:flex; flex-wrap:wrap; gap:14px 18px; margin-top:21px; }
.daily-image-links a { color:var(--sky-cyan); font-size:10px; text-decoration:none; }
.daily-image-links a:hover,.daily-image-links a:focus-visible { color:var(--sky-ink); outline:0; }
.daily-image-state { display:grid; gap:7px; max-width:560px; padding:62px 0; border-bottom:1px solid var(--sky-line); }
.daily-image-state strong { font-size:20px; font-weight:500; }
.daily-image-state span { color:var(--sky-muted); font-size:12px; line-height:1.7; }
.daily-image-state code { color:var(--sky-cyan); font:inherit; }
.daily-image-state button { justify-self:start; margin-top:8px; padding:0; color:var(--sky-cyan); font-size:11px; background:transparent; border:0; cursor:pointer; }
.daily-image-state button:hover,.daily-image-state button:focus-visible { color:var(--sky-ink); outline:0; }
/* ---------- 动画与响应式 ---------- */
@keyframes body-arrive { from { opacity:0; } to { opacity:var(--body-opacity,1); } }
@keyframes track-detail-reveal { from { opacity:.2; clip-path:inset(0 0 100% 0); } to { opacity:1; clip-path:inset(0 0 0 0); } }
@keyframes wave-reveal { from { opacity:.35; clip-path:inset(0 100% 0 0); } to { opacity:1; clip-path:inset(0 0 0 0); } }
@media (max-width:900px) {
  .sky-shell { grid-template-columns:1fr; }
  .sky-sidebar { position:relative; min-height:auto; padding:20px; }
  .sky-location { margin:22px 0; }
  .sky-menu { display:grid; grid-template-columns:repeat(4,1fr); }
  .sky-menu button { width:100%; margin:0; padding:9px 7px; min-height:58px; grid-template-columns:18px minmax(0,1fr) 20px; }
  .sky-menu .menu-icon { display:block; transform:scale(.82); }
  .sky-menu button.active .menu-icon { transform:translateX(1px) scale(.82); }
  .sidebar-source,.sky-sidebar > time { display:none; }
  .sky-content-scroll { height:auto; overflow:visible; }
  .score-factors { grid-template-columns:repeat(2,minmax(0,1fr)); }
  .score-factors > p { grid-column:1 / -1; padding-right:0; }
  .score-factors > em { grid-column:1 / -1; }
  .observing-targets { grid-template-columns:1fr; gap:16px; }
  .instrument-grid { grid-template-columns:repeat(2,1fr); }
  .instrument-grid article:nth-child(4n) { border-right:1px solid var(--sky-line); }
  .instrument-grid article:nth-child(2n) { border-right:0; }
  .instrument-grid article:nth-child(n+3) { border-top:1px solid var(--sky-line); }
  .axis-labels { padding-left:174px; }
  .track-detail { padding-left:18px; }
  .track-detail dl { grid-template-columns:repeat(3,minmax(0,1fr)); }
  .track-caveat { grid-column:1 / -1; }
  .track-detail-actions { grid-column:1 / -1; justify-content:start; }
  .curated-event-list > article > button { grid-template-columns:100px 62px minmax(0,1fr) 18px; }
  .curated-event-list > article > button > strong { grid-column:3; }
  .curated-event-detail { padding-left:178px; }
  .curated-event-detail dl { grid-template-columns:repeat(2,minmax(0,1fr)); }
  .image-window { padding:23px 20px 29px; }
}
@media (max-width:620px) {
  .page-stack { padding:20px 20px 55px; }
  .events-lead { display:block; }
  .next-moon-event { margin-top:26px; padding:14px 0 0; border-top:1px solid var(--sky-line); border-left:0; }
  .curated-event-list > article > button { grid-template-columns:1fr 18px; gap:8px; }
  .curated-event-list time,.event-kind,.curated-event-list strong { grid-column:1; }
  .curated-event-list > article > button > i { grid-column:2; grid-row:1; }
  .curated-event-detail { padding:0 0 22px; }
  .curated-event-detail dl { grid-template-columns:1fr; }
  .condition-hero { grid-template-columns:80px 1fr; gap:16px; padding:18px; }
  .moon-disc { width:72px; height:72px; }
  .moon-copy h2 { font-size:25px; }
  .moon-rise { grid-column:1 / -1; min-width:0; padding:14px 0 0; border-top:1px solid var(--sky-line); border-left:0; }
  .condition-verdict { grid-template-columns:1fr; }
  .condition-verdict > small { grid-column:1; }
  .observing-brief { grid-template-columns:1fr; gap:14px; align-items:start; padding:22px 20px 18px; }
  .observing-targets { padding:0 20px 22px; }
  .observing-targets-empty { padding:0 20px 22px; }
  .current-observation > header { align-items:start; flex-direction:column; padding:20px 20px 16px; }
  .current-observation > header > span { text-align:left; }
  .instrument-grid { grid-template-columns:1fr; }
  .instrument-grid article,.instrument-grid article:nth-child(2n),.instrument-grid article:nth-child(4n) { border-right:0; }
  .instrument-grid article:nth-child(n+2) { border-top:1px solid var(--sky-line); }
  .window-summary { grid-template-columns:repeat(2,1fr); }
  .window-summary article:nth-child(2) { border-right:0; }
  .night-grid { grid-template-columns:1fr; }
  .night-grid article { border-right:0; border-bottom:1px solid var(--sky-line); }
  .night-grid article:last-child { border-bottom:0; }
  .time-axis { gap:6px; }
  .axis-labels { padding-left:0; }
  .time-track-trigger { grid-template-columns:1fr; min-height:52px; }
  .time-track-label { min-height:52px; padding:0 16px; }
  .track-rail { align-self:center; height:8px; min-height:8px; margin:0 16px 12px; }
  .track-wave { align-self:center; width:100%; height:88px; margin:0 16px 10px; }
  .track-detail { grid-template-columns:1fr; gap:16px; padding:16px; }
  .track-detail dl { grid-template-columns:repeat(2,minmax(0,1fr)); gap:14px 18px; }
  .track-detail-actions { grid-column:1; justify-content:start; }
  .event-list article { grid-template-columns:62px 1fr; gap:12px; }
  .event-list > article > strong { grid-column:2; padding-bottom:12px; }
  .daily-image-page { padding-top:28px; }
  .image-stream-section + .image-stream-section { margin-top:48px; }
  .image-stream-heading { display:block; }
  .image-stream-heading p { margin-top:8px; text-align:left; }
  .image-wall { grid-template-columns:1fr; }
  .image-window,.image-window:nth-child(5n + 4),.image-window:nth-child(5n + 5) { grid-column:1; padding:23px 0 29px; border-right:0; }
  .image-window:last-child { border-bottom:0; }
  .image-window-media { aspect-ratio:1.45; }
  .horizon-field { min-height:380px; }
  .time-scrubber-inner { grid-template-columns:1fr; row-gap:12px; }
  .time-scrubber-now { margin-left:0; }
  .time-scrubber-inner > div:last-child { grid-column:1; }
  .heading-dial { right:20px; left:20px; }
  .heading-readout { top:11px; gap:4px; }
}
@media (prefers-reduced-motion:reduce) { .sky-menu button,.sky-body,.track-wave,.track-detail,.sky-night { transition:none; animation:none; } }
</style>
