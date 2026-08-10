<!--
THESIS: SKY is a local observing decision room, not a weather dashboard or a generic star-map clone.
OWN-WORLD: midnight blue-black, lunar white, horizon amber, flat instrument rails, and one continuous sky field.
STORY: establish local conditions, inspect the sky and its 24-hour rhythm, then plan around the next event.
FIRST VIEWPORT: a permanent left observatory rail frames a wide lunar condition reading and a plain-language verdict.
FORM: desktop field observatory; three focused workspaces share one clock, one location, and evidence boundaries.
-->
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { fetchObserverPlace, fetchObservingConditions, fetchMoonDay, fetchObservingScore, fetchLightPollution, type ObservingConditions, type MoonDay, type ObservingScore, type LightPollution } from '../api'
import { bodies, bearing, calculateTrack, calculateTwilight, daylightFactor, moonPhase, observeTips, observingStatus, upcomingMoonPhases, analyzeNight, type BodyId, type BodyTrack, type NightAnalysis } from '../astronomy'
import moonNearsideTexture from '../assets/solar/2k_moon.jpg'
import AuroraBrand from './AuroraBrand.vue'

type SkyPage = 'conditions' | 'sky' | 'events'

const emit = defineEmits<{ home: [] }>()

const menu = [
  { id: 'conditions' as const, numeral: 'Ⅰ', label: '观星条件', en: 'CONDITIONS' },
  { id: 'sky' as const, numeral: 'Ⅱ', label: '星图', en: 'SKY MAP' },
  { id: 'events' as const, numeral: 'Ⅲ', label: '天象', en: 'EVENTS' },
]

function pageFromHash(): SkyPage {
  const hash = window.location.hash
  if (hash === '#astronomy-events') return 'events'
  if (hash === '#astronomy-sky' || hash === '#astronomy-tonight' || hash === '#astronomy-windows' || hash === '#astronomy-targets') return 'sky'
  return 'conditions'
}

const activePage = ref<SkyPage>(pageFromHash())
const now = ref(new Date())
const minuteOfDay = ref(now.value.getHours() * 60 + now.value.getMinutes())
const skyViewAzimuth = ref(180)
const skyViewDragging = ref(false)
const locationLabel = ref('等待位置授权')
const latitude = ref<number | null>(null)
const longitude = ref<number | null>(null)
const locationStatus = ref<'idle' | 'locating' | 'resolving' | 'located' | 'partial' | 'denied' | 'unavailable'>('idle')
const conditions = ref<ObservingConditions | null>(null)
const conditionsStatus = ref<'idle' | 'loading' | 'ready' | 'error'>('idle')
// 后端数据源：月相每日、评分按时刻、光污染按地点；任一失败均回退本地计算/诚实占位。
const moonDay = ref<MoonDay | null>(null)
const backendScore = ref<ObservingScore | null>(null)
const lightPollution = ref<LightPollution | null>(null)
let scoreTimer: number | undefined
let scoreController: AbortController | undefined
const expandedBodyId = ref<BodyId | null>(null)
const skyLeaving = ref(false)
const moonCanvas = ref<HTMLCanvasElement | null>(null)
let clock: number | undefined
let minuteAnimationFrame: number | undefined
let locationRevision = 0
let locationLookupController: AbortController | undefined
let conditionsController: AbortController | undefined
let homeExitTimer: number | undefined
let moonTexturePixels: ImageData | undefined
let moonTextureWidth = 0
let moonTextureHeight = 0
let skyViewStartX = 0
let skyViewStartAzimuth = 180
const skyViewFieldOfView = 120
const skyHorizonBase = 15
const skyAltitudeSpan = 77

const activeCoordinates = computed(() => latitude.value != null && longitude.value != null ? { latitude: latitude.value, longitude: longitude.value } : null)
const simulatedTime = computed(() => {
  const value = new Date(now.value)
  value.setHours(Math.floor(minuteOfDay.value / 60), minuteOfDay.value % 60, 0, 0)
  return value
})
const timeLabel = computed(() => formatTime(simulatedTime.value))
const scrubFraction = computed(() => (minuteOfDay.value / 1439).toFixed(4))
const coordinateLabel = computed(() => activeCoordinates.value
  ? `${activeCoordinates.value.latitude >= 0 ? '北纬' : '南纬'} ${Math.abs(activeCoordinates.value.latitude).toFixed(1)}° · ${activeCoordinates.value.longitude >= 0 ? '东经' : '西经'} ${Math.abs(activeCoordinates.value.longitude).toFixed(1)}°`
  : '允许定位后生成本地数据')
