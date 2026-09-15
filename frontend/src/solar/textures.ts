/**
 * 太阳系纹理缓存：启动时把纹理真正加载成 THREE.Texture 对象（解码 + 颜色空间就绪），
 * 场景构造时直接复用——首次渲染预编译的就是带贴图的最终 shader 程序，
 * 避免飞行过程中每张纹理到达时的程序重编译卡顿。
 */
import * as THREE from 'three'
import { ALL_TEXTURE_URLS } from './data'

type TextureStatus = 'loading' | 'ready' | 'error'

interface TextureEntry {
  texture: THREE.Texture
  status: TextureStatus
  callbacks: Set<(texture: THREE.Texture) => void>
  settled: Promise<void>
}

const cache = new Map<string, TextureEntry>()

function loadSolarTexture(url: string): TextureEntry {
  const cached = cache.get(url)
  if (cached) return cached

  let settle!: () => void
  const callbacks = new Set<(texture: THREE.Texture) => void>()
  const settled = new Promise<void>((resolve) => {
    settle = resolve
  })
  let entry!: TextureEntry
  const texture = new THREE.TextureLoader().load(
    url,
    (loaded) => {
      entry.status = 'ready'
      try {
        for (const callback of entry.callbacks) callback(loaded)
      } finally {
        entry.callbacks.clear()
        settle()
      }
    },
    undefined,
    () => {
      entry.status = 'error'
      entry.callbacks.clear()
      settle()
    },
  )
  entry = { texture, status: 'loading', callbacks, settled }
  cache.set(url, entry)
  return entry
}

/**
 * 获取太阳系纹理：优先复用预热好的 Texture；未预热时现场加载（行为与旧版一致）。
 * 纹理已就绪时同步返回；回调在纹理可用时触发（用于需要 map 就绪后更新的材质）。
 */
export function solarTexture(url: string, onReady?: (texture: THREE.Texture) => void): THREE.Texture {
  const entry = loadSolarTexture(url)
  if (onReady) {
    if (entry.status === 'ready') onReady(entry.texture)
    else if (entry.status === 'loading') entry.callbacks.add(onReady)
  }
  return entry.texture
}

/** 启动预热：真正加载全部太阳系纹理（浏览器缓存命中时解码极快） */
export function preloadSolarTextures() {
  for (const url of ALL_TEXTURE_URLS) {
    loadSolarTexture(url)
    ensureDecoded(url) // 强制浏览器解码（8k JPG 解码耗时，提前到封面加载时完成）
  }
}

const decodedPromises = new Map<string, Promise<void>>()

/** 强制浏览器解码纹理（Image.decode 异步解码，不卡主线程；THREE 复用同一缓存） */
function ensureDecoded(url: string): Promise<void> {
  if (!decodedPromises.has(url)) {
    decodedPromises.set(
      url,
      new Promise((resolve) => {
        const img = new Image()
        img.crossOrigin = 'anonymous'
        img.onload = () => {
          img.decode().then(() => resolve()).catch(() => resolve())
        }
        img.onerror = () => resolve()
        img.src = url
      }),
    )
  }
  return decodedPromises.get(url)!
}

/**
 * 全部太阳系纹理就绪：同时等待 THREE.TextureLoader 的真实加载状态与浏览器解码。
 * 失败会正常结束等待，具体材质保持隐藏/降级，避免过渡永久卡住。
 */
export function solarTexturesReady(): Promise<void> {
  return Promise.all(
    ALL_TEXTURE_URLS.map((url) => Promise.all([loadSolarTexture(url).settled, ensureDecoded(url)])),
  ).then(() => undefined)
}
