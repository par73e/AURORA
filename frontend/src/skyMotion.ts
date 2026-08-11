export function normalizeAzimuth(value: number) {
  return (value % 360 + 360) % 360
}

// 返回从 current 到 target 的最短有符号转角，避免跨越正北时绕行近一整圈。
export function shortestAzimuthDelta(current: number, target: number) {
  return (normalizeAzimuth(target) - normalizeAzimuth(current) + 540) % 360 - 180
}

export function skyTurnDuration(delta: number) {
  return Math.min(700, Math.max(320, 280 + Math.abs(delta) * 2.2))
}

export function easeOutExpo(progress: number) {
  if (progress >= 1) return 1
  return 1 - 2 ** (-10 * Math.max(0, progress))
}
