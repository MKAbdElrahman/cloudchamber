import { createContext, useContext } from 'react'

import type { ColorMode } from './appearanceStorage'
import type { AppThemeId, AppThemeOption } from './themes'

export type ThemePreferenceContextValue = {
  themeId: AppThemeId
  option: AppThemeOption
  colorMode: ColorMode
  setThemeId: (id: AppThemeId) => void
  setColorMode: (mode: ColorMode) => void
}

export const ThemePreferenceContext = createContext<ThemePreferenceContextValue | null>(null)

export function useThemePreference() {
  const ctx = useContext(ThemePreferenceContext)
  if (!ctx) throw new Error('useThemePreference must be used within ThemePreferenceProvider')
  return ctx
}
