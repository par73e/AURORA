// 除地球/月球外的全部行星特写页共享配置。
// 页面骨架与动画（入场自转、晨昏线、退出淡出）由 PlanetScene.vue 统一实现（火星为独立 MarsScene 组件，
// 但共享 MARS_PAGE 配置中的档案数据）；行星差异（纹理、轴倾角、环、自转方向、太阳日、季节、主题色、档案数据）全部收敛在这里。
//
// 数据来源（与全站"数据透明"原则一致，特写页档案板块标注 NASA）：
// - 直径 / 距日 / 自转 / 公转 / 轴倾角 / 卫星数：NASA Science 行星事实页（2026-08 核实）
// - 纹理：Solar System Scope（CC BY 4.0），与太阳系页同源同缓存
// - 季节锚点：土星用 2009-08-11 北半球春分（Cassini 观测到的真实春分）；
//   火星用 2026-10-01 火星年 39 春分（知识库 08-06 记录）；
//   金星/木星/水星黄赤交角极小（≤3.13°），锚点为近似（误差 ≤3° 视觉无差别）；
//   天王星/海王星季节周期极长（84/165 年），锚点为近似（详见 planetSun 注释）

import venusUrl from './assets/solar/4k_venus_atmosphere.jpg'
import mercuryUrl from './assets/solar/4k_mercury.jpg'
import marsUrl from './assets/solar/4k_mars.jpg'
import jupiterUrl from './assets/solar/8k_jupiter.jpg'
import saturnUrl from './assets/solar/8k_saturn.jpg'
import uranusUrl from './assets/solar/2k_uranus.jpg'
import neptuneUrl from './assets/solar/2k_neptune.jpg'
import sunUrl from './assets/solar/8k_sun.jpg'
import { SATURN_RING_TEXTURE_URL } from './solar/data'

export type OuterPlanetKey = 'mercury' | 'venus' | 'mars' | 'saturn' | 'jupiter' | 'uranus' | 'neptune' | 'sun'

export interface PlanetProfile {
  /** 板块 kicker（英文，mono 小字） */
  kicker: string
  /** 直径（NASA） */
  diameter: string
  /** 距日（NASA） */
  distance: string
  /** 自转周期（NASA） */
  rotation: string
  /** 太阳日（NASA，日出到日落） */
  solarDay: string
  /** 公转周期（NASA） */
  orbit: string
  /** 轴倾角（NASA；>90° 表示逆向自转） */
  axialTilt: string
  /** 卫星数（NASA） */
  moons: string
  /** 环 */
  rings: string
  /** 简介（来自 NASA 页面） */
  description: string
}

/** 晨昏线真实太阳方向参数（与火星 marsSunDirection 同构，参数按行星真实值） */
export interface PlanetSun {
  /** 黄赤交角（度）：用于子日点赤纬幅度；金星 177.4° 等效季节倾角 = |180-177.4| = 2.64° */
  axialTiltDeg: number
  /** 太阳日（小时）：子日点经度推进 360°/太阳日（金星 116.75 天 = 2802h） */
  solarDayHours: number
  /** 季节周期（天）：赤纬按行星公转周期摆动 */
  seasonPeriodDays: number
  /** 季节相位锚点（毫秒，UTC）：该时刻太阳赤纬 = 0 且向北（春分）；
   *  金星/木星幅度极小，锚点为近似（见文件头注释） */
  seasonAnchorMs: number
}

export interface PlanetPageConfig {
  key: OuterPlanetKey
  /** 中文名 / 英文名 */
  name: string
  nameEn: string
  /** 主题 class 后缀（style.css 用 .desktop-app.<key> 覆盖全站色） */
  themeKey: string
  /** 恒星模式（太阳）：自发光材质、无晨昏线（工具栏隐藏昼夜开关）、无环 */
  star?: boolean
  /** 场景纹理（与太阳系页同源，solarTexture 缓存复用） */
  textureUrl: string
  /** 环（可选；inner/outer 以行星视觉半径为单位；kind: 'saturn' 用环带纹理，'uranus' 用程序化 13 细环） */
  ring?: { inner: number; outer: number; textureUrl?: string; color?: number; kind?: 'saturn' | 'uranus' }
  /** 场景视觉半径：以地球 ORBIT 页基准半径 2.15 为锚，按真实直径做 sqrt 压缩
   *  （radius = 2.15 × √(真实直径/地球直径 12,742km)），保持真实大小排序且差异可感知 */
  radius: number
  /** 真实轴倾角（度，场景 tiltPivot 使用；金星 177.4° = 倒置自转轴，逆向自转由轴倾角表达） */
  axialTiltDeg: number
  /** 入场自转方向（与太阳系场景一致：绕倾斜后的极轴正方向；逆向行星靠 >90° 轴倾角表达） */
  spinSign: 1 | -1
  /** 默认相机距离（土星带环略远，保证环完整入画） */
  defaultDistance: number
  /** 晨昏线太阳方向参数 */
  sun: PlanetSun
  /** 档案板块数据（NASA） */
  profile: PlanetProfile
  /** 人类探索板块数据（着陆点/任务终点；无则省略板块） */
  exploration?: PlanetExploration
}

