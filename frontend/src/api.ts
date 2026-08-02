import type { OrbitOverview } from './types'

export async function fetchOrbitOverview(): Promise<OrbitOverview> {
  const response = await fetch('/api/v1/orbit/overview')
  if (!response.ok) {
    throw new Error(`接口返回 ${response.status}`)
  }
  return response.json() as Promise<OrbitOverview>
}
