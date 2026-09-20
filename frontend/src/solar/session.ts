/**
 * 太阳系页面模式记忆：
 * "显示当前位置"（真实公转位置）模式记录在浏览器 localStorage 中，
 * 组件卸载/重挂载、页面刷新后都保留；首次访问（无记录）为默认一字排布。
 */
const STORAGE_KEY = 'aurora.solar.realPositions'
// 只有同时存在当前版本标记的值才视为用户主动选择；旧键或迁移期间误写的隐藏值一律忽略。
// 这样所有场景首次进入都默认显示，用户之后的主动切换仍能跨页面、刷新保留。
const SPACECRAFT_VISIBILITY_STORAGE_KEY = 'aurora.solar.spacecraftVisible.v4'
const SPACECRAFT_VISIBILITY_VERSION_KEY = 'aurora.solar.spacecraftVisibilityVersion'
const SPACECRAFT_VISIBILITY_VERSION = '1'

export const solarSession = {
  get realPositions(): boolean {
    try {
      return localStorage.getItem(STORAGE_KEY) === '1'
    } catch {
      return false
    }
  },
  set realPositions(value: boolean) {
    try {
      localStorage.setItem(STORAGE_KEY, value ? '1' : '0')
    } catch {
      // 隐私模式/禁用存储时静默忽略，仅本次会话生效
    }
  },
  get spacecraftVisible(): boolean {
    try {
      if (localStorage.getItem(SPACECRAFT_VISIBILITY_VERSION_KEY) !== SPACECRAFT_VISIBILITY_VERSION) return true
      return localStorage.getItem(SPACECRAFT_VISIBILITY_STORAGE_KEY) !== '0'
    } catch {
      return true
    }
  },
  set spacecraftVisible(value: boolean) {
    try {
      localStorage.setItem(SPACECRAFT_VISIBILITY_STORAGE_KEY, value ? '1' : '0')
      localStorage.setItem(SPACECRAFT_VISIBILITY_VERSION_KEY, SPACECRAFT_VISIBILITY_VERSION)
    } catch {
      // 本次会话继续可用；未写入版本标记时，下次仍安全回退为默认显示
    }
  },
}