/** 行星表面着陆点 / 任务终点（人类探索足迹）。
 *  数据来源：Wikipedia 任务页（2026-08 核实，与 NASA 页面交叉确认）；
 *  精确度：有坐标的为真实历史坐标（±150km 着陆精度），大气坠毁（气态行星无固体表面）无坐标，
 *  只入目录不做场景标记。 */
export interface PlanetSite {
  id: string
  /** 中文名 */
  name: string
  /** 英文名 */
  nameEn: string
  /** 任务名 */
  mission: string
  /** 机构 */
  operator: string
  /** 日期（着陆/撞击/坠毁） */
  date: string
  /** 纬度（无表面坐标的大气坠毁为 null） */
  latitude: number | null
  /** 经度（无表面坐标的大气坠毁为 null） */
  longitude: number | null
  /** 类型：landing=软着陆 / impact=表面撞击 / atmospheric=大气层坠毁 */
  kind: 'landing' | 'impact' | 'atmospheric'
  /** 图标：lander=着陆器 / probe=探测器 / impact=撞击 */
  icon: 'lander' | 'probe' | 'impact'
  /** 一句话简介 */
  description: string
}

export interface PlanetExploration {
  /** 板块标题（'着陆点' / '任务终点'） */
  title: string
  /** 板块 kicker（英文 mono 小字） */
  kicker: string
  /** 板块副标题 */
  sub: string
  sites: PlanetSite[]
}

export const VENUS_PAGE: PlanetPageConfig = {
  key: 'venus',
  name: '金星',
  nameEn: 'VENUS',
  themeKey: 'venus',
  textureUrl: venusUrl,
  radius: 2.1,
  axialTiltDeg: 177.4, // 逆向自转：自转轴几乎倒置（177.4° 表达"太阳从西边升起"）
  spinSign: 1, // 自转方向与太阳系一致（绕倾斜后的极轴正方向）
  defaultDistance: 9,
  sun: {
    axialTiltDeg: 177.4,
    // 金星太阳日 116.75 地球日（NASA：日出到日落 117 天）——子日点推进极慢（约 3.1°/天）
    solarDayHours: 116.75 * 24,
    seasonPeriodDays: 224.7,
    // 等效季节倾角仅 2.64°，锚点近似即可（任意误差 ≤ 2.64°，视觉无差别）
    seasonAnchorMs: Date.UTC(2026, 0, 1),
  },
  profile: {
    kicker: 'VENUS PROFILE',
    diameter: '12,104 km',
    distance: '0.72 AU',
    rotation: '243 天（逆向自转）',
    solarDay: '117 天',
    orbit: '225 天',
    axialTilt: '177.4°（逆向）',
    moons: '0',
    rings: '无',
    description: '被浓密硫酸云层包裹的炽热行星，太阳从西边升起。',
  },
  exploration: {
    title: '着陆点',
    kicker: 'VENUS LANDING SITES',
    sub: '着陆器 · 在金星表面着陆的航天器',
    sites: [
      {
        id: 'venera-7',
        name: 'Venera 7',
        nameEn: 'Venera 7',
        mission: 'Venera 计划',
        operator: '苏联',
        date: '1970-12-15',
        latitude: -5,
        longitude: 351,
        kind: 'landing',
        icon: 'lander',
        description: '首个在地球外软着陆并传回数据的航天器（表面信号 23 分钟）。',
      },
      {
        id: 'venera-8',
        name: 'Venera 8',
        nameEn: 'Venera 8',
        mission: 'Venera 计划',
        operator: '苏联',
        date: '1972-07-22',
        latitude: -10.7,
        longitude: 335.25,
        kind: 'landing',
        icon: 'lander',
        description: 'Vasilisa Regio，表面工作 50 分钟，测得照度足以拍照。',
      },
      {
        id: 'venera-9',
        name: 'Venera 9',
        nameEn: 'Venera 9',
        mission: 'Venera 计划',
        operator: '苏联',
        date: '1975-10-22',
        latitude: 31.01,
        longitude: 291.64,
        kind: 'landing',
        icon: 'lander',
        description: '首个传回金星表面黑白照片的着陆器。',
      },
      {
        id: 'venera-10',
        name: 'Venera 10',
        nameEn: 'Venera 10',
        mission: 'Venera 计划',
        operator: '苏联',
        date: '1975-10-25',
        latitude: 15.42,
        longitude: 291.51,
        kind: 'landing',
        icon: 'lander',
        description: 'Beta Regio 与 Hyndla Regio 交界处，表面工作约 65 分钟。',
      },
      {
        id: 'venera-11',
        name: 'Venera 11',
        nameEn: 'Venera 11',
        mission: 'Venera 计划',
        operator: '苏联',
        date: '1978-12-25',
        latitude: -14,
        longitude: 299,
        kind: 'landing',
        icon: 'lander',
        description: '着陆后 95 分钟，相机遮光罩未弹出未能成像。',
      },
      {
        id: 'venera-12',
        name: 'Venera 12',
        nameEn: 'Venera 12',
        mission: 'Venera 计划',
        operator: '苏联',
        date: '1978-12-21',
        latitude: -7,
        longitude: 294,
        kind: 'landing',
        icon: 'lander',
        description: '着陆后工作约 110 分钟，记录到金星大气放电。',
      },
      {
        id: 'venera-13',
        name: 'Venera 13',
        nameEn: 'Venera 13',
        mission: 'Venera 计划',
        operator: '苏联',
        date: '1982-03-01',
        latitude: -7.5,
        longitude: 303,
        kind: 'landing',
        icon: 'lander',
        description: '传回首张金星表面彩色照片，表面工作 127 分钟。',
      },
      {
        id: 'venera-14',
        name: 'Venera 14',
        nameEn: 'Venera 14',
        mission: 'Venera 计划',
        operator: '苏联',
        date: '1982-03-05',
        latitude: -13.25,
        longitude: 310,
        kind: 'landing',
        icon: 'lander',
        description: '着陆点约在 13.25°S 310°E（Phoebe Regio 东侧），表面工作约 57 分钟。',
      },
    ],
  },
}

