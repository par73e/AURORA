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
  /** 主要物质组成；行星写内部/大气主成分，恒星写元素丰度 */
  composition: string
  /** 简介（来自 NASA 页面） */
  description: string
  /** 恒星（太阳）专属档案字段 */
  type?: string
  mass?: string
  surfaceTemp?: string
  coreTemp?: string
  age?: string
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
  /** 3D 场景强调色（十六进制）：探测器轨迹线/标记圆点/着陆点标记统一用行星主题色 */
  sceneAccent: number
  /** 3D 场景强调色-弱（半透明轨迹/标签引线用） */
  sceneAccentDim?: number
  /** 恒星模式（太阳）：自发光材质、无晨昏线（工具栏隐藏昼夜开关）、无环 */
  star?: boolean
  /** 场景纹理（与太阳系页同源，solarTexture 缓存复用） */
  textureUrl: string
  /** 只旋转行星表面纹理，不改变轨迹/标记；用于把最有辨识度的地貌放进首屏 */
  surfaceYawDeg?: number
  /** 观测光强度；默认 3.1，纹理对比强的行星可单独收敛高光 */
  observationLightIntensity?: number
  /** 环（可选；inner/outer 以行星视觉半径为单位；kind: 'saturn' 用环带纹理，'uranus' 用程序化 13 细环） */
  ring?: { inner: number; outer: number; textureUrl?: string; color?: number; opacity?: number; kind?: 'saturn' | 'uranus' }
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
  /** 探测器/飞掠器板块（静态任务资料；轨迹为视觉示意，不冒充实时星历） */
  spacecraft?: PlanetSpacecraft
}

export type PlanetCraftStatus = '运行中' | '已结束' | '即将入轨' | '飞掠' | '在途'

interface PlanetCraftTrajectoryBase {
  /** 轨迹相对行星半径；太阳页同样使用相对半径，便于统一渲染 */
  radius: number
  /** 椭圆轨道偏心率（orbit）；只用于视觉压缩，不代表精密星历 */
  eccentricity?: number
  /** 轨道/飞掠平面相对行星赤道面的倾角 */
  inclinationDeg?: number
  /** 起始相位（度） */
  phaseDeg?: number
  /** 首屏标记在轨迹上的初始位置（0..1）；运行中任务从这里继续运动 */
  displayProgress?: number
  /** 飞掠弧线跨度（度） */
  spanDeg?: number
  /** 终点距球心的相对半径，1.04 表示上层大气/云顶附近 */
  endpointRadius?: number
}

/** 完整环绕轨道按真实时间运行，历史任务用注明阶段的代表性周期。 */
export interface PlanetOrbitTrajectory extends PlanetCraftTrajectoryBase {
  kind: 'orbit'
  /** 真实任务轨道周期（地球日）；贝皮科伦坡号暂保留原有展示。 */
  periodDays: number
  /** 周期对应阶段；历史动画不代表飞行器仍在执行任务。 */
  stage?: string
}

/** 开放路径（飞掠 / 接近 / 大气进入）：只表示任务阶段与方向，不是闭合轨道，没有周期 */
export interface PlanetOpenTrajectory extends PlanetCraftTrajectoryBase {
  kind: 'flyby' | 'approach' | 'entry'
}

export type PlanetCraftTrajectory = PlanetOrbitTrajectory | PlanetOpenTrajectory

export interface PlanetCraft {
  id: string
  name: string
  nameEn: string
  operator: string
  status: PlanetCraftStatus
  type: string
  /** 发射日期；抵达、入轨及飞掠日期写入描述或任务终点，避免混用。 */
  date: string
  description: string
  trajectory?: PlanetCraftTrajectory
  /** 任务终点文案；没有坐标时仍可显示在详情卡和目录中 */
  endpoint?: string
  /** 太阳任务若实际绕地球运行，关联 ORBIT 中的同一航天器。 */
  orbitCatalogId?: string
  /** 数据核实日期，用于静态任务资料的诚实标注 */
  verifiedAt?: string
  source?: string
}

export interface PlanetSpacecraft {
  title: string
  kicker: string
  /** 单条稀有任务使用紧凑档案，不展示无意义的搜索、筛选和分页 */
  compact?: boolean
  items: PlanetCraft[]
}

