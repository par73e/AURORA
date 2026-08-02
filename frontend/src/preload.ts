// 页面切换前的资源预热：利用过渡动画时间提前加载目标页的重纹理，
// 避免切换后出现"建模加载"式的卡顿（尤其远端的地球纹理）。

import { EARTH_DAY_TEXTURE_URL, EARTH_NIGHT_TEXTURE_URL } from './orbit/coordinates'
import { preloadSolarTextures as preloadSolarTextureObjects } from './solar/textures'

const preloaded = new Set<string>()

function warm(url: string) {
  if (preloaded.has(url)) return
  preloaded.add(url)
  const image = new Image()
  image.crossOrigin = 'anonymous'
  image.src = url
}

/** 预热太阳系纹理：真正加载成 THREE.Texture 对象，场景复用后首帧即带贴图编译 */
export function preloadSolarTextures() {
  preloadSolarTextureObjects()
}

/** 预热 ORBIT 地球纹理（远端 unpkg，提前加载可避免进场时地球灰模/弹出） */
export function preloadOrbitTextures() {
  warm(EARTH_DAY_TEXTURE_URL)
  warm(EARTH_NIGHT_TEXTURE_URL)
}
