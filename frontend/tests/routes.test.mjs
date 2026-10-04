import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { sectionFromHash, skyPageFromHash, surfaceFromHash } from '../src/routes.ts'

test('刷新和直接访问可恢复所有页面及下方栏目', () => {
  for (const hash of ['', '#home', '#unknown', '#astronomy-unknown']) assert.equal(surfaceFromHash(hash), 'cover')
  assert.equal(surfaceFromHash('#solar-system'), 'solar-system')
  for (const hash of ['#earth', '#objects', '#sites', '#launches']) {
    assert.equal(surfaceFromHash(hash), 'orbit')
    assert.equal(sectionFromHash(hash), hash.slice(1))
  }
  for (const body of ['moon', 'mars', 'mercury', 'venus', 'jupiter', 'saturn', 'uranus', 'neptune', 'sun']) {
    assert.equal(surfaceFromHash(`#${body}`), body)
    assert.equal(sectionFromHash(`#${body}`), `${body}-scene`)
    const sections = ['scene', 'profile', 'objects', ...(['uranus', 'neptune', 'sun'].includes(body) ? [] : ['sites'])]
    for (const section of sections) {
      const hash = `#${body}-${section}`
      assert.equal(surfaceFromHash(hash), body)
      assert.equal(sectionFromHash(hash), hash.slice(1))
    }
  }
})

test('天文观测入口与各子页面使用同一份路由，包括每日一图及旧地址', () => {
  const expected = { astronomy: 'conditions', 'astronomy-conditions': 'conditions', 'astronomy-sky': 'sky', 'astronomy-events': 'events', 'astronomy-daily-image': 'daily-image', 'astronomy-tonight': 'sky', 'astronomy-windows': 'sky', 'astronomy-targets': 'sky' }
  for (const [hash, page] of Object.entries(expected)) {
    assert.equal(surfaceFromHash(`#${hash}`), 'sky')
    assert.equal(skyPageFromHash(`#${hash}`), page)
    assert.equal(sectionFromHash(`#${hash}`), null)
  }
})

test('所有现有导航锚点都能恢复对应页面', async () => {
  const source = await readFile(new URL('../src/App.vue', import.meta.url), 'utf8')
  const hashes = [...source.matchAll(/href="(#[a-z-]+)"/g)].map(match => match[1])
  assert.ok(hashes.length >= 35)
  for (const hash of hashes) if (hash !== '#home') assert.notEqual(surfaceFromHash(hash), 'cover', hash)
})
