/** URL hashes are the source of truth on refresh and direct entry. */
export type AppSurface = 'cover' | 'sky' | 'solar-system' | 'orbit' | 'moon' | 'mars' | 'mercury' | 'venus' | 'saturn' | 'jupiter' | 'uranus' | 'neptune' | 'sun'
export type SkyPage = 'conditions' | 'sky' | 'events' | 'daily-image'

const skyPages: Record<string, SkyPage> = {
  '#astronomy': 'conditions',
  '#astronomy-conditions': 'conditions',
  '#astronomy-sky': 'sky',
  '#astronomy-events': 'events',
  '#astronomy-daily-image': 'daily-image',
  '#astronomy-tonight': 'sky',
  '#astronomy-windows': 'sky',
  '#astronomy-targets': 'sky',
}
const bodySections = {
  moon: ['scene', 'profile', 'objects', 'sites'],
  mars: ['scene', 'profile', 'objects', 'sites'],
  mercury: ['scene', 'profile', 'objects', 'sites'],
  venus: ['scene', 'profile', 'objects', 'sites'],
  jupiter: ['scene', 'profile', 'objects', 'sites'],
  saturn: ['scene', 'profile', 'objects', 'sites'],
  uranus: ['scene', 'profile', 'objects'],
  neptune: ['scene', 'profile', 'objects'],
  sun: ['scene', 'profile', 'objects'],
} as const

export function skyPageFromHash(hash: string): SkyPage {
  return skyPages[hash] ?? 'conditions'
}

export function surfaceFromHash(hash: string): AppSurface {
  if (Object.hasOwn(skyPages, hash)) return 'sky'
  if (hash === '#solar-system') return 'solar-system'
  if (['#earth', '#objects', '#sites', '#launches'].includes(hash)) return 'orbit'
  for (const [body, sections] of Object.entries(bodySections)) {
    if (hash === `#${body}` || sections.some(section => hash === `#${body}-${section}`)) return body as AppSurface
  }
  return 'cover'
}

/** Scrollable sections mount asynchronously, after the browser's native anchor jump. */
export function sectionFromHash(hash: string): string | null {
  const surface = surfaceFromHash(hash)
  if (surface === 'cover' || surface === 'sky' || surface === 'solar-system') return null
  if (surface === 'orbit') return hash.slice(1)
  return hash === `#${surface}` ? `${surface}-scene` : hash.slice(1)
}
