import { Selector } from '@astryxdesign/core/Selector'
import { PaintBrushIcon, SwatchIcon } from '@heroicons/react/24/outline'

import { APP_THEMES, type AppThemeId, type ColorMode } from '@/shared/theme'

import { ColorModeChoiceCards } from './ThemeAppearanceControls'
import { SettingsCard, SettingsRow } from './SettingsFormLayout'

const CONTROL_WIDTH = '12rem'

const PACK_OPTIONS = Object.values(APP_THEMES).map((t) => ({ value: t.id, label: t.label }))

type AppearancePanelProps = {
  themeId: AppThemeId
  setThemeId: (id: AppThemeId) => void
  colorMode: ColorMode
  setColorMode: (mode: ColorMode) => void
}

/** Appearance settings — color mode tiles + theme pack selector. */
export function AppearancePanel({
  themeId,
  setThemeId,
  colorMode,
  setColorMode,
}: AppearancePanelProps) {
  return (
    <SettingsCard>
      <SettingsRow
        title="Color theme"
        description="Applies to this browser only."
        icon={PaintBrushIcon}
        detail={
          <ColorModeChoiceCards
            value={colorMode}
            onChange={setColorMode}
            previewTheme={APP_THEMES[themeId].theme}
          />
        }
      />
      <SettingsRow
        title="Theme pack"
        description="Accent, surfaces, and radius for the product shell."
        icon={SwatchIcon}
        control={
          <Selector
            label="Theme pack"
            isLabelHidden
            width={CONTROL_WIDTH}
            options={PACK_OPTIONS}
            value={themeId}
            onChange={(v) => setThemeId(v as AppThemeId)}
          />
        }
      />
    </SettingsCard>
  )
}