const SUN_SPACECRAFT: PlanetSpacecraft = {
  title: '飞行器',
  kicker: 'SOLAR PROBES',
  items: [
    {
      id: 'parker-solar-probe', name: '帕克太阳探测器', nameEn: 'Parker Solar Probe', operator: 'NASA',
      status: '运行中', type: '日心轨道器', date: '2018-08-12', endpoint: '近日点 6.1M km · 0.04 AU',
      description: '目前仍在运行的太阳探测器，2024-12-24 抵达距太阳约 6.1M km 的历史最近点。',
      // 默认相位放在太阳右下侧的近端轨道，进入页面即可看到运行标记与标签。
      trajectory: { kind: 'orbit', radius: 2.3, eccentricity: 0.5, inclinationDeg: 8, phaseDeg: 18, displayProgress: 0.76, periodDays: 88, stage: '2024 年末最后一次金星借力后的近日轨道' },
      verifiedAt: '2026-08-06', source: 'NASA Parker Solar Probe',
    },
    {
      id: 'solar-orbiter', name: '太阳轨道器', nameEn: 'Solar Orbiter', operator: 'ESA / NASA',
      status: '运行中', type: '日心轨道器', date: '2020-02-09', endpoint: '太阳近距离成像与极区观测',
      description: '欧空局与 NASA 合作任务，结合遥感成像和原位测量研究太阳风与太阳极区。',
      // 采用标志性的 168 天科学轨道；其他变轨阶段不在本展示中复原。
      trajectory: { kind: 'orbit', radius: 2.6, eccentricity: 0.35, inclinationDeg: 23, phaseDeg: 265, displayProgress: 0.35, periodDays: 168, stage: '太阳科学观测阶段 · 168 天共振轨道' },
      verifiedAt: '2026-09-29', source: 'NASA Solar Orbiter',
    },
    {
      id: 'soho', name: '太阳和日球层探测器', nameEn: 'SOHO', operator: 'ESA / NASA',
      status: '运行中', type: '日地 L1 太阳观测器', date: '1995-12-02', endpoint: '日地 L1 附近运行',
      description: '在日地 L1 附近持续观测太阳内部、日冕和太阳风；没有近太阳飞掠轨道。',
      // 刻意不给轨迹：SOHO 在日地 L1 晕轨道运行（离太阳约 0.99 AU），既不是绕日轨道，
      // 在太阳页的尺度下也会远在画面之外。目录行标注"仅档案 · 场景无标记"。
      verifiedAt: '2026-09-29', source: 'NASA SOHO mission archive',
    },
    {
      id: 'stereo-a', name: '日地关系观测站 A', nameEn: 'STEREO-A', operator: 'NASA',
      status: '运行中', type: '日心轨道太阳观测器', date: '2006-10-26', endpoint: '持续观测太阳风暴',
      description: '双星任务中仍在工作的 A 星，从不同视角追踪日冕物质抛射。',
      // 刻意不给轨迹：STEREO-A 在约 1 AU 的日心轨道，太阳页尺度下会远在画面之外。
      verifiedAt: '2026-09-29', source: 'NASA STEREO mission archive',
    },
    {
      id: 'aditya-l1', name: '太阳神 L1 号', nameEn: 'Aditya-L1', operator: 'ISRO',
      status: '运行中', type: '日地 L1 太阳观测器', date: '2023-09-02', endpoint: '日地 L1 晕轨道',
      description: '印度首个专门研究太阳的空间观测站，2024-01-06 进入日地 L1 晕轨道。',
      // 刻意不给轨迹：与 SOHO 同理，L1 晕轨道在太阳页尺度下无法有意义地表示。
      verifiedAt: '2026-09-29', source: 'ISRO Aditya-L1 mission',
    },
    {
      id: 'helios-b', name: '太阳神 B 号', nameEn: 'Helios-B', operator: 'NASA / DLR',
      status: '已结束', type: '日心轨道器', date: '1976-01-15', endpoint: '近日点 0.29 AU',
      description: '1976 年进入高偏心日心轨道，创下长期保持的近太阳探测距离纪录。',
      trajectory: { kind: 'orbit', radius: 2.45, eccentricity: 0.55, inclinationDeg: 3, phaseDeg: 156, displayProgress: 0.1, periodDays: 185, stage: '1976 年日心科学轨道 · 历史轨道演示' },
      verifiedAt: '2026-08-06', source: 'NASA / DLR mission archive',
    },
    {
      id: 'ulysses', name: '尤利西斯号', nameEn: 'Ulysses', operator: 'ESA / NASA',
      status: '已结束', type: '太阳极区探测器', date: '1990-10-06', endpoint: '高倾角太阳极轨',
      description: '首个系统研究太阳南北极区的探测器，2009 年结束任务。',
      // 高倾角轨道做视觉压缩，避免默认标记落到首屏外；仍保留显著的太阳极轨形态。
      trajectory: { kind: 'orbit', radius: 2.2, eccentricity: 0.18, inclinationDeg: 66, phaseDeg: 235, displayProgress: 0.76, periodDays: 2260, stage: '太阳极区科学观测轨道 · 历史轨道演示' },
      verifiedAt: '2026-08-06', source: 'ESA Ulysses archive',
    },
    {
      id: 'sdo-solar', name: '太阳动力学观测站', nameEn: 'SDO', operator: 'NASA',
      status: '运行中', type: '地球同步轨道太阳观测站', date: '2010-02-11', endpoint: '地球倾斜同步轨道 · ORBIT 可查看实时位置',
      description: '持续拍摄太阳全日面与磁场变化。它环绕地球运行，轨道和飞行器实体在地球 ORBIT 页面展示。',
      orbitCatalogId: 'sdo', verifiedAt: '2026-09-29', source: 'NASA SDO · CelesTrak 36395',
    },
    {
      id: 'hinode-solar', name: '日出号', nameEn: 'Hinode', operator: 'JAXA / NASA / ESA',
      status: '运行中', type: '地球太阳同步轨道太阳观测站', date: '2006-09-23', endpoint: '地球太阳同步轨道 · ORBIT 可查看实时位置',
      description: '研究太阳磁场、色球层和日冕。它环绕地球运行，轨道和飞行器实体在地球 ORBIT 页面展示。',
      orbitCatalogId: 'hinode', verifiedAt: '2026-09-29', source: 'NASA Hinode · CelesTrak 29479',
    },
  ],
}

