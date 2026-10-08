function equal(a: unknown, b: unknown, field = ''): boolean {
  if (typeof a === 'number' && typeof b === 'number' && ['x', 'y', 'anchorX', 'anchorY', 'radiusPx'].includes(field)) return Math.abs(a - b) < 0.05
  if (a === b) return true
  if (!a || !b || typeof a !== 'object' || typeof b !== 'object') return false
  const x = a as Record<string, unknown>, y = b as Record<string, unknown>
  const keys = Object.keys(x)
  return keys.length === Object.keys(y).length && keys.every(key => Object.hasOwn(y, key) && equal(x[key], y[key], key))
}
/** Subpixel noise should not repeatedly invalidate Vue's entire label tree. */
export function stableLayouts<T extends object>(previous: T[], next: T[]): T[] {
  const old = new Map(previous.map(item => [(item as { id?: string }).id, item]))
  const result = next.map(item => {
    const value = { ...item } as Record<string, unknown>
    const cached = old.get((item as { id?: string }).id)
    return (cached && equal(cached, value) ? cached : value) as T
  })
  return previous.length === result.length && result.every((item, i) => item === previous[i]) ? previous : result
}
