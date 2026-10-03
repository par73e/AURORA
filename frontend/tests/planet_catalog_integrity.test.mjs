// 行星特写页档案数据的完整性守卫。
//
// 这一组断言针对的是「类型系统管不到、只能靠数据本身成立」的约定，
// 用来防止以下几类已经真实发生过的回归：
//   - 0ea0f65：改朱诺号描述时顺手删掉了 `periodDays: 53`，标记静默退化成默认速度；
//   - 新增档案条目忘了给轨迹，目录里能搜到、场景里却没有任何实体（点进去没有运镜目标）；
//   - `orbitCatalogId` 写了一个后端 ORBIT 目录里不存在的卫星 id，跳过去落空；
//   - 着陆点的 kind/icon 配错，导致图例与图标不一致。
//
// 之所以能直接 import 真实数据：见 tests/helpers/asset-stub-hooks.mjs。
import test from 'node:test'
import assert from 'node:assert/strict'
import { register } from 'node:module'
import { readdir, readFile } from 'node:fs/promises'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { join } from 'node:path'

register('./helpers/asset-stub-hooks.mjs', import.meta.url)

const {
  OUTER_PLANET_PAGES,
  VENUS_PAGE,
} = await import('../src/planetPages.ts')

const PAGES = { ...OUTER_PLANET_PAGES, venus: VENUS_PAGE }

/** 收集所有页面的飞行器档案，附带所属页面 key 便于定位失败用例。 */
function allCrafts() {
  const out = []
  for (const [page, config] of Object.entries(PAGES)) {
    for (const craft of config.spacecraft?.items ?? []) out.push({ page, craft })
  }
  return out
}

/** 收集所有页面的着陆点/任务终点档案。 */
function allSites() {
  const out = []
  for (const [page, config] of Object.entries(PAGES)) {
    for (const site of config.exploration?.sites ?? []) out.push({ page, site })
  }
  return out
}

const CRAFTS = allCrafts()
const SITES = allSites()

/**
 * 刻意不给场景实体的档案：日地 L1 / ~1 AU 轨道在太阳页尺度下无法有意义地表示，
 * 按 PRODUCT.md 的「数据诚实」原则宁可不画，也不画一条假轨道。
 * 这些条目在目录里显示「仅档案 · 场景无标记」，点击不触发运镜。
 *
 * 新增此类条目必须显式加到这里 —— 这一步就是审查点。
 */
const ARCHIVE_ONLY = new Set(['soho', 'stereo-a', 'aditya-l1'])

test('环绕轨道的 periodDays 必须是正的有限值（0ea0f65 回归守卫）', () => {
  // 注意：PlanetCraftTrajectory 是判别联合（kind 为判别键），因此
  //   - `kind: 'orbit'` 缺 periodDays  -> TS2322（Property 'periodDays' is missing）
  //   - `kind: 'flyby'` 多写 periodDays -> TS2353（Object literal may only specify known properties）
  // 两者在 vue-tsc 阶段就是编译错误，比运行时断言更早、更强。
  // 这里额外守住类型管不到的部分：periodDays 是正数且有限（0 / NaN 都能通过类型检查）。
  const orbits = CRAFTS.filter(({ craft }) => craft.trajectory?.kind === 'orbit')
  assert.ok(orbits.length > 0, '至少应存在一条环绕轨道')

  const bad = orbits
    .filter(({ craft }) => {
      const p = craft.trajectory.periodDays
      return !Number.isFinite(p) || p <= 0
    })
    .map(({ page, craft }) => `${page}/${craft.id} periodDays=${craft.trajectory.periodDays}`)

  assert.deepEqual(bad, [], '环绕轨道必须有正的真实周期，否则标记会静默退化成默认速度')
})

