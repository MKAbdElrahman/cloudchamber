import { Text } from '@astryxdesign/core/Text'
import { VStack } from '@astryxdesign/core/Layout'

import { APP_THEMES, useThemePreference } from '@/shared/theme'
import { WORKSPACE_GAP } from '@/shared/spacing'

import { ColorModeChoiceCards } from './ThemeAppearanceControls'

/** Appearance settings — color mode on the page, with no surrounding card. */
export function AppearancePanel() {
  const { themeId, colorMode, setColorMode } = useThemePreference()

  return (
    <VStack gap={WORKSPACE_GAP.group} width="100%">
      <Text type="label">Color mode</Text>
      <ColorModeChoiceCards
        value={colorMode}
        onChange={setColorMode}
        previewTheme={APP_THEMES[themeId].theme}
      />
    </VStack>
  )
}
