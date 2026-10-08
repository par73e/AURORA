/** 所有天体特写从太阳系进入时，共用的裸天体自转时长。 */
export const ENTRY_SPIN_DURATION_MS = 800

/** 从预置偏角单向转到最终朝向；二次减速使停稳时角速度归零。 */
export function entrySpinAngle(offset: number, startedAt: number, now: number): number {
  const progress = Math.max(0, Math.min(1, (now - startedAt) / ENTRY_SPIN_DURATION_MS))
  if (progress >= 1) return 0
  return offset * (1 - progress) ** 2
}

export function entrySpinFinished(startedAt: number, now: number): boolean {
  return now - startedAt >= ENTRY_SPIN_DURATION_MS
}
