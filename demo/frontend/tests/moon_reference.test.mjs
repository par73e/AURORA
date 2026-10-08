import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import { Body, Equator, Horizon, MoonPhase, Observer, SearchRiseSet } from 'astronomy-engine'

const fixturePath = path.join(path.dirname(fileURLToPath(import.meta.url)), '..', '..', 'backend', 'internal', 'observatory', 'testdata', 'moon_reference.json')
const fixture = JSON.parse(readFileSync(fixturePath, 'utf8'))

const MEAN_SYNODIC = 29.530588853
const TOLERANCE = { phase: 1, illumination: 0.02, age: 0.15, riseSetMinutes: 15, transitMinutes: 15 }

function illuminationFromPhase(deg) {
  return (1 - Math.cos(deg * Math.PI / 180)) / 2
}

function moonLabel(phase) {
  if (phase < 18 || phase >= 342) return '朔月'
  if (phase < 72) return '娥眉月'
  if (phase < 108) return '上弦月'
  if (phase < 162) return '盈凸月'
  if (phase < 198) return '满月'
  if (phase < 252) return '亏凸月'
  if (phase < 288) return '下弦月'
  return '残月'
}

// 与前端 SkyObservatory.vue 的 transitFor 一致：5 分钟采样取最高高度角。
function transitFor(latitude, longitude, dayStartEpoch) {
  const place = new Observer(latitude, longitude, 0)
  let best = null
  let bestAltitude = -Infinity
  for (let minute = 0; minute < 1440; minute += 5) {
    const at = new Date((dayStartEpoch + minute * 60) * 1000)
    const eq = Equator(Body.Moon, at, place, true, true)
    const altitude = Horizon(at, place, eq.ra, eq.dec, 'normal').altitude
    if (altitude > bestAltitude) {
      bestAltitude = altitude
      best = dayStartEpoch + minute * 60
    }
  }
  return best
}

function assertNear(actual, expected, toleranceSeconds, label) {
  if (actual == null && expected == null) return
  if (actual == null || expected == null) {
    assert.fail(`${label}: 一方缺失 go=${expected} ae=${actual}`)
  }
  const diff = Math.abs(actual - expected)
  assert.ok(
    diff <= toleranceSeconds,
    `${label}: 差 ${Math.round(diff / 60)} 分钟（go=${new Date(expected * 1000).toISOString()}，ae=${new Date(actual * 1000).toISOString()}）`,
  )
}

// 与 Go 端相同的窗口语义：只比较本地日 [dayStart, dayStart+24h) 内的升落。
// AE 的 SearchRiseSet 带 limitDays=1.2 会越过窗口找到次日的升落，需排除后再比较。
function inLocalDay(epoch, dayStart) {
  return epoch != null && epoch >= dayStart - 300 && epoch <= dayStart + 86400 + 300
}

test('Go 端月相基准值与 astronomy-engine 交叉校验', () => {
  for (const c of fixture.cases) {
    const at = new Date(c.at * 1000)
    const aePhase = MoonPhase(at)
    const aeIllumination = illuminationFromPhase(aePhase)
    const aeAge = aePhase / 360 * MEAN_SYNODIC

    assert.ok(
      Math.abs(aePhase - c.phase) <= TOLERANCE.phase,
      `${c.name}: 相位 go=${c.phase.toFixed(2)}° ae=${aePhase.toFixed(2)}°`,
    )
    assert.ok(
      Math.abs(aeIllumination - c.illumination) <= TOLERANCE.illumination,
      `${c.name}: 亮面占比 go=${c.illumination.toFixed(4)} ae=${aeIllumination.toFixed(4)}`,
    )
    assert.ok(
      Math.abs(aeAge - c.age) <= TOLERANCE.age,
      `${c.name}: 月龄 go=${c.age.toFixed(3)} ae=${aeAge.toFixed(3)}`,
    )
    assert.equal(moonLabel(aePhase), c.label, `${c.name}: 月相名`)

    const place = new Observer(c.latitude, c.longitude, 0)
    const dayStart = new Date(c.dayStart * 1000)
    const rise = SearchRiseSet(Body.Moon, place, 1, dayStart, 1.2)?.date.getTime() / 1000 ?? null
    const set = SearchRiseSet(Body.Moon, place, -1, dayStart, 1.2)?.date.getTime() / 1000 ?? null
    const transit = transitFor(c.latitude, c.longitude, c.dayStart)

    assertNear(inLocalDay(rise, c.dayStart) ? rise : null, c.moonrise, TOLERANCE.riseSetMinutes * 60, `${c.name}: 月出`)
    assertNear(inLocalDay(set, c.dayStart) ? set : null, c.moonset, TOLERANCE.riseSetMinutes * 60, `${c.name}: 月落`)
    assertNear(transit, c.transit, TOLERANCE.transitMinutes * 60, `${c.name}: 中天`)
  }
})
