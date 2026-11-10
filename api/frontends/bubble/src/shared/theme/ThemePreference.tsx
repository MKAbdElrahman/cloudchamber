import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'

import { readAppearance, writeAppearance, type ColorMode } from './appearanceStorage'
import { APP_THEMES, type AppThemeId } from './themes'
import { ThemePreferenceContext } from './useThemePreference'

export function ThemePreferenceProvider({ children }: { children: ReactNode }) {
  const initial = readAppearance()
  const [themeId, setThemeIdState] = useState<AppThemeId>(initial.themeId)
  const [colorMode, setColorModeState] = useState<ColorMode>(initial.colorMode)

  useEffect(() => {
    writeAppearance({ v: 1, themeId, colorMode })
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
