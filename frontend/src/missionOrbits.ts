import type { MoonSpacecraft } from './types'

/** 一圈严格使用真实秒数；按绝对经过时间计算，掉帧/后台暂停不会拖长周期。 */
export function orbitProgress(elapsedSeconds: number, periodSeconds: number, initialProgress = 0): number {
  const initial = Number.isFinite(initialProgress) ? initialProgress : 0
  const turns = Number.isFinite(periodSeconds) && periodSeconds > 0 && Number.isFinite(elapsedSeconds)
    ? elapsedSeconds / periodSeconds : 0
  return ((initial + turns) % 1 + 1) % 1
}

export function orbitPeriodSeconds(craft: { periodSeconds: number; snapshot?: { periodSeconds: number } | null }): number {
  const snapshotPeriod = craft.snapshot?.periodSeconds
  return snapshotPeriod != null && Number.isFinite(snapshotPeriod) && snapshotPeriod > 0
    ? snapshotPeriod : craft.periodSeconds
}

/** 贝皮科伦坡号是用户指定保留旧展示的唯一例外。 */
export function planetOrbitPeriodSeconds(id: string, periodDays: number): number {
  return id === 'bepicolombo' ? Math.min(3000, Math.max(30, periodDays / 0.1)) : periodDays * 86400
}

/** 面板与动画读取同一个数值，保留约数而非暗示轨道全程不变。 */
export function orbitPeriodText(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return '周期暂不可用'
  const number = (value: number) => Number(value.toFixed(1)).toString()
  if (seconds < 3600) return `约 ${number(seconds / 60)} 分钟`
  if (seconds <= 72 * 3600) return `约 ${number(seconds / 3600)} 小时`
  return `约 ${number(seconds / 86400)} 天`
}

const historicalMoonStages: Record<string, string> = {
  'kaguya': '2007–2009 年月球测绘阶段 · 历史轨道演示',
  'chandrayaan-1': '2008 年约 100 km 科学轨道 · 历史轨道演示',
  'change-1': '2007–2009 年约 200 km 测绘轨道 · 历史轨道演示',
  'grail-a': '2012 年主科学任务阶段 · 历史轨道演示',
  'grail-b': '2012 年主科学任务阶段 · 历史轨道演示',
  'luna-10': '1966 年首次环月科学任务 · 历史轨道演示',
  'apollo-8': '1968 年载人绕月阶段 · 历史轨道演示',
}

export function moonOrbitStage(craft: MoonSpacecraft): string {
  if (historicalMoonStages[craft.id]) return historicalMoonStages[craft.id]
  if (craft.id === 'queqiao2') return '2024 年进入的环月中继使命轨道'
  if (craft.snapshot) return '轨道快照所对应阶段'
  return '公开标称任务轨道'
}

/** 兼容旧 API 中被误设为 L2 定点的鹊桥二号；不影响鹊桥一号的晕轨道示意。
 * CNSA: https://www.cnsa.gov.cn/n6758823/n6758838/c10503942/content.html
 * 24 小时是真实使命周期；orbitA/orbitE 仅负责现有场景中的视觉压缩。
 */
export function normalizeMoonOrbit(craft: MoonSpacecraft): MoonSpacecraft {
  // NASA 19700011921 的任务表给出月球 10 号周期为 178 分钟。
  if (craft.id === 'luna-10') return { ...craft, periodSeconds: 178 * 60, snapshot: null, displayPeriod: '约 178 分钟' }
  if (craft.id !== 'queqiao2' || craft.kind !== 'stationary') return craft
  return {
    ...craft, kind: 'orbital', periodSeconds: 86400, snapshot: null,
    orbitA: 2.3, orbitE: 0.6, inclinationDeg: 62.4, raanDeg: 25, argPeriapsisDeg: 90,
    displayInclination: '约 62.4', displayEccentricity: '约 0.80', displayPeriod: '约 24 小时',
    description: '在周期约 24 小时的环月大椭圆使命轨道上，为嫦娥四号、嫦娥六号等月背任务提供中继通信；场景轨道尺寸为视觉压缩。',
    sourceName: 'CNSA 鹊桥二号中继星任务取得圆满成功 · 2024-04-12',
  }
}
