import type { NightWindow } from './astronomy'

/** 午夜后继续当前夜晚；晨光结束后再规划当晚。所有推荐都从现在起。 */
export function remainingNightWindow(now: Date, tonight: NightWindow | null, previous: NightWindow | null): NightWindow | null {
  const at = now.getTime()
  const night = previous && at >= previous.start.getTime() && at < previous.end.getTime() ? previous : tonight
  if (!night) return null
  // 与界面分钟精度一致，向上取整避免显示已经过去的这一分钟。
  const start = Math.ceil(Math.max(night.start.getTime(), at) / 60_000) * 60_000
  return start < night.end.getTime() ? { start: new Date(start), end: night.end } : null
}

/** 逐小时预报覆盖该小时，不能把相邻未来时段当成当前证据。 */
export function forecastAt<T extends { time: string }>(hours: T[], at: Date, parse: (value: string) => Date): T | null {
  const time = at.getTime()
  return hours.find((hour) => {
    const start = parse(hour.time).getTime()
    return time >= start && time < start + 60 * 60_000
  }) ?? null
}

export function nightForecast<T extends { time: string }>(hours: T[], night: NightWindow, parse: (value: string) => Date): T[] {
  return hours.filter((hour) => {
    const start = parse(hour.time).getTime()
    return start < night.end.getTime() && start + 60 * 60_000 > night.start.getTime()
  })
}