const MERCURY_SPACECRAFT: PlanetSpacecraft = {
  title: '飞行器',
  kicker: 'MERCURY SPACECRAFT',
  items: [
    {
      id: 'bepicolombo', name: '贝皮科伦坡号', nameEn: 'BepiColombo', operator: 'ESA / JAXA',
      status: '即将入轨', type: '接近中的轨道器', date: '2018-10-20', endpoint: '预计 2026-11 弱捕获入轨',
      description: '欧日联合水星任务，2026-06-15 关闭电推进，进入水星轨道捕获准备阶段。',
      trajectory: { kind: 'orbit', radius: 2.05, eccentricity: 0.48, inclinationDeg: 12, phaseDeg: 20, periodDays: 120 },
      verifiedAt: '2026-08-06', source: 'ESA BepiColombo',
    },
    {
      id: 'messenger-orbiter', name: '信使号', nameEn: 'MESSENGER', operator: 'NASA',
      status: '已结束', type: '轨道器', date: '2004-08-03', endpoint: '2015-04-30 撞击水星',
      description: '首个环绕水星运行的探测器，任务终点为水星表面的真实撞击点。',
      trajectory: { kind: 'orbit', radius: 1.7, eccentricity: 0.3, inclinationDeg: 7, phaseDeg: 190, periodDays: 0.5, stage: '2011 年主科学任务的 12 小时轨道 · 历史轨道演示' },
      verifiedAt: '2026-08-06', source: 'NASA MESSENGER mission archive',
    },
    {
      id: 'mariner-10', name: '水手 10 号', nameEn: 'Mariner 10', operator: 'NASA',
      status: '飞掠', type: '飞掠器', date: '1973-11-03', endpoint: '3 次水星飞掠',
      description: '首个抵达水星的飞行器，1974–1975 年完成三次水星飞掠。',
      trajectory: { kind: 'flyby', radius: 2.2, inclinationDeg: 18, phaseDeg: 28, spanDeg: 135 },
      verifiedAt: '2026-08-06', source: 'NASA Mariner 10 archive',
    },
  ],
}

