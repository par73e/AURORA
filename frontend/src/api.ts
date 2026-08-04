import type { DeepSpaceProbe, OrbitOverview } from './types'

export async function fetchOrbitOverview(): Promise<OrbitOverview> {
  const response = await fetch('/api/v1/orbit/overview')
  if (!response.ok) {
    throw new Error(`接口返回 ${response.status}`)
  }
  return response.json() as Promise<OrbitOverview>
}

export async function fetchDeepSpaceProbes(): Promise<DeepSpaceProbe[]> {
  const response = await fetch('/api/v1/voyage/probes')
  if (!response.ok) {
    throw new Error(`接口返回 ${response.status}`)
  }
  const data = (await response.json()) as { probes: DeepSpaceProbe[] }
  return data.probes ?? []
}
