import type { Spacecraft } from '../types'
import type { Vector3 } from 'three'
import { positionMix } from './positionInterpolation.ts'

interface Snapshot {
  generation: number; ids: string[]; start: number; end: number; positions: Float64Array
  tracks: { id: string; points: Float32Array }[]
}
export class OrbitPositions {
  private worker?: Worker
  private snapshot?: Snapshot
  private indices = new Map<string, number>()
  private generation = 0
  status() { return { worker: this.available, generation: this.generation, snapshotTime: this.snapshot?.start ?? null, count: this.indices.size } }
  get available() { return !!this.worker }
  constructor(private readonly onTracks: (tracks: Snapshot['tracks']) => void, private readonly onUnavailable?: () => void) {
    try {
      this.worker = new Worker(new URL('./positions.worker.ts', import.meta.url), { type: 'module' })
      this.worker.onmessage = ({ data }: MessageEvent<Snapshot>) => {
        if (data.generation !== this.generation) return
        this.snapshot = data
        this.indices = new Map(data.ids.map((id, i) => [id, i]))
        this.onTracks(data.tracks)
      }
      this.worker.onerror = () => { this.worker?.terminate(); this.worker = undefined; this.snapshot = undefined; this.onUnavailable?.() }
      document.addEventListener('visibilitychange', this.visibility)
    } catch { this.worker = undefined }
  }
  private visibility = () => { this.worker?.postMessage({ type: 'pause', paused: document.hidden }) }
  setCrafts(crafts: Spacecraft[]) {
    this.generation++; this.snapshot = undefined; this.indices.clear()
    this.worker?.postMessage({ type: 'init', generation: this.generation, crafts: JSON.parse(JSON.stringify(crafts)) })
    this.visibility()
  }
  position(id: string, target: Vector3, now: number) {
    const s = this.snapshot, index = this.indices.get(id)
    if (!s || index === undefined) return false
    const mix = positionMix(now, s.start, s.end)
    if (mix === null) return false
    const p = index * 6, a = s.positions
    if (!Number.isFinite(a[p]) || !Number.isFinite(a[p + 3])) return false
    target.set(a[p]! + (a[p + 3]! - a[p]!) * mix, a[p + 1]! + (a[p + 4]! - a[p + 1]!) * mix, a[p + 2]! + (a[p + 5]! - a[p + 2]!) * mix)
    return true
  }
  dispose() { document.removeEventListener('visibilitychange', this.visibility); this.worker?.terminate(); this.worker = undefined; this.snapshot = undefined }
}
