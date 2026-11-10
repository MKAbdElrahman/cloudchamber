/** Default max width for in-app pages (Astryx layout — one content line). */
export const WORKSPACE_CONTENT_WIDTH = 960

/** Main region padding for workspace pages. */
export const WORKSPACE_CONTENT_PADDING = 6 as const

/** Spacing tiers — local / group / section (spatial hierarchy). */
export const WORKSPACE_GAP = {
  local: 1 as const,
  group: 3 as const,
  section: 6 as const,
}

/** Muted card defaults for step panels and lists. */
export const WORKSPACE_CARD = {
  variant: 'muted' as const,
  elevation: 'low' as const,
}
