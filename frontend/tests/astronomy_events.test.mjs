import { test } from 'node:test'
import assert from 'node:assert/strict'

// 端到端验证 fetchAstronomyEvents 的 URL 构造与 5 个场景的响应处理逻辑。
// 通过全局 fetch mock 捕获请求 URL，验证前端只请求 AURORA API，
// 不在页面打开时请求 JPL/USNO/NASA/IMO。

const ORIGINAL_FETCH = globalThis.fetch

function mockFetch(responses) {
  const calls = []
  globalThis.fetch = (endpoint, options) => {
    calls.push({ endpoint, signal: options?.signal })
    const match = responses.find(r => r.endpoint === endpoint || r.match?.(endpoint))
    if (!match) return Promise.resolve({ ok: false, status: 404, json: () => Promise.reject(new Error('not found')) })
    return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(match.body) })
  }
  return calls
}

function restoreFetch() {
  globalThis.fetch = ORIGINAL_FETCH
}

// 动态 import 前端 api.ts，确认 fetchAstronomyEvents 存在并构造正确的 URL
test('fetchAstronomyEvents 构造正确的 AURORA API URL，不请求 JPL/USNO/NASA/IMO', async () => {
  const api = await import('../src/api.ts')
  const calls = mockFetch([
    {
      match: e => e.startsWith('/api/v1/astronomy/events'),
      body: { events: [], range: { from: '2026-08-12', to: '2026-09-12' }, locationVisibility: 'location_required' },
    },
  ])
  try {
    await api.fetchAstronomyEvents({})
    assert.ok(calls.length === 1, `只应请求一次 AURORA API，实际 ${calls.length} 次`)
    assert.ok(calls[0].endpoint.startsWith('/api/v1/astronomy/events'), `URL 应以 /api/v1/astronomy/events 开头，实际 ${calls[0].endpoint}`)
    // 绝不请求 JPL/USNO/NASA/IMO
    for (const call of calls) {
      assert.ok(!call.endpoint.includes('jpl.nasa.gov'), `不应请求 JPL: ${call.endpoint}`)
      assert.ok(!call.endpoint.includes('usno.navy.mil'), `不应请求 USNO: ${call.endpoint}`)
      assert.ok(!call.endpoint.includes('eclipse.gsfc.nasa.gov'), `不应请求 NASA/GSFC: ${call.endpoint}`)
      assert.ok(!call.endpoint.includes('imo.net'), `不应请求 IMO: ${call.endpoint}`)
    }
  } finally {
    restoreFetch()
  }
})

test('fetchImageWall 只请求 AURORA 聚合接口，浏览器不直连 NASA', async () => {
  const api = await import('../src/api.ts')
  const calls = mockFetch([
    { match: e => e === '/api/v1/astronomy/image-wall', body: { recent: [{ id: 'apod', sourceId: 'apod', sourceName: 'NASA Astronomy Picture of the Day', title: 'Perseids', mediaType: 'image', credit: 'NASA', sourceUrl: 'https://apod.nasa.gov/apod/astropix.html', selectionMode: 'daily', status: 'ready' }], collection: [], generatedAt: '2026-08-12T00:00:00Z' } },
  ])
  try {
    const wall = await api.fetchImageWall()
    assert.equal(wall.recent[0].title, 'Perseids')
    assert.equal(calls.length, 1)
    assert.equal(calls[0].endpoint, '/api/v1/astronomy/image-wall')
    assert.ok(!calls[0].endpoint.includes('api.nasa.gov'), `浏览器不应直接请求 NASA API: ${calls[0].endpoint}`)
  } finally {
    restoreFetch()
  }
})

// 场景 1: 上海（31.2304, 121.4737）——返回带 local 字段的事件，locationVisibility=partial
test('场景上海：返回带 local 字段的事件，locationVisibility=partial', async () => {
  const api = await import('../src/api.ts')
  mockFetch([
    {
      match: e => e.startsWith('/api/v1/astronomy/events'),
      body: {
        events: [
          { id: 'full_moon-20260924', kind: 'full_moon', title: '满月', titleEn: 'FULL MOON', startsAt: '2026-09-24T00:00:00Z', dateLabel: '2026年9月24日', summary: '满月', origin: 'computed', sourceName: 'AURORA', sourceUrl: 'https://aurora.local', verifiedAt: '2026-08-12', geometry: {}, local: { status: 'observable', bestAt: '2026-09-24T01:30:00+08:00', azimuthDegrees: 180, altitudeDegrees: 45, reason: '适合观测' } },
        ],
        range: { from: '2026-08-12', to: '2026-09-12' },
        locationVisibility: 'partial',
      },
    },
  ])
  try {
    const response = await api.fetchAstronomyEvents({ latitude: 31.2304, longitude: 121.4737, timezone: 'Asia/Shanghai' })
    assert.equal(response.locationVisibility, 'partial')
    assert.ok(response.events.length > 0)
    assert.ok(response.events[0].local, '上海场景事件应有 local 字段')
    assert.equal(response.events[0].local.status, 'observable')
  } finally {
    restoreFetch()
  }
})