const locationAction = computed(() => locationStatus.value === 'located' ? 'GPS' : locationStatus.value === 'resolving' || locationStatus.value === 'locating' ? '请求中' : '重试')
const moon = computed(() => moonPhase(now.value))
const elevation = computed(() => conditions.value?.elevation ?? 0)
const timezoneName = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
const tracks = computed<BodyTrack[]>(() => activeCoordinates.value
  ? bodies.map((body) => calculateTrack(body, simulatedTime.value, activeCoordinates.value!.latitude, activeCoordinates.value!.longitude, elevation.value))
  : [])
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
// 展开后行内就地替换时间条的高度角-时间波形（只在展开时计算）
const expandedWave = computed(() => {
  const track = expandedBodyId.value ? tracks.value.find((item) => item.id === expandedBodyId.value) : null
  return track ? altitudeCurve(track, ALTITUDE_CHART_ROW, minuteOfDay.value) : null
})
// 收起态时间条上的"当前时刻"小点：与展开波形同一平滑正弦源（idealWaveAltitude > 0 判定地平线上/下，x 与 minuteOfDay/1439 同源）
const railCurrentMarkers = computed(() => new Map(
  tracks.value.map((track) => [
    track.id,
    { left: minuteOfDay.value / 1439 * 100, aboveHorizon: idealWaveAltitude(track, minuteOfDay.value) > 0 },
  ]),
))
const twilight = computed(() => activeCoordinates.value ? calculateTwilight(simulatedTime.value, activeCoordinates.value.latitude, activeCoordinates.value.longitude, elevation.value) : null)
// 今夜夜空分析：天文夜窗口、无月黑夜与银河核心可见时段（纯本地星历）。
const nightAnalysis = computed<NightAnalysis | null>(() => activeCoordinates.value ? analyzeNight(now.value, activeCoordinates.value.latitude, activeCoordinates.value.longitude, elevation.value) : null)
function windowRange(window: NightAnalysis['astronomicalNight']) {
  return window ? `${formatTime(window.start)} – ${formatTime(window.end)}` : '—'
}
// 白昼系数 0（深夜）→ 1（正午）：以当天当地日出/日落为边界，天文晨光/昏影为渐变过渡带；星图视场与天体淡出共用。
const daylight = computed(() => twilight.value ? daylightFactor(twilight.value, simulatedTime.value) : 0)
const visibleSkyBodies = computed(() => tracks.value.filter((track) => track.visible))
const horizonBodies = computed(() => visibleSkyBodies.value.filter((track) => Math.abs(azimuthDelta(track.azimuth, skyViewAzimuth.value)) <= skyViewFieldOfView / 2 + 4))
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
const currentAir = computed(() => conditions.value?.airQuality[0] ?? null)
const observingScore = computed(() => {
  const forecast = conditions.value
  if (!forecast) return null
  const visibilityBonus = Math.min(24, forecast.current.visibilityMeters / 1000 * 2.4)
  const cloudPenalty = forecast.current.cloudCover * .52
  const moonPenalty = moon.value.illumination * 22
  const precipitationPenalty = forecast.current.precipitation > 0 ? 18 : 0
  return Math.max(0, Math.min(100, Math.round(50 + visibilityBonus - cloudPenalty - moonPenalty - precipitationPenalty)))
})
const scoreVerdict = computed(() => backendScore.value?.verdict ?? (observingScore.value == null ? '正在读取环境预报' : observingScore.value >= 70 ? '条件较好，适合安排观测' : observingScore.value >= 45 ? '条件一般，优先安排亮目标' : '条件受限，建议短时观察亮目标'))
// 后端评分优先（含时刻维度与光污染因子），未就绪时退回本地启发式。
const scorePanel = computed(() => backendScore.value?.score ?? observingScore.value)
// 评分因子分解（"为什么是这个分"）：后端 ScoreFactors 优先（含时刻维度），
// 未就绪或预报窗口外退回本地启发式——与后端公式完全一致，保证展示与评分同源。
interface ScoreFactorPart {
  key: string
  label: string
  value: number // 正数为加分、负数为扣分
  kind: 'bonus' | 'penalty'
}
const scoreFactors = computed<ScoreFactorPart[] | null>(() => {
  const forecast = conditions.value
  if (!forecast) return null
  const backend = backendScore.value
  const parts: ScoreFactorPart[] = [{ key: 'base', label: '基线', value: 50, kind: 'bonus' }]
  if (backend?.withinForecastWindow && backend.score != null) {
    parts.push({ key: 'visibility', label: '能见度', value: backend.factors.visibilityBonus, kind: 'bonus' })
    parts.push({ key: 'cloud', label: '云量', value: -backend.factors.cloudPenalty, kind: 'penalty' })
    parts.push({ key: 'moon', label: '月光', value: -backend.factors.moonPenalty, kind: 'penalty' })
    if (backend.factors.precipitationPenalty > 0) parts.push({ key: 'precip', label: '降水', value: -backend.factors.precipitationPenalty, kind: 'penalty' })
    if (backend.factors.lightPollutionPenalty > 0) parts.push({ key: 'light', label: '光污染', value: -backend.factors.lightPollutionPenalty, kind: 'penalty' })
  } else {
    parts.push({ key: 'visibility', label: '能见度', value: Math.min(24, forecast.current.visibilityMeters / 1000 * 2.4), kind: 'bonus' })
    parts.push({ key: 'cloud', label: '云量', value: -forecast.current.cloudCover * .52, kind: 'penalty' })
    parts.push({ key: 'moon', label: '月光', value: -moon.value.illumination * 22, kind: 'penalty' })
    if (forecast.current.precipitation > 0) parts.push({ key: 'precip', label: '降水', value: -18, kind: 'penalty' })
  }
  return parts
})
function factorBarWidth(part: ScoreFactorPart) {
  // 以各部分绝对值中最大者为满刻度（基线 50 除外），扣分按同刻度反向展示。
  const scale = scoreFactors.value?.filter((item) => item.key !== 'base').reduce((max, item) => Math.max(max, Math.abs(item.value)), 0) ?? 30
  return `${Math.max(6, Math.abs(part.value) / Math.max(1, scale) * 100)}%`
}
// 今夜建议（动态推荐）：器材选择 → 逐小时评分 × 天文夜窗口 × 逐目标月光/曙暮光 → 最佳窗口与推荐目标。
const equipment = ref<'naked' | 'binoculars' | 'telescope'>('naked')
const equipmentOptions = [
  { id: 'naked' as const, label: '裸眼', magnitudeLimit: 4.5, note: '只推荐最亮目标' },
  { id: 'binoculars' as const, label: '双筒', magnitudeLimit: 8.5, note: '可尝试天王星' },
  { id: 'telescope' as const, label: '望远镜', magnitudeLimit: Infinity, note: '全部九体' },
]
const equipmentNote = computed(() => equipmentOptions.find((item) => item.id === equipment.value)?.note ?? '')
// 某时刻的天体高度（取 track.samples 中最接近的采样点）
function altitudeAt(track: BodyTrack | undefined, at: Date): number | null {
  if (!track?.samples.length) return null
  let nearest = track.samples[0]
  let nearestDiff = Infinity
  for (const sample of track.samples) {
    const diff = Math.abs(sample.at.getTime() - at.getTime())
    if (diff < nearestDiff) {
      nearestDiff = diff
      nearest = sample
    }
  }
  return nearest.altitude
}
const recommendation = computed(() => {
  const hourly = conditions.value?.scores ?? []
  if (!hourly.length || !tracks.value.length || !twilight.value) return null
  const sunTrack = tracks.value.find((track) => track.id === 'sun')
  const moonTrack = tracks.value.find((track) => track.id === 'moon')
  const nightStart = twilight.value.astronomicalDusk
  const nightEnd = twilight.value.astronomicalDawn
  const inAstronomicalNight = (at: Date) => nightStart && nightEnd && at.getTime() >= nightStart.getTime() && at.getTime() <= nightEnd.getTime()
  const magnitudeLimit = equipmentOptions.find((item) => item.id === equipment.value)?.magnitudeLimit ?? Infinity

  // 最佳窗口：仅在天文夜内的逐小时评分里取最高分；夜内无评分时退回全天最高。
  const darkHours = nightStart && nightEnd
    ? hourly.filter((item) => {
      const at = new Date(item.time)
      return at.getTime() >= nightStart!.getTime() && at.getTime() <= nightEnd!.getTime()
    })
    : []
  const windowPool = darkHours.length ? darkHours : hourly
  const bestScore = Math.max(...windowPool.map((item) => item.score))
  if (bestScore < 40) {
    return { window: null, windowLabel: '未来 24 小时天气与月光条件有限', targets: [] }
  }
  const bestHour = windowPool.find((item) => item.score === bestScore) ?? null
  const windowLabel = bestHour ? `最佳窗口 ${bestHour.time.slice(11, 16)} · 评分 ${bestScore}` : '未来 24 小时未见理想窗口'

  // 逐目标：在当天每 15 分钟采样上评估 暗夜 × 高度 × 月光，取综合最佳时刻。
  const targets = tracks.value
    .filter((track) => track.id !== 'sun')
    .map((track) => {
      let best: { at: Date; altitude: number; score: number } | null = null
      for (const sample of track.samples) {
        if (!inAstronomicalNight(sample.at) || sample.altitude <= 10) continue
        const sunAltitude = altitudeAt(sunTrack, sample.at)
        if (sunAltitude == null || sunAltitude > -18) continue // 天文夜硬边界（防极昼误差）
        const moonAltitude = altitudeAt(moonTrack, sample.at) ?? -90
        const moonInterference = moonAltitude > 0 ? moonPanel.value.illumination * .65 : 0
        const score = sample.altitude * (1 - moonInterference)
        if (!best || score > best.score) best = { at: sample.at, altitude: sample.altitude, score }
      }
      return { track, best }
    })
    .filter((entry) => entry.best && (entry.track.magnitude ?? Infinity) <= magnitudeLimit)
    .sort((a, b) => (b.best?.score ?? -Infinity) - (a.best?.score ?? -Infinity))
    .slice(0, 3)
    .map((entry) => {
      const best = entry.best!
      const moonAltitude = altitudeAt(moonTrack, best.at) ?? -90
      const moonNote = moonAltitude > 0 ? (moonPanel.value.illumination > .5 ? '月光较强' : '月光较弱') : '无月光干扰'
      return {
        id: entry.track.id,
        name: entry.track.name,
        glyph: entry.track.glyph,
        tint: entry.track.tint,
        summary: `最佳 ${formatTime(best.at)} · ${Math.round(best.altitude)}° 高 · ${moonNote}`,
        tip: observeTips[entry.track.id],
      }
    })
  return { window: bestHour, windowLabel, targets }
})
const hourlyForecast = computed(() => conditions.value?.hourly.slice(0, 24) ?? [])
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