export const MARS_PAGE: PlanetPageConfig = {
  key: 'mars',
  name: '火星',
  nameEn: 'MARS',
  themeKey: 'mars',
  textureUrl: marsUrl,
  radius: 1.57,
  axialTiltDeg: 25.19,
  spinSign: 1,
  defaultDistance: 9,
  sun: {
    axialTiltDeg: 25.19,
    solarDayHours: 24.6229,
    seasonPeriodDays: 687,
    // 真实锚点：火星年 39 春分 Ls=0 ≈ 2026-10-01（dayOfYear 274，知识库 08-06 记录）
    seasonAnchorMs: Date.UTC(2026, 9, 1),
  },
  profile: {
    kicker: 'MARS PROFILE',
    diameter: '6,780 km',
    distance: '1.5 AU',
    rotation: '24.6 小时',
    solarDay: '24.7 小时（1 sol）',
    orbit: '687 天（669.6 sols）',
    axialTilt: '25°',
    moons: '2（Phobos / Deimos）',
    rings: '无',
    description: '因氧化铁而呈现红色的沙漠世界，拥有太阳系最大的火山与峡谷。',
  },
}

export const SATURN_PAGE: PlanetPageConfig = {
  key: 'saturn',
  name: '土星',
  nameEn: 'SATURN',
  themeKey: 'saturn',
  textureUrl: saturnUrl,
  ring: { inner: 1.24, outer: 2.33, textureUrl: SATURN_RING_TEXTURE_URL, kind: 'saturn' },
  radius: 6.6,
  axialTiltDeg: 26.73,
  spinSign: 1,
  defaultDistance: 20.8, // 视半径 ~18.5°（> 天王星 17°，< 木星 19.8°）
  sun: {
    axialTiltDeg: 26.73,
    solarDayHours: 10.656,
    seasonPeriodDays: 10759.22,
    // 真实锚点：2009-08-11 土星北半球春分（Cassini 观测到的真实春分，Ls=0）
    seasonAnchorMs: Date.UTC(2009, 7, 11),
  },
  profile: {
    kicker: 'SATURN PROFILE',
    diameter: '120,500 km',
    distance: '9.5 AU',
    rotation: '10.7 小时',
    solarDay: '10.7 小时',
    orbit: '29.4 年（10,756 天）',
    axialTilt: '26.73°',
    moons: '274（2025-03 确认）',
    rings: '有（延伸 282,000 km）',
    description: '拥有广阔而明亮行星环的气态巨行星，环由冰与岩石碎块构成。',
  },
  exploration: {
    title: '任务终点',
    kicker: 'SATURN MISSION ENDPOINTS',
    sub: '任务终点 · 结束任务并坠入土星的航天器',
    sites: [
      {
        id: 'cassini',
        name: 'Cassini 卡西尼号',
        nameEn: 'Cassini',
        mission: 'Cassini–Huygens',
        operator: 'NASA / ESA / ASI',
        date: '2017-09-15',
        latitude: null,
        longitude: null,
        kind: 'atmospheric',
        icon: 'probe',
        description: 'Grand Finale 终结：22 次穿越环缝后坠入土星上层大气烧毁（信号 11:55:46 UTC 消失）。',
      },
    ],
  },
}

