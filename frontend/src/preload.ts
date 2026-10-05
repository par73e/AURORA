// 页面切换前的资源预热：利用过渡动画时间提前加载目标页的重纹理，
// 避免切换后出现"建模加载"式的卡顿（尤其远端的地球纹理）。

import { previewTextureUrl } from './solar/textureLevels'
import { EARTH_DAY_TEXTURE_URL, EARTH_NIGHT_TEXTURE_URL } from './orbit/coordinates'
import { MARS_HD, MOON_HD } from './solar/data'
import { preloadSolarTextures as preloadSolarTextureObjects } from './solar/textures'

const preloaded = new Set<string>()
const decodePromises = new Map<string, Promise<void>>()

function warm(url: string) {
  if (preloaded.has(url)) return
  preloaded.add(url)
  const image = new Image()
  image.crossOrigin = 'anonymous'
  image.src = url
}

/** 强制浏览器解码（异步、不卡主线程）；返回就绪 Promise（失败也 resolve，避免卡死过渡） */
function warmAndDecode(url: string): Promise<void> {
  warm(url)
  if (!decodePromises.has(url)) {
    decodePromises.set(
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
  return decodePromises.get(url)!
}

/** 预热太阳系纹理：真正加载成 THREE.Texture 对象，场景复用后首帧即带贴图编译 */
export function preloadSolarTextures() {
  preloadSolarTextureObjects()
}

/** 预热 ORBIT 地球纹理（远端 unpkg，提前加载可避免进场时地球灰模/弹出） */
export function preloadOrbitTextures() {
  warm(previewTextureUrl(EARTH_DAY_TEXTURE_URL))
  warm(EARTH_NIGHT_TEXTURE_URL)
}

/** 地球纹理解码就绪（黑幕期间等待；就绪才揭示，避免大图解码卡顿） */
export function orbitTexturesReady(): Promise<void> {
  return Promise.all([warmAndDecode(previewTextureUrl(EARTH_DAY_TEXTURE_URL)), warmAndDecode(EARTH_NIGHT_TEXTURE_URL)]).then(() => undefined)
}

/** 预热月球预览贴图；高清贴图由独立场景按屏幕尺寸请求 */
export function preloadMoonHdTexture() {
  warm(previewTextureUrl(MOON_HD.textureUrl))
}

/** 月球预览纹理解码就绪（黑幕期间等待） */
export function moonHdReady(): Promise<void> {
  return warmAndDecode(previewTextureUrl(MOON_HD.textureUrl))
}

/** 预热火星预览贴图，高清贴图按需加载 */
export function preloadMarsHdTexture() {
  warm(previewTextureUrl(MARS_HD.textureUrl))
}

/** 火星预览纹理解码就绪（黑幕期间等待） */
export function marsHdReady(): Promise<void> {
  return warmAndDecode(previewTextureUrl(MARS_HD.textureUrl))
}
