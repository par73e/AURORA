/** A missing/stale snapshot must never be shown as a fresh satellite position. */
export function positionMix(time: number, start: number, end: number): number | null {
  if (!Number.isFinite(time) || end <= start || time - end > 5000) return null
  return Math.max(0, Math.min(1, (time - start) / (end - start)))
}
