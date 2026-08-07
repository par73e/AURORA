import { ref } from 'vue'

interface SnapshotResponse<T> {
  spacecraft?: T[]
  syncedAt?: string | null
}

interface LandingResponse<T> {
  landingSites?: T[]
}

interface PlanetSceneDataOptions<Craft, Site> {
  loadSpacecraft: (signal: AbortSignal) => Promise<SnapshotResponse<Craft>>
  loadLandingSites: (signal: AbortSignal) => Promise<LandingResponse<Site>>
  canCommit: () => boolean
  onLoaded: (crafts: Craft[], sites: Site[]) => void
  unavailableLabel: string
}

/**
 * 月球与火星都使用“两个端点原子提交”的数据语义：任一端失败，就不把半套数据渲染成
 * 看似完整的场景。这里统一请求取消、错误文案与提交时机，行星各自的 Three.js 构建仍留在组件内。
 */
export function usePlanetSceneData<Craft, Site>(options: PlanetSceneDataOptions<Craft, Site>) {
  const crafts = ref<Craft[]>([])
  const syncedAt = ref<string | null>(null)
  const dataLoading = ref(true)
  const dataError = ref('')
  let request: AbortController | undefined

  async function loadSceneData() {
    request?.abort()
    const controller = new AbortController()
    request = controller
    dataLoading.value = true
    dataError.value = ''
    try {
      const [craftData, siteData] = await Promise.all([
        options.loadSpacecraft(controller.signal),
        options.loadLandingSites(controller.signal),
      ])
      if (controller.signal.aborted || !options.canCommit()) return
      const nextCrafts = craftData.spacecraft ?? []
      const nextSites = siteData.landingSites ?? []
      crafts.value = nextCrafts
      syncedAt.value = craftData.syncedAt ?? null
      options.onLoaded(nextCrafts, nextSites)
    } catch (reason) {
      if (controller.signal.aborted) return
      dataError.value = reason instanceof Error
        ? `${options.unavailableLabel}：${reason.message}`
        : `${options.unavailableLabel}，请稍后重试`
    } finally {
      if (request === controller) {
        request = undefined
        dataLoading.value = false
      }
    }
  }

  return { crafts, syncedAt, dataLoading, dataError, loadSceneData, abortSceneData: () => request?.abort() }
}
