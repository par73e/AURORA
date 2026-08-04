// 太阳系示意数据与纹理资源。
// 纹理来源：Solar System Scope（https://www.solarsystemscope.com/textures/），CC BY 4.0。
// 布局为静态示意：太阳位于右上角（非居中），行星静止在各自轨道上沿右上→左下方向排成一线；
// 小行星带与柯伊伯带保留；大小与距离只保留大致相对关系，非等比。

import mercuryUrl from '../assets/solar/4k_mercury.jpg'
import venusUrl from '../assets/solar/4k_venus_atmosphere.jpg'
import earthUrl from '../assets/solar/2k_earth_daymap.jpg' // SSS 平光贴图：与全系行星风格统一（BMNG 自带烘焙阴影，不适配场景光照；4K 概率性卡顿，回退 2K）
import marsUrl from '../assets/solar/4k_mars.jpg'
import jupiterUrl from '../assets/solar/8k_jupiter.jpg'
import saturnUrl from '../assets/solar/8k_saturn.jpg'
import saturnRingUrl from '../assets/solar/8k_saturn_ring_alpha.png'
import uranusUrl from '../assets/solar/2k_uranus.jpg'
import neptuneUrl from '../assets/solar/2k_neptune.jpg'
import moonUrl from '../assets/solar/2k_moon.jpg'
import moon8kUrl from '../assets/solar/8k_moon.jpg'
import sunUrl from '../assets/solar/8k_sun.jpg'

export interface RingSpec {
  kind: 'saturn' | 'uranus'
  /** 环内/外半径，以行星视觉半径为单位 */
  inner: number
  outer: number
}

export interface PlanetSpec {
  id: string
  name: string
  nameEn: string
  note: string
  /** 视觉半径（示意尺度，只保留大致相对关系） */
  radius: number
  /** 轨道半径（示意尺度） */
  orbitRadius: number
  /** 真实自转轴倾角（度）；金星/天王星的逆向通过大于 90° 的倾角表达 */
  axialTiltDeg: number
  /** 真实自转周期（小时），仅用于按压缩比例展示自转 */
  rotationHours: number
  textureUrl: string
  ring?: RingSpec
  /** J2000 历元（2000-01-01 12:00 TT）平黄经（度） */
  meanLongitudeDeg: number
  /** 近日点黄经（度） */
  perihelionLongitudeDeg: number
  /** 公转周期（天） */
  periodDays: number
  /** 轨道离心率（用于 Kepler 方程求真近点角） */
  eccentricity: number
}

/** 月球（地球的天然卫星，独立数据条目）：半径约为地球 0.27，位置固定在地球右侧 */
export const MOON = {
  id: 'moon',
  name: '月球',
  nameEn: 'MOON',
  note: '地球唯一的天然卫星',
  radius: 0.68,
  distance: 3.6,
  /** 真实模式（太阳居中）下的公转周期（秒，视觉压缩） */
  orbitSeconds: 50,
  textureUrl: moonUrl,
}

/** 月球高清贴图（8k，月球独立页面使用） */
export const MOON_HD = { textureUrl: moon8kUrl }

export const SUN_RADIUS = 5.5
export const SUN_ROTATION_SECONDS = 240

/** 自转周期压缩：地球自转一圈约 100 秒（行星不公转，仅保留缓慢自转） */
export const ROTATION_TIME_SCALE = 24

export function rotationPeriodSeconds(realHours: number) {
  return Math.pow(Math.abs(realHours), 0.45) * ROTATION_TIME_SCALE
}

/** 行星在轨道上的固定方位角（黄道面 XZ 平面内世界坐标，0° = +x 方向）。
 *  屏幕方向 = (u·right, -u·upv) = (cos(θ+φ), sinε·sin(θ+φ))；
 *  俯仰 28° 下 θ=165.1° 时行星连线约 29°，与 16:9 屏幕对角线（29.4°）对齐 */
export const PLANET_LINE_ANGLE_DEG = 165.1