test('开放路径（飞掠 / 接近 / 大气进入）不得声明周期', () => {
  // 判别联合已在编译期拦下这种写法（见上一个用例的注释）；这里再断言一次数据本身，
  // 防止将来有人放宽类型后悄悄把周期塞进开放路径。
  const open = CRAFTS.filter(({ craft }) =>
    craft.trajectory && ['flyby', 'approach', 'entry'].includes(craft.trajectory.kind),
  )
  assert.ok(open.length > 0, '应存在开放路径轨迹')
  const wrong = open
    .filter(({ craft }) => 'periodDays' in craft.trajectory)
    .map(({ page, craft }) => `${page}/${craft.id}`)
  assert.deepEqual(wrong, [], '开放路径不是闭合轨道，不应有周期')
})

test('朱诺号使用卫星飞掠后约 33 天的周期', () => {
  const juno = CRAFTS.find(({ craft }) => craft.id === 'juno')
  assert.ok(juno, '朱诺号应存在于档案中')
  assert.equal(juno.craft.trajectory?.kind, 'orbit')
  // https://science.nasa.gov/mission/juno/juno-orbits/：2024-02 起为约 33 天，53 天为早期阶段。
  assert.equal(juno.craft.trajectory.periodDays, 33)
})

test('完整轨道注明代表性阶段，历史任务标注演示而不冒充当前位置', () => {
  for (const { craft } of CRAFTS) {
    if (craft.trajectory?.kind !== 'orbit' || craft.id === 'bepicolombo') continue
    assert.ok(craft.trajectory.stage, `${craft.id} 应注明周期对应的阶段`)
    if (craft.status === '已结束') assert.match(craft.trajectory.stage, /历史轨道演示/)
  }
  assert.equal(CRAFTS.find(({craft})=>craft.id === 'cassini').craft.trajectory.periodDays,6.5)
  assert.equal(CRAFTS.find(({craft})=>craft.id === 'galileo').craft.trajectory.periodDays,72)
})

test('飞行器 id 全局唯一', () => {
  const seen = new Map()
  const dupes = []
  for (const { page, craft } of CRAFTS) {
    if (seen.has(craft.id)) dupes.push(`${craft.id}（${seen.get(craft.id)} 与 ${page}）`)
    else seen.set(craft.id, page)
  }
  assert.deepEqual(dupes, [], 'id 重复会让选中/跳转落到错误的条目上')
})

test('轨迹几何有效：标记必须落在球体之外，否则会被行星表面埋住', () => {
  const bad = []
  for (const { page, craft } of CRAFTS) {
    const t = craft.trajectory
    if (!t) continue
    if (!(t.radius > 1)) bad.push(`${page}/${craft.id} radius=${t.radius}（应 > 1）`)
    if (t.endpointRadius != null && !(t.endpointRadius >= 1)) {
      bad.push(`${page}/${craft.id} endpointRadius=${t.endpointRadius}（应 >= 1）`)
    }
    if (t.eccentricity != null && !(t.eccentricity >= 0 && t.eccentricity < 1)) {
      bad.push(`${page}/${craft.id} eccentricity=${t.eccentricity}（应 ∈ [0,1)）`)
    }
    if (t.kind === 'orbit' && t.radius * (1 - (t.eccentricity ?? 0.12)) <= 1) {
      bad.push(`${page}/${craft.id} 近心点落入球体（应 > 1）`)
    }
  }
  assert.deepEqual(bad, [])
})

test('目录条目与场景实体一致：每个飞行器要么有轨迹、要么关联 ORBIT 目录、要么显式声明仅档案', () => {
  const noSceneEntity = CRAFTS
    .filter(({ craft }) => !craft.trajectory && !craft.orbitCatalogId)
    .map(({ craft }) => craft.id)

  assert.deepEqual(
    noSceneEntity.sort(),
    [...ARCHIVE_ONLY].sort(),
    '新增无场景实体的档案时必须显式加入 ARCHIVE_ONLY（表示"确认不画"），否则目录会出现点不动的条目',
  )

  // 白名单本身不能变成垃圾桶：已经补上场景实体的条目必须从白名单移除
  const stale = [...ARCHIVE_ONLY].filter((id) => {
    const hit = CRAFTS.find(({ craft }) => craft.id === id)
    return hit && (hit.craft.trajectory || hit.craft.orbitCatalogId)
  })
  assert.deepEqual(stale, [], `这些条目已有场景实体，应从 ARCHIVE_ONLY 移除：${stale.join(', ')}`)
})

