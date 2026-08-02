// 页面切换前的资源预热：利用过渡动画时间提前加载目标页的重纹理，
// 避免切换后出现"建模加载"式的卡顿（尤其远端的地球纹理）。

import { ALL_TEXTURE_URLS } from './solar/data'
import { EARTH_DAY_TEXTURE_URL, EARTH_NIGHT_TEXTURE_URL } from './orbit/coordinates'

const preloaded = new Set<string>()

function warm(url: string) {
  if (preloaded.has(url)) return
  preloaded.add(url)
  const image = new Image()
  image.crossOrigin = 'anonymous'
  image.src = url
}

/** 预热太阳系纹理（本地打包资源，浏览器解码缓存） */
export function preloadSolarTextures() {
  for (const url of ALL_TEXTURE_URLS) warm(url)
}

/** 预热 ORBIT 地球纹理（远端 unpkg，提前加载可避免进场时地球灰模/弹出） */
export function preloadOrbitTextures() {
  warm(EARTH_DAY_TEXTURE_URL)
  warm(EARTH_NIGHT_TEXTURE_URL)
}