// 场景 2: 北半球高纬（64.1466, -21.9426）——验证 latitude/longitude 正确传入
test('场景北半球高纬：latitude/longitude 正确传入 URL', async () => {
  const api = await import('../src/api.ts')
  const calls = mockFetch([
    {
      match: e => e.startsWith('/api/v1/astronomy/events'),
      body: { events: [], range: { from: '2026-08-12', to: '2026-09-12' }, locationVisibility: 'partial' },
    },
  ])
  try {
    await api.fetchAstronomyEvents({ latitude: 64.1466, longitude: -21.9426, timezone: 'Atlantic/Reykjavik' })
    assert.ok(calls[0].endpoint.includes('latitude=64.1466'), `URL 应含 latitude=64.1466，实际 ${calls[0].endpoint}`)
    assert.ok(calls[0].endpoint.includes('longitude=-21.9426'), `URL 应含 longitude=-21.9426，实际 ${calls[0].endpoint}`)
    assert.ok(calls[0].endpoint.includes('timezone=Atlantic%2FReykjavik') || calls[0].endpoint.includes('timezone=Atlantic/Reykjavik'), `URL 应含 timezone，实际 ${calls[0].endpoint}`)
  } finally {
    restoreFetch()
  }
})

// 场景 3: 南半球（-33.8688, 151.2093）——验证负纬度正确传入
test('场景南半球：负纬度正确传入 URL', async () => {
  const api = await import('../src/api.ts')
  const calls = mockFetch([
    {
      match: e => e.startsWith('/api/v1/astronomy/events'),
      body: { events: [], range: { from: '2026-08-12', to: '2026-09-12' }, locationVisibility: 'partial' },
    },
  ])
  try {
    await api.fetchAstronomyEvents({ latitude: -33.8688, longitude: 151.2093, timezone: 'Australia/Sydney' })
    assert.ok(calls[0].endpoint.includes('latitude=-33.8688'), `URL 应含 latitude=-33.8688，实际 ${calls[0].endpoint}`)
    assert.ok(calls[0].endpoint.includes('longitude=151.2093'), `URL 应含 longitude=151.2093，实际 ${calls[0].endpoint}`)
  } finally {
    restoreFetch()
  }
})

// 场景 4: 无可见窗口（接近北极点 89.5, 0）——事件 local.status 可为 not_visible
test('场景无可见窗口：事件 local.status 可为 not_visible', async () => {
  const api = await import('../src/api.ts')
  mockFetch([
    {
      match: e => e.startsWith('/api/v1/astronomy/events'),
      body: {
        events: [
          { id: 'full_moon-20260924', kind: 'full_moon', title: '满月', titleEn: 'FULL MOON', startsAt: '2026-09-24T00:00:00Z', dateLabel: '2026年9月24日', summary: '满月', origin: 'computed', sourceName: 'AURORA', sourceUrl: 'https://aurora.local', verifiedAt: '2026-08-12', geometry: {}, local: { status: 'not_visible', reason: '月球在地平线以下' } },
        ],
        range: { from: '2026-08-12', to: '2026-09-12' },
        locationVisibility: 'partial',
      },
    },
  ])
  try {
    const response = await api.fetchAstronomyEvents({ latitude: 89.5, longitude: 0, timezone: 'UTC' })
    assert.equal(response.events[0].local.status, 'not_visible')
    assert.ok(!response.events[0].local.bestAt, 'not_visible 事件不应有 bestAt')
  } finally {
    restoreFetch()
  }
})

// 场景 5: 无定位授权——不传坐标，返回全球日历，locationVisibility=location_required，事件无 local
test('场景无定位授权：返回全球日历，事件无 local 字段', async () => {
  const api = await import('../src/api.ts')
  const calls = mockFetch([
    {
      match: e => e.startsWith('/api/v1/astronomy/events'),
      body: {
        events: [
          { id: 'full_moon-20260924', kind: 'full_moon', title: '满月', titleEn: 'FULL MOON', startsAt: '2026-09-24T00:00:00Z', dateLabel: '2026年9月24日', summary: '满月', origin: 'computed', sourceName: 'AURORA', sourceUrl: 'https://aurora.local', verifiedAt: '2026-08-12', geometry: {} },
        ],
        range: { from: '2026-08-12', to: '2026-09-12' },
        locationVisibility: 'location_required',
      },
    },
  ])
  try {
    const response = await api.fetchAstronomyEvents({})
    assert.equal(response.locationVisibility, 'location_required')
    assert.ok(!response.events[0].local, '无定位授权时事件不应有 local 字段')
    // URL 不应含 latitude/longitude
    assert.ok(!calls[0].endpoint.includes('latitude'), `无定位授权时 URL 不应含 latitude，实际 ${calls[0].endpoint}`)
    assert.ok(!calls[0].endpoint.includes('longitude'), `无定位授权时 URL 不应含 longitude，实际 ${calls[0].endpoint}`)
  } finally {
    restoreFetch()
  }
})
