import { spacecraftPoint, sampleOrbit } from './coordinates'
import type { Spacecraft } from '../types'

let crafts: Spacecraft[] = []
let generation = 0
let paused = false
let timer: ReturnType<typeof setTimeout> | undefined
let lastTracks = -Infinity
const scope = self as unknown as { postMessage(message: unknown, transfer: Transferable[]): void; onmessage: ((event: MessageEvent) => void) | null }
function tick() {
  if (paused) return
  const start = Date.now(), end = start + 1000
  const positions = new Float64Array(crafts.length * 6)
  positions.fill(NaN)
  const tracks: { id: string; points: Float32Array }[] = []
  crafts.forEach((craft, index) => {
    const a = spacecraftPoint(craft, new Date(start)), b = spacecraftPoint(craft, new Date(end))
    if (a && b) { a.position.toArray(positions, index * 6); b.position.toArray(positions, index * 6 + 3) }
    if (start - lastTracks >= 60000 && (craft.category?.startsWith('LEO') || craft.category?.startsWith('SSO'))) {
      const orbit = sampleOrbit(craft, new Date(start))
      const points = new Float32Array(orbit.length * 3)
      orbit.forEach((p, i) => p.toArray(points, i * 3))
      tracks.push({ id: craft.id, points })
    }
  })
  lastTracks = start - lastTracks >= 60000 ? start : lastTracks
  scope.postMessage({ generation, ids: crafts.map(c => c.id), start, end, positions, tracks }, [positions.buffer, ...tracks.map(t => t.points.buffer)])
  timer = setTimeout(tick, 1000)
}
scope.onmessage = ({ data }) => {
  clearTimeout(timer)
  if (data.type === 'init') { crafts = data.crafts; generation = data.generation; lastTracks = -Infinity }
  else if (data.type === 'pause') paused = data.paused
  if (!paused) tick()
}