function formatTime(value: Date | null) {
  if (!value) return '—'
  return new Intl.DateTimeFormat('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false }).format(value)
}

function formatDistance(distanceAu: number | null) {
  if (distanceAu == null) return '—'
  if (distanceAu < .01) return `${Math.round(distanceAu * 149_597_870)} km`
  return `${distanceAu.toFixed(2)} AU`
}

function conditionDescription(code: number) {
  if (code <= 1) return '晴朗'
  if (code <= 3) return '多云'
  if (code <= 48) return '雾霾'
  if (code <= 67) return '降水'
  if (code <= 77) return '降雪'
  return '雷暴'
}

function weatherGlyph(code: number) {
  if (code <= 1) return '☼'
  if (code <= 3) return '☁'
  if (code <= 48) return '≋'
  if (code <= 67) return '☂'
  if (code <= 77) return '❄'
  return 'ϟ'
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

const ALTITUDE_CHART_ROW: AltitudeChartMetrics = { width: 960, height: 88, horizonY: 52, scale: .58 }

interface AltitudeCurve {
  fullPath: string
  belowPaths: string[]
  abovePaths: string[]
  current: { x: number; y: number; aboveHorizon: boolean }
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
  return value.getHours() * 60 + value.getMinutes() + value.getSeconds() / 60
}

function positiveMinuteDelta(value: number) {
  return (value % 1440 + 1440) % 1440
}

function idealWaveAltitude(track: BodyTrack, minute: number) {
  const sampledPeak = Math.max(...track.samples.map((sample) => sample.altitude))
  const peakAltitude = Math.max(50, Math.min(60, sampledPeak))
  const riseMinute = track.rise ? minuteInDay(track.rise) : null
  const setMinute = track.set ? minuteInDay(track.set) : null
  if (riseMinute != null && setMinute != null) {
    // 一天 = 一条连续正弦波：升起→落下为峰值 A 的拱弧（实线），其余时间为同幅反向
    // 正弦凹谷（虚线）——虚线延续拱弧的弧度、在升/落点平滑接续；不会因取模产生
    // 多余的第二次拱起，地平线以下的虚线自然连续。
    const duration = Math.max(60, positiveMinuteDelta(setMinute - riseMinute))
    const elapsed = positiveMinuteDelta(minute - riseMinute)
    if (elapsed <= duration) {
      return peakAltitude * Math.sin(Math.PI * elapsed / duration)
    }
    const troughSpan = 1440 - duration
    const belowElapsed = elapsed - duration
    return -peakAltitude * Math.sin(Math.PI * belowElapsed / troughSpan)
  }
  // 极昼极夜或数据缺失：以中天为中心取一段理想可见弧（与原实现一致）
  const transitMinute = track.transit ? minuteInDay(track.transit) : 720
  const elapsed = positiveMinuteDelta(minute - positiveMinuteDelta(transitMinute - 360))
  return peakAltitude * Math.sin(Math.PI * elapsed / 720)
}

function altitudeCurve(track: BodyTrack, metrics: AltitudeChartMetrics, currentMinute: number): AltitudeCurve {
  const pointCount = 193
  const points = Array.from({ length: pointCount }, (_, index) => {
    const minute = index / (pointCount - 1) * 1440
    return {
      x: index / (pointCount - 1) * metrics.width,
      y: altitudeToY(idealWaveAltitude(track, minute), metrics),
    }
  })
  const currentAltitude = idealWaveAltitude(track, currentMinute)
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
    current: {
      x: currentMinute / 1439 * metrics.width,
      y: altitudeToY(currentAltitude, metrics),
      aboveHorizon: currentAltitude > 0,
    },
  }
}

function normalizeAzimuth(value: number) {
  return (value % 360 + 360) % 360
}

function azimuthDelta(target: number, origin: number) {
  const delta = normalizeAzimuth(target - origin)
  return delta > 180 ? delta - 360 : delta
}

function horizonStyle(track: BodyTrack) {
  const relativeAzimuth = azimuthDelta(track.azimuth, skyViewAzimuth.value)
  // 白昼时除太阳外的天体按昼光淡出（月球保留低可见度，它常出现在白天天空）。
  const opacity = track.id === 'sun' ? 1 : track.id === 'moon' ? Math.max(.2, 1 - daylight.value) : Math.max(0, 1 - daylight.value)
  return {
    left: `${50 + relativeAzimuth / (skyViewFieldOfView / 2) * 45}%`,
    bottom: `${skyHorizonBase + Math.min(skyAltitudeSpan, Math.max(0, track.altitude) / 90 * skyAltitudeSpan)}%`,
    '--body-tint': track.tint,
    opacity: String(opacity),
  }
}

function horizonAltitudePercent(altitude: number) {
  return skyHorizonBase + altitude / 90 * skyAltitudeSpan
}

function altitudeGuidePath(altitude: number) {
  const edgeY = 100 - horizonAltitudePercent(altitude) + 1.5
  const crestY = edgeY - 7
  return `M 0 ${edgeY.toFixed(2)} Q 500 ${crestY.toFixed(2)} 1000 ${edgeY.toFixed(2)}`
}

function rotateSkyView(change: number) {
  skyViewAzimuth.value = normalizeAzimuth(skyViewAzimuth.value + change)
}

function turnSkyViewByWheel(event: WheelEvent) {
  const movement = Math.abs(event.deltaX) > Math.abs(event.deltaY) ? event.deltaX : event.deltaY
  rotateSkyView(movement * .08)
}

function beginSkyViewDrag(event: PointerEvent) {
  if (!activeCoordinates.value) return
  skyViewDragging.value = true
  skyViewStartX = event.clientX
  skyViewStartAzimuth = skyViewAzimuth.value
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
}

function dragSkyView(event: PointerEvent) {
  if (!skyViewDragging.value) return
  skyViewAzimuth.value = normalizeAzimuth(skyViewStartAzimuth - (event.clientX - skyViewStartX) * .28)
}

function endSkyViewDrag(event: PointerEvent) {
  if (!skyViewDragging.value) return
  skyViewDragging.value = false
  const field = event.currentTarget as HTMLElement
  if (field.hasPointerCapture(event.pointerId)) field.releasePointerCapture(event.pointerId)
}

function selectPage(page: SkyPage) {
  activePage.value = page
  window.history.pushState(null, '', `#astronomy-${page}`)
  document.querySelector('.sky-content-scroll')?.scrollTo({ top: 0, behavior: 'smooth' })
}

function selectBody(body: BodyId) {
  expandedBodyId.value = expandedBodyId.value === body ? null : body
}

function locateBody(body: BodyId) {
  expandedBodyId.value = body
  window.setTimeout(() => document.querySelector('.horizon-section')?.scrollIntoView({ behavior: 'smooth', block: 'start' }), 0)
}

function stopMinuteAnimation() {
  if (minuteAnimationFrame !== undefined) {
    cancelAnimationFrame(minuteAnimationFrame)
    minuteAnimationFrame = undefined
  }
}

function jumpToNow() {
  const current = new Date()
  const target = current.getHours() * 60 + current.getMinutes()
  const start = minuteOfDay.value
  stopMinuteAnimation()
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
    moonDay.value = await fetchMoonDay(currentLatitude, currentLongitude, elevation.value, timezoneName, Math.floor(now.value.getTime() / 1000))
  } catch {
    moonDay.value = null // 后端不可用时退回本地计算
  }
}

