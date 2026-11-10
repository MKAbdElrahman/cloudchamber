import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

import {
  applyAppearanceToDocument,
  readAppearance,
  writeAppearance,
  type ColorMode,
} from './appearanceStorage'
import { ensureThemePackAssets } from './themeAssets'
import { APP_THEMES, type AppThemeId, type AppThemeOption } from './themes'

export type { ColorMode }

type ThemePreferenceContextValue = {
  themeId: AppThemeId
  option: AppThemeOption
  colorMode: ColorMode
  setThemeId: (id: AppThemeId) => void
  setColorMode: (mode: ColorMode) => void
}

const ThemePreferenceContext = createContext<ThemePreferenceContextValue | null>(null)

function persist(themeId: AppThemeId, colorMode: ColorMode) {
  const snapshot = { v: 1 as const, themeId, colorMode }
  writeAppearance(snapshot)
  applyAppearanceToDocument(snapshot)
}

export function ThemePreferenceProvider({ children }: { children: ReactNode }) {
  const initial = readAppearance()
  const [themeId, setThemeIdState] = useState<AppThemeId>(initial.themeId)
  const [colorMode, setColorModeState] = useState<ColorMode>(initial.colorMode)

  useEffect(() => {
    void ensureThemePackAssets(themeId)
    persist(themeId, colorMode)
  }, [themeId, colorMode])

  const setThemeId = useCallback((id: AppThemeId) => setThemeIdState(id), [])
  const setColorMode = useCallback((mode: ColorMode) => setColorModeState(mode), [])

  const value = useMemo(
    () => ({
      themeId,
      option: APP_THEMES[themeId],
      colorMode,
      setThemeId,
      setColorMode,
    }),
    [themeId, colorMode, setThemeId, setColorMode],
  )

  return <ThemePreferenceContext.Provider value={value}>{children}</ThemePreferenceContext.Provider>
}

export function useThemePreference() {
  const ctx = useContext(ThemePreferenceContext)
  if (!ctx) throw new Error('useThemePreference must be used within ThemePreferenceProvider')
  return ctx
}