const VENUS_SPACECRAFT: PlanetSpacecraft = {
  title: '飞行器',
  kicker: 'VENUS SPACECRAFT',
  items: [
    {
      id: 'akatsuki', name: '晓号', nameEn: 'Akatsuki', operator: 'JAXA',
      status: '已结束', type: '轨道器', date: '2010-05-20', endpoint: '2024-05-29 失联',
      description: '日本金星气候轨道器，曾长期观测金星云层和大气环流，后于 2024 年失联。',
      trajectory: { kind: 'orbit', radius: 1.72, eccentricity: 0.28, inclinationDeg: 5, phaseDeg: 95, periodDays: 10.8, stage: '2016 年轨道调整后的气象观测阶段 · 历史轨道演示' },
      verifiedAt: '2026-08-06', source: 'JAXA Akatsuki archive',
    },
    {
      id: 'magellan', name: '麦哲伦号', nameEn: 'Magellan', operator: 'NASA',
      status: '已结束', type: '轨道器', date: '1989-05-04', endpoint: '1994-10-13 坠入金星大气',
      description: '完成金星表面雷达测绘，任务结束时进入大气层烧毁。',
      // 采用气动制动前的雷达测绘轨道，周期 195 分钟；动画按真实时间运行。
      trajectory: { kind: 'orbit', radius: 1.9, eccentricity: 0.18, inclinationDeg: 9, phaseDeg: 230, endpointRadius: 1.04, periodDays: 195 / 1440, stage: '1990–1993 年雷达测绘阶段 · 历史轨道演示' },
      verifiedAt: '2026-08-06', source: 'NASA/JPL Successful Aerobraking Experiment · 制动前 195 分钟轨道',
    },
    {
      id: 'pioneer-venus', name: '先驱者金星 1 号', nameEn: 'Pioneer Venus 1', operator: 'NASA',
      status: '已结束', type: '轨道器', date: '1978-05-20', endpoint: '1992 年进入金星大气',
      description: '美国首个金星轨道器，从轨道研究金星大气、磁场和太阳风相互作用。',
      trajectory: { kind: 'orbit', radius: 1.7, eccentricity: 0.22, inclinationDeg: 12, phaseDeg: 200, endpointRadius: 1.04, periodDays: 1, stage: '1978–1992 年金星科学观测阶段 · 历史轨道演示' },
      verifiedAt: '2026-09-29', source: 'NASA Venus Exploration',
    },
    {
      id: 'pioneer-venus-2', name: '先驱者金星 2 号', nameEn: 'Pioneer Venus 2', operator: 'NASA',
      status: '已结束', type: '多探测器大气任务', date: '1978-08-08', endpoint: '1978-12 四枚探测器进入金星大气',
      description: '一枚大型和三枚小型探测器同时采集金星大气剖面数据。',
      // 大气进入路径为历史示意：任务同时投放四枚探测器，此处只画一条代表弧线，不是闭合轨道。
      trajectory: { kind: 'entry', radius: 2.25, inclinationDeg: 15, phaseDeg: 40, displayProgress: 0.3 },
      verifiedAt: '2026-09-29', source: 'NASA Pioneer Venus 2 mission archive',
    },
    {
      id: 'venus-express', name: '金星快车号', nameEn: 'Venus Express', operator: 'ESA',
      status: '已结束', type: '轨道器', date: '2005-11-09', endpoint: '2014-12 任务结束',
      description: '欧洲首个金星轨道器，2006-04-11 入轨，长期研究金星大气、等离子体和表面。',
      trajectory: { kind: 'orbit', radius: 2.05, eccentricity: 0.28, inclinationDeg: 82, phaseDeg: 145, periodDays: 1, stage: '2006–2014 年常规科学观测阶段 · 历史轨道演示' },
      verifiedAt: '2026-09-29', source: 'NASA Venus Express mission archive',
    },
    {
      id: 'venera-15', name: '金星 15 号', nameEn: 'Venera 15', operator: '苏联',
      status: '已结束', type: '雷达测绘轨道器', date: '1983-06-02', endpoint: '金星北半球雷达测绘',
      description: '与金星 16 号协同，以侧视雷达绘制金星北部地形。',
      // 近极轨道（倾角约 87°）、周期约 24 小时；轨道为任务期示意，不是当前星历。
      trajectory: { kind: 'orbit', radius: 2.3, eccentricity: 0.25, inclinationDeg: 87, phaseDeg: 45, periodDays: 1, stage: '1983–1984 年北半球雷达测绘阶段 · 历史轨道演示' },
      verifiedAt: '2026-09-29', source: 'NASA Venus Exploration',
    },
    {
      id: 'venera-16', name: '金星 16 号', nameEn: 'Venera 16', operator: '苏联',
      status: '已结束', type: '雷达测绘轨道器', date: '1983-06-07', endpoint: '金星北半球雷达测绘',
      description: '与金星 15 号组成双轨道器雷达测绘任务。',
      // 近极轨道（倾角约 87°）、周期约 24 小时；取不同相位避免与 15 号的标记重合。
      trajectory: { kind: 'orbit', radius: 2.34, eccentricity: 0.26, inclinationDeg: 87, phaseDeg: 200, periodDays: 1, stage: '1983–1984 年北半球雷达测绘阶段 · 历史轨道演示' },
      verifiedAt: '2026-09-29', source: 'NASA Venus Exploration',
    },
    {
      id: 'mariner-2-venus', name: '水手 2 号', nameEn: 'Mariner 2', operator: 'NASA',
      status: '飞掠', type: '飞掠器', date: '1962-08-27', endpoint: '1962-12-14 飞掠金星',
      description: '首个成功飞掠另一颗行星的航天器，测量了金星大气与行星际环境。',
      trajectory: { kind: 'flyby', radius: 2.4, inclinationDeg: 11, phaseDeg: 32, spanDeg: 130 },
      verifiedAt: '2026-09-29', source: 'NASA Venus Exploration',
    },
    {
      id: 'vega-1-venus', name: '维加 1 号', nameEn: 'Vega 1', operator: '苏联',
      status: '飞掠', type: '飞掠器 + 着陆器 + 气球', date: '1984-12-15', endpoint: '1985-06-11 金星着陆与气球探测',
      description: '主飞行器在投放着陆器与大气气球后继续飞往哈雷彗星；着陆器落点见下方目录。',
      // 金星飞掠弧线为历史示意（1985-06-11 投放着陆器与气球后转往哈雷彗星），不是闭合轨道。
      trajectory: { kind: 'flyby', radius: 2.35, inclinationDeg: 13, phaseDeg: 55, spanDeg: 130 },
      verifiedAt: '2026-09-29', source: 'NASA Venus Exploration · Deep Space Chronicle',
    },
    {
      id: 'vega-2-venus', name: '维加 2 号', nameEn: 'Vega 2', operator: '苏联',
      status: '飞掠', type: '飞掠器 + 着陆器 + 气球', date: '1984-12-21', endpoint: '1985-06-15 金星着陆与气球探测',
      description: '第二组金星着陆器与大气气球，主飞行器后续前往哈雷彗星；着陆器落点见下方目录。',
      // 与维加 1 号同型的飞掠弧线，取不同相位避免两条弧线在首屏重叠。
      trajectory: { kind: 'flyby', radius: 2.42, inclinationDeg: 9, phaseDeg: 215, spanDeg: 130 },
      verifiedAt: '2026-09-29', source: 'NASA Venus Exploration · Deep Space Chronicle',
    },
    {
      id: 'venera-4', name: '金星 4 号', nameEn: 'Venera 4', operator: '苏联',
      status: '已结束', type: '大气探测器', date: '1967-06-12', endpoint: '1967-10-18 进入金星大气',
      description: '首次直接测得金星大气成分；探测器在下降过程中停止传输，未完成软着陆。入射线为历史示意。',
      trajectory: { kind: 'entry', radius: 2.3, inclinationDeg: 17, phaseDeg: 32, displayProgress: 0.32 },
      verifiedAt: '2026-09-29', source: 'ESA Past missions to Venus',
    },
    {
      id: 'venera-9-orbiter', name: '金星 9 号轨道器', nameEn: 'Venera 9 Orbiter', operator: '苏联',
      status: '已结束', type: '轨道器', date: '1975-06-08', endpoint: '1975-10 进入金星轨道；着陆器落点见下方',
      description: '金星 9 号分离着陆器后进入金星轨道，为着陆器数据提供中继。任务期轨道为视觉示意。',
      trajectory: { kind: 'orbit', radius: 2.24, eccentricity: 0.22, inclinationDeg: 36, phaseDeg: 70, displayProgress: 0.24, periodDays: 48.3 / 24, stage: '1975 年入轨后的科学与中继阶段 · 历史轨道演示' },
      verifiedAt: '2026-09-29', source: 'ESA Past missions to Venus',
    },
    {
      id: 'venera-10-orbiter', name: '金星 10 号轨道器', nameEn: 'Venera 10 Orbiter', operator: '苏联',
      status: '已结束', type: '轨道器', date: '1975-06-14', endpoint: '1975-10 进入金星轨道；着陆器落点见下方',
      description: '与金星 9 号组成双任务，轨道器完成观测及着陆器通信中继。任务期轨道为视觉示意。',
      trajectory: { kind: 'orbit', radius: 2.38, eccentricity: 0.27, inclinationDeg: 51, phaseDeg: 138, displayProgress: 0.61, periodDays: (49 * 60 + 23) / 1440, stage: '1975 年入轨后的科学与中继阶段 · 历史轨道演示' },
      verifiedAt: '2026-09-29', source: 'ESA Past missions to Venus',
    },
  ],
}

