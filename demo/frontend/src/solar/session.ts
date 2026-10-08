/**
 * 太阳系页面模式记忆：
 * "显示当前位置"（真实公转位置）只在当前应用运行期间保留。
 * 组件卸载/重挂载时继续记忆；刷新或重新打开应用后恢复默认一字排布。
 */
let realPositions = false

export const solarSession = {
  get realPositions(): boolean {
    return realPositions
  },
  set realPositions(value: boolean) {
    realPositions = value
  },
}
