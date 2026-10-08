import type { SceneLayers } from './types'

/**
 * 所有天体特写页面共用的图层状态（不含太阳系总览）。
 * 与地球 ORBIT 页的既有逻辑一致：应用启动时飞行器默认显示，
 * 用户在任一天体中的切换会跟随当前应用会话进入和退出其他天体。
 */
export function createDefaultSceneLayers(): SceneLayers {
  return { spacecraft: true, orbits: true, sites: true }
}