const JUPITER_SPACECRAFT: PlanetSpacecraft = {
  title: '飞行器',
  kicker: 'JUPITER SPACECRAFT',
  items: [
    {
      id: 'europa-clipper-jupiter', name: '欧罗巴快船', nameEn: 'Europa Clipper', operator: 'NASA',
      status: '在途', type: '木星系统探测器', date: '2024-10-14', endpoint: '预计 2030-04 抵达木星；目标为木卫二',
      description: '目前处于日心巡航途中，尚未进入木星轨道。场景开放线段仅表示接近方向，不是当前星历。',
      trajectory: { kind: 'approach', radius: 1.9, inclinationDeg: -10, phaseDeg: 12, displayProgress: 0.06 },
      verifiedAt: '2026-09-29', source: 'NASA Europa Clipper mission timeline',
    },
    {
      id: 'juice-jupiter', name: '木星冰卫星探测器', nameEn: 'Juice', operator: 'ESA',
      status: '在途', type: '木星系统探测器', date: '2023-04-14', endpoint: '预计 2031-07 抵达木星系统',
      description: '目前处于跨行星巡航途中，未来研究木卫三、木卫四和木卫二。场景开放线段不是当前星历。',
      trajectory: { kind: 'approach', radius: 2.0, inclinationDeg: -15, phaseDeg: -24, displayProgress: 0.08 },
      verifiedAt: '2026-09-29', source: 'ESA Juice mission overview',
    },
    {
      id: 'juno', name: '朱诺号', nameEn: 'Juno', operator: 'NASA',
      status: '运行中', type: '极轨轨道器', date: '2011-08-05', endpoint: '任务结束时间待官方确认',
      description: '2016-07-04 进入木星轨道，继续研究木星内部、磁场和极光。',
      // 视觉示意：保持极轨倾角，但把轨道收在镜头可读范围内，保证运行点不会长期游离出画面。
      // NASA Juno Orbits：初期为 53 天；卫星飞掠逐步缩短周期，2024-02 起采用约 33 天。
      trajectory: { kind: 'orbit', radius: 1.22, eccentricity: 0.18, inclinationDeg: 52, phaseDeg: 36, displayProgress: 0.34, periodDays: 33, stage: '2024 年 2 月木卫一飞掠后的 33 天轨道' },
      verifiedAt: '2026-09-29', source: 'NASA Juno Orbits · 2024-02 起的轨道周期',
    },
    {
      id: 'galileo', name: '伽利略号', nameEn: 'Galileo', operator: 'NASA',
      status: '已结束', type: '轨道器', date: '1989-10-18', endpoint: '2003-09-21 坠入木星大气',
      description: '为避免污染木卫二，燃料耗尽后按计划进入木星大气层焚毁。',
      // 采用 1996 年 G1 木卫三飞掠后的 72 天轨道；其他变轨阶段不在本展示中复原。
      trajectory: { kind: 'orbit', radius: 1.95, eccentricity: 0.22, inclinationDeg: 14, phaseDeg: 176, displayProgress: 0.02, endpointRadius: 1.04, periodDays: 72, stage: '1996 年 G1 木卫三飞掠后的主任务轨道 · 历史轨道演示' },
      verifiedAt: '2026-08-06', source: 'NASA Mission to Jupiter: A History of the Galileo Project · G1 后约 72 天轨道',
    },
    {
      id: 'galileo-probe-craft', name: '伽利略大气探测器', nameEn: 'Galileo Atmospheric Probe', operator: 'NASA',
      status: '已结束', type: '大气探测器', date: '1989-10-18', endpoint: '1995-12-07 进入木星大气',
      description: '随伽利略号发射，1995-07-13 从轨道器释放，在木星大气中持续回传约 58 分钟。',
      // 大气进入路径为历史示意（1995-12-07 进入木星大气），不是环绕轨道，故无周期。
      trajectory: { kind: 'entry', radius: 1.6, inclinationDeg: 14, phaseDeg: 176, displayProgress: 0.32 },
      verifiedAt: '2026-09-29', source: 'NASA Galileo Jupiter Atmospheric Probe',
    },
    {
      id: 'voyager-1-jupiter', name: '旅行者 1 号', nameEn: 'Voyager 1', operator: 'NASA',
      status: '飞掠', type: '飞掠器', date: '1977-09-05', endpoint: '1979-03-05 木星系统飞掠',
      description: '完成木星系统飞掠后继续前往外太阳系。',
      trajectory: { kind: 'flyby', radius: 1.45, inclinationDeg: 12, phaseDeg: 42, displayProgress: 1, spanDeg: 130 },
      verifiedAt: '2026-08-06', source: 'NASA Voyager archive',
    },
    {
      id: 'pioneer-10-jupiter', name: '先驱者 10 号', nameEn: 'Pioneer 10', operator: 'NASA',
      status: '飞掠', type: '飞掠器', date: '1972-03-02', endpoint: '1973-12-04 飞掠木星',
      description: '首个近距离探访木星的航天器，提供辐射环境与大气的先导观测。',
      trajectory: { kind: 'flyby', radius: 1.65, inclinationDeg: 16, phaseDeg: 210, spanDeg: 130 },
      verifiedAt: '2026-09-29', source: 'NASA Pioneer 10 mission archive',
    },
    {
      id: 'pioneer-11-jupiter', name: '先驱者 11 号', nameEn: 'Pioneer 11', operator: 'NASA',
      status: '飞掠', type: '飞掠器', date: '1973-04-06', endpoint: '1974-12-03 飞掠木星',
      description: '经木星极区飞掠后前往土星，获得木星极区近距离图像。',
      trajectory: { kind: 'flyby', radius: 1.75, inclinationDeg: 38, phaseDeg: 120, spanDeg: 135 },
      verifiedAt: '2026-09-29', source: 'NASA Pioneer 11 mission archive',
    },
    {
      id: 'voyager-2-jupiter', name: '旅行者 2 号', nameEn: 'Voyager 2', operator: 'NASA',
      status: '飞掠', type: '飞掠器', date: '1977-08-20', endpoint: '1979-07 木星系统飞掠',
      description: '继旅行者 1 号后研究木星及其卫星，随后继续前往土星、天王星与海王星。',
      trajectory: { kind: 'flyby', radius: 1.9, inclinationDeg: 10, phaseDeg: 75, spanDeg: 120 },
      verifiedAt: '2026-09-29', source: 'NASA Jupiter Exploration',
    },
    {
      id: 'new-horizons-jupiter', name: '新视野号', nameEn: 'New Horizons', operator: 'NASA',
      status: '飞掠', type: '飞掠器', date: '2006-01-19', endpoint: '2007-02-28 木星引力辅助',
      description: '飞往冥王星途中利用木星引力辅助，同时观测木星大气、环与卫星。',
      trajectory: { kind: 'flyby', radius: 2.1, inclinationDeg: 15, phaseDeg: 280, spanDeg: 125 },
      verifiedAt: '2026-09-29', source: 'NASA New Horizons mission archive',
    },
  ],
}

