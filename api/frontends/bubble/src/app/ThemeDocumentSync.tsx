import { useEffect } from 'react'

import { useTheme } from '@astryxdesign/core/theme'

/** Keeps document chrome (`theme-color`) in sync with active Astryx theme + mode. */
export function ThemeDocumentSync() {
  const { token, mode } = useTheme()

  useEffect(() => {
    const bg = token('--color-background-body')
    const elements = document.querySelectorAll('meta[name="theme-color"]')
    elements.forEach((el) => el.setAttribute('content', bg))
  }, [token, mode])

  return null
}
