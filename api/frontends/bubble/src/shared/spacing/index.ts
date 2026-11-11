/** Spacing tiers — local / group / section (spatial hierarchy). */
export const WORKSPACE_GAP = {
  local: 1 as const,
  group: 3 as const,
  section: 6 as const,
}

/**
 * Muted, low-elevation surface for panels, previews, and tiles.
 * Inner padding is the group step.
 */
export const WORKSPACE_CARD = {
  variant: 'muted' as const,
  elevation: 'low' as const,
  padding: WORKSPACE_GAP.group,
}