const SATURN_SPACECRAFT: PlanetSpacecraft = {
  title: '飞行器',
  kicker: 'SATURN SPACECRAFT',
  items: [
    {
      id: 'cassini', name: '卡西尼号', nameEn: 'Cassini', operator: 'NASA / ESA / ASI',
      status: '已结束', type: '轨道器', date: '1997-10-15', endpoint: '2017-09-15 坠入土星上层大气',
      description: '2004-07-01 入轨；搭载的惠更斯号于 2005 年降落土卫六。卡西尼号完成 22 次终章环缝穿越后坠入土星大气。',
      // 采用 2017 年 Grand Finale 的约 6.5 天轨道；其他变轨阶段不在本展示中复原。
      trajectory: { kind: 'orbit', radius: 1.72, eccentricity: 0.3, inclinationDeg: 24, phaseDeg: 142, displayProgress: 0.14, endpointRadius: 1.04, periodDays: 6.5, stage: '2017 年 Grand Finale 终章阶段 · 历史轨道演示' },
      verifiedAt: '2026-08-06', source: 'NASA Cassini Grand Finale Orbit Guide · 约 6.5 天轨道',
    },
    {
      id: 'pioneer-11-saturn', name: '先驱者 11 号', nameEn: 'Pioneer 11', operator: 'NASA',
      status: '飞掠', type: '飞掠器', date: '1973-04-06', endpoint: '1979-09-01 首次飞掠土星',
      description: '首个近距离探访土星的航天器，发现 F 环并研究环系与磁场。',
      trajectory: { kind: 'flyby', radius: 2.2, inclinationDeg: 17, phaseDeg: 105, spanDeg: 130 },
      verifiedAt: '2026-09-29', source: 'NASA Saturn Exploration',
    },
    {
      id: 'voyager-1-saturn', name: '旅行者 1 号', nameEn: 'Voyager 1', operator: 'NASA',
      status: '飞掠', type: '飞掠器', date: '1977-09-05', endpoint: '1980-11 土星系统飞掠',
      description: '揭示土星环细密结构，并近距离研究土卫六大气。',
      trajectory: { kind: 'flyby', radius: 2.35, inclinationDeg: 12, phaseDeg: 260, spanDeg: 125 },
      verifiedAt: '2026-09-29', source: 'NASA Saturn Exploration',
    },
    {
      id: 'voyager-2-saturn', name: '旅行者 2 号', nameEn: 'Voyager 2', operator: 'NASA',
      status: '飞掠', type: '飞掠器', date: '1977-08-20', endpoint: '1981-08 土星系统飞掠',
      description: '补充拍摄土星环与卫星，之后继续飞向天王星和海王星。',
      trajectory: { kind: 'flyby', radius: 2.5, inclinationDeg: 19, phaseDeg: 350, spanDeg: 125 },
      verifiedAt: '2026-09-29', source: 'NASA Saturn Exploration',
    },
  ],
}

const URANUS_SPACECRAFT: PlanetSpacecraft = {
  title: '飞行器',
  kicker: 'URANUS FLYBY',
  compact: true,
  items: [
    {
      id: 'voyager-2-uranus', name: '旅行者 2 号', nameEn: 'Voyager 2', operator: 'NASA',
      status: '飞掠', type: '飞掠器', date: '1977-08-20', endpoint: '1986-01-24 飞掠 · 距云顶约 81,500 km',
      description: '人类唯一一次天王星近距离探访，飞掠持续约 6 小时并发现了新的环与卫星。',
      trajectory: { kind: 'flyby', radius: 1.55, inclinationDeg: 18, phaseDeg: 135, displayProgress: 0.54, spanDeg: 145 },
      verifiedAt: '2026-08-06', source: 'NASA Voyager 2 archive',
    },
  ],
}

const NEPTUNE_SPACECRAFT: PlanetSpacecraft = {
  title: '飞行器',
  kicker: 'NEPTUNE FLYBY',
  compact: true,
  items: [
    {
      id: 'voyager-2-neptune', name: '旅行者 2 号', nameEn: 'Voyager 2', operator: 'NASA',
      status: '飞掠', type: '飞掠器', date: '1977-08-20', endpoint: '1989-08-25 飞掠 · 距云顶约 4,950 km',
      description: '完成海王星唯一一次近距离飞掠，观测到大暗斑和高速风暴。',
      trajectory: { kind: 'flyby', radius: 2.18, inclinationDeg: 34, phaseDeg: 198, displayProgress: 0.78, spanDeg: 145 },
      verifiedAt: '2026-08-06', source: 'NASA Voyager 2 archive',
    },
  ],
}

/** 行星表面着陆点 / 任务终点（人类探索足迹）。
 *  新增任务优先采用 NASA、JPL 等任务档案；早期条目的来源仍以各条记录为准。
 *  气态行星的大气进入经纬度是历史事件信息，不作为当前云图上的固定表面标记。 */
export interface PlanetSite {
  id: string
  /** 着陆所在天体；卫星着陆绝不能钉到母行星云图。 */
  body?: 'planet' | 'titan'
  /** 中文名 */
  name: string
  /** 英文名 */
  nameEn: string
  /** 任务名 */
  mission: string
  /** 机构 */
  operator: string
  /** 日期（着陆/撞击/大气进入） */
  date: string
  /** 纬度；只有官方发布或可由官方轨迹可靠复算时才填写 */
  latitude: number | null
  /** 东经；原始资料为西经时统一换算到 0..360°E */
  longitude: number | null
  /** 类型：landing=软着陆 / impact=表面撞击 / atmospheric=大气进入 */
  kind: 'landing' | 'impact' | 'atmospheric'
  /** 图标：lander=着陆器 / probe=探测器 / impact=撞击 */
  icon: 'lander' | 'probe' | 'impact'
  /** 一句话简介 */
  description: string
  /** 坐标/任务资料核实日期 */
  verifiedAt?: string
  /** 坐标/任务资料来源 */
  source?: string
}

