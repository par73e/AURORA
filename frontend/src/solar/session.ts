/**
 * 太阳系页面模式记忆：
 * "显示当前位置"（真实公转位置）模式记录在浏览器 localStorage 中，
 * 组件卸载/重挂载、页面刷新后都保留；首次访问（无记录）为默认一字排布。
 */
const STORAGE_KEY = 'aurora.solar.realPositions'
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
}
