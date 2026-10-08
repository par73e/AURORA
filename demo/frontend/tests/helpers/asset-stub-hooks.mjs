// 让 `node --test` 能直接 import 带静态资源导入的 TS 模块（例如 src/planetPages.ts）。
//
// 背景：planetPages.ts 顶部 `import venusUrl from './assets/solar/4k_venus_atmosphere.jpg'`，
// Vite 会把 .jpg 变成 URL 字符串，但 Node 解析不了；同时项目里用的是 Vite 风格的免扩展名
// 导入（`./solar/data`），Node 也解析不了。
//
// 这里做两件事，使测试能断言**真实数据结构**，而不是用正则去抠源码字符串：
//   1. 把图片/字体/媒体等资源导入替换成一个占位字符串；
//   2. 把免扩展名的相对导入补成 .ts / .mts / .js / .mjs。
//
// 由测试文件用 `register()` 显式挂载（见 planet_catalog_integrity.test.mjs）。
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
    // 只处理没有扩展名的裸相对导入；已带扩展名的交给默认解析（保持原有报错语义）
    if (!/\.[a-z0-9]+$/i.test(base)) {
      for (const ext of EXTENSIONS) {
        if (existsSync(base + ext)) {
          return { url: pathToFileURL(base + ext).href, shortCircuit: true }
        }
      }
    }
  }

  return nextResolve(specifier, context)
}
