import type { DefinedTheme } from '@astryxdesign/core/theme'
import { neutralTheme } from '@astryxdesign/theme-neutral/built'
import { stoneTheme } from '@astryxdesign/theme-stone/built'
import { matchaTheme } from '@astryxdesign/theme-matcha/built'

export type AppThemeId = 'neutral' | 'stone' | 'matcha'

export type AppThemeOption = {
  id: AppThemeId
  label: string
  description: string
  theme: DefinedTheme
}

export const APP_THEMES: Record<AppThemeId, AppThemeOption> = {
  neutral: {
    id: 'neutral',
    label: 'Neutral',
    description: 'Meta default — crisp product UI.',
    theme: neutralTheme,
  },
  stone: {
    id: 'stone',
    label: 'Stone',
    description: 'Warm slate, handcrafted feel.',
    theme: stoneTheme,
  },
  matcha: {
    id: 'matcha',
    label: 'Matcha',
    description: 'Organic green accent, calm studio vibe.',
    theme: matchaTheme,
  },
}

export const DEFAULT_THEME_ID: AppThemeId = 'neutral'