test('orbitCatalogId 必须能在后端 ORBIT 卫星目录中找到', async () => {
  const referenced = [
    ...new Set(CRAFTS.map(({ craft }) => craft.orbitCatalogId).filter(Boolean)),
  ]
  assert.ok(referenced.length > 0, '应存在跨页引用的 ORBIT 目录条目')

  const dir = fileURLToPath(new URL('../../backend/internal/database/migrations/', import.meta.url))
  const files = (await readdir(dir)).filter((f) => f.endsWith('.sql')).sort()

  // 只取 `INSERT INTO spacecraft(...) VALUES ('id', ...)` 的首列，避免把别的表里的同名字符串算进来
  const known = new Set()
  for (const file of files) {
    const sql = await readFile(pathToFileURL(join(dir, file)), 'utf8')
    const re = /INSERT\s+INTO\s+spacecraft\s*\(([^)]*)\)\s*VALUES([\s\S]*?)(?:ON\s+CONFLICT|;)/gi
    let block
    while ((block = re.exec(sql))) {
      const first = block[1].split(',')[0].trim()
      if (first !== 'id') continue
      for (const row of block[2].matchAll(/\(\s*'([^']+)'/g)) known.add(row[1])
    }
  }

  assert.ok(known.size > 0, '未能从后端迁移里解析出任何卫星 id，说明解析逻辑或路径已失效')

  const missing = referenced.filter((id) => !known.has(id))
  assert.deepEqual(
    missing,
    [],
    `这些 orbitCatalogId 在后端 ORBIT 目录里不存在，跳转后没有目标：${missing.join(', ')}`,
  )
})

test('着陆点的 kind 与 icon 必须成对匹配', () => {
  const EXPECTED = { landing: 'lander', impact: 'impact', atmospheric: 'probe' }
  const bad = SITES
    .filter(({ site }) => EXPECTED[site.kind] !== site.icon)
    .map(({ page, site }) => `${page}/${site.id} kind=${site.kind} icon=${site.icon}`)
  assert.deepEqual(bad, [], '图例与图标不一致会让着陆点类型显示错误')
})

test('着陆点坐标合法：纬度 ∈ [-90,90]，经度 ∈ [0,360)', () => {
  const bad = []
  for (const { page, site } of SITES) {
    const { latitude: lat, longitude: lon } = site
    // 类型允许 null（未公布/无法复算的坐标），此时两个都必须为空，不能只填一个
    if ((lat == null) !== (lon == null)) {
      bad.push(`${page}/${site.id} 坐标只填了一半 lat=${lat} lon=${lon}`)
      continue
    }
    if (lat == null) continue
    if (!(lat >= -90 && lat <= 90)) bad.push(`${page}/${site.id} latitude=${lat}`)
    if (!(lon >= 0 && lon < 360)) bad.push(`${page}/${site.id} longitude=${lon}（应换算到 0..360°E）`)
  }
  assert.deepEqual(bad, [])
})

test('土卫六着陆点只能挂在土星页', () => {
  const titan = SITES.filter(({ site }) => site.body === 'titan')
  assert.ok(titan.length > 0, '应存在土卫六着陆点（惠更斯号）')
  const misplaced = titan.filter(({ page }) => page !== 'saturn').map(({ page, site }) => `${page}/${site.id}`)
  assert.deepEqual(misplaced, [], '卫星着陆点不能钉到母行星云图上')
})