async function loadScore(currentLatitude: number, currentLongitude: number, at: number) {
  scoreController?.abort()
  const controller = new AbortController()
  scoreController = controller
  try {
    backendScore.value = await fetchObservingScore(currentLatitude, currentLongitude, at, controller.signal)
  } catch {
    if (!controller.signal.aborted) backendScore.value = null
  } finally {
    if (scoreController === controller) scoreController = undefined
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
    void loadScore(coordinates.latitude, coordinates.longitude, Math.floor(now.value.getTime() / 1000))
    void loadLightPollution(coordinates.latitude, coordinates.longitude)
  }
}, { immediate: true })
// 天气接口返回真实海拔后，用该海拔重取每日月相。
watch(elevation, () => {
  if (activeCoordinates.value) void loadMoonDay(activeCoordinates.value.latitude, activeCoordinates.value.longitude)
})
// 拖动时间条时按模拟时刻重取后端评分（防抖，避免连续滑动打满请求）。
watch(minuteOfDay, () => {
  window.clearTimeout(scoreTimer)
  scoreTimer = window.setTimeout(() => {
    if (activeCoordinates.value) {
      void loadScore(activeCoordinates.value.latitude, activeCoordinates.value.longitude, Math.floor(simulatedTime.value.getTime() / 1000))
    }
  }, 1200)
})
watch(moon, renderMoon)
watch(moonDay, renderMoon)
watch(moonCanvas, (canvas) => {
  if (canvas) renderMoon()
})

onMounted(() => {
  window.addEventListener('popstate', onPopState)
  clock = window.setInterval(() => { now.value = new Date() }, 30_000)
  loadMoonTexture()
  requestLocation()
})

onBeforeUnmount(() => {
  locationRevision += 1
  locationLookupController?.abort()
  conditionsController?.abort()
  scoreController?.abort()
  if (scoreTimer !== undefined) window.clearTimeout(scoreTimer)
  if (homeExitTimer !== undefined) window.clearTimeout(homeExitTimer)
  window.removeEventListener('popstate', onPopState)
  if (clock) window.clearInterval(clock)
  stopMinuteAnimation()
})
</script>

