import { APP_THEMES, DEFAULT_THEME_ID, type AppThemeId } from './themes'

export const APPEARANCE_STORAGE_KEY = 'bubble.appearance.v1'
const PREVIOUS_APPEARANCE_KEY = 'cloudchamber.appearance.v1'
const LEGACY_THEME_KEY = 'cloudchamber.astryx-theme'
const LEGACY_MODE_KEY = 'cloudchamber.astryx-color-mode'

export type ColorMode = 'light' | 'dark' | 'system'

export type AppearanceSnapshot = {
  v: 1
  themeId: AppThemeId
  colorMode: ColorMode
}

function isColorMode(v: unknown): v is ColorMode {
  return v === 'light' || v === 'dark' || v === 'system'
}

function isThemeId(v: unknown): v is AppThemeId {
  return typeof v === 'string' && v in APP_THEMES
}

function parseSnapshot(raw: string | null): AppearanceSnapshot | undefined {
  if (!raw) return undefined
  try {
    const parsed = JSON.parse(raw) as Partial<AppearanceSnapshot>
    if (parsed.v === 1 && isThemeId(parsed.themeId) && isColorMode(parsed.colorMode)) {
      return { v: 1, themeId: parsed.themeId, colorMode: parsed.colorMode }
    }
  } catch {
    return undefined
  }
  return undefined
}

export function readAppearance(): AppearanceSnapshot {
  try {
    const current = parseSnapshot(localStorage.getItem(APPEARANCE_STORAGE_KEY))
    if (current) return current
    const previous = parseSnapshot(localStorage.getItem(PREVIOUS_APPEARANCE_KEY))
    if (previous) {
      writeAppearance(previous)
      localStorage.removeItem(PREVIOUS_APPEARANCE_KEY)
      return previous
    }
  } catch {
    /* ignore */
  }

  let themeId: AppThemeId = DEFAULT_THEME_ID
  let colorMode: ColorMode = 'system'
  try {
    const legacyTheme = localStorage.getItem(LEGACY_THEME_KEY)
    const legacyMode = localStorage.getItem(LEGACY_MODE_KEY)
    if (isThemeId(legacyTheme)) themeId = legacyTheme
    if (isColorMode(legacyMode)) colorMode = legacyMode
  } catch {
    /* ignore */
  }

  const snapshot: AppearanceSnapshot = { v: 1, themeId, colorMode }
  writeAppearance(snapshot)
  return snapshot
}

export function writeAppearance(snapshot: AppearanceSnapshot) {
  try {
    localStorage.setItem(APPEARANCE_STORAGE_KEY, JSON.stringify(snapshot))
    localStorage.removeItem(PREVIOUS_APPEARANCE_KEY)
    localStorage.removeItem(LEGACY_THEME_KEY)
    localStorage.removeItem(LEGACY_MODE_KEY)
  } catch {
    /* ignore */
  }
}

/** Resolved light/dark for document bootstrap before React. */
export function resolveDocumentColorMode(colorMode: ColorMode): 'light' | 'dark' {
  if (colorMode === 'light' || colorMode === 'dark') return colorMode
  if (typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches) {
    return 'dark'
  }
  return 'light'
}

export function applyAppearanceToDocument(snapshot: AppearanceSnapshot) {
  const root = document.documentElement
  root.setAttribute('data-astryx-theme', snapshot.themeId)
  const resolved = resolveDocumentColorMode(snapshot.colorMode)
  root.setAttribute('data-theme', resolved)
}
