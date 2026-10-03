---
name: node-test-ts-vite-modules
description: 在 Vite + TypeScript 项目里用 `node --test` 直接 import 并断言真实的 TS 数据模块（含 .jpg/.svg 资源导入与免扩展名导入），而不是用正则去抠源码字符串。当需要给「数据完整性」写测试、发现源码解析式断言太脆弱、或遇到 ERR_MODULE_NOT_FOUND / ERR_UNKNOWN_FILE_EXTENSION 时使用。
agent_created: true
---

# 用 node --test 直接测试 Vite 项目的 TS 模块

## 问题

Vite 项目里 `node --test` 直接 import TS 模块会连续撞两堵墙：

1. `import x from './assets/a.jpg'` —— Node 解析不了图片。
2. `import { y } from './solar/data'` —— 免扩展名导入，Node 解析不了。

于是很多人退化成「读文件 + 正则抠源码」，断言脆、易漏、重构即失效。
AURORA 的 `tests/` 里原先就有这种测试，被本方案取代。

**正解**：注册一个 resolve 钩子，把资源导入换成占位字符串、把免扩展名补成 `.ts`。

## 关键坑：`--import` 不会注册钩子

```bash
# ❌ 这样写钩子一次都不会被调用（只是执行了模块副作用）
node --import ./hooks.mjs script.mjs
```

必须在被测文件里用 `module.register()`：

```js
import { register } from 'node:module'
register('./helpers/asset-stub-hooks.mjs', import.meta.url)  // 相对本文件解析
const mod = await import('../src/planetPages.ts')            // 之后的动态导入才会走钩子
```

`register()` 只对**之后**的导入生效，顺序不能反。
调试技巧：在钩子里 `process.stderr.write('[hook] ' + specifier + '\n')`，
一行都不打印就说明钩子根本没挂上。

## 钩子实现

`frontend/tests/helpers/asset-stub-hooks.mjs`（本项目已就位，可直接复用）：

```js
import { existsSync } from 'node:fs'
import { fileURLToPath, pathToFileURL } from 'node:url'

const ASSET = /\.(jpe?g|png|webp|avif|gif|svg|woff2?|ttf|otf|mp3|mp4|webm|glb|gltf|bin|hdr|exr)(\?.*)?$/i
const EXTENSIONS = ['.ts', '.mts', '.js', '.mjs']

export async function resolve(specifier, context, nextResolve) {
  if (ASSET.test(specifier)) {
    return { url: 'data:text/javascript,export default "asset-stub"', shortCircuit: true }
  }
  if (context.parentURL && /^\.{1,2}\//.test(specifier)) {
    const base = fileURLToPath(new URL(specifier, context.parentURL))
    if (!/\.[a-z0-9]+$/i.test(base)) {          // 只补没有扩展名的裸相对导入
      for (const ext of EXTENSIONS) {
        if (existsSync(base + ext)) return { url: pathToFileURL(base + ext).href, shortCircuit: true }
      }
    }
  }
  return nextResolve(specifier, context)
}
```

要点：
- `shortCircuit: true` 必须加，否则继续走默认解析。
- 已带扩展名的导入要放行给默认解析，保留原有报错语义。
- 钩子文件名**不要**匹配 `*.test.mjs`，否则 `node --test` 会把它当测试跑。
  放在 `tests/helpers/` 下即可（Node 只对 `test/` 单数目录做全量匹配，`tests/` 复数只按文件名匹配）。

## 前置条件

- Node ≥ 22.18（TS 类型擦除默认开启）。低版本需 `--experimental-strip-types`。
- 只能擦除**类型**：`enum`、`namespace`、构造函数参数属性会报错。
- 项目 `package.json` 需要 `"type": "module"`（或文件用 `.mjs`）。
- 测试里 import 裸包（如 `three`）没问题 —— 测试文件在项目内，`node_modules` 可达。

## 别只写测试：用变异测试证明断言不是摆设

断言全绿可能只是「什么也没检查」。手工破坏每一类不变量，确认对应测试确实变红：

```python
# 备份 -> 逐条变异 -> 跑测试 -> 期望 fail>0 -> 还原
MUTATIONS = [
    ('删掉必填字段', lambda s: s.replace(', periodDays: 53', '')),
    ('字段改成非法值', lambda s: s.replace('periodDays: 88', 'periodDays: 0')),
    ('新增一条缺字段的条目', ...),
    ('外键指向不存在的目标', ...),
]
```

跑完必须校验 `还原后文件 == 原文`，避免把变异残留提交进去。

## 相关经验

- **跨层一致性也值得测**：解析后端迁移 SQL（`INSERT INTO t(...) VALUES ('id', ...)` 的首列）
  来断言前端的 `orbitCatalogId` 等外键真的存在 —— 这类错漏类型系统看不见。
- **判别联合优先于运行时断言**：如果能把不变量做成 `kind` 判别的联合类型，
  缺失/多余字段会直接变成编译错误（TS2322 / TS2353），比任何测试都早。
  实测：`{ kind: 'flyby', periodDays: 5 }` 会触发 TS2353，`{ kind: 'orbit' }` 缺字段会触发 TS2322。
- **写参考实现对拍时，先确认参考的适用范围**：把「射线-球体」当参考去对拍
  「球面点遮挡」会失败 —— 目标点落在判定球内部时，纯射线判定会得出「被自己挡住」的结论。
  先做解析推导（判别式是否 < 0）再决定采样域，别急着改被测实现。
