import test from 'node:test'
import assert from 'node:assert/strict'
import { MemoryJSONClient } from '../src/dataCache.ts'

const moon = '/api/v1/moon/spacecraft'
const mars = '/api/v1/mars/spacecraft'
const site = '/api/v1/moon/landing-sites'
const epoch = Date.parse('2026-11-05T04:00:00Z')
const json = (value, headers) => new Response(JSON.stringify(value), { headers: { 'Content-Type': 'application/json', ...headers } })
const deferred = () => { let resolve; const promise = new Promise((r) => { resolve = r }); return { promise, resolve } }

test('re-entering a fresh page avoids network; expired data is fetched again without changing source time', async () => {
  let now = epoch, calls = 0
  const client = new MemoryJSONClient({ now: () => now, fetcher: async () => json({ revision: ++calls, syncedAt: 'original-source-time' }) })
  assert.equal((await client.request(moon)).revision, 1)
  now += 59_000
  assert.equal((await client.request(moon)).revision, 1)
  now += 1_001
  assert.deepEqual(await client.request(moon), { revision: 2, syncedAt: 'original-source-time' })
})

test('cached data is private to the tab and mutations do not leak between consumers', async () => {
  let calls = 0
  const options = { now: () => epoch, fetcher: async () => { calls++; return json({ spacecraft: [{ name: 'original' }] }) } }
  const client = new MemoryJSONClient(options)
  const first = await client.request(moon)
  first.spacecraft[0].name = 'changed by scene'
  assert.equal((await client.request(moon)).spacecraft[0].name, 'original')
  await new MemoryJSONClient(options).request(moon)
  assert.equal(calls, 2)
})

test('concurrent consumers share one request; cancelling one does not cancel the other', async () => {
  let calls = 0, networkSignal
  const pending = deferred()
  const client = new MemoryJSONClient({ fetcher: async (_url, options) => { calls++; networkSignal = options.signal; return pending.promise } })
  const a = new AbortController(), b = new AbortController()
  const first = client.request(moon, a.signal)
  const second = client.request(moon, b.signal)
  a.abort()
  await assert.rejects(first, { name: 'AbortError' })
  assert.equal(networkSignal.aborted, false)
  pending.resolve(json({ ok: true }))
  assert.deepEqual(await second, { ok: true })
  assert.equal(calls, 1)
})

test('last consumer leaving cancels the network; a late response cannot replace the next request', async () => {
  const old = deferred()
  let calls = 0, oldSignal
  const client = new MemoryJSONClient({ fetcher: async (_url, options) => {
    calls++
    if (calls === 1) { oldSignal = options.signal; return old.promise }
    return json({ revision: 2 })
  } })
  const controller = new AbortController()
  const first = client.request(moon, controller.signal)
  controller.abort()
  await assert.rejects(first, { name: 'AbortError' })
  assert.equal(oldSignal.aborted, true)
  assert.equal((await client.request(moon)).revision, 2)
  old.resolve(json({ revision: 1 }))
  await new Promise((resolve) => setImmediate(resolve))
  assert.equal((await client.request(moon)).revision, 2)
  assert.equal(calls, 2)
})

test('failed HTTP and malformed JSON responses are retried instead of cached', async () => {
  let calls = 0
  const client = new MemoryJSONClient({ fetcher: async () => {
    calls++
    if (calls === 1) return new Response('unavailable', { status: 502 })
    if (calls === 2) return new Response('broken json')
    return json({ recovered: true })
  } })
  await assert.rejects(client.request(moon), /502/)
  await assert.rejects(client.request(moon), SyntaxError)
  assert.deepEqual(await client.request(moon), { recovered: true })
  assert.deepEqual(await client.request(moon), { recovered: true })
  assert.equal(calls, 3)
})