export const JUPITER_PAGE: PlanetPageConfig = {
  key: 'jupiter',
  name: '木星',
  nameEn: 'JUPITER',
  themeKey: 'jupiter',
  textureUrl: jupiterUrl,
  radius: 7.1,
  axialTiltDeg: 3.13,
  spinSign: 1,
  defaultDistance: 21,
  sun: {
    axialTiltDeg: 3.13,
    solarDayHours: 9.925,
    seasonPeriodDays: 4332.589,
    // 黄赤交角仅 3.13°，锚点近似（误差 ≤ 3.13°，视觉无差别）
    seasonAnchorMs: Date.UTC(2021, 0, 1),
  },
  profile: {
    kicker: 'JUPITER PROFILE',
    diameter: '139,822 km',
    distance: '5.2 AU',
    rotation: '9.9 小时',
    solarDay: '9.9 小时',
    orbit: '11.9 年（4,333 天）',
    axialTilt: '3.13°',
    moons: '95（IAU 确认）',
    rings: '有（暗淡，不易见）',
    description: '太阳系最大的行星，云带与风暴（大红斑）是其标志。',
  },
  exploration: {
    title: '任务终点',
    kicker: 'JUPITER MISSION ENDPOINTS',
    sub: '任务终点 · 结束任务并坠入木星的航天器',
    sites: [
      {
        id: 'galileo',
        name: 'Galileo 伽利略号',
        nameEn: 'Galileo',
        mission: 'Galileo',
        operator: 'NASA',
        date: '2003-09-21',
        latitude: null,
        longitude: null,
        kind: 'atmospheric',
        icon: 'probe',
        description: '为防止污染木卫二，燃料耗尽后故意坠入木星大气焚毁。',
      },
    ],
  },
}

export const MERCURY_PAGE: PlanetPageConfig = {
  key: 'mercury',
  name: '水星',
  nameEn: 'MERCURY',
  themeKey: 'mercury',
  textureUrl: mercuryUrl,
  radius: 1.33,
  axialTiltDeg: 0.03,
  spinSign: 1,
  defaultDistance: 9,
  sun: {
    axialTiltDeg: 0.03,
    // 水星太阳日 176 地球日（NASA：一个昼夜循环 = 176 天，两个水星年）——子日点推进极慢
    solarDayHours: 176 * 24,
    seasonPeriodDays: 87.969,
    // 轴倾角仅 0.03°，几乎无季节，锚点任意（误差 ≤ 0.03°，无视觉影响）
    seasonAnchorMs: Date.UTC(2026, 0, 1),
  },
  profile: {
    kicker: 'MERCURY PROFILE',
    diameter: '4,880 km',
    distance: '0.4 AU',
    rotation: '59 天',
    solarDay: '176 天（两个水星年）',
    orbit: '88 天',
    axialTilt: '0.03°（几乎直立）',
    moons: '0',
    rings: '无',
    description: '最靠近太阳的行星，昼夜温差极大，表面布满陨石坑。',
  },
  exploration: {
    title: '任务终点',
    kicker: 'MERCURY MISSION ENDPOINTS',
    sub: '任务终点 · 结束任务并撞击水星表面的航天器',
    sites: [
      {
        id: 'messenger',
        name: 'MESSENGER',
        nameEn: 'MESSENGER',
        mission: 'MESSENGER',
        operator: 'NASA',
        date: '2015-04-30',
        latitude: 54.4,
        longitude: 210.1, // 149.9°W = 210.1°E（与 Venera 一致的东经约定）
        kind: 'impact',
        icon: 'impact',
        description: '燃料耗尽后主动撞击水星表面，留下一个约 15 米宽的撞击坑。',
      },
    ],
  },
}

