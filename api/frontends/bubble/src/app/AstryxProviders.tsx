import type { ReactNode } from 'react'

import { LayerProvider } from '@astryxdesign/core/Layer'
import { Theme } from '@astryxdesign/core/theme'

import { ThemeDocumentSync } from './ThemeDocumentSync'
import { ThemePreferenceProvider, useThemePreference } from '@/shared/theme'

export function AstryxProviders({ children }: { children: ReactNode }) {
  return (
    <ThemePreferenceProvider>
      <ThemedLayerRoot>{children}</ThemedLayerRoot>
    </ThemePreferenceProvider>
  )
}

function ThemedLayerRoot({ children }: { children: ReactNode }) {
  const { option, colorMode } = useThemePreference()

  return (
    <Theme theme={option.theme} mode={colorMode}>
      <ThemeDocumentSync />
      <LayerProvider toast={{ position: 'bottomEnd', maxVisible: 4 }}>{children}</LayerProvider>
    </Theme>
  )
}