<template>
  <section class="sky-shell" :class="{ 'is-leaving': skyLeaving }" aria-label="AURORA 天文观测">
    <aside class="sky-sidebar">
      <AuroraBrand class="sky-brand" subtitle="SKY OBSERVATORY" variant="sky" :leaving="skyLeaving" @click="beginHomeExit" />
      <div class="sidebar-divider" aria-hidden="true" />

      <button class="sky-location" type="button" :aria-busy="locationStatus === 'locating' || locationStatus === 'resolving'" @click="requestLocation">
        <span class="location-mark" aria-hidden="true" />
        <span aria-live="polite"><strong>{{ locationLabel }}</strong><small>{{ coordinateLabel }}</small></span>
        <i>{{ locationAction }}</i>
      </button>

      <nav class="sky-menu" aria-label="天文观测页面">
        <button v-for="item in menu" :key="item.id" type="button" :class="{ active: activePage === item.id }" @click="selectPage(item.id)">
          <b>{{ item.numeral }}</b><span>{{ item.label }}<small>{{ item.en }}</small></span><i v-if="activePage === item.id">↗</i>
        </button>
      </nav>

      <div class="sidebar-source"><span />{{ activeCoordinates ? '本地星历计算 · 实时地点' : '需要地点以计算本地天空' }}</div>
      <time>{{ new Intl.DateTimeFormat('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false }).format(now) }} <small>LOCAL</small></time>
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
          <div><strong>{{ scorePanel == null ? '—' : String(scorePanel).padStart(2, '0') }}<small>/100</small></strong></div>
          <div><h2>{{ scoreVerdict }}</h2><p v-if="conditions">云量 {{ Math.round(conditions.current.cloudCover) }}%，能见度 {{ (conditions.current.visibilityMeters / 1000).toFixed(1) }} km；{{ conditionDescription(conditions.current.weatherCode) }} 是当前的主导条件。</p><p v-else-if="conditionsStatus === 'error'">天气源暂不可用；本地星历仍可计算天体位置与升落。</p><p v-else>正在读取云层、能见度、湿度与风的未来 24 小时预报。</p></div>
          <small>评分解释：云量、能见度、降水与月光<template v-if="lightPollution">与光污染（Bortle {{ lightPollution.bortle }} · SQM {{ lightPollution.sqm.toFixed(1) }}）</template><template v-else>；暂未将未经校准的光污染等级伪装为 Bortle/SQM</template>。</small>
          <div v-if="scoreFactors" class="score-factors" aria-label="评分构成">
            <span v-for="part in scoreFactors" :key="part.key" :class="part.kind"><i :style="{ width: factorBarWidth(part) }" /><small>{{ part.label }}</small><strong>{{ part.value > 0 ? `+${part.value.toFixed(1)}` : part.value.toFixed(1) }}</strong></span>
          </div>
        </section>

        <section class="tonight-advice" v-if="recommendation">
          <div class="advice-window"><p>今夜建议</p><h3>{{ recommendation.windowLabel }}</h3><span v-if="recommendation.window">{{ recommendation.window.verdict }}</span></div>
          <div class="advice-equipment" role="group" aria-label="观测器材">
            <button v-for="option in equipmentOptions" :key="option.id" type="button" :class="{ active: equipment === option.id }" :title="option.note" @click="equipment = option.id">{{ option.label }}</button>
          </div>
          <ul v-if="recommendation.targets.length">
            <li v-for="target in recommendation.targets" :key="target.id"><i :style="{ color: target.tint }">{{ target.glyph }}</i><span><strong>{{ target.name }}</strong><small>{{ target.summary }}</small><em v-if="target.tip">{{ target.tip }}</em></span></li>
          </ul>
          <p v-else>未来 24 小时天气与月光条件有限，建议短时观察亮目标，或改日再安排。</p>
        </section>

        <section class="instrument-grid" aria-label="当前观测条件">
          <article><span>总云量</span><strong>{{ conditions ? `${Math.round(conditions.current.cloudCover)}%` : '—' }}</strong><small>低 / 中 / 高云见下方预报</small></article>
          <article><span>能见度</span><strong>{{ conditions ? `${(conditions.current.visibilityMeters / 1000).toFixed(1)} km` : '—' }}</strong><small>影响暗星与银河辨识</small></article>
          <article><span>湿度</span><strong>{{ conditions ? `${Math.round(conditions.current.humidity)}%` : '—' }}</strong><small>高湿可能造成结露</small></article>
          <article><span>风速 / 阵风</span><strong>{{ conditions ? `${Math.round(conditions.current.windSpeed)} / ${Math.round(conditions.current.windGusts)} km/h` : '—' }}</strong><small>影响脚架稳定与体感</small></article>
          <article><span>降水</span><strong>{{ conditions ? `${conditions.current.precipitation.toFixed(1)} mm` : '—' }}</strong><small>当前预报时段</small></article>
          <article><span>PM₂.₅ / 气溶胶</span><strong>{{ currentAir ? `${Math.round(currentAir.pm25)} μg/m³` : '—' }}</strong><small>{{ currentAir ? `AOD ${currentAir.aerosolOpticalDepth.toFixed(2)} · 仅作透明度参考` : '空气质量源暂不可用' }}</small></article>
        </section>

        <section class="night-analysis" v-if="nightAnalysis" aria-label="今夜夜空分析">
          <div class="section-heading night-heading"><div><p>NIGHT ANALYSIS / 本地星历</p><h2>今夜夜空</h2></div><span>天文夜 · 无月黑夜 · 银河核心</span></div>
          <div class="night-grid">
            <article v-if="nightAnalysis.astronomicalNight"><span>天文夜</span><strong>{{ windowRange(nightAnalysis.astronomicalNight) }}</strong><small>太阳低于地平线 −18°，深空目标不受曙暮光干扰</small></article>
            <article><span>无月黑夜</span><strong>{{ nightAnalysis.moonlessWindows.length ? nightAnalysis.moonlessWindows.map((window) => windowRange(window)).join(' / ') : '今夜无' }}</strong><small>天文夜内月亮在地平线以下，暗弱目标辨识最佳</small></article>
            <article><span>银河核心</span><strong>{{ nightAnalysis.galacticCore.maxTime ? `最高 ${nightAnalysis.galacticCore.maxAltitude}°（${formatTime(nightAnalysis.galacticCore.maxTime)}）` : '今夜低于 20°' }}</strong><small>{{ nightAnalysis.galacticCore.windows.length ? `可见窗口：${nightAnalysis.galacticCore.windows.map((window) => windowRange(window)).join(' / ')}` : '银河核心需要更南方的观测点或更暗的天空' }}</small></article>
          </div>
        </section>

        <section class="forecast-section">
          <div class="forecast-heading"><p>天气预报</p><span>未来 24 小时 · {{ conditions?.source ?? '等待天气源' }} · {{ conditions?.timezone ?? '—' }}</span></div>
          <div v-if="hourlyForecast.length" class="forecast-matrix" role="table" aria-label="未来24小时观测天气预报">
            <div class="matrix-labels" aria-hidden="true">
              <span>日期</span><span>时间</span><span>天气</span><span>云量</span><span>高云</span><span>中云</span><span>低云</span><span>气温</span><span>露点</span><span>湿度</span><span>降水</span><span>风速</span><span>风向</span><span>能见度</span><span>气溶胶</span>
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
          <div v-else class="integration-state"><span>01</span><div><h3>{{ conditionsStatus === 'error' ? '天气预报暂不可用' : '正在连接天气预报' }}</h3><p>当前页保留小时级原始节奏：上海等地区不会被伪装成半小时气象预报。</p></div></div>
        </section>
      </section>

      <section v-else-if="activePage === 'sky'" class="sky-map-page page-stack">
        <div class="section-heading"><h2>星图</h2></div>
        <section class="horizon-section">
          <div class="horizon-field" :class="{ 'has-location': activeCoordinates, 'is-dragging': skyViewDragging }" :style="{ '--sky-daylight': String(daylight) }" @pointerdown="beginSkyViewDrag" @pointermove="dragSkyView" @pointerup="endSkyViewDrag" @pointercancel="endSkyViewDrag">
            <div class="sky-night" aria-hidden="true" />
            <div class="star-grain" aria-hidden="true" />
            <svg class="altitude-guides" viewBox="0 0 1000 100" preserveAspectRatio="none" aria-hidden="true"><path v-for="line in [30, 60]" :key="line" :d="altitudeGuidePath(line)" vector-effect="non-scaling-stroke" /></svg>
            <span v-for="line in [30, 60]" :key="`label-${line}`" class="altitude-label" :style="{ top: `calc(${100 - horizonAltitudePercent(line)}% - 9px)` }">{{ line }}°</span>
            <div v-for="body in horizonBodies" :key="body.id" class="sky-body" :style="horizonStyle(body)"><i>{{ body.glyph }}</i><span>{{ body.name }}</span></div>
            <div class="horizon-ridge horizon-ridge-far" aria-hidden="true" />
            <div class="horizon-ridge horizon-ridge-near" aria-hidden="true" />
            <div class="horizon-line" aria-hidden="true" />
            <div v-if="!activeCoordinates" class="sky-empty"><strong>允许定位后生成本地地平天空</strong><span>星历计算不需要 Key；它只需要你的经纬度与时刻。</span><button type="button" @click="requestLocation">请求位置</button></div>
            <span v-if="activeCoordinates" class="sky-visible-count"><small>当前视野</small>{{ horizonBodies.length }}<small>天体</small></span>
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
                <svg class="heading-arc" viewBox="0 0 1000 48" preserveAspectRatio="none" aria-hidden="true"><path d="M 0 45 C 205 24 346 3 500 3 C 654 3 795 24 1000 45" fill="none" stroke="currentColor" stroke-width="1.5" vector-effect="non-scaling-stroke" stroke-linecap="round" /></svg>
                <i v-for="tick in skyHeadingTicks" :key="tick.key" class="heading-tick" :class="{ major: tick.major, direction: tick.direction }" :style="{ left: `${tick.position}%`, bottom: `${tick.lift}px` }"><span v-if="tick.label">{{ tick.label }}</span></i>
              </div>
              <div class="heading-readout"><span>{{ skyViewDirection }}</span><strong>{{ skyViewAzimuth.toFixed(2) }}°</strong></div>
              <i class="heading-lubber" aria-hidden="true" />
            </div>
          </div>
        </section>
        <div class="time-scrubber"><div class="time-scrubber-inner"><div><span>时刻</span><strong>{{ timeLabel }}</strong></div><div class="time-scrubber-track"><output class="time-scrubber-bubble" :style="{ '--scrub-f': scrubFraction }">{{ timeLabel }}</output><input v-model.number="minuteOfDay" type="range" min="0" max="1439" step="5" aria-label="时刻" @pointerdown="stopMinuteAnimation" @keydown="stopMinuteAnimation" /></div><button class="time-scrubber-now" type="button" @click="jumpToNow">现在</button><div><span>00:00</span><span>06:00</span><span>12:00</span><span>18:00</span><span>24:00</span></div></div></div>

        <section class="window-section">
          <div class="section-heading"><h2>行星升落</h2></div>
          <div class="window-summary" v-if="twilight"><article><span>日出</span><strong>{{ formatTime(twilight.sunrise) }}</strong></article><article><span>日落</span><strong>{{ formatTime(twilight.sunset) }}</strong></article><article><span>天文晨光</span><strong>{{ formatTime(twilight.astronomicalDawn) }}</strong></article><article><span>天文昏影</span><strong>{{ formatTime(twilight.astronomicalDusk) }}</strong></article></div>
          <div class="time-axis"><div class="axis-labels"><span>00</span><span>06</span><span>12</span><span>18</span><span>24</span></div><article v-for="track in tracks" :key="track.id" class="time-track" :class="{ expanded: expandedBodyId === track.id }"><button :id="`track-trigger-${track.id}`" class="time-track-trigger" type="button" :aria-expanded="expandedBodyId === track.id" :aria-controls="`track-detail-${track.id}`" @click="selectBody(track.id)"><span class="time-track-label"><i :style="{ color: track.tint }">{{ track.glyph }}</i><span>{{ track.name }}<small>{{ observingStatus(track) }}</small></span><strong>{{ track.visible ? `${Math.round(track.altitude)}° ${bearing(track.azimuth)}` : '地平线下' }}</strong></span><span v-if="expandedBodyId !== track.id" class="track-rail" aria-hidden="true"><i v-for="segment in visibleSegments(track)" :key="`${segment.left}-${segment.width}`" :style="{ left: `${segment.left}%`, width: `${segment.width}%`, backgroundColor: track.tint }" /><i class="rail-current" :class="{ 'is-below': !(railCurrentMarkers.get(track.id)?.aboveHorizon ?? true) }" :style="{ left: `${railCurrentMarkers.get(track.id)?.left ?? 0}%`, '--chart-tint': track.tint }" /></span><svg v-else class="track-wave" viewBox="0 0 960 88" aria-hidden="true" :style="{ '--chart-tint': track.tint }"><line class="altitude-horizon" x1="0" y1="52" x2="960" y2="52" /><path v-for="(path, index) in expandedWave?.belowPaths ?? []" :key="`below-${index}`" class="altitude-below" :d="path" /><path v-for="(path, index) in expandedWave?.abovePaths ?? []" :key="`above-${index}`" class="altitude-above" :d="path" /><circle v-if="expandedWave" class="altitude-current" :class="{ 'is-below': !expandedWave.current.aboveHorizon }" :cx="expandedWave.current.x" :cy="expandedWave.current.y" r="5" /></svg></button><div v-if="expandedBodyId === track.id" :id="`track-detail-${track.id}`" class="track-detail" role="region" :aria-labelledby="`track-trigger-${track.id}`"><div class="track-detail-state"><span :style="{ backgroundColor: track.tint }" /><div><p>当前位置</p><h3>{{ observingStatus(track) }} · {{ Math.round(track.altitude) }}° 高度角</h3><small>方位角 {{ Math.round(track.azimuth) }}° · {{ bearing(track.azimuth) }}方</small></div></div><dl><div><dt>升起</dt><dd>{{ formatTime(track.rise) }}</dd></div><div><dt>中天</dt><dd>{{ formatTime(track.transit) }}</dd></div><div><dt>落下</dt><dd>{{ formatTime(track.set) }}</dd></div><div><dt>最佳高度</dt><dd>{{ formatTime(track.best) }}</dd></div><div v-if="track.id === 'moon'"><dt>月面亮度</dt><dd>{{ Math.round((track.illumination ?? 0) * 100) }}%</dd></div><div v-else-if="track.id !== 'sun'"><dt>视星等</dt><dd>{{ track.magnitude?.toFixed(1) ?? '—' }}</dd></div><div><dt>{{ track.id === 'moon' ? '地月距离' : '地心距离' }}</dt><dd>{{ formatDistance(track.distanceAu) }}</dd></div></dl><p class="track-caveat" v-if="track.id === 'sun'">太阳观测必须使用合格的全口径太阳滤镜；绝不可用裸眼、墨镜或未加滤镜的器材直视太阳。</p><p class="track-caveat" v-else-if="track.id === 'moon'">月面适合在明暗交界附近观察；满月虽明亮，地形阴影反而较少。</p><p class="track-caveat" v-else>实际可见性还取决于云层、曙暮光、地平线遮挡与本地光污染。</p><div class="track-detail-actions"><button type="button" @click="locateBody(track.id)">在星图定位</button></div></div></article></div>
        </section>

      </section>

      <section v-else class="events-page page-stack">
        <section class="events-lead"><p>EPHEMERIS-BASED / 本地星历</p><h2>未来一个月的月相节奏</h2><span>以下时间由浏览器本地星历计算，不依赖外部 Key；地点会在后续版本参与可见性判定。</span></section>
        <div class="event-list">
          <article v-for="event in moonEvents" :key="event.target"><time><strong>{{ String(event.at.getDate()).padStart(2, '0') }}</strong><span>{{ new Intl.DateTimeFormat('en', { month: 'short' }).format(event.at).toUpperCase() }}</span></time><div><p>月相</p><h3>{{ event.label }}</h3><span>{{ event.description }}</span></div><strong>{{ formatTime(event.at) }}</strong></article>
        </div>
        <section class="curated-events"><div class="section-heading"><div><p>CURATED EVENTS / 需要编辑来源</p><h2>大型天象将在这里出现</h2></div><span>流星雨 · 合月 · 冲日 · 日月食</span></div><div class="integration-state"><span>03</span><div><h3>等待可信事件源与人工校核</h3><p>流星雨峰值、掩星与“本地是否可见”不应由一段静态文案冒充。这里将接入可追溯的天文机构数据，必要时保存为可校订条目。</p></div></div></section>
        <section class="image-feed"><div class="section-heading"><div><p>IMAGE FEED / 图像卡片</p><h2>每日抬头以外的宇宙</h2></div><span>后续接入开放授权内容</span></div><div class="image-placeholder-grid"><article><i>NASA</i><h3>每日天文图片</h3><p>需要 NASA API Key，接入后显示来源、作者和版权说明。</p></article><article><i>ESO</i><h3>欧洲南方天文台</h3><p>仅挑选可明确展示授权与署名的高质量图像。</p></article><article><i>JWST</i><h3>深空档案联动</h3><p>把图像连接到 AURORA 内部的太阳、行星与任务档案。</p></article></div></section>
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
.sky-sidebar { position:sticky; top:0; display:flex; flex-direction:column; min-height:100dvh; padding:30px 20px 18px; border-right:1px solid var(--sky-line); background:linear-gradient(180deg,var(--sky-panel) 0%,var(--sky-deep) 100%); }
.sky-brand { margin-left:5px; color:var(--sky-ink); } /* 图标轨道环向左探出约 5px，右移品牌使图标最左端与下方分隔线左端对齐 */
.sky-location { display:grid; grid-template-columns:22px 1fr auto; gap:10px; align-items:center; width:100%; margin:0 0 28px; padding:0; color:inherit; text-align:left; background:none; border:0; cursor:pointer; }
.sidebar-divider { margin:24px 0 16px; border-top:1px solid var(--sky-line); }
.location-mark { width:14px; height:14px; border:1px solid var(--sky-amber); border-radius:50% 50% 50% 0; transform:rotate(-45deg); }
.location-mark::after { content:""; display:block; width:4px; height:4px; margin:4px; border-radius:50%; background:var(--sky-amber); }
.sky-location strong,.sky-location small { display:block; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.sky-location strong { font-size:11px; font-weight:600; }
.sky-location small { margin-top:4px; color:var(--sky-muted); font:9px var(--font-mono,monospace); }
.sky-location i { color:var(--sky-cyan); font:8px var(--font-mono,monospace); font-style:normal; letter-spacing:.08em; }
.sky-menu { border-top:1px solid var(--sky-line); }
.sky-menu button { position:relative; display:grid; grid-template-columns:25px 1fr auto; align-items:center; width:calc(100% + 40px); min-height:60px; margin-left:-20px; padding:0 20px 0 28px; color:var(--sky-muted); text-align:left; background:none; border:0; border-bottom:1px solid var(--sky-line); cursor:pointer; transition:background .2s,color .2s; }
.sky-menu button::before { position:absolute; top:0; bottom:0; left:0; width:2px; background:var(--sky-amber); content:""; transform:scaleY(0); transform-origin:center; transition:transform .2s; }
.sky-menu button:hover,.sky-menu button.active { color:var(--sky-ink); background:rgba(160,182,216,.045); }
.sky-menu button.active::before { transform:scaleY(1); }
.sky-menu b { font:9px var(--font-mono,monospace); color:var(--sky-cyan); }
.sky-menu span { font-size:12px; font-weight:600; }
.sky-menu small { display:block; margin-top:3px; color:var(--sky-muted); font:8px var(--font-mono,monospace); letter-spacing:.13em; }
.sky-menu i { color:var(--sky-amber); font-style:normal; }
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
.condition-verdict h2 { margin:2px 0 8px; font-size:24px; font-weight:500; }
.condition-verdict div:nth-child(2) p { color:var(--sky-muted); font-family:inherit; letter-spacing:0; line-height:1.6; }
.condition-verdict > small { grid-column:2; color:var(--sky-muted); font-size:10px; line-height:1.5; }
.score-factors { grid-column:1 / -1; display:grid; grid-template-columns:repeat(auto-fit,minmax(108px,1fr)); gap:8px; margin-top:16px; }
.score-factors span { position:relative; display:grid; grid-template-columns:1fr auto; gap:2px 8px; align-items:end; padding:8px 10px; overflow:hidden; border:1px solid var(--sky-line); border-radius:8px; background:var(--sky-sunken); }
.score-factors span > i { position:absolute; inset:auto 0 0; height:2px; background:var(--sky-muted); opacity:.4; }
.score-factors span.bonus > i { background:var(--sky-amber); opacity:.65; }
.score-factors small { grid-column:1; color:var(--sky-muted); font-size:9px; }
.score-factors strong { grid-column:2; font:11px var(--font-mono,monospace); }
.score-factors span.bonus strong { color:var(--sky-amber); }
.tonight-advice { display:grid; grid-template-columns:auto 1fr; gap:26px; align-items:center; margin-top:18px; padding:20px 28px; border:1px solid var(--sky-line); background:rgba(172,193,226,.03); }
.tonight-advice p:first-child { margin:0; color:var(--sky-muted); font-size:10px; }
.tonight-advice h3 { margin:5px 0 3px; font-size:20px; font-weight:500; }
.tonight-advice div > span { color:var(--sky-muted); font-size:11px; }
.advice-equipment { display:flex; gap:6px; align-self:end; }
.advice-equipment button { padding:6px 12px; color:var(--sky-muted); font-size:10px; border:1px solid var(--sky-line); border-radius:999px; background:transparent; cursor:pointer; }
.advice-equipment button:hover { border-color:rgba(165,188,222,.4); }
.advice-equipment button.active { color:var(--sky-amber); border-color:rgba(234,196,120,.5); background:rgba(234,196,120,.08); }
.tonight-advice ul { grid-column:1 / -1; display:flex; gap:22px; margin:0; padding:0; list-style:none; }
.tonight-advice li { display:flex; gap:10px; align-items:flex-start; }
.tonight-advice li i { font-style:normal; font-size:22px; line-height:1; }
.tonight-advice li strong { display:block; font-size:14px; font-weight:500; }
.tonight-advice li small { display:block; margin-top:2px; color:var(--sky-muted); font-size:10px; }
.tonight-advice li em { display:block; margin-top:4px; max-width:34ch; color:var(--sky-muted); font-size:9px; font-style:normal; line-height:1.5; }
.tonight-advice > p { grid-column:1 / -1; margin:0; color:var(--sky-muted); font-size:12px; line-height:1.6; }
.instrument-grid { display:grid; grid-template-columns:repeat(3,1fr); border:1px solid var(--sky-line); border-bottom:0; }
.instrument-grid article { min-height:128px; padding:19px; border-right:1px solid var(--sky-line); border-bottom:1px solid var(--sky-line); }
.instrument-grid article:nth-child(3n) { border-right:0; }
.instrument-grid span,.instrument-grid small { display:block; color:var(--sky-muted); font-size:10px; }
.instrument-grid strong { display:block; margin:18px 0 9px; font:22px var(--font-mono,monospace); }
.instrument-grid small { line-height:1.4; }
.forecast-section,.window-section,.curated-events,.image-feed { margin-top:54px; }
.night-analysis { margin-top:54px; }
.night-grid { display:grid; grid-template-columns:repeat(3,1fr); border:1px solid var(--sky-line); border-radius:18px; overflow:hidden; background:var(--sky-sunken); }
.night-grid article { min-height:118px; padding:18px 20px; border-right:1px solid var(--sky-line); }
.night-grid article:last-child { border-right:0; }
.night-grid span { display:block; color:var(--sky-amber); font:9px var(--font-mono,monospace); letter-spacing:.1em; }
.night-grid strong { display:block; margin:10px 0 8px; font:15px var(--font-mono,monospace); line-height:1.45; }
.night-grid small { display:block; color:var(--sky-muted); font-size:10px; line-height:1.55; }
.section-heading { display:flex; align-items:end; justify-content:space-between; gap:18px; margin-bottom:18px; }
.section-heading h2 { margin:6px 0 0; font-size:25px; font-weight:500; letter-spacing:-.04em; }
.section-heading > span { color:var(--sky-muted); font-size:10px; text-align:right; }
.forecast-heading { display:flex; justify-content:space-between; align-items:baseline; gap:16px; margin-bottom:16px; }
.forecast-heading p { margin:0; color:var(--sky-amber); font:9px var(--font-mono,monospace); letter-spacing:.12em; }
.forecast-heading span { color:var(--sky-muted); font-size:10px; }
.forecast-matrix { display:grid; grid-template-columns:102px minmax(0,1fr); overflow:hidden; border:1px solid var(--sky-line); background:var(--sky-sunken); }
.matrix-labels { display:grid; grid-template-rows:34px 38px 58px repeat(12,44px); background:var(--sky-panel); border-right:1px solid var(--sky-line); }
.matrix-labels span { display:grid; place-items:center start; padding-left:14px; color:var(--sky-muted); font-size:10px; border-bottom:1px solid rgba(165,188,222,.1); }
.matrix-labels span:nth-child(1),.matrix-labels span:nth-child(2) { color:var(--sky-cyan); font:9px var(--font-mono,monospace); }
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
.horizon-section { border:1px solid var(--sky-line); background:var(--sky-panel); border-radius:18px; overflow:hidden; }
.horizon-field { position:relative; min-height:440px; overflow:hidden; background:linear-gradient(180deg,#2c5074 0%,#4c7295 55%,#9db2c3 100%); }
.horizon-field::before { position:absolute; inset:0; background:radial-gradient(circle at 18% 13%,rgba(172,193,226,.2) 0 1px,transparent 1.5px),radial-gradient(circle at 76% 24%,rgba(172,193,226,.14) 0 1px,transparent 1.5px),radial-gradient(circle at 61% 9%,rgba(172,193,226,.2) 0 1px,transparent 1.5px); content:""; opacity:calc(.7 * (1 - var(--sky-daylight,0))); }
.star-grain { position:absolute; inset:0; opacity:calc(.24 * (1 - var(--sky-daylight,0))); background-image:radial-gradient(#c9d8ee 1px,transparent 1px); background-size:79px 83px; }
.sky-night { position:absolute; inset:0; background:linear-gradient(180deg,#081322 0%,#0e2034 66%,#0a1322 100%); opacity:calc(1 - var(--sky-daylight,0)); pointer-events:none; transition:opacity .45s ease; }
.altitude-guides { position:absolute; inset:0; width:100%; height:100%; pointer-events:none; }
.altitude-guides path { fill:none; stroke:rgba(165,188,222,.18); stroke-width:1; stroke-dasharray:3 4; }
.altitude-label { position:absolute; z-index:1; left:8px; color:var(--sky-muted); font:8px var(--font-mono,monospace); pointer-events:none; }
.horizon-ridge { position:absolute; z-index:0; right:-3%; left:-3%; pointer-events:none; }
.horizon-ridge-far { bottom:3px; height:52px; background:linear-gradient(180deg,rgba(53,82,112,.72),rgba(18,34,52,.92)); clip-path:polygon(0 86%,8% 70%,17% 78%,29% 45%,38% 68%,47% 52%,57% 76%,68% 54%,78% 74%,89% 48%,100% 72%,100% 100%,0 100%); opacity:.48; }
.horizon-ridge-near { bottom:0; height:39px; background:linear-gradient(180deg,#17283b 0%,#0a1422 100%); clip-path:polygon(0 82%,11% 58%,21% 76%,33% 51%,43% 82%,56% 63%,66% 79%,79% 54%,90% 74%,100% 62%,100% 100%,0 100%); opacity:.62; }
.horizon-line { position:absolute; z-index:2; right:0; bottom:58px; left:0; height:1px; background:linear-gradient(90deg,transparent,rgba(174,196,226,.24) 7%,rgba(174,196,226,.34) 50%,rgba(174,196,226,.24) 93%,transparent); pointer-events:none; }
.compass { position:absolute; right:28px; bottom:14px; left:28px; display:flex; justify-content:space-between; color:rgba(165,188,222,.48); font:9px var(--font-mono,monospace); }
.sky-body { position:absolute; z-index:3; display:grid; justify-items:center; transform:translateX(-50%); animation:body-arrive .5s cubic-bezier(.22,1,.36,1); transition:opacity .45s ease; }
.sky-body i { display:grid; width:25px; height:25px; place-items:center; border-radius:50%; color:#0d1828; background:var(--body-tint); box-shadow:0 0 18px color-mix(in srgb,var(--body-tint) 45%,transparent); font-size:16px; font-style:normal; }
.sky-body span { margin-top:5px; padding:2px 5px; color:var(--sky-ink); background:rgba(11,19,34,.8); font-size:9px; white-space:nowrap; }
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
.heading-arc { position:absolute; inset:0; width:100%; height:100%; color:color-mix(in srgb,#0b1322 calc(var(--sky-daylight,0) * 100%),rgba(165,188,222,.32)); pointer-events:none; }
.heading-tick { position:absolute; width:1px; height:5px; background:color-mix(in srgb,rgba(10,20,36,.6) calc(var(--sky-daylight,0) * 100%),rgba(165,188,222,.32)); transform:translateX(-50%); }
.heading-tick.major { height:10px; background:color-mix(in srgb,rgba(10,20,36,.85) calc(var(--sky-daylight,0) * 100%),rgba(165,188,222,.68)); }
.heading-tick span { position:absolute; z-index:7; top:calc(100% + 3px); left:50%; padding:0 2px; color:color-mix(in srgb,#0a1422 calc(var(--sky-daylight,0) * 100%),rgba(184,202,227,.72)); font:7px var(--font-mono,monospace); white-space:nowrap; background:color-mix(in srgb,rgba(244,248,252,.9) calc(var(--sky-daylight,0) * 100%),rgba(7,17,30,.52)); border-radius:2px; text-shadow:color-mix(in srgb,rgba(255,255,255,.55) calc(var(--sky-daylight,0) * 100%),rgba(0,0,0,.9)); transform:translateX(-50%); }
.heading-tick.direction span { color:var(--sky-amber); font-size:9px; font-weight:600; letter-spacing:.08em; background:color-mix(in srgb,rgba(244,248,252,.95) calc(var(--sky-daylight,0) * 100%),rgba(7,17,30,.86)); }
.heading-readout { position:absolute; top:11px; left:50%; z-index:5; display:flex; gap:5px; align-items:baseline; justify-content:center; color:color-mix(in srgb,#0a1422 calc(var(--sky-daylight,0) * 100%),var(--sky-ink)); white-space:nowrap; text-shadow:color-mix(in srgb,rgba(255,255,255,.5) calc(var(--sky-daylight,0) * 100%),rgba(0,0,0,.72)); transform:translateX(-50%); }
.heading-readout span { color:var(--sky-amber); font-size:9px; font-weight:600; }
.heading-readout strong { font:12px var(--font-mono,monospace); font-weight:500; }
.heading-lubber { position:absolute; top:-1px; left:50%; z-index:6; width:8px; height:7px; background:var(--sky-amber); clip-path:polygon(0 0,100% 0,50% 100%); transform:translateX(-50%); }
.heading-dial:hover .heading-arc,.horizon-field.is-dragging .heading-arc { color:color-mix(in srgb,rgba(10,20,36,.9) calc(var(--sky-daylight,0) * 100%),rgba(165,188,222,.42)); }

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
.track-detail-actions button:hover { background:color-mix(in srgb,var(--sky-cyan) 8%,transparent); }
.track-detail-actions button:focus-visible { outline:1px solid var(--sky-cyan); outline-offset:2px; }

/* 展开详情：高度角-时间曲线（从地平线下穿出、拱起、再回到地平线下） */
.altitude-horizon { stroke:rgba(226,230,232,.3); stroke-width:1; }
.altitude-below { fill:none; stroke:rgba(226,230,232,.2); stroke-width:1.5; stroke-dasharray:3 4; }
.altitude-above { fill:none; stroke:var(--chart-tint); stroke-width:2.6; stroke-linecap:round; stroke-linejoin:round; }
.altitude-current { fill:var(--chart-tint); stroke:var(--sky-deep); stroke-width:2; filter:drop-shadow(0 2px 4px rgba(0,0,0,.42)); }
.altitude-current.is-below { fill:var(--sky-muted); opacity:.76; }

/* ---------- 天象与图片 ---------- */
.events-lead { padding:34px; border-top:2px solid var(--sky-amber); background:linear-gradient(90deg,rgba(172,193,226,.04),transparent); }
.events-lead h2 { margin:8px 0; font-size:34px; font-weight:500; }
.events-lead span { color:var(--sky-muted); font-size:12px; line-height:1.6; }
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
.image-placeholder-grid { display:grid; grid-template-columns:repeat(3,1fr); gap:12px; }
.image-placeholder-grid article { min-height:190px; padding:18px; border:1px solid var(--sky-line); background:linear-gradient(145deg,rgba(172,193,226,.07),rgba(11,19,34,.32)); }
.image-placeholder-grid article:nth-child(2) { background:linear-gradient(145deg,rgba(172,193,226,.1),rgba(11,19,34,.32)); }
.image-placeholder-grid article:nth-child(3) { background:linear-gradient(145deg,rgba(172,193,226,.055),rgba(11,19,34,.32)); }
.image-placeholder-grid i { color:var(--sky-amber); font:10px var(--font-mono,monospace); font-style:normal; letter-spacing:.12em; }
.image-placeholder-grid h3 { margin:50px 0 8px; font-size:16px; font-weight:500; }
.image-placeholder-grid p { margin:0; color:var(--sky-muted); font-size:11px; line-height:1.55; }

/* ---------- 动画与响应式 ---------- */
@keyframes body-arrive { from { opacity:0; transform:translate(-50%,6px); } to { opacity:1; transform:translate(-50%,0); } }
@keyframes track-detail-reveal { from { opacity:.2; clip-path:inset(0 0 100% 0); } to { opacity:1; clip-path:inset(0 0 0 0); } }
@keyframes wave-reveal { from { opacity:.35; clip-path:inset(0 100% 0 0); } to { opacity:1; clip-path:inset(0 0 0 0); } }
@media (max-width:900px) {
  .sky-shell { grid-template-columns:1fr; }
  .sky-sidebar { position:relative; min-height:auto; padding:20px; }
  .sky-location { margin:22px 0; }
  .sky-menu { display:grid; grid-template-columns:repeat(3,1fr); }
  .sky-menu button { width:100%; margin:0; padding:9px; min-height:58px; grid-template-columns:20px 1fr; }
  .sky-menu i { display:none; }
  .sidebar-source,.sky-sidebar > time { display:none; }
  .sky-content-scroll { height:auto; overflow:visible; }
  .instrument-grid { grid-template-columns:repeat(2,1fr); }
  .instrument-grid article:nth-child(3n) { border-right:1px solid var(--sky-line); }
  .instrument-grid article:nth-child(2n) { border-right:0; }
  .axis-labels { padding-left:174px; }
  .track-detail { padding-left:18px; }
  .track-detail dl { grid-template-columns:repeat(3,minmax(0,1fr)); }
  .track-caveat { grid-column:1 / -1; }
  .track-detail-actions { grid-column:1 / -1; justify-content:start; }
}
@media (max-width:620px) {
  .page-stack { padding:20px 20px 55px; }
  .condition-hero { grid-template-columns:80px 1fr; gap:16px; padding:18px; }
  .moon-disc { width:72px; height:72px; }
  .moon-copy h2 { font-size:25px; }
  .moon-rise { grid-column:1 / -1; min-width:0; padding:14px 0 0; border-top:1px solid var(--sky-line); border-left:0; }
  .condition-verdict { grid-template-columns:1fr; }
  .condition-verdict > small { grid-column:1; }
  .tonight-advice { grid-template-columns:1fr; gap:14px; }
  .tonight-advice ul { flex-direction:column; gap:14px; }
  .instrument-grid { grid-template-columns:1fr; }
  .instrument-grid article,.instrument-grid article:nth-child(3n) { border-right:0; }
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
  .image-placeholder-grid { grid-template-columns:1fr; }
  .horizon-field { min-height:380px; }
  .time-scrubber-inner { grid-template-columns:1fr; row-gap:12px; }
  .time-scrubber-now { margin-left:0; }
  .time-scrubber-inner > div:last-child { grid-column:1; }
  .heading-dial { right:20px; left:20px; }
  .heading-readout { top:11px; gap:4px; }
}
@media (prefers-reduced-motion:reduce) { .sky-menu button,.sky-body,.track-wave,.track-detail,.sky-night { transition:none; animation:none; } }
</style>
