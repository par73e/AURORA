import * as THREE from 'three'
import { solarTexture } from './textures'
import { wantsDetail } from './textureResolution.ts'
import { previewTextureUrl } from './textureLevels'

/** Keep the existing map while a requested tier loads; late callbacks cannot revive disposed scenes. */
export class DetailTexture {
  readonly preview: THREE.Texture
  private detailed = false
  private requested = false
  private disposed = false
  private high?: THREE.Texture
  constructor(private readonly url: string, private readonly apply: (texture: THREE.Texture) => void, ready?: () => void) {
    this.preview = solarTexture(url, () => ready?.())
  }
  update(pixels: number) {
    if (this.disposed || previewTextureUrl(this.url) === this.url) return
    const desired = wantsDetail(pixels, this.detailed)
    if (desired !== this.detailed) {
      this.detailed = desired
      if (!desired) this.apply(this.preview)
      else if (this.high) this.apply(this.high)
    }
    if (desired && !this.requested) {
      this.requested = true
      solarTexture(this.url, texture => {
        if (this.disposed || !texture.image) return
        texture.colorSpace = this.preview.colorSpace
        texture.anisotropy = this.preview.anisotropy
        this.high = texture
        if (this.detailed) this.apply(texture)
      }, true)
    }
  }
  dispose() { this.disposed = true }
}