export const URANUS_PAGE: PlanetPageConfig = {
  key: 'uranus',
  name: '天王星',
  nameEn: 'URANUS',
  themeKey: 'uranus',
  textureUrl: uranusUrl,
  ring: { inner: 1.58, outer: 2.4, kind: 'uranus' }, // 程序化 13 细环（Zeta..μ，覆盖真实环系范围）
  radius: 4.3,
  axialTiltDeg: 97.77,
  spinSign: 1,
  defaultDistance: 14.7, // 视半径 ~17°（> 地球 16.4°，< 土星 18.5°）
  sun: {
    axialTiltDeg: 97.77,
    solarDayHours: 17.24,
    seasonPeriodDays: 30688.5,
    // 等效季节倾角 = |180-97.77| = 82.23°：太阳直射点可达极高纬度（极端季节）。
    // 天王星最近春分（Ls=0）约 2007-12-07（Voyager 2 之后的观测记录），此处为近似锚点
    seasonAnchorMs: Date.UTC(2007, 11, 7),
  },
  profile: {
    kicker: 'URANUS PROFILE',
    diameter: '51,118 km',
    distance: '19.2 AU',
    rotation: '17 小时（逆向自转）',
    solarDay: '17 小时',
    orbit: '84 年（30,687 天）',
    axialTilt: '97.77°（侧躺）',
    moons: '28',
    rings: '有（13 条细环）',
    description: '近乎侧躺旋转的冰巨星，拥有太阳系最极端的季节。',
  },
}

export const NEPTUNE_PAGE: PlanetPageConfig = {
  key: 'neptune',
  name: '海王星',
  nameEn: 'NEPTUNE',
  themeKey: 'neptune',
  textureUrl: neptuneUrl,
  radius: 4.2,
  axialTiltDeg: 28.32,
  spinSign: 1,
  defaultDistance: 14.5, // 视半径 ~16.8°（> 地球 16.4°，< 天王星 17°）
  sun: {
    axialTiltDeg: 28.32,
    solarDayHours: 16.11,
    seasonPeriodDays: 60190.0,
    // 海王星季节周期 165 年，当前相位锚点为近似（季节极缓慢，误差视觉影响小）
    seasonAnchorMs: Date.UTC(2005, 0, 1),
  },
  profile: {
    kicker: 'NEPTUNE PROFILE',
    diameter: '49,528 km',
    distance: '30 AU',
    rotation: '16 小时',
    solarDay: '16 小时',
    orbit: '165 年（60,190 天）',
    axialTilt: '28.32°',
    moons: '16',
    rings: '有（5 条主环，暗淡）',
    description: '遥远而深蓝的冰巨星，拥有太阳系最快的狂风。',
  },
}

export const SUN_PAGE: PlanetPageConfig = {
  key: 'sun',
  name: '太阳',
  nameEn: 'SUN',
  themeKey: 'sun',
  star: true,
  textureUrl: sunUrl,
  radius: 22.5,
  axialTiltDeg: 7.25,
  spinSign: 1,
  defaultDistance: 66, // 视半径 ~19.9°（半视场 21° 内完整入画，接近满屏但不超出）
  // 恒星无昼夜/晨昏线：sun 字段仅占位（恒星模式不使用，见 PlanetScene star 分支）
  sun: {
    axialTiltDeg: 7.25,
    solarDayHours: 0,
    seasonPeriodDays: 0,
    seasonAnchorMs: Date.UTC(2026, 0, 1),
  },
  profile: {
    kicker: 'SUN PROFILE',
    diameter: '1,392,700 km',
    distance: '0（太阳系中心）',
    rotation: '25.4 天（赤道，较差自转）',
    solarDay: '—',
    orbit: '2.3 亿年（绕银河系中心）',
    axialTilt: '7.25°',
    moons: '8（八大行星）',
    rings: '无',
    description: '太阳系中心恒星，占太阳系总质量的 99.86%，以氢氦核聚变发光。',
  },
}

export const OUTER_PLANET_PAGES: Record<OuterPlanetKey, PlanetPageConfig> = {
  mercury: MERCURY_PAGE,
  venus: VENUS_PAGE,
  mars: MARS_PAGE,
  saturn: SATURN_PAGE,
  jupiter: JUPITER_PAGE,
  uranus: URANUS_PAGE,
  neptune: NEPTUNE_PAGE,
  sun: SUN_PAGE,
}