export interface PlanetExploration {
  /** 板块标题（'着陆点' / '任务终点'） */
  title: string
  /** 板块 kicker（英文 mono 小字） */
  kicker: string
  /** 少量任务终点使用紧凑档案，不展示无意义的搜索框 */
  compact?: boolean
  sites: PlanetSite[]
}

export const VENUS_PAGE: PlanetPageConfig = {
  key: 'venus',
  name: '金星',
  nameEn: 'VENUS',
  themeKey: 'venus',
  sceneAccent: 0xf0e0b2,
  sceneAccentDim: 0x8a7d5e,
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
    composition: '岩石与金属 · 大气以二氧化碳为主，硫酸云覆盖',
    description: '被浓密硫酸云层包裹的炽热行星，太阳从西边升起。',
  },
  exploration: {
    title: '着陆点',
    kicker: 'VENUS LANDING SITES',
    sites: [
      {
        id: 'venera-7',
        name: '金星 7 号',
        nameEn: 'Venera 7',
        mission: '金星计划',
        operator: '苏联',
        date: '1970-12-15',
        latitude: -5,
        longitude: 351,
        kind: 'landing',
        icon: 'lander',
        description: '首个在地球外软着陆并传回数据的飞行器（表面信号 23 分钟）。',
      },
      {
        id: 'venera-8',
        name: '金星 8 号',
        nameEn: 'Venera 8',
        mission: '金星计划',
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
        name: '金星 9 号',
        nameEn: 'Venera 9',
        mission: '金星计划',
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
        name: '金星 10 号',
        nameEn: 'Venera 10',
        mission: '金星计划',
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
        name: '金星 11 号',
        nameEn: 'Venera 11',
        mission: '金星计划',
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
        name: '金星 12 号',
        nameEn: 'Venera 12',
        mission: '金星计划',
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
        name: '金星 13 号',
        nameEn: 'Venera 13',
        mission: '金星计划',
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
        name: '金星 14 号',
        nameEn: 'Venera 14',
        mission: '金星计划',
        operator: '苏联',
        date: '1982-03-05',
        latitude: -13.25,
        longitude: 310,
        kind: 'landing',
        icon: 'lander',
        description: '着陆点约在 13.25°S 310°E（Phoebe Regio 东侧），表面工作约 57 分钟。',
      },
      {
        id: 'vega-1',
        name: '维加 1 号',
        nameEn: 'Vega 1',
        mission: '维加计划',
        operator: '苏联',
        date: '1985-06-11',
        latitude: 7.2,
        longitude: 177.8,
        kind: 'landing',
        icon: 'lander',
        description: '着陆器在金星夜面传回数据约 56 分钟；同任务气球在云层中漂浮约 46.5 小时。',
        verifiedAt: '2026-09-29',
        source: 'NASA Deep Space Chronicle',
      },
      {
        id: 'vega-2',
        name: '维加 2 号',
        nameEn: 'Vega 2',
        mission: '维加计划',
        operator: '苏联',
        date: '1985-06-15',
        latitude: -6.45,
        longitude: 181.08,
        kind: 'landing',
        icon: 'lander',
        description: '着陆器分析金星地表，同任务气球在云层中测量大气后主飞行器继续飞往哈雷彗星。',
        verifiedAt: '2026-09-29',
        source: 'NASA Deep Space Chronicle',
      },
    ],
  },
  spacecraft: VENUS_SPACECRAFT,
}

export const MARS_PAGE: PlanetPageConfig = {
  key: 'mars',
  name: '火星',
  nameEn: 'MARS',
  themeKey: 'mars',
  sceneAccent: 0xff8a5c,
  sceneAccentDim: 0x8a4a30,
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
    composition: '岩石与铁镍硫核心 · 大气以二氧化碳为主',
    description: '因氧化铁而呈现红色的沙漠世界，拥有太阳系最大的火山与峡谷。',
  },
  // 火星页面仍由独立 MarsScene 负责实时目录，这里不复制数据。
}

export const SATURN_PAGE: PlanetPageConfig = {
  key: 'saturn',
  name: '土星',
  nameEn: 'SATURN',
  themeKey: 'saturn',
  sceneAccent: 0xe7c987,
  sceneAccentDim: 0x8f7c50,
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
    composition: '氢与氦为主 · 云层含氨等微量物质',
    description: '拥有广阔而明亮行星环的气态巨行星，环由冰与岩石碎块构成。',
  },
  exploration: {
    title: '着陆点与任务终点',
    kicker: 'SATURN SYSTEM SITES',
    compact: true,
    sites: [
      {
        id: 'huygens-titan', body: 'titan', name: '惠更斯号', nameEn: 'Huygens',
        mission: 'Cassini–Huygens', operator: 'ESA / NASA / ASI', date: '2005-01-14',
        latitude: -10.3, longitude: 167.7, kind: 'landing', icon: 'lander',
        description: '由卡西尼号送达土星系统，在土卫六大气中降落并成功软着陆；落点位于土卫六南纬 10.3°、西经 192.3°。',
        verifiedAt: '2026-09-29', source: 'ESA Location of landing site · NASA Cassini–Huygens',
      },
      {
        id: 'cassini',
        name: '卡西尼号',
        nameEn: 'Cassini',
        mission: 'Cassini–Huygens',
        operator: 'NASA / ESA / ASI',
        date: '2017-09-15',
        latitude: 9.4,
        longitude: 307, // NASA: 53°W = 307°E
        kind: 'atmospheric',
        icon: 'probe',
        description: 'Grand Finale 终结：22 次穿越环缝后坠入土星上层大气烧毁（信号 11:55:46 UTC 消失）。',
        verifiedAt: '2026-08-09',
        source: 'NASA / JPL Cassini End of Mission',
      },
    ],
  },
  spacecraft: SATURN_SPACECRAFT,
}

