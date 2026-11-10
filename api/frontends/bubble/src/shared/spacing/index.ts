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

/** Same surface, flush edge, so a divider or list can meet the border. */
export const WORKSPACE_CARD_FLUSH = {
  variant: WORKSPACE_CARD.variant,
  elevation: WORKSPACE_CARD.elevation,
  padding: 0 as const,
}
