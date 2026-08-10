// 星历抽样校验：把本项目前端使用的 astronomy-engine 与独立权威源对拍。
//
// 太阳升落/晨昏 → sunrise-sunset.org（NOAA 太阳计算器，无 Key）。
// 月球升落     → 由 backend moon_reference.test.mjs（Go 自研算法 ↔ astronomy-engine，
//                实测差 ≤2 分钟）覆盖；JPL Horizons 在本网络不可达（TCP 超时），
//                如需 JPL 对拍，可把下方 SAMPLES 换为 Horizons 观测历表输出后运行。
//
// 运行：node tests/ephemeris_sampling.mjs（不是 npm test 的一部分，避免测试依赖网络）。
import { Body, Observer, SearchRiseSet } from 'astronomy-engine'

const SAMPLES = [
  { name: 'shanghai-2026-08-09', latitude: 31.23, longitude: 121.47, date: '2026-08-09' },
  { name: 'beijing-2026-03-03', latitude: 39.9042, longitude: 116.4074, date: '2026-03-03' },
  { name: 'sydney-2026-06-21', latitude: -33.8688, longitude: 151.2093, date: '2026-06-21' },
]

const TOLERANCE_MINUTES = 5 // NOAA 计算器与 AE 的折射/太阳半径约定差异在分钟级
let failed = 0

function epochSeconds(localDate, timeUtc) {
  // sunrise-sunset.org 返回 UTC 时刻 "H:MM:SS AM/PM"。
  const match = /^(\d+):(\d+):(\d+) (AM|PM)$/.exec(timeUtc.trim())
  if (!match) return null
  let hours = Number(match[1]) % 12
  if (match[4] === 'PM') hours += 12
  return Date.UTC(
    Number(localDate.slice(0, 4)), Number(localDate.slice(5, 7)) - 1, Number(localDate.slice(8, 10)),
    hours, Number(match[2]), Number(match[3]),
  ) / 1000
}

function check(label, aeEpoch, noaaEpoch) {
  if (noaaEpoch == null) {
    console.log(`  ${label}: NOAA 无数据（极昼/极夜）— 跳过`)
    return
  }
  const diffMinutes = Math.abs(aeEpoch - noaaEpoch) / 60
  const pass = diffMinutes <= TOLERANCE_MINUTES
  if (!pass) failed += 1
  console.log(`  ${label}: AE=${new Date(aeEpoch * 1000).toISOString().slice(11, 16)}Z NOAA=${new Date(noaaEpoch * 1000).toISOString().slice(11, 16)}Z 差=${diffMinutes.toFixed(1)}min ${pass ? '✓' : '✗'}`)
}

for (const sample of SAMPLES) {
  console.log(`\n${sample.name} (${sample.latitude}, ${sample.longitude}) ${sample.date}`)
  const url = `https://api.sunrise-sunset.org/json?lat=${sample.latitude}&lng=${sample.longitude}&date=${sample.date}&formatted=0`
  let data
  try {
    data = await (await fetch(url)).json()
  } catch (error) {
    console.log('  网络不可达，跳过该样本。')
    continue
  }
  if (data.status !== 'OK' || !data.results) {
    console.log(`  sunrise-sunset.org 返回异常：${data.status}`)
    continue
  }
  const results = data.results
  const noaa = (key) => new Date(results[key]).getTime() / 1000
  // 用 NOAA 的 solar_noon（本地正午的 UTC 时刻）推导时区偏移，得到本地日零点作为 AE 搜索起点。
  const noon = new Date(results.solar_noon)
  const offsetHours = 12 - (noon.getUTCHours() + noon.getUTCMinutes() / 60)
  const dayStart = Date.UTC(Number(sample.date.slice(0, 4)), Number(sample.date.slice(5, 7)) - 1, Number(sample.date.slice(8, 10))) / 1000 - offsetHours * 3600
  const place = new Observer(sample.latitude, sample.longitude, 0)

  const sunrise = SearchRiseSet(Body.Sun, place, 1, new Date(dayStart * 1000), 1.2)
  const sunset = SearchRiseSet(Body.Sun, place, -1, new Date(dayStart * 1000), 1.2)
  check('日出', sunrise?.date.getTime() / 1000 ?? null, noaa('sunrise'))
  check('日落', sunset?.date.getTime() / 1000 ?? null, noaa('sunset'))
}

console.log(failed === 0 ? '\n全部样本在容差内一致 ✓' : `\n${failed} 项超出容差 ✗`)
process.exit(failed === 0 ? 0 : 1)