export const JUPITER_PAGE: PlanetPageConfig = {
  key: 'jupiter',
  name: '木星',
  nameEn: 'JUPITER',
  themeKey: 'jupiter',
  sceneAccent: 0xcf9257,
  sceneAccentDim: 0x7d5230,
  textureUrl: jupiterUrl,
  surfaceYawDeg: 52,
  observationLightIntensity: 2.45,
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
    composition: '氢与氦为主 · 含水、氨、甲烷等微量物质',
    description: '太阳系最大的行星，云带与风暴（大红斑）是其标志。',
  },
  exploration: {
    title: '任务终点',
    kicker: 'JUPITER MISSION ENDPOINTS',
    compact: true,
    sites: [
      {
        id: 'galileo',
        name: '伽利略号',
        nameEn: 'Galileo',
        mission: 'Galileo',
        operator: 'NASA',
        date: '2003-09-21',
        // JPL 任务资料给出约 0.25°S；Horizons GALILEO_MERGED 在 2003-09-21 18:57 UTC
        // 给出 0.27376°S、System III 167.58681°W，统一换算为 192.41319°E。
        latitude: -0.27376,
        longitude: 192.41319,
        kind: 'atmospheric',
        icon: 'probe',
        description: '为防止污染木卫二，燃料耗尽后故意坠入木星大气焚毁。',
        verifiedAt: '2026-08-09',
        source: 'NASA/JPL Galileo End of Mission · JPL Horizons GALILEO_MERGED',
      },
      {
        id: 'galileo-probe',
        name: '伽利略大气探测器',
        nameEn: 'Galileo Atmospheric Probe',
        mission: 'Galileo',
        operator: 'NASA',
        date: '1995-12-07',
        latitude: 6.5,
        longitude: 355.6, // NASA 给出 4.4°W，统一换算为东经。
        kind: 'atmospheric',
        icon: 'probe',
        description: '由伽利略轨道器释放，进入木星大气并持续回传约 58 分钟；与 2003 年轨道器的任务终点不同。',
        verifiedAt: '2026-09-29',
        source: 'NASA Galileo Jupiter Atmospheric Probe',
      },
    ],
  },
  spacecraft: JUPITER_SPACECRAFT,
}

export const MERCURY_PAGE: PlanetPageConfig = {
  key: 'mercury',
  name: '水星',
  nameEn: 'MERCURY',
  themeKey: 'mercury',
  sceneAccent: 0xc8b8a0,
  sceneAccentDim: 0x6e6252,
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
    composition: '大型金属核心 · 岩石地幔与固态地壳',
    description: '最靠近太阳的行星，昼夜温差极大，表面布满陨石坑。',
  },
  exploration: {
    title: '任务终点',
    kicker: 'MERCURY MISSION ENDPOINTS',
    sites: [
      {
        id: 'messenger',
        name: '信使号',
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
  spacecraft: MERCURY_SPACECRAFT,
}

export const URANUS_PAGE: PlanetPageConfig = {
  key: 'uranus',
  name: '天王星',
  nameEn: 'URANUS',
  themeKey: 'uranus',
  sceneAccent: 0x8fd8d0,
  sceneAccentDim: 0x4e8a84,
  textureUrl: uranusUrl,
  ring: { inner: 1.58, outer: 2.4, color: 0xa9c9c8, opacity: 0.52, kind: 'uranus' }, // 程序化 13 细环（Zeta..μ，覆盖真实环系范围）
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
    composition: '水、甲烷与氨等冰物质 · 氢氦大气',
    description: '近乎侧躺旋转的冰巨星，拥有太阳系最极端的季节。',
  },
  spacecraft: URANUS_SPACECRAFT,
}

export const NEPTUNE_PAGE: PlanetPageConfig = {
  key: 'neptune',
  name: '海王星',
  nameEn: 'NEPTUNE',
  themeKey: 'neptune',
  sceneAccent: 0x6aa8e0,
  sceneAccentDim: 0x3e5e84,
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
    composition: '水、甲烷与氨等冰物质 · 氢氦大气',
    description: '遥远而深蓝的冰巨星，拥有太阳系最快的狂风。',
  },
  spacecraft: NEPTUNE_SPACECRAFT,
}

export const SUN_PAGE: PlanetPageConfig = {
  key: 'sun',
  name: '太阳',
  nameEn: 'SUN',
  themeKey: 'sun',
  sceneAccent: 0xff6b4a,
  sceneAccentDim: 0x8a3526,
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
    type: 'G2V 黄矮星（主序星）',
    diameter: '1,392,700 km',
    mass: '1.989 × 10³⁰ kg（约 33 万倍地球）',
    surfaceTemp: '约 5,500 °C（表面）',
    coreTemp: '约 1,500 万 °C（核心）',
    distance: '距地球约 1.5 亿 km（1 AU）',
    age: '约 46 亿年',
    rotation: '25.4 天（赤道，较差自转）',
    composition: '氢约 73% · 氦约 25%',
    // 以下为行星模板必填字段（恒星模板不使用，占位）
    solarDay: '—',
    orbit: '—',
    axialTilt: '—',
    moons: '—',
    rings: '—',
    description: '太阳系中心恒星，占太阳系总质量的 99.86%，以氢氦核聚变发光。',
  },
  spacecraft: SUN_SPACECRAFT,
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
