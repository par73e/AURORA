/**
 * 月球轨道飞行器数据（示意展示用，非实时星历）：
 * 数据来源与地球页一致原则——公开可溯源的官方/权威轨道数据，
 * 但环月飞行器不在 CelesTrak GP 目录中，此处使用公开报道的标称轨道参数，
 * 页面读数区会标注来源。
 */
export interface MoonCraftSpec {
  id: string
  name: string
  nameEn: string
  note: string
  source: string
  /** 类型标识（面板 kicker，如 NASA LUNAR ORBITER） */
  type: string
  /** 运营方/机构（与地球面板统一字段） */
  operator: string
  /** 发射信息（日期 · 发射场 · 运载） */
  launch: string
  /** 运行状态 */
  status: string
  /** 任务说明（详尽） */
  mission: string
  /** 轨道描述 */
  orbit: string
  /** 轨道倾角（度，展示用） */
  inclination: number
  /** 偏心率（展示用） */
  eccentricity: string
  /** 轨道周期（展示用） */
  period: string
  /** orbital = 绕月轨道；stationary = 定点（如地月拉格朗日点，示意） */
  kind: 'orbital' | 'stationary'
  /** stationary 时的固定位置（月球中心偏移，场景单位） */
  stationaryOffset?: [number, number, number]
  /** 轨道半长轴（场景单位，月球半径 2.6 的等比例缩放） */
  a: number
  /** 离心率 */
  e: number
  /** 轨道倾角（度） */
  inclinationDeg: number
  /** 升交点经度（度，视觉排布用） */
  raanDeg: number
  /** 近心点幅角（度） */
  argPeriapsisDeg: number
  /** 视觉公转周期（秒） */
  periodSeconds: number
}

export const MOON_CRAFTS: MoonCraftSpec[] = [
  {
    id: 'lro',
    name: '月球勘测轨道飞行器',
    kind: 'orbital',
    type: 'NASA LUNAR ORBITER',
    operator: 'NASA（美国国家航空航天局）',
    launch: '2009-06-18 · 卡纳维拉尔角 · Atlas V 401',
    status: '运行中（2009 年至今）',
    mission: '对月球表面进行高精度测绘（LOLA 激光测高、LROC 影像），探测极区水冰与辐射环境；2009 年与 LCROSS 协同完成月背撞击探测，为后续载人登月选址提供数据。',
    orbit: '近极地圆轨道 · 高度约 50 km',
    inclination: 90,
    eccentricity: '≈ 0.001',
    period: '约 113 分钟',
    nameEn: 'LRO',
    note: '环月近极地圆轨道：高度约 50 km，周期约 113 分钟，2009 年至今持续测绘月球表面',
    source: 'NASA LRO 公开任务数据',
    a: 2.675, // 2.6 + 50km/1737km*2.6
    e: 0,
    inclinationDeg: 90,
    raanDeg: 40,
    argPeriapsisDeg: 0,
    periodSeconds: 300, // 视觉公转周期：调慢到 5 分钟一圈
  },
  {
    id: 'queqiao2',
    name: '鹊桥二号中继星',
    nameEn: 'QUEQIAO-2',
    kind: 'stationary',
    type: 'CNSA RELAY SATELLITE',
    operator: 'CNSA（中国国家航天局）',
    launch: '2024-03-20 · 文昌航天发射场 · 长征八号',
    status: '运行中',
    mission: '运行于地月拉格朗日 L2 点附近的晕轨道，为嫦娥六号等月背采样任务提供地月中继通信，并携带极紫外相机等科学载荷；本页示意为定点。',
    orbit: '大椭圆冻结轨道 · 近月点约 300 km / 远月点约 8,600 km',
    inclination: 57,
    eccentricity: '≈ 0.93',
    period: '约 24 小时',
    note: '运行于地月拉格朗日 L2 点附近的晕轨道，为嫦娥任务提供中继通信（示意为定点）',
    source: 'CNSA 公开轨道信息',
    stationaryOffset: [2.9, 1.0, -1.6], // 距月球中心 3.46（月球半径 2.6，悬浮于球外空中）
    a: 0,
    e: 0,
    inclinationDeg: 0,
    raanDeg: 0,
    argPeriapsisDeg: 0,
    periodSeconds: 0,
  },
]
