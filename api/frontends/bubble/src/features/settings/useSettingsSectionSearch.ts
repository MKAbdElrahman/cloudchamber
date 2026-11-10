import { useMemo, useState } from 'react'

import type { SettingsSectionId } from '@/shared/layout/workspace'

export type SettingsSearchHit = {
  id: SettingsSectionId
  label: string
  description: string
}

const INDEX: SettingsSearchHit[] = [
  {
    id: 'appearance',
    label: 'Appearance',
    description: 'Color theme and theme pack',
  },
  {
    id: 'account',
    label: 'Account',
    description: 'Session identity, roles, token expiry',
  },
  {
    id: 'developer',
    label: 'Developer',
    description: 'API bases, Astryx references, environment',
  },
]

function matchesQuery(hit: SettingsSearchHit, query: string) {
  const hay = `${hit.label} ${hit.description} ${hit.id}`.toLowerCase()
  return hay.includes(query)
}

export function useSettingsSectionSearch() {
  const [query, setQuery] = useState('')
  const normalized = query.trim().toLowerCase()

  const results = useMemo(() => {
    if (!normalized) return INDEX
    return INDEX.filter((hit) => matchesQuery(hit, normalized))
  }, [normalized])

  return { query, setQuery, results, isFiltering: normalized.length > 0 }
}