export const planets: PlanetSpec[] = [
  {
    id: 'mercury', name: '水星', nameEn: 'MERCURY', note: '最靠近太阳的岩石行星',
    radius: 1.6, orbitRadius: 14, axialTiltDeg: 0.03, rotationHours: 1407.6, textureUrl: mercuryUrl, meanLongitudeDeg: 252.25084, perihelionLongitudeDeg: 77.45645, periodDays: 87.969, eccentricity: 0.20563,
  },
  {
    id: 'venus', name: '金星', nameEn: 'VENUS', note: '被浓密云层包裹的行星',
    radius: 2.3, orbitRadius: 30, axialTiltDeg: 177.4, rotationHours: 5832.5, textureUrl: venusUrl, meanLongitudeDeg: 181.97973, perihelionLongitudeDeg: 131.76746, periodDays: 224.701, eccentricity: 0.006772,
  },
  {
    id: 'earth', name: '地球', nameEn: 'EARTH', note: '进入近地轨道与航天活动观测',
    radius: 2.5, orbitRadius: 46, axialTiltDeg: 23.44, rotationHours: 23.93, textureUrl: earthUrl, meanLongitudeDeg: 100.46435, perihelionLongitudeDeg: 102.93735, periodDays: 365.256, eccentricity: 0.016708,
  },
  {
    id: 'mars', name: '火星', nameEn: 'MARS', note: '具有氧化铁红色地表的行星',
    radius: 2.0, orbitRadius: 60, axialTiltDeg: 25.19, rotationHours: 24.62, textureUrl: marsUrl, meanLongitudeDeg: 355.45332, perihelionLongitudeDeg: 336.04084, periodDays: 686.98, eccentricity: 0.0934,
  },
  {
    id: 'jupiter', name: '木星', nameEn: 'JUPITER', note: '太阳系中体积最大的行星',
    radius: 6.0, orbitRadius: 82, axialTiltDeg: 3.13, rotationHours: 9.93, textureUrl: jupiterUrl, meanLongitudeDeg: 34.40438, perihelionLongitudeDeg: 14.75385, periodDays: 4332.589, eccentricity: 0.048494,
  },
  {
    id: 'saturn', name: '土星', nameEn: 'SATURN', note: '拥有广阔而明亮的行星环',
    radius: 5.0, orbitRadius: 98, axialTiltDeg: 26.73, rotationHours: 10.66, textureUrl: saturnUrl, meanLongitudeDeg: 49.94432, perihelionLongitudeDeg: 92.43194, periodDays: 10759.22, eccentricity: 0.055509,
    ring: { kind: 'saturn', inner: 1.24, outer: 2.33 },
  },
  {
    id: 'uranus', name: '天王星', nameEn: 'URANUS', note: '近乎侧躺旋转的冰巨星',
    radius: 3.5, orbitRadius: 114, axialTiltDeg: 97.77, rotationHours: 17.24, textureUrl: uranusUrl, meanLongitudeDeg: 313.23218, perihelionLongitudeDeg: 170.96424, periodDays: 30688.5, eccentricity: 0.046381,
    ring: { kind: 'uranus', inner: 1.6, outer: 1.66 }, // 细环：约一条线的宽度
  },
  {
    id: 'neptune', name: '海王星', nameEn: 'NEPTUNE', note: '遥远而深蓝的冰巨星',
    radius: 3.4, orbitRadius: 130, axialTiltDeg: 28.32, rotationHours: 16.11, textureUrl: neptuneUrl, meanLongitudeDeg: 304.88003, perihelionLongitudeDeg: 44.97135, periodDays: 60182.0, eccentricity: 0.009456,
  },
]

/** 小行星带：火星（60）与木星（82）轨道之间 */
export const ASTEROID_BELT = {
  inner: 66,
  outer: 74,
  count: 2600,
  sizeMin: 0.055,
  sizeMax: 0.24,
  spreadY: 0.9,
  // 公转动画压缩比：真实 3-6 年 → 900s（15 分钟一圈）——缓慢漂移而非快速旋转
  periodSeconds: 900,
  color: 0x9a8f80,
}

/** 柯伊伯带：海王星（130）轨道之外 */
export const KUIPER_BELT = {
  inner: 136,
  outer: 160,
  count: 4200,
  sizeMin: 0.06,
  sizeMax: 0.3,
  spreadY: 1.6,
  periodSeconds: 1500,
  color: 0x8fa8b5,
}

export const STAR_COUNT = 5000

/** 相机构图：斜俯视黄道面；主视角中心对准小行星带（火星与木星之间），
 *  整体画面上移；太阳落在右上、海王星落在左下，行星连线与屏幕对角线对齐 */
export const VIEW = {
  fov: 42,
  elevationDeg: 28,
  azimuthDeg: -35,
  /** 滚轮缩放近限：可贴近到行星表面之上，仅保留防穿模下限 */
  minDistance: 0.15,
  /** 构图距离下限：尽量贴近以拉大太阳-海王星跨度（相机仍保持在柯伊伯带之外） */
  composeMinDistance: 115,
  composeMaxDistance: 900,
  /** 视角中心（小行星带）锚定位置：水平居中、垂直略偏上（整体上移） */
  anchorScreenX: 0.5,
  anchorScreenY: 0.45,
  /** 真实位置模式：太阳锚定位置（屏幕从上往下比例；0.38 = 太阳偏高、横向居中） */
  realAnchorScreenY: 0.38,
  /** 真实位置模式：水平边距系数（0.75 = 再放大一点点，海王星轨道部分出屏） */
  realFitMargin: 0.75,
}

export const SUN = {
  name: '太阳',
  nameEn: 'SUN',
  note: '太阳系中心恒星，占太阳系总质量的 99.86%',
  textureUrl: sunUrl,
}

/** 全部纹理 URL（用于切换页面时的预加载预热） */
export const ALL_TEXTURE_URLS = [
  sunUrl,
  mercuryUrl,
  venusUrl,
  earthUrl,
  marsUrl,
  jupiterUrl,
  saturnUrl,
  uranusUrl,
  neptuneUrl,
  saturnRingUrl,
  moonUrl,
]

/** 土星环透明度纹理（径向条带，2048×125 RGBA） */
export const SATURN_RING_TEXTURE_URL = saturnRingUrl
