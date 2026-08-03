/**
 * 太阳系纹理缓存：启动时把纹理真正加载成 THREE.Texture 对象（解码 + 颜色空间就绪），
 * 场景构造时直接复用——首次渲染预编译的就是带贴图的最终 shader 程序，
 * 避免飞行过程中每张纹理到达时的程序重编译卡顿。
 */
import * as THREE from 'three'
import { ALL_TEXTURE_URLS } from './data'

const cache = new Map<string, THREE.Texture>()

/**
 * 获取太阳系纹理：优先复用预热好的 Texture；未预热时现场加载（行为与旧版一致）。
 * 纹理已就绪时同步返回；回调在纹理可用时触发（用于需要 map 就绪后更新的材质）。
 */
export function solarTexture(url: string, onReady?: (texture: THREE.Texture) => void): THREE.Texture {
  const existing = cache.get(url)
  if (existing) {
    if (existing.image && onReady) onReady(existing)
    return existing
  }
  const texture = new THREE.TextureLoader().load(url, (t) => {
    if (onReady) onReady(t)
  })
  cache.set(url, texture)
  return texture
}

/** 启动预热：真正加载全部太阳系纹理（浏览器缓存命中时解码极快） */
export function preloadSolarTextures() {
  for (const url of ALL_TEXTURE_URLS) {
    solarTexture(url)
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

/** 全部太阳系纹理解码就绪（进入动画等待此 Promise：就绪即开始，超时由调用方兜底） */
export function solarTexturesReady(): Promise<void> {
  return Promise.all(ALL_TEXTURE_URLS.map(ensureDecoded)).then(() => undefined)
}
