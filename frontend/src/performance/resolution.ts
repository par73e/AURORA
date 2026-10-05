/** Only sustained foreground frame pressure changes resolution; isolated stalls do not. */
export class AdaptiveResolution {
  readonly maximum: number
  readonly minimum: number
  ratio: number
  private bad = 0
  private good = 0
  private changedAt: number

  constructor(maximum: number, startedAt = 0) {
    this.maximum = Math.max(0.5, Math.min(2, maximum || 1))
    this.minimum = Math.min(1, this.maximum)
    this.ratio = this.maximum
    this.changedAt = startedAt
  }

  resetStreaks() { this.bad = this.good = 0 }

  observe(meanMs: number, p95Ms: number, now: number): boolean {
    if (now - this.changedAt < 3000) { this.resetStreaks(); return false }
    this.bad = meanMs > 1000 / 45 ? this.bad + 1 : 0
    this.good = meanMs < 17.5 && p95Ms < 21 ? this.good + 1 : 0
    let next = this.ratio
    if (this.bad >= 2) next = Math.max(this.minimum, Math.round(this.ratio * 0.85 * 100) / 100)
    else if (this.good >= 6) next = Math.min(this.maximum, Math.round((this.ratio + 0.1) * 100) / 100)
    if (next === this.ratio) return false
    this.ratio = next
    this.changedAt = now
    this.resetStreaks()
    return true
  }
}
