import type { IconColor } from '@astryxdesign/core/Icon'
import type { TokenColor } from '@astryxdesign/core/Token'

/**
 * Icon colors — one rule everywhere (Meta `shell-side-nav`, `login-split`):
 * primary for UI chrome, secondary for empty / de-emphasized states.
 */
export const SHELL_ICON_COLOR = 'primary' satisfies IconColor
export const MUTED_ICON_COLOR = 'secondary' satisfies IconColor

/** Neutral metadata chips only (file kind, roles). */
export const META_TOKEN_COLOR = 'gray' satisfies TokenColor