test('LRU retains frequently viewed pages while enforcing entry and string-data budgets', async () => {
  let calls = 0
  const client = new MemoryJSONClient({ maxEntries: 2, fetcher: async () => json({ revision: ++calls }) })
  await client.request(moon); await client.request(mars); await client.request(moon)
  await client.request(site); await client.request(moon)
  assert.equal(calls, 3)
  await client.request(mars)
  assert.equal(calls, 4)
  let largeCalls = 0
  const bounded = new MemoryJSONClient({ maxBytes: 100, fetcher: async () => { largeCalls++; return json({ data: 'x'.repeat(100) }) } })
  await bounded.request(moon); await bounded.request(moon)
  assert.equal(largeCalls, 2, 'a response larger than the budget is not retained')
  let budgetCalls = 0
  const small = new MemoryJSONClient({ maxBytes: 300, fetcher: async () => { budgetCalls++; return json({ data: 'x'.repeat(80) }) } })
  await small.request(moon); await small.request(moon)
  assert.equal(budgetCalls, 1, 'an individual response fits in the budget')
  await small.request(mars); await small.request(moon)
  assert.equal(budgetCalls, 3, 'combined responses exceeding the budget evict the oldest')
})

test('different locations, elevations, dates and timezones cannot reuse each other’s data', async () => {
  let calls = 0
  const client = new MemoryJSONClient({ fetcher: async () => json({ revision: ++calls }) })
  const query = '/api/v1/astronomy/events?latitude=31.23&longitude=121.47&elevation=10&timezone=Asia%2FShanghai&from=2026-11-05'
  await client.request(query)
  await client.request('/api/v1/astronomy/events?from=2026-11-05&timezone=Asia%2FShanghai&elevation=10&longitude=121.47&latitude=31.23')
  assert.equal(calls, 1, 'query order does not affect reuse')
  for (const changed of [query.replace('31.23', '31.24'), query.replace('elevation=10', 'elevation=20'), query.replace('Asia%2FShanghai', 'UTC'), query.replace('2026-11-05', '2026-11-06')]) await client.request(changed)
  assert.equal(calls, 5)
})

test('real-time scores, time-selected weather and regular-expression searches bypass the cache', async () => {
  let calls = 0
  const client = new MemoryJSONClient({ fetcher: async () => json({ revision: ++calls }) })
  for (const url of ['/api/v1/astronomy/score?at=123', '/api/v1/astronomy/moon?at=123', '/api/v1/astronomy/conditions?time=123', '/api/v1/orbit/spacecraft?mode=regex']) {
    await client.request(url); await client.request(url)
  }
  assert.equal(calls, 8)
})

test('local data cannot survive an implicit-date midnight boundary or an already cancelled request', async () => {
  let now = Date.parse('2026-11-05T23:59:59.500Z'), calls = 0
  const client = new MemoryJSONClient({ now: () => now, fetcher: async () => json({ revision: ++calls }) })
  await client.request('/api/v1/astronomy/image-wall')
  now += 501
  assert.equal((await client.request('/api/v1/astronomy/image-wall')).revision, 2)
  const cancelled = new AbortController(); cancelled.abort()
  await assert.rejects(client.request('/api/v1/astronomy/image-wall', cancelled.signal), { name: 'AbortError' })
  assert.equal(calls, 2)
})

test('no-store, private and varying responses are not retained', async () => {
  for (const headers of [{ 'Cache-Control': 'no-store' }, { 'Cache-Control': 'private' }, { Vary: 'Authorization' }]) {
    let calls = 0
    const client = new MemoryJSONClient({ fetcher: async () => { calls++; return json({ ok: true }, headers) } })
    await client.request(moon); await client.request(moon)
    assert.equal(calls, 2)
  }
})

test('same-origin and compression variants can reuse already decoded JSON within the tab', async () => {
  let calls = 0
  const client = new MemoryJSONClient({ fetcher: async () => { calls++; return json({ ok: true }, { Vary: 'Origin, Accept-Encoding' }) } })
  await client.request(moon); await client.request(moon)
  assert.equal(calls, 1)
})
