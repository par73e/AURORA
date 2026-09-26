export type CuratedEventKind = 'meteor' | 'eclipse' | 'planet'

export interface CuratedSkyEvent {
  id: string
  kind: CuratedEventKind
  title: string
  titleEn: string
  startsAt: string
  dateLabel: string
  summary: string
  localStatus: string
  localStatusKind: 'ready' | 'caution' | 'pending'
  observingWindow: string
  direction: string
  moonlight: string
  advice: string
  sourceName: string
  sourceUrl: string
  verifiedAt: string
  focusMinute?: number
  focusAzimuth?: number
}

/**
 * 年度人工校订天象：不是第三方实时接口，也不把全球事件误写成本地可见。
 *
 * 维护约定：每年更新一次；新条目必须带权威来源、核验日期和本地可见性边界。
 */
export const CURATED_SKY_EVENTS_2026: CuratedSkyEvent[] = [
  // 多行星同场只有通过本地可见性校验后，才作为普通的 planet 条目加入这里。
  {
    id: 'perseids-2026',
    kind: 'meteor',
    title: '英仙座流星雨极大',
    titleEn: 'PERSEIDS',
    startsAt: '2026-08-12T16:00:00Z',
    dateLabel: '8月12–13日夜间',
    summary: '北半球夏季最稳定、最适合入门守候的流星雨之一。',
    localStatus: '北半球优先 · 需避开城市灯光',
    localStatusKind: 'ready',
    observingWindow: '当地午夜后至黎明前',
    direction: '东北方；辐射点升高后更有利',
    moonlight: '峰值夜月面亮度约 1%，月光干扰很低',
    advice: '不必直盯辐射点；选择开阔天空，给眼睛至少 20 分钟适应黑暗。天气只在活动前 24 小时再复核。',
    sourceName: 'International Meteor Organization · 2026 Meteor Shower Calendar',
    sourceUrl: 'https://imo.net/files/meteor-shower/cal2026.pdf',
    verifiedAt: '2026-08-11',
    focusMinute: 90,
    focusAzimuth: 45,
  },
  {
    id: 'total-solar-eclipse-2026',
    kind: 'eclipse',
    title: '日全食',
    titleEn: 'TOTAL SOLAR ECLIPSE',
    startsAt: '2026-08-12T17:13:00Z',
    dateLabel: '2026年8月12日',
    summary: '日全食仅在特定地区可见，具体时刻随观测地点变化。',
    localStatus: '全球发生 · 当地可见性待按地点判定',
    localStatusKind: 'pending',
    observingWindow: '请以 NASA 路径图与当地接触时刻为准',
    direction: '不同地点差异很大',
    moonlight: '不适用',
    advice: '仅在路径覆盖区才可见；任何阶段都必须使用合格的太阳观测滤镜，不能裸眼或以普通墨镜替代。',
    sourceName: 'NASA/GSFC Solar Eclipse Catalog',
    sourceUrl: 'https://eclipse.gsfc.nasa.gov/SEpath/SEpath.html',
    verifiedAt: '2026-08-11',
  },
  {
    id: 'venus-greatest-eastern-elongation-2026',
    kind: 'planet',
    title: '金星东大距',
    titleEn: 'VENUS EAST ELONGATION',
    startsAt: '2026-08-15T05:59:00Z',
    dateLabel: '8月15日',
    summary: '金星与太阳的角距达到近期最大，是安排傍晚低空观测的好时机。',
    localStatus: '傍晚低空 · 需有开阔西方地平线',
    localStatusKind: 'caution',
    observingWindow: '日落后不久；具体高度随地点和日期变化',
    direction: '西方至西南方低空',
    moonlight: '通常不是主要限制因素',
    advice: '不要在太阳仍在地平线上方时寻找金星。等待日落后再观察，并避开建筑物与山体遮挡。',
    sourceName: 'NASA/GSFC Sky Events Calendar',
    sourceUrl: 'https://eclipse.gsfc.nasa.gov/SKYCAL/SKYCAL.html?cal=2026',
    verifiedAt: '2026-08-11',
  },
  {
    id: 'kappa-cygnids-2026',
    kind: 'meteor',
    title: '天鹅座 κ 流星雨极大',
    titleEn: 'KAPPA CYGNIDS',
    startsAt: '2026-08-17T00:00:00Z',
    dateLabel: '8月17日',
    summary: '北半球可整夜守候的小型流星雨，但 2026 年不预期有明显爆发。',
    localStatus: '北半球可见 · 强度低',
    localStatusKind: 'caution',
    observingWindow: '入夜后至黎明前',
    direction: '北方天空；辐射点位于天鹅座一带',
    moonlight: '请结合当日月相判断',
    advice: '适合作为夏夜附加目标，不建议为它单独远行。记录明亮火流星仍有观测价值。',
    sourceName: 'International Meteor Organization · 2026 Meteor Shower Calendar',
    sourceUrl: 'https://imo.net/files/meteor-shower/cal2026.pdf',
    verifiedAt: '2026-08-11',
  },
  {
    id: 'partial-lunar-eclipse-2026',
    kind: 'eclipse',
    title: '月偏食',
    titleEn: 'PARTIAL LUNAR ECLIPSE',
    startsAt: '2026-08-28T04:14:00Z',
    dateLabel: '8月28日',
    summary: '月球部分进入地球本影的全球性天象；不同地区可见程度与月亮高度不同。',
    localStatus: '部分地区可见 · 须按地点核对',
    localStatusKind: 'pending',
    observingWindow: '最大食分约在 04:14 UTC；请查阅当地可见时间',
    direction: '以当地月球位置为准',
    moonlight: '不适用',
    advice: '月食可裸眼安全观看；但月亮在地平线下或白昼时，当地不会有可观测画面。',
    sourceName: 'NASA/GSFC Lunar Eclipse Catalog',
    sourceUrl: 'https://eclipse.gsfc.nasa.gov/LEdecade/LEdecade2021.html',
    verifiedAt: '2026-08-11',
  },
  {
    id: 'aurigids-2026',
    kind: 'meteor',
    title: '御夫座流星雨极大',
    titleEn: 'AURIGIDS',
    startsAt: '2026-09-01T00:00:00Z',
    dateLabel: '9月1日',
    summary: '规模较小、峰值短促的北半球流星雨，适合作为早秋的附加观察目标。',
    localStatus: '北半球优先 · 常规强度较低',
    localStatusKind: 'caution',
    observingWindow: '后半夜至黎明前',
    direction: '东北方至北方天空',
    moonlight: '请结合当日月相判断',
    advice: '优先选择避开直射灯光的开阔场地；云量较高时可改日观测。',
    sourceName: 'International Meteor Organization · 2026 Meteor Shower Calendar',
    sourceUrl: 'https://imo.net/files/meteor-shower/cal2026.pdf',
    verifiedAt: '2026-08-11',
  },
  {
    id: 'september-epsilon-perseids-2026',
    kind: 'meteor',
    title: '九月 ε 英仙座流星雨极大',
    titleEn: 'SEPTEMBER EPSILON PERSEIDS',
    startsAt: '2026-09-09T18:00:00Z',
    dateLabel: '9月9–10日',
    summary: '可能出现明亮流星的小型流星雨；2026 年增强活动仍存在不确定性。',
    localStatus: '北半球可见 · 活动强度待观测确认',
    localStatusKind: 'pending',
    observingWindow: '峰值约 9月9日 18:00 UTC 后的当地夜间',
    direction: '东北方天空',
    moonlight: '请结合当日月相判断',
    advice: '活动强度尚不确定；若流星明显增多，可记录时间、方向与亮度。',
    sourceName: 'International Meteor Organization · 2026 Meteor Shower Calendar',
    sourceUrl: 'https://imo.net/files/meteor-shower/cal2026.pdf',
    verifiedAt: '2026-08-11',
  },
]
