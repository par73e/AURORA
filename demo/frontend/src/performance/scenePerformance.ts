import type { WebGLRenderer } from 'three'
import { AdaptiveResolution } from './resolution.ts'

interface WindowMetrics {
  atMs: number
  activity: 'interaction' | 'idle'
  fps: number
  frameMeanMs: number
  frameP95Ms: number
  workMeanMs: number
  slowFrames: number
  longTasks: number
  longTaskMs: number
  pixelRatio: number
  drawCalls: number
  triangles: number
  geometries: number
  textures: number
}

let nextSceneId = 0
const active = new Set<ScenePerformance>()
const completed: ReturnType<ScenePerformance['snapshot']>[] = []
const round = (value: number) => Math.round(value * 100) / 100
const debug = () => new URLSearchParams(location.search).get('perf') === '1'

/** Read-only diagnostics: no telemetry is sent to a server. */
export function scenePerformanceSnapshot() {
  return { active: [...active].map(item => item.snapshot()), completed: [...completed] }
}

/** Measures frame cadence and JS/render-submission work, not GPU execution time. */
export class ScenePerformance {
  private readonly id = ++nextSceneId
  private diagnostics: (() => Record<string, unknown>) | undefined
  private readonly startedAt = performance.now()
  private readonly resolution: AdaptiveResolution
  private readonly history: WindowMetrics[] = []
  private intervals: number[] = []
  private work: number[] = []
  private lastFrame: number | null = null
  private windowAt = 0
  private workAt = 0
  private firstFrameMs: number | null = null
  private interactionUntil = 0
  private interacted = false
  private longTasks = 0
  private longTaskMs = 0
  private observer?: PerformanceObserver
  private paused = false
  private disposed = false
  private readonly adaptive = new URLSearchParams(location.search).get('adaptiveDpr') !== '0'

  private readonly name: string
  private readonly renderer: WebGLRenderer

  constructor(name: string, renderer: WebGLRenderer) {
    this.name = name
    this.renderer = renderer
    this.resolution = new AdaptiveResolution(renderer.getPixelRatio(), this.startedAt)
    active.add(this)
    document.addEventListener('visibilitychange', this.resetWindow)
    renderer.domElement.addEventListener('pointerdown', this.interaction, { passive: true })
    renderer.domElement.addEventListener('pointermove', this.drag, { passive: true })
    renderer.domElement.addEventListener('wheel', this.interaction, { passive: true })
    try {
      this.observer = new PerformanceObserver(list => {
        if (document.hidden) return
        for (const entry of list.getEntries()) {
          if (entry.startTime < this.startedAt) continue
          this.longTasks++
          this.longTaskMs += entry.duration
        }
      })
      this.observer.observe({ entryTypes: ['longtask'] })
    } catch { this.observer?.disconnect(); this.observer = undefined }
    if (debug()) {
      Object.assign(window, { __auroraPerformance: scenePerformanceSnapshot })
    }
  }

  private interaction = () => { this.interactionUntil = performance.now() + 600 }
  private drag = (event: PointerEvent) => { if (event.buttons) this.interaction() }
  private resetWindow = () => {
    this.lastFrame = null
    this.windowAt = 0
    this.intervals = []
    this.work = []
    this.longTasks = this.longTaskMs = 0
    this.interacted = false
    this.resolution.resetStreaks()
  }

  beginFrame() {
    this.workAt = performance.now()
    if (document.hidden || this.paused || this.disposed) return
    if (this.firstFrameMs === null) this.firstFrameMs = round(this.workAt - this.startedAt)
    if (this.lastFrame !== null) this.intervals.push(this.workAt - this.lastFrame)
    this.lastFrame = this.workAt
    if (!this.windowAt) this.windowAt = this.workAt
    this.interacted ||= this.workAt < this.interactionUntil
  }

  endFrame() {
    if (document.hidden || this.paused || this.disposed) return
    const now = performance.now()
    this.work.push(now - this.workAt)
    if (now - this.windowAt < 1000 || this.intervals.length < 10) return
    const sorted = [...this.intervals].sort((a, b) => a - b)
    const mean = this.intervals.reduce((a, b) => a + b, 0) / this.intervals.length
    const p95 = sorted[Math.min(sorted.length - 1, Math.floor(sorted.length * 0.95))]!
    const info = this.renderer.info
    const metrics: WindowMetrics = {
      atMs: round(now - this.startedAt), activity: this.interacted ? 'interaction' : 'idle',
      fps: round(1000 / mean), frameMeanMs: round(mean), frameP95Ms: round(p95),
      workMeanMs: round(this.work.reduce((a, b) => a + b, 0) / this.work.length),
      slowFrames: this.intervals.filter(ms => ms > 1000 / 30).length,
      longTasks: this.longTasks, longTaskMs: round(this.longTaskMs),
      pixelRatio: this.renderer.getPixelRatio(), drawCalls: info.render.calls,
      triangles: info.render.triangles, geometries: info.memory.geometries, textures: info.memory.textures,
    }
    this.history.push(metrics)
    if (this.history.length > 60) this.history.shift()
    // setPixelRatio keeps CSS size/camera geometry intact; it changes only drawing-buffer resolution.
    if (this.adaptive && this.resolution.observe(mean, p95, now)) {
      this.renderer.setPixelRatio(this.resolution.ratio)
    }
    this.intervals = []; this.work = []; this.windowAt = now
    this.longTasks = this.longTaskMs = 0; this.interacted = false
  }

  setDiagnostics(read: () => Record<string, unknown>) { this.diagnostics = read }

  pause() { this.paused = true; this.resetWindow() }
  resume() { this.paused = false; this.resetWindow() }

  snapshot() {
    return { id: this.id, scene: this.name, details: this.diagnostics?.() ?? {}, adaptive: this.adaptive, disposed: this.disposed, paused: this.paused,
      firstAnimationFrameMs: this.firstFrameMs, longTasksSupported: !!this.observer,
      maximumPixelRatio: this.resolution.maximum, minimumPixelRatio: this.resolution.minimum,
      currentPixelRatio: this.renderer.getPixelRatio(), windows: [...this.history] }
  }

  dispose() {
    if (this.disposed) return
    this.disposed = true
    this.observer?.disconnect()
    document.removeEventListener('visibilitychange', this.resetWindow)
    this.renderer.domElement.removeEventListener('pointerdown', this.interaction)
    this.renderer.domElement.removeEventListener('pointermove', this.drag)
    this.renderer.domElement.removeEventListener('wheel', this.interaction)
    active.delete(this)
    completed.push(this.snapshot())
    if (completed.length > 12) completed.shift()
  }
}
