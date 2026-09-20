/**
 * 太阳系页面模式记忆：
 * "显示当前位置"（真实公转位置）模式记录在浏览器 localStorage 中，
 * 组件卸载/重挂载、页面刷新后都保留；首次访问（无记录）为默认一字排布。
 */
const STORAGE_KEY = 'aurora.solar.realPositions'
// v5 是完全独立的用户偏好：旧版本在初始化和迁移期间写入的隐藏状态不再参与判断。
// 键不存在时默认显示；只有用户主动切换后才写入，并在跨天体、刷新时保留。
const SPACECRAFT_VISIBILITY_STORAGE_KEY = 'aurora.solar.spacecraftVisible.v5'

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
      return localStorage.getItem(SPACECRAFT_VISIBILITY_STORAGE_KEY) !== '0'
    } catch {
      return true
    }
  },
  set spacecraftVisible(value: boolean) {
    try {
      localStorage.setItem(SPACECRAFT_VISIBILITY_STORAGE_KEY, value ? '1' : '0')
    } catch {
      // 本次会话继续可用；下次无有效记录时仍安全回退为默认显示
    }
  },
}
