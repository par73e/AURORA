/** Short-lived JSON data only. No storage, service worker, timer, image or GPU cache. */
interface CacheEntry {
  text: string
  expiresAt: number
  bytes: number
}

interface Flight {
  controller: AbortController
  promise: Promise<string>
  waiters: number
  settled: boolean
}

interface ClientOptions {
  fetcher?: typeof fetch
  now?: () => number
  maxBytes?: number
  maxEntries?: number
}

function abortReason(signal?: AbortSignal): unknown {
  return signal?.reason ?? new DOMException('请求已取消', 'AbortError')
}

/** Exact location/timezone parameters remain separate; real-time results are excluded. */
function cachePolicy(endpoint: string, now: number): { key: string; ttl: number } | undefined {
  const url = new URL(endpoint, 'https://aurora.invalid')
  if (url.origin !== 'https://aurora.invalid') return
  let ttl: number
  switch (url.pathname) {
    case '/api/v1/orbit/overview': ttl = 30_000; break
    case '/api/v1/orbit/spacecraft':
      if (url.searchParams.get('mode') === 'regex') return
      ttl = 30_000; break
    case '/api/v1/moon/spacecraft':
    case '/api/v1/mars/spacecraft':
    case '/api/v1/voyage/probes':
    case '/api/v1/astronomy/events':
    case '/api/v1/astronomy/image-wall': ttl = 60_000; break
    case '/api/v1/astronomy/conditions':
      if (url.searchParams.has('time')) return
      ttl = 60_000; break
    case '/api/v1/moon/landing-sites':
    case '/api/v1/mars/landing-sites':
    case '/api/v1/astronomy/light-pollution':
    case '/api/v1/location/reverse': ttl = 10 * 60_000; break
    case '/api/v1/location/search': ttl = 5 * 60_000; break
    default: return
  }
  url.searchParams.sort()
  // Default event ranges, forecasts and image selections depend on the current day.
  const midnight = (Math.floor(now / 86_400_000) + 1) * 86_400_000
  return { key: url.pathname + url.search, ttl: Math.min(ttl, midnight - now) }
}

export class MemoryJSONClient {
  private entries = new Map<string, CacheEntry>()
  private flights = new Map<string, Flight>()
  private bytes = 0
  private fetcher: typeof fetch
  private now: () => number
  private maxBytes: number
  private maxEntries: number

  constructor(options: ClientOptions = {}) {
    this.fetcher = options.fetcher ?? ((...args) => globalThis.fetch(...args))
    this.now = options.now ?? Date.now
    this.maxBytes = options.maxBytes ?? 5 * 1024 * 1024
    this.maxEntries = options.maxEntries ?? 64
  }

  private remove(key: string) {
    const entry = this.entries.get(key)
    if (entry) this.bytes -= entry.bytes
    this.entries.delete(key)
  }

  private remember(key: string, text: string, expiresAt: number) {
    // Budget retained JS strings conservatively at two bytes per UTF-16 code unit.
    // Parsed scene data and browser object overhead are separate from this budget.
    const bytes = (key.length + text.length) * 2
    if (bytes > this.maxBytes || this.maxEntries <= 0 || expiresAt <= this.now()) return
    this.remove(key)
    while (this.entries.size >= this.maxEntries || this.bytes + bytes > this.maxBytes) {
      const oldest = this.entries.keys().next().value
      if (oldest === undefined) break
      this.remove(oldest)
    }
    this.entries.set(key, { text, expiresAt, bytes })
    this.bytes += bytes
  }

  private async load(endpoint: string, signal?: AbortSignal): Promise<{ text: string; reusable: boolean }> {
    const response = await this.fetcher(endpoint, { signal })
    if (!response.ok) throw new Error(`接口返回 ${response.status}`)
    const text = await response.text()
    JSON.parse(text) // Malformed success responses must never poison the cache.
    const control = response.headers.get('Cache-Control')?.toLowerCase() ?? ''
    // Origin is fixed within this tab and fetch has already decoded compression.
    // Other variants (e.g. Authorization or language) cannot share an entry.
    const varies = response.headers.get('Vary')?.toLowerCase().split(',').map((name) => name.trim()).filter(Boolean) ?? []
    return { text, reusable: !/\b(no-store|private)\b/.test(control) && varies.every((name) => name === 'origin' || name === 'accept-encoding') }
  }

  async request<T>(endpoint: string, signal?: AbortSignal): Promise<T> {
    if (signal?.aborted) throw abortReason(signal)
    const now = this.now()
    // Bounded lazy cleanup; no expiry timers or periodic background work.
    for (const [key, entry] of this.entries) if (entry.expiresAt <= now) this.remove(key)
    const policy = cachePolicy(endpoint, now)
    if (!policy) return JSON.parse((await this.load(endpoint, signal)).text) as T
    const cached = this.entries.get(policy.key)
    if (cached) {
      this.entries.delete(policy.key)
      this.entries.set(policy.key, cached)
      return JSON.parse(cached.text) as T // Callers cannot mutate another scene's data.
    }

    let flight = this.flights.get(policy.key)
    if (!flight) {
      const controller = new AbortController()
      const next: Flight = { controller, promise: Promise.resolve(''), waiters: 0, settled: false }
      next.promise = this.load(endpoint, controller.signal).then(({ text, reusable }) => {
        if (controller.signal.aborted) throw abortReason(controller.signal)
        if (reusable) this.remember(policy.key, text, now + policy.ttl)
        return text
      }).finally(() => {
        next.settled = true
        if (this.flights.get(policy.key) === next) this.flights.delete(policy.key)
      })
      this.flights.set(policy.key, next)
      flight = next
    }
    return this.subscribe<T>(policy.key, flight, signal)
  }

  private subscribe<T>(key: string, flight: Flight, signal?: AbortSignal): Promise<T> {
    return new Promise<T>((resolve, reject) => {
      flight.waiters++
      let completed = false
      const release = () => {
        if (completed) return false
        completed = true
        signal?.removeEventListener('abort', abort)
        flight.waiters--
        if (flight.waiters === 0 && !flight.settled) {
          if (this.flights.get(key) === flight) this.flights.delete(key)
          flight.controller.abort()
        }
        return true
      }
      const abort = () => { if (release()) reject(abortReason(signal)) }
      signal?.addEventListener('abort', abort, { once: true })
      flight.promise.then((text) => {
        if (!release()) return
        try { resolve(JSON.parse(text) as T) } catch (reason) { reject(reason) }
      }, (reason: unknown) => { if (release()) reject(reason) })
    })
  }
}
