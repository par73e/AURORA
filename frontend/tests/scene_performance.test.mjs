import { test } from 'node:test'
import assert from 'node:assert/strict'
import { ScenePerformance, scenePerformanceSnapshot } from '../src/performance/scenePerformance.ts'

test('scene monitor applies sustained resolution changes, resets on hide, and cleans up on disposal', () => {
  const previous = Object.fromEntries(['document', 'window', 'location', 'performance'].map(k => [k, globalThis[k]]))
  let now = 0
  const documentEvents = new Map(), canvasEvents = new Map()
  const canvas = { addEventListener: (k,v) => canvasEvents.set(k,v), removeEventListener: k => canvasEvents.delete(k) }
  globalThis.performance = { now: () => now }
  globalThis.location = { search: '?perf=1' }
  globalThis.window = {}
  globalThis.document = { hidden: false, addEventListener: (k,v) => documentEvents.set(k,v), removeEventListener: k => documentEvents.delete(k) }
  let dpr = 2
  const changes = []
  const renderer = { domElement: canvas, getPixelRatio: () => dpr, setPixelRatio: v => { dpr=v; changes.push(v) }, info: { render: { calls: 5, triangles: 20 }, memory: { geometries: 2, textures: 3 } } }
  const monitor = new ScenePerformance('test-scene', renderer)
  const frame = (dt) => { now += dt; monitor.beginFrame(); now += 2; monitor.endFrame() }
  try {
    for (let i=0; i<180; i++) frame(30)
    assert.ok(changes.length > 0)
    assert.ok(dpr < 2 && dpr >= 1)
    assert.equal(window.__auroraPerformance().active.filter(s=>s.scene==='test-scene').length, 1)
    const metrics = monitor.snapshot().windows.at(-1)
    assert.equal(metrics.drawCalls, 5)
    assert.equal(metrics.workMeanMs, 2)
    assert.ok(metrics.frameMeanMs >= 30)
    document.hidden = true; documentEvents.get('visibilitychange')(); now += 60000
    frame(30000)
    document.hidden = false; documentEvents.get('visibilitychange')()
    for (let i=0; i<70; i++) frame(14.67)
    assert.ok(monitor.snapshot().windows.at(-1).frameMeanMs < 18)
    canvasEvents.get('pointerdown')()
    for (let i=0; i<65; i++) frame(14.67)
    assert.equal(monitor.snapshot().windows.at(-1).activity, 'interaction')
    const beforePause = monitor.snapshot()
    monitor.pause()
    for (let i=0; i<100; i++) frame(1000)
    assert.equal(monitor.snapshot().paused, true)
    assert.deepEqual(monitor.snapshot().windows, beforePause.windows)
    monitor.resume()
    for (let i=0; i<70; i++) frame(14.67)
    assert.equal(monitor.snapshot().id, beforePause.id)
    assert.equal(monitor.snapshot().paused, false)
    assert.ok(monitor.snapshot().windows.at(-1).frameMeanMs < 18)
    monitor.dispose(); monitor.dispose()
    assert.equal(canvasEvents.size, 0)
    assert.equal(documentEvents.size, 0)
    assert.equal(scenePerformanceSnapshot().active.length, 0)
    assert.equal(scenePerformanceSnapshot().completed.filter(s=>s.scene==='test-scene').length, 1)
  } finally {
    monitor.dispose()
    for (const [k,v] of Object.entries(previous)) { if (v===undefined) delete globalThis[k]; else globalThis[k]=v }
  }
})
